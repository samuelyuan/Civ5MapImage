package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"github.com/fogleman/gg"
	"github.com/samuelyuan/Civ5MapImage/fileio"
)

const (
	radius = 10.0
)

var (
	NeighborOdd  = [6][2]int{{-1, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}}
	NeighborEven = [6][2]int{{-1, 0}, {-1, -1}, {0, -1}, {1, 0}, {0, 1}, {-1, 1}}
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

func drawTiles(dc *gg.Context, mapData *fileio.Civ5MapData, mapHeight int, mapWidth int) {
	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			x, y := getImagePosition(i, j)
			dc.DrawRegularPolygon(6, x, y, radius, math.Pi/2)

			terrainType := mapData.MapTiles[i][j].TerrainType
			elevation := mapData.MapTiles[i][j].Elevation
			terrainString := mapData.TerrainList[terrainType]
			switch terrainString {
			case "TERRAIN_GRASS":
				dc.SetRGB255(105, 125, 54)
			case "TERRAIN_PLAINS":
				dc.SetRGB255(127, 121, 71)
			case "TERRAIN_DESERT":
				dc.SetRGB255(200, 200, 164)
			case "TERRAIN_TUNDRA":
				dc.SetRGB255(118, 123, 117)
			case "TERRAIN_SNOW":
				dc.SetRGB255(238, 249, 255)
			case "TERRAIN_COAST":
				dc.SetRGB255(95, 149, 149)
			case "TERRAIN_OCEAN":
				dc.SetRGB255(47, 74, 93)
			default:
				dc.SetRGB255(0, 0, 0)
			}

			dc.Fill()

			// Draw mountains
			if elevation == 2 {
				dc.DrawRegularPolygon(3, x, y, radius, math.Pi)
				dc.SetRGB255(89, 90, 86)
				dc.Fill()
				dc.DrawRegularPolygon(3, x, y+(radius/2), radius/2, math.Pi)
				dc.SetRGB255(234, 244, 253)
				dc.Fill()
			}

			// Draw cities
			if mapData.MapTileImprovements[i][j].CityId != -1 {
				dc.DrawRectangle(x-(radius/5), y-(radius/5), radius/2, radius/2)
				dc.SetRGB255(255, 255, 255)
				dc.Fill()
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
							dc.SetRGB255(76, 51, 0)
						} else if routeType == 0 {
							// Road
							dc.SetRGB255(51, 51, 51)
						} else {
							// Unknown
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

func drawMap(mapData *fileio.Civ5MapData, outputFilename string) {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := getImagePosition(mapHeight, mapWidth)
	dc := gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	dc.InvertY()

	drawTiles(dc, mapData, mapHeight, mapWidth)
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
			dc.DrawString(tile.CityName, x-(5.0*float64(len(tile.CityName))/2.0), y-radius*1.5)
		}
	}

	dc.SavePNG(outputFilename)
	fmt.Println("Saved image to", outputFilename)
}

func main() {
	inputPtr := flag.String("input", "", "Input filename")
	outputPtr := flag.String("output", "output.png", "Output filename")
	flag.Parse()

	fmt.Println("Input filename: ", *inputPtr)
	fmt.Println("Output filename: ", *outputPtr)
	mapData, err := fileio.ReadCiv5MapFile(*inputPtr)
	if err != nil {
		log.Fatal("Failed to read input file: ", err)
	}

	drawMap(mapData, *outputPtr)
}
