package fileio

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"strings"
)

// Save file format markers
const (
	// Save file version that includes unit/unit-class/building-class name arrays
	SaveVersionWithUnitClassData = 0x0B

	// Some unknown array lengths are one greater than the true length past this threshold
	ArrayLengthCorrectionThreshold = 150
)

type Civ5SaveData struct {
	PlayerCiv       string
	IsReplayFile    bool
	AllCivs         []Civ5ReplayCiv
	AllReplayEvents []Civ5ReplayEvent
}

func readClimateName(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "climateName1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:12",
			VariableName: "paddingAfterClimateName1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "climateNameType",
		},
		{
			VariableType: "varstring",
			VariableName: "climateNameDescription",
		},
		{
			VariableType: "varstring",
			VariableName: "climateName2",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "int32",
			VariableName: "desertPercentChange",
		},
		{
			VariableType: "uint32",
			VariableName: "jungleLatitude",
		},
		{
			VariableType: "uint32",
			VariableName: "hillRange",
		},
		{
			VariableType: "uint32",
			VariableName: "mountainPercent",
		},
		{
			VariableType: "float32",
			VariableName: "snowLatitudeChange",
		},
		{
			VariableType: "float32",
			VariableName: "tundraLatitudeChange",
		},
		{
			VariableType: "float32",
			VariableName: "grassLatitudeChange",
		},
		{
			VariableType: "float32",
			VariableName: "desertBottomLatitudeChange",
		},
		{
			VariableType: "float32",
			VariableName: "desertTopLatitudeChange",
		},
		{
			VariableType: "float32",
			VariableName: "iceLatitude",
		},
		{
			VariableType: "float32",
			VariableName: "randIceLatitude",
		},
	})
}

func readSeaLevel(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "seaLevelName1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:12",
			VariableName: "paddingAfterSeaLevel1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "seaLevelNameType",
		},
		{
			VariableType: "varstring",
			VariableName: "seaLevelNameDescription",
		},
		{
			VariableType: "varstring",
			VariableName: "seaLevelName2",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:5",
			VariableName: "paddingAfterSeaLevel2",
		},
	})
}

func readTurnSpeedData(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "turnTimerId1",
		},
		{
			VariableType: "uint32",
			VariableName: "turnTimerUnknown1",
		},
		{
			VariableType: "varstring",
			VariableName: "turnTimeName1",
		},
		{
			VariableType: "bytearray:12",
			VariableName: "paddingAfterTurnTime1",
		},
		{
			VariableType: "varstring",
			VariableName: "turnTimeNameType",
		},
		{
			VariableType: "varstring",
			VariableName: "turnTimeNameDescription",
		},
		{
			VariableType: "varstring",
			VariableName: "turnTimeName2",
		},
		{
			VariableType: "uint32",
			VariableName: "turnTimerBase",
		},
		{
			VariableType: "uint32",
			VariableName: "turnTimerCity",
		},
		{
			VariableType: "uint32",
			VariableName: "turnTimerUnit",
		},
		{
			VariableType: "uint32",
			VariableName: "turnTimerFirstTurnMultiplayer",
		},
		{
			VariableType: "uint32",
			VariableName: "turnTimerId2",
		},
		{
			VariableType: "uint8",
			VariableName: "turnTimerUnknown2",
		},
	})

	readArray(streamReader, "turnTimerVictoryFlags", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8", // length is usually 5
			VariableName: "victoryFlag",
		},
	})
}

