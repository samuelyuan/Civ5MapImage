package graphics

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// DrawingConfig holds configuration for map drawing
type DrawingConfig struct {
	Radius float64
}

// DefaultDrawingConfig returns the default drawing configuration
func DefaultDrawingConfig() *DrawingConfig {
	return &DrawingConfig{
		Radius: 16.0,
	}
}

// MapRenderer handles the rendering of Civ5 maps using the abstracted canvas
type MapRenderer struct {
	config *DrawingConfig
}

// NewMapRenderer creates a new map renderer with the given configuration
func NewMapRenderer(config *DrawingConfig) *MapRenderer {
	return &MapRenderer{
		config: config,
	}
}

// DrawMountain draws a mountain icon at the specified position
func (mr *MapRenderer) DrawMountain(canvas Canvas, imageX, imageY float64) {
	// Draw base
	canvas.DrawRegularPolygon(3, imageX, imageY, mr.config.Radius, math.Pi)
	canvas.SetColor(89, 90, 86) // gray
	canvas.Fill()

	// Draw mountain peak
	canvas.DrawRegularPolygon(3, imageX, imageY+(mr.config.Radius/2), mr.config.Radius/2, math.Pi)
	canvas.SetColor(234, 244, 253) // white
	canvas.Fill()
}

// GetNewCityColor returns a modified city color for better visibility
func (mr *MapRenderer) GetNewCityColor(cityColor color.RGBA) color.RGBA {
	return mr.InterpolateColor(cityColor, color.RGBA{255, 255, 255, 255}, 0.2)
}

// DrawCityIcon draws a city icon at the specified position
func (mr *MapRenderer) DrawCityIcon(canvas Canvas, imageX, imageY float64, cityColor color.RGBA) {
	iconColor := mr.GetNewCityColor(cityColor)
	canvas.DrawRectangle(imageX-(mr.config.Radius/5), imageY-(mr.config.Radius/5),
		mr.config.Radius/2, mr.config.Radius/2)
	canvas.SetColor(iconColor.R, iconColor.G, iconColor.B)
	canvas.Fill()
}

// drawEntity draws a single Entity using the shape appropriate to its Type.
func (mr *MapRenderer) drawEntity(canvas Canvas, entity Entity) {
	switch entity.Type {
	case EntityMountain:
		mr.DrawMountain(canvas, entity.X, entity.Y)
	case EntityCity:
		mr.DrawCityIcon(canvas, entity.X, entity.Y, color.RGBA{entity.R, entity.G, entity.B, 255})
	}
}

// DrawTerrainTiles draws all terrain tiles for the physical map
func (mr *MapRenderer) DrawTerrainTiles(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			hex := PhysicalHexTile(mapData, i, j, mr.config.Radius)
			canvas.DrawRegularPolygon(6, hex.X, hex.Y, mr.config.Radius, math.Pi/2)
			canvas.SetColor(hex.R, hex.G, hex.B)
			canvas.Fill()

			for _, entity := range TileEntities(mapData, i, j, mr.config.Radius, color.RGBA{255, 255, 255, 255}) {
				mr.drawEntity(canvas, entity)
			}
		}
	}
}

// InterpolateColor blends two colors by the given factor (0.0 = color1, 1.0 = color2).
func (mr *MapRenderer) InterpolateColor(color1, color2 color.RGBA, t float64) color.RGBA {
	return blendColor(color1, color2, t)
}

// DrawTerritoryTiles draws territory tiles for the political map
func (mr *MapRenderer) DrawTerritoryTiles(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			hex, cityColor := PoliticalHexTile(mapData, i, j, mr.config.Radius)
			canvas.DrawRegularPolygon(6, hex.X, hex.Y, mr.config.Radius, math.Pi/2)
			canvas.SetColor(hex.R, hex.G, hex.B)
			canvas.Fill()

			for _, entity := range TileEntities(mapData, i, j, mr.config.Radius, cityColor) {
				mr.drawEntity(canvas, entity)
			}
		}
	}
}

