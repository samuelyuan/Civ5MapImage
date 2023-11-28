package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics"
)

func exportFileToJson(inputFilename string, outputFilename string) {
	inputFileExtension := filepath.Ext(inputFilename)

	if strings.ToLower(inputFileExtension) == ".civ5map" {
		fmt.Println("Reading civ5map file")
		mapData, err := fileio.ReadCiv5MapFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}

		fmt.Println("Exporting map to", outputFilename)
		fileio.ExportCiv5MapFile(mapData, outputFilename)
	} else if strings.ToLower(inputFileExtension) == ".civ5replay" {
		fmt.Println("Importing civ5replay data")
		replayData, err := fileio.ReadCiv5ReplayFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read replay data: ", err)
		}

		fmt.Println("Exporting replay to", outputFilename)
		fileio.ExportCiv5ReplayFile(replayData, outputFilename)
	} else if strings.ToLower(inputFileExtension) == ".civ5save" {
		fmt.Println("Reading civ5save file")
		fileio.ReadCiv5SaveFile(inputFilename, outputFilename)
	} else {
		log.Fatal("Unable to export file", inputFilename, "to json")
	}
}

func loadMapDataFromFile(filename string) *fileio.Civ5MapData {
	mapFileExtension := filepath.Ext(filename)
	if strings.ToLower(mapFileExtension) == ".json" {
		fmt.Println("Importing map file from json")
		mapData := fileio.ImportCiv5MapFileFromJson(filename)
		graphics.OverrideColorMap(mapData.CivColorOverrides)
		return mapData
	} else if strings.ToLower(mapFileExtension) == ".civ5map" {
		fmt.Println("Reading map from .civ5map file")
		mapData, err := fileio.ReadCiv5MapFile(filename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}
		return mapData
	} else {
		log.Fatal(fmt.Sprintf("Input map file has invalid file extension. Filename: %s, extension: %s", filename, mapFileExtension))
	}
	return nil
}

func loadReplayDataFromFile(replayFilename string) *fileio.Civ5ReplayData {
	replayFileExtension := filepath.Ext(replayFilename)
	if strings.ToLower(replayFileExtension) == ".civ5replay" {
		fmt.Println("Reading replay from .civ5replay file")
		replayData, err := fileio.ReadCiv5ReplayFile(replayFilename)
		if err != nil {
			log.Fatal("Failed to read replay data: ", err)
		}

		return replayData
	} else if strings.ToLower(replayFileExtension) == ".json" {
		fmt.Println("Importing replay data from json")
		replayData := fileio.ImportCiv5ReplayFileFromJson(replayFilename)
		return replayData
	} else {
		log.Fatal("Replay file has invalid file extension")
	}
	return nil
}

func main() {
	inputPtr := flag.String("input", "", "Input filename")
	outputPtr := flag.String("output", "output.png", "Output filename")
	replayFilePtr := flag.String("replay", "", "Replay filename for replay mode")
	modePtr := flag.String("mode", "physical", "Drawing mode")

	flag.Parse()

	inputFilename := *inputPtr
	outputFilename := *outputPtr
	mode := *modePtr
	fmt.Println("Input filename: ", inputFilename)
	fmt.Println("Output filename: ", outputFilename)
	fmt.Println("Mode: ", mode)

	if mode == "exportjson" {
		exportFileToJson(inputFilename, outputFilename)
		return
	}

	mapData := loadMapDataFromFile(inputFilename)

	if mode == "physical" {
		graphics.SaveImage(outputFilename, graphics.DrawPhysicalMap(mapData))
		return
	} else if mode == "political" {
		graphics.SaveImage(outputFilename, graphics.DrawPoliticalMap(mapData))
		return
	} else if mode == "replay" {
		replayFilename := *replayFilePtr
		replayData := loadReplayDataFromFile(replayFilename)
		graphics.DrawReplay(mapData, replayData, outputFilename)
		return
	} else {
		log.Fatal("Invalid drawing mode: " + mode + ". Mode must be in this list [phyiscal, political, replay].")
	}
}