func readWorldSizeData(streamReader *io.SectionReader) {
	numberBeforeWorldSize := unsafeReadUint32(streamReader)
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "portraitIndex1",
		},
	})

	// Should be related to map version
	if numberBeforeWorldSize == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
				VariableName: "numBeforeWorldSize",
			},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "worldSize1",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSizeHelp",
		},
		{
			VariableType: "bytearray:8",
			VariableName: "paddingAfterWorldSize1",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSizeType",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSizeDescription",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSize2",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "defaultPlayers",
		},
		{
			VariableType: "uint32",
			VariableName: "defaultMinorCivs",
		},
		{
			VariableType: "uint32",
			VariableName: "fogTilesPerBarbarianCamp",
		},
		{
			VariableType: "uint32",
			VariableName: "numNaturalWonders",
		},
		{
			VariableType: "uint32",
			VariableName: "unitNameModifier",
		},
		{
			VariableType: "uint32",
			VariableName: "targetNumCities",
		},
		{
			VariableType: "uint32",
			VariableName: "numFreeBuildingResources",
		},
		{
			VariableType: "uint32",
			VariableName: "buildingClassPrereqModifier",
		},
		{
			VariableType: "int32",
			VariableName: "maxConscriptModifier",
		},
		{
			VariableType: "uint32",
			VariableName: "gridWidth",
		},
		{
			VariableType: "uint32",
			VariableName: "gridHeight",
		},
	})

	if numberBeforeWorldSize == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
				VariableName: "maxActiveReligions",
			},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "int32",
			VariableName: "terrainGrainChange",
		},
		{
			VariableType: "int32",
			VariableName: "featureGrainChange",
		},
		{
			VariableType: "uint32",
			VariableName: "researchPercent",
		},
		{
			VariableType: "uint32",
			VariableName: "advancedStartPointsMod",
		},
		{
			VariableType: "uint32",
			VariableName: "numCitiesUnhappinessPercent",
		},
		{
			VariableType: "uint32",
			VariableName: "numCitiesPolicyCostMod",
		},
		{
			VariableType: "uint32",
			VariableName: "numCitiesTechCostMod",
		},
	})

	if numberBeforeWorldSize == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
				VariableName: "portraitIndex2",
			},
		})
	}
}

func readGameOptions(streamReader *io.SectionReader) {
	readArray(streamReader, "gameOptionArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "gameOption",
		},
		{
			VariableType: "uint32",
			VariableName: "gameOptionEnabled",
		},
	})
}

func buildReaderForDecompressedFile(compressedStreamReader *io.SectionReader, outputFilename string) (*bytes.Reader, int, error) {
	decompressedFileReader, err := zlib.NewReader(compressedStreamReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create zlib new reader: %w", err)
	}
	defer decompressedFileReader.Close()

	decompressedContents, err := io.ReadAll(decompressedFileReader)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			fmt.Println("Read file into memory and still succeeded, err:", err)
		} else if err != nil {
			fmt.Println("Error when decompressing zlib, still attempt to continue:", err)
		}

		fmt.Println("Decompressed contents size:", len(decompressedContents))
		err = os.WriteFile(outputFilename, decompressedContents, 0644)
		if err != nil {
			return nil, 0, fmt.Errorf("error writing to %q: %w", outputFilename, err)
		}
	}

	return bytes.NewReader(decompressedContents), len(decompressedContents), nil
}

// readVarStringArrayOrPanic reads a count-prefixed array of variable-length strings.
// Matches the file's existing convention of treating a malformed stream at this point
// as unrecoverable, rather than threading an error back through every caller.
func readVarStringArrayOrPanic(reader *io.SectionReader, count uint32, varStringLabel, panicPhrase string) []string {
	values := make([]string, count)
	for i := 0; i < int(count); i++ {
		value, err := readVarString(reader, varStringLabel)
		if err != nil {
			panic(fmt.Sprintf("failed to read %s: %v", panicPhrase, err))
		}
		values[i] = value
	}
	return values
}

// readDynamicPaddingBlock reads a size-prefixed padding block whose length is derived from a
// marker value read just before it, matching the file's "(marker+1) groups of 4 bytes" pattern.
// A marker of 0 means the block is absent.
func readDynamicPaddingBlock(reader *io.SectionReader, marker uint32, blockName string) {
	if marker == 0 {
		return
	}
	readFileConfig(reader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: fmt.Sprintf("bytearray:%d", (marker+1)*4),
			VariableName: blockName,
		},
	})
}

// openSaveFileReader opens a save file and returns a section reader spanning its entire contents
func openSaveFileReader(filename string) (*os.File, int64, *io.SectionReader, error) {
	inputFile, err := os.Open(filename)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to open file %q: %w", filename, err)
	}
	fi, err := inputFile.Stat()
	if err != nil {
		inputFile.Close()
		return nil, 0, nil, fmt.Errorf("failed to get file info for %q: %w", filename, err)
	}
	saveFileLength := fi.Size()
	return inputFile, saveFileLength, io.NewSectionReader(inputFile, int64(0), saveFileLength), nil
}

