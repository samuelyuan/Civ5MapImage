package fileio

import (
	"strings"
	"testing"
)

func TestGetNeighborsOddRow(t *testing.T) {
	// row 1 is odd, should use NeighborOdd offsets
	got := GetNeighbors(TilePos{Row: 1, Col: 5})
	want := [6]TilePos{{Row: 2, Col: 6}, {Row: 2, Col: 5}, {Row: 1, Col: 4}, {Row: 0, Col: 5}, {Row: 0, Col: 6}, {Row: 1, Col: 6}}
	if got != want {
		t.Errorf("GetNeighbors(row 1, col 5) = %v, want %v", got, want)
	}
}

func TestGetNeighborsEvenRow(t *testing.T) {
	// row 2 is even, should use NeighborEven offsets
	got := GetNeighbors(TilePos{Row: 2, Col: 5})
	want := [6]TilePos{{Row: 3, Col: 5}, {Row: 3, Col: 4}, {Row: 2, Col: 4}, {Row: 1, Col: 4}, {Row: 1, Col: 5}, {Row: 2, Col: 6}}
	if got != want {
		t.Errorf("GetNeighbors(row 2, col 5) = %v, want %v", got, want)
	}
}

func TestTilePosInMap(t *testing.T) {
	const height, width = 3, 4
	tests := []struct {
		pos  TilePos
		want bool
	}{
		{TilePos{Row: 0, Col: 0}, true},
		{TilePos{Row: 2, Col: 3}, true},
		{TilePos{Row: 3, Col: 0}, false}, // row is the first axis, bounded by the height
		{TilePos{Row: 0, Col: 4}, false}, // col is bounded by the width
		{TilePos{Row: -1, Col: 0}, false},
		{TilePos{Row: 0, Col: -1}, false},
	}
	for _, tt := range tests {
		if got := tt.pos.InMap(MapSize{Height: height, Width: width}); got != tt.want {
			t.Errorf("%+v.InMap(%d, %d) = %v, want %v", tt.pos, height, width, got, tt.want)
		}
	}
}

func newTestMapData() *Civ5MapData {
	return &Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"},
		MapTiles: [][]*Civ5MapTilePhysical{
			{
				{TerrainType: 0, Elevation: 0},
				{TerrainType: 1, Elevation: 2},
			},
		},
		MapTileImprovements: [][]*Civ5MapTileImprovement{
			{
				{CityId: 0, Owner: 0},
				{CityId: InvalidCityId, Owner: 0xFF},
			},
		},
		Civ5PlayerData: []*Civ5PlayerData{
			{Index: 0, CivType: "CIVILIZATION_ROME", TeamColor: "PLAYERCOLOR_RED"},
		},
		CityOwnerIndexMap: map[int]int{0: 0},
	}
}

func TestGetTerrainStringBounds(t *testing.T) {
	mapData := newTestMapData()

	if got := GetTerrainString(mapData, TilePos{Row: 0, Col: 0}); got != "TERRAIN_GRASS" {
		t.Errorf("GetTerrainString(0,0) = %q, want TERRAIN_GRASS", got)
	}
	if got := GetTerrainString(mapData, TilePos{Row: 0, Col: 1}); got != "TERRAIN_OCEAN" {
		t.Errorf("GetTerrainString(0,1) = %q, want TERRAIN_OCEAN", got)
	}
	// Out of bounds should not panic and should return ""
	if got := GetTerrainString(mapData, TilePos{Row: -1, Col: 0}); got != "" {
		t.Errorf("GetTerrainString(-1,0) = %q, want \"\"", got)
	}
	if got := GetTerrainString(mapData, TilePos{Row: 5, Col: 0}); got != "" {
		t.Errorf("GetTerrainString(5,0) = %q, want \"\"", got)
	}
	if got := GetTerrainString(mapData, TilePos{Row: 0, Col: 5}); got != "" {
		t.Errorf("GetTerrainString(0,5) = %q, want \"\"", got)
	}
}

func TestIsWaterTile(t *testing.T) {
	mapData := newTestMapData()

	if IsWaterTile(mapData, TilePos{Row: 0, Col: 0}) {
		t.Errorf("expected (0,0) grass tile to not be water")
	}
	if !IsWaterTile(mapData, TilePos{Row: 0, Col: 1}) {
		t.Errorf("expected (0,1) ocean tile to be water")
	}
	// out of bounds should be false, not panic
	if IsWaterTile(mapData, TilePos{Row: 10, Col: 10}) {
		t.Errorf("expected out of bounds tile to not be water")
	}
}

