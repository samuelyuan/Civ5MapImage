package raster

import "testing"

func TestBorderProbesAreWithinReachExcludingSelf(t *testing.T) {
	probes := BorderProbes(2)
	for _, p := range probes {
		d := max(p.X, -p.X) + max(p.Y, -p.Y)
		if d == 0 || d > 2 {
			t.Errorf("probe %v has Manhattan distance %d, want 1 to 2", p, d)
		}
	}
	if got := len(probes); got != 12 { // the diamond of Manhattan distance 1 or 2 around a pixel
		t.Errorf("BorderProbes(2) has %d probes, want 12", got)
	}
}

// a 2-row grid: region 0 fills row 0, region 1 fills row 1.
func gridAt(ids [][]int32) func(x, y int) int32 {
	return func(x, y int) int32 {
		if y < 0 || y >= len(ids) || x < 0 || x >= len(ids[y]) {
			return -1
		}
		return ids[y][x]
	}
}

func TestIsOutlineOnlyChecksRightAndDown(t *testing.T) {
	at := gridAt([][]int32{{0, 0, 1}, {0, 0, 0}})
	if IsOutline(at, 0, 0, 0) {
		t.Error("pixel (0,0): right and down are both region 0, want not outline")
	}
	if !IsOutline(at, 1, 0, 0) {
		t.Error("pixel (1,0): right is region 1, want outline")
	}
	if IsOutline(at, 2, 1, 0) {
		t.Error("pixel (2,1): off-grid right and down don't count, want not outline")
	}
}

func TestNearOtherFindsAnotherRegionWithinReach(t *testing.T) {
	at := gridAt([][]int32{{0, 0, 0, 1}})
	probes := BorderProbes(1)

	if !NearOther(at, probes, 2, 0, 0) {
		t.Error("pixel (2,0) is next to region 1, want near")
	}
	if NearOther(at, probes, 0, 0, 0) {
		t.Error("pixel (0,0) is 3 px from region 1 and off-grid pixels don't count, want not near")
	}
}
