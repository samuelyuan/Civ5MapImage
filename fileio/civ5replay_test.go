package fileio

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// writeVarString appends a length-prefixed string in the format readVarString expects.
func writeVarString(buf *bytes.Buffer, s string) {
	binary.Write(buf, binary.LittleEndian, uint32(len(s)))
	buf.WriteString(s)
}

func TestGroupEventsByTurn(t *testing.T) {
	events := []Civ5ReplayEvent{
		{Turn: 1, TypeId: 0, Text: "a"},
		{Turn: 2, TypeId: 1, Text: "b"},
		{Turn: 1, TypeId: 2, Text: "c"},
	}

	grouped := GroupEventsByTurn(events)

	if len(grouped) != 2 {
		t.Fatalf("GroupEventsByTurn() returned %d turns, want 2", len(grouped))
	}
	if len(grouped[1]) != 2 {
		t.Errorf("turn 1 has %d events, want 2", len(grouped[1]))
	}
	if len(grouped[2]) != 1 {
		t.Errorf("turn 2 has %d events, want 1", len(grouped[2]))
	}
	if grouped[1][0].Text != "a" || grouped[1][1].Text != "c" {
		t.Errorf("turn 1 events out of order: %+v", grouped[1])
	}
}

func TestGroupEventsByTurnEmpty(t *testing.T) {
	grouped := GroupEventsByTurn(nil)
	if len(grouped) != 0 {
		t.Errorf("GroupEventsByTurn(nil) = %v, want empty map", grouped)
	}
}

func TestReadCivs(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // civsLength
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // unknownVariable1
	binary.Write(&buf, binary.LittleEndian, uint32(2)) // unknownVariable2
	binary.Write(&buf, binary.LittleEndian, uint32(3)) // unknownVariable3
	binary.Write(&buf, binary.LittleEndian, uint32(4)) // unknownVariable4
	writeVarString(&buf, "Augustus")
	writeVarString(&buf, "Roman Empire")
	writeVarString(&buf, "Rome")
	writeVarString(&buf, "Romans")

	civs := readCivs(newSectionReader(buf.Bytes()))

	if len(civs) != 1 {
		t.Fatalf("readCivs() returned %d civs, want 1", len(civs))
	}
	got := civs[0]
	want := Civ5ReplayCiv{
		UnknownVariables: [4]int{1, 2, 3, 4},
		Leader:           "Augustus",
		LongName:         "Roman Empire",
		Name:             "Rome",
		Demonym:          "Romans",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readCivs()[0] = %+v, want %+v", got, want)
	}
}

func TestReadCivsEmpty(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	civs := readCivs(newSectionReader(buf.Bytes()))
	if len(civs) != 0 {
		t.Errorf("readCivs() returned %d civs, want 0", len(civs))
	}
}

func TestReadEvents(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // eventsLength
	binary.Write(&buf, binary.LittleEndian, uint32(5)) // turn
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // typeId
	binary.Write(&buf, binary.LittleEndian, uint32(2)) // numTiles
	binary.Write(&buf, binary.LittleEndian, uint16(3)) // tile0 X
	binary.Write(&buf, binary.LittleEndian, uint16(4)) // tile0 Y
	binary.Write(&buf, binary.LittleEndian, uint16(7)) // tile1 X
	binary.Write(&buf, binary.LittleEndian, uint16(8)) // tile1 Y
	binary.Write(&buf, binary.LittleEndian, uint32(2)) // civId
	writeVarString(&buf, "Rome is founded.")

	events := readEvents(newSectionReader(buf.Bytes()))

	if len(events) != 1 {
		t.Fatalf("readEvents() returned %d events, want 1", len(events))
	}
	got := events[0]
	want := Civ5ReplayEvent{
		Turn:   5,
		TypeId: 1,
		Tiles: []Civ5ReplayEventTile{
			{X: 3, Y: 4},
			{X: 7, Y: 8},
		},
		CivId: 2,
		Text:  "Rome is founded.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readEvents()[0] = %+v, want %+v", got, want)
	}
}

