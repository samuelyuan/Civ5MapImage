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

	NumVictoryPointAwards = 5

	MaxPlayers = 64
)

type Civ5SaveData struct {
	PlayerCiv       string
	IsReplayFile    bool
	AllCivs         []Civ5ReplayCiv
	AllReplayEvents []Civ5ReplayEvent
}

func readClimateSection(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "climateEnum"},
		{VariableType: "uint32", VariableName: "climateInfoId"},
		{VariableType: "uint32", VariableName: "climateInfoCivilopedia"},
	})
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "climateDisplayName"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "climateHelp"},
		{VariableType: "uint32", VariableName: "climateDisabledHelp"},
		{VariableType: "uint32", VariableName: "climateStrategy"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "climateType"},
		{VariableType: "varstring", VariableName: "climateTextKey"},
		{VariableType: "varstring", VariableName: "climateDisplayName2"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "desertPercentChange"},
		{VariableType: "uint32", VariableName: "jungleLatitude"},
		{VariableType: "uint32", VariableName: "hillRange"},
		{VariableType: "uint32", VariableName: "mountainPercent"},
		{VariableType: "float32", VariableName: "snowLatitudeChange"},
		{VariableType: "float32", VariableName: "tundraLatitudeChange"},
		{VariableType: "float32", VariableName: "grassLatitudeChange"},
		{VariableType: "float32", VariableName: "desertBottomLatitudeChange"},
		{VariableType: "float32", VariableName: "desertTopLatitudeChange"},
		{VariableType: "float32", VariableName: "iceLatitude"},
		{VariableType: "float32", VariableName: "randIceLatitude"},
	})
}

func readSeaLevel(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "seaLevelEnum"},
		{VariableType: "uint32", VariableName: "seaLevelInfoId"},
		{VariableType: "uint32", VariableName: "seaLevelInfoCivilopedia"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "seaLevelDisplayName"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "seaLevelHelp"},
		{VariableType: "uint32", VariableName: "seaLevelDisabledHelp"},
		{VariableType: "uint32", VariableName: "seaLevelStrategy"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "seaLevelType"},
		{VariableType: "varstring", VariableName: "seaLevelTextKey"},
		{VariableType: "varstring", VariableName: "seaLevelDisplayName2"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "seaLevelChange"},
		{VariableType: "uint8", VariableName: "seaLevelDummyValue2"},
	})
}

func readTurnSpeedData(streamReader *io.SectionReader) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "turnTimerId"},
		{VariableType: "uint32", VariableName: "turnTimerCivilopedia"},
		{VariableType: "varstring", VariableName: "turnTimerDisplayName"},
		{VariableType: "uint32", VariableName: "turnTimerHelp"},
		{VariableType: "uint32", VariableName: "turnTimerDisabledHelp"},
		{VariableType: "uint32", VariableName: "turnTimerStrategy"},
		{VariableType: "varstring", VariableName: "turnTimerType"},
		{VariableType: "varstring", VariableName: "turnTimerTextKey"},
		{VariableType: "varstring", VariableName: "turnTimerDisplayName2"},
		{VariableType: "uint32", VariableName: "turnTimerBaseTime"},
		{VariableType: "uint32", VariableName: "turnTimerCityBonus"},
		{VariableType: "uint32", VariableName: "turnTimerUnitBonus"},
		{VariableType: "uint32", VariableName: "turnTimerFirstTurnMultiplayer"},
		{VariableType: "uint32", VariableName: "turnTimerTypeEnum"},
		{VariableType: "uint8", VariableName: "turnTimerCityScreenBlocked"},
	})

	readArray(streamReader, "turnTimerVictoryFlags", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint8", VariableName: "victoryFlag"},
	})
}

