package raster

import (
	"image"
	"image/color"
	"math"
	"testing"
)

var (
	unitBlack = color.RGBA{0, 0, 0, 255}
	unitRed   = color.RGBA{255, 0, 0, 255}
	unitBlue  = color.RGBA{0, 0, 255, 255}
)

func testPalette(n int) color.Palette {
	palette := color.Palette{color.RGBA{0, 0, 0, 255}}
	for i := 1; i < n; i++ {
		palette = append(palette, color.RGBA{uint8(i * 3), uint8(255 - i*2), uint8(i * 5), 255})
	}
	return palette
}

// paintedPixels returns every pixel that isn't the background, with its palette index.
func paintedPixels(c *PalettedCanvas) map[image.Point]uint8 {
	painted := map[image.Point]uint8{}
	b := c.img.Rect
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if v := c.IndexAt(x, y); v != c.background {
				painted[image.Pt(x, y)] = v
			}
		}
	}
	return painted
}

func unitCanvas(w, h int) *PalettedCanvas {
	return NewPalettedCanvas(w, h, color.Palette{unitBlack, unitRed, unitBlue})
}

// A span [a, b) covers the pixels whose centers (index + 0.5) are at or after a and before b: a shape edge that lands
// exactly on a pixel center owns that pixel only on its low side, so shapes sharing an edge neither overlap nor gap.
func TestPalettedCanvasFillIsHalfOpenAtPixelCenters(t *testing.T) {
	c := unitCanvas(8, 8)
	c.SetColor(255, 0, 0)
	c.DrawRectangle(0.5, 0.5, 3, 2) // edges on the centers of column 0 / 3 and row 0 / 2
	c.Fill()

	want := map[image.Point]bool{}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			want[image.Pt(x, y)] = true
		}
	}
	got := paintedPixels(c)
	if len(got) != len(want) {
		t.Fatalf("painted %d pixels, want %d: %v", len(got), len(want), got)
	}
	for p := range want {
		if _, ok := got[p]; !ok {
			t.Errorf("pixel %v not painted", p)
		}
	}
}

// A vertex that lands exactly on a pixel center joins two edges, and the row through it must still be filled: the edge
// that ends there doesn't count and the one that starts there does, so the row has one crossing on each side, not two.
func TestPalettedCanvasFillCountsAVertexOnAPixelCenterOnce(t *testing.T) {
	c := unitCanvas(20, 20)
	c.SetColor(255, 0, 0)
	// A hexagon with exact coordinates: its side vertices are at y = 4.5 and 12.5, the centers of rows 4 and 12.
	c.subpaths = append(c.subpaths, subpath{closed: true, pts: []Point{
		{8, 0.5}, {16, 4.5}, {16, 12.5}, {8, 16.5}, {0, 12.5}, {0, 4.5},
	}})
	c.Fill()

	painted := paintedPixels(c)
	for y := 1; y <= 15; y++ { // row 0 is the apex alone, an empty span
		first, last, count := math.MaxInt, -1, 0
		for x := 0; x < 20; x++ {
			if _, ok := painted[image.Pt(x, y)]; ok {
				first, last, count = min(first, x), max(last, x), count+1
			}
		}
		if count == 0 || last-first+1 != count {
			t.Errorf("row %d: %d painted pixels spanning %d..%d, want one non-empty contiguous span", y, count, first, last)
		}
	}
}

func TestPalettedCanvasRectanglesSharingAPixelCenterEdgeTileExactly(t *testing.T) {
	c := unitCanvas(6, 6)
	c.SetColor(255, 0, 0)
	c.DrawRectangle(0, 0.5, 6, 2) // rows 0-1
	c.Fill()
	c.SetColor(0, 0, 255)
	c.DrawRectangle(0, 2.5, 6, 2) // rows 2-3, starting exactly where the red one ends
	c.Fill()

	red, blue := c.IndexFor(255, 0, 0), c.IndexFor(0, 0, 255)
	for y := 0; y < 6; y++ {
		want := c.Background()
		switch {
		case y < 2:
			want = red
		case y < 4:
			want = blue
		}
		for x := 0; x < 6; x++ {
			if got := c.IndexAt(x, y); got != want {
				t.Fatalf("pixel (%d, %d) = index %d, want %d", x, y, got, want)
			}
		}
	}
}