func TestTileHasCity(t *testing.T) {
	mapData := newTestMapData()

	if !TileHasCity(mapData, TilePos{Row: 0, Col: 0}) {
		t.Errorf("expected (0,0) to have a city")
	}
	if TileHasCity(mapData, TilePos{Row: 0, Col: 1}) {
		t.Errorf("expected (0,1) to not have a city")
	}
	if TileHasCity(mapData, TilePos{Row: -1, Col: 0}) {
		t.Errorf("expected out of bounds to not have a city")
	}
}

func TestTileHasMountain(t *testing.T) {
	mapData := newTestMapData()

	if TileHasMountain(mapData, TilePos{Row: 0, Col: 0}) {
		t.Errorf("expected (0,0) elevation 0 to not be a mountain")
	}
	if !TileHasMountain(mapData, TilePos{Row: 0, Col: 1}) {
		t.Errorf("expected (0,1) elevation 2 to be a mountain")
	}
	if TileHasMountain(mapData, TilePos{Row: 99, Col: 99}) {
		t.Errorf("expected out of bounds to not be a mountain")
	}
}

func TestIsInvalidTileOwner(t *testing.T) {
	tests := []struct {
		value int
		want  bool
	}{
		{0xFF, true},
		{0xFFFF, true},
		{-1, true},
		{0, false},
		{5, false},
	}
	for _, tt := range tests {
		if got := IsInvalidTileOwner(tt.value); got != tt.want {
			t.Errorf("IsInvalidTileOwner(%v) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestGetTileCivName(t *testing.T) {
	mapData := newTestMapData()

	if got := GetTileCivName(mapData, TilePos{Row: 0, Col: 0}); got != "CIVILIZATION_ROME" {
		t.Errorf("GetTileCivName(0,0) = %q, want CIVILIZATION_ROME", got)
	}
	// invalid owner should return ""
	if got := GetTileCivName(mapData, TilePos{Row: 0, Col: 1}); got != "" {
		t.Errorf("GetTileCivName(0,1) = %q, want \"\"", got)
	}
	// out of bounds should return "" without panicking
	if got := GetTileCivName(mapData, TilePos{Row: 50, Col: 50}); got != "" {
		t.Errorf("GetTileCivName(50,50) = %q, want \"\"", got)
	}
}

func TestGetPoliticalMapTileColor(t *testing.T) {
	mapData := newTestMapData()

	if got := GetPoliticalMapTileColor(mapData, TilePos{Row: 0, Col: 0}); got != "PLAYERCOLOR_RED" {
		t.Errorf("GetPoliticalMapTileColor(0,0) = %q, want PLAYERCOLOR_RED", got)
	}
	if got := GetPoliticalMapTileColor(mapData, TilePos{Row: 0, Col: 1}); got != "" {
		t.Errorf("GetPoliticalMapTileColor(0,1) = %q, want \"\"", got)
	}
	if got := GetPoliticalMapTileColor(mapData, TilePos{Row: 50, Col: 50}); got != "" {
		t.Errorf("GetPoliticalMapTileColor(50,50) = %q, want \"\"", got)
	}
}

// newTestReplayMapData builds a 1x2 map data grid for ApplyReplayEvent tests.
func newTestReplayMapData() *Civ5MapData {
	return &Civ5MapData{
		MapTileImprovements: [][]*Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, CityId: -1, Owner: -1},
				{X: 1, Y: 0, CityId: -1, Owner: -1},
			},
		},
	}
}

