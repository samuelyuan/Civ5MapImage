package raster

import (
	"math"
	"testing"
)

func TestRegularPolygonTriangleOrientation(t *testing.T) {
	// Rotation pi points a triangle's apex along +y from the center; the base is on the other side.
	tri := RegularPolygon(3, 10, 20, 8, math.Pi)
	if got := tri[0]; math.Abs(got.X-10) > 1e-9 || math.Abs(got.Y-28) > 1e-9 {
		t.Errorf("apex = %v, want (10, 28)", got)
	}
	for _, p := range tri {
		if d := math.Hypot(p.X-10, p.Y-20); math.Abs(d-8) > 1e-9 {
			t.Errorf("vertex %v is %v from the center, want 8", p, d)
		}
	}
}

func TestPixelAtOrAfter(t *testing.T) {
	for _, tt := range []struct {
		v    float64
		want int
	}{{0, 0}, {0.5, 0}, {0.51, 1}, {1.5, 1}, {2, 2}, {-0.5, -1}, {-0.4, 0}} {
		if got := pixelAtOrAfter(tt.v); got != tt.want {
			t.Errorf("pixelAtOrAfter(%v) = %d, want %d", tt.v, got, tt.want)
		}
	}
}

func TestSegmentDistSq(t *testing.T) {
	s := newSegment(Point{0, 0}, Point{10, 0})
	for _, tt := range []struct {
		px, py, want float64
	}{{5, 3, 9}, {-4, 3, 25}, {13, 4, 25}, {0, 0, 0}, {10, 0, 0}} {
		if got := s.distSq(tt.px, tt.py); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("distSq(%v, %v) = %v, want %v", tt.px, tt.py, got, tt.want)
		}
	}
	dot := newSegment(Point{2, 2}, Point{2, 2})
	if got := dot.distSq(5, 6); math.Abs(got-25) > 1e-9 {
		t.Errorf("single-point segment distSq = %v, want 25", got)
	}
}
