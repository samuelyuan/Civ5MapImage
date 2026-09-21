package graphics

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// Line represents a line with start and end points.
type Line struct {
	X1, Y1, X2, Y2 float64
}

// hexVertex returns vertex i (0-5) of the pointy-top hex at (centerX, centerY) in y-down pixels: vertex 0 is 30 degrees
// above the x axis on screen, and each next one is 60 degrees counterclockwise.
func hexVertex(i int, centerX, centerY, radius float64) (x, y float64) {
	angle := (math.Pi / 6) + float64(i)*(math.Pi/3)
	return centerX + radius*math.Cos(angle), centerY - radius*math.Sin(angle)
}

// getHexEdge returns edge edgeIndex (0-5) of the hexagon at (centerX, centerY), between vertices edgeIndex and edgeIndex+1.
func getHexEdge(edgeIndex int, centerX, centerY, radius float64) Line {
	x1, y1 := hexVertex(edgeIndex, centerX, centerY, radius)
	x2, y2 := hexVertex(edgeIndex+1, centerX, centerY, radius)
	return Line{X1: x1, Y1: y1, X2: x2, Y2: y2}
}

// tileLayout places tiles on a canvas in y-down pixels. The map file numbers rows from the bottom, so GetImagePosition's
// y points up; center flips it against the canvas's integer height.
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

// repaintRectPad is the margin a tile's repaint rect adds around its hex, for river and border stroke half-widths right at the tile's radius.
const repaintRectPad = 2.0

// repaintRect returns the pixel rect that repainting the tile at pos can touch, clipped to bounds.
func (l tileLayout) repaintRect(pos fileio.TilePos, bounds image.Rectangle) image.Rectangle {
	x, y := l.center(pos)
	return image.Rect(
		int(x-l.radius-repaintRectPad), int(y-l.radius-repaintRectPad),
		int(x+l.radius+repaintRectPad)+1, int(y+l.radius+repaintRectPad)+1,
	).Intersect(bounds)
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

// PoliticalHexTile returns the political position and fill of the tile at pos, plus its city icon color (white if unowned).
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

// RiverEdgesForTile decodes a RiverData bitmask into hex edge lines (southwest, southeast and east only; the rest belong to neighbors).
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

// RoadSegmentsForTile returns lines from the tile at pos to each neighbor with a route or city, or nil if it has no route (255).
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

// tileBorderColor returns the civ border color of the tile at pos, white if unrecognized.
func tileBorderColor(mapData *fileio.Civ5MapData, pos fileio.TilePos) color.RGBA {
	renderColor, ok := civColorMap[fileio.GetPoliticalMapTileColor(mapData, pos)]
	if !ok {
		return color.RGBA{255, 255, 255, 255}
	}
	return shadesFor(renderColor, tileIsMinor(mapData, pos)).accent
}

// BorderSegmentsForTile returns border lines against neighbors with a different owner, or nil if the tile has no valid owner.
func BorderSegmentsForTile(mapData *fileio.Civ5MapData, mapSize fileio.MapSize, pos fileio.TilePos, l tileLayout) []ColoredLine {
	currentTileOwner := mapData.TileImprovement(pos).Owner
	if fileio.IsInvalidTileOwner(currentTileOwner) {
		return nil
	}

	x1, y1 := l.center(pos)

	borderColor := tileBorderColor(mapData, pos)

	var segments []ColoredLine
	for n, neighbor := range fileio.GetNeighbors(pos) {
		if !neighbor.InMap(mapSize) {
			continue
		}

		otherTileOwner := mapData.TileImprovement(neighbor).Owner
		if currentTileOwner == otherTileOwner {
			continue
		}

		segments = append(segments, ColoredLine{
			Line:      getHexEdge(n, x1, y1, l.radius-1),
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

// trimCityName trims a stored city name at its first null byte.
func trimCityName(name string) string {
	if i := strings.IndexByte(name, 0); i >= 0 {
		return name[:i]
	}
	return name
}

// cityNameText returns the display city name of the tile at pos.
func cityNameText(mapData *fileio.Civ5MapData, pos fileio.TilePos) string {
	return trimCityName(mapData.TileImprovement(pos).CityName)
}

// cityLabelPosition returns the anchor for a tile's city label, centered above the tile.
func cityLabelPosition(l tileLayout, pos fileio.TilePos, cityName string) (float64, float64) {
	halfWidth := 6.0 * float64(len(cityName)) / 2.0
	x, y := l.center(pos)
	return x - halfWidth, y - l.radius/2
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

// PhysicalCityNameLabel returns the city label of the tile at pos, in white.
func PhysicalCityNameLabel(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout) ColoredText {
	cityName := cityNameText(mapData, pos)
	x, y := cityLabelPosition(l, pos, cityName)
	return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
}

// PoliticalCityNameLabel returns the city label of the tile at pos, in its owning civ's color (white if unrecognized).
func PoliticalCityNameLabel(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout) ColoredText {
	cityName := cityNameText(mapData, pos)
	x, y := cityLabelPosition(l, pos, cityName)

	tileColor := fileio.GetPoliticalMapTileColor(mapData, pos)
	renderColor, ok := civColorMap[tileColor]
	if !ok {
		return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
	}

	textColor := markerColor(shadesFor(renderColor, tileIsMinor(mapData, pos)).accent)
	return ColoredText{Text: cityName, X: x, Y: y, R: textColor.R, G: textColor.G, B: textColor.B}
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

// imageSize returns the width and height of the image a map of the given size needs: where the tile just past its far corner would be.
func imageSize(mapSize fileio.MapSize, radius float64) (width, height float64) {
	return GetImagePosition(fileio.TilePos{Row: mapSize.Height, Col: mapSize.Width}, radius)
}
