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

func getImagePosition(i int, j int) (float64, float64) {
	angle := math.Pi / 6

	x := (radius * 1.5) + float64(j)*(2*radius*math.Cos(angle))
	y := radius + float64(i)*radius*(1+math.Sin(angle))
	if i%2 == 1 {
		x += radius * math.Cos(angle)
	}
	return x, y
}

func drawMap(mapData *fileio.Civ5MapData, outputFilename string) {
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := getImagePosition(mapHeight, mapWidth)
	dc := gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	dc.InvertY()

	for i := 0; i < len(mapData.MapTiles); i++ {
		for j := 0; j < len(mapData.MapTiles[i]); j++ {
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
