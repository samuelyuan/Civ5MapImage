package fileio

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestByteArrayToStringArray(t *testing.T) {
	input := []byte("foo\x00bar\x00baz\x00")
	got := byteArrayToStringArray(input)
	want := []string{"foo", "bar", "baz"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("byteArrayToStringArray() = %v, want %v", got, want)
	}
}

func TestByteArrayToStringArrayNoTrailingNull(t *testing.T) {
	// Text after the last null byte is dropped, matching the streaming parser behavior.
	input := []byte("foo\x00bar")
	got := byteArrayToStringArray(input)
	want := []string{"foo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("byteArrayToStringArray() = %v, want %v", got, want)
	}
}

func TestNullTerminatedString(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"has null", []byte("hello\x00world"), "hello"},
		{"no null", []byte("hello"), "hello"},
		{"empty", []byte{}, ""},
		{"leading null", []byte{0, 'a', 'b'}, ""},
	}
	for _, tt := range tests {
		if got := nullTerminatedString(tt.in); got != tt.want {
			t.Errorf("%s: nullTerminatedString(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestMaxUnitCountForVersion(t *testing.T) {
	if got := maxUnitCountForVersion(UnitDataSizeV12*3, MapVersion12); got != 3 {
		t.Errorf("maxUnitCountForVersion(v12) = %d, want 3", got)
	}
	if got := maxUnitCountForVersion(UnitDataSizeV11*4, MapVersion11); got != 4 {
		t.Errorf("maxUnitCountForVersion(v11) = %d, want 4", got)
	}
}

func TestBuildingDataSizeForVersion(t *testing.T) {
	if got := buildingDataSizeForVersion(MapVersion12); got != BuildingDataSizeV12 {
		t.Errorf("buildingDataSizeForVersion(v12) = %d, want %d", got, BuildingDataSizeV12)
	}
	if got := buildingDataSizeForVersion(MapVersion11); got != BuildingDataSizeV11 {
		t.Errorf("buildingDataSizeForVersion(v11) = %d, want %d", got, BuildingDataSizeV11)
	}
	if got := buildingDataSizeForVersion(99); got != BuildingDataSizeV11 {
		t.Errorf("buildingDataSizeForVersion(unknown) = %d, want default %d", got, BuildingDataSizeV11)
	}
}

func TestAdjustedCityOwner(t *testing.T) {
	if got := adjustedCityOwner(10); got != 10 {
		t.Errorf("adjustedCityOwner(10) = %d, want 10", got)
	}
	if got := adjustedCityOwner(CityStateOffset + 3); got != 3 {
		t.Errorf("adjustedCityOwner(CityStateOffset+3) = %d, want 3", got)
	}
}

func TestMapVersionAndScenario(t *testing.T) {
	// scenarioVersion packs scenario into the high nibble and version into the low nibble
	scenarioVersion := uint8((2 << ScenarioBits) | MapVersion12)
	if got := mapVersion(scenarioVersion); got != MapVersion12 {
		t.Errorf("mapVersion() = %d, want %d", got, MapVersion12)
	}
	if got := mapScenario(scenarioVersion); got != 2 {
		t.Errorf("mapScenario() = %d, want 2", got)
	}
}

func TestGetSortedKeysInt(t *testing.T) {
	m := map[int]string{3: "c", 1: "a", 2: "b"}
	got := GetSortedKeys(m)
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetSortedKeys(int map) = %v, want %v", got, want)
	}
}

func TestGetSortedKeysString(t *testing.T) {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	got := GetSortedKeys(m)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetSortedKeys(string map) = %v, want %v", got, want)
	}
}

func TestFindMaxCityId(t *testing.T) {
	improvements := [][]*Civ5MapTileImprovement{
		{
			{CityId: 2}, {CityId: InvalidCityId},
		},
		{
			{CityId: 5}, {CityId: 1},
		},
	}
	if got := findMaxCityId(improvements, 2, 2); got != 5 {
		t.Errorf("findMaxCityId() = %d, want 5", got)
	}
}

func TestFindMaxCityIdAllInvalid(t *testing.T) {
	improvements := [][]*Civ5MapTileImprovement{
		{
			{CityId: InvalidCityId}, {CityId: InvalidCityId},
		},
	}
	if got := findMaxCityId(improvements, 1, 2); got != 0 {
		t.Errorf("findMaxCityId() = %d, want 0", got)
	}
}

