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

type Civ5ReplayData struct {
	PlayerCiv       string
	AllCivs         []Civ5ReplayCiv
	AllReplayEvents []Civ5ReplayEvent
}

func readFileConfig(reader *io.SectionReader, fileConfigEntries []Civ5ReplayFileConfigEntry) {
	for i := 0; i < len(fileConfigEntries); i++ {
		fileConfigEntry := fileConfigEntries[i]
		if fileConfigEntry.VariableType == "varstring" {
			_ = readVarString(reader)
		} else if fileConfigEntry.VariableType == "uint32" {
			_ = unsafeReadUint32(reader)
		} else if fileConfigEntry.VariableType == "int32" {
			signedIntValue := int32(0)
			if err := binary.Read(reader, binary.LittleEndian, &signedIntValue); err != nil {
				log.Fatal("Failed to load int32: ", err)
			}
		} else if fileConfigEntry.VariableType == "uint16" {
			_ = unsafeReadUint16(reader)
		} else if fileConfigEntry.VariableType == "uint8" {
			unsignedIntValue := uint8(0)
			if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
				log.Fatal("Failed to load uint8: ", err)
			}
		} else if strings.Contains(fileConfigEntry.VariableType, "bytearray") {
			byteArrayLength, err := strconv.Atoi(fileConfigEntry.VariableType[len("bytearray:"):])
			if err != nil {
				log.Fatal("Invalid byte array type in file config:", err)
			}

			byteBlock := make([]byte, byteArrayLength)
			if err := binary.Read(reader, binary.LittleEndian, &byteBlock); err != nil {
				log.Fatal("Invalid byte array data", err)
			}
		} else {
			fmt.Println("Unknown variable type:", fileConfigEntry.VariableType)
		}
	}
}

func readVarString(reader *io.SectionReader) string {
	variableLength := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &variableLength); err != nil {
		log.Fatal("Failed to load variable length: ", err)
	}

	stringValue := make([]byte, variableLength)
	if err := binary.Read(reader, binary.LittleEndian, &stringValue); err != nil {
		log.Fatal("Failed to load string value: ", err)
	}

	return string(stringValue[:])
}

func readArray(reader *io.SectionReader, arrayName string, fileConfigEntries []Civ5ReplayFileConfigEntry) {
	arrayLength := unsafeReadUint32(reader)
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
		leader := readVarString(reader)
		longName := readVarString(reader)
		name := readVarString(reader)
		demonym := readVarString(reader)

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
		eventText := readVarString(reader)

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
			VariableType: "bytearray:4",
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
			VariableType: "bytearray:5",
			VariableName: "unknownBlock2",
		},
	})

	playerCiv := readVarString(streamReader)

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
			VariableName: "mapScript",
		},
	})

	readArray(streamReader, "dlc", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:16",
			VariableName: "id",
		},
		{
			VariableType: "bytearray:4",
			VariableName: "enabled",
		},
		{
			VariableType: "varstring",
			VariableName: "name",
		},
	})

	readArray(streamReader, "mods", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "id",
		},
		{
			VariableType: "bytearray:4",
			VariableName: "version",
		},
		{
			VariableType: "varstring",
			VariableName: "name",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "unknownBlock3",
		},
		{
			VariableType: "varstring",
			VariableName: "unknownBlock4",
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
			VariableName: "mapScript2",
		},
	})

	for {
		unknownBlock := [4]byte{}
		if err := binary.Read(streamReader, binary.LittleEndian, &unknownBlock); err != nil {
			log.Fatal("Failed to read block: ", err)
		}
		if unknownBlock[0] == 255 && unknownBlock[1] == 255 && unknownBlock[2] == 255 && unknownBlock[3] == 255 {
			unknownBlock2 := [1]byte{}
			if err := binary.Read(streamReader, binary.LittleEndian, &unknownBlock2); err != nil {
				log.Fatal("Failed to read block: ", err)
			}
			break
		}
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

	readArray(streamReader, "datasets", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "name",
		},
	})

	datasetValuesArray1Length := unsafeReadUint32(streamReader)
	for i := 0; i < int(datasetValuesArray1Length); i++ {
		datasetValuesArray2Length := unsafeReadUint32(streamReader)
		for j := 0; j < int(datasetValuesArray2Length); j++ {
			readArray(streamReader, "datasetValues", []Civ5ReplayFileConfigEntry{
				{
					VariableType: "uint32",
					VariableName: "turn",
				},
				{
					VariableType: "uint32",
					VariableName: "value",
				},
			})
		}
	}

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
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
	}

	return &replayData, nil
}
