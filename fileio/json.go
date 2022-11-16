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