// readSaveHeader reads the game name/version/build/turn number header and the active player's civ
func readSaveHeader(streamReader *io.SectionReader) (string, error) {
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
			VariableType: "uint32",
			VariableName: "currentTurnNumber",
		},
		{
			VariableType: "bytearray:1",
			VariableName: "unknownBlock2",
		},
	})

	playerCiv, err := readVarString(streamReader, "playerCiv")
	if err != nil {
		return "", fmt.Errorf("failed to read player civ: %w", err)
	}
	fmt.Println("Player civ:", playerCiv)

	return playerCiv, nil
}

// readGameSettingsAndContent reads the difficulty/era/speed/world-size block and the DLC and mod lists
func readGameSettingsAndContent(streamReader *io.SectionReader) {
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
			VariableName: "mapFilename1",
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
}

// readPlayerAndMapInfo reads the player civ block, the player name array, and a handful of
// arrays of unknown purpose that follow it
func readPlayerAndMapInfo(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "playerCivName",
		},
		{
			VariableType: "varstring",
			VariableName: "playerLeaderName",
		},
		{
			VariableType: "varstring",
			VariableName: "playerColor",
		},
		{
			VariableType: "bytearray:16",
			VariableName: "unknownId1",
		},
		{
			VariableType: "varstring",
			VariableName: "version",
		},
		{
			VariableType: "bytearray:16",
			VariableName: "unknownId2",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownId3-1",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownId3-2",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownId3-3",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownId3-4",
		},
		{
			VariableType: "varstring",
			VariableName: "mapFilename2",
		},
	})

	readArray(streamReader, "unknownBlock3", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "int32",
			VariableName: "unknownBlock3Var",
		},
	})

	readArray(streamReader, "playerNameArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "playerName",
		},
	})

	// 4 arrays, but value is unknown
	for i := 0; i < 4; i++ {
		readArray(streamReader, fmt.Sprintf("unknownArrayBlock1-%d", i), []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
				VariableName: fmt.Sprintf("unknownArrayBlock1-%d", i),
			},
		})
	}
}

// readCivRoster reads the list of civilization names and builds the initial civ roster
func readCivRoster(streamReader *io.SectionReader) []Civ5ReplayCiv {
	civNamesLength := unsafeReadUint32(streamReader)
	fmt.Println("CivNamesLength:", civNamesLength)
	civNameArr := readVarStringArrayOrPanic(streamReader, civNamesLength, "civName", "civ name")
	fmt.Println("CivNames:", civNameArr)

	allCivs := make([]Civ5ReplayCiv, 0, len(civNameArr))
	for _, civName := range civNameArr {
		allCivs = append(allCivs, Civ5ReplayCiv{
			UnknownVariables: [4]int{0, 0, 0, 0},
			Leader:           "",
			LongName:         "",
			Name:             civName,
			Demonym:          "",
		})
	}
	return allCivs
}

// readLeadersAndCivArrays reads the leader name array and several more arrays of unknown purpose
func readLeadersAndCivArrays(streamReader *io.SectionReader) {
	readArray(streamReader, "leaderArray1", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "arrLeaderName",
		},
	})

	unknownBlock5Number := unsafeReadUint32(streamReader)
	if unknownBlock5Number != 0 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:12",
				VariableName: "unknownBlock5",
			},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "computerUsername1",
		},
	})

	readArray(streamReader, "unknownBlock6-1", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "int32",
			VariableName: "unknownBlock6-1",
		},
	})
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:53",
			VariableName: "unknownBlock6-2",
		},
	})

	readArray(streamReader, "unknownArray1", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownArray1Var",
		},
	})

	readArray(streamReader, "civArray1", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "civName",
		},
	})

	readArray(streamReader, "unknownArray2", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownArray2Var",
		},
	})

	readArray(streamReader, "civArray2", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "civArray2String",
		},
	})
}

// readClimateSection reads a variable-length unknown block followed by the climate name section
func readClimateSection(streamReader *io.SectionReader) {
	unknownBlock7Number := unsafeReadUint32(streamReader)
	readDynamicPaddingBlock(streamReader, unknownBlock7Number, "unknownBlock7-1")

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:8",
			VariableName: "unknownBlock7-2",
		},
	})
	readClimateName(streamReader)
}

