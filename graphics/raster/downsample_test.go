package raster

import (
	"bytes"
	"image"
	"image/color"
	"math/rand"
	"testing"
)

// randomBlocks returns a paletted image of flat regions with scattered single-pixel noise, so some 2x2 blocks are one color and some are not.
func randomBlocks(rng *rand.Rand, w, h int) (*image.Paletted, []rgb) {
	palette := make(color.Palette, 11)
	colors := make([]rgb, len(palette))
	for i := range palette {
		c := color.RGBA{uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256)), 255}
		palette[i] = c
		colors[i] = rgb{uint32(c.R), uint32(c.G), uint32(c.B)}
	}
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			index := uint8((x/6 + y/5) % len(palette))
			if rng.Intn(6) == 0 {
				index = uint8(rng.Intn(len(palette)))
			}
			img.Pix[y*img.Stride+x] = index
		}
	}
	return img, colors
}

// averageBlocks gives, byte for byte, the plain rounded mean of each block's four colors.
func TestAverageBlocksMatchesThePlainMeanOfEachBlock(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for round := 0; round < 20; round++ {
		src, colors := randomBlocks(rng, 60, 40)
		got := image.NewRGBA(image.Rect(0, 0, 30, 20))
		averageBlocks(src, got, colors, 0, 20)

		want := image.NewRGBA(image.Rect(0, 0, 30, 20))
		for y := 0; y < 20; y++ {
			for x := 0; x < 30; x++ {
				var r, g, b uint32
				for _, p := range [4]image.Point{{2 * x, 2 * y}, {2*x + 1, 2 * y}, {2 * x, 2*y + 1}, {2*x + 1, 2*y + 1}} {
					c := colors[src.ColorIndexAt(p.X, p.Y)]
					r, g, b = r+c.r, g+c.g, b+c.b
				}
				want.SetRGBA(x, y, color.RGBA{uint8((r + 2) / 4), uint8((g + 2) / 4), uint8((b + 2) / 4), 255})
			}
		}
		if !bytes.Equal(got.Pix, want.Pix) {
			t.Fatalf("round %d: averageBlocks differs from the plain mean", round)
		}
	}
}

// Downsample2x gives the same result whether it runs as one band or several: dst.Rect.Dx() spans exactly two bands.
func TestDownsample2xMatchesAcrossBandBoundaries(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	src, _ := randomBlocks(rng, 40, 200) // 100 output rows: two 64-row bands

	got := Downsample2x(src)

	want := image.NewRGBA(image.Rect(0, 0, got.Rect.Dx(), got.Rect.Dy()))
	averageBlocks(src, want, paletteToRGB(src.Palette), 0, got.Rect.Dy())
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Error("Downsample2x() in parallel bands differs from one pass over every row")
	}
}

// Downsample2x always produces an opaque image at a quarter the source's area.
func TestDownsample2xSizeAndOpacity(t *testing.T) {
	src := image.NewPaletted(image.Rect(0, 0, 12, 8), color.Palette{color.RGBA{10, 20, 30, 255}})

	got := Downsample2x(src)

	if got.Rect.Dx() != 6 || got.Rect.Dy() != 4 {
		t.Fatalf("size = %v, want 6x4", got.Rect)
	}
	for _, px := range [][2]int{{0, 0}, {5, 3}} {
		if got.RGBAAt(px[0], px[1]) != (color.RGBA{10, 20, 30, 255}) {
			t.Errorf("pixel %v = %v, want the source's only color, opaque", px, got.RGBAAt(px[0], px[1]))
		}
	}
}
