package raster

import (
	"image"
	"math/rand"
	"slices"
	"testing"
)

const testGap = 8

func TestMergeNearbyRects(t *testing.T) {
	tests := []struct {
		name string
		in   []image.Rectangle
		want []image.Rectangle
	}{
		{"nothing", nil, nil},
		{"one rect is unchanged", []image.Rectangle{image.Rect(5, 5, 15, 15)}, []image.Rectangle{image.Rect(5, 5, 15, 15)}},
		{"far apart stay separate",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(100, 100, 110, 110)},
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(100, 100, 110, 110)}},
		{"overlapping merge",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(5, 5, 20, 20)},
			[]image.Rectangle{image.Rect(0, 0, 20, 20)}},
		{"touching merge",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10, 0, 20, 10)},
			[]image.Rectangle{image.Rect(0, 0, 20, 10)}},
		{"just inside the gap merge",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10+testGap-1, 0, 30, 10)},
			[]image.Rectangle{image.Rect(0, 0, 30, 10)}},
		{"exactly the gap apart stay separate",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10+testGap, 0, 30, 10)},
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10+testGap, 0, 30, 10)}},
		{"empty rects are dropped",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), {}, image.Rect(50, 50, 50, 60)},
			[]image.Rectangle{image.Rect(0, 0, 10, 10)}},
	}
	for _, tt := range tests {
		if got := MergeNearbyRects(tt.in, testGap); !slices.Equal(got, tt.want) {
			t.Errorf("%s: MergeNearbyRects(%v) = %v, want %v", tt.name, tt.in, got, tt.want)
		}
	}
}

// Merging two rects grows their box, and the bigger box can reach a third rect that neither was near alone; the input order must not matter.
func TestMergeNearbyRectsMergesChainsInAnyOrder(t *testing.T) {
	tall := image.Rect(0, 0, 10, 50)
	wide := image.Rect(15, 0, 60, 10)    // near tall, so they merge into (0, 0, 60, 50)
	inside := image.Rect(45, 30, 55, 40) // far from both alone, but inside their merged box
	want := []image.Rectangle{image.Rect(0, 0, 60, 50)}
	orders := [][]image.Rectangle{{tall, wide, inside}, {inside, tall, wide}, {wide, inside, tall}, {inside, wide, tall}}
	for _, in := range orders {
		if got := MergeNearbyRects(in, testGap); !slices.Equal(got, want) {
			t.Errorf("MergeNearbyRects(%v) = %v, want %v", in, got, want)
		}
	}
}

// For any input, every rect is covered by some cluster, no two clusters are within the gap of each other, and the input order doesn't matter.
func TestMergeNearbyRectsCoversInputWithSeparatedClusters(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 200; trial++ {
		var in []image.Rectangle
		for i, n := 0, 1+rng.Intn(60); i < n; i++ {
			x, y := rng.Intn(400), rng.Intn(400)
			in = append(in, image.Rect(x, y, x+1+rng.Intn(40), y+1+rng.Intn(40)))
		}
		clusters := MergeNearbyRects(in, testGap)
		reversed := slices.Clone(in)
		slices.Reverse(reversed)
		if again := MergeNearbyRects(reversed, testGap); !slices.Equal(clusters, again) {
			t.Fatalf("trial %d: reversing the input gave %v, want %v", trial, again, clusters)
		}
		for _, r := range in {
			if !slices.ContainsFunc(clusters, func(c image.Rectangle) bool { return r.In(c) }) {
				t.Fatalf("trial %d: %v is in no cluster of %v", trial, r, clusters)
			}
		}
		for i, a := range clusters {
			for _, b := range clusters[i+1:] {
				if a.Inset(-testGap).Overlaps(b) {
					t.Fatalf("trial %d: clusters %v and %v are within the gap", trial, a, b)
				}
			}
		}
	}
}

func TestBoundsOf(t *testing.T) {
	if got := BoundsOf(nil); got != (image.Rectangle{}) {
		t.Errorf("BoundsOf(nil) = %v, want the zero rect", got)
	}
	in := []image.Rectangle{image.Rect(5, 10, 15, 20), image.Rect(-3, 40, 8, 45)}
	if got, want := BoundsOf(in), image.Rect(-3, 10, 15, 45); got != want {
		t.Errorf("BoundsOf(%v) = %v, want %v", in, got, want)
	}
}
