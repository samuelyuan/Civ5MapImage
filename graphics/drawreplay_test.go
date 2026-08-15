package graphics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// newTestReplayMapData builds a 1x2 map data grid for applyReplayEvent tests.
func newTestReplayMapData() *fileio.Civ5MapData {
	return &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, CityId: -1, Owner: -1},
				{X: 1, Y: 0, CityId: -1, Owner: -1},
			},
		},
	}
}

func TestApplyReplayEventCityFounded(t *testing.T) {
	mapData := newTestReplayMapData()
	event := fileio.Civ5ReplayEvent{
		TypeId: ReplayEventCityFounded,
		Text:   "Rome is founded.",
		Tiles:  []fileio.Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	nextCityId := applyReplayEvent(mapData, event, 0)

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
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, CityId: -1},
				{X: 1, Y: 0, CityId: -1},
			},
		},
	}
	event := fileio.Civ5ReplayEvent{
		TypeId: ReplayEventCityFounded,
		Text:   "Paris is founded.",
		Tiles: []fileio.Civ5ReplayEventTile{
			{X: 0, Y: 0},
			{X: 1, Y: 0},
		},
	}

	nextCityId := applyReplayEvent(mapData, event, 5)

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
	event := fileio.Civ5ReplayEvent{
		TypeId: ReplayEventTilesClaimed,
		CivId:  3,
		Tiles:  []fileio.Civ5ReplayEventTile{{X: 1, Y: 0}},
	}

	nextCityId := applyReplayEvent(mapData, event, 2)

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
	event := fileio.Civ5ReplayEvent{
		TypeId: ReplayEventCityTransferred,
		CivId:  7,
		Tiles:  []fileio.Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	applyReplayEvent(mapData, event, 0)

	if mapData.MapTileImprovements[0][0].Owner != 7 {
		t.Errorf("Owner = %d, want 7", mapData.MapTileImprovements[0][0].Owner)
	}
}

func TestApplyReplayEventTilesRazed(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, CityId: 4, CityName: "Carthage", Owner: 2, RouteType: 0},
			},
		},
	}
	event := fileio.Civ5ReplayEvent{
		TypeId: ReplayEventTilesRazed,
		Tiles:  []fileio.Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	nextCityId := applyReplayEvent(mapData, event, 5)

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
	if tile.RouteType != 2 {
		t.Errorf("RouteType = %d, want 2 (road)", tile.RouteType)
	}
	// Razing does not found a city, so the counter is unaffected.
	if nextCityId != 5 {
		t.Errorf("nextCityId = %d, want unchanged 5", nextCityId)
	}
}

func TestApplyReplayEventUnknownTypeIsNoOp(t *testing.T) {
	mapData := newTestReplayMapData()
	event := fileio.Civ5ReplayEvent{
		TypeId: 99,
		Tiles:  []fileio.Civ5ReplayEventTile{{X: 0, Y: 0}},
	}

	nextCityId := applyReplayEvent(mapData, event, 3)

	tile := mapData.MapTileImprovements[0][0]
	if tile.CityId != -1 || tile.Owner != -1 {
		t.Errorf("unknown event type mutated tile: %+v", tile)
	}
	if nextCityId != 3 {
		t.Errorf("nextCityId = %d, want unchanged 3", nextCityId)
	}
}

