package graphics

// Incremental replay rendering. Each turn after the first is drawn in four steps, in file order:
//  1. tileTracker.takeChanges: which tiles' records changed since the last turn.
//  2. planRedraw: which tiles to draw and which rects of the canvas will change (dirtyRects).
//  3. RedrawDirtyTiles: draw those tiles on a staging canvas, then copy only dirtyRects onto the real canvas.
//  4. appendGifFrames: merge nearby dirtyRects and emit each merged rect as a GIF frame.

import (
	"cmp"
	"fmt"
	"image"
	"image/gif"
	"maps"
	"math"
	"slices"
	"sort"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// ---- Step 1: find the changed tiles ----

// tileTracker reports which tiles' records changed since it last looked; comparing state can't miss an event's effect.
type tileTracker struct {
	mapData *fileio.Civ5MapData
	prev    []fileio.Civ5MapTileImprovement
}

// newTileTracker starts tracking from mapData's current tile records.
func newTileTracker(mapData *fileio.Civ5MapData) *tileTracker {
	t := &tileTracker{mapData: mapData, prev: make([]fileio.Civ5MapTileImprovement, 0, len(mapData.MapTileImprovements)*len(mapData.MapTileImprovements[0]))}
	for _, row := range mapData.MapTileImprovements {
		for _, tile := range row {
			t.prev = append(t.prev, *tile)
		}
	}
	return t
}

// tileChanges maps each changed tile to its record before the change.
type tileChanges map[fileio.TilePos]fileio.Civ5MapTileImprovement

// tiles returns the changed tiles as a set.
func (c tileChanges) tiles() tileSet {
	set := make(tileSet, len(c))
	for rc := range c {
		set[rc] = true
	}
	return set
}

// takeChanges returns the tiles whose record differs from the last call (or from newTileTracker), with their earlier records, and makes the current state the new baseline.
func (t *tileTracker) takeChanges() tileChanges {
	changed := tileChanges{}
	i := 0
	for row, tiles := range t.mapData.MapTileImprovements {
		for col, tile := range tiles {
			if *tile != t.prev[i] {
				changed[fileio.TilePos{Row: row, Col: col}] = t.prev[i]
				t.prev[i] = *tile
			}
			i++
		}
	}
	return changed
}

// ---- Step 2: plan what to redraw ----

// tileSet is a set of map tiles.
type tileSet map[fileio.TilePos]bool

// withNeighbors adds each tile's in-bounds neighbors.
// planRedraw uses it twice: a changed tile's border and road depend on its neighbors, so those are repainted;
// and the tiles a dirty rect covers need one more ring around them, so the pixels near the rect's edge are drawn in full.
func withNeighbors(tiles tileSet, mapSize fileio.MapSize) tileSet {
	expanded := make(tileSet, len(tiles)*3)
	for rc := range tiles {
		expanded[rc] = true
		for _, n := range fileio.GetNeighbors(rc) {
			if n.InMap(mapSize) {
				expanded[n] = true
			}
		}
	}
	return expanded
}

// redrawPlan is what one incremental repaint touches, decided before anything is drawn.
type redrawPlan struct {
	dirtyRects  []image.Rectangle // the only pixels of the real canvas that change
	tilesToDraw []fileio.TilePos  // tiles to draw on the staging canvas, in row-major order
}

// planRedraw decides what repainting the changed tiles takes; it draws nothing.
func planRedraw(canvas Canvas, grid *tileGrid, changed tileChanges, labels []cityLabel) redrawPlan {
	mapSize := grid.mapSize
	canvasBounds := image.Rect(0, 0, grid.width, grid.height)

	// A tile's border and road depend on its neighbors, so those are repainted too.
	changedAndNeighbors := withNeighbors(changed.tiles(), mapSize)
	tileRects := tileRepaintRects(changedAndNeighbors, canvasBounds, grid.layout)
	// A city label may stick out past its tile, before or after the change.
	labelRects := changedLabelRects(canvas, grid.layout, changed, labels, canvasBounds)
	dirtyRects := append(tileRects, labelRects...)

	// Each dirty rect is a box larger than its hex, so it holds pixels of the tiles around it.
	// Those tiles are drawn as well, plus one ring beyond them, so every copied pixel is fully drawn.
	tilesUnderLabels := maps.Clone(changedAndNeighbors)
	for _, rect := range labelRects {
		maps.Copy(tilesUnderLabels, grid.tilesIn(rect))
	}
	tilesToDraw := sortedTiles(withNeighbors(tilesUnderLabels, mapSize))

	return redrawPlan{dirtyRects: dirtyRects, tilesToDraw: tilesToDraw}
}

// tileRepaintRects returns each tile's repaint rect, in row-major tile order.
func tileRepaintRects(tiles tileSet, canvasBounds image.Rectangle, l tileLayout) []image.Rectangle {
	sorted := sortedTiles(tiles)
	rects := make([]image.Rectangle, len(sorted))
	for i, rc := range sorted {
		rects[i] = l.repaintRect(rc, canvasBounds)
	}
	return rects
}

// changedLabelRects returns the rects of the changed tiles' labels as they are now and as they were before.
func changedLabelRects(canvas Canvas, layout tileLayout, changed tileChanges, labels []cityLabel, bounds image.Rectangle) []image.Rectangle {
	var rects []image.Rectangle
	for _, l := range labels {
		if _, ok := changed[l.pos]; ok && !l.rect.Empty() {
			rects = append(rects, l.rect)
		}
	}
	for _, rc := range sortedTiles(changed) {
		if name := trimCityName(changed[rc].CityName); name != "" {
			x, y := cityLabelPosition(layout, rc, name)
			if rect := labelRect(canvas, ColoredText{Text: name, X: x, Y: y}, bounds); !rect.Empty() {
				rects = append(rects, rect)
			}
		}
	}
	return rects
}

// sortedTiles returns the tiles that are keys of set, in row-major order.
func sortedTiles[V any](set map[fileio.TilePos]V) []fileio.TilePos {
	tiles := make([]fileio.TilePos, 0, len(set))
	for rc := range set {
		tiles = append(tiles, rc)
	}
	sort.Slice(tiles, func(a, b int) bool {
		if tiles[a].Row != tiles[b].Row {
			return tiles[a].Row < tiles[b].Row
		}
		return tiles[a].Col < tiles[b].Col
	})
	return tiles
}

// cityLabel is a city's label plus the rect its pixels occupy.
type cityLabel struct {
	pos  fileio.TilePos
	text ColoredText
	rect image.Rectangle
}

// cityLabels returns every city label in row-major draw order, with rects from real font metrics.
func (mr *MapRenderer) cityLabels(canvas Canvas, layout tileLayout, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, mainBounds image.Rectangle) []cityLabel {
	var labels []cityLabel
	for row := 0; row < mapSize.Height; row++ {
		for col := 0; col < mapSize.Width; col++ {
			pos := fileio.TilePos{Row: row, Col: col}
			if mapData.TileImprovement(pos).CityName == "" {
				continue
			}
			text := PoliticalCityNameLabel(mapData, pos, layout)
			if text.Text == "" {
				continue
			}
			labels = append(labels, cityLabel{pos, text, labelRect(canvas, text, mainBounds)})
		}
	}
	return labels
}

// labelRect returns the rect text occupies on canvas, from real font metrics, clipped to bounds.
func labelRect(canvas Canvas, text ColoredText, bounds image.Rectangle) image.Rectangle {
	const pad = 1.0 // rounding safety margin
	w, h := canvas.MeasureString(text.Text)
	return image.Rect(
		int(text.X-pad), int(text.Y-h-pad),
		int(text.X+w+pad)+1, int(text.Y+pad)+1,
	).Intersect(bounds)
}

// ---- Step 3: draw on a staging canvas, then copy the dirty rects back ----

// RedrawDirtyTiles repaints the changed tiles, their neighbors and their city labels (old and new) and returns every rect it may have changed.
// It draws on a staging canvas and copies only the dirty rects back, so the tiles drawn only to complete those rects never touch the real canvas.
func (mr *MapRenderer) RedrawDirtyTiles(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, changed tileChanges) []image.Rectangle {
	if len(changed) == 0 {
		return nil
	}
	grid := mr.tileGridFor(mapSize)
	labels := mr.cityLabels(canvas, grid.layout, mapData, mapSize, image.Rect(0, 0, grid.width, grid.height))
	plan := planRedraw(canvas, grid, changed, labels)

	// The staging canvas covers all dirty rects at once; tiles are drawn in real-canvas coordinates through its origin.
	stagingBounds := boundsOf(plan.dirtyRects)
	staging := canvas.NewSibling(stagingBounds.Dx(), stagingBounds.Dy())
	staging.SetOrigin(stagingBounds.Min)
	mr.paintTiles(staging, mapData, mapSize, plan.tilesToDraw)
	drawLabelsOver(staging, labels, plan.dirtyRects)
	copyRectsBack(canvas, staging.Image(), plan.dirtyRects, stagingBounds)

	return plan.dirtyRects
}

// copyRectsBack copies each rect from the staging image onto canvas.
func copyRectsBack(canvas Canvas, stagingImage image.Image, rects []image.Rectangle, stagingBounds image.Rectangle) {
	for _, rect := range rects {
		canvas.PasteRegion(stagingImage, rect, rect.Min.Sub(stagingBounds.Min))
	}
}

// boundsOf returns the smallest rect containing all of rects.
func boundsOf(rects []image.Rectangle) (bounds image.Rectangle) {
	for _, rect := range rects {
		bounds = bounds.Union(rect)
	}
	return bounds
}

// ---- Tile drawing: the first frame draws every tile with it, step 3 only the planned ones ----

// DrawPoliticalMapTileMajor renders the whole political map onto canvas, the first frame every later redraw builds on.
func (mr *MapRenderer) DrawPoliticalMapTileMajor(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData) image.Image {
	mapSize := mapData.Size()

	maxImageWidth, maxImageHeight := imageSize(mapSize, mr.config.Radius)
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapSize.Height, ", width: ", mapSize.Width)

	tiles := make([]fileio.TilePos, 0, mapSize.Height*mapSize.Width)
	for i := 0; i < mapSize.Height; i++ {
		for j := 0; j < mapSize.Width; j++ {
			tiles = append(tiles, fileio.TilePos{Row: i, Col: j})
		}
	}
	mr.paintTiles(canvas, mapData, mapSize, tiles)

	for _, l := range mr.cityLabels(canvas, mr.tileGridFor(mapSize).layout, mapData, mapSize, canvas.Image().Bounds()) {
		drawLabel(canvas, l)
	}

	return canvas.Image()
}

