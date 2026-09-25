package raster

import (
	"math"
	"sort"
)

// Point is a position or vector in pixels.
type Point struct{ X, Y float64 }

// subpath is a polyline; if closed, the last point joins back to the first.
type subpath struct {
	pts    []Point
	closed bool
}

// segmentCount includes the closing segment if closed.
func (sp subpath) segmentCount() int {
	if sp.closed {
		return len(sp.pts)
	}
	return max(0, len(sp.pts)-1)
}

func yRange(subpaths []subpath) (minY, maxY float64) {
	minY, maxY = math.Inf(1), math.Inf(-1)
	for _, sp := range subpaths {
		for _, p := range sp.pts {
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		}
	}
	return minY, maxY
}

// crossesY reports whether the segment a-b crosses height y, counting its lower end but not its upper so a vertex shared by two edges counts once, and never a horizontal segment.
func crossesY(a, b Point, y float64) bool {
	return a.Y != b.Y && y >= math.Min(a.Y, b.Y) && y < math.Max(a.Y, b.Y)
}

// lerp is the point t of the way from a to b; t outside 0..1 lies beyond them.
func lerp(a, b Point, t float64) Point { return a.plus(b.minus(a).times(t)) }

// xAt returns the x where the line through a and b is at height y; a and b must differ in y.
func xAt(a, b Point, y float64) float64 { return lerp(a, b, (y-a.Y)/(b.Y-a.Y)).X }

// appendCrossings appends, sorted, the x where each closed edge crosses scanline y (half-open in y).
func appendCrossings(xs []float64, subpaths []subpath, y float64) []float64 {
	for _, sp := range subpaths {
		for i, a := range sp.pts {
			if b := sp.pts[(i+1)%len(sp.pts)]; crossesY(a, b, y) {
				xs = append(xs, xAt(a, b, y))
			}
		}
	}
	sort.Float64s(xs)
	return xs
}

// unitAt is the unit vector angle radians from +x toward +y.
func unitAt(angle float64) Point { return Point{math.Cos(angle), math.Sin(angle)} }

// RegularPolygon returns the vertices of a polygon centered on (x, y); at rotation 0 odd sides point up, even sides sit flat.
func RegularPolygon(sides int, x, y, radius, rotation float64) []Point {
	center := Point{x, y}
	angle := 2 * math.Pi / float64(sides)
	rotation -= math.Pi / 2
	if sides%2 == 0 {
		rotation += angle / 2
	}
	pts := make([]Point, sides)
	for i := range pts {
		pts[i] = center.plus(unitAt(rotation + angle*float64(i)).times(radius))
	}
	return pts
}

// pixelAtOrAfter returns the first pixel whose center is at or after v, so spans [a, b) sharing an edge neither overlap nor gap.
func pixelAtOrAfter(v float64) int { return int(math.Ceil(v - 0.5)) }

func (p Point) plus(q Point) Point    { return Point{p.X + q.X, p.Y + q.Y} }
func (p Point) minus(q Point) Point   { return Point{p.X - q.X, p.Y - q.Y} }
func (p Point) times(f float64) Point { return Point{p.X * f, p.Y * f} }

// direction returns the unit vector pointing from a to b; a zero-length segment points along +x.
func direction(a, b Point) Point {
	d := b.minus(a)
	length := math.Hypot(d.X, d.Y)
	if length == 0 {
		return Point{1, 0}
	}
	return Point{d.X / length, d.Y / length}
}

// perpendicular returns v turned a quarter turn, to the left of v on a y-up plane and to the right on the y-down canvas.
func perpendicular(v Point) Point { return Point{-v.Y, v.X} }

// thickLine returns the outline of the segment a-b as a line of width 2*half with square caps, so it reaches half past each end.
func thickLine(a, b Point, half float64) subpath {
	along := direction(a, b)
	side := perpendicular(along).times(half)
	start, end := a.minus(along.times(half)), b.plus(along.times(half))
	return subpath{closed: true, pts: []Point{start.plus(side), end.plus(side), end.minus(side), start.minus(side)}}
}
