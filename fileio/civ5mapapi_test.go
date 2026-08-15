package fileio

import (
	"image/color"
	"math"
	"testing"
)

func TestGetNeighborsOddRow(t *testing.T) {
	// y=1 is odd, should use NeighborOdd offsets
	got := GetNeighbors(5, 1)
	want := [6][2]int{{6, 2}, {5, 2}, {4, 1}, {5, 0}, {6, 0}, {6, 1}}
	if got != want {
		t.Errorf("GetNeighbors(5, 1) = %v, want %v", got, want)
	}
}

func TestGetNeighborsEvenRow(t *testing.T) {
	// y=2 is even, should use NeighborEven offsets
	got := GetNeighbors(5, 2)
	want := [6][2]int{{5, 3}, {4, 3}, {4, 2}, {4, 1}, {5, 1}, {6, 2}}
	if got != want {
		t.Errorf("GetNeighbors(5, 2) = %v, want %v", got, want)
	}
}

func TestGetImagePosition(t *testing.T) {
	x, y := GetImagePosition(0, 0, 16.0)
	angle := math.Pi / 6
	wantX := 16.0 * 1.5
	wantY := 16.0
	if math.Abs(x-wantX) > 1e-9 || math.Abs(y-wantY) > 1e-9 {
		t.Errorf("GetImagePosition(0, 0, 16.0) = (%v, %v), want (%v, %v)", x, y, wantX, wantY)
	}

	// Odd row should shift x by radius*cos(angle)
	xOdd, _ := GetImagePosition(1, 0, 16.0)
	wantXOdd := wantX + 16.0*math.Cos(angle)
	if math.Abs(xOdd-wantXOdd) > 1e-9 {
		t.Errorf("GetImagePosition(1, 0, 16.0) x = %v, want %v", xOdd, wantXOdd)
	}
}

func TestGetPhysicalMapTileColor(t *testing.T) {
	tests := []struct {
		terrain string
		want    color.RGBA
	}{
		{"TERRAIN_GRASS", color.RGBA{105, 125, 54, 255}},
		{"TERRAIN_OCEAN", color.RGBA{47, 74, 93, 255}},
		{"TERRAIN_UNKNOWN", color.RGBA{0, 0, 0, 255}},
		{"", color.RGBA{0, 0, 0, 255}},
	}
	for _, tt := range tests {
		if got := GetPhysicalMapTileColor(tt.terrain); got != tt.want {
			t.Errorf("GetPhysicalMapTileColor(%q) = %v, want %v", tt.terrain, got, tt.want)
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

	if got := GetTerrainString(mapData, 0, 0); got != "TERRAIN_GRASS" {
		t.Errorf("GetTerrainString(0,0) = %q, want TERRAIN_GRASS", got)
	}
	if got := GetTerrainString(mapData, 0, 1); got != "TERRAIN_OCEAN" {
		t.Errorf("GetTerrainString(0,1) = %q, want TERRAIN_OCEAN", got)
	}
	// Out of bounds should not panic and should return ""
	if got := GetTerrainString(mapData, -1, 0); got != "" {
		t.Errorf("GetTerrainString(-1,0) = %q, want \"\"", got)
	}
	if got := GetTerrainString(mapData, 5, 0); got != "" {
		t.Errorf("GetTerrainString(5,0) = %q, want \"\"", got)
	}
	if got := GetTerrainString(mapData, 0, 5); got != "" {
		t.Errorf("GetTerrainString(0,5) = %q, want \"\"", got)
	}
}

func TestIsWaterTile(t *testing.T) {
	mapData := newTestMapData()

	if IsWaterTile(mapData, 0, 0) {
		t.Errorf("expected (0,0) grass tile to not be water")
	}
	if !IsWaterTile(mapData, 0, 1) {
		t.Errorf("expected (0,1) ocean tile to be water")
	}
	// out of bounds should be false, not panic
	if IsWaterTile(mapData, 10, 10) {
		t.Errorf("expected out of bounds tile to not be water")
	}
}

func TestTileHasCity(t *testing.T) {
	mapData := newTestMapData()

	if !TileHasCity(mapData, 0, 0) {
		t.Errorf("expected (0,0) to have a city")
	}
	if TileHasCity(mapData, 0, 1) {
		t.Errorf("expected (0,1) to not have a city")
	}
	if TileHasCity(mapData, -1, 0) {
		t.Errorf("expected out of bounds to not have a city")
	}
}

func TestTileHasMountain(t *testing.T) {
	mapData := newTestMapData()

	if TileHasMountain(mapData, 0, 0) {
		t.Errorf("expected (0,0) elevation 0 to not be a mountain")
	}
	if !TileHasMountain(mapData, 0, 1) {
		t.Errorf("expected (0,1) elevation 2 to be a mountain")
	}
	if TileHasMountain(mapData, 99, 99) {
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

	if got := GetTileCivName(mapData, 0, 0); got != "CIVILIZATION_ROME" {
		t.Errorf("GetTileCivName(0,0) = %q, want CIVILIZATION_ROME", got)
	}
	// invalid owner should return ""
	if got := GetTileCivName(mapData, 0, 1); got != "" {
		t.Errorf("GetTileCivName(0,1) = %q, want \"\"", got)
	}
	// out of bounds should return "" without panicking
	if got := GetTileCivName(mapData, 50, 50); got != "" {
		t.Errorf("GetTileCivName(50,50) = %q, want \"\"", got)
	}
}

func TestGetPoliticalMapTileColor(t *testing.T) {
	mapData := newTestMapData()

	if got := GetPoliticalMapTileColor(mapData, 0, 0); got != "PLAYERCOLOR_RED" {
		t.Errorf("GetPoliticalMapTileColor(0,0) = %q, want PLAYERCOLOR_RED", got)
	}
	if got := GetPoliticalMapTileColor(mapData, 0, 1); got != "" {
		t.Errorf("GetPoliticalMapTileColor(0,1) = %q, want \"\"", got)
	}
	if got := GetPoliticalMapTileColor(mapData, 50, 50); got != "" {
		t.Errorf("GetPoliticalMapTileColor(50,50) = %q, want \"\"", got)
	}
}