// Stroking a closed shape draws every side, including the one joining the last point back to the first.
func TestPalettedCanvasStrokeDrawsTheClosingSideOfAClosedShape(t *testing.T) {
	c := unitCanvas(20, 20)
	c.SetColor(255, 0, 0)
	c.SetLineWidth(1)
	c.DrawRectangle(4, 4, 10, 8)
	c.Stroke()

	red := c.IndexFor(255, 0, 0)
	// DrawRectangle's points run (4,4) (14,4) (14,12) (4,12), so the left side x=4 is the closing one.
	if c.IndexAt(4, 8) != red && c.IndexAt(3, 8) != red {
		t.Error("the closing (left) side of the rectangle was not stroked")
	}
	if c.IndexAt(9, 4) != red && c.IndexAt(9, 3) != red {
		t.Error("the top side of the rectangle was not stroked")
	}
	if c.IndexAt(9, 8) == red {
		t.Error("the inside of the rectangle was stroked")
	}
}

func TestPalettedCanvasStrokeOfALineIsNotClosed(t *testing.T) {
	c := unitCanvas(20, 20)
	c.SetColor(255, 0, 0)
	c.DrawLine(2, 10, 12, 10)
	c.Stroke()
	// An open two-point path is one segment; treating it as closed would just draw it twice, so check its extent.
	red := c.IndexFor(255, 0, 0)
	if c.IndexAt(7, 10) != red && c.IndexAt(7, 9) != red {
		t.Error("the line was not stroked")
	}
	if c.IndexAt(16, 10) == red {
		t.Error("stroke extends past the end of the line")
	}
}

func TestColorTableSnapsOffPaletteColorsAndCountsEachOnce(t *testing.T) {
	c := unitCanvas(4, 4)
	if c.inexact != 0 {
		t.Fatalf("a fresh canvas has %d inexact colors", c.inexact)
	}
	if got := c.IndexFor(250, 10, 10); got != c.IndexFor(255, 0, 0) {
		t.Errorf("(250, 10, 10) snapped to index %d, want the red entry", got)
	}
	if c.inexact != 1 {
		t.Errorf("inexact = %d after one off-palette color, want 1", c.inexact)
	}
	c.IndexFor(250, 10, 10)
	c.IndexFor(0, 0, 255)
	if c.inexact != 1 {
		t.Errorf("inexact = %d after repeating it and using an exact color, want 1", c.inexact)
	}
}

// Sibling canvases share the table, so a color one of them can't find in the palette shows up on the parent.
func TestSiblingCanvasSharesTheColorTable(t *testing.T) {
	c := unitCanvas(4, 4)
	sibling := c.NewSibling(2, 2)
	sibling.SetColor(10, 10, 240)
	if c.inexact != 1 {
		t.Errorf("parent inexact = %d after its sibling canvas drew an off-palette color, want 1", c.inexact)
	}
}

func TestPalettedCanvasResizeKeepsTrackingRegionIDs(t *testing.T) {
	c := unitCanvas(4, 4)
	c.TrackIDs()
	c.Resize(6, 5)
	if len(c.ids) != 30 {
		t.Fatalf("after Resize(6, 5) ids has %d entries, want 30", len(c.ids))
	}
	for i, id := range c.ids {
		if id != -1 {
			t.Fatalf("ids[%d] = %d after Resize, want -1", i, id)
		}
	}
	c.SetID(7)
	c.DrawRectangle(0.5, 0.5, 2, 1)
	c.Fill()
	if c.ids[0] != 7 || c.ids[1] != 7 || c.ids[2] != -1 {
		t.Errorf("first row ids = %v, want 7 7 -1 ...", c.ids[:3])
	}
}

