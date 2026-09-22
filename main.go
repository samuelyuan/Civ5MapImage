package main

import (
	"flag"
	"fmt"
	"image"
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

// cliArgs holds the resolved command-line configuration, after applying parseArgs' defaults.
type cliArgs struct {
	inputFilename  string
	outputFilename string
	replayFilename string
	mode           string
	maxTurns       int
}

// parseArgs parses the command line and resolves defaults: -map (preferred) or -input (alias);
// -replay alone implies -mode=replay; replay mode defaults -output to "output.gif".
func parseArgs() cliArgs {
	inputPtr := flag.String("input", "", "Map filename (.civ5map or .json) - alias for -map")
	mapPtr := flag.String("map", "", "Map filename (.civ5map or .json)")
	outputPtr := flag.String("output", "output.png", "Output filename")
	replayFilePtr := flag.String("replay", "", "Replay filename (.civ5replay, .civ5save, or .json). Passing -replay without -mode generates a replay gif directly.")
	modePtr := flag.String("mode", "physical", "Drawing mode")
	maxTurnsPtr := flag.Int("maxturns", 0, "Replay mode only: render at most this many turns (0 = all). Useful for a quick preview instead of a full, possibly long, render.")

	flag.Parse()

	// Tracks explicit flags, to tell a default value apart from a real choice below.
	modeSet, outputSet := false, false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "mode":
			modeSet = true
		case "output":
			outputSet = true
		}
	})

	args := cliArgs{
		inputFilename:  *inputPtr,
		outputFilename: *outputPtr,
		replayFilename: *replayFilePtr,
		mode:           *modePtr,
		maxTurns:       *maxTurnsPtr,
	}
	if *mapPtr != "" {
		args.inputFilename = *mapPtr
	}
	if !modeSet && args.replayFilename != "" {
		args.mode = string(ModeReplay)
	}
	if args.mode == string(ModeReplay) && !outputSet {
		args.outputFilename = "output.gif"
	}
	return args
}

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

// renderMap runs a physical/political render+save, differing only in which MapRenderer method
// produces the image (passed as a method expression, e.g. (*graphics.MapRenderer).DrawPhysicalMap).
func renderMap(mapData *fileio.Civ5MapData, outputFilename string, draw func(*graphics.MapRenderer, graphics.Canvas, *fileio.Civ5MapData) image.Image) {
	config := graphics.DefaultDrawingConfig()
	renderer := graphics.NewMapRenderer(config)
	canvas := graphics.NewMapCanvas()
	draw(renderer, canvas, mapData)
	if err := renderer.SaveImage(canvas, outputFilename); err != nil {
		log.Fatal("Failed to save image: ", err)
	}
}

// runReplayMode validates the map/replay pair and, if compatible, draws the replay gif.
func runReplayMode(args cliArgs) {
	if args.replayFilename == "" {
		log.Fatal("replay mode requires -replay <file.civ5replay>")
	}
	if err := fileio.ValidateFileExtension(args.outputFilename, ".gif"); err != nil {
		log.Fatalf("Invalid replay output filename: %v. Replays are always saved as an animated GIF, so -output must end in .gif.", err)
	}

	mapData := loadMapDataFromFile(args.inputFilename)
	replayData := fileio.LoadReplayDataFromFile(args.replayFilename)

	validateMapReplayCompatibility(args.inputFilename, mapData, replayData)

	if err := fileio.PrepareReplay(mapData, replayData); err != nil {
		log.Fatal("Failed to prepare replay: ", err)
	}
	if err := graphics.DrawReplay(mapData, replayData, args.outputFilename, args.maxTurns); err != nil {
		log.Fatal("Failed to draw replay: ", err)
	}
}

// validateMapReplayCompatibility prints the map/replay compatibility report. Only a geography
// mismatch (compatErr) aborts; a filename mismatch is always just a warning.
func validateMapReplayCompatibility(inputFilename string, mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData) {
	fmt.Println("\n=== Validating map and replay compatibility ===")
	compatResults, compatErr := fileio.ValidateMapReplayCompatible(mapData, replayData)
	if len(compatResults) > 0 {
		fileio.PrintValidationResults(compatResults)
	}
	if filenameResult := fileio.ValidateMapFilenameMatch(inputFilename, replayData); filenameResult != nil {
		fmt.Println(filenameResult)
		if filenameResult.Passed != filenameResult.Total {
			fmt.Printf("WARNING: %s\n", filenameResult.Notes)
		}
	}
	fmt.Println()
	if compatErr != nil {
		log.Fatalf("Map and replay are not compatible: %v", compatErr)
	}
}

func main() {
	args := parseArgs()

	fmt.Println("Input filename: ", args.inputFilename)
	fmt.Println("Output filename: ", args.outputFilename)
	fmt.Println("Mode: ", args.mode)

	switch args.mode {
	case string(ModeExportJSON):
		fileio.ExportFileToJson(args.inputFilename, args.outputFilename)
	case string(ModePhysical):
		renderMap(loadMapDataFromFile(args.inputFilename), args.outputFilename, (*graphics.MapRenderer).DrawPhysicalMap)
	case string(ModePolitical):
		renderMap(loadMapDataFromFile(args.inputFilename), args.outputFilename, (*graphics.MapRenderer).DrawPoliticalMap)
	case string(ModeReplay):
		runReplayMode(args)
	default:
		log.Fatal("Invalid drawing mode: " + args.mode + ". Mode must be in this list [physical, political, replay, exportjson].")
	}
}