// readGameNameAndTurnInfo reads the save's game name, current turn number, and a trailing
// array whose presence depends on a peeked-ahead marker value
func readGameNameAndTurnInfo(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownUint32",
		},
	})

	readArray(streamReader, "unknownBlock8-1", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownBlock8-1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownUint32",
		},
	})

	readArray(streamReader, "unknownBlock8-2", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownBlock8-2",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:15",
			VariableName: "unknownBlock8-3",
		},
	})

	gameName, err := readVarString(streamReader, "gameName")
	if err != nil {
		panic(fmt.Sprintf("failed to read game name: %v", err))
	}
	fmt.Println("Game name:", gameName)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownUint32", // usually equal to 2
		},
		{
			VariableType: "uint8",
			VariableName: "unknownUint8",
		},
		{
			VariableType: "uint32",
			VariableName: "currentTurnNumber",
		},
		{
			VariableType: "bytearray:5",
			VariableName: "unknownBlock9",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownUint32",
		},
	})

	readArray(streamReader, "unknownArray3", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownArray3Var",
		},
	})

	// Some save files missing extra array
	nextByte := unsafeReadUint16(streamReader)
	if nextByte != 0 {
		streamReader.Seek(-2, io.SeekCurrent)
		readArray(streamReader, "unknownArray4", []Civ5ReplayFileConfigEntry{
			{
				VariableType: "int32", // a lot of negative values
				VariableName: "unknownArray4Var",
			},
		})
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:2",
				VariableName: "unknownBlock10",
			},
		})
	}
}

// readLeaderArray2AndPlayerSetup reads the second leader name array and the computer username/map block
func readLeaderArray2AndPlayerSetup(streamReader *io.SectionReader) {
	readArray(streamReader, "leaderArray2", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "leaderArrName",
		},
	})

	unknownBlock11Number := unsafeReadUint32(streamReader)
	readDynamicPaddingBlock(streamReader, unknownBlock11Number, "unknownBlock11")

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "computerUsername2",
		},
		{
			VariableType: "bytearray:7",
			VariableName: "unknownId4",
		},
		{
			VariableType: "varstring",
			VariableName: "mapFilename3",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownBlock12-1",
		},
		{
			VariableType: "uint32",
			VariableName: "maxTurns",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownBlock12-3",
		},
	})
}

// readMinorCivNames reads the minor civ (city-state) names and patches matching entries in the civ roster
func readMinorCivNames(streamReader *io.SectionReader, allCivs []Civ5ReplayCiv) {
	minorCivNamesLength := unsafeReadUint32(streamReader)
	minorCivNameArr := readVarStringArrayOrPanic(streamReader, minorCivNamesLength, "minorCivName", "minor civ name")
	for i, minorCivName := range minorCivNameArr {
		if strings.Contains(minorCivName, "MINOR_CIV") {
			allCivs[i].Name = minorCivName
		}
	}
	fmt.Println("minorCivArray:", minorCivNameArr)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:77",
			VariableName: "unknownBlock13",
		},
	})
}

// readPlayerArraysAndColors reads several player-related arrays and patches the civ roster with player colors
func readPlayerArraysAndColors(streamReader *io.SectionReader, allCivs []Civ5ReplayCiv) {
	readArray(streamReader, "unknownArray5", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "int32", // a lot of negative values
			VariableName: "unknownArray5Var",
		},
	})

	readArray(streamReader, "playerArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "playerArrName",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:8",
			VariableName: "unknownBlock14",
		},
	})

	readArray(streamReader, "unknownArray6", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "unknownArray6Var",
		},
	})

	playerColorLength := unsafeReadUint32(streamReader)
	playerColorArr := readVarStringArrayOrPanic(streamReader, playerColorLength, "playerColorName", "player color name")
	for i, playerColorName := range playerColorArr {
		allCivs[i].LongName = playerColorName
	}
	fmt.Println("playerColorArr:", playerColorArr)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:10",
			VariableName: "unknownBlock15",
		},
	})

	readArray(streamReader, "unknownArray7", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "unknownArray7Var",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:12",
			VariableName: "unknownBlock16",
		},
	})
}

