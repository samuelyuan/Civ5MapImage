package graphics

import (
	"fmt"
	"image"
	"math"
	"slices"
	"sort"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// drawTileFill draws one tile's hex in its political (or terrain) color.
func (mr *MapRenderer) drawTileFill(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, rc tileCoord) {
	hex, _ := PoliticalHexTile(mapData, rc.row, rc.col, l)
	canvas.DrawRegularPolygon(6, hex.X, hex.Y, l.radius, math.Pi/2)
	canvas.SetColor(hex.R, hex.G, hex.B)
	canvas.Fill()
}

// drawTileEntities draws one tile's mountain and city markers.
func (mr *MapRenderer) drawTileEntities(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, rc tileCoord) {
	_, cityColor := PoliticalHexTile(mapData, rc.row, rc.col, l)
	for _, entity := range TileEntities(mapData, rc.row, rc.col, l, cityColor) {
		mr.drawEntity(canvas, l, entity)
	}
}

// drawTileRoads draws the roads from one tile to its connected neighbors.
func (mr *MapRenderer) drawTileRoads(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, rc tileCoord) {
	if len(mapData.MapTileImprovements) == 0 {
		return
	}
	for _, segment := range RoadSegmentsForTile(mapData, mapHeight, mapWidth, rc.row, rc.col, l) {
		canvas.SetLineWidth(segment.LineWidth)
		canvas.SetColor(segment.R, segment.G, segment.B)
		canvas.DrawLine(segment.Line.X1, segment.Line.Y1, segment.Line.X2, segment.Line.Y2)
		canvas.Stroke()
	}
}

// paintTiles draws tiles layer by layer (fills, outlines, entities, borders, roads, rivers).
func (mr *MapRenderer) paintTiles(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, tiles []tileCoord) {
	grid := mr.tileGridFor(mapHeight, mapWidth)
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
		mr.drawTileRoads(canvas, l, mapData, mapHeight, mapWidth, rc)
	}
	for _, rc := range tiles {
		mr.drawRiverTile(canvas, l, mapData, rc.row, rc.col)
	}
}

// DrawPoliticalMapTileMajor renders the political map onto canvas, the unit RedrawDirtyTiles works in.
func (mr *MapRenderer) DrawPoliticalMapTileMajor(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := GetImagePosition(mapHeight, mapWidth, mr.config.Radius)
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	tiles := make([]tileCoord, 0, mapHeight*mapWidth)
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			tiles = append(tiles, tileCoord{i, j})
		}
	}
	mr.paintTiles(canvas, mapData, mapHeight, mapWidth, tiles)

	mr.DrawPoliticalCityNames(canvas, mapData, mapHeight, mapWidth, mr.tileGridFor(mapHeight, mapWidth).layout)

	return canvas.Image()
}

// tileCoord is a map tile's position; tileSet is a set of them.
type tileCoord struct{ row, col int }

type tileSet map[tileCoord]bool

// expandWithNeighbors adds each tile's in-bounds neighbors, since a tile's border and road depend on them.
func expandWithNeighbors(tiles tileSet, mapHeight, mapWidth int) tileSet {
	expanded := make(tileSet, len(tiles)*3)
	for rc := range tiles {
		expanded[rc] = true
		for _, n := range fileio.GetNeighbors(rc.col, rc.row) {
			nx, ny := n[0], n[1]
			if nx >= 0 && ny >= 0 && nx < mapWidth && ny < mapHeight {
				expanded[tileCoord{ny, nx}] = true
			}
		}
	}
	return expanded
}

// RedrawDirtyTiles repaints the changed tiles and their neighbors via a scratch canvas (with one context ring)
// pasted back, and returns every rect it may have changed.
func (mr *MapRenderer) RedrawDirtyTiles(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, changed tileSet) []image.Rectangle {
	if len(changed) == 0 {
		return nil
	}
	paintSet := expandWithNeighbors(changed, mapHeight, mapWidth)

	grid := mr.tileGridFor(mapHeight, mapWidth)
	mainBounds := image.Rect(0, 0, grid.width, grid.height)

	paintTiles, scratchBounds := paintTileRects(paintSet, mainBounds, grid.layout)
	contextTiles := sortedContextTiles(paintSet, mapHeight, mapWidth)
	scratchImage := mr.paintTilesToScratch(canvas, mapData, mapHeight, mapWidth, contextTiles, scratchBounds)
	dirtyRects := pasteBack(canvas, scratchImage, paintTiles, scratchBounds)

	dirtyRects = append(dirtyRects, mr.restampLabels(canvas, grid.layout, mapData, mapHeight, mapWidth, paintSet, dirtyRects, mainBounds)...)

	return dirtyRects
}

// paintTileRect pairs a paint-set tile with the rect on the main canvas its own repaint occupies.
type paintTileRect struct {
	rc   tileCoord
	rect image.Rectangle
}

