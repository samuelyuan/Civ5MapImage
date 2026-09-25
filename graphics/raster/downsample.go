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
	const band = 64
	for firstRow := 0; firstRow < h; firstRow += band {
		wg.Go(func() { averageBlocks(src, dst, colors, firstRow, min(firstRow+band, h)) })
	}
	wg.Wait()
	return dst
}

func paletteToRGB(palette color.Palette) []rgb {
	colors := make([]rgb, len(palette))
	for i, col := range palette {
		r, g, b, _ := col.RGBA()
		colors[i] = rgb{r >> 8, g >> 8, b >> 8}
	}
	return colors
}

// averageBlocks sets output rows firstRow..endRow of dst to the average of each 2x2 block of src.
// Pix is a flat array, so row r starts at r*Stride; output row y covers source rows 2y and 2y+1.
func averageBlocks(src *image.Paletted, dst *image.RGBA, colors []rgb, firstRow, endRow int) {
	width := dst.Rect.Dx()
	for y := firstRow; y < endRow; y++ {
		dstRow := dst.Pix[y*dst.Stride:]
		srcTopRow := src.Pix[2*y*src.Stride:]
		srcBottomRow := src.Pix[(2*y+1)*src.Stride:]
		for x := 0; x < width; x++ {
			// The block's four palette indices: output column x covers source columns 2x and 2x+1.
			topLeft, topRight := srcTopRow[2*x], srcTopRow[2*x+1]
			bottomLeft, bottomRight := srcBottomRow[2*x], srcBottomRow[2*x+1]
			outRGBA := dstRow[x*4:][:4]
			if topLeft == topRight && topRight == bottomLeft && bottomLeft == bottomRight { // a flat block: no averaging needed
				flat := colors[topLeft]
				outRGBA[0], outRGBA[1], outRGBA[2], outRGBA[3] = uint8(flat.r), uint8(flat.g), uint8(flat.b), 255
				continue
			}
			tl, tr, bl, br := colors[topLeft], colors[topRight], colors[bottomLeft], colors[bottomRight]
			// Sum of four, +2 to round, >>2 to divide by four.
			outRGBA[0] = uint8((tl.r + tr.r + bl.r + br.r + 2) >> 2)
			outRGBA[1] = uint8((tl.g + tr.g + bl.g + br.g + 2) >> 2)
			outRGBA[2] = uint8((tl.b + tr.b + bl.b + br.b + 2) >> 2)
			outRGBA[3] = 255
		}
	}
}