func readWorldSizeData(streamReader *io.SectionReader) {
	worldInfoVersion := unsafeReadUint32(streamReader)
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "worldInfoId"},
	})

	if worldInfoVersion == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{VariableType: "uint32", VariableName: "worldInfoCivilopedia"},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "worldSizeDisplayName"},
		{VariableType: "varstring", VariableName: "worldSizeHelp"},
		{VariableType: "uint32", VariableName: "worldSizeDisabledHelp"},
		{VariableType: "uint32", VariableName: "worldSizeStrategy"},
		{VariableType: "varstring", VariableName: "worldSizeType"},
		{VariableType: "varstring", VariableName: "worldSizeTextKey"},
		{VariableType: "varstring", VariableName: "worldSizeDisplayName2"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "defaultPlayers"},
		{VariableType: "uint32", VariableName: "defaultMinorCivs"},
		{VariableType: "uint32", VariableName: "fogTilesPerBarbarianCamp"},
		{VariableType: "uint32", VariableName: "numNaturalWonders"},
		{VariableType: "uint32", VariableName: "unitNameModifier"},
		{VariableType: "uint32", VariableName: "targetNumCities"},
		{VariableType: "uint32", VariableName: "numFreeBuildingResources"},
		{VariableType: "uint32", VariableName: "buildingClassPrereqModifier"},
		{VariableType: "int32", VariableName: "maxConscriptModifier"},
		{VariableType: "uint32", VariableName: "gridWidth"},
		{VariableType: "uint32", VariableName: "gridHeight"},
	})

	if worldInfoVersion == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{VariableType: "uint32", VariableName: "maxActiveReligions"},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "terrainGrainChange"},
		{VariableType: "int32", VariableName: "featureGrainChange"},
		{VariableType: "uint32", VariableName: "researchPercent"},
		{VariableType: "uint32", VariableName: "advancedStartPointsMod"},
		{VariableType: "uint32", VariableName: "numCitiesUnhappinessPercent"},
		{VariableType: "uint32", VariableName: "numCitiesPolicyCostMod"},
		{VariableType: "uint32", VariableName: "numCitiesTechCostMod"},
	})

	if worldInfoVersion == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{VariableType: "uint32", VariableName: "worldSizeEnum"},
		})
	}
}

func readGameOptions(streamReader *io.SectionReader) {
	readArray(streamReader, "gameOptionArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "gameOption"},
		{VariableType: "uint32", VariableName: "gameOptionEnabled"},
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
		{VariableType: fmt.Sprintf("bytearray:%d", (marker+1)*4), VariableName: blockName},
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
		{VariableType: "bytearray:4", VariableName: "gameName"},
		{VariableType: "bytearray:4", VariableName: "unknownBlock1"},
		{VariableType: "varstring", VariableName: "gameVersion"},
		{VariableType: "varstring", VariableName: "gameBuild"},
		{VariableType: "uint32", VariableName: "currentTurnNumber"},
		{VariableType: "bytearray:1", VariableName: "unknownBlock2"},
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
		{VariableType: "varstring", VariableName: "difficulty"},
		{VariableType: "varstring", VariableName: "eraStart"},
		{VariableType: "varstring", VariableName: "eraEnd"},
		{VariableType: "varstring", VariableName: "gameSpeed"},
		{VariableType: "varstring", VariableName: "worldSize"},
		{VariableType: "varstring", VariableName: "mapFilename1"},
	})

	readArray(streamReader, "dlc", []Civ5ReplayFileConfigEntry{
		{VariableType: "bytearray:16", VariableName: "dlcId"},
		{VariableType: "bytearray:4", VariableName: "dlcEnabled"},
		{VariableType: "varstring", VariableName: "dlcName"},
	})

	readArray(streamReader, "mods", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "modId"},
		{VariableType: "bytearray:4", VariableName: "modVersion"},
		{VariableType: "varstring", VariableName: "modName"},
	})
}

