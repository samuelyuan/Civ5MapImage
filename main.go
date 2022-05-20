package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"

	"github.com/fogleman/gg"
)

type MapHeader struct {
	ScenarioVersion        uint8
	Width                  uint32
	Height                 uint32
	Players                uint8
	Settings               uint32
	TerrainDataSize        uint32
	FeatureTerrainDataSize uint32
	FeatureWonderDataSize  uint32
	ResourceDataSize       uint32
	ModDataSize            uint32
	MapNameLength          uint32
	MapDescriptionLength   uint32
}

type MapTile struct {
	TerrainType        uint8
	ResourceType       uint8
	FeatureTerrainType uint8
	RiverData          uint8
	Elevation          uint8
	Continent          uint8
	FeatureWonderType  uint8
	ResourceArmount    uint8
}

type MapData struct {
	MapHeader   MapHeader
	TerrainList []string
	MapTiles    [][]*MapTile
}

func byteArrayToStringArray(byteArray []byte) []string {
	str := ""
	arr := make([]string, 0)
	for i := 0; i < len(byteArray); i++ {
		if byteArray[i] == 0 {
			arr = append(arr, str)
			str = ""
		} else {
			str += string(byteArray[i])
		}
	}
	return arr
}

func readData(filename string) (*MapData, error) {
	inputFile, err := os.Open(filename)
	defer inputFile.Close()
	if err != nil {
		log.Fatal("Failed to load map: ", err)
		return nil, err
	}
	fi, err := inputFile.Stat()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	fileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), fileLength)

	mapHeader := MapHeader{}
	if err := binary.Read(streamReader, binary.LittleEndian, &mapHeader); err != nil {
		return nil, err
	}

	version := mapHeader.ScenarioVersion & 0xF
	scenario := mapHeader.ScenarioVersion >> 4
	fmt.Println("Scenario: ", scenario)
	fmt.Println("Version: ", version)

	terrainDataBytes := make([]byte, mapHeader.TerrainDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &terrainDataBytes); err != nil {
		return nil, err
	}
	terrainList := byteArrayToStringArray(terrainDataBytes)
	fmt.Println("Terrain data: ", terrainList)

	featureTerrainDataBytes := make([]byte, mapHeader.FeatureTerrainDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &featureTerrainDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Feature terrain data: ", byteArrayToStringArray(featureTerrainDataBytes))

	featureWonderDataBytes := make([]byte, mapHeader.FeatureWonderDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &featureWonderDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Feature wonder data: ", byteArrayToStringArray(featureWonderDataBytes))

	resourceDataBytes := make([]byte, mapHeader.ResourceDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &resourceDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Resource data: ", byteArrayToStringArray(resourceDataBytes))

	modDataBytes := make([]byte, mapHeader.ModDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &modDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Mod data: ", string(modDataBytes))

	mapNameBytes := make([]byte, mapHeader.MapNameLength)
	if err := binary.Read(streamReader, binary.LittleEndian, &mapNameBytes); err != nil {
		return nil, err
	}
	fmt.Println("Map name: ", string(mapNameBytes))

	mapDescriptionBytes := make([]byte, mapHeader.MapDescriptionLength)
	if err := binary.Read(streamReader, binary.LittleEndian, &mapDescriptionBytes); err != nil {
		return nil, err
	}
	fmt.Println("Map description: ", string(mapDescriptionBytes))

	// Earlier versions don't have this field
	if version >= 11 {
		unknownStringLength := uint32(0)
		if err := binary.Read(streamReader, binary.LittleEndian, &unknownStringLength); err != nil {
			return nil, err
		}

		unknownStringBytes := make([]byte, unknownStringLength)
		if err := binary.Read(streamReader, binary.LittleEndian, &unknownStringBytes); err != nil {
			return nil, err
		}
		fmt.Println("Unknown string: ", string(unknownStringBytes))
	}

	mapTiles := make([][]*MapTile, mapHeader.Height)
	for i := 0; i < int(mapHeader.Height); i++ {
		mapTiles[i] = make([]*MapTile, mapHeader.Width)
		for j := 0; j < int(mapHeader.Width); j++ {
			tile := MapTile{}
			if err := binary.Read(streamReader, binary.LittleEndian, &tile); err != nil {
				return nil, err
			}
			mapTiles[i][j] = &tile
		}
	}

	mapData := &MapData{
		MapHeader:   mapHeader,
		TerrainList: terrainList,
		MapTiles:    mapTiles,
	}
	return mapData, err
}

func drawMap(mapData *MapData, outputFilename string) {
	radius := 10.0
	angle := math.Pi / 6
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	maxImageWidth := (radius * 1.5) + float64(mapWidth)*(2*radius*math.Cos(angle))
	maxImageHeight := radius + float64(mapHeight)*radius*(1+math.Sin(angle))
	dc := gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	fmt.Println("Map height: ", mapHeight, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	dc.InvertY()

	for i := 0; i < len(mapData.MapTiles); i++ {
		for j := 0; j < len(mapData.MapTiles[i]); j++ {
			x := (radius * 1.5) + float64(j)*(2*radius*math.Cos(angle))
			y := radius + float64(i)*radius*(1+math.Sin(angle))
			if i%2 == 1 {
				x += radius * math.Cos(angle)
			}
			dc.DrawRegularPolygon(6, x, y, radius, math.Pi/2)

			terrainType := mapData.MapTiles[i][j].TerrainType
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
	mapData, err := readData(*inputPtr)
	if err != nil {
		log.Fatal("Failed to read input file: ", err)
	}

	drawMap(mapData, *outputPtr)
}
