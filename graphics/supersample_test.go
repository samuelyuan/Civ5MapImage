package graphics

import (
	"image"
	"image/color"
	"math"
	"testing"
)

var (
	testOrange = color.RGBA{200, 100, 50, 255}
	testBlue   = color.RGBA{20, 40, 60, 255}
	testWhite  = color.RGBA{255, 255, 255, 255}
)

// newTestSuperCanvas returns a width x height canvas whose palette holds colors.
func newTestSuperCanvas(width, height int, colors ...color.RGBA) *SupersampledCanvas {
	c := NewMapCanvas()
	for _, col := range colors {
		c.SetColor(col.R, col.G, col.B)
	}
	c.Resize(width, height)
	return c
}

func pixelAt(c *SupersampledCanvas, x, y int) color.RGBA { return c.Image().(*image.RGBA).RGBAAt(x, y) }

func TestNewMapCanvasPaletteIsBackgroundThenColorsInFirstUseOrder(t *testing.T) {
	c := NewMapCanvas()
	c.SetColor(1, 2, 3)
	c.SetColor(4, 5, 6)
	c.SetColor(1, 2, 3)
	want := []color.RGBA{{0, 0, 0, 255}, {1, 2, 3, 255}, {4, 5, 6, 255}}
	palette := c.big.Palette()
	if len(palette) != len(want) {
		t.Fatalf("palette = %v, want %v", palette, want)
	}
	for i, w := range want {
		if palette[i] != w {
			t.Errorf("palette[%d] = %v, want %v", i, palette[i], w)
		}
	}
}

func TestSupersampledFillsAreExactAndOpaque(t *testing.T) {
	c := newTestSuperCanvas(8, 8, testOrange)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawRectangle(2, 2, 4, 4)
	c.Fill()

	if got := c.Image().Bounds(); got != image.Rect(0, 0, 8, 8) {
		t.Fatalf("image bounds = %v, want the size given to Resize", got)
	}
	if got := pixelAt(c, 3, 3); got != testOrange {
		t.Errorf("interior = %v, want exactly %v", got, testOrange)
	}
	if got := pixelAt(c, 0, 0); got != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("background = %v, want opaque black", got)
	}
}

// A shape edge halfway through a pixel blends to half color.
func TestSupersampledPartialCoverageBlends(t *testing.T) {
	c := newTestSuperCanvas(8, 8, testOrange)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawRectangle(1.5, 2, 4, 4)
	c.Fill()

	if got, want := pixelAt(c, 1, 3), (color.RGBA{100, 50, 25, 255}); got != want {
		t.Errorf("edge pixel = %v, want half of the color over black, %v", got, want)
	}
}

// Two fills that share an edge leave no seam: the pixel on the edge is the mix of the two, not darkened by the background.
func TestSupersampledAdjacentFillsLeaveNoSeam(t *testing.T) {
	c := newTestSuperCanvas(8, 8, testOrange, testBlue)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawRectangle(0, 0, 3.5, 8)
	c.Fill()
	c.SetColor(testBlue.R, testBlue.G, testBlue.B)
	c.DrawRectangle(3.5, 0, 4.5, 8)
	c.Fill()

	want := color.RGBA{(200 + 20) / 2, (100 + 40) / 2, (50 + 60) / 2, 255}
	if got := pixelAt(c, 3, 4); got != want {
		t.Errorf("shared-edge pixel = %v, want the even mix %v", got, want)
	}
}

// Line widths are in final pixels: a 3 px line centered on a pixel row covers that row and the rows beside it, and no more.
func TestSupersampledLineWidthIsInFinalPixels(t *testing.T) {
	c := newTestSuperCanvas(9, 9, testWhite)
	c.SetColor(testWhite.R, testWhite.G, testWhite.B)
	c.SetLineWidth(3)
	c.DrawLine(0, 4.5, 9, 4.5)
	c.Stroke()

	for y, want := range map[int]color.RGBA{2: {0, 0, 0, 255}, 3: testWhite, 4: testWhite, 5: testWhite, 6: {0, 0, 0, 255}} {
		if got := pixelAt(c, 4, y); got != want {
			t.Errorf("row %d = %v, want %v", y, got, want)
		}
	}
}

// A polygon's radius is in final pixels, like its center.
func TestSupersampledPolygonRadiusIsInFinalPixels(t *testing.T) {
	c := newTestSuperCanvas(8, 8, testOrange)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawRegularPolygon(6, 4, 4, 3, math.Pi/2)
	c.Fill()

	if got := pixelAt(c, 5, 4); got != testOrange {
		t.Errorf("pixel 1.5 px from the center of a radius 3 hexagon = %v, want %v", got, testOrange)
	}
	if got := pixelAt(c, 0, 0); got != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("corner = %v, want untouched", got)
	}
}

