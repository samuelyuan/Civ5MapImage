package graphics

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// tileRadius is the radius in pixels of a hex tile.
const tileRadius = 16.0

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

// drawHexes draws all fills, then all outlines; mountains and cities come later, so only routes cover a mountain and nothing covers a city.
func drawHexes(canvas Canvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, hexOf func(fileio.TilePos) HexTile) {
	type tile struct {
		pos fileio.TilePos
		hex HexTile
	}
	tiles := make([]tile, 0, mapSize.Height*mapSize.Width)
	for _, pos := range allTiles(mapSize) {
		hex := hexOf(pos)
		if fileio.TileHasMountain(mapData, pos) {
			hex.Fill = mountainTileColor(hex.Fill)
		}
		tiles = append(tiles, tile{pos, hex})
	}

	for _, t := range tiles {
		fillHex(canvas, t.hex, tileRadius)
	}
	for _, t := range tiles {
		outline := tileOutlineColor(t.hex.Fill)
		canvas.SetColor(outline.R, outline.G, outline.B)
		canvas.SetLineWidth(1)
		for _, edge := range ownedHexEdges {
			line := getHexEdge(edge, t.hex.X, t.hex.Y, tileRadius)
			canvas.DrawLine(line.X1, line.Y1, line.X2, line.Y2)
		}
		canvas.Stroke()
	}
}

// drawMountains draws peaks after borders and rivers so neither covers one; routes cross them.
func drawMountains(canvas Canvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	for _, p := range rangePeaks(mapData, mapSize, layoutForMap(mapSize, tileRadius)) {
		drawPeak(canvas, p)
	}
}

// cityMarkerRadius is the radius of a city's marker, as a fraction of a tile's radius.
const cityMarkerRadius = 0.3

// cityMarkerHalo is the width in pixels of the contrasting ring around a city's marker.
const cityMarkerHalo = 1

// cityMarkerOuterRadius returns how far a city's marker, halo included, reaches from its center on tiles of this radius.
func cityMarkerOuterRadius(radius float64) float64 {
	return radius*cityMarkerRadius + cityMarkerHalo
}

// drawCityMarkers draws an outlined circle on each city, after the routes so none covers a marker.
func drawCityMarkers(canvas Canvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, cityColorOf func(fileio.TilePos) color.RGBA) {
	l := layoutForMap(mapSize, tileRadius)
	radius := tileRadius * cityMarkerRadius
	for _, pos := range allTiles(mapSize) {
		if !fileio.TileHasCity(mapData, pos) {
			continue
		}
		x, y := l.center(pos)
		fill := markerColor(cityColorOf(pos))
		drawDisc(canvas, x, y, radius+cityMarkerHalo, labelHaloColor(fill))
		drawDisc(canvas, x, y, radius, fill)
	}
}

// mapRiverWidth is the width of a river on a map, one and a half times a tile outline's.
const mapRiverWidth = 1.5

// drawRiverTile draws a single tile's river edges, in the given color and width.
func drawRiverTile(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, pos fileio.TilePos, fill color.RGBA, width float64) {
	x, y := l.center(pos)
	canvas.SetColor(fill.R, fill.G, fill.B)
	canvas.SetLineWidth(width)

	for _, edge := range RiverEdgesForTile(mapData.PhysicalTile(pos).RiverData, x, y, l) {
		canvas.DrawLine(edge.X1, edge.Y1, edge.X2, edge.Y2)
		canvas.Stroke()
	}
}

// drawRivers draws rivers on the map
func drawRivers(canvas Canvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	l := layoutForMap(mapSize, tileRadius)
	for i := 0; i < mapSize.Height; i++ {
		for j := 0; j < mapSize.Width; j++ {
			drawRiverTile(canvas, l, mapData, fileio.TilePos{Row: i, Col: j}, mapRiverColor, mapRiverWidth)
		}
	}
}

