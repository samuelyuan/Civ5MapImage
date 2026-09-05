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

var worldSizeNames = []string{
	"WORLDSIZE_DUEL",
	"WORLDSIZE_TINY",
	"WORLDSIZE_SMALL",
	"WORLDSIZE_STANDARD",
	"WORLDSIZE_LARGE",
	"WORLDSIZE_HUGE",
}

var gameSpeedNames = []string{
	"GAMESPEED_MARATHON",
	"GAMESPEED_EPIC",
	"GAMESPEED_STANDARD",
	"GAMESPEED_QUICK",
}

var climateNames = []string{
	"CLIMATE_TEMPERATE",
	"CLIMATE_TROPICAL",
	"CLIMATE_ARID",
	"CLIMATE_ROCKY",
	"CLIMATE_COLD",
}

var seaLevelNames = []string{
	"SEALEVEL_LOW",
	"SEALEVEL_MEDIUM",
	"SEALEVEL_HIGH",
}

var eraNames = []string{
	"ERA_ANCIENT",
	"ERA_CLASSICAL",
	"ERA_MEDIEVAL",
	"ERA_RENAISSANCE",
	"ERA_INDUSTRIAL",
	"ERA_MODERN",
	"ERA_POSTMODERN",
	"ERA_FUTURE",
}

var victoryTypeNames = []string{
	"VICTORY_TIME",
	"VICTORY_SPACE_RACE",
	"VICTORY_DOMINATION",
	"VICTORY_CULTURAL",
	"VICTORY_DIPLOMATIC",
}

var handicapNames = []string{
	"HANDICAP_SETTLER",
	"HANDICAP_CHIEFTAIN",
	"HANDICAP_WARLORD",
	"HANDICAP_PRINCE",
	"HANDICAP_KING",
	"HANDICAP_EMPEROR",
	"HANDICAP_IMMORTAL",
	"HANDICAP_DEITY",
}

var plotTypeNames = []string{
	"PLOT_MOUNTAIN",
	"PLOT_HILLS",
	"PLOT_LAND",
	"PLOT_OCEAN",
}

var gameOptionNames = []string{
	"GAMEOPTION_NO_CITY_RAZING",
	"GAMEOPTION_NO_BARBARIANS",
	"GAMEOPTION_RAGING_BARBARIANS",
	"GAMEOPTION_ALWAYS_WAR",
	"GAMEOPTION_ALWAYS_PEACE",
	"GAMEOPTION_ONE_CITY_CHALLENGE",
	"GAMEOPTION_NO_CHANGING_WAR_PEACE",
	"GAMEOPTION_NEW_RANDOM_SEED",
	"GAMEOPTION_LOCK_MODS",
	"GAMEOPTION_COMPLETE_KILLS",
	"GAMEOPTION_NO_GOODY_HUTS",
	"GAMEOPTION_RANDOM_PERSONALITIES",
	"GAMEOPTION_POLICY_SAVING",
	"GAMEOPTION_PROMOTION_SAVING",
	"GAMEOPTION_END_TURN_TIMER_ENABLED",
	"GAMEOPTION_QUICK_COMBAT",
	"GAMEOPTION_DISABLE_START_BIAS",
	"GAMEOPTION_NO_SCIENCE",
	"GAMEOPTION_NO_POLICIES",
	"GAMEOPTION_NO_HAPPINESS",
	"GAMEOPTION_NO_TUTORIAL",
	"GAMEOPTION_NO_RELIGION",
}

type Civ5ReplayCiv struct {
	CivilizationIndex int
	LeaderTypeIndex   int
	PlayerColorIndex  int
	Difficulty        string
	Leader            string
	LongName          string
	Name              string
	Demonym           string
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
	// MapWidth and MapHeight are the dimensions of the map this replay was recorded on, as
	// embedded in the .civ5replay file itself. They are 0 when unknown (e.g. a replay
	// converted from a .civ5save file, which doesn't carry this information).
	MapWidth  int
	MapHeight int

	WorldSize       string
	Climate         string
	SeaLevel        string
	GameSpeed       string
	Era             string
	VictoryTypes    []string
	VictoryAchieved string
	GameOptions     []string
}

