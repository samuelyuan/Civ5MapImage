package raster

import (
	"math"
	"math/rand"
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

// rowSpan may be too wide but never too narrow: every pixel center distSq accepts on a row lies inside it, for segments of every direction.
func TestRowSpanCoversEveryPixelCenterWithinReach(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	segments := []segment{
		newSegment(Point{10, 10}, Point{10, 10}),     // a point
		newSegment(Point{2, 7.5}, Point{30, 7.5}),    // horizontal
		newSegment(Point{12.3, 1}, Point{12.3, 30}),  // vertical
		newSegment(Point{5, 5}, Point{25, 25}),       // diagonal
		newSegment(Point{25, 5}, Point{5, 25}),       // the other diagonal
		newSegment(Point{9.5, 4.5}, Point{9.5, 4.5}), // a point on a pixel center
	}
	for i := 0; i < 500; i++ {
		segments = append(segments, newSegment(Point{rng.Float64() * 40, rng.Float64() * 40}, Point{rng.Float64() * 40, rng.Float64() * 40}))
	}
	for _, seg := range segments {
		for _, half := range []float64{0.5, 1, 1.5, 3} {
			for py := -5; py < 50; py++ {
				y := float64(py) + 0.5
				lo, hi, ok := seg.rowSpan(y, half)
				for px := -5; px < 50; px++ {
					x := float64(px) + 0.5
					if seg.distSq(x, y) > half*half {
						continue
					}
					if !ok || x < lo || x > hi {
						t.Fatalf("segment %v half %v: pixel center (%v, %v) is within reach but rowSpan = [%v, %v] ok=%v", seg, half, x, y, lo, hi, ok)
					}
				}
			}
		}
	}
}
