package graphics

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/fogleman/gg"
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

// DrawTerrainTiles draws all terrain tiles for the physical map
func (mr *MapRenderer) DrawTerrainTiles(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x, y := fileio.GetImagePosition(i, j, mr.config.Radius)

			canvas.DrawRegularPolygon(6, x, y, mr.config.Radius, math.Pi/2)
			terrainString := fileio.GetTerrainString(mapData, i, j)
			tileColor := fileio.GetPhysicalMapTileColor(terrainString)
			canvas.SetColor(tileColor.R, tileColor.G, tileColor.B)
			canvas.Fill()

			// Draw mountains
			if fileio.TileHasMountain(mapData, i, j) {
				mr.DrawMountain(canvas, x, y)
			}

			// Draw cities
			if len(mapData.MapTileImprovements) > 0 {
				if fileio.TileHasCity(mapData, i, j) {
					mr.DrawCityIcon(canvas, x, y, color.RGBA{255, 255, 255, 255})
				}
			}
		}
	}
}

// InterpolateColor blends two colors by the given factor
func (mr *MapRenderer) InterpolateColor(color1, color2 color.RGBA, t float64) color.RGBA {
	// t should be between 0.0 and 1.0
	return color.RGBA{
		uint8(float64(color1.R) + (float64(color2.R)-float64(color1.R))*t),
		uint8(float64(color1.G) + (float64(color2.G)-float64(color1.G))*t),
		uint8(float64(color1.B) + (float64(color2.B)-float64(color1.B))*t),
		255,
	}
}

// DrawTerritoryTiles draws territory tiles for the political map
func (mr *MapRenderer) DrawTerritoryTiles(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x, y := fileio.GetImagePosition(i, j, mr.config.Radius)

			canvas.DrawRegularPolygon(6, x, y, mr.config.Radius, math.Pi/2)

			cityColor := color.RGBA{255, 255, 255, 255}
			if fileio.IsWaterTile(mapData, i, j) {
				terrainString := fileio.GetTerrainString(mapData, i, j)
				terrainTileColor := fileio.GetPhysicalMapTileColor(terrainString)
				canvas.SetColor(terrainTileColor.R, terrainTileColor.G, terrainTileColor.B)
				canvas.Fill()
			} else {
				tileColor := fileio.GetPoliticalMapTileColor(mapData, i, j)

				renderColor, ok := civColorMap[tileColor]

				if ok {
					white := color.RGBA{255, 255, 255, 255}
					if strings.Contains(fileio.GetTileCivName(mapData, i, j), "MINOR") {
						// Invert city state colors
						background := renderColor.InnerColor
						cityColor = renderColor.OuterColor
						newBackground := mr.InterpolateColor(background, white, 0.1)
						canvas.SetColor(newBackground.R, newBackground.G, newBackground.B)
					} else {
						background := renderColor.OuterColor
						cityColor = renderColor.InnerColor
						newBackground := mr.InterpolateColor(background, white, 0.2)
						canvas.SetColor(newBackground.R, newBackground.G, newBackground.B)
					}
					canvas.Fill()
				} else if tileColor != "" {
					// No color, but tile is owned by civ or city state
					canvas.SetColor(0, 0, 0)
					canvas.Fill()
				} else {
					// Territory not owned by anyone
					terrainString := fileio.GetTerrainString(mapData, i, j)
					terrainTileColor := fileio.GetPhysicalMapTileColor(terrainString)
					canvas.SetColor(terrainTileColor.R, terrainTileColor.G, terrainTileColor.B)
					canvas.Fill()
				}
			}

			// Draw mountains
			if mapData.MapTiles[i][j].Elevation == 2 {
				mr.DrawMountain(canvas, x, y)
			}

			// Draw cities
			if fileio.TileHasCity(mapData, i, j) {
				mr.DrawCityIcon(canvas, x, y, cityColor)
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

			riverData := mapData.MapTiles[i][j].RiverData
			isRiverSouthwest := ((riverData >> 2) & 1) != 0
			isRiverSoutheast := ((riverData >> 1) & 1) != 0
			isRiverEast := (riverData & 1) != 0

			// Southwest river
			if isRiverSouthwest {
				angleSW1 := (math.Pi / 6) + float64(3)*(math.Pi/3)
				angleSW2 := (math.Pi / 6) + float64(4)*(math.Pi/3)
				x1 := x + mr.config.Radius*math.Cos(angleSW1)
				y1 := y + mr.config.Radius*math.Sin(angleSW1)
				x2 := x + mr.config.Radius*math.Cos(angleSW2)
				y2 := y + mr.config.Radius*math.Sin(angleSW2)
				canvas.DrawLine(x1, y1, x2, y2)
				canvas.Stroke()
			}

			// Southeast river
			if isRiverSoutheast {
				angleSE1 := (math.Pi / 6) + float64(4)*(math.Pi/3)
				angleSE2 := (math.Pi / 6) + float64(5)*(math.Pi/3)
				x1 := x + mr.config.Radius*math.Cos(angleSE1)
				y1 := y + mr.config.Radius*math.Sin(angleSE1)
				x2 := x + mr.config.Radius*math.Cos(angleSE2)
				y2 := y + mr.config.Radius*math.Sin(angleSE2)
				canvas.DrawLine(x1, y1, x2, y2)
				canvas.Stroke()
			}

			// East river
			if isRiverEast {
				angleE1 := (math.Pi / 6) + float64(5)*(math.Pi/3)
				angleE2 := (math.Pi / 6) + float64(6)*(math.Pi/3)
				x1 := x + mr.config.Radius*math.Cos(angleE1)
				y1 := y + mr.config.Radius*math.Sin(angleE1)
				x2 := x + mr.config.Radius*math.Cos(angleE2)
				y2 := y + mr.config.Radius*math.Sin(angleE2)
				canvas.DrawLine(x1, y1, x2, y2)
				canvas.Stroke()
			}
		}
	}
}

