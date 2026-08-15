package quantize

import (
	"image"
	"image/color"
	"testing"
)

func makeTestImage(colors []color.RGBA, width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	i := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, colors[i%len(colors)])
			i++
		}
	}
	return img
}

func TestQuantizeFewerColorsThanLimit(t *testing.T) {
	// Only 2 distinct colors, well under the requested palette size.
	colors := []color.RGBA{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
	}
	src := makeTestImage(colors, 4, 4)

	dst := image.NewPaletted(src.Bounds(), nil)
	q := &MedianCutQuantizer{NumColor: 16}
	q.Quantize(dst, src.Bounds(), src, image.Point{})

	if len(dst.Palette) != 2 {
		t.Errorf("Quantize() palette length = %d, want 2 (fewer colors than NumColor)", len(dst.Palette))
	}
}

func TestQuantizeReducesToRequestedPalette(t *testing.T) {
	// Many distinct colors, should be reduced to at most NumColor.
	colors := make([]color.RGBA, 0, 64)
	for r := 0; r < 4; r++ {
		for g := 0; g < 4; g++ {
			for b := 0; b < 4; b++ {
				colors = append(colors, color.RGBA{uint8(r * 60), uint8(g * 60), uint8(b * 60), 255})
			}
		}
	}
	src := makeTestImage(colors, 8, 8)

	dst := image.NewPaletted(src.Bounds(), nil)
	q := &MedianCutQuantizer{NumColor: 8}
	q.Quantize(dst, src.Bounds(), src, image.Point{})

	if len(dst.Palette) > 8 {
		t.Errorf("Quantize() palette length = %d, want <= 8", len(dst.Palette))
	}
	if len(dst.Palette) == 0 {
		t.Errorf("Quantize() palette is empty, want at least one color")
	}
}

func TestQuantizeNumColorZero(t *testing.T) {
	src := makeTestImage([]color.RGBA{{1, 2, 3, 255}}, 2, 2)
	dst := image.NewPaletted(src.Bounds(), nil)
	q := &MedianCutQuantizer{NumColor: 0}
	// medianCut returns an empty palette when NumColor is 0, but that path is
	// only reached when the image has more distinct colors than NumColor (0),
	// which is always true for any non-empty image.
	q.Quantize(dst, src.Bounds(), src, image.Point{})
	if len(dst.Palette) != 0 {
		t.Errorf("Quantize() with NumColor=0 palette length = %d, want 0", len(dst.Palette))
	}
}

func TestUseExistingPalette(t *testing.T) {
	src := makeTestImage([]color.RGBA{{10, 20, 30, 255}}, 2, 2)
	dst := image.NewPaletted(src.Bounds(), nil)
	palette := color.Palette{color.RGBA{10, 20, 30, 255}, color.RGBA{40, 50, 60, 255}}

	q := &MedianCutQuantizer{NumColor: 2}
	q.UseExistingPalette(dst, src.Bounds(), src, image.Point{}, palette)

	if len(dst.Palette) != 2 {
		t.Fatalf("UseExistingPalette() palette length = %d, want 2", len(dst.Palette))
	}
	if dst.Palette[0] != palette[0] || dst.Palette[1] != palette[1] {
		t.Errorf("UseExistingPalette() palette = %v, want %v", dst.Palette, palette)
	}
}