// drawRiverTile draws a single tile's river edges.
func (mr *MapRenderer) drawRiverTile(canvas Canvas, mapData *fileio.Civ5MapData, row, col int) {
	x, y := fileio.GetImagePosition(row, col, mr.config.Radius)
	canvas.SetColor(95, 150, 148)
	canvas.SetLineWidth(1.0)

	for _, edge := range RiverEdgesForTile(mapData.MapTiles[row][col].RiverData, x, y, mr.config.Radius) {
		canvas.DrawLine(edge.X1, edge.Y1, edge.X2, edge.Y2)
		canvas.Stroke()
	}
}

// DrawRivers draws rivers on the map
func (mr *MapRenderer) DrawRivers(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			mr.drawRiverTile(canvas, mapData, i, j)
		}
	}
}

// DrawRoads draws roads between tiles
func (mr *MapRenderer) DrawRoads(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			for _, segment := range RoadSegmentsForTile(mapData, mapHeight, mapWidth, i, j, mr.config.Radius) {
				canvas.SetLineWidth(segment.LineWidth)
				canvas.SetColor(segment.R, segment.G, segment.B)
				canvas.DrawLine(segment.Line.X1, segment.Line.Y1, segment.Line.X2, segment.Line.Y2)
				canvas.Stroke()
			}
		}
	}
}

// DrawPhysicalMap creates a physical map image using the abstracted canvas
func (mr *MapRenderer) DrawPhysicalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, mr.config.Radius)

	// Resize canvas to fit the map
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	canvas.InvertY()

	mr.DrawTerrainTiles(canvas, mapData, mapHeight, mapWidth)
	mr.DrawRivers(canvas, mapData, mapHeight, mapWidth)
	mr.DrawRoads(canvas, mapData, mapHeight, mapWidth)

	// Draw city names on top of hexes
	canvas.InvertY()
	mr.DrawPhysicalCityNames(canvas, mapData, mapHeight, mapWidth)

	return canvas.Image()
}

// DrawBorders draws borders between different territories
func (mr *MapRenderer) DrawBorders(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			for _, segment := range BorderSegmentsForTile(mapData, mapHeight, mapWidth, i, j, mr.config.Radius) {
				canvas.SetColor(segment.R, segment.G, segment.B)
				canvas.SetLineWidth(segment.LineWidth)
				canvas.DrawLine(segment.Line.X1, segment.Line.Y1, segment.Line.X2, segment.Line.Y2)
				canvas.Stroke()
			}
		}
	}
}

// DrawPhysicalCityNames draws city names on the map (white text for physical maps)
func (mr *MapRenderer) DrawPhysicalCityNames(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			label := PhysicalCityNameLabel(mapData, mapHeight, mapWidth, i, j, mr.config.Radius)
			canvas.SetColor(label.R, label.G, label.B)
			canvas.DrawString(label.Text, label.X, label.Y)
		}
	}
}

// drawPoliticalCityNameTile draws a single tile's city name label, in political colors.
func (mr *MapRenderer) drawPoliticalCityNameTile(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int) {
	// Skip the label build and DrawString for the common no-city tile.
	if mapData.MapTileImprovements[row][col].CityName == "" {
		return
	}
	label := PoliticalCityNameLabel(mapData, mapHeight, mapWidth, row, col, mr.config.Radius)
	if label.Text == "" {
		return
	}
	canvas.SetColor(label.R, label.G, label.B)
	canvas.DrawString(label.Text, label.X, label.Y)
}

// DrawPoliticalCityNames draws city names with political colors
func (mr *MapRenderer) DrawPoliticalCityNames(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			mr.drawPoliticalCityNameTile(canvas, mapData, mapHeight, mapWidth, i, j)
		}
	}
}

// DrawPoliticalMap creates a political map image using the abstracted canvas
func (mr *MapRenderer) DrawPoliticalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, mr.config.Radius)

	// Resize canvas to fit the map
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	canvas.InvertY()

	mr.DrawTerritoryTiles(canvas, mapData, mapHeight, mapWidth)
	mr.DrawBorders(canvas, mapData, mapHeight, mapWidth)
	mr.DrawRivers(canvas, mapData, mapHeight, mapWidth)
	mr.DrawRoads(canvas, mapData, mapHeight, mapWidth)

	canvas.InvertY()
	// Draw city names on top of hexes
	mr.DrawPoliticalCityNames(canvas, mapData, mapHeight, mapWidth)

	return canvas.Image()
}