// DrawRoads draws roads between tiles
func (mr *MapRenderer) DrawRoads(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	// Draw roads between tiles
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x1, y1 := fileio.GetImagePosition(i, j, mr.config.Radius)

			routeType := mapData.MapTileImprovements[i][j].RouteType
			if routeType == 255 {
				continue
			}

			neighbors := fileio.GetNeighbors(j, i)
			for n := 0; n < len(neighbors); n++ {
				newX := neighbors[n][0]
				newY := neighbors[n][1]
				if newX >= 0 && newY >= 0 && newX < mapWidth && newY < mapHeight {
					if mapData.MapTileImprovements[newY][newX].RouteType != 255 ||
						mapData.MapTileImprovements[newY][newX].CityName != "" {
						x2, y2 := fileio.GetImagePosition(newY, newX, mr.config.Radius)

						if routeType == 1 {
							// Railroad
							canvas.SetLineWidth(2.0)
							canvas.SetColor(76, 51, 0)
						} else if routeType == 0 {
							// Road
							canvas.SetLineWidth(1.0)
							canvas.SetColor(51, 51, 51)
						} else {
							// Unknown
							canvas.SetLineWidth(1.0)
							canvas.SetColor(0, 0, 0)
						}

						// Draw only up to midpoint, which would be the tile border
						borderX := (x1 + x2) / 2.0
						borderY := (y1 + y2) / 2.0

						canvas.DrawLine(x1, y1, borderX, borderY)
						canvas.Stroke()
					}
				}
			}
		}
	}
}

// DrawPhysicalMap creates a physical map image using the abstracted canvas
func (mr *MapRenderer) DrawPhysicalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, mr.config.Radius)

	// Set canvas size if it's a DrawingContext
	if dc, ok := canvas.(*DrawingContext); ok {
		dc.dc = gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	}

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
		mr.DrawCityNames(canvas, mapData, mapHeight, mapWidth)
	}

	return canvas.Image()
}