func readPlayerAndMapInfo(streamReader *io.SectionReader) uint32 {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "playerCivName"},
		{VariableType: "varstring", VariableName: "playerLeaderName"},
		{VariableType: "varstring", VariableName: "playerColor"},
		{VariableType: "bytearray:16", VariableName: "unknownId1"},
		{VariableType: "varstring", VariableName: "version"},
		{VariableType: "bytearray:16", VariableName: "unknownId2"},
		{VariableType: "uint32", VariableName: "unknownId3-1"},
	})

	slotHintsVersion := unsafeReadUint32(streamReader)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "unknownId3-3"},
		{VariableType: "uint32", VariableName: "unknownId3-4"},
		{VariableType: "varstring", VariableName: "mapFilename2"},
	})

	readArray(streamReader, "civilizationIndexArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "civilizationIndex"},
	})
	readArray(streamReader, "nicknameArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "nickname"},
	})
	readArray(streamReader, "slotStatusArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "slotStatus"},
	})
	readArray(streamReader, "slotClaimArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "slotClaim"},
	})
	readArray(streamReader, "teamTypeArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "teamType"},
	})
	readArray(streamReader, "handicapArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "handicap"},
	})

	return slotHintsVersion
}

func readCivRoster(streamReader *io.SectionReader, slotHintsVersion uint32) []Civ5ReplayCiv {
	fmt.Println("slotHintsVersion:", slotHintsVersion)
	if slotHintsVersion < 3 {
		return make([]Civ5ReplayCiv, MaxPlayers)
	}

	civilizationKeyArrLength := unsafeReadUint32(streamReader)
	fmt.Println("CivilizationKeyArrLength:", civilizationKeyArrLength)
	civilizationKeyArr := readVarStringArrayOrPanic(streamReader, civilizationKeyArrLength, "civilizationKey", "civilization key")
	fmt.Println("CivilizationKeys:", civilizationKeyArr)

	allCivs := make([]Civ5ReplayCiv, 0, len(civilizationKeyArr))
	for _, civilizationKey := range civilizationKeyArr {
		allCivs = append(allCivs, Civ5ReplayCiv{
			Name: civilizationKey,
		})
	}
	return allCivs
}

func readLeadersAndCivArrays(streamReader *io.SectionReader, slotHintsVersion uint32) uint32 {
	if slotHintsVersion >= 3 {
		readArray(streamReader, "leaderKeyArr", []Civ5ReplayFileConfigEntry{
			{VariableType: "varstring", VariableName: "leaderKey"},
		})
	}

	preGameFormatMarker := unsafeReadUint32(streamReader)
	fmt.Println("preGameFormatMarker:", preGameFormatMarker)
	if preGameFormatMarker != 0 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{VariableType: "bytearray:12", VariableName: "unknownBlock5"},
		})
	}

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "computerUsername1"},
	})

	readArray(streamReader, "unknownBlock6-1", []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "unknownBlock6-1"},
	})
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "bytearray:53", VariableName: "unknownBlock6-2"},
	})

	readArray(streamReader, "unknownArray1", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "unknownArray1Var"},
	})

	readArray(streamReader, "civArray1", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "civName"},
	})

	readArray(streamReader, "unknownArray2", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "unknownArray2Var"},
	})

	readArray(streamReader, "civArray2", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "civArray2String"},
	})

	if slotHintsVersion < 3 {
		readArray(streamReader, "extraArrBeforeClimate", []Civ5ReplayFileConfigEntry{
			{VariableType: "uint32", VariableName: "extraArrBeforeClimateVar"},
		})
	}

	return preGameFormatMarker
}

