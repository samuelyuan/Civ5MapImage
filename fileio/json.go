package fileio

import (
	"encoding/json"
	"io/ioutil"
	"log"
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

func ImportCiv5MapFileFromJson(inputFilename string) *Civ5MapData {
	jsonFile, err := os.Open(inputFilename)
	if err != nil {
		log.Fatal("Failed to open json file", err)
	}
	defer jsonFile.Close()

	jsonContents, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	var civ5MapJson *Civ5MapJson
	json.Unmarshal(jsonContents, &civ5MapJson)

	if civ5MapJson == nil {
		log.Fatal("The json data in " + inputFilename + " is missing or incorrect")
	}

	return civ5MapJson.MapData
}

func ExportCiv5MapFile(mapData *Civ5MapData, outputFilename string) {
	civ5MapJson := &Civ5MapJson{
		GameName:   "Civilization 5",
		FileFormat: ".Civ5Map",
		MapData:    mapData,
	}

	file, err := json.MarshalIndent(civ5MapJson, "", " ")
	if err != nil {
		log.Fatal("Failed to marshal map data: ", err)
	}

	err = ioutil.WriteFile(outputFilename, file, 0644)
	if err != nil {
		log.Fatal("Error writing to ", outputFilename)
	}
}

func ImportCiv5ReplayFileFromJson(inputFilename string) *Civ5ReplayData {
	jsonFile, err := os.Open(inputFilename)
	if err != nil {
		log.Fatal("Failed to open json file", err)
	}
	defer jsonFile.Close()

	jsonContents, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	var civ5ReplayJson *Civ5ReplayJson
	json.Unmarshal(jsonContents, &civ5ReplayJson)

	if civ5ReplayJson == nil {
		log.Fatal("The json data in " + inputFilename + " is missing or incorrect")
	}

	return civ5ReplayJson.ReplayData
}

func ExportCiv5ReplayFile(replayData *Civ5ReplayData, outputFilename string) {
	civ5ReplayJson := &Civ5ReplayJson{
		GameName:   "Civilization 5",
		FileFormat: ".Civ5Replay",
		ReplayData: replayData,
	}

	file, err := json.MarshalIndent(civ5ReplayJson, "", " ")
	if err != nil {
		log.Fatal("Failed to marshal replay data: ", err)
	}

	err = ioutil.WriteFile(outputFilename, file, 0644)
	if err != nil {
		log.Fatal("Error writing to ", outputFilename)
	}
}

func ExportCiv5SaveFile(saveData *Civ5SaveData, outputFilename string) {
	civ5SaveJson := &Civ5SaveJson{
		GameName:   "Civilization 5",
		FileFormat: ".Civ5Save",
		ReplayData: saveData,
	}

	file, err := json.MarshalIndent(civ5SaveJson, "", " ")
	if err != nil {
		log.Fatal("Failed to marshal save data: ", err)
	}

	err = ioutil.WriteFile(outputFilename, file, 0644)
	if err != nil {
		log.Fatal("Error writing to ", outputFilename)
	}
}
