package graphics

import (
	"image/color"
	"math"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// HexTile is a hex tile's screen position and fill color.
type HexTile struct {
	X, Y    float64
	R, G, B uint8
}

// civShades are the colors a civ's tiles are drawn with; city states swap the pair.
type civShades struct {
	fill   color.RGBA // territory fill: the pair's background color washed toward white
	border color.RGBA // borders, and the base of city markers and names
}

func shadesFor(c CivColor, minor bool) civShades {
	white := color.RGBA{255, 255, 255, 255}
	if minor {
		return civShades{fill: blendColor(c.InnerColor, white, 0.1), border: c.OuterColor}
	}
	return civShades{fill: blendColor(c.OuterColor, white, 0.2), border: c.InnerColor}
}

// markerColor is the color a city icon or name is drawn in over its civ's mark, lightened for visibility.
func markerColor(mark color.RGBA) color.RGBA {
	return blendColor(mark, color.RGBA{255, 255, 255, 255}, 0.2)
}

// tileIsMinor reports whether the tile at pos belongs to a city state.
func tileIsMinor(mapData *fileio.Civ5MapData, pos fileio.TilePos) bool {
	return strings.Contains(fileio.GetTileCivName(mapData, pos), "MINOR")
}

// PhysicalHexTile returns the position and terrain fill color of the tile at pos, for the physical map.
func PhysicalHexTile(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout) HexTile {
	x, y := l.center(pos)
	c := GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, pos))
	return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}
}

// PoliticalHexTile returns the tile's political position and fill, plus its city icon color (white if unowned).
func PoliticalHexTile(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout) (HexTile, color.RGBA) {
	x, y := l.center(pos)
	cityColor := color.RGBA{255, 255, 255, 255}

	if fileio.IsWaterTile(mapData, pos) {
		c := GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, pos))
		return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}, cityColor
	}

	tileColor := fileio.GetPoliticalMapTileColor(mapData, pos)
	renderColor, ok := civColorMap[tileColor]
	if !ok {
		if tileColor != "" {
			// No color, but tile is owned by a civ or city state.
			return HexTile{X: x, Y: y}, cityColor
		}
		// Territory not owned by anyone.
		c := GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, pos))
		return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}, cityColor
	}

	shades := shadesFor(renderColor, tileIsMinor(mapData, pos))
	return HexTile{X: x, Y: y, R: shades.fill.R, G: shades.fill.G, B: shades.fill.B}, shades.border
}

// tileBorderColor returns the civ border color of the tile at pos, white if unrecognized.
func tileBorderColor(mapData *fileio.Civ5MapData, pos fileio.TilePos) color.RGBA {
	renderColor, ok := civColorMap[fileio.GetPoliticalMapTileColor(mapData, pos)]
	if !ok {
		return color.RGBA{255, 255, 255, 255}
	}
	return shadesFor(renderColor, tileIsMinor(mapData, pos)).border
}

// ColoredLine is a line segment with its width and color, and the wider, lighter casing drawn under it on maps.
type ColoredLine struct {
	Line                      Line
	LineWidth                 float64
	R, G, B                   uint8
	CasingWidth               float64
	CasingR, CasingG, CasingB uint8
}

// casingExtra is how much wider than its line a route's casing is: one pixel of casing on each side.
const casingExtra = 2

// RiverEdgesForTile decodes a RiverData bitmask into the SW, SE and E edges; the others belong to neighbors.
func RiverEdgesForTile(riverData int, centerX, centerY float64, l tileLayout) []Line {
	var edges []Line
	if (riverData>>2)&1 != 0 { // Southwest (edge 3)
		edges = append(edges, getHexEdge(3, centerX, centerY, l.radius))
	}
	if (riverData>>1)&1 != 0 { // Southeast (edge 4)
		edges = append(edges, getHexEdge(4, centerX, centerY, l.radius))
	}
	if riverData&1 != 0 { // East (edge 5)
		edges = append(edges, getHexEdge(5, centerX, centerY, l.radius))
	}
	return edges
}

