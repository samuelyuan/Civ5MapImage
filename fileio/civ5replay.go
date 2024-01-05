package fileio

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type Civ5ReplayFileConfigEntry struct {
	VariableType string
	VariableName string
}

type Civ5ReplayCiv struct {
	UnknownVariables [4]int
	Leader           string
	LongName         string
	Name             string
	Demonym          string
}

type Civ5ReplayEventTile struct {
	X int
	Y int
}

type Civ5ReplayEvent struct {
	Turn   int
	TypeId int
	Tiles  []Civ5ReplayEventTile
	CivId  int
	Text   string
}

type Civ5ReplayCivDataset struct {
	CivIndex      int
	DatasetValues map[string][]Civ5ReplayDataEntry
}

type Civ5ReplayDataEntry struct {
	Turn  int
	Value int
}

type Civ5ReplayData struct {
	PlayerCiv       string
	IsReplayFile    bool
	AllCivs         []Civ5ReplayCiv
	AllReplayEvents []Civ5ReplayEvent
	DatasetNames    []string
	DatasetValues   []Civ5ReplayCivDataset
}

func readFileConfig(reader *io.SectionReader, fileConfigEntries []Civ5ReplayFileConfigEntry) []string {
	fieldValues := make([]string, 0)

	for i := 0; i < len(fileConfigEntries); i++ {
		fileConfigEntry := fileConfigEntries[i]
		if fileConfigEntry.VariableType == "varstring" {
			value := readVarString(reader, "varstring_"+fileConfigEntry.VariableName)
			fieldValues = append(fieldValues, fmt.Sprintf("%v(str):%v", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "float32" {
			value := float32(0)
			if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
				log.Fatal("Failed to load float32: ", err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(f32):%f", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "uint32" {
			value := unsafeReadUint32(reader)
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u32):%d", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "int32" {
			signedIntValue := int32(0)
			if err := binary.Read(reader, binary.LittleEndian, &signedIntValue); err != nil {
				log.Fatal("Failed to load int32: ", err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(i32):%d", fileConfigEntry.VariableName, signedIntValue))
		} else if fileConfigEntry.VariableType == "uint16" {
			value := unsafeReadUint16(reader)
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u16):%d", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "uint8" {
			unsignedIntValue := uint8(0)
			if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
				log.Fatal("Failed to load uint8: ", err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u8):%d", fileConfigEntry.VariableName, unsignedIntValue))
		} else if strings.Contains(fileConfigEntry.VariableType, "bytearray") {
			byteArrayLength, err := strconv.Atoi(fileConfigEntry.VariableType[len("bytearray:"):])
			if err != nil {
				log.Fatal("Invalid byte array type in file config:", err)
			}

			byteBlock := make([]byte, byteArrayLength)
			if err := binary.Read(reader, binary.LittleEndian, &byteBlock); err != nil {
				log.Fatal("Invalid byte array data", err)
			}

			fieldValues = append(fieldValues, fmt.Sprintf("%v(bytearray):%v", fileConfigEntry.VariableName, byteBlock))
		} else {
			fmt.Println("Unknown variable type:", fileConfigEntry.VariableType)
		}
	}

	return fieldValues
}

func readVarString(reader *io.SectionReader, varName string) string {
	variableLength := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &variableLength); err != nil {
		log.Fatal("Failed to load variable length: ", err)
	}

	stringValue := make([]byte, variableLength)
	if err := binary.Read(reader, binary.LittleEndian, &stringValue); err != nil {
		log.Fatal(fmt.Sprintf("Failed to load string value. Variable length: %v, name: %s. Error:", variableLength, varName), err)
	}

	return string(stringValue[:])
}

func readArray(reader *io.SectionReader, arrayName string, fileConfigEntries []Civ5ReplayFileConfigEntry) {
	arrayLength := unsafeReadUint32(reader)
	if arrayLength > 100000 {
		log.Fatal("Array length may be too long:", arrayLength)
	}
	for i := 0; i < int(arrayLength); i++ {
		readFileConfig(reader, fileConfigEntries)
	}
}

func unsafeReadUint32(reader *io.SectionReader) uint32 {
	unsignedIntValue := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load uint32: ", err)
	}
	return unsignedIntValue
}

func unsafeReadUint16(reader *io.SectionReader) uint16 {
	unsignedIntValue := uint16(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load uint16: ", err)
	}
	return unsignedIntValue
}

func readCivs(reader *io.SectionReader) []Civ5ReplayCiv {
	civsLength := unsafeReadUint32(reader)
	allCivs := make([]Civ5ReplayCiv, 0)

	for i := 0; i < int(civsLength); i++ {
		unknownVariable1 := unsafeReadUint32(reader)
		unknownVariable2 := unsafeReadUint32(reader)
		unknownVariable3 := unsafeReadUint32(reader)
		unknownVariable4 := unsafeReadUint32(reader)
		leader := readVarString(reader, "leader")
		longName := readVarString(reader, "longName")
		name := readVarString(reader, "name")
		demonym := readVarString(reader, "demonym")

		civData := Civ5ReplayCiv{
			UnknownVariables: [4]int{int(unknownVariable1), int(unknownVariable2), int(unknownVariable3), int(unknownVariable4)},
			Leader:           leader,
			LongName:         longName,
			Name:             name,
			Demonym:          demonym,
		}
		allCivs = append(allCivs, civData)
	}

	return allCivs
}

func readEvents(reader *io.SectionReader) []Civ5ReplayEvent {
	eventsLength := unsafeReadUint32(reader)
	allReplayEvents := make([]Civ5ReplayEvent, eventsLength)

	for i := 0; i < int(eventsLength); i++ {
		turn := unsafeReadUint32(reader)
		typeId := unsafeReadUint32(reader)

		numTiles := unsafeReadUint32(reader)
		tileData := make([]Civ5ReplayEventTile, numTiles)
		for i := 0; i < int(numTiles); i++ {
			tileX := unsafeReadUint16(reader)
			tileY := unsafeReadUint16(reader)

			tileData[i] = Civ5ReplayEventTile{
				X: int(tileX),
				Y: int(tileY),
			}
		}

		civId := unsafeReadUint32(reader)
		eventText := readVarString(reader, "eventText")

		allReplayEvents[i] = Civ5ReplayEvent{
			Turn:   int(turn),
			TypeId: int(typeId),
			Tiles:  tileData,
			CivId:  int(civId),
			Text:   eventText,
		}
	}

	return allReplayEvents
}

func GroupEventsByTurn(replayEvents []Civ5ReplayEvent) map[int][]Civ5ReplayEvent {
	replayTurns := make(map[int][]Civ5ReplayEvent)

	for i := 0; i < len(replayEvents); i++ {
		turn := replayEvents[i].Turn
		_, ok := replayTurns[turn]
		if !ok {
			replayTurns[turn] = make([]Civ5ReplayEvent, 0)
		}
		replayTurns[turn] = append(replayTurns[turn], replayEvents[i])
	}
	return replayTurns
}

func readDatasetNames(streamReader *io.SectionReader) []string {
	datasetLength := unsafeReadUint32(streamReader)
	datasetNames := make([]string, int(datasetLength))
	for i := 0; i < int(datasetLength); i++ {
		datasetNames[i] = readVarString(streamReader, "datasetNames")
	}
	return datasetNames
}

func readDatasetValues(streamReader *io.SectionReader) [][][]Civ5ReplayDataEntry {
	datasetValuesArray1Length := unsafeReadUint32(streamReader)
	datasetByCiv := make([][][]Civ5ReplayDataEntry, int(datasetValuesArray1Length))

	for i := 0; i < int(datasetValuesArray1Length); i++ {
		datasetValuesArray2Length := unsafeReadUint32(streamReader)
		datasetByCategory := make([][]Civ5ReplayDataEntry, int(datasetValuesArray2Length))

		for j := 0; j < int(datasetValuesArray2Length); j++ {
			numDatasetValues := unsafeReadUint32(streamReader)

			datasetArray := make([]Civ5ReplayDataEntry, numDatasetValues)
			for k := 0; k < int(numDatasetValues); k++ {
				turn := unsafeReadUint32(streamReader)
				value := unsafeReadUint32(streamReader)
				datasetArray[k] = Civ5ReplayDataEntry{
					Turn:  int(turn),
					Value: int(value),
				}
			}

			datasetByCategory[j] = datasetArray
		}
		datasetByCiv[i] = datasetByCategory
	}
	return datasetByCiv
}

func buildCivDatasetValues(streamReader *io.SectionReader, datasetNames []string) []Civ5ReplayCivDataset {
	datasetValues := readDatasetValues(streamReader)

	allCivDatasetValues := make([]Civ5ReplayCivDataset, len(datasetValues))
	for civIndex := 0; civIndex < len(datasetValues); civIndex++ {
		dataMap := make(map[string][]Civ5ReplayDataEntry, 0)
		for datasetNameIndex := 0; datasetNameIndex < len(datasetNames); datasetNameIndex++ {
			datasetName := datasetNames[datasetNameIndex]
			dataMap[datasetName] = datasetValues[civIndex][datasetNameIndex]
		}

		allCivDatasetValues[civIndex] = Civ5ReplayCivDataset{
			CivIndex:      civIndex,
			DatasetValues: dataMap,
		}
	}
	return allCivDatasetValues
}

func ReadCiv5ReplayFile(filename string) (*Civ5ReplayData, error) {
	inputFile, err := os.Open(filename)
	defer inputFile.Close()
	if err != nil {
		log.Fatal("Failed to load replay file: ", err)
		return nil, err
	}

	fi, err := inputFile.Stat()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	fileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), fileLength)
	fmt.Println("Loading Civ5Replay...")

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:4",
			VariableName: "gameName",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownBlock1",
		},
		{
			VariableType: "varstring",
			VariableName: "gameVersion",
		},
		{
			VariableType: "varstring",
			VariableName: "gameBuild",
		},
		{
			VariableType: "uint32",
			VariableName: "currentTurnNumber",
		},
		{
			VariableType: "bytearray:1",
			VariableName: "unknownBlock2",
		},
	})

	playerCiv := readVarString(streamReader, "playerCiv")

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "difficulty",
		},
		{
			VariableType: "varstring",
			VariableName: "eraStart",
		},
		{
			VariableType: "varstring",
			VariableName: "eraEnd",
		},
		{
			VariableType: "varstring",
			VariableName: "gameSpeed",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSize",
		},
		{
			VariableType: "varstring",
			VariableName: "mapFilename",
		},
	})

	readArray(streamReader, "dlc", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:16",
			VariableName: "dlcId",
		},
		{
			VariableType: "bytearray:4",
			VariableName: "dlcEnabled",
		},
		{
			VariableType: "varstring",
			VariableName: "dlcName",
		},
	})

	readArray(streamReader, "mods", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "modId",
		},
		{
			VariableType: "bytearray:4",
			VariableName: "modVersion",
		},
		{
			VariableType: "varstring",
			VariableName: "modName",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "civName",
		},
		{
			VariableType: "varstring",
			VariableName: "leaderName",
		},
		{
			VariableType: "varstring",
			VariableName: "playerColor",
		},
		{
			VariableType: "bytearray:8",
			VariableName: "unknownBlock5",
		},
		{
			VariableType: "varstring",
			VariableName: "mapFilename2",
		},
	})

	// This block doesn't seem to have a pattern
	unknownVersion := unsafeReadUint32(streamReader)
	if unknownVersion == 2 {
		for i := 0; i < 7; i++ {
			_ = unsafeReadUint32(streamReader)
		}
	} else {
		for i := 0; i < 9; i++ {
			_ = unsafeReadUint32(streamReader)
		}
	}

	unknownCount := unsafeReadUint32(streamReader)
	for i := 0; i < int(unknownCount)+1; i++ {
		_ = unsafeReadUint32(streamReader)
	}

	// Read one byte of padding
	unknownBlock2 := [1]byte{}
	if err := binary.Read(streamReader, binary.LittleEndian, &unknownBlock2); err != nil {
		log.Fatal("Failed to read block: ", err)
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "startTurn",
		},
		{
			VariableType: "int32", // startYear can be negative, e.g. 4000 BC
			VariableName: "startYear",
		},
		{
			VariableType: "uint32",
			VariableName: "endTurn",
		},
		{
			VariableType: "varstring",
			VariableName: "endYear",
		},
		{
			VariableType: "uint32",
			VariableName: "zeroStartYear",
		},
		{
			VariableType: "uint32",
			VariableName: "zeroEndYear",
		},
	})

	allCivs := readCivs(streamReader)

	datasetNames := readDatasetNames(streamReader)
	datasetValues := buildCivDatasetValues(streamReader, datasetNames)

	// Read unknown value
	_ = unsafeReadUint32(streamReader)

	allReplayEvents := readEvents(streamReader)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "mapWidth",
		},
		{
			VariableType: "uint32",
			VariableName: "mapHeight",
		},
	})

	readArray(streamReader, "tiles", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownVariable1",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownVariable2",
		},
		{
			VariableType: "uint8",
			VariableName: "elevationId",
		},
		{
			VariableType: "uint8",
			VariableName: "typeId",
		},
		{
			VariableType: "uint8",
			VariableName: "featureId",
		},
		{
			VariableType: "uint8",
			VariableName: "unknownVariable3",
		},
	})

	replayData := Civ5ReplayData{
		PlayerCiv:       playerCiv,
		IsReplayFile:    true,
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
		DatasetNames:    datasetNames,
		DatasetValues:   datasetValues,
	}

	return &replayData, nil
}