func readGameNameAndTurnInfo(streamReader *io.SectionReader, preGameFormatMarker uint32) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "eraEnum"},
	})

	readArray(streamReader, "emailAddressArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "emailAddress",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "float32", VariableName: "endTurnTimerLength"},
	})

	readArray(streamReader, "flagDecalArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "flagDecal",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "deprecatedForceControlsCount"},
		{VariableType: "bytearray:7", VariableName: "deprecatedForceControlsFlags"},
		{VariableType: "int32", VariableName: "gameModeEnum"},
	})

	gameName, err := readVarString(streamReader, "gameName")
	if err != nil {
		panic(fmt.Sprintf("failed to read game name: %v", err))
	}
	fmt.Println("Game name:", gameName)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "gameSpeedEnum",
		},
		{
			VariableType: "uint8",
			VariableName: "gameStarted",
		},
		{
			VariableType: "uint32",
			VariableName: "currentTurnNumber",
		},
		{
			VariableType: "int32",
			VariableName: "legacyGameTypeDummy", // uiVersion==0 branch - unused
		},
		{
			VariableType: "uint8",
			VariableName: "networkMultiplayerGame",
		},
		{
			VariableType: "uint32",
			VariableName: "gameUpdateTime",
		},
	})

	readArray(streamReader, "handicapArr2", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "handicap"},
	})

	if preGameFormatMarker != 4 {
		readArray(streamReader, "trackedPlayerHandicapArr", []Civ5ReplayFileConfigEntry{
			{VariableType: "int32", VariableName: "trackedPlayerHandicap"},
		})
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{VariableType: "bytearray:2", VariableName: "unknownBlock10"},
		})
	} else {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{VariableType: "bytearray:2", VariableName: "reservedBytesWhenHandicapArrAbsent"},
		})
	}
}

func readLeaderArray2AndPlayerSetup(streamReader *io.SectionReader) {
	readArray(streamReader, "leaderNameArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "leaderName"},
	})

	unknownBlock11Number := unsafeReadUint32(streamReader)
	readDynamicPaddingBlock(streamReader, unknownBlock11Number, "unknownBlock11")

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "computerUsername2"},
		{VariableType: "bytearray:7", VariableName: "unknownId4"},
		{VariableType: "varstring", VariableName: "mapFilename3"},
		{VariableType: "uint32", VariableName: "maxCityElimination"},
		{VariableType: "uint32", VariableName: "maxTurns"},
		{VariableType: "uint32", VariableName: "numMinorCivs"},
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
	fmt.Println("minorCivArray:", minorCivNameArr) // read via a DB lookup (MinorCivilizations table) - variable count, not MAX_PLAYERS

	readArray(streamReader, "minorNationCivsArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "minorNationCiv",
		},
	})
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "dummyValue",
		},
	})
	readArray(streamReader, "multiplayerOptionsArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "multiplayerOption",
		},
	})
}

// readPlayerArraysAndColors reads several player-related arrays and patches the civ roster with player colors
func readPlayerArraysAndColors(streamReader *io.SectionReader, allCivs []Civ5ReplayCiv) {
	readArray(streamReader, "netIdArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "int32",
			VariableName: "netId", // -1 = no network ID, expected for non-multiplayer saves
		},
	})

	readArray(streamReader, "nicknameArr2", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "nickname2", // a second, independent nickname array (see nicknameArr in readPlayerAndMapInfo)
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "int32",
			VariableName: "numVictoryInfos", // the ruleset's own victory type count
		},
		{
			VariableType: "int32",
			VariableName: "pitBossTurnTime",
		},
	})

	readArray(streamReader, "playableCivsArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "playableCiv",
		},
	})

	playerColorLength := unsafeReadUint32(streamReader)
	playerColorArr := readVarStringArrayOrPanic(streamReader, playerColorLength, "playerColorName", "player color name")
	for i, playerColorName := range playerColorArr {
		allCivs[i].LongName = playerColorName
	}
	fmt.Println("playerColorArr:", playerColorArr) // read via a DB lookup (PlayerColors table) - variable count, not MAX_PLAYERS

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "privateGame",
		},
		{
			VariableType: "uint8",
			VariableName: "quickCombat",
		},
		{
			VariableType: "uint8",
			VariableName: "quickCombatDefault",
		},
		{
			VariableType: "int32",
			VariableName: "quickHandicap",
		},
		{
			VariableType: "uint8",
			VariableName: "quickstart",
		},
		{
			VariableType: "uint8",
			VariableName: "randomWorldSize",
		},
		{
			VariableType: "uint8",
			VariableName: "randomMapScript",
		},
	})

	readArray(streamReader, "readyPlayersArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint8", VariableName: "readyPlayer"},
	})
}