// Drawing after the image was read shows up in the next read.
func TestSupersampledImageReflectsLaterDrawing(t *testing.T) {
	c := newTestSuperCanvas(8, 8, testOrange, testBlue)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawRectangle(0, 0, 4, 8)
	c.Fill()
	_ = c.Image()

	c.SetColor(testBlue.R, testBlue.G, testBlue.B)
	c.DrawRectangle(4, 0, 4, 8)
	c.Fill()
	if got := pixelAt(c, 6, 4); got != testBlue {
		t.Errorf("after a second fill = %v, want %v", got, testBlue)
	}
	c.DrawLine(0, 1, 8, 1)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.SetLineWidth(2)
	c.Stroke()
	if got := pixelAt(c, 6, 1); got != testOrange {
		t.Errorf("after a stroke = %v, want %v", got, testOrange)
	}
}

// Text is drawn at the final size over the averaged shapes, with no blending, even if it was queued before them.
func TestSupersampledTextIsSolidAndOverShapes(t *testing.T) {
	c := newTestSuperCanvas(60, 20, testOrange, testWhite)
	c.SetColor(testWhite.R, testWhite.G, testWhite.B)
	c.DrawString("Rome", 5, 14)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawRectangle(0, 0, 60, 20)
	c.Fill()

	white, orange := 0, 0
	for y := 0; y < 20; y++ {
		for x := 0; x < 60; x++ {
			switch pixelAt(c, x, y) {
			case testWhite:
				white++
			case testOrange:
				orange++
			default:
				t.Fatalf("pixel (%d, %d) = %v, want exactly the text or the fill color", x, y, pixelAt(c, x, y))
			}
		}
	}
	if white == 0 || orange == 0 {
		t.Errorf("white text pixels = %d, orange fill pixels = %d, want both", white, orange)
	}
}

func TestSupersampledResizeClearsShapesAndText(t *testing.T) {
	c := newTestSuperCanvas(8, 8, testOrange)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawRectangle(0, 0, 8, 8)
	c.Fill()
	c.DrawString("x", 1, 6)
	_ = c.Image()

	c.Resize(4, 4)
	if got := c.Image().Bounds(); got != image.Rect(0, 0, 4, 4) {
		t.Fatalf("bounds = %v, want 4x4", got)
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if got := pixelAt(c, x, y); got != (color.RGBA{0, 0, 0, 255}) {
				t.Fatalf("pixel (%d, %d) after Resize = %v, want opaque black", x, y, got)
			}
		}
	}
}

// A painted pixel is exactly that color in the final image, and its neighbors are untouched.
func TestSupersampledPaintPixelSetsOneFinalPixel(t *testing.T) {
	c := NewMapCanvas()
	c.Resize(6, 6)
	c.PaintPixel(2, 3, c.IndexFor(200, 100, 50))

	if got := pixelAt(c, 2, 3); got != testOrange {
		t.Errorf("painted pixel = %v, want exactly %v", got, testOrange)
	}
	black := color.RGBA{0, 0, 0, 255}
	for _, p := range [][2]int{{1, 3}, {3, 3}, {2, 2}, {2, 4}, {0, 0}} {
		if got := pixelAt(c, p[0], p[1]); got != black {
			t.Errorf("pixel (%d, %d) = %v, want untouched", p[0], p[1], got)
		}
	}

	c.PaintPixel(0, 0, c.IndexFor(200, 100, 50)) // after the image was read
	if got := pixelAt(c, 0, 0); got != testOrange {
		t.Errorf("pixel painted after reading the image = %v, want %v", got, testOrange)
	}
}

// Painted pixels keep their place in the drawing order: a fill drawn afterward covers them.
func TestSupersampledPaintPixelKeepsDrawingOrder(t *testing.T) {
	c := NewMapCanvas()
	c.Resize(6, 6)
	c.PaintPixel(1, 1, c.IndexFor(200, 100, 50))
	c.SetColor(testBlue.R, testBlue.G, testBlue.B)
	c.DrawRectangle(1, 1, 1, 1)
	c.Fill()
	c.PaintPixel(4, 4, c.IndexFor(200, 100, 50))

	if got := pixelAt(c, 1, 1); got != testBlue {
		t.Errorf("pixel painted, then filled over = %v, want the fill %v", got, testBlue)
	}
	if got := pixelAt(c, 4, 4); got != testOrange {
		t.Errorf("pixel painted last = %v, want %v", got, testOrange)
	}
}

// A triangle is given in final-image pixels, like every other shape: its half-resolution edge blends to half color.
func TestSupersampledTriangleIsInFinalPixels(t *testing.T) {
	c := newTestSuperCanvas(8, 8, testOrange)
	c.SetColor(testOrange.R, testOrange.G, testOrange.B)
	c.DrawTriangle(1, 1, 7, 1, 1, 7) // a right triangle whose hypotenuse passes through pixel corners
	c.Fill()

	if got := pixelAt(c, 2, 2); got != testOrange {
		t.Errorf("inside = %v, want %v", got, testOrange)
	}
	if got := pixelAt(c, 6, 6); got != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("outside the hypotenuse = %v, want the background", got)
	}
	if got := pixelAt(c, 0, 0); got != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("outside the corner = %v, want the background", got)
	}
}
