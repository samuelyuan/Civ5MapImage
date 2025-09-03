package fileio

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// FileType represents the type of Civilization 5 file
type FileType string

const (
	FileTypeCiv5Map    FileType = ".civ5map"
	FileTypeCiv5Replay FileType = ".civ5replay"
	FileTypeCiv5Save   FileType = ".civ5save"
	FileTypeJSON       FileType = ".json"
)

// ExportFileToJson exports a Civilization 5 file to JSON format
func ExportFileToJson(inputFilename string, outputFilename string) {
	inputFileExtension := filepath.Ext(inputFilename)

	switch strings.ToLower(inputFileExtension) {
	case string(FileTypeCiv5Map):
		fmt.Println("Reading civ5map file")
		mapData, err := ReadCiv5MapFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}

		fmt.Println("Exporting map to", outputFilename)
		if err := ExportCiv5MapFile(mapData, outputFilename); err != nil {
			log.Fatal("Failed to export map: ", err)
		}
	case string(FileTypeCiv5Replay):
		fmt.Println("Importing civ5replay data")
		replayData, err := ReadCiv5ReplayFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read replay data: ", err)
		}

		fmt.Println("Exporting replay to", outputFilename)
		if err := ExportCiv5ReplayFile(replayData, outputFilename); err != nil {
			log.Fatal("Failed to export replay: ", err)
		}
	case string(FileTypeCiv5Save):
		fmt.Println("Reading civ5save file")
		saveData, err := ReadCiv5SaveFile(inputFilename, outputFilename+".decomp")
		if err != nil {
			log.Fatal("Failed to read save data: ", err)
		}

		fmt.Println("Exporting save to", outputFilename)
		if err := ExportCiv5SaveFile(saveData, outputFilename); err != nil {
			log.Fatal("Failed to export save: ", err)
		}
	default:
		log.Fatal("Unable to export file", inputFilename, "to json")
	}
}

// LoadReplayDataFromFile loads replay data from a file (either .civ5replay or .json)
func LoadReplayDataFromFile(replayFilename string) *Civ5ReplayData {
	replayFileExtension := filepath.Ext(replayFilename)

	switch strings.ToLower(replayFileExtension) {
	case string(FileTypeCiv5Replay):
		fmt.Println("Reading replay from .civ5replay file")
		replayData, err := ReadCiv5ReplayFile(replayFilename)
		if err != nil {
			log.Fatal("Failed to read replay data: ", err)
		}
		return replayData
	case string(FileTypeJSON):
		fmt.Println("Importing replay data from json")
		replayData, err := ImportCiv5ReplayFileFromJson(replayFilename)
		if err != nil {
			log.Fatal("Failed to import replay from json: ", err)
		}
		return replayData
	default:
		log.Fatal("Replay file has invalid file extension")
	}
	return nil
}
