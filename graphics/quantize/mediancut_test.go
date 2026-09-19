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

func TestBuildPaletteFewerColorsThanLimit(t *testing.T) {
	src := makeTestImage([]color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}}, 4, 4)

	palette := BuildPalette(src, src.Bounds(), 16)

	if len(palette) != 2 {
		t.Errorf("BuildPalette() length = %d, want 2 (fewer colors than the limit)", len(palette))
	}
}

func TestBuildPaletteReducesToLimit(t *testing.T) {
	colors := make([]color.RGBA, 0, 64)
	for r := 0; r < 4; r++ {
		for g := 0; g < 4; g++ {
			for b := 0; b < 4; b++ {
				colors = append(colors, color.RGBA{uint8(r * 60), uint8(g * 60), uint8(b * 60), 255})
			}
		}
	}
	src := makeTestImage(colors, 8, 8)

	palette := BuildPalette(src, src.Bounds(), 8)

	if len(palette) == 0 || len(palette) > 8 {
		t.Errorf("BuildPalette() length = %d, want 1..8", len(palette))
	}
}

func TestBuildPaletteNothingToBuild(t *testing.T) {
	src := makeTestImage([]color.RGBA{{1, 2, 3, 255}}, 2, 2)

	if got := BuildPalette(src, src.Bounds(), 0); len(got) != 0 {
		t.Errorf("BuildPalette() with maxColors=0 length = %d, want 0", len(got))
	}
	if got := BuildPalette(src, image.Rect(10, 10, 20, 20), 8); len(got) != 0 {
		t.Errorf("BuildPalette() over a region outside src length = %d, want 0", len(got))
	}
}

func TestPaletteMapperFill(t *testing.T) {
	src := makeTestImage([]color.RGBA{{10, 20, 30, 255}, {40, 50, 60, 255}}, 4, 2)
	palette := color.Palette{color.RGBA{40, 50, 60, 255}, color.RGBA{10, 20, 30, 255}}
	dst := image.NewPaletted(src.Bounds(), nil)

	NewPaletteMapper(palette).Fill(dst, src.Bounds(), src, image.Point{})

	if len(dst.Palette) != 2 {
		t.Fatalf("Fill() palette length = %d, want 2", len(dst.Palette))
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			want := palette.Index(src.RGBAAt(x, y))
			if got := int(dst.ColorIndexAt(x, y)); got != want {
				t.Errorf("Fill() pixel (%d,%d) index = %d, want %d", x, y, got, want)
			}
		}
	}
}

func TestPaletteMapperFillSubRectangle(t *testing.T) {
	src := makeTestImage([]color.RGBA{{10, 20, 30, 255}, {40, 50, 60, 255}}, 4, 4)
	palette := color.Palette{color.RGBA{10, 20, 30, 255}, color.RGBA{40, 50, 60, 255}}
	rect := image.Rect(1, 1, 3, 3)
	dst := image.NewPaletted(rect, nil)

	NewPaletteMapper(palette).Fill(dst, rect, src, rect.Min)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if got, want := int(dst.ColorIndexAt(x, y)), palette.Index(src.RGBAAt(x, y)); got != want {
				t.Errorf("Fill() pixel (%d,%d) index = %d, want %d", x, y, got, want)
			}
		}
	}
}

// flatPlusNoise returns an image of a few flat colors plus many one-off colors (like map fills and
// anti-aliased edges), which needs real median cut.
func flatPlusNoise() (*image.RGBA, []color.RGBA) {
	flat := []color.RGBA{{20, 20, 20, 255}, {90, 150, 148, 255}, {230, 230, 230, 255}, {105, 125, 54, 255}, {200, 200, 164, 255}}
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	seed := uint32(1)
	next := func() uint8 {
		seed = seed*1664525 + 1013904223
		return uint8(seed >> 24)
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			if (x+y)%10 == 0 {
				img.SetRGBA(x, y, color.RGBA{next(), next(), next(), 255})
			} else {
				img.SetRGBA(x, y, flat[(y/40)%len(flat)])
			}
		}
	}
	return img, flat
}

// TestBuildPaletteKeepsDominantColorsExact checks that colors covering most of the image get their
// own palette entry even when thousands of rare colors compete.
func TestBuildPaletteKeepsDominantColorsExact(t *testing.T) {
	img, flat := flatPlusNoise()
	palette := BuildPalette(img, img.Bounds(), 256)
	if len(palette) != 256 {
		t.Fatalf("palette has %d entries, want 256", len(palette))
	}
	dst := image.NewPaletted(img.Bounds(), nil)
	NewPaletteMapper(palette).Fill(dst, img.Bounds(), img, image.Point{})

	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			if (x+y)%10 == 0 {
				continue
			}
			want := flat[(y/40)%len(flat)]
			gr, gg, gb, _ := dst.At(x, y).RGBA()
			if uint8(gr>>8) != want.R || uint8(gg>>8) != want.G || uint8(gb>>8) != want.B {
				t.Fatalf("flat pixel (%d,%d) mapped to %v, want exactly %v", x, y, dst.At(x, y), want)
			}
		}
	}
}

// TestBuildPaletteDeterministic guards against map iteration order leaking into the palette.
func TestBuildPaletteDeterministic(t *testing.T) {
	img, _ := flatPlusNoise()
	first := BuildPalette(img, img.Bounds(), 256)
	for i := 1; i < 5; i++ {
		got := BuildPalette(img, img.Bounds(), 256)
		for j := range first {
			if first[j] != got[j] {
				t.Fatalf("run %d: palette entry %d = %v, first run had %v", i, j, got[j], first[j])
			}
		}
	}
}
