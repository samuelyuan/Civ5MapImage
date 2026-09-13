package fileio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

type Civ5ReplayFileConfigEntry struct {
	VariableType string
	VariableName string
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

// Civ5ReplayTile is one tile's physical-geography snapshot. PlotType is hardcoded:
// 0=PLOT_MOUNTAIN, 1=PLOT_HILLS, 2=PLOT_LAND, 3=PLOT_OCEAN.
type Civ5ReplayTile struct {
	PlotType    int8
	TerrainType int8
	Feature     int8
	RiverBits   int8
}

type Civ5ReplayData struct {
	PlayerCiv       string
	IsReplayFile    bool
	AllCivs         []Civ5ReplayCiv
	AllReplayEvents []Civ5ReplayEvent
	DatasetNames    []string
	DatasetValues   []Civ5ReplayCivDataset
	MapFileStem     string // empty when unknown (e.g. converted from a .civ5save)
	// MapWidth/MapHeight are the map's dimensions, as embedded in the .civ5replay. 0 when unknown.
	MapWidth  int
	MapHeight int
	// Tiles is the per-plot geography snapshot, row-major: index = y*MapWidth+x. Empty when
	// MapWidth/MapHeight are 0.
	Tiles []Civ5ReplayTile

	WorldSize       string
	Climate         string
	SeaLevel        string
	GameSpeed       string
	Era             string
	VictoryTypes    []string
	VictoryAchieved string
	GameOptions     []string
}

// mapFilenameStem extracts the map name from a full file path, removing the directory and extension
func mapFilenameStem(rawPath string) string {
	normalized := strings.ReplaceAll(rawPath, `\`, "/")
	base := path.Base(normalized)
	return strings.TrimSuffix(base, path.Ext(base))
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
	if eventsLength > MaxArrayLength {
		panic(fmt.Sprintf("events array length may be too long: %d", eventsLength))
	}
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

	gameName := unsafeReadFixedBytes(streamReader, 4)
	unknownUint1 := unsafeReadUint32(streamReader)
	gameVersion, err := readVarString(streamReader, "gameVersion")
	if err != nil {
		return nil, fmt.Errorf("failed to read gameVersion: %w", err)
	}
	gameBuild, err := readVarString(streamReader, "gameBuild")
	if err != nil {
		return nil, fmt.Errorf("failed to read gameBuild: %w", err)
	}
	currentTurnNumber := unsafeReadUint32(streamReader)
	unknownByte1 := unsafeReadFixedBytes(streamReader, 1)
	fmt.Printf("gameName=% X unknownUint1=%d gameVersion=%q gameBuild=%q currentTurnNumber=%d unknownByte1=% X\n",
		gameName, unknownUint1, gameVersion, gameBuild, currentTurnNumber, unknownByte1)

	playerCiv, err := readVarString(streamReader, "playerCiv")
	if err != nil {
		return nil, fmt.Errorf("failed to read player civ: %w", err)
	}

	difficulty, err := readVarString(streamReader, "difficulty")
	if err != nil {
		return nil, fmt.Errorf("failed to read difficulty: %w", err)
	}
	eraStart, err := readVarString(streamReader, "eraStart")
	if err != nil {
		return nil, fmt.Errorf("failed to read eraStart: %w", err)
	}
	eraEnd, err := readVarString(streamReader, "eraEnd")
	if err != nil {
		return nil, fmt.Errorf("failed to read eraEnd: %w", err)
	}
	gameSpeedLabel, err := readVarString(streamReader, "gameSpeed")
	if err != nil {
		return nil, fmt.Errorf("failed to read gameSpeed: %w", err)
	}
	worldSizeLabel, err := readVarString(streamReader, "worldSize")
	if err != nil {
		return nil, fmt.Errorf("failed to read worldSize: %w", err)
	}
	fmt.Printf("difficulty=%q eraStart=%q eraEnd=%q gameSpeed=%q worldSize=%q\n", difficulty, eraStart, eraEnd, gameSpeedLabel, worldSizeLabel)

	mapFilename, err := readVarString(streamReader, "mapFilename")
	if err != nil {
		return nil, fmt.Errorf("failed to read mapFilename: %w", err)
	}

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

	civName, err := readVarString(streamReader, "civName")
	if err != nil {
		return nil, fmt.Errorf("failed to read civName: %w", err)
	}
	leaderName, err := readVarString(streamReader, "leaderName")
	if err != nil {
		return nil, fmt.Errorf("failed to read leaderName: %w", err)
	}
	playerColor, err := readVarString(streamReader, "playerColor")
	if err != nil {
		return nil, fmt.Errorf("failed to read playerColor: %w", err)
	}
	replayVersion := unsafeReadUint32(streamReader)
	activePlayerIndex := unsafeReadUint32(streamReader)
	mapFilename2, err := readVarString(streamReader, "mapFilename2")
	if err != nil {
		return nil, fmt.Errorf("failed to read mapFilename2: %w", err)
	}
	fmt.Printf("civName=%q leaderName=%q playerColor=%q replayVersion=%d activePlayerIndex=%d mapFilename2=%q\n",
		civName, leaderName, playerColor, replayVersion, activePlayerIndex, mapFilename2)

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

	startTurn := unsafeReadUint32(streamReader)
	startYear := int32(unsafeReadUint32(streamReader)) // can be negative, e.g. 4000 BC
	endTurn := unsafeReadUint32(streamReader)
	endYear, err := readVarString(streamReader, "endYear")
	if err != nil {
		return nil, fmt.Errorf("failed to read endYear: %w", err)
	}
	zeroStartYear := unsafeReadUint32(streamReader)
	zeroEndYear := unsafeReadUint32(streamReader)
	fmt.Printf("startTurn=%d startYear=%d endTurn=%d endYear=%q zeroStartYear=%d zeroEndYear=%d\n",
		startTurn, startYear, endTurn, endYear, zeroStartYear, zeroEndYear)

	allCivs := readCivs(streamReader)

	datasetNames := readDatasetNames(streamReader)
	datasetValues := buildCivDatasetValues(streamReader, datasetNames)

	// Read unknown value
	_ = unsafeReadUint32(streamReader)

	allReplayEvents := readEvents(streamReader)

	mapWidth := unsafeReadUint32(streamReader)
	mapHeight := unsafeReadUint32(streamReader)
	fmt.Println("Map width:", mapWidth, ", height:", mapHeight)

	tileCount := unsafeReadUint32(streamReader)
	tiles := make([]Civ5ReplayTile, tileCount)
	for i := range tiles {
		unsafeReadUint32(streamReader) // mapEntryCount, always 1
		unsafeReadUint32(streamReader) // turnKey, always equals this file's own "End turn"
		tiles[i] = Civ5ReplayTile{
			PlotType:    int8(unsafeReadByte(streamReader)),
			TerrainType: int8(unsafeReadByte(streamReader)),
			Feature:     int8(unsafeReadByte(streamReader)),
			RiverBits:   int8(unsafeReadByte(streamReader)),
		}
	}

	replayData := Civ5ReplayData{
		PlayerCiv:       playerCiv,
		IsReplayFile:    true,
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
		DatasetNames:    datasetNames,
		DatasetValues:   datasetValues,
		MapFileStem:     mapFilenameStem(mapFilename),
		MapWidth:        int(mapWidth),
		MapHeight:       int(mapHeight),
		Tiles:           tiles,
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
