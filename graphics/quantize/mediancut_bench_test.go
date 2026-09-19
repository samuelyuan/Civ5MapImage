package quantize

import (
	"image"
	"image/color"
	"testing"
)

// benchPalettedSet is the old per-pixel dst.Set(...) path, benchmarked against PaletteMapper.Fill.
func benchPalettedSet(dst *image.Paletted, r image.Rectangle, src *image.RGBA, sp image.Point) {
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			dst.Set(sp.X+x, sp.Y+y, src.At(r.Min.X+x, r.Min.Y+y))
		}
	}
}

// buildRealisticFrame builds a flat-fill image approximating a real quickreplay frame.
func buildRealisticFrame(width, height, numColors int) (*image.RGBA, color.Palette) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	palette := make(color.Palette, numColors)
	for i := 0; i < numColors; i++ {
		palette[i] = color.RGBA{uint8(i * 7 % 256), uint8(i * 13 % 256), uint8(i * 29 % 256), 255}
	}
	rowsPerColor := height / numColors
	if rowsPerColor == 0 {
		rowsPerColor = 1
	}
	for y := 0; y < height; y++ {
		c := palette[(y/rowsPerColor)%numColors].(color.RGBA)
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img, palette
}

func BenchmarkOldPalettedSet(b *testing.B) {
	src, palette := buildRealisticFrame(837, 484, 120)
	r := src.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst := image.NewPaletted(r, palette)
		benchPalettedSet(dst, r, src, image.Point{})
	}
}

func BenchmarkPaletteMapperFill(b *testing.B) {
	src, palette := buildRealisticFrame(837, 484, 120)
	r := src.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst := image.NewPaletted(r, palette)
		NewPaletteMapper(palette).Fill(dst, r, src, image.Point{})
	}
}

func BenchmarkBuildPaletteMedianCut(b *testing.B) {
	src, _ := flatPlusNoise()
	big := image.NewRGBA(image.Rect(0, 0, 1600, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 1600; x++ {
			big.SetRGBA(x, y, src.RGBAAt(x%200, y%200))
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildPalette(big, big.Bounds(), 256)
	}
}
