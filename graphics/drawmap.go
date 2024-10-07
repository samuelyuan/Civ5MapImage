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

const (
	radius = 16.0
)

var (
	NeighborOdd  = [6][2]int{{1, 1}, {0, 1}, {-1, 0}, {0, -1}, {1, -1}, {1, 0}}
	NeighborEven = [6][2]int{{0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, 0}}
)

func getNeighbors(x int, y int) [6][2]int {
	var offset [6][2]int
	if y%2 == 1 {
		offset = NeighborOdd
	} else {
		offset = NeighborEven
	}

	neighbors := [6][2]int{}
	for i := 0; i < 6; i++ {
		newX := x + offset[i][0]
		newY := y + offset[i][1]
		neighbors[i][0] = newX
		neighbors[i][1] = newY
	}
	return neighbors
}

func getImagePosition(i int, j int) (float64, float64) {
	angle := math.Pi / 6

	x := (radius * 1.5) + float64(j)*(2*radius*math.Cos(angle))
	y := radius + float64(i)*radius*(1+math.Sin(angle))
	if i%2 == 1 {
		x += radius * math.Cos(angle)
	}
	return x, y
}

func getPhysicalMapTileColor(mapData *fileio.Civ5MapData, row int, column int) color.RGBA {
	terrainString := fileio.GetTerrainString(mapData, row, column)
	switch terrainString {
	case "TERRAIN_GRASS":
		return color.RGBA{105, 125, 54, 255}
	case "TERRAIN_PLAINS":
		return color.RGBA{127, 121, 71, 255}
	case "TERRAIN_DESERT":
		return color.RGBA{200, 200, 164, 255}
	case "TERRAIN_TUNDRA":
		return color.RGBA{118, 123, 117, 255}
	case "TERRAIN_SNOW":
		return color.RGBA{238, 249, 255, 255}
	case "TERRAIN_COAST":
		return color.RGBA{95, 149, 149, 255}
	case "TERRAIN_OCEAN":
		return color.RGBA{47, 74, 93, 255}
	}

	// default
	return color.RGBA{0, 0, 0, 255}
}

func drawMountain(dc *gg.Context, imageX float64, imageY float64) {
	// draw base
	dc.DrawRegularPolygon(3, imageX, imageY, radius, math.Pi)
	dc.SetRGB255(89, 90, 86) // gray
	dc.Fill()

	// draw mountain peak
	dc.DrawRegularPolygon(3, imageX, imageY+(radius/2), radius/2, math.Pi)
	dc.SetRGB255(234, 244, 253) // white
	dc.Fill()
}

func getNewCityColor(cityColor color.RGBA) color.RGBA {
	return interpolateColor(cityColor, color.RGBA{255, 255, 255, 255}, 0.2)
}

func drawCityIcon(dc *gg.Context, imageX float64, imageY float64, cityColor color.RGBA) {
	iconColor := getNewCityColor(cityColor)
	dc.DrawRectangle(imageX-(radius/5), imageY-(radius/5), radius/2, radius/2)
	dc.SetRGB255(int(iconColor.R), int(iconColor.G), int(iconColor.B))
	dc.Fill()
}

