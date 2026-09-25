package raster

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
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

func TestColorTableSnapsOffPaletteColorsToTheNearestEntry(t *testing.T) {
	c := unitCanvas(4, 4)
	size := len(c.Palette())
	red := c.IndexFor(255, 0, 0)
	for i := 0; i < 2; i++ { // the second lookup is remembered
		if got := c.IndexFor(250, 10, 10); got != red {
			t.Errorf("(250, 10, 10) snapped to index %d, want the red entry %d", got, red)
		}
	}
	if len(c.Palette()) != size {
		t.Errorf("palette grew from %d to %d colors, want a fixed palette to stay as it is", size, len(c.Palette()))
	}
}

// Sibling canvases share the table, so a color one of them adds to a growing palette shows up on the parent.
func TestSiblingCanvasSharesTheColorTable(t *testing.T) {
	c := NewGrowingPalettedCanvas(4, 4)
	size := len(c.Palette())
	sibling := c.NewSibling(2, 2)
	sibling.SetColor(10, 10, 240)
	if len(c.Palette()) != size+1 {
		t.Errorf("parent palette has %d colors after its sibling drew a new one, want %d", len(c.Palette()), size+1)
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

// Stroke paints exactly the pixels whose centers lie inside the segment's rectangle, which is half the line width wide on each side and reaches half the line width past each end.
func TestPalettedCanvasStrokeMatchesAScanOfEveryPixel(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 200; i++ {
		a := Point{rng.Float64()*50 - 5, rng.Float64()*50 - 5}
		b := Point{rng.Float64()*50 - 5, rng.Float64()*50 - 5}
		if i%10 == 0 {
			b = a
		}
		half := []float64{0.5, 1, 1.75}[i%3]
		ux, uy := 1.0, 0.0
		if length := math.Hypot(b.X-a.X, b.Y-a.Y); length > 0 {
			ux, uy = (b.X-a.X)/length, (b.Y-a.Y)/length
		}
		length := math.Hypot(b.X-a.X, b.Y-a.Y)

		c := NewPalettedCanvas(40, 40, color.Palette{unitBlack, unitRed})
		c.SetColor(255, 0, 0)
		c.SetLineWidth(2 * half)
		c.DrawLine(a.X, a.Y, b.X, b.Y)
		c.Stroke()

		for y := 0; y < 40; y++ {
			for x := 0; x < 40; x++ {
				dx, dy := float64(x)+0.5-a.X, float64(y)+0.5-a.Y
				along, across := dx*ux+dy*uy, math.Abs(-dx*uy+dy*ux)
				inside := along >= -half && along <= length+half && across <= half
				clearlyInside := along > -half+1e-9 && along < length+half-1e-9 && across < half-1e-9
				clearlyOutside := along < -half-1e-9 || along > length+half+1e-9 || across > half+1e-9
				painted := c.IndexAt(x, y) != 0
				if clearlyInside && !painted || clearlyOutside && painted {
					t.Fatalf("line %v-%v half %v: pixel (%d, %d) inside=%v painted=%v", a, b, half, x, y, inside, painted)
				}
			}
		}
	}
}

// A 1px line whose ends sit on pixel-center rows is a tie the last float bit decides, so a staging canvas must build the same polygon as the full canvas.
func TestPalettedCanvasStrokeMatchesAcrossAnOrigin(t *testing.T) {
	palette := color.Palette{unitBlack, unitRed}
	for _, line := range [][4]float64{
		{259.55890982936734, 152, 259.5589098293673, 136},
		{259.5589098293673, 136, 259.55890982936734, 152},
		{100.3, 40, 180.9, 40},
	} {
		full := NewPalettedCanvas(400, 200, palette)
		full.SetColor(255, 0, 0)
		full.DrawLine(line[0], line[1], line[2], line[3])
		full.Stroke()

		staging := full.NewSibling(200, 60)
		staging.SetOrigin(image.Pt(102, 126))
		staging.SetColor(255, 0, 0)
		staging.DrawLine(line[0], line[1], line[2], line[3])
		staging.Stroke()

		for py := 0; py < 60; py++ {
			for px := 0; px < 200; px++ {
				if got, want := staging.IndexAt(px, py), full.IndexAt(px+102, py+126); got != want {
					t.Fatalf("line %v: staging pixel (%d, %d) = %d, want the full canvas's %d", line, px, py, got, want)
				}
			}
		}
	}
}

// The glyph bitmaps were checked against the anti-aliased renderer this canvas replaced, so they pin the font, the baseline and the pixel rounding.
func TestPalettedCanvasDrawStringDrawsSolidGlyphs(t *testing.T) {
	tests := []struct {
		name string
		text string
		x, y float64
		want []string
	}{
		{"whole pixel position", "Hi", 2, 13, []string{
			"..................",
			"..................",
			"..................",
			"..................",
			"..#....#..........",
			"..#....#....#.....",
			"..#....#..........",
			"..#....#...##.....",
			"..######....#.....",
			"..#....#....#.....",
			"..#....#....#.....",
			"..#....#....#.....",
			"..#....#..#####...",
			"..................",
			"..................",
			"..................",
		}},
		{"fractional position, descender", "Rj", 0.4, 12.6, []string{
			"..................",
			"..................",
			"..................",
			"..................",
			"#####.............",
			"#....#......#.....",
			"#....#............",
			"#....#.....##.....",
			"#####.......#.....",
			"#.#.........#.....",
			"#..#........#.....",
			"#...#.......#.....",
			"#....#..#...#.....",
			"........#...#.....",
			".........###......",
			"..................",
		}},
	}
	for _, tt := range tests {
		c := NewPalettedCanvas(18, 16, color.Palette{unitBlack, unitRed})
		c.SetColor(255, 0, 0)
		c.DrawString(tt.text, tt.x, tt.y)
		for y, wantRow := range tt.want {
			var row strings.Builder
			for x := 0; x < 18; x++ {
				if c.IndexAt(x, y) != 0 {
					row.WriteByte('#')
				} else {
					row.WriteByte('.')
				}
			}
			if row.String() != wantRow {
				t.Fatalf("%s: row %d = %q, want %q", tt.name, y, row.String(), wantRow)
			}
		}
	}
}

// Text starting left of a staging canvas's origin has a negative local position, which must round the same way as on the full canvas.
func TestPalettedCanvasDrawStringMatchesAcrossAnOrigin(t *testing.T) {
	palette := color.Palette{unitBlack, unitRed}
	for _, x := range []float64{100.3, 100.49, 100.5, 100.51, 100.75} {
		full := NewPalettedCanvas(200, 30, palette)
		full.SetColor(255, 0, 0)
		full.DrawString("Rome", x, 20)

		staging := full.NewSibling(60, 30)
		staging.SetOrigin(image.Pt(110, 0))
		staging.SetColor(255, 0, 0)
		staging.DrawString("Rome", x, 20)

		for py := 0; py < 30; py++ {
			for px := 0; px < 60; px++ {
				if got, want := staging.IndexAt(px, py), full.IndexAt(px+110, py); got != want {
					t.Fatalf("text at x=%v: staging pixel (%d, %d) = %d, want the full canvas's %d", x, px, py, got, want)
				}
			}
		}
	}
}

func TestPalettedCanvasMeasureStringIsSevenPixelsPerCharacterAndThirteenTall(t *testing.T) {
	c := NewPalettedCanvas(1, 1, color.Palette{unitBlack})
	for _, text := range []string{"", "I", "Samarkand 42"} {
		w, h := c.MeasureString(text)
		if w != float64(7*len(text)) || h != 13 {
			t.Errorf("MeasureString(%q) = (%v, %v), want (%d, 13)", text, w, h, 7*len(text))
		}
	}
}

// A fill paints exactly the pixels whose centers are inside the shape. Coordinates avoid putting a center on an edge, where the choice is arbitrary.
func TestPalettedCanvasFillsExactlyThePixelCentersInsideAShape(t *testing.T) {
	shapes := map[string][]Point{
		"hexagon":   RegularPolygon(6, 30.3, 40.2, 16, math.Pi/2),
		"triangle":  RegularPolygon(3, 62.7, 25.9, 14, math.Pi),
		"rectangle": {{50.2, 55.5}, {70.5, 55.5}, {70.5, 73.6}, {50.2, 73.6}},
	}
	for name, pts := range shapes {
		c := NewPalettedCanvas(90, 90, color.Palette{unitBlack, unitRed})
		c.SetColor(255, 0, 0)
		c.subpaths = append(c.subpaths, subpath{pts: pts, closed: true})
		c.Fill()
		for y := 0; y < 90; y++ {
			for x := 0; x < 90; x++ {
				want := pointInPolygon(float64(x)+0.5, float64(y)+0.5, pts)
				if got := c.IndexAt(x, y) != 0; got != want {
					t.Fatalf("%s: pixel (%d, %d) painted=%v, want %v", name, x, y, got, want)
				}
			}
		}
	}
}

func pointInPolygon(px, py float64, pts []Point) bool {
	inside := false
	for i, a := range pts {
		b := pts[(i+1)%len(pts)]
		if (a.Y > py) != (b.Y > py) && px < a.X+(py-a.Y)/(b.Y-a.Y)*(b.X-a.X) {
			inside = !inside
		}
	}
	return inside
}

// The two faces of a peak share an edge, so together they cover exactly the triangle they split, with no gap and no overlap.
func TestPalettedCanvasTrianglesSharingAnEdgeTileExactly(t *testing.T) {
	whole := unitCanvas(10, 10)
	whole.SetColor(255, 0, 0)
	whole.DrawTriangle(4, 0, 0, 8, 8, 8)
	whole.Fill()
	want := paintedPixels(whole)
	if len(want) == 0 {
		t.Fatal("DrawTriangle() painted nothing")
	}

	halves := unitCanvas(10, 10)
	halves.SetColor(255, 0, 0)
	halves.DrawTriangle(4, 0, 0, 8, 4, 8)
	halves.Fill()
	halves.SetColor(0, 0, 255)
	halves.DrawTriangle(4, 0, 4, 8, 8, 8)
	halves.Fill()

	got := paintedPixels(halves)
	if len(got) != len(want) {
		t.Fatalf("halves painted %d pixels, the whole %d", len(got), len(want))
	}
	for p := range want {
		if _, ok := got[p]; !ok {
			t.Errorf("pixel %v is in the whole triangle but not the halves", p)
		}
	}
	// the halves meet at x=4, a pixel edge: the left one owns columns 0-3 and the right one 4-7
	for p, v := range got {
		if wantRed := p.X < 4; (v == halves.IndexFor(255, 0, 0)) != wantRed {
			t.Errorf("pixel %v has index %d, on the wrong side of the shared edge", p, v)
		}
	}
}

func TestGrowingCanvasPaletteHoldsExactlyTheColorsDrawn(t *testing.T) {
	c := NewGrowingPalettedCanvas(4, 4)
	red := c.IndexFor(255, 0, 0)
	blue := c.IndexFor(0, 0, 255)
	if red == blue || c.IndexFor(255, 0, 0) != red {
		t.Fatalf("indexes red=%d blue=%d, want distinct and stable", red, blue)
	}
	c.PaintPixel(1, 1, blue)

	img := c.Image().(*image.Paletted)
	if len(img.Palette) != 3 { // black, red, blue
		t.Fatalf("palette has %d colors, want 3", len(img.Palette))
	}
	if got := img.At(1, 1); got != (color.RGBA{0, 0, 255, 255}) {
		t.Errorf("pixel = %v, want blue", got)
	}
}
