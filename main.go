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

	inputFileExtension := filepath.Ext(inputFilename)
	outputFileExtension := filepath.Ext(outputFilename)

	var mapData *fileio.Civ5MapData
	var err error
	if strings.ToLower(inputFileExtension) == ".json" {
		fmt.Println("Importing map file from json")
		mapData = fileio.ImportCiv5MapFileFromJson(inputFilename)
		graphics.OverrideColorMap(mapData.CivColorOverrides)
	} else if strings.ToLower(inputFileExtension) == ".civ5map" {
		fmt.Println("Reading civ5map file")
		mapData, err = fileio.ReadCiv5MapFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}
	} else {
		log.Fatal("Input file has invalid file extension")
	}

	if outputFileExtension == ".json" {
		fmt.Println("Exporting map to", outputFilename)
		fileio.ExportCiv5MapFile(mapData, outputFilename)
		return
	}

	if mode == "physical" {
		graphics.SaveImage(outputFilename, graphics.DrawPhysicalMap(mapData))
	} else if mode == "political" {
		graphics.SaveImage(outputFilename, graphics.DrawPoliticalMap(mapData))
	} else if mode == "replay" {
		replayData, err := fileio.ReadCiv5ReplayFile(*replayFilePtr)
		if err != nil {
			log.Fatal("Failed to read replay data: ", err)
		}

		graphics.DrawReplay(mapData, replayData, outputFilename)
	} else {
		log.Fatal("Invalid drawing mode: " + mode + ". Mode must be in this list [phyiscal, political, replay].")
	}
}