func TestReadEventsNegativeCivId(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // eventsLength
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // turn
	binary.Write(&buf, binary.LittleEndian, uint32(4)) // typeId (razed)
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // numTiles
	binary.Write(&buf, binary.LittleEndian, int32(-1)) // civId, stored as signed
	writeVarString(&buf, "")

	events := readEvents(newSectionReader(buf.Bytes()))

	if events[0].CivId != -1 {
		t.Errorf("readEvents()[0].CivId = %d, want -1", events[0].CivId)
	}
	if len(events[0].Tiles) != 0 {
		t.Errorf("readEvents()[0].Tiles = %v, want empty", events[0].Tiles)
	}
}

func TestReadDatasetNames(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(2))
	writeVarString(&buf, "Score")
	writeVarString(&buf, "Gold")

	names := readDatasetNames(newSectionReader(buf.Bytes()))
	want := []string{"Score", "Gold"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("readDatasetNames() = %v, want %v", names, want)
	}
}

func TestReadDatasetValues(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1))   // 1 civ
	binary.Write(&buf, binary.LittleEndian, uint32(1))   // 1 category
	binary.Write(&buf, binary.LittleEndian, uint32(2))   // 2 entries
	binary.Write(&buf, binary.LittleEndian, uint32(1))   // turn
	binary.Write(&buf, binary.LittleEndian, uint32(100)) // value
	binary.Write(&buf, binary.LittleEndian, uint32(2))   // turn
	binary.Write(&buf, binary.LittleEndian, uint32(150)) // value

	values := readDatasetValues(newSectionReader(buf.Bytes()))

	if len(values) != 1 || len(values[0]) != 1 || len(values[0][0]) != 2 {
		t.Fatalf("readDatasetValues() shape = %v, want [1][1][2]", values)
	}
	want := []Civ5ReplayDataEntry{{Turn: 1, Value: 100}, {Turn: 2, Value: 150}}
	if !reflect.DeepEqual(values[0][0], want) {
		t.Errorf("readDatasetValues()[0][0] = %v, want %v", values[0][0], want)
	}
}

func TestBuildCivDatasetValues(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1))  // 1 civ
	binary.Write(&buf, binary.LittleEndian, uint32(2))  // 2 categories (matches datasetNames below)
	binary.Write(&buf, binary.LittleEndian, uint32(1))  // 1 entry for "Score"
	binary.Write(&buf, binary.LittleEndian, uint32(1))  // turn
	binary.Write(&buf, binary.LittleEndian, uint32(42)) // value
	binary.Write(&buf, binary.LittleEndian, uint32(1))  // 1 entry for "Gold"
	binary.Write(&buf, binary.LittleEndian, uint32(1))  // turn
	binary.Write(&buf, binary.LittleEndian, uint32(99)) // value

	datasetNames := []string{"Score", "Gold"}
	civDatasets := buildCivDatasetValues(newSectionReader(buf.Bytes()), datasetNames)

	if len(civDatasets) != 1 {
		t.Fatalf("buildCivDatasetValues() returned %d civs, want 1", len(civDatasets))
	}
	if civDatasets[0].CivIndex != 0 {
		t.Errorf("CivIndex = %d, want 0", civDatasets[0].CivIndex)
	}
	scoreValues := civDatasets[0].DatasetValues["Score"]
	if len(scoreValues) != 1 || scoreValues[0].Value != 42 {
		t.Errorf("DatasetValues[Score] = %v, want [{Turn:1 Value:42}]", scoreValues)
	}
	goldValues := civDatasets[0].DatasetValues["Gold"]
	if len(goldValues) != 1 || goldValues[0].Value != 99 {
		t.Errorf("DatasetValues[Gold] = %v, want [{Turn:1 Value:99}]", goldValues)
	}
}