func readCivs(reader *io.SectionReader) []Civ5ReplayCiv {
	civsLength := unsafeReadUint32(reader)
	allCivs := make([]Civ5ReplayCiv, 0)

	for i := 0; i < int(civsLength); i++ {
		civilizationIndex := unsafeReadUint32(reader)
		leaderTypeIndex := unsafeReadUint32(reader)
		playerColorIndex := unsafeReadUint32(reader)
		difficultyIndex := unsafeReadUint32(reader)
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
			CivilizationIndex: int(civilizationIndex),
			LeaderTypeIndex:   int(leaderTypeIndex),
			PlayerColorIndex:  int(playerColorIndex),
			Difficulty:        typeName(handicapNames, int(difficultyIndex)),
			Leader:            leader,
			LongName:          longName,
			Name:              name,
			Demonym:           demonym,
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
			VariableName: "unknownUint1",
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
			VariableName: "unknownByte1",
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
			VariableType: "uint32",
			VariableName: "replayVersion",
		},
		{
			VariableType: "uint32",
			VariableName: "activePlayerIndex",
		},
		{
			VariableType: "varstring",
			VariableName: "mapFilename2",
		},
	})

	worldSizeIndex := unsafeReadUint32(streamReader)
	worldSizeName := typeName(worldSizeNames, int(worldSizeIndex))
	fmt.Println("World size:", worldSizeName)

	climateIndex := unsafeReadUint32(streamReader)
	seaLevelIndex := unsafeReadUint32(streamReader)
	eraIndex := unsafeReadUint32(streamReader)
	gameSpeedIndex := unsafeReadUint32(streamReader)
	climateName := typeName(climateNames, int(climateIndex))
	seaLevelName := typeName(seaLevelNames, int(seaLevelIndex))
	eraName := typeName(eraNames, int(eraIndex))
	gameSpeedName := typeName(gameSpeedNames, int(gameSpeedIndex))
	fmt.Println("Climate:", climateName)
	fmt.Println("Sea level:", seaLevelName)
	fmt.Println("Era:", eraName)
	fmt.Println("Game speed:", gameSpeedName)

	gameOptionCount := unsafeReadUint32(streamReader)
	gameOptions := make([]string, 0, gameOptionCount)
	for i := 0; i < int(gameOptionCount); i++ {
		value := unsafeReadUint32(streamReader)
		gameOptions = append(gameOptions, typeName(gameOptionNames, int(value)))
	}
	fmt.Println("Game options enabled:", gameOptions)

	victoryTypeCount := unsafeReadUint32(streamReader)
	victoryTypes := make([]string, 0, victoryTypeCount)
	for i := 0; i < int(victoryTypeCount); i++ {
		value := unsafeReadUint32(streamReader)
		victoryTypes = append(victoryTypes, typeName(victoryTypeNames, int(value)))
	}
	fmt.Println("Victory types enabled:", victoryTypes)

	victoryTypeIndex := int32(unsafeReadUint32(streamReader))
	victoryTypeName := typeName(victoryTypeNames, int(victoryTypeIndex))
	fmt.Println("Victory type (index):", victoryTypeIndex, "name:", victoryTypeName)

	unknownByte2 := [1]byte{}
	if err := binary.Read(streamReader, binary.LittleEndian, &unknownByte2); err != nil {
		return nil, fmt.Errorf("failed to read block: %w", err)
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

	mapWidth := unsafeReadUint32(streamReader)
	mapHeight := unsafeReadUint32(streamReader)
	fmt.Println("Map width:", mapWidth, ", height:", mapHeight)

	readArray(streamReader, "tiles", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "mapEntryCount",
		},
		{
			VariableType: "uint32",
			VariableName: "turnKey",
		},
		{
			VariableType: "uint8",
			VariableName: "plotTypeIndex",
		},
		{
			VariableType: "uint8",
			VariableName: "terrainIndex",
		},
		{
			VariableType: "uint8",
			VariableName: "featureIndex",
		},
		{
			VariableType: "uint8",
			VariableName: "riverBits",
		},
	})

	replayData := Civ5ReplayData{
		PlayerCiv:       playerCiv,
		IsReplayFile:    true,
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
		DatasetNames:    datasetNames,
		DatasetValues:   datasetValues,
		MapWidth:        int(mapWidth),
		MapHeight:       int(mapHeight),
		WorldSize:       worldSizeName,
		Climate:         climateName,
		SeaLevel:        seaLevelName,
		GameSpeed:       gameSpeedName,
		Era:             eraName,
		VictoryTypes:    victoryTypes,
		GameOptions:     gameOptions,
		VictoryAchieved: victoryTypeName,
	}

	return &replayData, nil
}