// readSeaLevelAndWorldSettings reads the sea level, turn speed, world size, and game option sections
func readSeaLevelAndWorldSettings(streamReader *io.SectionReader) {
	readSeaLevel(streamReader)

	readArray(streamReader, "unknownArray8", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownArray8Var",
		},
	})

	readArray(streamReader, "unknownArray9", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownArray9Var",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:12",
			VariableName: "unknownBlock17",
		},
	})

	readArray(streamReader, "unknownArray10", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownArray10Var",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:1",
			VariableName: "unknownBlock18",
		},
	})

	readTurnSpeedData(streamReader)
	readArray(streamReader, "unknownArrayAfterTurnSpeed", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "unknownArrayAfterTurnSpeedVar",
		},
	})
	readWorldSizeData(streamReader)
	readGameOptions(streamReader)

	readArray(streamReader, "unknownArrayAfterGameOptions", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:9",
			VariableName: "valueAfterGameOptions",
		},
	})
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "gameVersion2",
		},
	})

	readArray(streamReader, "unknownArray12", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "unknownArray12Var",
		},
	})

	readArray(streamReader, "unknownArray13", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "unknownArray13Var",
		},
	})

	readArray(streamReader, "unknownArray14", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknownArray14Var",
		},
	})
}

// locateCompressedBlock skips the padding before the compressed block and returns a reader
// positioned at its start
func locateCompressedBlock(streamReader *io.SectionReader, inputFile *os.File, saveFileLength int64) (*io.SectionReader, error) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:8", // value is always [2 0 0 0 0 0 1 0]
			VariableName: "paddingBeforeCompressedBlock",
		},
	})

	// Header of compressed block should begin with 0x789C
	offsetToCompressedBlock, err := streamReader.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to get current position: %w", err)
	}
	fmt.Println("Offset to compressed data:", offsetToCompressedBlock)

	return io.NewSectionReader(inputFile, offsetToCompressedBlock, saveFileLength-offsetToCompressedBlock), nil
}

func ReadCiv5SaveFile(filename string, outputFilename string) (*Civ5SaveData, error) {
	inputFile, saveFileLength, streamReader, err := openSaveFileReader(filename)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()
	fmt.Println("Loading Civ5Save...")

	playerCiv, err := readSaveHeader(streamReader)
	if err != nil {
		return nil, err
	}

	readGameSettingsAndContent(streamReader)
	readPlayerAndMapInfo(streamReader)

	allCivs := readCivRoster(streamReader)

	readLeadersAndCivArrays(streamReader)
	readClimateSection(streamReader)
	readGameNameAndTurnInfo(streamReader)
	readLeaderArray2AndPlayerSetup(streamReader)
	readMinorCivNames(streamReader, allCivs)
	readPlayerArraysAndColors(streamReader, allCivs)
	readSeaLevelAndWorldSettings(streamReader)

	compressedStreamReader, err := locateCompressedBlock(streamReader, inputFile, saveFileLength)
	if err != nil {
		return nil, err
	}

	decompressedStreamReader, decompressedContentsSize, err := buildReaderForDecompressedFile(compressedStreamReader, outputFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress file: %w", err)
	}
	allReplayEvents := readDecompressed(decompressedStreamReader, decompressedContentsSize)

	return &Civ5SaveData{
		PlayerCiv:       playerCiv,
		IsReplayFile:    false,
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
	}, nil
}

// readDecompressedHeader reads the decompressed block's version/turn header and two leading unknown sections
func readDecompressedHeader(streamReader *io.SectionReader) uint32 {
	saveFileVersion := unsafeReadUint32(streamReader)
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32", // value is usually 0
			VariableName: "unknown2",
		},
		{
			VariableType: "uint32",
			VariableName: "turnNumber",
		},
		{
			VariableType: "uint32",
			VariableName: "unknown3",
		},
		{
			VariableType: "uint32",
			VariableName: "unknown4",
		},
		{
			VariableType: "int32",
			VariableName: "startYear",
		},
	})

	for i := 0; i < 24; i++ {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "int32",
				VariableName: fmt.Sprintf("unknownSection1-%d", i),
			},
		})
	}

	// Seems to be a list of flags
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:10",
			VariableName: "unknownSection2",
		},
	})

	return saveFileVersion
}