func TestSetupCivPlayerDataRebuildsFromReplay(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		Civ5PlayerData: []*fileio.Civ5PlayerData{}, // empty triggers rebuild
	}
	replayData := &fileio.Civ5ReplayData{
		IsReplayFile: true,
		AllCivs: []fileio.Civ5ReplayCiv{
			{Name: "CIVILIZATION_ROME", LongName: "PLAYERCOLOR_RED"},
			{Name: "Attila", LongName: ""}, // not a recognized civ/minor civ name
		},
	}

	setupCivPlayerData(mapData, replayData)

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
	mapData := &fileio.Civ5MapData{
		Civ5PlayerData: []*fileio.Civ5PlayerData{
			{Index: 0, CivType: "CIVILIZATION_ROME"},
			{Index: 1, CivType: "CIVILIZATION_GREECE"},
			{Index: 2, CivType: "CIVILIZATION_EGYPT"},
		},
	}
	replayData := &fileio.Civ5ReplayData{
		IsReplayFile: true,
		PlayerCiv:    "CIVILIZATION_EGYPT",
	}

	setupCivPlayerData(mapData, replayData)

	if mapData.Civ5PlayerData[0].CivType != "CIVILIZATION_EGYPT" {
		t.Errorf("player 0 CivType = %q, want CIVILIZATION_EGYPT (swapped in)", mapData.Civ5PlayerData[0].CivType)
	}
	if mapData.Civ5PlayerData[2].CivType != "CIVILIZATION_ROME" {
		t.Errorf("player 2 CivType = %q, want CIVILIZATION_ROME (swapped out)", mapData.Civ5PlayerData[2].CivType)
	}
}

func TestSetupCivPlayerDataPlayerNotFoundLeavesOrderUnchanged(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		Civ5PlayerData: []*fileio.Civ5PlayerData{
			{Index: 0, CivType: "CIVILIZATION_ROME"},
			{Index: 1, CivType: "CIVILIZATION_GREECE"},
		},
	}
	replayData := &fileio.Civ5ReplayData{
		IsReplayFile: true,
		PlayerCiv:    "CIVILIZATION_UNKNOWN",
	}

	setupCivPlayerData(mapData, replayData)

	if mapData.Civ5PlayerData[0].CivType != "CIVILIZATION_ROME" {
		t.Errorf("player 0 CivType = %q, want unchanged CIVILIZATION_ROME", mapData.Civ5PlayerData[0].CivType)
	}
}

// newValidReplayFixtures returns a minimal mapData/replayData pair that passes
// ValidateReplayCompatibility, for use as a baseline in the tests below.
func newValidReplayFixtures() (*fileio.Civ5MapData, *fileio.Civ5ReplayData) {
	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS"},
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{TerrainType: 0, Elevation: 0}},
		},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{Owner: -1, CityId: -1, RouteType: 255}},
		},
		CityOwnerIndexMap: map[int]int{},
	}
	replayData := &fileio.Civ5ReplayData{
		IsReplayFile: true,
		PlayerCiv:    "CIVILIZATION_ROME",
		AllCivs:      []fileio.Civ5ReplayCiv{{Name: "CIVILIZATION_ROME"}},
		AllReplayEvents: []fileio.Civ5ReplayEvent{
			{Turn: 1, TypeId: ReplayEventTilesClaimed, CivId: 0, Tiles: []fileio.Civ5ReplayEventTile{{X: 0, Y: 0}}},
		},
		// Matches the 1x1 map above, as a real .civ5replay file's embedded dimensions would.
		MapWidth:  1,
		MapHeight: 1,
	}
	return mapData, replayData
}

func TestValidateReplayCompatibilityValid(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	if err := ValidateReplayCompatibility(mapData, replayData); err != nil {
		t.Errorf("ValidateReplayCompatibility() = %v, want nil", err)
	}
}

func TestValidateReplayCompatibilityNilInputs(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()

	if err := ValidateReplayCompatibility(nil, replayData); err == nil {
		t.Error("ValidateReplayCompatibility(nil map, ...) = nil, want error")
	}
	if err := ValidateReplayCompatibility(mapData, nil); err == nil {
		t.Error("ValidateReplayCompatibility(..., nil replay) = nil, want error")
	}
}

func TestValidateReplayCompatibilityEmptyMapTiles(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	mapData.MapTiles = nil

	if err := ValidateReplayCompatibility(mapData, replayData); err == nil {
		t.Error("ValidateReplayCompatibility() with no map tiles = nil, want error")
	}
}

func TestValidateReplayCompatibilityMissingImprovements(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	mapData.MapTileImprovements = nil

	err := ValidateReplayCompatibility(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayCompatibility() with no tile improvements = nil, want error")
	}
	if !strings.Contains(err.Error(), "tile improvement data") {
		t.Errorf("ValidateReplayCompatibility() error = %q, want it to mention missing tile improvement data", err)
	}
}

