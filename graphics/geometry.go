package graphics

import (
	"image/color"
	"math"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// Line represents a line with start and end points.
type Line struct {
	X1, Y1, X2, Y2 float64
}

// getHexEdge calculates the coordinates for a specific hexagon edge.
// edgeIndex: 0-5 representing the 6 edges of a hexagon
// centerX, centerY: center of the hexagon
// radius: radius of the hexagon (can be adjusted for inner/outer edges)
func getHexEdge(edgeIndex int, centerX, centerY, radius float64) Line {
	angle1 := (math.Pi / 6) + float64(edgeIndex)*(math.Pi/3)
	angle2 := (math.Pi / 6) + float64(edgeIndex+1)*(math.Pi/3)

	return Line{
		X1: centerX + radius*math.Cos(angle1),
		Y1: centerY + radius*math.Sin(angle1),
		X2: centerX + radius*math.Cos(angle2),
		Y2: centerY + radius*math.Sin(angle2),
	}
}

// InvertedRow mirrors a row against mapHeight, for city name labels only: they're drawn after a
// second InvertY() cancels the first back to identity (InvertY()'s transform mirrors text
// glyphs, not just repositions them), so label math must supply this inversion itself instead of
// relying on the canvas transform like every other draw call does.
func InvertedRow(mapHeight, row int) int {
	return mapHeight - row
}

// ColoredLine is a line segment plus the width and color to draw it with. Used for both road
// and border segments.
type ColoredLine struct {
	Line      Line
	LineWidth float64
	R, G, B   uint8
}

// HexTile is a hex tile's screen position and fill color.
type HexTile struct {
	X, Y    float64
	R, G, B uint8
}

// PhysicalHexTile returns tile (row, col)'s position and terrain fill color for the physical map.
func PhysicalHexTile(mapData *fileio.Civ5MapData, row, col int, radius float64) HexTile {
	x, y := fileio.GetImagePosition(row, col, radius)
	c := fileio.GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, row, col))
	return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}
}

// PoliticalHexTile returns tile (row, col)'s position and fill color for the political map, plus
// the color a city icon on this tile should use (white if the tile has no recognized owner).
func PoliticalHexTile(mapData *fileio.Civ5MapData, row, col int, radius float64) (HexTile, color.RGBA) {
	x, y := fileio.GetImagePosition(row, col, radius)
	cityColor := color.RGBA{255, 255, 255, 255}

	if fileio.IsWaterTile(mapData, row, col) {
		c := fileio.GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, row, col))
		return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}, cityColor
	}

	tileColor := fileio.GetPoliticalMapTileColor(mapData, row, col)
	renderColor, ok := civColorMap[tileColor]
	if !ok {
		if tileColor != "" {
			// No color, but tile is owned by a civ or city state.
			return HexTile{X: x, Y: y}, cityColor
		}
		// Territory not owned by anyone.
		c := fileio.GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, row, col))
		return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}, cityColor
	}

	white := color.RGBA{255, 255, 255, 255}
	var background color.RGBA
	if strings.Contains(fileio.GetTileCivName(mapData, row, col), "MINOR") {
		cityColor = renderColor.OuterColor
		background = blendColor(renderColor.InnerColor, white, 0.1)
	} else {
		cityColor = renderColor.InnerColor
		background = blendColor(renderColor.OuterColor, white, 0.2)
	}
	return HexTile{X: x, Y: y, R: background.R, G: background.G, B: background.B}, cityColor
}

// EntityType identifies what a map marker represents. The shape used to draw each type (e.g. a
// triangle for a mountain, a square for a city) is a drawing-layer decision, not encoded here.
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

// TileEntities returns the mountain/city markers for tile (row, col), or nil if it has neither.
// cityColor is used for a city marker, if present.
func TileEntities(mapData *fileio.Civ5MapData, row, col int, radius float64, cityColor color.RGBA) []Entity {
	x, y := fileio.GetImagePosition(row, col, radius)

	var entities []Entity
	if fileio.TileHasMountain(mapData, row, col) {
		entities = append(entities, Entity{Type: EntityMountain, X: x, Y: y})
	}
	if fileio.TileHasCity(mapData, row, col) {
		entities = append(entities, Entity{Type: EntityCity, X: x, Y: y, R: cityColor.R, G: cityColor.G, B: cityColor.B})
	}
	return entities
}

// RiverEdgesForTile decodes a RiverData bitmask into hex edge lines. Only southwest/southeast/
// east are produced; the other three edges belong to the neighboring tile's own RiverData.
func RiverEdgesForTile(riverData int, centerX, centerY, radius float64) []Line {
	var edges []Line
	if (riverData>>2)&1 != 0 { // Southwest (edge 3)
		edges = append(edges, getHexEdge(3, centerX, centerY, radius))
	}
	if (riverData>>1)&1 != 0 { // Southeast (edge 4)
		edges = append(edges, getHexEdge(4, centerX, centerY, radius))
	}
	if riverData&1 != 0 { // East (edge 5)
		edges = append(edges, getHexEdge(5, centerX, centerY, radius))
	}
	return edges
}