// paintTiles draws tiles layer by layer (fills, outlines, entities, borders, roads, rivers).
func (mr *MapRenderer) paintTiles(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, tiles []fileio.TilePos) {
	grid := mr.tileGridFor(mapSize)
	l := grid.layout
	for _, rc := range tiles {
		mr.drawTileFill(canvas, l, mapData, rc)
	}
	mr.drawTileOutlines(canvas, mapData, grid, tiles)
	for _, rc := range tiles {
		mr.drawTileEntities(canvas, l, mapData, rc)
	}
	mr.drawTileBorders(canvas, mapData, grid, tiles)
	for _, rc := range tiles {
		mr.drawTileRoads(canvas, l, mapData, mapSize, rc)
	}
	for _, rc := range tiles {
		mr.drawRiverTile(canvas, l, mapData, rc)
	}
}

// drawTileFill draws one tile's hex in its political (or terrain) color.
func (mr *MapRenderer) drawTileFill(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, rc fileio.TilePos) {
	hex, _ := PoliticalHexTile(mapData, rc, l)
	canvas.DrawRegularPolygon(6, hex.X, hex.Y, l.radius, math.Pi/2)
	canvas.SetColor(hex.R, hex.G, hex.B)
	canvas.Fill()
}

// drawTileEntities draws one tile's mountain and city markers.
func (mr *MapRenderer) drawTileEntities(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, rc fileio.TilePos) {
	_, cityColor := PoliticalHexTile(mapData, rc, l)
	for _, entity := range TileEntities(mapData, rc, l, cityColor) {
		mr.drawEntity(canvas, l, entity)
	}
}