func readWorldSettings(streamReader *io.SectionReader) {
	readArray(streamReader, "slotClaimArr2", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "slotClaim"},
	})

	readArray(streamReader, "slotStatusArr2", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "slotStatus"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "smtpHost"},
		{VariableType: "uint32", VariableName: "syncRandomSeed"},
		{VariableType: "int32", VariableName: "targetScore"},
	})

	readArray(streamReader, "teamTypeArr2", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "teamType"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint8", VariableName: "transferredMap"},
	})

	readTurnSpeedData(streamReader)
	readArray(streamReader, "whiteFlagArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint8", VariableName: "whiteFlag"},
	})
	readWorldSizeData(streamReader)
	readGameOptions(streamReader)

	readArray(streamReader, "mapOptionArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "mapOptionName"},
		{VariableType: "uint32", VariableName: "mapOptionValue"},
	})
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "gameVersion2"},
	})

	readArray(streamReader, "shouldNotifySteamInviteArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint8", VariableName: "shouldNotifySteamInvite"},
	})

	readArray(streamReader, "shouldNotifyEmailArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "uint8", VariableName: "shouldNotifyEmail"},
	})

	readArray(streamReader, "turnNotifyEmailAddressArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "turnNotifyEmailAddress"},
	})
}

// locateCompressedBlock skips the padding before the compressed block and returns a reader
// positioned at its start
func locateCompressedBlock(streamReader *io.SectionReader, inputFile *os.File, saveFileLength int64) (*io.SectionReader, error) {
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "bytearray:8", VariableName: "paddingBeforeCompressedBlock"}, // value is always [2 0 0 0 0 0 1 0]
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
	slotHintsVersion := readPlayerAndMapInfo(streamReader)

	allCivs := readCivRoster(streamReader, slotHintsVersion)

	preGameFormatMarker := readLeadersAndCivArrays(streamReader, slotHintsVersion)
	readClimateSection(streamReader)
	readGameNameAndTurnInfo(streamReader, preGameFormatMarker)
	readLeaderArray2AndPlayerSetup(streamReader)
	readMinorCivNames(streamReader, allCivs)
	readPlayerArraysAndColors(streamReader, allCivs)
	readSeaLevel(streamReader)
	readWorldSettings(streamReader)

	compressedStreamReader, err := locateCompressedBlock(streamReader, inputFile, saveFileLength)
	if err != nil {
		return nil, err
	}

	decompressedStreamReader, decompressedContentsSize, err := buildReaderForDecompressedFile(compressedStreamReader, outputFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress file: %w", err)
	}
	allReplayEvents := readDecompressed(decompressedStreamReader, decompressedContentsSize, allCivs)

	return &Civ5SaveData{
		PlayerCiv:       playerCiv,
		IsReplayFile:    false,
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
	}, nil
}

