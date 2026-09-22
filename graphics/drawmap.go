package graphics

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// DrawingConfig holds map drawing settings.
type DrawingConfig struct {
	Radius float64
}

// DefaultDrawingConfig returns the default settings.
func DefaultDrawingConfig() *DrawingConfig {
	return &DrawingConfig{
		Radius: 16.0,
	}
}

// Fixed (non-civ) colors the renderer draws with; replayPinnedColors lists them for the GIF palette.
var (
	mountainBaseColor = color.RGBA{89, 90, 86, 255}
	mountainPeakColor = color.RGBA{234, 244, 253, 255}
	riverColor        = color.RGBA{95, 150, 148, 255}
	mapRiverColor     = color.RGBA{84, 148, 210, 255} // a bluer river than the replay's, to show against tile outlines and any ground
	railroadColor     = color.RGBA{76, 51, 0, 255}
	roadColor         = color.RGBA{51, 51, 51, 255}
	unknownRouteColor = color.RGBA{0, 0, 0, 255}
)

// Route casings: the lighter outline under a route, warm for railroads and grey for roads.
var (
	railroadCasingColor = color.RGBA{190, 155, 100, 255}
	roadCasingColor     = color.RGBA{150, 150, 150, 255}
)

// tileOutlineDarken is how far a tile's outline is blended toward black from its fill.
const tileOutlineDarken = 0.12

// tileOutlineColor returns the outline color for a tile filled with fill.
func tileOutlineColor(fill color.RGBA) color.RGBA {
	return blendColor(fill, color.RGBA{0, 0, 0, 255}, tileOutlineDarken)
}

// MapRenderer renders Civ5 maps onto a Canvas.
type MapRenderer struct {
	config *DrawingConfig
	grid   *tileGrid // built on first use by tileGridFor
}

// NewMapRenderer returns a renderer using config.
func NewMapRenderer(config *DrawingConfig) *MapRenderer {
	return &MapRenderer{
		config: config,
	}
}

// DrawTerrainTiles draws every terrain tile.
func (mr *MapRenderer) DrawTerrainTiles(canvas Painter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	l := layoutForMap(mapSize, mr.config.Radius)
	mr.drawHexes(canvas, mapSize, func(pos fileio.TilePos) HexTile { return PhysicalHexTile(mapData, pos, l) }, mapData)
}

// ownedHexEdges are the SW, SE and E edges, so each shared edge is drawn once.
var ownedHexEdges = [...]int{3, 4, 5}

// drawHexes draws all fills, then all outlines; mountains and cities come later, so only routes cover a mountain and nothing covers a city.
func (mr *MapRenderer) drawHexes(canvas Painter, mapSize fileio.MapSize, hexOf func(fileio.TilePos) HexTile, mapData *fileio.Civ5MapData) {
	type tile struct {
		pos fileio.TilePos
		hex HexTile
	}
	tiles := make([]tile, 0, mapSize.Height*mapSize.Width)
	for _, pos := range allTiles(mapSize) {
		hex := hexOf(pos)
		if fileio.TileHasMountain(mapData, pos) {
			tint := mountainTileColor(color.RGBA{hex.R, hex.G, hex.B, 255})
			hex.R, hex.G, hex.B = tint.R, tint.G, tint.B
		}
		tiles = append(tiles, tile{pos, hex})
	}

	for _, t := range tiles {
		fillHex(canvas, t.hex, mr.config.Radius)
	}
	for _, t := range tiles {
		outline := tileOutlineColor(color.RGBA{t.hex.R, t.hex.G, t.hex.B, 255})
		canvas.SetColor(outline.R, outline.G, outline.B)
		canvas.SetLineWidth(1)
		for _, edge := range ownedHexEdges {
			line := getHexEdge(edge, t.hex.X, t.hex.Y, mr.config.Radius)
			canvas.DrawLine(line.X1, line.Y1, line.X2, line.Y2)
		}
		canvas.Stroke()
	}
}

