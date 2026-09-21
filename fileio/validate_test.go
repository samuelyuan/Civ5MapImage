package fileio

import (
	"strings"
	"testing"
)

// newValidReplayFixtures returns a minimal mapData/replayData pair that passes
// ValidateReplayRenderable, for use as a baseline in the tests below.
func newValidReplayFixtures() (*Civ5MapData, *Civ5ReplayData) {
	mapData := &Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS"},
		MapTiles: [][]*Civ5MapTilePhysical{
			{{TerrainType: 0, Elevation: 0}},
		},
		MapTileImprovements: [][]*Civ5MapTileImprovement{
			{{Owner: -1, CityId: -1, RouteType: 255}},
		},
		CityOwnerIndexMap: map[int]int{},
	}
	replayData := &Civ5ReplayData{
		IsReplayFile: true,
		PlayerCiv:    "CIVILIZATION_ROME",
		AllCivs:      []Civ5ReplayCiv{{Name: "CIVILIZATION_ROME"}},
		AllReplayEvents: []Civ5ReplayEvent{
			{Turn: 1, TypeId: ReplayEventTilesClaimed, CivId: 0, Tiles: []Civ5ReplayEventTile{{X: 0, Y: 0}}},
		},
		// Matches the 1x1 map above, as a real .civ5replay file's embedded dimensions would.
		MapWidth:  1,
		MapHeight: 1,
	}
	return mapData, replayData
}

func TestValidateReplayRenderableValid(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	if err := ValidateReplayRenderable(mapData, replayData); err != nil {
		t.Errorf("ValidateReplayRenderable() = %v, want nil", err)
	}
}

func TestValidateReplayRenderableNilInputs(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()

	if err := ValidateReplayRenderable(nil, replayData); err == nil {
		t.Error("ValidateReplayRenderable(nil map, ...) = nil, want error")
	}
	if err := ValidateReplayRenderable(mapData, nil); err == nil {
		t.Error("ValidateReplayRenderable(..., nil replay) = nil, want error")
	}
}

func TestValidateReplayRenderableEmptyMapTiles(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	mapData.MapTiles = nil

	if err := ValidateReplayRenderable(mapData, replayData); err == nil {
		t.Error("ValidateReplayRenderable() with no map tiles = nil, want error")
	}
}

func TestValidateReplayRenderableMissingImprovements(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	mapData.MapTileImprovements = nil

	err := ValidateReplayRenderable(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayRenderable() with no tile improvements = nil, want error")
	}
	if !strings.Contains(err.Error(), "tile improvement data") {
		t.Errorf("ValidateReplayRenderable() error = %q, want it to mention missing tile improvement data", err)
	}
}

func TestValidateReplayRenderableNoEvents(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	replayData.AllReplayEvents = nil

	err := ValidateReplayRenderable(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayRenderable() with no events = nil, want error")
	}
	if !strings.Contains(err.Error(), "no events") {
		t.Errorf("ValidateReplayRenderable() error = %q, want it to mention no events", err)
	}
}

func TestValidateReplayRenderableTileOutOfBounds(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// The map is 1x1, so tile (5, 5) is out of bounds.
	replayData.AllReplayEvents = []Civ5ReplayEvent{
		{Turn: 3, TypeId: ReplayEventTilesClaimed, Tiles: []Civ5ReplayEventTile{{X: 5, Y: 5}}},
	}

	err := ValidateReplayRenderable(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayRenderable() with out-of-bounds tile = nil, want error")
	}
	if !strings.Contains(err.Error(), "turn 3") {
		t.Errorf("ValidateReplayRenderable() error = %q, want it to mention turn 3", err)
	}
}

func TestValidateReplayRenderableMapDimensionMismatch(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// The .civ5replay file says it was recorded on a 128x80 map, but the provided map is 1x1.
	replayData.MapWidth = 128
	replayData.MapHeight = 80

	err := ValidateReplayRenderable(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayRenderable() with mismatched map dimensions = nil, want error")
	}
	for _, want := range []string{"128x80", "1x1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ValidateReplayRenderable() error = %q, want it to mention %q", err, want)
		}
	}
}

func TestValidateReplayRenderableMapDimensionMatch(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	replayData.MapWidth = 1
	replayData.MapHeight = 1

	if err := ValidateReplayRenderable(mapData, replayData); err != nil {
		t.Errorf("ValidateReplayRenderable() with matching map dimensions = %v, want nil", err)
	}
}