func readDecompressedHeader(streamReader *io.SectionReader) uint32 {
	saveFileVersion := unsafeReadUint32(streamReader)
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "endTurnMessagesSent"},
		{VariableType: "uint32", VariableName: "elapsedGameTurns"},
		{VariableType: "uint32", VariableName: "startTurn"},
		{VariableType: "uint32", VariableName: "winningTurn"},
		{VariableType: "int32", VariableName: "startYear"},
		{VariableType: "int32", VariableName: "estimateEndTurn"},
		{VariableType: "int32", VariableName: "defaultEstimateEndTurn"},
		{VariableType: "int32", VariableName: "turnSlice"},
		{VariableType: "int32", VariableName: "cutoffSlice"},
		{VariableType: "int32", VariableName: "numCities"},
		{VariableType: "int32", VariableName: "totalPopulation"},
		{VariableType: "int32", VariableName: "noNukesCount"},
		{VariableType: "int32", VariableName: "nukesExploded"},
		{VariableType: "int32", VariableName: "maxPopulation"},
		{VariableType: "int32", VariableName: "unused1"},
		{VariableType: "int32", VariableName: "unused2"},
		{VariableType: "int32", VariableName: "unused3"},
		{VariableType: "int32", VariableName: "initPopulation"},
		{VariableType: "int32", VariableName: "initLand"},
		{VariableType: "int32", VariableName: "initTech"},
		{VariableType: "int32", VariableName: "initWonders"},
		{VariableType: "int32", VariableName: "aiAutoPlay"},
		{VariableType: "int32", VariableName: "totalReligionTechCost"},
		{VariableType: "int32", VariableName: "cachedWorldReligionTechProgress"},
		{VariableType: "int32", VariableName: "unitedNationsCountdown"},
		{VariableType: "int32", VariableName: "numVictoryVotesTallied"},
		{VariableType: "int32", VariableName: "numVictoryVotesExpected"},
		{VariableType: "int32", VariableName: "votesNeededForDiploVictory"},
		{VariableType: "int32", VariableName: "mapScoreMod"},
		{VariableType: "bytearray:1", VariableName: "scoreDirty"},
		{VariableType: "bytearray:1", VariableName: "circumnavigated"},
		{VariableType: "bytearray:1", VariableName: "finalInitialized"},
		{VariableType: "bytearray:1", VariableName: "hotPbemBetweenTurns"},
		{VariableType: "bytearray:1", VariableName: "nukesValid"},
		{VariableType: "bytearray:1", VariableName: "endGameTechResearched"},
		{VariableType: "bytearray:1", VariableName: "tunerEverConnected"},
		{VariableType: "bytearray:1", VariableName: "tutorialEverAttacked"},
		{VariableType: "bytearray:1", VariableName: "staticTutorialActive"},
		{VariableType: "bytearray:1", VariableName: "everRightClickMoved"},
	})

	readArray(streamReader, "advisorMessagesViewed", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "advisorMessageId"},
	})

	handicap := unsafeReadUint32(streamReader)
	fmt.Println("Handicap:", typeName(handicapNames, int(handicap)))
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "pausePlayer"},
		{VariableType: "int32", VariableName: "aiAutoPlayReturnPlayer"},
		{VariableType: "int32", VariableName: "bestLandUnit"},
		{VariableType: "int32", VariableName: "winner"},
	})
	victory := unsafeReadUint32(streamReader)
	fmt.Println("Victory:", typeName(victoryTypeNames, int(victory)))
	gameState := unsafeReadUint32(streamReader)
	fmt.Println("GameState:", typeName(gameStateNames, int(gameState)))
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "bestWondersPlayer"},
		{VariableType: "int32", VariableName: "bestPoliciesPlayer"},
		{VariableType: "int32", VariableName: "bestGreatPeoplePlayer"},
		{VariableType: "int32", VariableName: "religionTech"},
		{VariableType: "int32", VariableName: "industrialRoute"},
		{VariableType: "varstring", VariableName: "scriptData"},
	})

	endTurnMessagesReceived := unsafeReadFixedInt32Array(streamReader, 64)
	rankPlayer := unsafeReadFixedInt32Array(streamReader, 64)
	playerRank := unsafeReadFixedInt32Array(streamReader, 64)
	playerScore := unsafeReadFixedInt32Array(streamReader, 64)
	rankTeam := unsafeReadFixedInt32Array(streamReader, 64)
	teamRank := unsafeReadFixedInt32Array(streamReader, 64)
	teamScore := unsafeReadFixedInt32Array(streamReader, 64)
	fmt.Println("End turn messages received:", endTurnMessagesReceived)
	fmt.Println("Rank -> player:", rankPlayer, " player -> rank:", playerRank)
	fmt.Println("Player score:", playerScore)
	fmt.Println("Rank -> team:", rankTeam, " team -> rank:", teamRank)
	fmt.Println("Team score:", teamScore)

	return saveFileVersion
}

