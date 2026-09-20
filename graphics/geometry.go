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

// hexVertex returns vertex i (0-5) of the pointy-top hex at (centerX, centerY), starting 30 degrees from the x axis.
func hexVertex(i int, centerX, centerY, radius float64) (x, y float64) {
	angle := (math.Pi / 6) + float64(i)*(math.Pi/3)
	return centerX + radius*math.Cos(angle), centerY + radius*math.Sin(angle)
}

// getHexEdge returns edge edgeIndex (0-5) of the hexagon at (centerX, centerY).
func getHexEdge(edgeIndex int, centerX, centerY, radius float64) Line {
	x1, y1 := hexVertex(edgeIndex, centerX, centerY, radius)
	x2, y2 := hexVertex(edgeIndex+1, centerX, centerY, radius)
	return Line{X1: x1, Y1: y1, X2: x2, Y2: y2}
}

// InvertedRow mirrors row against mapHeight; city labels need it because they're drawn after the canvas flip is undone.
func InvertedRow(mapHeight, row int) int {
	return mapHeight - row
}

// tileLayout places tiles and marks on a canvas: mapLayout keeps the map file's y-up space (the canvas flips it), pixelLayout flips once into y-down pixels.
type tileLayout struct {
	radius       float64
	canvasHeight int // pixel layouts only
	pixels       bool
}

// mapLayout is the y-up layout the physical and political maps draw in.
func mapLayout(radius float64) tileLayout { return tileLayout{radius: radius} }

// pixelLayout is the y-down layout of a canvas canvasHeight pixels tall, flipped against that integer height.
func pixelLayout(radius float64, canvasHeight int) tileLayout {
	return tileLayout{radius: radius, canvasHeight: canvasHeight, pixels: true}
}

// center returns the center of tile (row, col).
func (l tileLayout) center(row, col int) (x, y float64) {
	x, y = GetImagePosition(row, col, l.radius)
	if l.pixels {
		y = float64(l.canvasHeight) - y
	}
	return x, y
}

// up returns dy as an offset toward the top of the picture.
func (l tileLayout) up(dy float64) float64 {
	if l.pixels {
		return -dy
	}
	return dy
}

// apexUpRotation is the DrawRegularPolygon rotation that points a triangle's apex toward the top of the picture.
func (l tileLayout) apexUpRotation() float64 {
	if l.pixels {
		return 0
	}
	return math.Pi
}

// hexEdge is getHexEdge on the tile edge that looks the same in either layout: the pixel layout mirrors it vertically.
func (l tileLayout) hexEdge(edgeIndex int, centerX, centerY float64) Line {
	edge := getHexEdge(edgeIndex, centerX, centerY, l.radius)
	if l.pixels {
		edge.Y1, edge.Y2 = 2*centerY-edge.Y1, 2*centerY-edge.Y2
	}
	return edge
}

// ColoredLine is a line segment with its width and color.
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

// civShades are the colors a civ's tiles are drawn with; city states swap the pair.
type civShades struct {
	fill   color.RGBA // territory fill: the pair's background color washed toward white
	accent color.RGBA // borders, and the base of city markers and names
}

func shadesFor(c CivColor, minor bool) civShades {
	white := color.RGBA{255, 255, 255, 255}
	if minor {
		return civShades{fill: blendColor(c.InnerColor, white, 0.1), accent: c.OuterColor}
	}
	return civShades{fill: blendColor(c.OuterColor, white, 0.2), accent: c.InnerColor}
}

// markerColor is the color a city icon or name is drawn in over its civ's accent, lightened for visibility.
func markerColor(accent color.RGBA) color.RGBA {
	return blendColor(accent, color.RGBA{255, 255, 255, 255}, 0.2)
}

// tileIsMinor reports whether tile (row, col) belongs to a city state.
func tileIsMinor(mapData *fileio.Civ5MapData, row, col int) bool {
	return strings.Contains(fileio.GetTileCivName(mapData, row, col), "MINOR")
}

// PhysicalHexTile returns tile (row, col)'s position and terrain fill color for the physical map.
func PhysicalHexTile(mapData *fileio.Civ5MapData, row, col int, radius float64) HexTile {
	x, y := GetImagePosition(row, col, radius)
	c := GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, row, col))
	return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}
}

// PoliticalHexTile returns tile (row, col)'s political position and fill, plus its city icon color (white if unowned).
func PoliticalHexTile(mapData *fileio.Civ5MapData, row, col int, l tileLayout) (HexTile, color.RGBA) {
	x, y := l.center(row, col)
	cityColor := color.RGBA{255, 255, 255, 255}

	if fileio.IsWaterTile(mapData, row, col) {
		c := GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, row, col))
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
		c := GetPhysicalMapTileColor(fileio.GetTerrainString(mapData, row, col))
		return HexTile{X: x, Y: y, R: c.R, G: c.G, B: c.B}, cityColor
	}

	shades := shadesFor(renderColor, tileIsMinor(mapData, row, col))
	return HexTile{X: x, Y: y, R: shades.fill.R, G: shades.fill.G, B: shades.fill.B}, shades.accent
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

