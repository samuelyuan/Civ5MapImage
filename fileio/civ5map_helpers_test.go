package fileio

import (
	"bytes"
	"encoding/binary"
	"io"
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

func TestParseUnitNameArray(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(2))
	record1 := make([]byte, UnitNameRecordSize)
	copy(record1, "Custom Unit 1")
	record2 := make([]byte, UnitNameRecordSize)
	copy(record2, "Custom Unit 2")
	buf.Write(record1)
	buf.Write(record2)

	got := parseUnitNameArray(buf.Bytes())
	want := []string{"Custom Unit 1", "Custom Unit 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseUnitNameArray() = %v, want %v", got, want)
	}
}

func TestParseUnitNameArrayTooShort(t *testing.T) {
	if got := parseUnitNameArray([]byte{1, 2, 3}); got != nil {
		t.Errorf("parseUnitNameArray(too short) = %v, want nil", got)
	}
}

func TestCustomUnitName(t *testing.T) {
	unitNameList := []string{"Custom Unit 1", "Custom Unit 2"}
	if got := customUnitName(unitNameList, 1); got != "Custom Unit 2" {
		t.Errorf("customUnitName(1) = %q, want Custom Unit 2", got)
	}
	if got := customUnitName(unitNameList, NoNameIndexSentinel); got != "" {
		t.Errorf("customUnitName(NoNameIndexSentinel) = %q, want empty", got)
	}
	if got := customUnitName(unitNameList, 99); got != "" {
		t.Errorf("customUnitName(out of range) = %q, want empty", got)
	}
}

