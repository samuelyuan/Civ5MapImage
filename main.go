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

// DrawingMode represents the available drawing modes
type DrawingMode string

const (
	ModePhysical   DrawingMode = "physical"
	ModePolitical  DrawingMode = "political"
	ModeReplay     DrawingMode = "replay"
	ModeExportJSON DrawingMode = "exportjson"
)

func loadMapDataFromFile(filename string) *fileio.Civ5MapData {
	mapFileExtension := filepath.Ext(filename)

	switch strings.ToLower(mapFileExtension) {
	case string(fileio.FileTypeJSON):
		fmt.Println("Importing map file from json")
		mapData, err := fileio.ImportCiv5MapFileFromJson(filename)
		if err != nil {
			log.Fatal("Failed to import map from json: ", err)
		}
		graphics.OverrideColorMap(mapData.CivColorOverrides)
		return mapData
	case string(fileio.FileTypeCiv5Map):
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
		fileio.ExportFileToJson(inputFilename, outputFilename)
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
		replayData := fileio.LoadReplayDataFromFile(replayFilename)
		graphics.DrawReplay(mapData, replayData, outputFilename)
		return
	default:
		log.Fatal("Invalid drawing mode: " + mode + ". Mode must be in this list [physical, political, replay, exportjson].")
	}
}
