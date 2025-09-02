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

// FileType represents the type of Civilization 5 file
type FileType string

const (
	FileTypeCiv5Map    FileType = ".civ5map"
	FileTypeCiv5Replay FileType = ".civ5replay"
	FileTypeCiv5Save   FileType = ".civ5save"
	FileTypeJSON       FileType = ".json"
)

// DrawingMode represents the available drawing modes
type DrawingMode string

const (
	ModePhysical   DrawingMode = "physical"
	ModePolitical  DrawingMode = "political"
	ModeReplay     DrawingMode = "replay"
	ModeExportJSON DrawingMode = "exportjson"
)

func exportFileToJson(inputFilename string, outputFilename string) {
	inputFileExtension := filepath.Ext(inputFilename)

	switch strings.ToLower(inputFileExtension) {
	case string(FileTypeCiv5Map):
		fmt.Println("Reading civ5map file")
		mapData, err := fileio.ReadCiv5MapFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}

		fmt.Println("Exporting map to", outputFilename)
		fileio.ExportCiv5MapFile(mapData, outputFilename)
	case string(FileTypeCiv5Replay):
		fmt.Println("Importing civ5replay data")
		replayData, err := fileio.ReadCiv5ReplayFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read replay data: ", err)
		}

		fmt.Println("Exporting replay to", outputFilename)
		fileio.ExportCiv5ReplayFile(replayData, outputFilename)
	case string(FileTypeCiv5Save):
		fmt.Println("Reading civ5save file")
		saveData, err := fileio.ReadCiv5SaveFile(inputFilename, outputFilename+".decomp")
		if err != nil {
			log.Fatal("Failed to read save data: ", err)
		}

		fmt.Println("Exporting save to", outputFilename)
		fileio.ExportCiv5SaveFile(saveData, outputFilename)
	default:
		log.Fatal("Unable to export file", inputFilename, "to json")
	}
}

func loadMapDataFromFile(filename string) *fileio.Civ5MapData {
	mapFileExtension := filepath.Ext(filename)

	switch strings.ToLower(mapFileExtension) {
	case string(FileTypeJSON):
		fmt.Println("Importing map file from json")
		mapData := fileio.ImportCiv5MapFileFromJson(filename)
		graphics.OverrideColorMap(mapData.CivColorOverrides)
		return mapData
	case string(FileTypeCiv5Map):
		fmt.Println("Reading map from .civ5map file")
		mapData, err := fileio.ReadCiv5MapFile(filename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}
		return mapData
	default:
		log.Fatalf("Input map file has invalid file extension. Filename: %s, extension: %s", filename, mapFileExtension)
	}
	return nil
}

func loadReplayDataFromFile(replayFilename string) *fileio.Civ5ReplayData {
	replayFileExtension := filepath.Ext(replayFilename)

	switch strings.ToLower(replayFileExtension) {
	case string(FileTypeCiv5Replay):
		fmt.Println("Reading replay from .civ5replay file")
		replayData, err := fileio.ReadCiv5ReplayFile(replayFilename)
		if err != nil {
			log.Fatal("Failed to read replay data: ", err)
		}
		return replayData
	case string(FileTypeJSON):
		fmt.Println("Importing replay data from json")
		replayData := fileio.ImportCiv5ReplayFileFromJson(replayFilename)
		return replayData
	default:
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

	if mode == string(ModeExportJSON) {
		exportFileToJson(inputFilename, outputFilename)
		return
	}

	mapData := loadMapDataFromFile(inputFilename)

	switch mode {
	case string(ModePhysical):
		graphics.SaveImage(outputFilename, graphics.DrawPhysicalMap(mapData))
		return
	case string(ModePolitical):
		graphics.SaveImage(outputFilename, graphics.DrawPoliticalMap(mapData))
		return
	case string(ModeReplay):
		replayFilename := *replayFilePtr
		replayData := loadReplayDataFromFile(replayFilename)
		graphics.DrawReplay(mapData, replayData, outputFilename)
		return
	default:
		log.Fatal("Invalid drawing mode: " + mode + ". Mode must be in this list [physical, political, replay, exportjson].")
	}
}