func TestValidateReplayRenderableUnknownDimensionsFallsBackToTileScan(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// Simulate a replay converted from a .civ5save file, which doesn't carry map dimensions.
	replayData.MapWidth = 0
	replayData.MapHeight = 0
	replayData.AllReplayEvents = []Civ5ReplayEvent{
		{Turn: 3, TypeId: ReplayEventTilesClaimed, Tiles: []Civ5ReplayEventTile{{X: 5, Y: 5}}},
	}

	err := ValidateReplayRenderable(mapData, replayData)
	if err == nil {
		t.Fatal("ValidateReplayRenderable() with unknown dimensions and out-of-bounds tile = nil, want error")
	}
	if !strings.Contains(err.Error(), "turn 3") {
		t.Errorf("ValidateReplayRenderable() error = %q, want the tile-scan fallback to catch it (mentioning turn 3)", err)
	}
}

// newCrossCheckFixtures returns a 3 wide x 2 high map and a replay whose tiles all agree with it. The map isn't square,
// and every tile differs, so swapping rows with columns or width with height changes the result.
func newCrossCheckFixtures() (*Civ5MapData, *Civ5ReplayData) {
	mapData := &Civ5MapData{
		MapHeader:   Civ5MapHeader{Width: 3, Height: 2},
		TerrainList: []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"},
		MapTiles: [][]*Civ5MapTilePhysical{
			{{TerrainType: 1}, {TerrainType: 0, Elevation: 2}, {TerrainType: 0, Elevation: 1}},
			{{TerrainType: 0}, {TerrainType: 0, Elevation: 2, RiverData: 1}, {TerrainType: 1}},
		},
	}
	replayData := &Civ5ReplayData{
		MapWidth:  3,
		MapHeight: 2,
		Tiles: []Civ5ReplayTile{ // row-major: PlotType 3 ocean, 0 mountain, 1 hills, 2 land
			{PlotType: 3, TerrainType: 1}, {PlotType: 0, TerrainType: 0}, {PlotType: 1, TerrainType: 0},
			{PlotType: 2, TerrainType: 0}, {PlotType: 0, TerrainType: 0, RiverBits: 4}, {PlotType: 3, TerrainType: 1},
		},
	}
	return mapData, replayData
}

func findResult(t *testing.T, results []ValidationResult, namePrefix string) ValidationResult {
	t.Helper()
	for _, r := range results {
		if strings.HasPrefix(r.Name, namePrefix) {
			return r
		}
	}
	t.Fatalf("no %q result in %v", namePrefix, results)
	return ValidationResult{}
}

func TestValidateMapAndReplayAcceptsAMatchingNonSquareMap(t *testing.T) {
	mapData, replayData := newCrossCheckFixtures()
	results := ValidateMapAndReplay(mapData, replayData)
	for _, prefix := range []string{"grid dimensions", "tile PlotType", "tile TerrainType", "river edges"} {
		if r := findResult(t, results, prefix); r.Passed != r.Total || r.Total == 0 {
			t.Errorf("%s: passed %d of %d, want all of a nonzero total", r.Name, r.Passed, r.Total)
		}
	}
}

func TestValidateMapAndReplayCountsEachMismatchedTile(t *testing.T) {
	mapData, replayData := newCrossCheckFixtures()
	replayData.Tiles[4].PlotType = 2    // the map's tile at row 1, col 1 is a mountain
	replayData.Tiles[5].TerrainType = 0 // the map's tile at row 1, col 2 is ocean
	replayData.Tiles[2].RiverBits = 2   // a river edge (SE) the map's tile at row 0, col 2 doesn't have

	results := ValidateMapAndReplay(mapData, replayData)
	if r := findResult(t, results, "tile PlotType"); r.Passed != 5 || r.Total != 6 {
		t.Errorf("PlotType: passed %d of %d, want 5 of 6", r.Passed, r.Total)
	}
	if r := findResult(t, results, "tile TerrainType"); r.Passed != 5 || r.Total != 6 {
		t.Errorf("TerrainType: passed %d of %d, want 5 of 6", r.Passed, r.Total)
	}
	if r := findResult(t, results, "river edges"); r.Passed != 17 || r.Total != 18 {
		t.Errorf("river edges: passed %d of %d, want 17 of 18", r.Passed, r.Total)
	}
}

// A replay recorded on the transposed grid (2 wide x 3 high) has the same tile count but is a different map.
func TestValidateMapAndReplayRejectsTransposedDimensions(t *testing.T) {
	mapData, replayData := newCrossCheckFixtures()
	replayData.MapWidth, replayData.MapHeight = 2, 3

	results := ValidateMapAndReplay(mapData, replayData)
	if len(results) != 1 {
		t.Fatalf("got %d results, want only the dimensions check when they differ: %v", len(results), results)
	}
	if r := findResult(t, results, "grid dimensions"); r.Passed != 0 {
		t.Errorf("grid dimensions passed = %d, want 0 for a transposed grid", r.Passed)
	}
}