// RoadSegmentsForTile returns lines from the tile to each neighbor with a route or city, nil if it has no route (255).
func RoadSegmentsForTile(mapData *fileio.Civ5MapData, mapSize fileio.MapSize, pos fileio.TilePos, l tileLayout) []ColoredLine {
	routeType := mapData.TileImprovement(pos).RouteType
	if routeType == 255 {
		return nil
	}

	x1, y1 := l.center(pos)

	var segments []ColoredLine
	for _, neighbor := range fileio.GetNeighbors(pos) {
		if !neighbor.InMap(mapSize) {
			continue
		}

		neighborTile := mapData.TileImprovement(neighbor)
		if neighborTile.RouteType == 255 && neighborTile.CityName == "" {
			continue
		}

		x2, y2 := l.center(neighbor)

		var lineWidth float64
		var line, casing color.RGBA
		switch routeType {
		case 1: // Railroad
			lineWidth, line, casing = 2.0, railroadColor, railroadCasingColor
		case 0: // Road
			lineWidth, line, casing = 1.0, roadColor, roadCasingColor
		default: // Unknown
			lineWidth, line, casing = 1.0, unknownRouteColor, roadCasingColor
		}

		// Draw only up to the midpoint, which is the shared tile border.
		borderX := (x1 + x2) / 2.0
		borderY := (y1 + y2) / 2.0

		segments = append(segments, ColoredLine{
			Line:        Line{X1: x1, Y1: y1, X2: borderX, Y2: borderY},
			LineWidth:   lineWidth,
			R:           line.R,
			G:           line.G,
			B:           line.B,
			CasingWidth: lineWidth + casingExtra,
			CasingR:     casing.R,
			CasingG:     casing.G,
			CasingB:     casing.B,
		})
	}
	return segments
}

// Line represents a line with start and end points.
type Line struct {
	X1, Y1, X2, Y2 float64
}

// hexVertex returns vertex i (0-5) of the pointy-top hex in y-down pixels; vertex 0 is 30 degrees above the x axis, each next 60 counterclockwise.
func hexVertex(i int, centerX, centerY, radius float64) (x, y float64) {
	angle := (math.Pi / 6) + float64(i)*(math.Pi/3)
	return centerX + radius*math.Cos(angle), centerY - radius*math.Sin(angle)
}

// getHexEdge returns the edge between vertices edgeIndex and edgeIndex+1.
func getHexEdge(edgeIndex int, centerX, centerY, radius float64) Line {
	x1, y1 := hexVertex(edgeIndex, centerX, centerY, radius)
	x2, y2 := hexVertex(edgeIndex+1, centerX, centerY, radius)
	return Line{X1: x1, Y1: y1, X2: x2, Y2: y2}
}

// tileLayout places tiles in y-down pixels; center flips GetImagePosition's y-up (rows count from the bottom) against canvasHeight.
type tileLayout struct {
	radius       float64
	canvasHeight int
}

// newTileLayout is the layout of a canvas canvasHeight pixels tall.
func newTileLayout(radius float64, canvasHeight int) tileLayout {
	return tileLayout{radius: radius, canvasHeight: canvasHeight}
}

// layoutForMap is the layout of the canvas a map of the given size is drawn on.
func layoutForMap(mapSize fileio.MapSize, radius float64) tileLayout {
	_, height := imageSize(mapSize, radius)
	return newTileLayout(radius, int(height))
}

// center returns the center of the tile at pos.
func (l tileLayout) center(pos fileio.TilePos) (x, y float64) {
	x, y = GetImagePosition(pos, l.radius)
	return x, float64(l.canvasHeight) - y
}

// GetImagePosition returns the center of the tile at pos, in the map's y-up layout.
func GetImagePosition(pos fileio.TilePos, radius float64) (float64, float64) {
	angle := math.Pi / 6

	x := (radius * 1.5) + float64(pos.Col)*(2*radius*math.Cos(angle))
	y := radius + float64(pos.Row)*radius*(1+math.Sin(angle))
	if pos.Row%2 == 1 {
		x += radius * math.Cos(angle)
	}
	return x, y
}

// imageSize returns the image size a map needs: the position of the tile just past its far corner.
func imageSize(mapSize fileio.MapSize, radius float64) (width, height float64) {
	return GetImagePosition(fileio.TilePos{Row: mapSize.Height, Col: mapSize.Width}, radius)
}
