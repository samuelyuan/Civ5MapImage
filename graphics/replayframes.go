package graphics

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// ---- Step 1: find the changed tiles ----

// tileTracker reports which tiles' records changed since it last looked.
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

// takeChanges returns the tiles changed since the last call, with their earlier records, and makes the current state the baseline.
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

// allTiles returns every tile of a map of this size, in row-major order.
func allTiles(mapSize fileio.MapSize) []fileio.TilePos {
	tiles := make([]fileio.TilePos, 0, mapSize.Height*mapSize.Width)
	for row := 0; row < mapSize.Height; row++ {
		for col := 0; col < mapSize.Width; col++ {
			tiles = append(tiles, fileio.TilePos{Row: row, Col: col})
		}
	}
	return tiles
}

// tileSet is a set of map tiles.
type tileSet map[fileio.TilePos]bool

// withNeighbors adds each tile's in-bounds neighbors.
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
func planRedraw(canvas TextDrawer, grid *tileGrid, changed tileChanges, labels []cityLabel) redrawPlan {
	mapSize := grid.mapSize
	canvasBounds := image.Rect(0, 0, grid.width, grid.height)

	// A tile's border and road depend on its neighbors, so those are repainted too.
	changedAndNeighbors := withNeighbors(changed.tiles(), mapSize)
	tileRects := tileRepaintRects(changedAndNeighbors, canvasBounds, grid.layout)
	// A city label may stick out past its tile, before or after the change.
	labelRects := changedLabelRects(canvas, grid.layout, changed, labels, canvasBounds)
	dirtyRects := append(tileRects, labelRects...)

	// Dirty rects hold pixels of surrounding tiles, so those are drawn too, plus one ring, so every copied pixel is complete.
	tilesUnderLabels := maps.Clone(changedAndNeighbors)
	for _, rect := range labelRects {
		maps.Copy(tilesUnderLabels, grid.tilesIn(rect))
	}
	tilesToDraw := sortedTiles(withNeighbors(tilesUnderLabels, mapSize))

	return redrawPlan{dirtyRects: dirtyRects, tilesToDraw: tilesToDraw}
}

// repaintRectPad is the margin around a hex for river and border strokes at its radius.
const repaintRectPad = 2.0

