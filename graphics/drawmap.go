package graphics

import (
	"fmt"
	"image"
	"image/color"
	"math"

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
	railroadColor     = color.RGBA{76, 51, 0, 255}
	roadColor         = color.RGBA{51, 51, 51, 255}
	unknownRouteColor = color.RGBA{0, 0, 0, 255}
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

// DrawMountain draws a mountain icon at (imageX, imageY).
func (mr *MapRenderer) DrawMountain(canvas Canvas, l tileLayout, imageX, imageY float64) {
	// Draw base
	canvas.DrawRegularPolygon(3, imageX, imageY, l.radius, l.apexUpRotation())
	canvas.SetColor(mountainBaseColor.R, mountainBaseColor.G, mountainBaseColor.B)
	canvas.Fill()

	// Draw mountain peak
	canvas.DrawRegularPolygon(3, imageX, imageY+l.up(l.radius/2), l.radius/2, l.apexUpRotation())
	canvas.SetColor(mountainPeakColor.R, mountainPeakColor.G, mountainPeakColor.B)
	canvas.Fill()
}

// GetNewCityColor returns cityColor lightened for visibility.
func (mr *MapRenderer) GetNewCityColor(cityColor color.RGBA) color.RGBA {
	return markerColor(cityColor)
}

// DrawCityIcon draws a city icon at (imageX, imageY).
func (mr *MapRenderer) DrawCityIcon(canvas Canvas, l tileLayout, imageX, imageY float64, cityColor color.RGBA) {
	iconColor := mr.GetNewCityColor(cityColor)
	// The icon reaches 0.3r above the tile's center and 0.2r below it.
	top := math.Min(imageY+l.up(l.radius*3/10), imageY+l.up(-l.radius/5))
	canvas.DrawRectangle(imageX-(l.radius/5), top, l.radius/2, l.radius/2)
	canvas.SetColor(iconColor.R, iconColor.G, iconColor.B)
	canvas.Fill()
}

// drawEntity draws entity in the shape for its type.
func (mr *MapRenderer) drawEntity(canvas Canvas, l tileLayout, entity Entity) {
	switch entity.Type {
	case EntityMountain:
		mr.DrawMountain(canvas, l, entity.X, entity.Y)
	case EntityCity:
		mr.DrawCityIcon(canvas, l, entity.X, entity.Y, color.RGBA{entity.R, entity.G, entity.B, 255})
	}
}

// DrawTerrainTiles draws every terrain tile.
func (mr *MapRenderer) DrawTerrainTiles(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	l := mapLayout(mr.config.Radius)
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			hex := PhysicalHexTile(mapData, i, j, mr.config.Radius)
			canvas.DrawRegularPolygon(6, hex.X, hex.Y, mr.config.Radius, math.Pi/2)
			canvas.SetColor(hex.R, hex.G, hex.B)
			canvas.Fill()

			for _, entity := range TileEntities(mapData, i, j, l, color.RGBA{255, 255, 255, 255}) {
				mr.drawEntity(canvas, l, entity)
			}
		}
	}
}

// InterpolateColor blends two colors by the given factor (0.0 = color1, 1.0 = color2).
func (mr *MapRenderer) InterpolateColor(color1, color2 color.RGBA, t float64) color.RGBA {
	return blendColor(color1, color2, t)
}

// DrawTerritoryTiles draws every territory tile.
func (mr *MapRenderer) DrawTerritoryTiles(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	l := mapLayout(mr.config.Radius)
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			hex, cityColor := PoliticalHexTile(mapData, i, j, l)
			canvas.DrawRegularPolygon(6, hex.X, hex.Y, mr.config.Radius, math.Pi/2)
			canvas.SetColor(hex.R, hex.G, hex.B)
			canvas.Fill()

			for _, entity := range TileEntities(mapData, i, j, l, cityColor) {
				mr.drawEntity(canvas, l, entity)
			}
		}
	}
}

// drawRiverTile draws a single tile's river edges.
func (mr *MapRenderer) drawRiverTile(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, row, col int) {
	x, y := l.center(row, col)
	canvas.SetColor(riverColor.R, riverColor.G, riverColor.B)
	canvas.SetLineWidth(1.0)

	for _, edge := range RiverEdgesForTile(mapData.MapTiles[row][col].RiverData, x, y, l) {
		canvas.DrawLine(edge.X1, edge.Y1, edge.X2, edge.Y2)
		canvas.Stroke()
	}
}

// DrawRivers draws rivers on the map
func (mr *MapRenderer) DrawRivers(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	l := mapLayout(mr.config.Radius)
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			mr.drawRiverTile(canvas, l, mapData, i, j)
		}
	}
}

// DrawRoads draws roads between tiles
func (mr *MapRenderer) DrawRoads(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	l := mapLayout(mr.config.Radius)
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			for _, segment := range RoadSegmentsForTile(mapData, mapHeight, mapWidth, i, j, l) {
				canvas.SetLineWidth(segment.LineWidth)
				canvas.SetColor(segment.R, segment.G, segment.B)
				canvas.DrawLine(segment.Line.X1, segment.Line.Y1, segment.Line.X2, segment.Line.Y2)
				canvas.Stroke()
			}
		}
	}
}

// DrawPhysicalMap renders the physical map onto canvas.
func (mr *MapRenderer) DrawPhysicalMap(canvas FlippableCanvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := GetImagePosition(mapHeight, mapWidth, mr.config.Radius)

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

// DrawBorders draws borders between territories.
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

// DrawPhysicalCityNames draws white city names.
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
func (mr *MapRenderer) drawPoliticalCityNameTile(canvas Canvas, l tileLayout, mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int) {
	// Skip the label build and DrawString for the common no-city tile.
	if mapData.MapTileImprovements[row][col].CityName == "" {
		return
	}
	label := PoliticalCityNameLabel(mapData, mapHeight, mapWidth, row, col, l)
	if label.Text == "" {
		return
	}
	canvas.SetColor(label.R, label.G, label.B)
	canvas.DrawString(label.Text, label.X, label.Y)
}

// DrawPoliticalCityNames draws city names in political colors.
func (mr *MapRenderer) DrawPoliticalCityNames(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, l tileLayout) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			mr.drawPoliticalCityNameTile(canvas, l, mapData, mapHeight, mapWidth, i, j)
		}
	}
}

// DrawPoliticalMap renders the political map onto canvas.
func (mr *MapRenderer) DrawPoliticalMap(canvas FlippableCanvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := GetImagePosition(mapHeight, mapWidth, mr.config.Radius)

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
	mr.DrawPoliticalCityNames(canvas, mapData, mapHeight, mapWidth, mapLayout(mr.config.Radius))

	return canvas.Image()
}

// SaveImage saves canvas to outputFilename.
func (mr *MapRenderer) SaveImage(canvas Canvas, outputFilename string) error {
	return canvas.SavePNG(outputFilename)
}