// TileEntities returns tile (row, col)'s mountain/city markers (cityColor for a city), or nil.
func TileEntities(mapData *fileio.Civ5MapData, row, col int, l tileLayout, cityColor color.RGBA) []Entity {
	x, y := l.center(row, col)

	var entities []Entity
	if fileio.TileHasMountain(mapData, row, col) {
		entities = append(entities, Entity{Type: EntityMountain, X: x, Y: y})
	}
	if fileio.TileHasCity(mapData, row, col) {
		entities = append(entities, Entity{Type: EntityCity, X: x, Y: y, R: cityColor.R, G: cityColor.G, B: cityColor.B})
	}
	return entities
}

// RiverEdgesForTile decodes a RiverData bitmask into hex edge lines (southwest, southeast and east only; the rest belong to neighbors).
func RiverEdgesForTile(riverData int, centerX, centerY float64, l tileLayout) []Line {
	var edges []Line
	if (riverData>>2)&1 != 0 { // Southwest (edge 3)
		edges = append(edges, l.hexEdge(3, centerX, centerY))
	}
	if (riverData>>1)&1 != 0 { // Southeast (edge 4)
		edges = append(edges, l.hexEdge(4, centerX, centerY))
	}
	if riverData&1 != 0 { // East (edge 5)
		edges = append(edges, l.hexEdge(5, centerX, centerY))
	}
	return edges
}

// RoadSegmentsForTile returns lines from tile (row, col) to each neighbor with a route or city, or nil if it has no route (255).
func RoadSegmentsForTile(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, l tileLayout) []ColoredLine {
	routeType := mapData.MapTileImprovements[row][col].RouteType
	if routeType == 255 {
		return nil
	}

	x1, y1 := l.center(row, col)

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

		x2, y2 := l.center(newY, newX)

		var lineWidth float64
		var r, g, b uint8
		switch routeType {
		case 1: // Railroad
			lineWidth, r, g, b = 2.0, railroadColor.R, railroadColor.G, railroadColor.B
		case 0: // Road
			lineWidth, r, g, b = 1.0, roadColor.R, roadColor.G, roadColor.B
		default: // Unknown
			lineWidth, r, g, b = 1.0, unknownRouteColor.R, unknownRouteColor.G, unknownRouteColor.B
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

// tileBorderColor returns tile (row, col)'s civ border color, white if unrecognized.
func tileBorderColor(mapData *fileio.Civ5MapData, row, col int) color.RGBA {
	renderColor, ok := civColorMap[fileio.GetPoliticalMapTileColor(mapData, row, col)]
	if !ok {
		return color.RGBA{255, 255, 255, 255}
	}
	return shadesFor(renderColor, tileIsMinor(mapData, row, col)).accent
}

// BorderSegmentsForTile returns border lines against neighbors with a different owner, or nil if the tile has no valid owner.
func BorderSegmentsForTile(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) []ColoredLine {
	currentTileOwner := mapData.MapTileImprovements[row][col].Owner
	if fileio.IsInvalidTileOwner(currentTileOwner) {
		return nil
	}

	x1, y1 := GetImagePosition(row, col, radius)

	borderColor := tileBorderColor(mapData, row, col)

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
	name := mapData.MapTileImprovements[row][col].CityName
	if i := strings.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	return name
}

// cityLabelPosition returns the anchor for a tile's city label, centered above the tile (via InvertedRow in the y-up layout).
func cityLabelPosition(l tileLayout, mapHeight, row, col int, cityName string) (float64, float64) {
	halfWidth := 6.0 * float64(len(cityName)) / 2.0
	if l.pixels {
		x, y := l.center(row, col)
		return x - halfWidth, y - l.radius/2
	}
	x, y := GetImagePosition(InvertedRow(mapHeight, row), col, l.radius)
	return x - halfWidth, y - l.radius*1.5
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

// PhysicalCityNameLabel returns tile (row, col)'s city label in white.
func PhysicalCityNameLabel(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) ColoredText {
	cityName := cityNameText(mapData, row, col)
	x, y := cityLabelPosition(mapLayout(radius), mapHeight, row, col, cityName)
	return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
}

// PoliticalCityNameLabel returns tile (row, col)'s city label in its owning civ's color (white if unrecognized).
func PoliticalCityNameLabel(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, l tileLayout) ColoredText {
	cityName := cityNameText(mapData, row, col)
	x, y := cityLabelPosition(l, mapHeight, row, col, cityName)

	tileColor := fileio.GetPoliticalMapTileColor(mapData, row, col)
	renderColor, ok := civColorMap[tileColor]
	if !ok {
		return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
	}

	textColor := markerColor(shadesFor(renderColor, tileIsMinor(mapData, row, col)).accent)
	return ColoredText{Text: cityName, X: x, Y: y, R: textColor.R, G: textColor.G, B: textColor.B}
}

func GetImagePosition(i int, j int, radius float64) (float64, float64) {
	angle := math.Pi / 6

	x := (radius * 1.5) + float64(j)*(2*radius*math.Cos(angle))
	y := radius + float64(i)*radius*(1+math.Sin(angle))
	if i%2 == 1 {
		x += radius * math.Cos(angle)
	}
	return x, y
}