// RoadSegmentsForTile returns lines from tile (row, col) to each connected neighbor, or nil if
// the tile has no route (RouteType 255). A neighbor is connected if it has a route too, or a
// city (roads visibly terminate at cities even without a route type set).
func RoadSegmentsForTile(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) []ColoredLine {
	routeType := mapData.MapTileImprovements[row][col].RouteType
	if routeType == 255 {
		return nil
	}

	x1, y1 := fileio.GetImagePosition(row, col, radius)

	var segments []ColoredLine
	neighbors := fileio.GetNeighbors(col, row)
	for n := 0; n < len(neighbors); n++ {
		newX := neighbors[n][0]
		newY := neighbors[n][1]
		if newX < 0 || newY < 0 || newX >= mapWidth || newY >= mapHeight {
			continue
		}

		neighborTile := mapData.MapTileImprovements[newY][newX]
		if neighborTile.RouteType == 255 && neighborTile.CityName == "" {
			continue
		}

		x2, y2 := fileio.GetImagePosition(newY, newX, radius)

		var lineWidth float64
		var r, g, b uint8
		switch routeType {
		case 1: // Railroad
			lineWidth, r, g, b = 2.0, 76, 51, 0
		case 0: // Road
			lineWidth, r, g, b = 1.0, 51, 51, 51
		default: // Unknown
			lineWidth, r, g, b = 1.0, 0, 0, 0
		}

		// Draw only up to the midpoint, which is the shared tile border.
		borderX := (x1 + x2) / 2.0
		borderY := (y1 + y2) / 2.0

		segments = append(segments, ColoredLine{
			Line:      Line{X1: x1, Y1: y1, X2: borderX, Y2: borderY},
			LineWidth: lineWidth,
			R:         r,
			G:         g,
			B:         b,
		})
	}
	return segments
}

// BorderLineWidth is the width every territory border segment draws with.
const BorderLineWidth = 1.5

// BorderSegmentsForTile returns border lines around tile (row, col) against neighbors with a
// different owner, colored with the owning civ's border color (white if unrecognized). Returns
// nil if the tile has no valid owner.
func BorderSegmentsForTile(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) []ColoredLine {
	currentTileOwner := mapData.MapTileImprovements[row][col].Owner
	if fileio.IsInvalidTileOwner(currentTileOwner) {
		return nil
	}

	x1, y1 := fileio.GetImagePosition(row, col, radius)

	tileColor := fileio.GetPoliticalMapTileColor(mapData, row, col)
	renderColor, ok := civColorMap[tileColor]
	borderColor := color.RGBA{255, 255, 255, 255}
	if ok {
		if strings.Contains(fileio.GetTileCivName(mapData, row, col), "MINOR") {
			// Invert city state colors, matching DrawTerritoryTiles' convention.
			borderColor = renderColor.OuterColor
		} else {
			borderColor = renderColor.InnerColor
		}
	}

	var segments []ColoredLine
	neighbors := fileio.GetNeighbors(col, row)
	for n := 0; n < len(neighbors); n++ {
		newX := neighbors[n][0]
		newY := neighbors[n][1]
		if newX < 0 || newY < 0 || newX >= mapWidth || newY >= mapHeight {
			continue
		}

		otherTileOwner := mapData.MapTileImprovements[newY][newX].Owner
		if currentTileOwner == otherTileOwner {
			continue
		}

		segments = append(segments, ColoredLine{
			Line:      getHexEdge(n, x1, y1, radius-1),
			LineWidth: BorderLineWidth,
			R:         borderColor.R,
			G:         borderColor.G,
			B:         borderColor.B,
		})
	}
	return segments
}

// ColoredText is a text label plus the position and color to draw it at.
type ColoredText struct {
	Text    string
	X, Y    float64
	R, G, B uint8
}

// cityNameText returns tile (row, col)'s display city name, trimmed at the first null byte.
func cityNameText(mapData *fileio.Civ5MapData, row, col int) string {
	return strings.Split(mapData.MapTileImprovements[row][col].CityName, "\x00")[0]
}

// cityLabelPosition returns the anchor point for a tile's city name label: the tile's screen
// position (row-inverted, per InvertedRow), horizontally centered and floated above the tile.
func cityLabelPosition(mapHeight, row, col int, radius float64, cityName string) (float64, float64) {
	x, y := fileio.GetImagePosition(InvertedRow(mapHeight, row), col, radius)
	return x - (6.0 * float64(len(cityName)) / 2.0), y - radius*1.5
}

// blendColor linearly interpolates between two colors by t (0 = c1, 1 = c2).
func blendColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		uint8(float64(c1.R) + (float64(c2.R)-float64(c1.R))*t),
		uint8(float64(c1.G) + (float64(c2.G)-float64(c1.G))*t),
		uint8(float64(c1.B) + (float64(c2.B)-float64(c1.B))*t),
		255,
	}
}

// CityNameLabel returns tile (row, col)'s city name label in white -- used for the physical map,
// where labels aren't colored by ownership.
func CityNameLabel(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) ColoredText {
	cityName := cityNameText(mapData, row, col)
	x, y := cityLabelPosition(mapHeight, row, col, radius, cityName)
	return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
}

// PoliticalCityNameLabel returns tile (row, col)'s city name label colored by its owning civ
// (white if unrecognized) -- used for the political map.
func PoliticalCityNameLabel(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) ColoredText {
	cityName := cityNameText(mapData, row, col)
	x, y := cityLabelPosition(mapHeight, row, col, radius, cityName)

	tileColor := fileio.GetPoliticalMapTileColor(mapData, row, col)
	renderColor, ok := civColorMap[tileColor]
	if !ok {
		return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
	}

	var cityColor color.RGBA
	if strings.Contains(fileio.GetTileCivName(mapData, row, col), "MINOR") {
		cityColor = renderColor.OuterColor
	} else {
		cityColor = renderColor.InnerColor
	}
	textColor := blendColor(cityColor, color.RGBA{255, 255, 255, 255}, 0.2)
	return ColoredText{Text: cityName, X: x, Y: y, R: textColor.R, G: textColor.G, B: textColor.B}
}