// paintTileRects returns each paint tile's rect (hex radius plus stroke padding) and their union, in row-major order.
func paintTileRects(paintSet tileSet, mainBounds image.Rectangle, l tileLayout) (tiles []paintTileRect, union image.Rectangle) {
	const pad = 2.0 // safety margin for river/border stroke half-widths right at a tile's radius
	tiles = make([]paintTileRect, 0, len(paintSet))
	for rc := range paintSet {
		x, y := l.center(rc.row, rc.col)
		rect := image.Rect(
			int(x-l.radius-pad), int(y-l.radius-pad),
			int(x+l.radius+pad)+1, int(y+l.radius+pad)+1,
		).Intersect(mainBounds)
		tiles = append(tiles, paintTileRect{rc, rect})
		if union.Empty() {
			union = rect
		} else {
			union = union.Union(rect)
		}
	}
	sort.Slice(tiles, func(a, b int) bool {
		if tiles[a].rc.row != tiles[b].rc.row {
			return tiles[a].rc.row < tiles[b].rc.row
		}
		return tiles[a].rc.col < tiles[b].rc.col
	})
	return tiles, union
}

// sortedContextTiles returns paintSet plus one more neighbor ring (drawn but never pasted back), in row-major order.
func sortedContextTiles(paintSet tileSet, mapHeight, mapWidth int) []tileCoord {
	contextSet := expandWithNeighbors(paintSet, mapHeight, mapWidth)
	tiles := make([]tileCoord, 0, len(contextSet))
	for rc := range contextSet {
		tiles = append(tiles, rc)
	}
	sort.Slice(tiles, func(a, b int) bool {
		if tiles[a].row != tiles[b].row {
			return tiles[a].row < tiles[b].row
		}
		return tiles[a].col < tiles[b].col
	})
	return tiles
}

// paintTilesToScratch draws contextTiles into a blank canvas covering scratchBounds.
func (mr *MapRenderer) paintTilesToScratch(like *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, contextTiles []tileCoord, scratchBounds image.Rectangle) image.Image {
	scratch := like.NewSibling(scratchBounds.Dx(), scratchBounds.Dy())
	scratch.SetOrigin(scratchBounds.Min)
	mr.paintTiles(scratch, mapData, mapHeight, mapWidth, contextTiles)
	return scratch.Image()
}

// pasteBack copies each paint tile's rect from scratchImage onto canvas and returns those rects.
func pasteBack(canvas Canvas, scratchImage image.Image, paintTiles []paintTileRect, scratchBounds image.Rectangle) []image.Rectangle {
	dirtyRects := make([]image.Rectangle, 0, len(paintTiles))
	for _, pt := range paintTiles {
		srcPoint := pt.rect.Min.Sub(scratchBounds.Min)
		canvas.PasteRegion(scratchImage, pt.rect, srcPoint)
		dirtyRects = append(dirtyRects, pt.rect)
	}
	return dirtyRects
}

// cityLabel is a city's label plus the rect its pixels occupy.
type cityLabel struct {
	row, col int
	text     ColoredText
	rect     image.Rectangle
}

// cityLabels returns every city label in DrawPoliticalCityNames's order, with rects from real font metrics.
func (mr *MapRenderer) cityLabels(canvas Canvas, layout tileLayout, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, mainBounds image.Rectangle) []cityLabel {
	const labelPad = 1.0 // rounding safety margin
	var labels []cityLabel
	for row := 0; row < mapHeight; row++ {
		for col := 0; col < mapWidth; col++ {
			if mapData.MapTileImprovements[row][col].CityName == "" {
				continue
			}
			text := PoliticalCityNameLabel(mapData, mapHeight, mapWidth, row, col, layout)
			if text.Text == "" {
				continue
			}
			w, h := canvas.MeasureString(text.Text)
			rect := image.Rect(
				int(text.X-labelPad), int(text.Y-h-labelPad),
				int(text.X+w+labelPad)+1, int(text.Y+labelPad)+1,
			).Intersect(mainBounds)
			labels = append(labels, cityLabel{row, col, text, rect})
		}
	}
	return labels
}

// restampLabels redraws labels a paste could have erased, plus any overlapping them, in full-redraw order, and returns the rects of changed ones.
func (mr *MapRenderer) restampLabels(canvas Canvas, layout tileLayout, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, paintSet tileSet, pasted []image.Rectangle, mainBounds image.Rectangle) (changed []image.Rectangle) {
	labels := mr.cityLabels(canvas, layout, mapData, mapHeight, mapWidth, mainBounds)
	restamp, changed := labelsToRestamp(labels, paintSet, pasted)
	for i, l := range labels {
		if restamp[i] {
			canvas.SetColor(l.text.R, l.text.G, l.text.B)
			canvas.DrawString(l.text.Text, l.text.X, l.text.Y)
		}
	}
	return changed
}

// labelsToRestamp marks the labels to redraw: those on painted tiles or under a pasted rect, then, transitively, any overlapping a marked one, since draw order decides which shows on top.
// changed holds the rects of the labels on painted tiles.
func labelsToRestamp(labels []cityLabel, paintSet tileSet, pasted []image.Rectangle) (restamp []bool, changed []image.Rectangle) {
	restamp = make([]bool, len(labels))
	var queue []int // marked labels whose overlaps haven't been checked yet
	mark := func(i int) {
		restamp[i] = true
		queue = append(queue, i)
	}
	for i, l := range labels {
		if paintSet[tileCoord{l.row, l.col}] {
			mark(i)
			changed = append(changed, l.rect)
		} else if slices.ContainsFunc(pasted, l.rect.Overlaps) {
			mark(i)
		}
	}
	for ; len(queue) > 0; queue = queue[1:] {
		marked := labels[queue[0]]
		for i, l := range labels {
			if !restamp[i] && l.rect.Overlaps(marked.rect) {
				mark(i)
			}
		}
	}
	return restamp, changed
}
