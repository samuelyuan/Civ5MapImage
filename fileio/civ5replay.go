package fileio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
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

func readCivs(reader *io.SectionReader) []Civ5ReplayCiv {
	civsLength := unsafeReadUint32(reader)
	allCivs := make([]Civ5ReplayCiv, 0)

	for i := 0; i < int(civsLength); i++ {
		unknownVariable1 := unsafeReadUint32(reader)
		unknownVariable2 := unsafeReadUint32(reader)
		unknownVariable3 := unsafeReadUint32(reader)
		unknownVariable4 := unsafeReadUint32(reader)
		leader, err := readVarString(reader, "leader")
		if err != nil {
			panic(fmt.Sprintf("failed to read leader: %v", err))
		}
		longName, err := readVarString(reader, "longName")
		if err != nil {
			panic(fmt.Sprintf("failed to read longName: %v", err))
		}
		name, err := readVarString(reader, "name")
		if err != nil {
			panic(fmt.Sprintf("failed to read name: %v", err))
		}
		demonym, err := readVarString(reader, "demonym")
		if err != nil {
			panic(fmt.Sprintf("failed to read demonym: %v", err))
		}

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

		civId := int32(unsafeReadUint32(reader))
		eventText, err := readVarString(reader, "eventText")
		if err != nil {
			panic(fmt.Sprintf("failed to read eventText: %v", err))
		}

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
		name, err := readVarString(streamReader, "datasetNames")
		if err != nil {
			panic(fmt.Sprintf("failed to read dataset name %d: %v", i, err))
		}
		datasetNames[i] = name
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
	if err != nil {
		return nil, fmt.Errorf("failed to load replay file %q: %w", filename, err)
	}
	defer inputFile.Close()

	fi, err := inputFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %q: %w", filename, err)
	}
	fileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), fileLength)
	fmt.Println("Loading Civ5Replay...")

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
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
	if err != nil {
		return nil, fmt.Errorf("failed to read initial file config: %w", err)
	}

	playerCiv, err := readVarString(streamReader, "playerCiv")
	if err != nil {
		return nil, fmt.Errorf("failed to read player civ: %w", err)
	}

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
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

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
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
	unknownArr := make([]int, 0)
	fmt.Println("Unknown version:", unknownVersion)
	unknownArr = append(unknownArr, int(unknownVersion))
	for i := 0; i < 4; i++ {
		value := unsafeReadUint32(streamReader)
		unknownArr = append(unknownArr, int(value))
	}

	unknownCount := unsafeReadUint32(streamReader)
	unknownArr = append(unknownArr, int(unknownCount))

	for i := 0; i < int(unknownCount); i++ {
		value := unsafeReadUint32(streamReader)
		unknownArr = append(unknownArr, int(value))
	}

	unknownCount2 := unsafeReadUint32(streamReader)
	unknownArr = append(unknownArr, int(unknownCount2))

	for i := 0; i < int(unknownCount2)+1; i++ {
		value := unsafeReadUint32(streamReader)
		unknownArr = append(unknownArr, int(value))
	}
	fmt.Println("Unknown array:", unknownArr)

	// Read one byte of padding
	unknownBlock2 := [1]byte{}
	if err := binary.Read(streamReader, binary.LittleEndian, &unknownBlock2); err != nil {
		return nil, fmt.Errorf("failed to read padding block: %w", err)
	}

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
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

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
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