// drawTileRoads draws the roads from one tile to its connected neighbors.
func (mr *MapRenderer) drawTileRoads(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, rc fileio.TilePos) {
	if len(mapData.MapTileImprovements) == 0 {
		return
	}
	for _, segment := range RoadSegmentsForTile(mapData, mapSize, rc, l) {
		canvas.SetLineWidth(segment.LineWidth)
		canvas.SetColor(segment.R, segment.G, segment.B)
		canvas.DrawLine(segment.Line.X1, segment.Line.Y1, segment.Line.X2, segment.Line.Y2)
		canvas.Stroke()
	}
}

// drawLabelsOver draws, in full-redraw order so overlapping labels stack the same way, the labels that touch any of rects.
func drawLabelsOver(canvas Canvas, labels []cityLabel, rects []image.Rectangle) {
	bounds := boundsOf(rects) // a cheap first cut before checking each rect
	for _, l := range labels {
		if l.rect.Overlaps(bounds) && slices.ContainsFunc(rects, l.rect.Overlaps) {
			drawLabel(canvas, l)
		}
	}
}

// drawLabel draws a city label in its color.
func drawLabel(canvas Canvas, l cityLabel) {
	canvas.SetColor(l.text.R, l.text.G, l.text.B)
	canvas.DrawString(l.text.Text, l.text.X, l.text.Y)
}