func drawTerrainTiles(dc *gg.Context, mapData *fileio.Civ5MapData, mapHeight int, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x, y := getImagePosition(i, j)

			dc.DrawRegularPolygon(6, x, y, radius, math.Pi/2)
			tileColor := getPhysicalMapTileColor(mapData, i, j)
			dc.SetRGB255(int(tileColor.R), int(tileColor.G), int(tileColor.B))
			dc.Fill()

			// Draw mountains
			if fileio.TileHasMountain(mapData, i, j) {
				drawMountain(dc, x, y)
			}

			// Draw cities
			if fileio.TileHasCity(mapData, i, j) {
				drawCityIcon(dc, x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
}

func getTileCivName(mapData *fileio.Civ5MapData, row int, column int) string {
	tileOwner := mapData.MapTileImprovements[row][column].Owner
	if tileOwner == 0xFF {
		return ""
	}
	civIndex := mapData.CityOwnerIndexMap[tileOwner]
	if civIndex < len(mapData.Civ5PlayerData) {
		return mapData.Civ5PlayerData[civIndex].CivType
	}
	return ""
}

func interpolateColor(color1 color.RGBA, color2 color.RGBA, t float64) color.RGBA {
	// t should be between 0.0 and 1.0
	return color.RGBA{
		uint8(float64(color1.R) + (float64(color2.R)-float64(color1.R))*t),
		uint8(float64(color1.G) + (float64(color2.G)-float64(color1.G))*t),
		uint8(float64(color1.B) + (float64(color2.B)-float64(color1.B))*t),
		255,
	}
}

func drawTerritoryTiles(dc *gg.Context, mapData *fileio.Civ5MapData, mapHeight int, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x, y := getImagePosition(i, j)

			dc.DrawRegularPolygon(6, x, y, radius, math.Pi/2)

			cityColor := color.RGBA{255, 255, 255, 255}
			if fileio.IsWaterTile(mapData, i, j) {
				terrainTileColor := getPhysicalMapTileColor(mapData, i, j)
				dc.SetRGB255(int(terrainTileColor.R), int(terrainTileColor.G), int(terrainTileColor.B))
				dc.Fill()
			} else {
				tileColor := fileio.GetPoliticalMapTileColor(mapData, i, j)

				renderColor, ok := civColorMap[tileColor]

				if ok {
					white := color.RGBA{255, 255, 255, 255}
					if strings.Contains(getTileCivName(mapData, i, j), "MINOR") {
						// Invert city state colors
						background := renderColor.InnerColor
						cityColor = renderColor.OuterColor
						newBackground := interpolateColor(background, white, 0.1)
						dc.SetRGB255(int(newBackground.R), int(newBackground.G), int(newBackground.B))
					} else {
						background := renderColor.OuterColor
						cityColor = renderColor.InnerColor
						newBackground := interpolateColor(background, white, 0.2)
						dc.SetRGB255(int(newBackground.R), int(newBackground.G), int(newBackground.B))
					}
					dc.Fill()
				} else if tileColor != "" {
					// No color, but tile is owned by civ or city state
					dc.SetRGB255(0, 0, 0)
					dc.Fill()
				} else {
					// Territory not owned by anyone
					terrainTileColor := getPhysicalMapTileColor(mapData, i, j)
					dc.SetRGB255(int(terrainTileColor.R), int(terrainTileColor.G), int(terrainTileColor.B))
					dc.Fill()
				}
			}

			// Draw mountains
			if mapData.MapTiles[i][j].Elevation == 2 {
				drawMountain(dc, x, y)
			}

			// Draw cities
			if fileio.TileHasCity(mapData, i, j) {
				drawCityIcon(dc, x, y, cityColor)
			}
		}
	}
}

func drawRivers(dc *gg.Context, mapData *fileio.Civ5MapData, mapHeight int, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x, y := getImagePosition(i, j)
			dc.SetRGB255(95, 150, 148)

			riverData := mapData.MapTiles[i][j].RiverData
			isRiverSouthwest := ((riverData >> 2) & 1) != 0
			isRiverSoutheast := ((riverData >> 1) & 1) != 0
			isRiverEast := (riverData & 1) != 0

			// Southwest river
			if isRiverSouthwest {
				angleSW1 := (math.Pi / 6) + float64(3)*(math.Pi/3)
				angleSW2 := (math.Pi / 6) + float64(4)*(math.Pi/3)
				x1 := x + radius*math.Cos(angleSW1)
				y1 := y + radius*math.Sin(angleSW1)
				x2 := x + radius*math.Cos(angleSW2)
				y2 := y + radius*math.Sin(angleSW2)
				dc.DrawLine(x1, y1, x2, y2)
				dc.Stroke()
			}

			// Southeast river
			if isRiverSoutheast {
				angleSE1 := (math.Pi / 6) + float64(4)*(math.Pi/3)
				angleSE2 := (math.Pi / 6) + float64(5)*(math.Pi/3)
				x1 := x + radius*math.Cos(angleSE1)
				y1 := y + radius*math.Sin(angleSE1)
				x2 := x + radius*math.Cos(angleSE2)
				y2 := y + radius*math.Sin(angleSE2)
				dc.DrawLine(x1, y1, x2, y2)
				dc.Stroke()
			}

			// East river
			if isRiverEast {
				angleE1 := (math.Pi / 6) + float64(5)*(math.Pi/3)
				angleE2 := (math.Pi / 6) + float64(6)*(math.Pi/3)
				x1 := x + radius*math.Cos(angleE1)
				y1 := y + radius*math.Sin(angleE1)
				x2 := x + radius*math.Cos(angleE2)
				y2 := y + radius*math.Sin(angleE2)
				dc.DrawLine(x1, y1, x2, y2)
				dc.Stroke()
			}
		}
	}
}