// repaintRect returns the pixel rect that repainting the tile at pos can touch, clipped to bounds.
func (l tileLayout) repaintRect(pos fileio.TilePos, bounds image.Rectangle) image.Rectangle {
	x, y := l.center(pos)
	return image.Rect(
		int(x-l.radius-repaintRectPad), int(y-l.radius-repaintRectPad),
		int(x+l.radius+repaintRectPad)+1, int(y+l.radius+repaintRectPad)+1,
	).Intersect(bounds)
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
func changedLabelRects(canvas TextDrawer, layout tileLayout, changed tileChanges, labels []cityLabel, bounds image.Rectangle) []image.Rectangle {
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
func (mr *MapRenderer) cityLabels(canvas TextDrawer, layout tileLayout, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, mainBounds image.Rectangle) []cityLabel {
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
func labelRect(canvas TextDrawer, text ColoredText, bounds image.Rectangle) image.Rectangle {
	const pad = 1.0 // rounding safety margin
	w, h := canvas.MeasureString(text.Text)
	return image.Rect(
		int(text.X-pad), int(text.Y-h-pad),
		int(text.X+w+pad)+1, int(text.Y+pad)+1,
	).Intersect(bounds)
}

// ---- Step 3: draw on a staging canvas, then copy the dirty rects back ----

// RedrawDirtyTiles repaints the changed tiles, their neighbors and their labels (old and new) on a staging canvas, copies only the dirty rects back, and returns them.
func (mr *MapRenderer) RedrawDirtyTiles(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, changed tileChanges) []image.Rectangle {
	if len(changed) == 0 {
		return nil
	}
	grid := mr.tileGridFor(mapSize)
	labels := mr.cityLabels(canvas, grid.layout, mapData, mapSize, image.Rect(0, 0, grid.width, grid.height))
	plan := planRedraw(canvas, grid, changed, labels)

	// The staging canvas covers all dirty rects and draws in real-canvas coordinates via its origin.
	stagingBounds := raster.BoundsOf(plan.dirtyRects)
	staging := canvas.NewSibling(stagingBounds.Dx(), stagingBounds.Dy())
	staging.SetOrigin(stagingBounds.Min)
	mr.paintTiles(staging, mapData, mapSize, plan.tilesToDraw)
	drawLabelsOver(staging, labels, plan.dirtyRects)
	copyRectsBack(canvas, staging.Image(), plan.dirtyRects, stagingBounds)

	return plan.dirtyRects
}

// copyRectsBack copies each rect from the staging image onto canvas.
func copyRectsBack(canvas *raster.PalettedCanvas, stagingImage image.Image, rects []image.Rectangle, stagingBounds image.Rectangle) {
	for _, rect := range rects {
		canvas.PasteRegion(stagingImage, rect, rect.Min.Sub(stagingBounds.Min))
	}
}

// ---- Tile drawing: the first frame draws every tile with it, step 3 only the planned ones ----

// DrawPoliticalMapTileMajor renders the whole political map: the first frame later redraws build on.
func (mr *MapRenderer) DrawPoliticalMapTileMajor(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData) image.Image {
	mapSize := mapData.Size()

	maxImageWidth, maxImageHeight := imageSize(mapSize, mr.config.Radius)
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapSize.Height, ", width: ", mapSize.Width)

	mr.paintTiles(canvas, mapData, mapSize, allTiles(mapSize))

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
		mr.drawRiverTile(canvas, l, mapData, rc, riverColor, 1.0)
	}
}

// drawTileFill draws one tile's hex in its political (or terrain) color.
func (mr *MapRenderer) drawTileFill(canvas Painter, l tileLayout, mapData *fileio.Civ5MapData, rc fileio.TilePos) {
	hex, _ := PoliticalHexTile(mapData, rc, l)
	fillHex(canvas, hex, l.radius)
}

// drawTileRoads draws the roads from one tile to its connected neighbors.
func (mr *MapRenderer) drawTileRoads(canvas Painter, l tileLayout, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, rc fileio.TilePos) {
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

// drawLabelsOver draws the labels touching rects, in full-redraw order so overlaps stack the same.
func drawLabelsOver(canvas TextPainter, labels []cityLabel, rects []image.Rectangle) {
	bounds := raster.BoundsOf(rects) // a cheap first cut before checking each rect
	for _, l := range labels {
		if l.rect.Overlaps(bounds) && slices.ContainsFunc(rects, l.rect.Overlaps) {
			drawLabel(canvas, l)
		}
	}
}

// drawLabel draws a city label in its color.
func drawLabel(canvas TextPainter, l cityLabel) {
	canvas.SetColor(l.text.R, l.text.G, l.text.B)
	canvas.DrawString(l.text.Text, l.text.X, l.text.Y)
}

// ---- Step 4: turn the dirty rects into GIF frames ----

// mergeGap is the pixel distance under which frame rects are merged.
const mergeGap = 8

// noopRect is a 1x1 frame, which keeps a turn's delay when nothing on screen changed.
var noopRect = image.Rect(0, 0, 1, 1)

// gifFrameRects returns dirtyRects merged within mergeGap, or noopRect if none is left.
func gifFrameRects(dirtyRects []image.Rectangle) []image.Rectangle {
	if frameRects := raster.MergeNearbyRects(dirtyRects, mergeGap); len(frameRects) > 0 {
		return frameRects
	}
	return []image.Rectangle{noopRect}
}

// appendGifFrames repaints the changes and appends one GIF block per frame rect (a 1x1 no-op if none); only the last has a delay.
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

// addFrame appends frame with the delay; DisposalNone leaves earlier frames under it.
func addFrame(outGif *gif.GIF, frame *image.Paletted, delay int) {
	outGif.Image = append(outGif.Image, frame)
	outGif.Delay = append(outGif.Delay, delay)
	outGif.Disposal = append(outGif.Disposal, gif.DisposalNone)
}

// EntityType identifies what a map marker represents.
type EntityType string

const (
	EntityMountain EntityType = "mountain"
	EntityCity     EntityType = "city"
)

// Entity is a marker to draw at a tile position.
type Entity struct {
	Type    EntityType
	X, Y    float64
	R, G, B uint8
}

// TileEntities returns the mountain/city markers of the tile at pos (cityColor for a city), or nil.
func TileEntities(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout, cityColor color.RGBA) []Entity {
	x, y := l.center(pos)

	var entities []Entity
	if fileio.TileHasMountain(mapData, pos) {
		entities = append(entities, Entity{Type: EntityMountain, X: x, Y: y})
	}
	if fileio.TileHasCity(mapData, pos) {
		entities = append(entities, Entity{Type: EntityCity, X: x, Y: y, R: cityColor.R, G: cityColor.G, B: cityColor.B})
	}
	return entities
}

// DrawMountain draws a mountain icon at (imageX, imageY).
func (mr *MapRenderer) DrawMountain(canvas Painter, l tileLayout, imageX, imageY float64) {
	// Draw base
	canvas.DrawRegularPolygon(3, imageX, imageY, l.radius, 0)
	canvas.SetColor(mountainBaseColor.R, mountainBaseColor.G, mountainBaseColor.B)
	canvas.Fill()

	// Draw mountain peak
	canvas.DrawRegularPolygon(3, imageX, imageY-l.radius/2, l.radius/2, 0)
	canvas.SetColor(mountainPeakColor.R, mountainPeakColor.G, mountainPeakColor.B)
	canvas.Fill()
}

// DrawCityIcon draws a city icon at (imageX, imageY).
func (mr *MapRenderer) DrawCityIcon(canvas Painter, l tileLayout, imageX, imageY float64, cityColor color.RGBA) {
	iconColor := markerColor(cityColor)
	// The icon reaches 0.3r above the tile's center and 0.2r below it.
	canvas.DrawRectangle(imageX-(l.radius/5), imageY-l.radius*3/10, l.radius/2, l.radius/2)
	canvas.SetColor(iconColor.R, iconColor.G, iconColor.B)
	canvas.Fill()
}

// drawEntity draws entity in the shape for its type.
func (mr *MapRenderer) drawEntity(canvas Painter, l tileLayout, entity Entity) {
	switch entity.Type {
	case EntityMountain:
		mr.DrawMountain(canvas, l, entity.X, entity.Y)
	case EntityCity:
		mr.DrawCityIcon(canvas, l, entity.X, entity.Y, color.RGBA{entity.R, entity.G, entity.B, 255})
	}
}

// drawTileEntities draws one tile's mountain and city markers.
func (mr *MapRenderer) drawTileEntities(canvas Painter, l tileLayout, mapData *fileio.Civ5MapData, rc fileio.TilePos) {
	_, cityColor := PoliticalHexTile(mapData, rc, l)
	for _, entity := range TileEntities(mapData, rc, l, cityColor) {
		mr.drawEntity(canvas, l, entity)
	}
}

// terrainColors are the physical map's terrain colors.
var terrainColors = map[string]color.RGBA{
	"TERRAIN_GRASS":  {105, 125, 54, 255},
	"TERRAIN_PLAINS": {127, 121, 71, 255},
	"TERRAIN_DESERT": {200, 200, 164, 255},
	"TERRAIN_TUNDRA": {118, 123, 117, 255},
	"TERRAIN_SNOW":   {238, 249, 255, 255},
	"TERRAIN_COAST":  {95, 149, 149, 255},
	"TERRAIN_OCEAN":  {47, 74, 93, 255},
}

// unknownTerrainColor is what a terrain missing from terrainColors draws as.
var unknownTerrainColor = color.RGBA{0, 0, 0, 255}

// GetPhysicalMapTileColor returns the color of a terrain, or unknownTerrainColor if it has none.
func GetPhysicalMapTileColor(terrain string) color.RGBA {
	if c, ok := terrainColors[terrain]; ok {
		return c
	}
	return unknownTerrainColor
}

// replayPinnedColors lists every flat color the renderer draws.
func replayPinnedColors(mapData *fileio.Civ5MapData) []color.RGBA {
	white := color.RGBA{255, 255, 255, 255}
	colors := []color.RGBA{
		{0, 0, 0, 255}, white,
		mountainBaseColor, mountainPeakColor, riverColor, railroadColor, roadColor, unknownRouteColor,
	}
	terrainShades := []color.RGBA{unknownTerrainColor}
	for _, terrain := range slices.Sorted(maps.Keys(terrainColors)) {
		terrainShades = append(terrainShades, terrainColors[terrain])
	}
	for _, c := range terrainShades {
		colors = append(colors, c, tileOutlineColor(c))
	}
	for _, player := range mapData.Civ5PlayerData {
		civColor, ok := civColorMap[player.TeamColor]
		if !ok {
			continue
		}
		shades := shadesFor(civColor, strings.Contains(player.CivType, "MINOR"))
		colors = append(colors, shades.fill, shades.border, markerColor(shades.border), tileOutlineColor(shades.fill))
	}
	return colors
}

// replayPalette is every color the replay renderer can draw, deduplicated, as an exact GIF palette.
func replayPalette(mapData *fileio.Civ5MapData) color.Palette {
	seen := map[color.RGBA]bool{}
	var palette color.Palette
	for _, c := range replayPinnedColors(mapData) {
		if !seen[c] {
			seen[c] = true
			palette = append(palette, c)
		}
	}
	return palette
}
