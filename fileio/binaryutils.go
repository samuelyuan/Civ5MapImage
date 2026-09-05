package fileio

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"strconv"
	"strings"
)

// Constants for binary reading operations
const (
	MaxArrayLength = 100000 // Maximum reasonable array length to prevent memory issues
)

// readVarString reads a variable-length string from the binary stream
// Format: [length:uint32][string:bytes]
func readVarString(reader *io.SectionReader, varName string) (string, error) {
	variableLength := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &variableLength); err != nil {
		return "", fmt.Errorf("failed to load variable length for %s: %w", varName, err)
	}

	stringValue := make([]byte, variableLength)
	if err := binary.Read(reader, binary.LittleEndian, &stringValue); err != nil {
		return "", fmt.Errorf("failed to load string value for %s (length: %v): %w", varName, variableLength, err)
	}

	return string(stringValue[:]), nil
}

// readArray reads an array of file config entries from the binary stream
func readArray(reader *io.SectionReader, arrayName string, fileConfigEntries []Civ5ReplayFileConfigEntry) error {
	arrayLength := unsafeReadUint32(reader)
	if arrayLength > MaxArrayLength {
		return fmt.Errorf("array length may be too long for %s: %d", arrayName, arrayLength)
	}
	for i := 0; i < int(arrayLength); i++ {
		if _, err := readFileConfig(reader, fileConfigEntries); err != nil {
			return fmt.Errorf("failed to read array element %d for %s: %w", i, arrayName, err)
		}
	}
	return nil
}

// readFileConfig reads a file configuration entry from the binary stream
func readFileConfig(reader *io.SectionReader, fileConfigEntries []Civ5ReplayFileConfigEntry) ([]string, error) {
	pos, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}

	fieldValues := make([]string, 0)

	for i := 0; i < len(fileConfigEntries); i++ {
		fileConfigEntry := fileConfigEntries[i]
		if fileConfigEntry.VariableType == "varstring" {
			value, err := readVarString(reader, "varstring_"+fileConfigEntry.VariableName)
			if err != nil {
				return nil, err
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(str):%v", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "float32" {
			value := float32(0)
			if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
				return nil, fmt.Errorf("failed to load float32 for %s: %w", fileConfigEntry.VariableName, err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(f32):%f", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "uint32" {
			value := unsafeReadUint32(reader)
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u32):%d", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "int32" {
			signedIntValue := int32(0)
			if err := binary.Read(reader, binary.LittleEndian, &signedIntValue); err != nil {
				return nil, fmt.Errorf("failed to load int32 for %s: %w", fileConfigEntry.VariableName, err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(i32):%d", fileConfigEntry.VariableName, signedIntValue))
		} else if fileConfigEntry.VariableType == "uint16" {
			value := unsafeReadUint16(reader)
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u16):%d", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "uint8" {
			unsignedIntValue := uint8(0)
			if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
				return nil, fmt.Errorf("failed to load uint8 for %s: %w", fileConfigEntry.VariableName, err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u8):%d", fileConfigEntry.VariableName, unsignedIntValue))
		} else if strings.Contains(fileConfigEntry.VariableType, "bytearray") {
			byteArrayLength, err := strconv.Atoi(fileConfigEntry.VariableType[len("bytearray:"):])
			if err != nil {
				return nil, fmt.Errorf("invalid byte array type in file config for %s: %w", fileConfigEntry.VariableName, err)
			}

			byteBlock := make([]byte, byteArrayLength)
			if err := binary.Read(reader, binary.LittleEndian, &byteBlock); err != nil {
				return nil, fmt.Errorf("invalid byte array data for %s: %w", fileConfigEntry.VariableName, err)
			}

			fieldValues = append(fieldValues, fmt.Sprintf("%v(bytearray):%v", fileConfigEntry.VariableName, byteBlock))
		} else {
			fmt.Println("Unknown variable type:", fileConfigEntry.VariableType)
		}
	}

	fmt.Printf("File Pos: 0x%X, ", pos)
	fmt.Println("Field values:", fieldValues)
	return fieldValues, nil
}

// unsafeReadUint32 reads a uint32 from the binary stream, panicking on error
// This is used for internal operations where errors should not occur
func unsafeReadUint32(reader *io.SectionReader) uint32 {
	unsignedIntValue := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		panic(fmt.Sprintf("failed to load uint32: %v", err))
	}
	return unsignedIntValue
}