func readCreatedCountHashArrays(streamReader *io.SectionReader) {
	unitCreatedCountLen := unsafeReadUint32(streamReader)
	unitCreatedCount := readHashValuePairs(streamReader, int(unitCreatedCountLen))

	unitClassCreatedCountLen := unsafeReadUint32(streamReader)
	unitClassCreatedCount := readHashValuePairs(streamReader, int(unitClassCreatedCountLen))

	buildingClassCreatedCountLen := unsafeReadUint32(streamReader)
	buildingClassCreatedCount := readHashValuePairs(streamReader, int(buildingClassCreatedCountLen))

	fmt.Println("Unit created count entries:", len(unitCreatedCount))
	fmt.Println("Unit class created count entries:", len(unitClassCreatedCount))
	fmt.Println("Building class created count entries:", len(buildingClassCreatedCount))
}

func readWorldCongressVotingState(streamReader *io.SectionReader) {
	projectCreatedCountLen := unsafeReadUint32(streamReader)
	projectCreatedCount := readHashValuePairs(streamReader, int(projectCreatedCountLen))

	voteOutcomeLen := unsafeReadUint32(streamReader)
	voteOutcome := readHashValuePairs(streamReader, int(voteOutcomeLen))

	secretaryGeneralTimerLen := unsafeReadUint32(streamReader)
	secretaryGeneralTimer := readHashValuePairs(streamReader, int(secretaryGeneralTimerLen))

	voteTimerLen := unsafeReadUint32(streamReader)
	voteTimer := readHashValuePairs(streamReader, int(voteTimerLen))

	diploVoteLen := unsafeReadUint32(streamReader)
	diploVote := readHashValuePairs(streamReader, int(diploVoteLen))

	fmt.Println("Project created count:", projectCreatedCount)
	fmt.Println("Vote outcome:", voteOutcome)
	fmt.Println("Secretary general timer:", secretaryGeneralTimer)
	fmt.Println("Vote timer:", voteTimer)
	fmt.Println("Diplo vote:", diploVote)

	votesCast := unsafeReadFixedInt32Array(streamReader, 63)
	previousVotesCast := unsafeReadFixedInt32Array(streamReader, 63)
	numVotesForTeam := unsafeReadFixedInt32Array(streamReader, 63)
	fmt.Println("Votes cast:", votesCast)
	fmt.Println("Previous votes cast:", previousVotesCast)
	fmt.Println("Num votes for team:", numVotesForTeam)

	specialUnitValidLen := unsafeReadUint32(streamReader)
	specialUnitValid := readHashBoolPairs(streamReader, int(specialUnitValidLen))
	fmt.Println("Special unit valid:", specialUnitValid)

	teamVictoryRankLen := unsafeReadUint32(streamReader)
	teamVictoryRank := readHashIntArrayPairs(streamReader, int(teamVictoryRankLen), NumVictoryPointAwards)
	fmt.Println("Team victory rank:", teamVictoryRank)
}

func readVoteSelectionAndTriggeredArrays(streamReader *io.SectionReader) {
	readEmptyFreeListTrashArray(streamReader, "voteSelections")
	readEmptyFreeListTrashArray(streamReader, "votesTriggered")
}

func readRandomAndReplayMessageVersion(streamReader *io.SectionReader) {
	mapRand := readRandomState(streamReader)
	otherRand := readRandomState(streamReader)
	replayMessageVersion := unsafeReadUint32(streamReader)
	fmt.Println("Map rand:", mapRand, "Other rand:", otherRand, "Replay message version:", replayMessageVersion)
}