// drawPoliticalTile draws one tile's fill, entities, border and road, all of which stay within its hex.
// Rivers are excluded: their stroke straddles the shared edge, so they get their own pass after every fill.
func (mr *MapRenderer) drawPoliticalTile(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int) {
	hex, cityColor := PoliticalHexTile(mapData, row, col, mr.config.Radius)
	canvas.DrawRegularPolygon(6, hex.X, hex.Y, mr.config.Radius, math.Pi/2)
	canvas.SetColor(hex.R, hex.G, hex.B)
	canvas.Fill()

	for _, entity := range TileEntities(mapData, row, col, mr.config.Radius, cityColor) {
		mr.drawEntity(canvas, entity)
	}

	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for _, segment := range BorderSegmentsForTile(mapData, mapHeight, mapWidth, row, col, mr.config.Radius) {
		canvas.SetColor(segment.R, segment.G, segment.B)
		canvas.SetLineWidth(segment.LineWidth)
		canvas.DrawLine(segment.Line.X1, segment.Line.Y1, segment.Line.X2, segment.Line.Y2)
		canvas.Stroke()
	}
	for _, segment := range RoadSegmentsForTile(mapData, mapHeight, mapWidth, row, col, mr.config.Radius) {
		canvas.SetLineWidth(segment.LineWidth)
		canvas.SetColor(segment.R, segment.G, segment.B)
		canvas.DrawLine(segment.Line.X1, segment.Line.Y1, segment.Line.X2, segment.Line.Y2)
		canvas.Stroke()
	}
}

// DrawPoliticalMapTileMajor renders DrawPoliticalMap tile by tile, the unit RedrawDirtyTiles works in;
// anti-aliasing order differs slightly (max channel delta 60).
func (mr *MapRenderer) DrawPoliticalMapTileMajor(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, mr.config.Radius)
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	canvas.InvertY()
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			mr.drawPoliticalTile(canvas, mapData, mapHeight, mapWidth, i, j)
		}
	}
	mr.DrawRivers(canvas, mapData, mapHeight, mapWidth)

	canvas.InvertY()
	mr.DrawPoliticalCityNames(canvas, mapData, mapHeight, mapWidth)

	return canvas.Image()
}

// tileCoord is a map tile's position; tileSet is a set of them.
type tileCoord struct{ row, col int }
type tileSet map[tileCoord]bool

// RedrawDirtyTiles repaints paintSet tiles (fill, entities, border, road, river) without touching
// the rest of the canvas. paintSet must include each dirty tile's neighbors.
//
// Draws into a blank scratch canvas sized to paintSet's bounding box and pastes the tiles back:
// gg blends anti-aliasing against existing pixels, so repainting in place would drift at shared
// edges.
//
// Returns every rect it may have changed; everything outside is identical to the previous frame.
func (mr *MapRenderer) RedrawDirtyTiles(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, paintSet tileSet) []image.Rectangle {
	if len(paintSet) == 0 {
		return nil
	}

	maxImageWidth, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, mr.config.Radius)
	mainBounds := image.Rect(0, 0, int(maxImageWidth), int(maxImageHeight))
	radius := mr.config.Radius

	paintTiles, scratchBounds := paintTileRects(paintSet, mainBounds, maxImageHeight, radius)
	contextTiles := sortedContextTiles(paintSet, mapHeight, mapWidth)
	scratchImage := mr.paintTilesToScratch(mapData, mapHeight, mapWidth, contextTiles, scratchBounds, mainBounds.Dy())
	dirtyRects := pasteBack(canvas, scratchImage, paintTiles, scratchBounds)

	dirtyRects = append(dirtyRects, mr.restampLabels(canvas, mapData, mapHeight, mapWidth, paintSet, dirtyRects, mainBounds)...)

	return dirtyRects
}

// paintTileRect pairs a paint-set tile with the rect on the main canvas its own repaint occupies.
type paintTileRect struct {
	rc   tileCoord
	rect image.Rectangle
}

