package raster

import (
	"image"
	"testing"
)

func TestBorderProbesAreWithinReachExcludingSelf(t *testing.T) {
	probes := BorderProbes(2)
	for _, p := range probes {
		d := max(p[0], -p[0]) + max(p[1], -p[1])
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

func TestRimReachTagsTheNeighborItFound(t *testing.T) {
	at := gridAt([][]int32{{0, 1}})
	probes := BorderProbes(1)
	neighbors := []int32{1, -1} // region 1 is neighbor 0, no neighbor 1

	reach := RimReach(at, probes, 0, 0, 0, neighbors)
	if !reach.Intersects(NeighborMask(1)) {
		t.Errorf("reach = %v, want neighbor 0 (bit 0) set", reach)
	}
	if reach.Intersects(NeighborMask(2)) {
		t.Errorf("reach = %v, want neighbor 1 (bit 1) unset", reach)
	}
}

func TestBuildRowSpansAndInterior(t *testing.T) {
	// region 0 is a solid 5x5 block: with reach 2, only its center pixel is far enough from every edge.
	ids := make([][]int32, 5)
	for y := range ids {
		ids[y] = []int32{0, 0, 0, 0, 0}
	}
	spans := BuildRowSpans(gridAt(ids), 0, image.Rect(0, 0, 5, 5))

	if lo, hi := spans.Interior(2, 2); lo != 2 || hi != 3 {
		t.Errorf("row 2 (the center row): interior = [%d, %d), want [2, 3) (just the center column)", lo, hi)
	}
	if lo, hi := spans.Interior(0, 2); lo != 0 || hi != 0 {
		t.Errorf("row 0 (at the top edge): interior = [%d, %d), want empty", lo, hi)
	}
	if lo, hi := spans.Interior(1, 2); lo != 0 || hi != 0 {
		t.Errorf("row 1 (1 row from the top edge, reach 2): interior = [%d, %d), want empty", lo, hi)
	}
}