func TestParseUnitDataV12(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // number of units

	header := Civ5UnitHeaderV12{
		Experience:      10,
		Health:          100,
		UnitType:        3,
		Owner:           1,
		FacingDirection: 2,
		Status:          0,
	}
	binary.Write(&buf, binary.LittleEndian, header)

	units, err := ParseUnitData(buf.Bytes(), MapVersion12)
	if err != nil {
		t.Fatalf("ParseUnitData returned error: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("ParseUnitData() returned %d units, want 1", len(units))
	}
	got := units[0]
	if got.Experience != 10 || got.Health != 100 || got.UnitType != 3 || got.Owner != 1 || got.FacingDirection != 2 {
		t.Errorf("ParseUnitData() unit = %+v, want matching header fields", got)
	}
}

func TestParseUnitDataEmpty(t *testing.T) {
	units, err := ParseUnitData([]byte{}, MapVersion12)
	if err != nil {
		t.Fatalf("ParseUnitData returned error: %v", err)
	}
	if units != nil {
		t.Errorf("ParseUnitData(empty) = %v, want nil", units)
	}
}

func TestParseUnitDataClampsCorruptCount(t *testing.T) {
	var buf bytes.Buffer
	// Claim far more units than the buffer could possibly contain.
	binary.Write(&buf, binary.LittleEndian, uint32(1000000))

	header := Civ5UnitHeaderV12{UnitType: 1}
	binary.Write(&buf, binary.LittleEndian, header)

	units, err := ParseUnitData(buf.Bytes(), MapVersion12)
	if err != nil {
		t.Fatalf("ParseUnitData returned error: %v", err)
	}
	// Should be clamped to what actually fits (1 unit's worth of data follows the count).
	if len(units) != 1 {
		t.Fatalf("ParseUnitData() returned %d units, want 1 (clamped)", len(units))
	}
}

func TestParseCityData(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // number of cities

	nameBytes := [PlayerNameSize]byte{}
	copy(nameBytes[:], "Rome")
	cityHeader := Civ5CityHeader{
		Name:       nameBytes,
		Owner:      0,
		Flags:      0,
		Population: 5,
		Health:     100000,
	}
	binary.Write(&buf, binary.LittleEndian, cityHeader)
	buf.Write(make([]byte, BuildingDataSizeV11))

	cities, err := ParseCityData(buf.Bytes(), MapVersion11, 0)
	if err != nil {
		t.Fatalf("ParseCityData returned error: %v", err)
	}
	if len(cities) != 1 {
		t.Fatalf("ParseCityData() returned %d cities, want 1", len(cities))
	}
	if cities[0].Name != "Rome" {
		t.Errorf("ParseCityData() city name = %q, want Rome", cities[0].Name)
	}
	if cities[0].Population != 5 {
		t.Errorf("ParseCityData() population = %d, want 5", cities[0].Population)
	}
}

func TestParseCivData(t *testing.T) {
	header := Civ5PlayerHeader{}
	copy(header.CivType[:], "CIVILIZATION_ROME")
	copy(header.TeamColor[:], "PLAYERCOLOR_RED")

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, header)

	players, err := ParseCivData(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseCivData returned error: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("ParseCivData() returned %d players, want 1", len(players))
	}
	if players[0].CivType != "CIVILIZATION_ROME" {
		t.Errorf("ParseCivData() CivType = %q, want CIVILIZATION_ROME", players[0].CivType)
	}
	if players[0].TeamColor != "PLAYERCOLOR_RED" {
		t.Errorf("ParseCivData() TeamColor = %q, want PLAYERCOLOR_RED", players[0].TeamColor)
	}
}

func TestParseMapTileProperties(t *testing.T) {
	var buf bytes.Buffer
	// 2 tiles (1x2 map)
	for i := 0; i < 2; i++ {
		tile := Civ5MapTileHeader{
			CityId:      uint16(RawNoCityId),
			Owner:       0xFF,
			Improvement: 0,
			RouteType:   0,
			RouteOwner:  0,
		}
		binary.Write(&buf, binary.LittleEndian, tile)
	}

	improvements, err := ParseMapTileProperties(buf.Bytes(), 1, 2)
	if err != nil {
		t.Fatalf("ParseMapTileProperties returned error: %v", err)
	}
	if len(improvements) != 1 || len(improvements[0]) != 2 {
		t.Fatalf("ParseMapTileProperties() returned wrong dimensions: %v", improvements)
	}
	if improvements[0][0].CityId != InvalidCityId {
		t.Errorf("ParseMapTileProperties() CityId = %d, want InvalidCityId (%d)", improvements[0][0].CityId, InvalidCityId)
	}
}

func TestParseMapTilePropertiesTooShort(t *testing.T) {
	_, err := ParseMapTileProperties([]byte{1, 2, 3}, 2, 2)
	if err == nil {
		t.Errorf("expected error for insufficient tile data, got nil")
	}
}
