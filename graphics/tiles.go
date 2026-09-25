package graphics

import (
	"image/color"
	"math"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// HexTile is a hex tile's screen position and fill color.
type HexTile struct {
	X, Y float64
	Fill color.RGBA
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
	return HexTile{X: x, Y: y, Fill: GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, pos))}
}

// PoliticalHexTile returns the tile's political position and fill, plus its city icon color (white if unowned).
func PoliticalHexTile(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout) (HexTile, color.RGBA) {
	cityColor := color.RGBA{255, 255, 255, 255}

	if fileio.IsWaterTile(mapData, pos) {
		return PhysicalHexTile(mapData, pos, l), cityColor
	}

	colorKey := fileio.GetPoliticalMapTileColor(mapData, pos)
	civColor, known := civColorMap[colorKey]
	switch {
	case known:
		x, y := l.center(pos)
		shades := shadesFor(civColor, tileIsMinor(mapData, pos))
		return HexTile{X: x, Y: y, Fill: shades.fill}, shades.border
	case colorKey != "":
		// No color, but tile is owned by a civ or city state.
		x, y := l.center(pos)
		return HexTile{X: x, Y: y, Fill: color.RGBA{0, 0, 0, 255}}, cityColor
	default:
		// Territory not owned by anyone.
		return PhysicalHexTile(mapData, pos, l), cityColor
	}
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

// Indexes of the hex edges a tile draws for itself; the other three are its neighbors' edges. See hexVertex for the numbering.
const (
	hexEdgeSW = 3
	hexEdgeSE = 4
	hexEdgeE  = 5
)

// ownedHexEdges are the SW, SE and E edges, so each shared edge is drawn once.
var ownedHexEdges = [...]int{hexEdgeSW, hexEdgeSE, hexEdgeE}

// RiverData bits for the edges a tile owns.
const (
	riverBitSW = 1 << 2
	riverBitSE = 1 << 1
	riverBitE  = 1 << 0
)

// RiverEdgesForTile decodes a RiverData bitmask into the SW, SE and E edges; the others belong to neighbors.
func RiverEdgesForTile(riverData int, centerX, centerY float64, l tileLayout) []Line {
	var edges []Line
	for _, river := range [...]struct{ bit, edge int }{{riverBitSW, hexEdgeSW}, {riverBitSE, hexEdgeSE}, {riverBitE, hexEdgeE}} {
		if riverData&river.bit != 0 {
			edges = append(edges, getHexEdge(river.edge, centerX, centerY, l.radius))
		}
	}
	return edges
}

// routeStyle returns the line width, line color and casing color a route of this type is drawn in.
func routeStyle(routeType int) (width float64, line, casing color.RGBA) {
	switch routeType {
	case fileio.RouteRailroad:
		return 2.0, railroadColor, railroadCasingColor
	case fileio.RouteRoad:
		return 1.0, roadColor, roadCasingColor
	default:
		return 1.0, unknownRouteColor, roadCasingColor
	}
}

// RoadSegmentsForTile returns lines from the tile to each neighbor with a route or city, nil if it has no route (255).
func RoadSegmentsForTile(mapData *fileio.Civ5MapData, mapSize fileio.MapSize, pos fileio.TilePos, l tileLayout) []ColoredLine {
	routeType := mapData.TileImprovement(pos).RouteType
	if routeType == fileio.RouteNone {
		return nil
	}
	lineWidth, lineColor, casingColor := routeStyle(routeType)

	x1, y1 := l.center(pos)

	var segments []ColoredLine
	for _, neighbor := range fileio.GetNeighbors(pos) {
		if !neighbor.InMap(mapSize) {
			continue
		}

		neighborTile := mapData.TileImprovement(neighbor)
		if neighborTile.RouteType == fileio.RouteNone && neighborTile.CityName == "" {
			continue
		}

		x2, y2 := l.center(neighbor)

		// Draw only up to the midpoint, which is the shared tile border.
		borderX := (x1 + x2) / 2.0
		borderY := (y1 + y2) / 2.0

		segments = append(segments, ColoredLine{
			Line:        Line{X1: x1, Y1: y1, X2: borderX, Y2: borderY},
			LineWidth:   lineWidth,
			R:           lineColor.R,
			G:           lineColor.G,
			B:           lineColor.B,
			CasingWidth: lineWidth + casingExtra,
			CasingR:     casingColor.R,
			CasingG:     casingColor.G,
			CasingB:     casingColor.B,
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

// tileLayout places the tiles of a map rows rows tall in y-down pixels; row 0 is the bottom row, so it is drawn lowest.
type tileLayout struct {
	radius float64
	rows   int
}

func newTileLayout(radius float64, rows int) tileLayout {
	return tileLayout{radius: radius, rows: rows}
}

// layoutForMap is the layout of a map of the given size.
func layoutForMap(mapSize fileio.MapSize, radius float64) tileLayout {
	return newTileLayout(radius, mapSize.Height)
}

func (l tileLayout) halfWidth() float64 { return l.radius * math.Cos(math.Pi/6) }       // half a pointy-top hex's width
func (l tileLayout) rowHeight() float64 { return l.radius * (1 + math.Sin(math.Pi/6)) } // vertical distance between rows, whose hexes interlock

// center returns the center of the tile at pos.
func (l tileLayout) center(pos fileio.TilePos) (x, y float64) {
	x = l.radius*1.5 + float64(pos.Col)*2*l.halfWidth()
	if pos.Row%2 == 1 {
		x += l.halfWidth() // odd rows are shifted right by half a tile
	}
	return x, l.rowHeight() * float64(l.rows-pos.Row)
}

// imageSize returns the image size a map needs: wide enough for the tile one past the far corner, tall enough for the bottom row's lowest vertex.
func imageSize(mapSize fileio.MapSize, radius float64) (width, height float64) {
	l := layoutForMap(mapSize, radius)
	width, _ = l.center(fileio.TilePos{Row: mapSize.Height, Col: mapSize.Width})
	return width, l.radius + float64(mapSize.Height)*l.rowHeight()
}
