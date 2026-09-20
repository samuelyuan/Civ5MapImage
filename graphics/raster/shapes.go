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

// appendCrossings appends, sorted, the x where each closed subpath edge crosses the scanline y (half-open in y, so shared vertices count once).
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

// RegularPolygon returns a regular polygon's vertices centered on (x, y); at rotation 0, odd sides point up and even sides sit flat on top.
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

// pixelAtOrAfter returns the first pixel whose center (index + 0.5) is at or after v, so spans [a, b) sharing an edge neither overlap nor gap.
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