// ---- Step 4: turn the dirty rects into GIF frames ----

// mergeGap is the pixel distance under which mergeNearbyRects merges rects.
const mergeGap = 8

// mergeNearbyRects merges rects within mergeGap into disjoint bounding boxes, so scattered tiles don't become one huge box; empty rects are dropped.
// The result is the same for any input order, sorted by top-left corner (top to bottom, then left to right).
func mergeNearbyRects(rects []image.Rectangle) []image.Rectangle {
	var merged []image.Rectangle // never within mergeGap of each other
	pending := slices.Clone(rects)
	for len(pending) > 0 {
		r := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if r.Empty() {
			continue
		}
		near := func(m image.Rectangle) bool { return r.Inset(-mergeGap).Overlaps(m) }
		if i := slices.IndexFunc(merged, near); i >= 0 {
			pending = append(pending, r.Union(merged[i])) // the bigger box may now reach other merged rects
			merged = slices.Delete(merged, i, i+1)
			continue
		}
		merged = append(merged, r)
	}
	slices.SortFunc(merged, func(a, b image.Rectangle) int {
		return cmp.Or(cmp.Compare(a.Min.Y, b.Min.Y), cmp.Compare(a.Min.X, b.Min.X))
	})
	return merged
}

// noopRect is a 1x1 frame, which keeps a turn's delay when nothing on screen changed.
var noopRect = image.Rect(0, 0, 1, 1)

// gifFrameRects returns the rects to emit as GIF frames for dirtyRects: the merged rects, or noopRect if none is left.
func gifFrameRects(dirtyRects []image.Rectangle) []image.Rectangle {
	if frameRects := mergeNearbyRects(dirtyRects); len(frameRects) > 0 {
		return frameRects
	}
	return []image.Rectangle{noopRect}
}

// appendGifFrames repaints the changed tiles and appends one GIF block per frame rect (a 1x1 no-op if none); only the last has a delay.
func appendGifFrames(outGif *gif.GIF, renderer *MapRenderer, canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, changed tileChanges) {
	dirtyRects := renderer.RedrawDirtyTiles(canvas, mapData, mapSize, changed)

	frameRects := gifFrameRects(dirtyRects)
	for i, rect := range frameRects {
		delay := 0
		if i == len(frameRects)-1 {
			delay = GIF_DELAY
		}
		addFrame(outGif, canvas.Snapshot(rect), delay)
	}
}

// addFrame appends frame with the given delay; DisposalNone leaves it on screen under later frames, which only cover their own regions.
func addFrame(outGif *gif.GIF, frame *image.Paletted, delay int) {
	outGif.Image = append(outGif.Image, frame)
	outGif.Delay = append(outGif.Delay, delay)
	outGif.Disposal = append(outGif.Disposal, gif.DisposalNone)
}
