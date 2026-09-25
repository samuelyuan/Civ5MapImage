package raster

import (
	"image"
	"slices"
)

type Pixel struct{ X, Y int32 }

// IsOther reports whether a pixel holding region other belongs to a region different from id; negative ids lie outside every region.
func IsOther(other, id int32) bool { return other >= 0 && other != id }

// IsOutline reports whether pixel (x, y) of region id has a right or lower neighbor in another region, so a boundary is outlined one pixel wide, on the upper-left region's side.
func IsOutline(at func(x, y int) int32, x, y int, id int32) bool {
	return IsOther(at(x+1, y), id) || IsOther(at(x, y+1), id)
}

// NearOther reports whether any probe offset from pixel (x, y) of region id lands in another region.
func NearOther(at func(x, y int) int32, probes []image.Point, x, y int, id int32) bool {
	return slices.ContainsFunc(probes, func(p image.Point) bool { return IsOther(at(x+p.X, y+p.Y), id) })
}

// BorderProbes returns the offsets within reach of a pixel by Manhattan distance, excluding the pixel itself.
func BorderProbes(reach int) []image.Point {
	var probes []image.Point
	for dy := -reach; dy <= reach; dy++ {
		width := reach - max(dy, -dy) // the diamond's half width on this row
		for dx := -width; dx <= width; dx++ {
			if dx != 0 || dy != 0 {
				probes = append(probes, image.Pt(dx, dy))
			}
		}
	}
	return probes
}
