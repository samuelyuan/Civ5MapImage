package graphics

import (
	"image/color"
	"sort"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// A mountain tile's fill darkens toward mountainGround so a range reads as a region.
const mountainGroundBlend = 0.45

var (
	mountainGround   = color.RGBA{96, 92, 84, 255}
	peakLitColor     = color.RGBA{132, 132, 124, 255}
	peakShadeColor   = color.RGBA{70, 70, 66, 255}
	peakSnowLitColor = color.RGBA{250, 252, 255, 255}
	peakSnowShade    = color.RGBA{190, 200, 216, 255}
	peakOutlineColor = color.RGBA{40, 38, 36, 255}
)

const (
	peakHalfWidth = 0.55 // a peak's half width and height, as fractions of a tile's radius
	peakHeight    = 1.0
	peakBase      = 0.4 // how far below the tile's center the peak's base is, as a fraction of the radius
	peakSnowLine  = 0.3 // the fraction of the peak's height that is snow
	gapPeakScale  = 0.6 // the size of the peak between two adjacent mountain tiles, as a fraction of a tile's peak
	trioPeakScale = 0.7 // the size of the peak in the middle of three mutually adjacent mountain tiles
)

// How far in pixels a peak's outline reaches past its triangle.
const (
	peakOutlineTop  = 2.0
	peakOutlineSide = 1.5
	peakOutlineBase = 1.0
)

// mountainTileColor returns the fill of a mountain tile whose ground would otherwise be fill.
func mountainTileColor(fill color.RGBA) color.RGBA {
	return blendColor(fill, mountainGround, mountainGroundBlend)
}

// peak is a mountain to draw, standing on the ground line base at x.
type peak struct{ x, base, half, height float64 }

// peakAt returns the peak for a tile centered on (x, y), scale times the size of a lone mountain's.
func peakAt(x, y, radius, scale float64) peak {
	return peak{x, y + peakBase*radius, scale * peakHalfWidth * radius, scale * peakHeight * radius}
}

// tileBefore reports whether a comes before b in row-major order.
func tileBefore(a, b fileio.TilePos) bool {
	return a.Row < b.Row || (a.Row == b.Row && a.Col < b.Col)
}

// rangePeaks returns peaks back to front: one per mountain tile, one per three mutually adjacent tiles, and a smaller one per other adjacent pair.
func rangePeaks(mapData *fileio.Civ5MapData, mapSize fileio.MapSize, l tileLayout) []peak {
	isMountain := func(pos fileio.TilePos) bool { return pos.InMap(mapSize) && fileio.TileHasMountain(mapData, pos) }
	var tiles, fillers []peak
	for _, pos := range allTiles(mapSize) {
		if !isMountain(pos) {
			continue
		}
		x, y := l.center(pos)
		tiles = append(tiles, peakAt(x, y, l.radius, 1))

		// the neighbors go around the tile, so each is adjacent to the one before and after it
		neighbors := fileio.GetNeighbors(pos)
		for i, neighbor := range neighbors {
			next, prev := neighbors[(i+1)%6], neighbors[(i+5)%6]
			// a three is added from its first tile, so once
			if isMountain(neighbor) && isMountain(next) && tileBefore(pos, neighbor) && tileBefore(pos, next) {
				neighborX, neighborY := l.center(neighbor)
				nextX, nextY := l.center(next)
				fillers = append(fillers, peakAt((x+neighborX+nextX)/3, (y+neighborY+nextY)/3, l.radius, trioPeakScale))
			}
			// a pair is added from its first tile, unless a mountain next to both puts it in a three
			if isMountain(neighbor) && tileBefore(pos, neighbor) && !isMountain(next) && !isMountain(prev) {
				neighborX, neighborY := l.center(neighbor)
				fillers = append(fillers, peakAt((x+neighborX)/2, (y+neighborY)/2, l.radius, gapPeakScale))
			}
		}
	}
	peaks := append(tiles, fillers...)
	sort.SliceStable(peaks, func(i, j int) bool { return peaks[i].base < peaks[j].base })
	return peaks
}

// drawPeak draws a snow-capped peak with a lit left face and a shaded right face, in an outline.
func drawPeak(canvas Canvas, p peak) {
	top := p.base - p.height
	outlineTop, outlineBase, outlineHalf := top-peakOutlineTop, p.base+peakOutlineBase, p.half+peakOutlineSide
	fillTriangle(canvas, peakOutlineColor, p.x, outlineTop, p.x-outlineHalf, outlineBase, p.x+outlineHalf, outlineBase)
	fillTriangle(canvas, peakLitColor, p.x, top, p.x-p.half, p.base, p.x, p.base)
	fillTriangle(canvas, peakShadeColor, p.x, top, p.x, p.base, p.x+p.half, p.base)

	snowY, snowHalf := top+peakSnowLine*p.height, peakSnowLine*p.half
	fillTriangle(canvas, peakSnowLitColor, p.x, top, p.x-snowHalf, snowY, p.x, snowY)
	fillTriangle(canvas, peakSnowShade, p.x, top, p.x, snowY, p.x+snowHalf, snowY)
}
