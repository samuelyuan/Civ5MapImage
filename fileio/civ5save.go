package fileio

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
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

func buildReaderForDecompressedFile(compressedStreamReader *io.SectionReader, outputFilename string) (*bytes.Reader, int) {
	decompressedFileReader, err := zlib.NewReader(compressedStreamReader)
	if err != nil {
		log.Fatal("Failed to create zlib new reader:", err)
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
		err = ioutil.WriteFile(outputFilename, decompressedContents, 0644)
		if err != nil {
			log.Fatal("Error writing to "+outputFilename, err)
		}
	}

	return bytes.NewReader(decompressedContents), len(decompressedContents)
}

func ReadCiv5SaveFile(filename string, outputFilename string) (*Civ5SaveData, error) {
	inputFile, err := os.Open(filename)
	defer inputFile.Close()

	fi, err := inputFile.Stat()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	saveFileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), saveFileLength)
	fmt.Println("Loading Civ5Save...")

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

	playerCiv := readVarString(streamReader, "playerCiv")
	fmt.Println("Player civ:", playerCiv)

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

	civNamesLength := unsafeReadUint32(streamReader)
	fmt.Println("CivNamesLength:", civNamesLength)
	civNameArr := make([]string, civNamesLength)
	for i := 0; i < int(civNamesLength); i++ {
		civName := readVarString(streamReader, "civName")
		civNameArr[i] = civName
	}
	fmt.Println("CivNames:", civNameArr)

	allCivs := make([]Civ5ReplayCiv, 0)
	for i := 0; i < len(civNameArr); i++ {
		civData := Civ5ReplayCiv{
			UnknownVariables: [4]int{0, 0, 0, 0},
			Leader:           "",
			LongName:         "",
			Name:             civNameArr[i],
			Demonym:          "",
		}
		allCivs = append(allCivs, civData)
	}

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

	unknownBlock7Number := unsafeReadUint32(streamReader)
	if unknownBlock7Number != 0 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: fmt.Sprintf("bytearray:%d", (unknownBlock7Number + 1) * 4),
				VariableName: "unknownBlock7-1",
			},
		})
	}
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:8",
			VariableName: "unknownBlock7-2",
		},
	})
	readClimateName(streamReader)

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

	gameName := readVarString(streamReader, "gameName")
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

	readArray(streamReader, "leaderArray2", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "leaderArrName",
		},
	})

	unknownBlock11Number := unsafeReadUint32(streamReader)
	if unknownBlock11Number != 0 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: fmt.Sprintf("bytearray:%d", (unknownBlock11Number + 1) * 4),
				VariableName: "unknownBlock11",
			},
		})
	}

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

	minorCivNamesLength := unsafeReadUint32(streamReader)
	minorCivNameArr := make([]string, minorCivNamesLength)
	for i := 0; i < int(minorCivNamesLength); i++ {
		minorCivName := readVarString(streamReader, "minorCivName")
		minorCivNameArr[i] = minorCivName

		if strings.Contains(minorCivNameArr[i], "MINOR_CIV") {
			allCivs[i].Name = minorCivNameArr[i]
		}
	}
	fmt.Println("minorCivArray:", minorCivNameArr)

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:77",
			VariableName: "unknownBlock13",
		},
	})

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
	playerColorArr := make([]string, playerColorLength)
	for i := 0; i < int(playerColorLength); i++ {
		playerColorName := readVarString(streamReader, "playerColorName")
		playerColorArr[i] = playerColorName

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

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:8", // value is always [2 0 0 0 0 0 1 0]
			VariableName: "paddingBeforeCompressedBlock",
		},
	})

	// Header of compressed block should begin with 0x789C
	offsetToCompressedBlock, err := streamReader.Seek(0, io.SeekCurrent)
	if err != nil {
		log.Fatal("Failed to call fseek", err)
	}
	fmt.Println("Offset to compressed data:", offsetToCompressedBlock)
	compressedStreamReader := io.NewSectionReader(inputFile, int64(offsetToCompressedBlock), saveFileLength-int64(offsetToCompressedBlock))

	decompressedStreamReader, decompressedContentsSize := buildReaderForDecompressedFile(compressedStreamReader, outputFilename)
	allReplayEvents := readDecompressed(decompressedStreamReader, decompressedContentsSize)

	civ5SaveData := &Civ5SaveData{
		PlayerCiv:       playerCiv,
		IsReplayFile:    false,
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
	}
	return civ5SaveData, nil
}

func readDecompressed(reader *bytes.Reader, decompressedFileLength int) []Civ5ReplayEvent {
	streamReader := io.NewSectionReader(reader, int64(0), int64(decompressedFileLength))

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
			VariableName: fmt.Sprintf("unknownSection2"),
		},
	})

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

	if saveFileVersion == 0x0B {
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
	} else {
		// Three unknown arrays
		// The size of the first and the third arrays are the same unless first array size is greater than 150, which means the first array
		// length is one more than the third array length. The second array is much smaller.
		// Each array element is 8 bytes. The first 4 bytes are usually consistent between different save files. The last 4 bytes can vary.

		// Array 1 length: Usually 128 or 132, but some files have other values like [127, 154, 157]
		arrayLength := unsafeReadUint32(streamReader)
		// Can be one less for some save files
		if arrayLength >= 150 {
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

	allReplayEvents := readEvents(streamReader)
	fmt.Println(fmt.Sprintf("Read %d replay events", len(allReplayEvents)))

	return allReplayEvents
}
