package graphics

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// Text must match DrawingContext's exactly: same glyph pixels, same measured size.
func TestPalettedCanvasDrawStringMatchesDrawingContext(t *testing.T) {
	palette := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{250, 240, 230, 255}}
	ours := raster.NewPalettedCanvas(120, 30, palette)
	ref := NewDrawingContext(120, 30)
	for _, c := range []Canvas{ours, ref} {
		c.SetColor(250, 240, 230)
		c.DrawString("Samarkand 42", 5.4, 20.6)
	}
	refImage := ref.Image().(*image.RGBA)
	for y := 0; y < 30; y++ {
		for x := 0; x < 120; x++ {
			want := refImage.RGBAAt(x, y).A != 0
			got := ours.IndexAt(x, y) != 0
			if got != want {
				t.Fatalf("pixel (%d, %d): painted=%v, DrawingContext painted=%v", x, y, got, want)
			}
		}
	}
	gw, gh := ref.MeasureString("Samarkand 42")
	if ow, oh := ours.MeasureString("Samarkand 42"); ow != gw || oh != gh {
		t.Errorf("MeasureString = (%v, %v), DrawingContext = (%v, %v)", ow, oh, gw, gh)
	}
}

// Fills must cover exactly the pixels DrawingContext covers fully and leave empty ones alone (its anti-aliased edge pixels may go either way).
func TestPalettedCanvasFillsMatchDrawingContext(t *testing.T) {
	palette := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{200, 100, 50, 255}}
	ours := raster.NewPalettedCanvas(90, 90, palette)
	ref := NewDrawingContext(90, 90)
	for _, c := range []Canvas{ours, ref} {
		c.SetColor(200, 100, 50)
		c.DrawRegularPolygon(6, 30.3, 40.2, 16, math.Pi/2)
		c.DrawRegularPolygon(3, 62.7, 25.9, 14, math.Pi)
		c.DrawRectangle(50.2, 55.5, 20.3, 18.1)
		c.Fill()
	}
	refImage := ref.Image().(*image.RGBA)
	for y := 0; y < 90; y++ {
		for x := 0; x < 90; x++ {
			px := refImage.RGBAAt(x, y)
			got := ours.IndexAt(x, y) != 0
			switch {
			case px.A == 255 && px.R == 200 && px.G == 100 && px.B == 50 && !got:
				t.Fatalf("pixel (%d, %d): DrawingContext fully covers it, ours is empty", x, y)
			case px.A == 0 && got:
				t.Fatalf("pixel (%d, %d): DrawingContext leaves it empty, ours is painted", x, y)
			}
		}
	}
}