func TestPalettedCanvasPasteRegion(t *testing.T) {
	t.Run("same-palette sibling canvas is copied as indices", func(t *testing.T) {
		c := unitCanvas(6, 6)
		sibling := c.NewSibling(3, 3)
		sibling.SetColor(0, 0, 255)
		sibling.DrawRectangle(0, 0, 3, 3)
		sibling.Fill()

		c.PasteRegion(sibling.Image(), image.Rect(2, 2, 4, 4), image.Pt(1, 1))
		blue := c.IndexFor(0, 0, 255)
		for y := 0; y < 6; y++ {
			for x := 0; x < 6; x++ {
				want := c.Background()
				if x >= 2 && x < 4 && y >= 2 && y < 4 {
					want = blue
				}
				if got := c.IndexAt(x, y); got != want {
					t.Fatalf("pixel (%d, %d) = %d, want %d", x, y, got, want)
				}
			}
		}
	})

	t.Run("a paletted image on another palette is converted by color", func(t *testing.T) {
		c := unitCanvas(4, 4)
		other := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{unitBlack, unitBlue}) // index 1 is blue here, red in c
		for i := range other.Pix {
			other.Pix[i] = 1
		}
		c.PasteRegion(other, image.Rect(0, 0, 2, 2), image.Pt(0, 0))
		if got := c.IndexAt(0, 0); got != c.IndexFor(0, 0, 255) {
			t.Errorf("pasted pixel = index %d, want the blue entry %d (a raw index copy would give the red one)", got, c.IndexFor(0, 0, 255))
		}
	})

	t.Run("other images are converted to the nearest palette color", func(t *testing.T) {
		c := unitCanvas(4, 4)
		src := image.NewRGBA(image.Rect(0, 0, 4, 4))
		for i := 0; i < len(src.Pix); i += 4 {
			src.Pix[i], src.Pix[i+3] = 250, 255 // nearly red
		}
		c.PasteRegion(src, image.Rect(1, 1, 3, 3), image.Pt(0, 0))
		if got := c.IndexAt(1, 1); got != c.IndexFor(255, 0, 0) {
			t.Errorf("pasted pixel = index %d, want the red entry", got)
		}
		if got := c.IndexAt(0, 0); got != c.Background() {
			t.Errorf("pixel outside the pasted rect = index %d, want the background", got)
		}
	})
}

// A snapshot is a copy: later drawing doesn't change it, and its rect is where the pixels sit in the GIF.
func TestPalettedCanvasSnapshotIsAnIndependentCopy(t *testing.T) {
	c := unitCanvas(8, 8)
	c.SetColor(255, 0, 0)
	c.DrawRectangle(2, 2, 3, 3)
	c.Fill()

	rect := image.Rect(1, 1, 6, 6)
	frame := c.Snapshot(rect)
	if frame.Rect != rect {
		t.Fatalf("frame rect = %v, want %v", frame.Rect, rect)
	}
	red := c.IndexFor(255, 0, 0)
	if frame.ColorIndexAt(2, 2) != red || frame.ColorIndexAt(1, 1) != c.Background() {
		t.Error("the snapshot doesn't match the canvas")
	}

	c.SetColor(0, 0, 255)
	c.DrawRectangle(0, 0, 8, 8)
	c.Fill()
	if frame.ColorIndexAt(2, 2) != red {
		t.Error("drawing after Snapshot changed the frame")
	}
}

// A stroke of width >= 1 at any angle must be one 8-connected run of pixels reaching both ends.
func TestPalettedCanvasStrokeIsConnected(t *testing.T) {
	for _, width := range []float64{1.0, 1.5, 2.0} {
		for deg := 0; deg < 180; deg += 7 {
			canvas := NewPalettedCanvas(64, 64, testPalette(4))
			canvas.SetColor(testPalette(4)[1].(color.RGBA).R, testPalette(4)[1].(color.RGBA).G, testPalette(4)[1].(color.RGBA).B)
			canvas.SetLineWidth(width)
			a := float64(deg) * math.Pi / 180
			canvas.DrawLine(32.3-16*math.Cos(a), 32.7-16*math.Sin(a), 32.3+16*math.Cos(a), 32.7+16*math.Sin(a))
			canvas.Stroke()

			painted := paintedPixels(canvas)
			if len(painted) == 0 {
				t.Fatalf("width %v angle %d: nothing painted", width, deg)
			}
			var start image.Point
			for p := range painted {
				start = p
				break
			}
			seen := map[image.Point]bool{start: true}
			queue := []image.Point{start}
			for len(queue) > 0 {
				p := queue[0]
				queue = queue[1:]
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						n := image.Pt(p.X+dx, p.Y+dy)
						if _, ok := painted[n]; ok && !seen[n] {
							seen[n] = true
							queue = append(queue, n)
						}
					}
				}
			}
			if len(seen) != len(painted) {
				t.Errorf("width %v angle %d: stroke split into pieces (%d of %d pixels connected)", width, deg, len(seen), len(painted))
			}
		}
	}
}
