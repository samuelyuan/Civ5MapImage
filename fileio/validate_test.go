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
