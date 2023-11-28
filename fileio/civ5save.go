package fileio

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type Civ5SaveData struct {
	AllReplayEvents []Civ5ReplayEvent
}

func ReadCiv5SaveFile(filename string, outputFilename string) (*Civ5SaveData, error) {
	inputFile, err := os.Open(filename)
	defer inputFile.Close()

	fi, err := inputFile.Stat()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	fileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), fileLength)
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
			VariableType: "bytearray:16",
			VariableName: "unknownId3",
		},
		{
			VariableType: "varstring",
			VariableName: "mapFilename2",
		},
	})

	readArray(streamReader, "unknownBlock3", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
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
		readArray(streamReader, fmt.Sprintf("unknownArrayBlock%d", i), []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
				VariableName: fmt.Sprintf("unknownArrayBlock%d", i),
			},
		})
	}

	readArray(streamReader, "civNames", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "arrCivName",
		},
	})

	readArray(streamReader, "leaderNames", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "arrLeaderName",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:16",
			VariableName: "unknownBlock5",
		},
		{
			VariableType: "varstring",
			VariableName: "computerUsername1",
		},
		{
			VariableType: "bytearray:313",
			VariableName: "unknownBlock6",
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

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:12",
			VariableName: "unknownBlock7",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "climateName1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:12",
			VariableName: "unknownBlock8",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "climateName2",
		},
		{
			VariableType: "varstring",
			VariableName: "climateName3",
		},
		{
			VariableType: "varstring",
			VariableName: "climateName4",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:587",
			VariableName: "unknownBlock9",
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
			VariableName: "unknownBlock10",
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
				VariableType: "uint32",
				VariableName: "unknownArray4Var",
			},
		})
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:2",
				VariableName: "unknownBlock11",
			},
		})
	}

	readArray(streamReader, "leaderArray", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "leaderArrName",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:4",
			VariableName: "unknownBlock12",
		},
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
			VariableType: "bytearray:12",
			VariableName: "unknownBlock13",
		},
	})

	readArray(streamReader, "minorCivArray", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "minorCivArrName",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:77",
			VariableName: "unknownBlock14",
		},
	})

	readArray(streamReader, "unknownArray5", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
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
			VariableName: "unknownBlock15",
		},
	})

	readArray(streamReader, "unknownArray6", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "unknownArray6Var",
		},
	})

	readArray(streamReader, "playerColorArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "playerColorArrName",
		},
	})

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
			VariableName: "seaLevelName2",
		},
		{
			VariableType: "varstring",
			VariableName: "seaLevelName3",
		},
		{
			VariableType: "varstring",
			VariableName: "seaLevelName4",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:5",
			VariableName: "paddingAfterSeaLevel2",
		},
	})

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
			VariableType: "bytearray:9",
			VariableName: "unknownBlock18",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "turnTimeName1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:12",
			VariableName: "paddingAfterTurnTime1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "turnTimeName2",
		},
		{
			VariableType: "varstring",
			VariableName: "turnTimeName3",
		},
		{
			VariableType: "varstring",
			VariableName: "turnTimeName4",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
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
			VariableName: "turnTimerUnknown1",
		},
		{
			VariableType: "uint8",
			VariableName: "turnTimerUnknown2",
		},
		{
			VariableType: "varstring", // length is usually 5
			VariableName: "turnTimerUnknown3",
		},
	})

	readArray(streamReader, "unknownArray11", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint8",
			VariableName: "unknownArray11Var",
		},
	})

	numberBeforeWorldSize := unsafeReadUint32(streamReader)
	// Should be related to map version
	if numberBeforeWorldSize == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
				VariableName: "numBeforeWorldSize1",
			},
			{
				VariableType: "uint32",
				VariableName: "numBeforeWorldSize2",
			},
		})
	} else {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
				VariableName: "numBeforeWorldSize1",
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
			VariableName: "worldSize2",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:8",
			VariableName: "paddingAfterWorldSize1",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "worldSize3",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSize4",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSize5",
		},
	})

	if numberBeforeWorldSize == 2 {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:80",
				VariableName: "paddingAfterWorldSize2",
			},
		})
	} else {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "bytearray:72",
				VariableName: "paddingAfterWorldSize2",
			},
		})
	}

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

	_ = unsafeReadUint32(streamReader)

	gameVersion := readVarString(streamReader, "gameVersion")
	fmt.Println("Game version:", gameVersion)

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
			VariableType: "bytearray:8",
			VariableName: "paddingBeforeCompressedBlock",
		},
	})

	offsetToCompressedBlock, err := streamReader.Seek(0, io.SeekCurrent)
	if err != nil {
		log.Fatal("Failed to call fseek", err)
	}
	fmt.Println("Offset to compressed data:", offsetToCompressedBlock)

	compressedStreamReader := io.NewSectionReader(inputFile, int64(offsetToCompressedBlock), fileLength-int64(offsetToCompressedBlock))

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
			log.Fatal("Failed to decompress zlib:", err)
		}

		fmt.Println("Decompressed contents size:", len(decompressedContents))
		err = ioutil.WriteFile(outputFilename, decompressedContents, 0644)
		if err != nil {
			log.Fatal("Error writing to "+outputFilename, err)
		}
	}

	decompressedStreamReader := bytes.NewReader(decompressedContents)
	allReplayEvents := readDecompressed(decompressedStreamReader, len(decompressedContents))

	civ5SaveData := &Civ5SaveData{
		AllReplayEvents: allReplayEvents,
	}
	return civ5SaveData, nil
}

func readDecompressed(reader *bytes.Reader, fileLength int) []Civ5ReplayEvent {
	streamReader := io.NewSectionReader(reader, int64(0), int64(fileLength))
	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "unknown1",
		},
		{
			VariableType: "uint32",
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
			VariableType: "uint32",
			VariableName: "startYear",
		},
	})

	for i := 0; i < 24; i++ {
		readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
			{
				VariableType: "uint32",
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

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:3968", // TODO: replace with calculation, different for each file
			VariableName: "unknownSection4",
		},
	})

	readArray(streamReader, "greatPersonArr", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "greatPersonName",
		},
	})

	readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:150",
			VariableName: "unknownSection8",
		},
	})

	allReplayEvents := readEvents(streamReader)
	fmt.Println(fmt.Sprintf("Read %d replay events", len(allReplayEvents)))

	return allReplayEvents
}
