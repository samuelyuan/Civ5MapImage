package fileio

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// newSectionReader wraps raw bytes in an io.SectionReader, mirroring how the
// production code reads from files.
func newSectionReader(data []byte) *io.SectionReader {
	return io.NewSectionReader(bytes.NewReader(data), 0, int64(len(data)))
}

func TestReadVarString(t *testing.T) {
	var buf bytes.Buffer
	value := "hello world"
	binary.Write(&buf, binary.LittleEndian, uint32(len(value)))
	buf.WriteString(value)

	reader := newSectionReader(buf.Bytes())
	got, err := readVarString(reader, "testVar")
	if err != nil {
		t.Fatalf("readVarString returned error: %v", err)
	}
	if got != value {
		t.Errorf("readVarString() = %q, want %q", got, value)
	}
}

func TestReadVarStringEmpty(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	reader := newSectionReader(buf.Bytes())
	got, err := readVarString(reader, "empty")
	if err != nil {
		t.Fatalf("readVarString returned error: %v", err)
	}
	if got != "" {
		t.Errorf("readVarString() = %q, want empty string", got)
	}
}

func TestReadVarStringTruncated(t *testing.T) {
	// Claims a length of 100 bytes but only provides the length prefix.
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(100))

	reader := newSectionReader(buf.Bytes())
	if _, err := readVarString(reader, "truncated"); err == nil {
		t.Errorf("expected error reading truncated string, got nil")
	}
}

func TestUnsafeReadUint32(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0xDEADBEEF))

	reader := newSectionReader(buf.Bytes())
	got := unsafeReadUint32(reader)
	if got != 0xDEADBEEF {
		t.Errorf("unsafeReadUint32() = %#x, want %#x", got, 0xDEADBEEF)
	}
}

func TestUnsafeReadUint32Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic reading uint32 from empty reader")
		}
	}()
	reader := newSectionReader([]byte{})
	unsafeReadUint32(reader)
}

func TestUnsafeReadUint16(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(0xBEEF))

	reader := newSectionReader(buf.Bytes())
	got := unsafeReadUint16(reader)
	if got != 0xBEEF {
		t.Errorf("unsafeReadUint16() = %#x, want %#x", got, 0xBEEF)
	}
}

func TestReadFileConfig(t *testing.T) {
	var buf bytes.Buffer
	// varstring field
	name := "abc"
	binary.Write(&buf, binary.LittleEndian, uint32(len(name)))
	buf.WriteString(name)
	// uint32 field
	binary.Write(&buf, binary.LittleEndian, uint32(42))
	// uint8 field
	binary.Write(&buf, binary.LittleEndian, uint8(7))

	entries := []Civ5ReplayFileConfigEntry{
		{VariableType: "varstring", VariableName: "name"},
		{VariableType: "uint32", VariableName: "count"},
		{VariableType: "uint8", VariableName: "flag"},
	}

	reader := newSectionReader(buf.Bytes())
	fieldValues, err := readFileConfig(reader, entries)
	if err != nil {
		t.Fatalf("readFileConfig returned error: %v", err)
	}
	if len(fieldValues) != 3 {
		t.Fatalf("readFileConfig() returned %d values, want 3", len(fieldValues))
	}
	if fieldValues[0] != "name(str):abc" {
		t.Errorf("fieldValues[0] = %q, want %q", fieldValues[0], "name(str):abc")
	}
	if fieldValues[1] != "count(u32):42" {
		t.Errorf("fieldValues[1] = %q, want %q", fieldValues[1], "count(u32):42")
	}
	if fieldValues[2] != "flag(u8):7" {
		t.Errorf("fieldValues[2] = %q, want %q", fieldValues[2], "flag(u8):7")
	}
}

func TestReadArray(t *testing.T) {
	var buf bytes.Buffer
	// array length = 2
	binary.Write(&buf, binary.LittleEndian, uint32(2))
	// two uint32 entries
	binary.Write(&buf, binary.LittleEndian, uint32(1))
	binary.Write(&buf, binary.LittleEndian, uint32(2))

	entries := []Civ5ReplayFileConfigEntry{
		{VariableType: "uint32", VariableName: "value"},
	}

	reader := newSectionReader(buf.Bytes())
	if err := readArray(reader, "testArray", entries); err != nil {
		t.Fatalf("readArray returned error: %v", err)
	}
}

func TestReadArrayTooLong(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(MaxArrayLength+1))

	reader := newSectionReader(buf.Bytes())
	if err := readArray(reader, "hugeArray", nil); err == nil {
		t.Errorf("expected error for array length exceeding MaxArrayLength, got nil")
	}
}