func TestApplyReplayEventCityFounded(t *testing.T) {
	mapData := newTestReplayMapData()
	event := Civ5ReplayEvent{
		TypeId: ReplayEventCityFounded,
		Text:   "Rome is founded.",
		Tiles:  []Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	nextCityId := ApplyReplayEvent(mapData, event, 0)

	tile := mapData.MapTileImprovements[0][0]
	if tile.CityId != 0 {
		t.Errorf("CityId = %d, want 0", tile.CityId)
	}
	if tile.CityName != "Rome" {
		t.Errorf("CityName = %q, want Rome", tile.CityName)
	}
	if nextCityId != 1 {
		t.Errorf("nextCityId = %d, want 1", nextCityId)
	}
}

func TestApplyReplayEventCityFoundedMultipleTiles(t *testing.T) {
	mapData := &Civ5MapData{
		MapTileImprovements: [][]*Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, CityId: -1},
				{X: 1, Y: 0, CityId: -1},
			},
		},
	}
	event := Civ5ReplayEvent{
		TypeId: ReplayEventCityFounded,
		Text:   "Paris is founded.",
		Tiles: []Civ5ReplayEventTile{
			{X: 0, Y: 0},
			{X: 1, Y: 0},
		},
	}

	nextCityId := ApplyReplayEvent(mapData, event, 5)

	// Each tile in the event gets its own incrementing city id.
	if mapData.MapTileImprovements[0][0].CityId != 5 {
		t.Errorf("tile 0 CityId = %d, want 5", mapData.MapTileImprovements[0][0].CityId)
	}
	if mapData.MapTileImprovements[0][1].CityId != 6 {
		t.Errorf("tile 1 CityId = %d, want 6", mapData.MapTileImprovements[0][1].CityId)
	}
	if nextCityId != 7 {
		t.Errorf("nextCityId = %d, want 7", nextCityId)
	}
}

func TestApplyReplayEventTilesClaimed(t *testing.T) {
	mapData := newTestReplayMapData()
	event := Civ5ReplayEvent{
		TypeId: ReplayEventTilesClaimed,
		CivId:  3,
		Tiles:  []Civ5ReplayEventTile{{X: 1, Y: 0}},
	}

	nextCityId := ApplyReplayEvent(mapData, event, 2)

	if mapData.MapTileImprovements[0][1].Owner != 3 {
		t.Errorf("Owner = %d, want 3", mapData.MapTileImprovements[0][1].Owner)
	}
	// Non-founding events must not advance the city id counter.
	if nextCityId != 2 {
		t.Errorf("nextCityId = %d, want unchanged 2", nextCityId)
	}
}

func TestApplyReplayEventCityTransferred(t *testing.T) {
	mapData := newTestReplayMapData()
	event := Civ5ReplayEvent{
		TypeId: ReplayEventCityTransferred,
		CivId:  7,
		Tiles:  []Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	ApplyReplayEvent(mapData, event, 0)

	if mapData.MapTileImprovements[0][0].Owner != 7 {
		t.Errorf("Owner = %d, want 7", mapData.MapTileImprovements[0][0].Owner)
	}
}

func TestApplyReplayEventTilesRazed(t *testing.T) {
	mapData := &Civ5MapData{
		MapTileImprovements: [][]*Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, CityId: 4, CityName: "Carthage", Owner: 2, RouteType: 0},
			},
		},
	}
	event := Civ5ReplayEvent{
		TypeId: ReplayEventTilesRazed,
		Tiles:  []Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	nextCityId := ApplyReplayEvent(mapData, event, 5)

	tile := mapData.MapTileImprovements[0][0]
	if tile.Owner != -1 {
		t.Errorf("Owner = %d, want -1", tile.Owner)
	}
	if tile.CityId != -1 {
		t.Errorf("CityId = %d, want -1", tile.CityId)
	}
	if tile.CityName != "" {
		t.Errorf("CityName = %q, want empty", tile.CityName)
	}
	if tile.RouteType != RouteRoad {
		t.Errorf("RouteType = %d, want %d (road)", tile.RouteType, RouteRoad)
	}
	// Razing does not found a city, so the counter is unaffected.
	if nextCityId != 5 {
		t.Errorf("nextCityId = %d, want unchanged 5", nextCityId)
	}
}

