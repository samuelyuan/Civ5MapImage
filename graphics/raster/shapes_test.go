package raster

import (
	"math"
	"slices"
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

func TestDirectionIsAUnitVectorFromAToB(t *testing.T) {
	for _, tt := range []struct{ a, b, want Point }{
		{Point{2, 3}, Point{8, 3}, Point{1, 0}},
		{Point{2, 3}, Point{2, -1}, Point{0, -1}},
		{Point{0, 0}, Point{3, 4}, Point{0.6, 0.8}},
		{Point{5, 5}, Point{5, 5}, Point{1, 0}}, // no length: along +x
	} {
		if got := direction(tt.a, tt.b); math.Abs(got.X-tt.want.X) > 1e-12 || math.Abs(got.Y-tt.want.Y) > 1e-12 {
			t.Errorf("direction(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPerpendicularTurnsAQuarterTurn(t *testing.T) {
	for _, tt := range []struct{ v, want Point }{{Point{1, 0}, Point{0, 1}}, {Point{0, 1}, Point{-1, 0}}, {Point{3, -2}, Point{2, 3}}} {
		if got := perpendicular(tt.v); got != tt.want {
			t.Errorf("perpendicular(%v) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

// The line is the rectangle around a-b, half a width to each side and half a width past each end.
func TestThickLineIsARectangleAroundTheSegment(t *testing.T) {
	horizontal := thickLine(Point{2, 3}, Point{8, 3}, 1)
	if want := []Point{{1, 4}, {9, 4}, {9, 2}, {1, 2}}; !slices.Equal(horizontal.pts, want) || !horizontal.closed {
		t.Errorf("horizontal thickLine = %v (closed %v), want %v closed", horizontal.pts, horizontal.closed, want)
	}
	dot := thickLine(Point{5, 5}, Point{5, 5}, 2) // no length: a square
	if want := []Point{{3, 7}, {7, 7}, {7, 3}, {3, 3}}; !slices.Equal(dot.pts, want) {
		t.Errorf("zero-length thickLine = %v, want %v", dot.pts, want)
	}
	for _, p := range thickLine(Point{0, 0}, Point{3, 4}, 0.5).pts { // 3-4-5: every corner is half a width from the line and half past an end
		along, across := (p.X*3+p.Y*4)/5, (p.X*-4+p.Y*3)/5
		if math.Abs(math.Abs(across)-0.5) > 1e-12 || (math.Abs(along+0.5) > 1e-12 && math.Abs(along-5.5) > 1e-12) {
			t.Errorf("corner %v is at (along %v, across %v), want across +/-0.5 and along -0.5 or 5.5", p, along, across)
		}
	}
}

func TestUnitAtPointsFromPlusXTowardPlusY(t *testing.T) {
	for _, tt := range []struct {
		angle float64
		want  Point
	}{{0, Point{1, 0}}, {math.Pi / 2, Point{0, 1}}, {math.Pi, Point{-1, 0}}, {-math.Pi / 2, Point{0, -1}}} {
		if got := unitAt(tt.angle); math.Abs(got.X-tt.want.X) > 1e-12 || math.Abs(got.Y-tt.want.Y) > 1e-12 {
			t.Errorf("unitAt(%v) = %v, want %v", tt.angle, got, tt.want)
		}
	}
}

// A segment crosses the heights from its lower end up to but not including its upper end, so two edges meeting at a vertex count it once.
func TestCrossesYIsHalfOpenAndIgnoresHorizontalSegments(t *testing.T) {
	up, down := [2]Point{{0, 2}, {5, 6}}, [2]Point{{5, 6}, {0, 2}}
	for _, seg := range [][2]Point{up, down} {
		for _, tt := range []struct {
			y    float64
			want bool
		}{{1.9, false}, {2, true}, {4, true}, {5.9, true}, {6, false}} {
			if got := crossesY(seg[0], seg[1], tt.y); got != tt.want {
				t.Errorf("crossesY(%v, %v, %v) = %v, want %v", seg[0], seg[1], tt.y, got, tt.want)
			}
		}
	}
	if crossesY(Point{0, 3}, Point{9, 3}, 3) {
		t.Error("a horizontal segment crosses no height")
	}
}

func TestLerpIsTheFractionOfTheWayFromAToB(t *testing.T) {
	a, b := Point{2, 1}, Point{10, 5}
	for _, tt := range []struct {
		t    float64
		want Point
	}{{0, a}, {1, b}, {0.5, Point{6, 3}}, {0.25, Point{4, 2}}, {-0.5, Point{-2, -1}}, {1.5, Point{14, 7}}} {
		if got := lerp(a, b, tt.t); math.Abs(got.X-tt.want.X) > 1e-12 || math.Abs(got.Y-tt.want.Y) > 1e-12 {
			t.Errorf("lerp(%v, %v, %v) = %v, want %v", a, b, tt.t, got, tt.want)
		}
	}
}

func TestXAtIsWhereTheLineReachesTheHeight(t *testing.T) {
	a, b := Point{2, 1}, Point{10, 5}
	for _, tt := range []struct{ y, want float64 }{{1, 2}, {3, 6}, {5, 10}, {0, 0}, {7, 14}} {
		if got := xAt(a, b, tt.y); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("xAt(%v, %v, %v) = %v, want %v", a, b, tt.y, got, tt.want)
		}
	}
}
