package graphics

import (
	"fmt"
	"image"
	"image/color"
	"math"

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

// DrawRivers draws rivers on the map
func (mr *MapRenderer) DrawRivers(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x, y := fileio.GetImagePosition(i, j, mr.config.Radius)
			canvas.SetColor(95, 150, 148)

			for _, edge := range RiverEdgesForTile(mapData.MapTiles[i][j].RiverData, x, y, mr.config.Radius) {
				canvas.DrawLine(edge.X1, edge.Y1, edge.X2, edge.Y2)
				canvas.Stroke()
			}
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
	if len(mapData.MapTileImprovements) > 0 {
		mr.DrawRoads(canvas, mapData, mapHeight, mapWidth)
	}

	// Draw city names on top of hexes
	canvas.InvertY()

	if len(mapData.MapTileImprovements) > 0 {
		mr.DrawPhysicalCityNames(canvas, mapData, mapHeight, mapWidth)
	}

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
	canvas.SetLineWidth(1.0)
}

// DrawPhysicalCityNames draws city names on the map (white text for physical maps)
func (mr *MapRenderer) DrawPhysicalCityNames(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			label := CityNameLabel(mapData, mapHeight, mapWidth, i, j, mr.config.Radius)
			canvas.SetColor(label.R, label.G, label.B)
			canvas.DrawString(label.Text, label.X, label.Y)
		}
	}
}

// DrawPoliticalCityNames draws city names with political colors
func (mr *MapRenderer) DrawPoliticalCityNames(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Early exit if no improvement data is present
	if len(mapData.MapTileImprovements) == 0 {
		return
	}

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			label := PoliticalCityNameLabel(mapData, mapHeight, mapWidth, i, j, mr.config.Radius)
			canvas.SetColor(label.R, label.G, label.B)
			canvas.DrawString(label.Text, label.X, label.Y)
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

// SaveImage saves the image to a file
func (mr *MapRenderer) SaveImage(canvas Canvas, outputFilename string) error {
	return canvas.SavePNG(outputFilename)
}