// DrawMountains draws peaks after borders and rivers so neither covers one; routes cross them.
func (mr *MapRenderer) DrawMountains(canvas Painter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	for _, p := range rangePeaks(mapData, mapSize, layoutForMap(mapSize, mr.config.Radius)) {
		drawPeak(canvas, p)
	}
}

// DrawTerritoryTiles draws every territory tile.
func (mr *MapRenderer) DrawTerritoryTiles(canvas Painter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	l := layoutForMap(mapSize, mr.config.Radius)
	mr.drawHexes(canvas, mapSize, func(pos fileio.TilePos) HexTile {
		hex, _ := PoliticalHexTile(mapData, pos, l)
		return hex
	}, mapData)
}

// cityMarkerRadius is the radius of a city's marker, as a fraction of a tile's radius.
const cityMarkerRadius = 0.3

// DrawCityMarkers draws an outlined circle on each city, after the routes so none covers a marker.
func (mr *MapRenderer) DrawCityMarkers(canvas Painter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, cityColorOf func(fileio.TilePos) color.RGBA) {
	l := layoutForMap(mapSize, mr.config.Radius)
	radius := mr.config.Radius * cityMarkerRadius
	for _, pos := range allTiles(mapSize) {
		if !fileio.TileHasCity(mapData, pos) {
			continue
		}
		x, y := l.center(pos)
		fill := markerColor(cityColorOf(pos))
		drawDisc(canvas, x, y, radius+1, labelHaloColor(fill))
		drawDisc(canvas, x, y, radius, fill)
	}
}

// mapRiverWidth is the width of a river on a map, one and a half times a tile outline's.
const mapRiverWidth = 1.5

// drawRiverTile draws a single tile's river edges, in the given color and width.
func (mr *MapRenderer) drawRiverTile(canvas Painter, l tileLayout, mapData *fileio.Civ5MapData, pos fileio.TilePos, fill color.RGBA, width float64) {
	x, y := l.center(pos)
	canvas.SetColor(fill.R, fill.G, fill.B)
	canvas.SetLineWidth(width)

	for _, edge := range RiverEdgesForTile(mapData.PhysicalTile(pos).RiverData, x, y, l) {
		canvas.DrawLine(edge.X1, edge.Y1, edge.X2, edge.Y2)
		canvas.Stroke()
	}
}

// DrawRivers draws rivers on the map
func (mr *MapRenderer) DrawRivers(canvas Painter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	l := layoutForMap(mapSize, mr.config.Radius)
	for i := 0; i < mapSize.Height; i++ {
		for j := 0; j < mapSize.Width; j++ {
			mr.drawRiverTile(canvas, l, mapData, fileio.TilePos{Row: i, Col: j}, mapRiverColor, mapRiverWidth)
		}
	}
}

// DrawRoads draws roads between tiles, each with a lighter casing under it so it shows on any ground.
func (mr *MapRenderer) DrawRoads(canvas Painter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	l := layoutForMap(mapSize, mr.config.Radius)
	var segments []ColoredLine
	for _, pos := range allTiles(mapSize) {
		segments = append(segments, RoadSegmentsForTile(mapData, mapSize, pos, l)...)
	}

	// Casings first so none covers another route's line; then lines, wider (railroads) last.
	for _, segment := range segments {
		strokeSegment(canvas, segment.Line, segment.CasingWidth, segment.CasingR, segment.CasingG, segment.CasingB)
	}
	slices.SortStableFunc(segments, func(a, b ColoredLine) int { return cmp.Compare(a.LineWidth, b.LineWidth) })
	for _, segment := range segments {
		strokeSegment(canvas, segment.Line, segment.LineWidth, segment.R, segment.G, segment.B)
	}
}

// DrawPhysicalMap renders the physical map onto canvas.
func (mr *MapRenderer) DrawPhysicalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	mapSize := mapData.Size()

	maxImageWidth, maxImageHeight := imageSize(mapSize, mr.config.Radius)

	// Resize canvas to fit the map
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapSize.Height, ", width: ", mapSize.Width)

	mr.DrawTerrainTiles(canvas, mapData, mapSize)
	mr.DrawRivers(canvas, mapData, mapSize)
	mr.DrawMountains(canvas, mapData, mapSize)
	mr.DrawRoads(canvas, mapData, mapSize)
	mr.DrawCityMarkers(canvas, mapData, mapSize, func(fileio.TilePos) color.RGBA { return color.RGBA{255, 255, 255, 255} })

	// Draw city names on top of hexes
	mr.DrawPhysicalCityNames(canvas, mapData, mapSize)

	return canvas.Image()
}