// unsafeReadUint16 reads a uint16 from the binary stream, panicking on error
// This is used for internal operations where errors should not occur
func unsafeReadUint16(reader *io.SectionReader) uint16 {
	unsignedIntValue := uint16(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		panic(fmt.Sprintf("failed to load uint16: %v", err))
	}
	return unsignedIntValue
}

// unsafeReadFixedInt32Array reads a fixed-length array of int32 values with no length prefix
func unsafeReadFixedInt32Array(reader *io.SectionReader, count int) []int32 {
	values := make([]int32, count)
	for i := 0; i < count; i++ {
		values[i] = int32(unsafeReadUint32(reader))
	}
	return values
}

// unsafeReadByte reads a single byte from the binary stream, panicking on error.
func unsafeReadByte(reader *io.SectionReader) byte {
	value := make([]byte, 1)
	if _, err := io.ReadFull(reader, value); err != nil {
		panic(fmt.Sprintf("failed to load byte: %v", err))
	}
	return value[0]
}

type HashValuePair struct {
	Hash  uint32
	Value int32
}

func readHashValuePairs(reader *io.SectionReader, count int) []HashValuePair {
	pairs := make([]HashValuePair, count)
	for i := 0; i < count; i++ {
		hash := unsafeReadUint32(reader)
		var value int32
		if hash != 0 {
			value = int32(unsafeReadUint32(reader))
		}
		pairs[i] = HashValuePair{Hash: hash, Value: value}
	}
	return pairs
}

type HashBoolPair struct {
	Hash  uint32
	Value bool
}

func readHashBoolPairs(reader *io.SectionReader, count int) []HashBoolPair {
	pairs := make([]HashBoolPair, count)
	for i := 0; i < count; i++ {
		hash := unsafeReadUint32(reader)
		var value bool
		if hash != 0 {
			value = unsafeReadByte(reader) != 0
		}
		pairs[i] = HashBoolPair{Hash: hash, Value: value}
	}
	return pairs
}

type HashIntArrayPair struct {
	Hash   uint32
	Values []int32
}

func readHashIntArrayPairs(reader *io.SectionReader, count int, subArraySize int) []HashIntArrayPair {
	pairs := make([]HashIntArrayPair, count)
	for i := 0; i < count; i++ {
		hash := unsafeReadUint32(reader)
		var values []int32
		if hash != 0 {
			values = unsafeReadFixedInt32Array(reader, subArraySize)
		}
		pairs[i] = HashIntArrayPair{Hash: hash, Values: values}
	}
	return pairs
}

type FreeListArrayHeader struct {
	NumSlots      int32
	LastIndex     int32
	FreeListHead  int32
	FreeListCount int32
	CurrentID     int32
	NextFreeIndex []int32
}

func readFreeListArrayHeader(reader *io.SectionReader) FreeListArrayHeader {
	numSlots := int32(unsafeReadUint32(reader))
	lastIndex := int32(unsafeReadUint32(reader))
	freeListHead := int32(unsafeReadUint32(reader))
	freeListCount := int32(unsafeReadUint32(reader))
	currentID := int32(unsafeReadUint32(reader))
	nextFreeIndex := unsafeReadFixedInt32Array(reader, int(numSlots))
	return FreeListArrayHeader{
		NumSlots:      numSlots,
		LastIndex:     lastIndex,
		FreeListHead:  freeListHead,
		FreeListCount: freeListCount,
		CurrentID:     currentID,
		NextFreeIndex: nextFreeIndex,
	}
}

func readEmptyFreeListTrashArray(reader *io.SectionReader, name string) FreeListArrayHeader {
	header := readFreeListArrayHeader(reader)
	count := unsafeReadUint32(reader)
	if count != 0 {
		panic(fmt.Sprintf("%s: non-empty FreeListTrashArray (count=%d) not supported", name, count))
	}
	return header
}

type RandomState struct {
	Version    uint32
	Seed       uint32
	CallCount  uint32
	ResetCount uint32
}

func readRandomState(reader *io.SectionReader) RandomState {
	version := unsafeReadUint32(reader)
	seed := unsafeReadUint32(reader)
	callCount := unsafeReadUint32(reader)
	resetCount := unsafeReadUint32(reader)
	unsafeReadByte(reader) // extended callstack debugging flag - always false in a release build
	return RandomState{Version: version, Seed: seed, CallCount: callCount, ResetCount: resetCount}
}

func civ5StringHash(s string) uint32 {
	table := crc32.MakeTable(crc32.IEEE)
	crc := uint32(0xFFFFFFFF)
	for i := 0; i < len(s); i++ {
		crc = table[byte(crc)^s[i]] ^ (crc >> 8)
	}
	return crc
}
