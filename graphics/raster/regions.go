package raster

import (
	"image"
	"math"
)

// NeighborMask is a set of a region's neighbors; bit j means the j-th entry of whatever neighbor list the caller uses.
type NeighborMask uint8

func (m *NeighborMask) Add(j int)                     { *m |= 1 << j }
func (m NeighborMask) Intersects(o NeighborMask) bool { return m&o != 0 }

// Pixel is a pixel coordinate.
type Pixel struct{ X, Y int32 }

// RimPixel is a pixel near another region; Reach holds which neighbors it's near.
type RimPixel struct {
	X, Y  int32
	Reach NeighborMask
}

// RowSpans holds, per row of a region's bounds, the pixels [lo, hi) it covers.
type RowSpans struct{ lo, hi []int }

// BuildRowSpans returns the spans of region id in each row of bounds, using at to test each pixel.
func BuildRowSpans(at func(x, y int) int32, id int32, bounds image.Rectangle) RowSpans {
	spans := RowSpans{lo: make([]int, bounds.Dy()), hi: make([]int, bounds.Dy())}
	for row := range spans.lo {
		y := bounds.Min.Y + row
		spans.lo[row], spans.hi[row] = math.MaxInt, math.MinInt
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if at(x, y) == id {
				spans.lo[row] = min(spans.lo[row], x)
				spans.hi[row] = x + 1
			}
		}
	}
	return spans
}

// Interior returns the pixels [lo, hi) of a row whose every neighbor within reach is in the region; empty near the region's top or bottom.
func (s RowSpans) Interior(row, reach int) (lo, hi int) {
	lo, hi = math.MinInt, math.MaxInt
	for dy := -reach; dy <= reach; dy++ {
		near := row + dy
		if near < 0 || near >= len(s.lo) || s.lo[near] > s.hi[near] {
			return 0, 0
		}
		r := reach - max(dy, -dy)
		lo, hi = max(lo, s.lo[near]+r), min(hi, s.hi[near]-r)
	}
	return lo, hi
}

// IsOutline reports whether pixel (x, y) of region id has a right or lower neighbor in a different region.
func IsOutline(at func(x, y int) int32, x, y int, id int32) bool {
	right, down := at(x+1, y), at(x, y+1)
	return (right >= 0 && right != id) || (down >= 0 && down != id)
}

// RimReach returns which of neighbors (by index) a probe within reach of pixel (x, y) of region id found nearby.
func RimReach(at func(x, y int) int32, probes [][2]int, x, y int, id int32, neighbors []int32) (reach NeighborMask) {
	for _, p := range probes {
		other := at(x+p[0], y+p[1])
		if other < 0 || other == id {
			continue
		}
		for j, n := range neighbors {
			if n == other {
				reach.Add(j)
			}
		}
	}
	return reach
}

// BorderProbes returns the offsets within reach (Manhattan distance, excluding the pixel itself) of a pixel.
func BorderProbes(reach int) [][2]int {
	var probes [][2]int
	for dy := -reach; dy <= reach; dy++ {
		for dx := -reach; dx <= reach; dx++ {
			if d := max(dx, -dx) + max(dy, -dy); d > 0 && d <= reach {
				probes = append(probes, [2]int{dx, dy})
			}
		}
	}
	return probes
}