// DrawBorders draws borders between different territories
func (mr *MapRenderer) DrawBorders(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x1, y1 := fileio.GetImagePosition(i, j, mr.config.Radius)
			neighbors := fileio.GetNeighbors(j, i)
			currentTileOwner := mapData.MapTileImprovements[i][j].Owner
			if fileio.IsInvalidTileOwner(currentTileOwner) {
				continue
			}

			tileColor := fileio.GetPoliticalMapTileColor(mapData, i, j)
			renderColor, ok := civColorMap[tileColor]
			borderColor := color.RGBA{255, 255, 255, 255}
			if ok {
				if strings.Contains(fileio.GetTileCivName(mapData, i, j), "MINOR") {
					// invert city state colors
					borderColor = renderColor.OuterColor
				} else {
					borderColor = renderColor.InnerColor
				}
			}

			for n := 0; n < len(neighbors); n++ {
				newX := neighbors[n][0]
				newY := neighbors[n][1]
				if newX >= 0 && newY >= 0 && newX < mapWidth && newY < mapHeight {
					otherTileOwner := mapData.MapTileImprovements[newY][newX].Owner
					if currentTileOwner != otherTileOwner {
						angle1 := (math.Pi / 6) + float64(n)*(math.Pi/3)
						angle2 := (math.Pi / 6) + float64(n+1)*(math.Pi/3)
						edgeX1 := x1 + (mr.config.Radius-1)*math.Cos(angle1)
						edgeY1 := y1 + (mr.config.Radius-1)*math.Sin(angle1)
						edgeX2 := x1 + (mr.config.Radius-1)*math.Cos(angle2)
						edgeY2 := y1 + (mr.config.Radius-1)*math.Sin(angle2)

						canvas.SetColor(borderColor.R, borderColor.G, borderColor.B)
						canvas.SetLineWidth(1.5)
						canvas.DrawLine(edgeX1, edgeY1, edgeX2, edgeY2)
						canvas.Stroke()
					}
				}
			}
		}
	}
	canvas.SetLineWidth(1.0)
}

// DrawCityNames draws city names on the map
func (mr *MapRenderer) DrawCityNames(canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			// Invert depth because the map is inverted
			x, y := fileio.GetImagePosition(mapHeight-i, j, mr.config.Radius)

			tile := mapData.MapTileImprovements[i][j]
			tileColor := fileio.GetPoliticalMapTileColor(mapData, i, j)
			renderColor, ok := civColorMap[tileColor]
			if ok {
				var cityColor color.RGBA
				if strings.Contains(fileio.GetTileCivName(mapData, i, j), "MINOR") {
					cityColor = renderColor.OuterColor
				} else {
					cityColor = renderColor.InnerColor
				}
				textColor := mr.GetNewCityColor(cityColor)
				canvas.SetColor(textColor.R, textColor.G, cityColor.B)
			} else {
				canvas.SetColor(255, 255, 255)
			}

			cityName := string(strings.Split(string(tile.CityName[:]), "\x00")[0])
			canvas.DrawString(cityName, x-(6.0*float64(len(cityName))/2.0), y-mr.config.Radius*1.5)
		}
	}
}

// DrawPoliticalMap creates a political map image using the abstracted canvas
func (mr *MapRenderer) DrawPoliticalMap(canvas Canvas, mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, mr.config.Radius)

	// Set canvas size if it's a DrawingContext
	if dc, ok := canvas.(*DrawingContext); ok {
		dc.dc = gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	}

	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	canvas.InvertY()

	mr.DrawTerritoryTiles(canvas, mapData, mapHeight, mapWidth)
	mr.DrawBorders(canvas, mapData, mapHeight, mapWidth)
	mr.DrawRivers(canvas, mapData, mapHeight, mapWidth)
	mr.DrawRoads(canvas, mapData, mapHeight, mapWidth)

	canvas.InvertY()
	// Draw city names on top of hexes
	mr.DrawCityNames(canvas, mapData, mapHeight, mapWidth)

	return canvas.Image()
}

// SaveImage saves the image to a file
func (mr *MapRenderer) SaveImage(canvas Canvas, outputFilename string) error {
	return canvas.SavePNG(outputFilename)
}
