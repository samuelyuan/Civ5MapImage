package raster

import (
	"image"
	"image/color"
	"sync"
)

// rgb is a palette color as 8-bit channels held in wider integers, so a block's colors can be summed without overflow.
type rgb struct{ r, g, b uint32 }

// Downsample2x averages each 2x2 block of src into one opaque pixel of an image a quarter its area, in parallel bands of rows.
func Downsample2x(src *image.Paletted) *image.RGBA {
	w, h := src.Rect.Dx()/2, src.Rect.Dy()/2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	colors := paletteToRGB(src.Palette)

	var wg sync.WaitGroup
	const band = 64 // rows per goroutine
	for y0 := 0; y0 < h; y0 += band {
		wg.Go(func() { averageBlocks(src, dst, colors, y0, min(y0+band, h)) })
	}
	wg.Wait()
	return dst
}

// paletteToRGB converts a palette to rgb for summing.
func paletteToRGB(palette color.Palette) []rgb {
	colors := make([]rgb, len(palette))
	for i, col := range palette {
		r, g, b, _ := col.RGBA()
		colors[i] = rgb{r >> 8, g >> 8, b >> 8}
	}
	return colors
}

// averageBlocks sets rows y0..y1 of dst to the average of each 2x2 block of src.
func averageBlocks(src *image.Paletted, dst *image.RGBA, colors []rgb, y0, y1 int) {
	w := dst.Rect.Dx()
	for y := y0; y < y1; y++ {
		out := dst.Pix[y*dst.Stride:]
		top := src.Pix[2*y*src.Stride:]
		bottom := src.Pix[(2*y+1)*src.Stride:]
		for x := 0; x < w; x++ {
			a, b, c, d := top[2*x], top[2*x+1], bottom[2*x], bottom[2*x+1]
			px := out[x*4:][:4]
			if a == b && b == c && c == d {
				col := colors[a]
				px[0], px[1], px[2], px[3] = uint8(col.r), uint8(col.g), uint8(col.b), 255
				continue
			}
			ca, cb, cc, cd := colors[a], colors[b], colors[c], colors[d]
			px[0] = uint8((ca.r + cb.r + cc.r + cd.r + 2) >> 2)
			px[1] = uint8((ca.g + cb.g + cc.g + cd.g + 2) >> 2)
			px[2] = uint8((ca.b + cb.b + cc.b + cd.b + 2) >> 2)
			px[3] = 255
		}
	}
}
