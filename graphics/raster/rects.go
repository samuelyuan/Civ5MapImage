package raster

import (
	"cmp"
	"image"
	"slices"
)

// MergeNearbyRects merges rects within gap of each other into disjoint bounding boxes, dropping empty ones.
// The result is sorted by top-left corner regardless of input order.
func MergeNearbyRects(rects []image.Rectangle, gap int) []image.Rectangle {
	var merged []image.Rectangle // never within gap of each other
	pending := slices.Clone(rects)
	for len(pending) > 0 {
		r := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if r.Empty() {
			continue
		}
		near := func(m image.Rectangle) bool { return r.Inset(-gap).Overlaps(m) }
		if i := slices.IndexFunc(merged, near); i >= 0 {
			pending = append(pending, r.Union(merged[i])) // the bigger box may now reach other merged rects
			merged = slices.Delete(merged, i, i+1)
			continue
		}
		merged = append(merged, r)
	}
	slices.SortFunc(merged, func(a, b image.Rectangle) int {
		return cmp.Or(cmp.Compare(a.Min.Y, b.Min.Y), cmp.Compare(a.Min.X, b.Min.X))
	})
	return merged
}

// BoundsOf returns the smallest rect containing all of rects.
func BoundsOf(rects []image.Rectangle) (bounds image.Rectangle) {
	for _, rect := range rects {
		bounds = bounds.Union(rect)
	}
	return bounds
}