func TestValidateReplayCompatibilityNoEvents(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	replayData.AllReplayEvents = nil

	err := ValidateReplayCompatibility(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayCompatibility() with no events = nil, want error")
	}
	if !strings.Contains(err.Error(), "no events") {
		t.Errorf("ValidateReplayCompatibility() error = %q, want it to mention no events", err)
	}
}

func TestValidateReplayCompatibilityTileOutOfBounds(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// The map is 1x1, so tile (5, 5) is out of bounds.
	replayData.AllReplayEvents = []fileio.Civ5ReplayEvent{
		{Turn: 3, TypeId: ReplayEventTilesClaimed, Tiles: []fileio.Civ5ReplayEventTile{{X: 5, Y: 5}}},
	}

	err := ValidateReplayCompatibility(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayCompatibility() with out-of-bounds tile = nil, want error")
	}
	if !strings.Contains(err.Error(), "turn 3") {
		t.Errorf("ValidateReplayCompatibility() error = %q, want it to mention turn 3", err)
	}
}

func TestValidateReplayCompatibilityMapDimensionMismatch(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// The .civ5replay file says it was recorded on a 128x80 map, but the provided map is 1x1.
	replayData.MapWidth = 128
	replayData.MapHeight = 80

	err := ValidateReplayCompatibility(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayCompatibility() with mismatched map dimensions = nil, want error")
	}
	for _, want := range []string{"128x80", "1x1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ValidateReplayCompatibility() error = %q, want it to mention %q", err, want)
		}
	}
}

func TestValidateReplayCompatibilityMapDimensionMatch(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	replayData.MapWidth = 1
	replayData.MapHeight = 1

	if err := ValidateReplayCompatibility(mapData, replayData); err != nil {
		t.Errorf("ValidateReplayCompatibility() with matching map dimensions = %v, want nil", err)
	}
}

func TestValidateReplayCompatibilityUnknownDimensionsFallsBackToTileScan(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// Simulate a replay converted from a .civ5save file, which doesn't carry map dimensions.
	replayData.MapWidth = 0
	replayData.MapHeight = 0
	replayData.AllReplayEvents = []fileio.Civ5ReplayEvent{
		{Turn: 3, TypeId: ReplayEventTilesClaimed, Tiles: []fileio.Civ5ReplayEventTile{{X: 5, Y: 5}}},
	}

	err := ValidateReplayCompatibility(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayCompatibility() with unknown dimensions and out-of-bounds tile = nil, want error")
	}
	if !strings.Contains(err.Error(), "turn 3") {
		t.Errorf("ValidateReplayCompatibility() error = %q, want the tile-scan fallback to catch it (mentioning turn 3)", err)
	}
}

func TestDrawReplayReturnsErrorForIncompatibleData(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	replayData.AllReplayEvents = nil // makes the pair incompatible

	outputPath := filepath.Join(t.TempDir(), "replay.gif")
	err := DrawReplay(mapData, replayData, outputPath)
	if err == nil {
		t.Fatal("DrawReplay() with incompatible data = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "not compatible") {
		t.Errorf("DrawReplay() error = %q, want it to mention compatibility", err)
	}
	if _, statErr := os.Stat(outputPath); statErr == nil {
		t.Error("DrawReplay() wrote an output file despite failing validation")
	}
}

func TestDrawReplaySucceedsAndWritesFile(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()

	outputPath := filepath.Join(t.TempDir(), "replay.gif")
	if err := DrawReplay(mapData, replayData, outputPath); err != nil {
		t.Fatalf("DrawReplay() returned error: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("DrawReplay() did not write an output file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("DrawReplay() wrote an empty output file")
	}
}

func TestDrawReplayHandlesNilCityOwnerIndexMap(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// Simulate a map loaded from a bare JSON export where this map was never initialized.
	mapData.CityOwnerIndexMap = nil

	outputPath := filepath.Join(t.TempDir(), "replay.gif")
	if err := DrawReplay(mapData, replayData, outputPath); err != nil {
		t.Fatalf("DrawReplay() with nil CityOwnerIndexMap returned error: %v", err)
	}
}