// drawRoads draws roads between tiles, each with a lighter casing under it so it shows on any ground.
func drawRoads(canvas Canvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	l := layoutForMap(mapSize, tileRadius)
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

// fitCanvas resizes canvas to fit a map of mapSize and logs the size.
func fitCanvas(canvas Canvas, mapSize fileio.MapSize) {
	width, height := imageSize(mapSize, tileRadius)
	canvas.Resize(int(width), int(height))
	fmt.Println("Map height: ", mapSize.Height, ", width: ", mapSize.Width)
}

// drawBorders draws a band of pixels along each territory boundary, in the owner's color.
func drawBorders(canvas Canvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize) {
	drawTileBorders(canvas, mapData, buildTileGrid(mapSize, tileRadius), allTiles(mapSize))
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

// haloOffsets are the four steps of labelHaloWidth a label is repeated at to form its halo.
var haloOffsets = [4][2]float64{{-labelHaloWidth, 0}, {labelHaloWidth, 0}, {0, -labelHaloWidth}, {0, labelHaloWidth}}

// drawHaloedLabel draws a city label in its color over a thin halo, so it reads on any ground.
func drawHaloedLabel(canvas Canvas, label ColoredText) {
	halo := labelHaloColor(color.RGBA{label.R, label.G, label.B, 255})
	canvas.SetColor(halo.R, halo.G, halo.B)
	for _, offset := range haloOffsets {
		canvas.DrawString(label.Text, label.X+offset[0], label.Y+offset[1])
	}
	canvas.SetColor(label.R, label.G, label.B)
	canvas.DrawString(label.Text, label.X, label.Y)
}

// mapStyle is what differs between the physical and the political map.
type mapStyle struct {
	tile    func(pos fileio.TilePos) (fill HexTile, cityColor color.RGBA) // a tile's fill, and the color of a city marker on it
	label   func(pos fileio.TilePos) ColoredText
	borders bool // whether territory borders are drawn
}

func physicalStyle(mapData *fileio.Civ5MapData, l tileLayout) mapStyle {
	white := color.RGBA{255, 255, 255, 255}
	return mapStyle{
		tile:  func(pos fileio.TilePos) (HexTile, color.RGBA) { return PhysicalHexTile(mapData, pos, l), white },
		label: func(pos fileio.TilePos) ColoredText { return PhysicalCityNameLabel(mapData, pos, l) },
	}
}

func politicalStyle(mapData *fileio.Civ5MapData, l tileLayout) mapStyle {
	return mapStyle{
		tile:    func(pos fileio.TilePos) (HexTile, color.RGBA) { return PoliticalHexTile(mapData, pos, l) },
		label:   func(pos fileio.TilePos) ColoredText { return PoliticalCityNameLabel(mapData, pos, l) },
		borders: true,
	}
}

// DrawPhysicalMap renders the physical map onto canvas.
func DrawPhysicalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	return drawMap(canvas, mapData, physicalStyle)
}

// DrawPoliticalMap renders the political map onto canvas.
func DrawPoliticalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	return drawMap(canvas, mapData, politicalStyle)
}

// drawMap draws a map layer by layer, each over the last: tiles, borders, rivers, mountains, routes, city markers, then city names.
func drawMap(canvas Canvas, mapData *fileio.Civ5MapData, styleFor func(*fileio.Civ5MapData, tileLayout) mapStyle) image.Image {
	mapSize := mapData.Size()
	l := layoutForMap(mapSize, tileRadius)
	style := styleFor(mapData, l)

	fitCanvas(canvas, mapSize)
	drawHexes(canvas, mapData, mapSize, func(pos fileio.TilePos) HexTile {
		fill, _ := style.tile(pos)
		return fill
	})
	if style.borders {
		drawBorders(canvas, mapData, mapSize)
	}
	drawRivers(canvas, mapData, mapSize)
	drawMountains(canvas, mapData, mapSize)
	drawRoads(canvas, mapData, mapSize)
	drawCityMarkers(canvas, mapData, mapSize, func(pos fileio.TilePos) color.RGBA {
		_, cityColor := style.tile(pos)
		return cityColor
	})
	drawCityNames(canvas, mapData, mapSize, l, style.label)
	return canvas.Image()
}

// drawCityNames draws each city's name over a halo, moved apart where labels would overlap.
func drawCityNames(canvas Canvas, mapData *fileio.Civ5MapData, mapSize fileio.MapSize, l tileLayout, labelOf func(fileio.TilePos) ColoredText) {
	if len(mapData.MapTileImprovements) == 0 {
		return
	}
	for _, label := range placedCityLabels(mapData, mapSize, l, labelOf) {
		drawHaloedLabel(canvas, label)
	}
}
