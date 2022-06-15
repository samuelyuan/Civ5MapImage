package fileio

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
)

type Civ5MapHeader struct {
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

type Civ5MapTile struct {
	TerrainType        uint8
	ResourceType       uint8
	FeatureTerrainType uint8
	RiverData          uint8
	Elevation          uint8
	Continent          uint8
	FeatureWonderType  uint8
	ResourceAmount    uint8
}

type Civ5MapData struct {
	MapHeader   Civ5MapHeader
	TerrainList []string
	MapTiles    [][]*Civ5MapTile
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

func ReadCiv5MapFile(filename string) (*Civ5MapData, error) {
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

	mapHeader := Civ5MapHeader{}
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

	mapTiles := make([][]*Civ5MapTile, mapHeader.Height)
	for i := 0; i < int(mapHeader.Height); i++ {
		mapTiles[i] = make([]*Civ5MapTile, mapHeader.Width)
		for j := 0; j < int(mapHeader.Width); j++ {
			tile := Civ5MapTile{}
			if err := binary.Read(streamReader, binary.LittleEndian, &tile); err != nil {
				return nil, err
			}
			mapTiles[i][j] = &tile
		}
	}

	mapData := &Civ5MapData{
		MapHeader:   mapHeader,
		TerrainList: terrainList,
		MapTiles:    mapTiles,
	}
	return mapData, err
}