func readVersionDependentUnitData(streamReader *io.SectionReader, saveFileVersion uint32) {
	if saveFileVersion == SaveVersionWithUnitClassData {
		readArray(streamReader, "unitNameArr", []Civ5ReplayFileConfigEntry{
			{VariableType: "varstring", VariableName: "unitName"},
			{VariableType: "uint32", VariableName: "unitCreatedCount"},
		})
		readArray(streamReader, "unitClassArr", []Civ5ReplayFileConfigEntry{
			{VariableType: "varstring", VariableName: "unitClass"},
			{VariableType: "uint32", VariableName: "unitClassCreatedCount"},
		})
		readArray(streamReader, "buildingClassArr", []Civ5ReplayFileConfigEntry{
			{VariableType: "varstring", VariableName: "buildingClass"},
			{VariableType: "uint32", VariableName: "buildingClassCreatedCount"},
		})

		// TODO: find padding
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{VariableType: "bytearray:2366", VariableName: "unknownPadding"}, // for RED WW2 save files, likely different for other mods
		})
	} else {
		readCreatedCountHashArrays(streamReader)
		readWorldCongressVotingState(streamReader)
		readArray(streamReader, "destroyedCitiesArr", []Civ5ReplayFileConfigEntry{
			{VariableType: "varstring", VariableName: "destroyedCityName"},
		})
	}
}

func readDecompressed(reader *bytes.Reader, decompressedFileLength int, allCivs []Civ5ReplayCiv) []Civ5ReplayEvent {
	streamReader := io.NewSectionReader(reader, int64(0), int64(decompressedFileLength))

	saveFileVersion := readDecompressedHeader(streamReader)
	readVersionDependentUnitData(streamReader, saveFileVersion)
	readArray(streamReader, "greatPersonArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "greatPersonName"},
	})
	readVoteSelectionAndTriggeredArrays(streamReader)
	readRandomAndReplayMessageVersion(streamReader)

	allReplayEvents := readEvents(streamReader)
	printReplayEvents(allReplayEvents)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "numSessions"},
	})

	plotExtraYields := readPlotExtraYields(streamReader)
	fmt.Println("plotExtraYields:", plotExtraYields)

	readArray(streamReader, "plotExtraCostArr", []Civ5ReplayFileConfigEntry{
		{VariableType: "int32", VariableName: "plotExtraCostX"},
		{VariableType: "int32", VariableName: "plotExtraCostY"},
		{VariableType: "int32", VariableName: "plotExtraCost"},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{VariableType: "uint8", VariableName: "archaeologyTriggered"},
		{VariableType: "int32", VariableName: "earliestBarbarianReleaseTurn"},
	})

	gameDeals := readGameDeals(streamReader)
	printGameDeals(allCivs, gameDeals)

	gameReligions := readGameReligions(streamReader)
	printGameReligions(gameReligions)

	gameCulture := readGameCulture(streamReader)
	printGameCulture(allCivs, gameCulture)

	gameLeagues := readGameLeagues(streamReader)
	printGameLeagues(gameLeagues)

	gameTrade := readGameTrade(streamReader)
	printGameTrade(allCivs, allReplayEvents, gameTrade)

	embeddedDatabase := readEmbeddedDatabase(streamReader)
	fmt.Printf("embeddedDatabase: %d bytes, magic=%q\n", len(embeddedDatabase), string(embeddedDatabase[:min(16, len(embeddedDatabase))]))

	mapHeader := readMapHeader(streamReader)
	fmt.Printf("mapHeader: version=%d grid=%dx%d landPlots=%d ownedPlots=%d naturalWonders=%d latitude=%d/%d wrap=%v/%v resourceTypes=%d/%d\n",
		mapHeader.Version, mapHeader.GridWidth, mapHeader.GridHeight, mapHeader.LandPlotCount, mapHeader.OwnedPlotCount, mapHeader.NumNaturalWonders,
		mapHeader.TopLatitude, mapHeader.BottomLatitude, mapHeader.WrapX, mapHeader.WrapY, len(mapHeader.ResourceCounts), len(mapHeader.ResourceCountsOnLand))

	return allReplayEvents
}