// readOptionsAndPadding reads the game options array and a large fixed-size padding block
func readOptionsAndPadding(streamReader *io.SectionReader) {
	readArray(streamReader, "optionsArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "optionsArrName",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:1844", // consistent between files
			VariableName: "unknownSection3",
		},
	})
}

// readVersionDependentUnitData reads the section whose layout differs between save file versions:
// newer saves (SaveVersionWithUnitClassData) store named unit/unit-class/building-class arrays,
// while older saves store a set of fixed-width unknown arrays and blocks instead
func readVersionDependentUnitData(streamReader *io.SectionReader, saveFileVersion uint32) {
	if saveFileVersion == SaveVersionWithUnitClassData {
		readArray(streamReader, "unitNameArr", []Civ5ReplayFileConfigEntry{
			{
				VariableType: "varstring",
				VariableName: "unitName",
			},
			{
				VariableType: "uint32",
				VariableName: "unknownValue",
			},
		})
		readArray(streamReader, "unitClassArr", []Civ5ReplayFileConfigEntry{
			{
				VariableType: "varstring",
				VariableName: "unitClass",
			},
			{
				VariableType: "uint32",
				VariableName: "unknownValue",
			},
		})
		readArray(streamReader, "buildingClassArr", []Civ5ReplayFileConfigEntry{
			{
				VariableType: "varstring",
				VariableName: "buildingClass",
			},
			{
				VariableType: "uint32",
				VariableName: "unknownValue",
			},
		})

		// TODO: find padding
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:2366", // for RED WW2 save files, but different for other mods
				VariableName: "unknownPadding",
			},
		})
		return
	}

	// Three unknown arrays
	// The size of the first and the third arrays are the same unless first array size is greater than 150, which means the first array
	// length is one more than the third array length. The second array is much smaller.
	// Each array element is 8 bytes. The first 4 bytes are usually consistent between different save files. The last 4 bytes can vary.

	// Array 1 length: Usually 128 or 132, but some files have other values like [127, 154, 157]
	arrayLength := unsafeReadUint32(streamReader)
	// Can be one less for some save files
	if arrayLength >= ArrayLengthCorrectionThreshold {
		arrayLength = arrayLength - 1
	}
	for i := 0; i < int(arrayLength); i++ {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:8",
				VariableName: "unknownSection4-1",
			},
		})
	}

	// Array 2 length: Usually 83, but can be 85 in a save file when array 1 length is greater than 150
	readArray(streamReader, "unknownSection4-2", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:8",
			VariableName: "unknownSection4-2",
		},
	})

	// Array 3 length: Usually 128 or 132, but some files have other values like [127, 153, 156]
	arrayLength3 := unsafeReadUint32(streamReader)
	for i := 0; i < int(arrayLength3)-1; i++ {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:8",
				VariableName: "unknownSection4-3",
			},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:128",
			VariableName: "unknownSection5-1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:756",
			VariableName: "unknownSection5-2",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:128",
			VariableName: "unknownSection5-3",
		},
	})
}

// readGreatPersonAndTrailingBlocks reads the great person array and the fixed-size blocks that follow it
func readGreatPersonAndTrailingBlocks(streamReader *io.SectionReader) {
	readArray(streamReader, "greatPersonArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "greatPersonName",
		},
	})

	for i := 0; i < 2; i++ {
		// Constant block
		// [8 0 0 0 255 255 255 255 255 255 255 255 0 0 0 0
		// 0 32 0 0 255 255 255 255 255 255 255 255 255 255 255 255
		// 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255
		// 255 255 255 255 0 0 0 0]
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:56",
				VariableName: "constantBlock",
			},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:38",
			VariableName: "unknownSectionAfterGreatPerson",
		},
	})
}

func readDecompressed(reader *bytes.Reader, decompressedFileLength int) []Civ5ReplayEvent {
	streamReader := io.NewSectionReader(reader, int64(0), int64(decompressedFileLength))

	saveFileVersion := readDecompressedHeader(streamReader)
	readOptionsAndPadding(streamReader)
	readVersionDependentUnitData(streamReader, saveFileVersion)
	readGreatPersonAndTrailingBlocks(streamReader)

	allReplayEvents := readEvents(streamReader)
	fmt.Printf("Read %d replay events\n", len(allReplayEvents))

	return allReplayEvents
}