// DrawBorders draws a band of pixels along each territory boundary, in the owner's color.
func (mr *MapRenderer) DrawBorders(canvas PixelPainter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	mr.drawTileBorders(canvas, mapData, mr.tileGridFor(mapSize), allTiles(mapSize))
}

// DrawPhysicalCityNames draws white city names, each with a halo, moved apart where they would overlap.
func (mr *MapRenderer) DrawPhysicalCityNames(canvas TextPainter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	l := layoutForMap(mapSize, mr.config.Radius)
	for _, label := range placedCityLabels(mapData, mapSize, l, func(pos fileio.TilePos) ColoredText { return PhysicalCityNameLabel(mapData, pos, l) }) {
		drawHaloedLabel(canvas, label)
	}
}

var (
	labelHaloDark  = color.RGBA{18, 18, 18, 255}
	labelHaloLight = color.RGBA{244, 244, 244, 255}
)

// labelHaloColor returns the halo for text of this color: dark or light, whichever contrasts more with the text.
func labelHaloColor(text color.RGBA) color.RGBA {
	if contrastRatio(text, labelHaloDark) >= contrastRatio(text, labelHaloLight) {
		return labelHaloDark
	}
	return labelHaloLight
}

// haloOffsets are the four one-pixel steps a label is repeated at to form its halo.
var haloOffsets = [4][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

// drawHaloedLabel draws a city label in its color over a thin halo, so it reads on any ground.
func drawHaloedLabel(canvas TextPainter, label ColoredText) {
	halo := labelHaloColor(color.RGBA{label.R, label.G, label.B, 255})
	canvas.SetColor(halo.R, halo.G, halo.B)
	for _, offset := range haloOffsets {
		canvas.DrawString(label.Text, label.X+offset[0], label.Y+offset[1])
	}
	canvas.SetColor(label.R, label.G, label.B)
	canvas.DrawString(label.Text, label.X, label.Y)
}

// DrawPoliticalCityNames draws haloed city names in civ colors, moved apart where they overlap.
func (mr *MapRenderer) DrawPoliticalCityNames(canvas TextPainter, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, l tileLayout) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for _, label := range placedCityLabels(mapData, mapSize, l, func(pos fileio.TilePos) ColoredText { return PoliticalCityNameLabel(mapData, pos, l) }) {
		drawHaloedLabel(canvas, label)
	}
}

// DrawPoliticalMap renders the political map onto canvas.
func (mr *MapRenderer) DrawPoliticalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	mapSize := mapData.Size()

	maxImageWidth, maxImageHeight := imageSize(mapSize, mr.config.Radius)

	// Resize canvas to fit the map
	canvas.Resize(int(maxImageWidth), int(maxImageHeight))

	fmt.Println("Map height: ", mapSize.Height, ", width: ", mapSize.Width)

	mr.DrawTerritoryTiles(canvas, mapData, mapSize)
	mr.DrawBorders(canvas, mapData, mapSize)
	mr.DrawRivers(canvas, mapData, mapSize)
	mr.DrawMountains(canvas, mapData, mapSize)
	mr.DrawRoads(canvas, mapData, mapSize)
	l := layoutForMap(mapSize, mr.config.Radius)
	mr.DrawCityMarkers(canvas, mapData, mapSize, func(pos fileio.TilePos) color.RGBA {
		_, cityColor := PoliticalHexTile(mapData, pos, l)
		return cityColor
	})

	// Draw city names on top of hexes
	mr.DrawPoliticalCityNames(canvas, mapData, mapSize, l)

	return canvas.Image()
}

// SaveImage saves canvas to outputFilename.
func (mr *MapRenderer) SaveImage(canvas Surface, outputFilename string) error {
	return canvas.SavePNG(outputFilename)
}