// TestTriangularPairCount checks the formula against team counts
func TestTriangularPairCount(t *testing.T) {
	cases := []struct{ n, want int }{
		{0, 0}, {1, 0}, {2, 1}, {3, 3}, {4, 6},
		{42, 861}, {50, 1225},
	}
	for _, c := range cases {
		if got := triangularPairCount(c.n); got != c.want {
			t.Errorf("triangularPairCount(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

// TestRelationshipBitsetSize checks per-type byte sizes for various team counts.
func TestRelationshipBitsetSize(t *testing.T) {
	cases := []struct{ teamCount, want int }{
		{3, 1},
		{42, 108},
		{50, 154},
	}
	for _, c := range cases {
		if got := relationshipBitsetSize(c.teamCount); got != c.want {
			t.Errorf("relationshipBitsetSize(%d) = %d, want %d", c.teamCount, got, c.want)
		}
	}
}

func TestRelationshipBitIndex(t *testing.T) {
	cases := []struct{ large, small, want int }{
		{1, 0, 0},
		{2, 0, 1},
		{2, 1, 2},
	}
	for _, c := range cases {
		if got := relationshipBitIndex(c.large, c.small); got != c.want {
			t.Errorf("relationshipBitIndex(%d, %d) = %d, want %d", c.large, c.small, got, c.want)
		}
	}
}

func TestDecodeRelationshipPairs(t *testing.T) {
	// bit0 (pair 1-0) and bit2 (pair 2-1) set; bit1 (pair 2-0) clear.
	got := decodeRelationshipPairs([]byte{0b101}, 3)
	want := [][2]int{{1, 0}, {2, 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decodeRelationshipPairs(0b101, 3) = %v, want %v", got, want)
	}
}

func TestDecodeRelationshipPairsShorterThanNeeded(t *testing.T) {
	// Empty data: every bit is treated as unset rather than panicking on out-of-range access.
	if got := decodeRelationshipPairs([]byte{}, 3); got != nil {
		t.Errorf("decodeRelationshipPairs(empty, 3) = %v, want nil", got)
	}
}

func TestParseTeamNames(t *testing.T) {
	teamNameData := make([]byte, 0, 3*TeamNameRecordSize)
	for _, name := range []string{"Team 1", "Team 2", "Team 3"} {
		record := make([]byte, TeamNameRecordSize)
		copy(record, name)
		teamNameData = append(teamNameData, record...)
	}

	teams := ParseTeamNames(teamNameData, 3)
	if len(teams) != 3 {
		t.Fatalf("ParseTeamNames() returned %d teams, want 3", len(teams))
	}
	if teams[0].Name != "Team 1" || teams[1].Name != "Team 2" || teams[2].Name != "Team 3" {
		t.Errorf("team names = %q, %q, %q", teams[0].Name, teams[1].Name, teams[2].Name)
	}
}

// TestParseTeamRelationships reconstructs a 3-team relationship scenario.
func TestParseTeamRelationships(t *testing.T) {
	// One byte per relationship type: team1-team0 in contact and at war; team2-team1 has a defensive pact.
	relationshipData := []byte{0b001, 0b001, 0b000, 0b000, 0b100}

	rel := ParseTeamRelationships(relationshipData, 3)

	if !reflect.DeepEqual(rel.InContactWith[0], []int{1}) {
		t.Errorf("InContactWith[0] = %v, want [1]", rel.InContactWith[0])
	}
	if !reflect.DeepEqual(rel.InContactWith[1], []int{0}) {
		t.Errorf("InContactWith[1] = %v, want [0]", rel.InContactWith[1])
	}
	if !reflect.DeepEqual(rel.AtWarWith[0], []int{1}) {
		t.Errorf("AtWarWith[0] = %v, want [1]", rel.AtWarWith[0])
	}
	if !reflect.DeepEqual(rel.AtWarWith[1], []int{0}) {
		t.Errorf("AtWarWith[1] = %v, want [0]", rel.AtWarWith[1])
	}
	if rel.PermanentWarOrPeaceWith[0] != nil || rel.OpenBordersWith[0] != nil {
		t.Errorf("team0 should have no PermanentWarOrPeace/OpenBorders relationships, got %+v", rel)
	}
	if !reflect.DeepEqual(rel.DefensivePactWith[1], []int{2}) {
		t.Errorf("DefensivePactWith[1] = %v, want [2]", rel.DefensivePactWith[1])
	}
	if !reflect.DeepEqual(rel.DefensivePactWith[2], []int{1}) {
		t.Errorf("DefensivePactWith[2] = %v, want [1]", rel.DefensivePactWith[2])
	}
	if rel.AtWarWith[2] != nil || rel.InContactWith[2] != nil {
		t.Errorf("team2 should have no AtWar/InContact relationships, got %+v", rel)
	}
}

func TestParseTeamVisibility(t *testing.T) {
	// 2x2 map: team0 sees (0,0),(1,1); team1 sees nothing; team2 sees (0,1).
	visibilityData := []byte{0b00001001, 0b00000100}

	visible := ParseTeamVisibility(visibilityData, MapSize{Height: 2, Width: 2}, 3)
	if len(visible) != 3 {
		t.Fatalf("ParseTeamVisibility() returned %d teams, want 3", len(visible))
	}
	if !reflect.DeepEqual(visible[0], [][2]int{{0, 0}, {1, 1}}) {
		t.Errorf("visible[0] = %v, want [[0 0] [1 1]]", visible[0])
	}
	if visible[1] != nil {
		t.Errorf("visible[1] = %v, want nil", visible[1])
	}
	if !reflect.DeepEqual(visible[2], [][2]int{{0, 1}}) {
		t.Errorf("visible[2] = %v, want [[0 1]]", visible[2])
	}
}

// On a 3 wide x 2 high map a bit is team*6 + y*3 + x, so a swapped width and height would read different tiles.
func TestParseTeamVisibilityOnANonSquareMap(t *testing.T) {
	// team0 sees (x=2, y=0) and (x=0, y=1): bits 2 and 3. team1 sees (x=1, y=1): bit 6+4 = 10.
	visibilityData := []byte{0b00001100, 0b00000100}

	visible := ParseTeamVisibility(visibilityData, MapSize{Height: 2, Width: 3}, 2)
	if !reflect.DeepEqual(visible[0], [][2]int{{2, 0}, {0, 1}}) {
		t.Errorf("visible[0] = %v, want [[2 0] [0 1]]", visible[0])
	}
	if !reflect.DeepEqual(visible[1], [][2]int{{1, 1}}) {
		t.Errorf("visible[1] = %v, want [[1 1]]", visible[1])
	}
}

func TestParseTeamNamesZeroTeams(t *testing.T) {
	if teams := ParseTeamNames(nil, 0); len(teams) != 0 {
		t.Errorf("ParseTeamNames(0 teams) = %v, want empty", teams)
	}
}

func TestParseCityStateInfluence(t *testing.T) {
	playerCount, cityStateCount := 2, 2
	data := make([]byte, playerCount*MaxCityStatesPerPlayer*4)
	putInt32 := func(slotIndex int, value int32) {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(value))
		copy(data[slotIndex*4:], b)
	}
	player1Row := MaxCityStatesPerPlayer // where player 1's row starts, not at cityStateCount
	putInt32(0, 100)                     // player 0, city-state 0
	putInt32(1, 0)                       // player 0, city-state 1 - untouched, stays 0
	putInt32(player1Row, 200)            // player 1, city-state 0
	putInt32(player1Row+1, -1)           // player 1, city-state 1 - negative values decode correctly

	result := ParseCityStateInfluence(data, playerCount, cityStateCount)
	if len(result) != playerCount {
		t.Fatalf("ParseCityStateInfluence() returned %d players, want %d", len(result), playerCount)
	}
	if result[0][0] != 100 || result[0][1] != 0 {
		t.Errorf("player 0 influence = %v, want {0:100, 1:0}", result[0])
	}
	if result[1][0] != 200 || result[1][1] != -1 {
		t.Errorf("player 1 influence = %v, want {0:200, 1:-1}", result[1])
	}
}

func TestParseCityStateInfluenceShorterThanNeeded(t *testing.T) {
	// Only enough data for player 0's row; player 1 should come back empty, not panic.
	data := make([]byte, MaxCityStatesPerPlayer*4)
	binary.LittleEndian.PutUint32(data[0:4], 42)

	result := ParseCityStateInfluence(data, 2, 1)
	if len(result) != 2 {
		t.Fatalf("ParseCityStateInfluence() returned %d players, want 2", len(result))
	}
	if result[0][0] != 42 {
		t.Errorf("player 0 influence = %v, want {0:42}", result[0])
	}
	if len(result[1]) != 0 {
		t.Errorf("player 1 influence = %v, want empty (data too short)", result[1])
	}
}

// TestFacingDirectionName checks every raw value against its expected direction name.
func TestFacingDirectionName(t *testing.T) {
	cases := []struct {
		raw  int
		want string
	}{
		{0, "Random Direction"},
		{1, "Northeast"},
		{2, "East"},
		{3, "Southeast"},
		{4, "Southwest"},
		{5, "West"},
		{6, "Northwest"},
	}
	for _, c := range cases {
		if got := facingDirectionName(c.raw); got != c.want {
			t.Errorf("facingDirectionName(%d) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestFacingDirectionNameOutOfRange(t *testing.T) {
	if got := facingDirectionName(7); got != "" {
		t.Errorf("facingDirectionName(7) = %q, want empty", got)
	}
	if got := facingDirectionName(-1); got != "" {
		t.Errorf("facingDirectionName(-1) = %q, want empty", got)
	}
}

func TestDecodeBitset(t *testing.T) {
	typeList := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	// bits 0, 2, and 8 set (the last one lands in a second byte)
	data := []byte{0b00000101, 0b00000001}
	want := []string{"A", "C", "I"}
	if got := decodeBitset(data, typeList); !reflect.DeepEqual(got, want) {
		t.Errorf("decodeBitset() = %v, want %v", got, want)
	}
}

func TestDecodeBitsetShorterThanTypeList(t *testing.T) {
	// typeList has more entries than data has bits for; extra entries are simply unreachable.
	typeList := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	data := []byte{0b00000001}
	want := []string{"A"}
	if got := decodeBitset(data, typeList); !reflect.DeepEqual(got, want) {
		t.Errorf("decodeBitset() = %v, want %v", got, want)
	}
}

func TestParsePhysicalMapTilesPassesThroughRiverData(t *testing.T) {
	var buf bytes.Buffer
	// Tile 0: bits 6 and 0 set; undecoded bits must still round-trip untouched.
	binary.Write(&buf, binary.LittleEndian, Civ5MapTile{RiverData: 1<<6 | 1})
	// Tile 1: bit 7 set only.
	binary.Write(&buf, binary.LittleEndian, Civ5MapTile{RiverData: 1 << 7})

	header := &Civ5MapHeader{Width: 2, Height: 1}
	reader := io.NewSectionReader(bytes.NewReader(buf.Bytes()), 0, int64(buf.Len()))
	tiles, err := parsePhysicalMapTiles(reader, header)
	if err != nil {
		t.Fatalf("parsePhysicalMapTiles returned error: %v", err)
	}

	if got := tiles[0][0]; got.RiverData != 1<<6|1 {
		t.Errorf("tile(0,0).RiverData = %d, want %d", got.RiverData, 1<<6|1)
	}
	if got := tiles[0][1]; got.RiverData != 1<<7 {
		t.Errorf("tile(0,1).RiverData = %d, want %d", got.RiverData, 1<<7)
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
		StackedUnitHandle: 42,
		NameIndex:         1,
		Experience:        10,
		Health:            100,
		UnitType:          3,
		Owner:             1,
		FacingDirection:   2,
		Status:            1<<FortifiedBit | 1<<GarrisonedBit,
	}
	binary.Write(&buf, binary.LittleEndian, header)

	typeLists := Civ5GameTypeLists{
		PromotionTypeList: []string{"PROMOTION_DRILL_1"},
		UnitTypeList:      []string{"UNIT_WARRIOR", "UNIT_SCOUT", "UNIT_SETTLER", "UNIT_WORKER"},
		UnitNameList:      []string{"Custom Unit 1", "Custom Unit 2"},
	}
	units, err := ParseUnitData(buf.Bytes(), MapVersion12, typeLists)
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
	if got.FacingDirectionName != "East" {
		t.Errorf("ParseUnitData() FacingDirectionName = %q, want East", got.FacingDirectionName)
	}
	if got.Name != "UNIT_WORKER" {
		t.Errorf("ParseUnitData() Name = %q, want UNIT_WORKER", got.Name)
	}
	if got.CustomName != "Custom Unit 2" {
		t.Errorf("ParseUnitData() CustomName = %q, want Custom Unit 2", got.CustomName)
	}
	if got.Status != 1<<FortifiedBit|1<<GarrisonedBit {
		t.Errorf("ParseUnitData() Status = %d, want %d", got.Status, 1<<FortifiedBit|1<<GarrisonedBit)
	}
	if got.StackedUnitId != 42 {
		t.Errorf("ParseUnitData() StackedUnitId = %d, want 42", got.StackedUnitId)
	}
}

func TestParseUnitDataStackedUnitSentinel(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1))
	header := Civ5UnitHeaderV12{StackedUnitHandle: NoUnitIdSentinel}
	binary.Write(&buf, binary.LittleEndian, header)

	units, err := ParseUnitData(buf.Bytes(), MapVersion12, Civ5GameTypeLists{})
	if err != nil {
		t.Fatalf("ParseUnitData returned error: %v", err)
	}
	if got := units[0].StackedUnitId; got != InvalidUnitId {
		t.Errorf("ParseUnitData() StackedUnitId = %d, want InvalidUnitId", got)
	}
}

func TestParseUnitDataEmpty(t *testing.T) {
	units, err := ParseUnitData([]byte{}, MapVersion12, Civ5GameTypeLists{})
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

	units, err := ParseUnitData(buf.Bytes(), MapVersion12, Civ5GameTypeLists{})
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
	buildingInfo := make([]byte, BuildingDataSizeV11)
	buildingInfo[0] = 1 // bit 0 set: has the first building in buildingTypeList
	buf.Write(buildingInfo)

	buildingTypeList := []string{"BUILDING_PALACE", "BUILDING_GRANARY"}
	cities, err := ParseCityData(buf.Bytes(), MapVersion11, 0, buildingTypeList)
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
	if want := []string{"BUILDING_PALACE"}; !reflect.DeepEqual(cities[0].Buildings, want) {
		t.Errorf("ParseCityData() Buildings = %v, want %v", cities[0].Buildings, want)
	}
}

func TestParseCivData(t *testing.T) {
	header := Civ5PlayerHeader{}
	copy(header.CivType[:], "CIVILIZATION_ROME")
	copy(header.TeamColor[:], "PLAYERCOLOR_RED")
	header.Policies[0] = 1 << 2 // bit 2 set: has the third policy in policyTypeList

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, header)

	policyTypeList := []string{"POLICY_TRADITION", "POLICY_LIBERTY", "POLICY_HONOR"}
	players, err := ParseCivData(buf.Bytes(), policyTypeList)
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
	if want := []string{"POLICY_HONOR"}; !reflect.DeepEqual(players[0].Policies, want) {
		t.Errorf("ParseCivData() Policies = %v, want %v", players[0].Policies, want)
	}
}

func TestParseMapTileProperties(t *testing.T) {
	var buf bytes.Buffer
	// Tile 0: no city, no unit. Tile 1: unit index 5
	tile0 := Civ5MapTileHeader{
		CityId:      uint16(NoCityIdSentinel),
		UnitId:      uint16(NoUnitIdSentinel),
		Owner:       0xFF,
		Improvement: 0,
		RouteType:   0,
		RouteOwner:  0,
	}
	tile1 := tile0
	tile1.UnitId = 5
	binary.Write(&buf, binary.LittleEndian, tile0)
	binary.Write(&buf, binary.LittleEndian, tile1)

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
	if improvements[0][0].UnitId != InvalidUnitId {
		t.Errorf("ParseMapTileProperties() tile 0 UnitId = %d, want InvalidUnitId (%d)", improvements[0][0].UnitId, InvalidUnitId)
	}
	if improvements[0][1].UnitId != 5 {
		t.Errorf("ParseMapTileProperties() tile 1 UnitId = %d, want 5", improvements[0][1].UnitId)
	}
}

func TestParseMapTilePropertiesTooShort(t *testing.T) {
	_, err := ParseMapTileProperties([]byte{1, 2, 3}, 2, 2)
	if err == nil {
		t.Errorf("expected error for insufficient tile data, got nil")
	}
}