// paintTileRects returns each paint tile's rect (hex radius plus stroke padding) and their union.
func paintTileRects(paintSet tileSet, mainBounds image.Rectangle, maxImageHeight, radius float64) (tiles []paintTileRect, union image.Rectangle) {
	const pad = 2.0 // safety margin for river/border stroke half-widths right at a tile's radius
	tiles = make([]paintTileRect, 0, len(paintSet))
	for rc := range paintSet {
		x, y := fileio.GetImagePosition(rc.row, rc.col, radius)
		finalX, finalY := x, maxImageHeight-y
		rect := image.Rect(
			int(finalX-radius-pad), int(finalY-radius-pad),
			int(finalX+radius+pad)+1, int(finalY+radius+pad)+1,
		).Intersect(mainBounds)
		tiles = append(tiles, paintTileRect{rc, rect})
		if union.Empty() {
			union = rect
		} else {
			union = union.Union(rect)
		}
	}
	return tiles, union
}

// sortedContextTiles returns paintSet plus one more neighbor ring, in row-major order. The ring is
// drawn so edge tiles blend against real neighbors, but never pasted back.
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

// paintTilesToScratch draws contextTiles into a blank canvas of scratchBounds' size, in the main
// canvas's coordinates shifted so scratchBounds.Min is (0, 0).
func (mr *MapRenderer) paintTilesToScratch(mapData *fileio.Civ5MapData, mapHeight, mapWidth int, contextTiles []tileCoord, scratchBounds image.Rectangle, mainCanvasHeight int) image.Image {
	scratch := NewDrawingContext(scratchBounds.Dx(), scratchBounds.Dy())
	// Use the main canvas's integer pixel height: its own InvertY() flips against that, not a raw float height.
	scratch.TranslatedInvertY(-float64(scratchBounds.Min.X), float64(mainCanvasHeight)-float64(scratchBounds.Min.Y))
	for _, rc := range contextTiles {
		mr.drawPoliticalTile(scratch, mapData, mapHeight, mapWidth, rc.row, rc.col)
	}
	for _, rc := range contextTiles {
		mr.drawRiverTile(scratch, mapData, rc.row, rc.col)
	}
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
func (mr *MapRenderer) cityLabels(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, mainBounds image.Rectangle) []cityLabel {
	const labelPad = 1.0 // rounding safety margin
	var labels []cityLabel
	for row := 0; row < mapHeight; row++ {
		for col := 0; col < mapWidth; col++ {
			if mapData.MapTileImprovements[row][col].CityName == "" {
				continue
			}
			text := PoliticalCityNameLabel(mapData, mapHeight, mapWidth, row, col, mr.config.Radius)
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

// restampLabels redraws labels a paste could have erased (those overlapping a pasted rect, or whose
// tile changed) plus any label overlapping one of those, in order so stacking matches a full
// redraw. Returns the rects of labels whose own tile changed. Re-stamping is idempotent because
// gg's default font isn't anti-aliased.
func (mr *MapRenderer) restampLabels(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, paintSet tileSet, pasted []image.Rectangle, mainBounds image.Rectangle) (changed []image.Rectangle) {
	labels := mr.cityLabels(canvas, mapData, mapHeight, mapWidth, mainBounds)
	restamp := make([]bool, len(labels))
	for i, l := range labels {
		if paintSet[tileCoord{l.row, l.col}] {
			restamp[i] = true
			changed = append(changed, l.rect)
			continue
		}
		for _, p := range pasted {
			if l.rect.Overlaps(p) {
				restamp[i] = true
				break
			}
		}
	}
	for grew := true; grew; {
		grew = false
		for i := range labels {
			if restamp[i] {
				continue
			}
			for j := range labels {
				if restamp[j] && labels[i].rect.Overlaps(labels[j].rect) {
					restamp[i], grew = true, true
					break
				}
			}
		}
	}
	for i, l := range labels {
		if restamp[i] {
			canvas.SetColor(l.text.R, l.text.G, l.text.B)
			canvas.DrawString(l.text.Text, l.text.X, l.text.Y)
		}
	}
	return changed
}

// SaveImage saves the image to a file
func (mr *MapRenderer) SaveImage(canvas Canvas, outputFilename string) error {
	return canvas.SavePNG(outputFilename)
}
