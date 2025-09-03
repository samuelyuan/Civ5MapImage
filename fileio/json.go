package fileio

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Civ5MapJson struct {
	GameName   string
	FileFormat string
	MapData    *Civ5MapData
}

type Civ5ReplayJson struct {
	GameName   string
	FileFormat string
	ReplayData *Civ5ReplayData
}

type Civ5SaveJson struct {
	GameName   string
	FileFormat string
	ReplayData *Civ5SaveData
}

func ImportCiv5MapFileFromJson(inputFilename string) (*Civ5MapData, error) {
	jsonFile, err := os.Open(inputFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to open json file %q: %w", inputFilename, err)
	}
	defer jsonFile.Close()

	jsonContents, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read json file %q: %w", inputFilename, err)
	}

	var civ5MapJson *Civ5MapJson
	if err := json.Unmarshal(jsonContents, &civ5MapJson); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json from %q: %w", inputFilename, err)
	}

	if civ5MapJson == nil {
		return nil, fmt.Errorf("json data in %q is missing or incorrect", inputFilename)
	}

	return civ5MapJson.MapData, nil
}

func ExportCiv5MapFile(mapData *Civ5MapData, outputFilename string) error {
	civ5MapJson := &Civ5MapJson{
		GameName:   "Civilization 5",
		FileFormat: ".Civ5Map",
		MapData:    mapData,
	}

	file, err := json.MarshalIndent(civ5MapJson, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal map data: %w", err)
	}

	err = os.WriteFile(outputFilename, file, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to %q: %w", outputFilename, err)
	}

	return nil
}

func ImportCiv5ReplayFileFromJson(inputFilename string) (*Civ5ReplayData, error) {
	jsonFile, err := os.Open(inputFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to open json file %q: %w", inputFilename, err)
	}
	defer jsonFile.Close()

	jsonContents, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read json file %q: %w", inputFilename, err)
	}

	var civ5ReplayJson *Civ5ReplayJson
	if err := json.Unmarshal(jsonContents, &civ5ReplayJson); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json from %q: %w", inputFilename, err)
	}

	if civ5ReplayJson == nil {
		return nil, fmt.Errorf("json data in %q is missing or incorrect", inputFilename)
	}

	return civ5ReplayJson.ReplayData, nil
}

func ExportCiv5ReplayFile(replayData *Civ5ReplayData, outputFilename string) error {
	civ5ReplayJson := &Civ5ReplayJson{
		GameName:   "Civilization 5",
		FileFormat: ".Civ5Replay",
		ReplayData: replayData,
	}

	file, err := json.MarshalIndent(civ5ReplayJson, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal replay data: %w", err)
	}

	err = os.WriteFile(outputFilename, file, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to %q: %w", outputFilename, err)
	}

	return nil
}

func ExportCiv5SaveFile(saveData *Civ5SaveData, outputFilename string) error {
	civ5SaveJson := &Civ5SaveJson{
		GameName:   "Civilization 5",
		FileFormat: ".Civ5Save",
		ReplayData: saveData,
	}

	file, err := json.MarshalIndent(civ5SaveJson, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal save data: %w", err)
	}

	err = os.WriteFile(outputFilename, file, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to %q: %w", outputFilename, err)
	}

	return nil
}
