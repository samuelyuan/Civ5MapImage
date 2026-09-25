package raster

import (
	"cmp"
	"image"
	"slices"
)

// MergeOverlappingRects merges rects that overlap into their bounding boxes until none overlap, dropping empty ones.
// The result is sorted by top-left corner regardless of input order.
func MergeOverlappingRects(rects []image.Rectangle) []image.Rectangle {
	var merged []image.Rectangle // none overlap
	pending := slices.Clone(rects)
	for len(pending) > 0 {
		r := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if r.Empty() {
			continue
		}
		if i := slices.IndexFunc(merged, r.Overlaps); i >= 0 {
			pending = append(pending, r.Union(merged[i])) // the bigger box may now overlap other merged rects
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

func BoundsOf(rects []image.Rectangle) (bounds image.Rectangle) {
	for _, rect := range rects {
		bounds = bounds.Union(rect)
	}
	return bounds
}