func TestApplyReplayEventUnknownTypeIsNoOp(t *testing.T) {
	mapData := newTestReplayMapData()
	event := Civ5ReplayEvent{
		TypeId: 99,
		Tiles:  []Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	nextCityId := ApplyReplayEvent(mapData, event, 3)

	tile := mapData.MapTileImprovements[0][0]
	if tile.CityId != -1 || tile.Owner != -1 {
		t.Errorf("unknown event type mutated tile: %+v", tile)
	}
	if nextCityId != 3 {
		t.Errorf("nextCityId = %d, want unchanged 3", nextCityId)
	}
}

func TestSetupCivPlayerDataRebuildsFromReplay(t *testing.T) {
	mapData := &Civ5MapData{
		Civ5PlayerData: []*Civ5PlayerData{}, // empty triggers rebuild
	}
	replayData := &Civ5ReplayData{
		IsReplayFile: true,
		AllCivs: []Civ5ReplayCiv{
			{Name: "CIVILIZATION_ROME", LongName: "PLAYERCOLOR_RED"},
			{Name: "Attila", LongName: ""}, // not a recognized civ/minor civ name
		},
	}

	SetupCivPlayerData(mapData, replayData)

	if len(mapData.Civ5PlayerData) != 2 {
		t.Fatalf("Civ5PlayerData length = %d, want 2", len(mapData.Civ5PlayerData))
	}
	if mapData.Civ5PlayerData[0].CivType != "CIVILIZATION_ROME" {
		t.Errorf("player 0 CivType = %q, want CIVILIZATION_ROME", mapData.Civ5PlayerData[0].CivType)
	}
	if mapData.Civ5PlayerData[0].TeamColor != "PLAYERCOLOR_RED" {
		t.Errorf("player 0 TeamColor = %q, want PLAYERCOLOR_RED", mapData.Civ5PlayerData[0].TeamColor)
	}
	// Names that aren't already CIVILIZATION_/MINOR_CIV get synthesized.
	if mapData.Civ5PlayerData[1].CivType != "CIVILIZATION_ATTILA" {
		t.Errorf("player 1 CivType = %q, want CIVILIZATION_ATTILA", mapData.Civ5PlayerData[1].CivType)
	}
	if mapData.Civ5PlayerData[1].TeamColor != "PLAYERCOLOR_ATTILA" {
		t.Errorf("player 1 TeamColor = %q, want PLAYERCOLOR_ATTILA", mapData.Civ5PlayerData[1].TeamColor)
	}
}

func TestSetupCivPlayerDataSwapsPlayerToIndexZero(t *testing.T) {
	mapData := &Civ5MapData{
		Civ5PlayerData: []*Civ5PlayerData{
			{Index: 0, CivType: "CIVILIZATION_ROME"},
			{Index: 1, CivType: "CIVILIZATION_GREECE"},
			{Index: 2, CivType: "CIVILIZATION_EGYPT"},
		},
	}
	replayData := &Civ5ReplayData{
		IsReplayFile: true,
		PlayerCiv:    "CIVILIZATION_EGYPT",
	}

	SetupCivPlayerData(mapData, replayData)

	if mapData.Civ5PlayerData[0].CivType != "CIVILIZATION_EGYPT" {
		t.Errorf("player 0 CivType = %q, want CIVILIZATION_EGYPT (swapped in)", mapData.Civ5PlayerData[0].CivType)
	}
	if mapData.Civ5PlayerData[2].CivType != "CIVILIZATION_ROME" {
		t.Errorf("player 2 CivType = %q, want CIVILIZATION_ROME (swapped out)", mapData.Civ5PlayerData[2].CivType)
	}
}

func TestSetupCivPlayerDataPlayerNotFoundLeavesOrderUnchanged(t *testing.T) {
	mapData := &Civ5MapData{
		Civ5PlayerData: []*Civ5PlayerData{
			{Index: 0, CivType: "CIVILIZATION_ROME"},
			{Index: 1, CivType: "CIVILIZATION_GREECE"},
		},
	}
	replayData := &Civ5ReplayData{
		IsReplayFile: true,
		PlayerCiv:    "CIVILIZATION_UNKNOWN",
	}

	SetupCivPlayerData(mapData, replayData)

	if mapData.Civ5PlayerData[0].CivType != "CIVILIZATION_ROME" {
		t.Errorf("player 0 CivType = %q, want unchanged CIVILIZATION_ROME", mapData.Civ5PlayerData[0].CivType)
	}
}

func TestPrepareReplayRejectsIncompatibleDataWithoutTouchingTheMap(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	replayData.AllReplayEvents = nil // makes the pair incompatible

	err := PrepareReplay(mapData, replayData)
	if err == nil {
		t.Fatal("PrepareReplay() with incompatible data = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "not compatible") {
		t.Errorf("PrepareReplay() error = %q, want it to mention compatibility", err)
	}
	if len(mapData.Civ5PlayerData) != 0 {
		t.Errorf("PrepareReplay() changed the map's player data despite failing validation")
	}
}

func TestPrepareReplaySetsUpPlayersAndCityOwners(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	mapData.CityOwnerIndexMap = nil // as loaded from a bare JSON export

	if err := PrepareReplay(mapData, replayData); err != nil {
		t.Fatalf("PrepareReplay() = %v", err)
	}
	if len(mapData.Civ5PlayerData) != 1 || mapData.Civ5PlayerData[0].CivType != "CIVILIZATION_ROME" {
		t.Errorf("Civ5PlayerData = %v, want one CIVILIZATION_ROME player", mapData.Civ5PlayerData)
	}
	if got, ok := mapData.CityOwnerIndexMap[0]; !ok || got != 0 {
		t.Errorf("CityOwnerIndexMap[0] = %d (present %v), want identity 0", got, ok)
	}
}

// A 3 wide x 2 high map whose tiles are all distinct, so a swapped row and column reads a different tile or none.
func newAccessorTestMap() *Civ5MapData {
	physical := [][]*Civ5MapTilePhysical{
		{{Elevation: 10}, {Elevation: 11}, {Elevation: 12}},
		{{Elevation: 20}, {Elevation: 21}, {Elevation: 22}},
	}
	improvements := [][]*Civ5MapTileImprovement{
		{{CityId: 10}, {CityId: 11}, {CityId: 12}},
		{{CityId: 20}, {CityId: 21}, {CityId: 22}},
	}
	return &Civ5MapData{MapTiles: physical, MapTileImprovements: improvements}
}

func TestPhysicalTileReadsRowThenColumn(t *testing.T) {
	m := newAccessorTestMap()
	if got := m.PhysicalTile(TilePos{Row: 1, Col: 2}); got != m.MapTiles[1][2] || got.Elevation != 22 {
		t.Errorf("PhysicalTile(row 1, col 2) = %+v, want the tile at MapTiles[1][2]", got)
	}
	if got := m.PhysicalTile(TilePos{Row: 0, Col: 1}); got.Elevation != 11 {
		t.Errorf("PhysicalTile(row 0, col 1).Elevation = %d, want 11", got.Elevation)
	}
	for _, pos := range []TilePos{{Row: 2, Col: 0}, {Row: 0, Col: 3}, {Row: -1, Col: 0}, {Row: 0, Col: -1}, {Row: 2, Col: 1}} {
		if got := m.PhysicalTile(pos); got != nil {
			t.Errorf("PhysicalTile(%+v) = %+v, want nil off the 2 x 3 map", pos, got)
		}
	}
}

func TestTileImprovementReadsRowThenColumn(t *testing.T) {
	m := newAccessorTestMap()
	if got := m.TileImprovement(TilePos{Row: 1, Col: 2}); got != m.MapTileImprovements[1][2] || got.CityId != 22 {
		t.Errorf("TileImprovement(row 1, col 2) = %+v, want the record at MapTileImprovements[1][2]", got)
	}
	for _, pos := range []TilePos{{Row: 2, Col: 0}, {Row: 0, Col: 3}, {Row: -1, Col: 0}, {Row: 0, Col: -1}} {
		if got := m.TileImprovement(pos); got != nil {
			t.Errorf("TileImprovement(%+v) = %+v, want nil off the 2 x 3 map", pos, got)
		}
	}
}

func TestTileImprovementIsNilWhenTheMapHasNoImprovementData(t *testing.T) {
	m := newAccessorTestMap()
	m.MapTileImprovements = nil
	if got := m.TileImprovement(TilePos{Row: 0, Col: 0}); got != nil {
		t.Errorf("TileImprovement on a map with no improvement data = %+v, want nil", got)
	}
}

func TestReplayEventTilePosIsRowFromYAndColumnFromX(t *testing.T) {
	if got, want := (Civ5ReplayEventTile{X: 4, Y: 7}).Pos(), (TilePos{Row: 7, Col: 4}); got != want {
		t.Errorf("Civ5ReplayEventTile{X: 4, Y: 7}.Pos() = %+v, want %+v", got, want)
	}
}