func drawRoads(dc *gg.Context, mapData *fileio.Civ5MapData, mapHeight int, mapWidth int) {
	// Draw roads between tiles
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x1, y1 := getImagePosition(i, j)

			routeType := mapData.MapTileImprovements[i][j].RouteType
			if routeType == 255 {
				continue
			}

			neighbors := getNeighbors(j, i)
			for n := 0; n < len(neighbors); n++ {
				newX := neighbors[n][0]
				newY := neighbors[n][1]
				if newX >= 0 && newY >= 0 && newX < mapWidth && newY < mapHeight {
					if mapData.MapTileImprovements[newY][newX].RouteType != 255 ||
						mapData.MapTileImprovements[newY][newX].CityName != "" {
						x2, y2 := getImagePosition(newY, newX)

						if routeType == 1 {
							// Railroad
							dc.SetLineWidth(2.0)
							dc.SetRGB255(76, 51, 0)
						} else if routeType == 0 {
							// Road
							dc.SetLineWidth(1.0)
							dc.SetRGB255(51, 51, 51)
						} else {
							// Unknown
							dc.SetLineWidth(1.0)
							dc.SetRGB255(0, 0, 0)
						}

						// Draw only up to midpoint, which would be the tile border
						borderX := (x1 + x2) / 2.0
						borderY := (y1 + y2) / 2.0

						dc.DrawLine(x1, y1, borderX, borderY)
						dc.Stroke()
					}
				}
			}
		}
	}
}

func DrawPhysicalMap(mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := getImagePosition(mapHeight, mapWidth)
	dc := gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	dc.InvertY()

	drawTerrainTiles(dc, mapData, mapHeight, mapWidth)
	drawRivers(dc, mapData, mapHeight, mapWidth)
	drawRoads(dc, mapData, mapHeight, mapWidth)

	// Draw city names on top of hexes
	dc.InvertY()
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			// Invert depth because the map is inverted
			x, y := getImagePosition(mapHeight-i, j)

			tile := mapData.MapTileImprovements[i][j]
			dc.SetRGB255(255, 255, 255)
			cityName := string(strings.Split(string(tile.CityName[:]), "\x00")[0])
			dc.DrawString(cityName, x-(5.0*float64(len(cityName))/2.0), y-radius*1.5)
		}
	}

	return dc.Image()
}

func drawBorders(dc *gg.Context, mapData *fileio.Civ5MapData, mapHeight int, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x1, y1 := getImagePosition(i, j)
			neighbors := getNeighbors(j, i)
			currentTileOwner := mapData.MapTileImprovements[i][j].Owner
			if currentTileOwner == 0xFF {
				continue
			}

			tileColor := fileio.GetPoliticalMapTileColor(mapData, i, j)
			renderColor, ok := civColorMap[tileColor]
			borderColor := color.RGBA{255, 255, 255, 255}
			if ok {
				if strings.Contains(getTileCivName(mapData, i, j), "MINOR") {
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
						edgeX1 := x1 + (radius-1)*math.Cos(angle1)
						edgeY1 := y1 + (radius-1)*math.Sin(angle1)
						edgeX2 := x1 + (radius-1)*math.Cos(angle2)
						edgeY2 := y1 + (radius-1)*math.Sin(angle2)

						dc.SetRGB255(int(borderColor.R), int(borderColor.G), int(borderColor.B))
						dc.SetLineWidth(1.5)
						dc.DrawLine(edgeX1, edgeY1, edgeX2, edgeY2)
						dc.Stroke()
					}
				}
			}
		}
	}
	dc.SetLineWidth(1.0)
}

func drawCityNames(dc *gg.Context, mapData *fileio.Civ5MapData, mapHeight int, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			// Invert depth because the map is inverted
			x, y := getImagePosition(mapHeight-i, j)

			tile := mapData.MapTileImprovements[i][j]
			tileColor := fileio.GetPoliticalMapTileColor(mapData, i, j)
			renderColor, ok := civColorMap[tileColor]
			if ok {
				var cityColor color.RGBA
				if strings.Contains(getTileCivName(mapData, i, j), "MINOR") {
					cityColor = renderColor.OuterColor
				} else {
					cityColor = renderColor.InnerColor
				}
				textColor := getNewCityColor(cityColor)
				dc.SetRGB255(int(textColor.R), int(textColor.G), int(cityColor.B))
			} else {
				dc.SetRGB255(255, 255, 255)
			}

			cityName := string(strings.Split(string(tile.CityName[:]), "\x00")[0])
			dc.DrawString(cityName, x-(6.0*float64(len(cityName))/2.0), y-radius*1.5)
		}
	}
}

func DrawPoliticalMap(mapData *fileio.Civ5MapData) image.Image {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := getImagePosition(mapHeight, mapWidth)
	dc := gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	dc.InvertY()

	drawTerritoryTiles(dc, mapData, mapHeight, mapWidth)
	drawBorders(dc, mapData, mapHeight, mapWidth)
	drawRivers(dc, mapData, mapHeight, mapWidth)
	drawRoads(dc, mapData, mapHeight, mapWidth)

	dc.InvertY()
	// Draw city names on top of hexes
	drawCityNames(dc, mapData, mapHeight, mapWidth)

	return dc.Image()
}

func SaveImage(outputFilename string, im image.Image) {
	gg.SavePNG(outputFilename, im)
	fmt.Println("Saved image to", outputFilename)
}
