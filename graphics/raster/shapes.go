package raster

import (
	"image"
	"math"
	"sort"
)

// Point is a position in pixels.
type Point struct{ X, Y float64 }

// subpath is a polyline; if closed, the last point joins back to the first.
type subpath struct {
	pts    []Point
	closed bool
}

// segmentCount returns the number of line segments joining the points, including the closing one if closed.
func (sp subpath) segmentCount() int {
	if sp.closed {
		return len(sp.pts)
	}
	return max(0, len(sp.pts)-1)
}

// yRange returns the smallest and largest y among the subpaths' points.
func yRange(subpaths []subpath) (minY, maxY float64) {
	minY, maxY = math.Inf(1), math.Inf(-1)
	for _, sp := range subpaths {
		for _, p := range sp.pts {
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		}
	}
	return minY, maxY
}

// appendCrossings appends, sorted, the x where each closed edge crosses scanline y (half-open in y).
func appendCrossings(xs []float64, subpaths []subpath, y float64) []float64 {
	for _, sp := range subpaths {
		for i, a := range sp.pts {
			b := sp.pts[(i+1)%len(sp.pts)]
			if a.Y == b.Y || y < math.Min(a.Y, b.Y) || y >= math.Max(a.Y, b.Y) {
				continue
			}
			xs = append(xs, a.X+(y-a.Y)/(b.Y-a.Y)*(b.X-a.X))
		}
	}
	sort.Float64s(xs)
	return xs
}

// RegularPolygon returns the vertices of a polygon centered on (x, y); at rotation 0 odd sides point up, even sides sit flat.
func RegularPolygon(sides int, x, y, radius, rotation float64) []Point {
	angle := 2 * math.Pi / float64(sides)
	rotation -= math.Pi / 2
	if sides%2 == 0 {
		rotation += angle / 2
	}
	pts := make([]Point, sides)
	for i := range pts {
		a := rotation + angle*float64(i)
		pts[i] = Point{x + radius*math.Cos(a), y + radius*math.Sin(a)}
	}
	return pts
}

// pixelAtOrAfter returns the first pixel whose center is at or after v, so spans [a, b) sharing an edge neither overlap nor gap.
func pixelAtOrAfter(v float64) int { return int(math.Ceil(v - 0.5)) }

// segment is a line segment prepared for distance queries.
type segment struct {
	a, b     Point
	dx, dy   float64 // b - a
	lengthSq float64
}

func newSegment(a, b Point) segment {
	dx, dy := b.X-a.X, b.Y-a.Y
	return segment{a: a, b: b, dx: dx, dy: dy, lengthSq: dx*dx + dy*dy}
}

// pixelBounds returns the pixels that could lie within half of the segment.
func (s segment) pixelBounds(half float64) image.Rectangle {
	return image.Rect(
		int(math.Floor(math.Min(s.a.X, s.b.X)-half)), int(math.Floor(math.Min(s.a.Y, s.b.Y)-half)),
		int(math.Ceil(math.Max(s.a.X, s.b.X)+half))+1, int(math.Ceil(math.Max(s.a.Y, s.b.Y)+half))+1)
}

// rowSpan returns x bounds on row y containing every point within half of the segment (possibly wider, never narrower), and false if none.
func (s segment) rowSpan(y, half float64) (lo, hi float64, ok bool) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, end := range [2]Point{s.a, s.b} { // the round caps
		if dy := y - end.Y; math.Abs(dy) <= half {
			dx := math.Sqrt(half*half - dy*dy)
			lo, hi, ok = math.Min(lo, end.X-dx), math.Max(hi, end.X+dx), true
		}
	}
	if s.lengthSq == 0 {
		return lo, hi, ok
	}

	// The body: within half of the line (v) and between the ends (u). With dx, dy the direction, each bound is a range of x.
	length := math.Sqrt(s.lengthSq)
	ux, uy := s.dx/length, s.dy/length // along the segment
	w := y - s.a.Y
	bodyLo, bodyHi := math.Inf(-1), math.Inf(1)
	limit := func(coefX, offset, from, to float64) bool { // from <= coefX*(x-a.x) + offset <= to
		if math.Abs(coefX) < 1e-12 {
			return from <= offset && offset <= to
		}
		x0, x1 := (from-offset)/coefX, (to-offset)/coefX
		bodyLo, bodyHi = math.Max(bodyLo, s.a.X+math.Min(x0, x1)), math.Min(bodyHi, s.a.X+math.Max(x0, x1))
		return true
	}
	if limit(-uy, w*ux, -half, half) && limit(ux, w*uy, 0, length) && bodyLo <= bodyHi {
		lo, hi, ok = math.Min(lo, bodyLo), math.Max(hi, bodyHi), true
	}
	return lo, hi, ok
}

// distSq returns the squared distance from (px, py) to the segment.
func (s segment) distSq(px, py float64) float64 {
	px, py = px-s.a.X, py-s.a.Y
	t := 0.0
	if s.lengthSq > 0 {
		t = math.Max(0, math.Min(1, (px*s.dx+py*s.dy)/s.lengthSq))
	}
	ex, ey := px-t*s.dx, py-t*s.dy
	return ex*ex + ey*ey
}
