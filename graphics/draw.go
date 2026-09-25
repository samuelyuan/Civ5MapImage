package graphics

import (
	"image/color"
	"math"
)

// Filled and stroked shapes drawn in one call, on any Canvas.

// fillHex fills the pointy-top hex of the given radius centered on hex in its color.
func fillHex(canvas Canvas, hex HexTile, radius float64) {
	canvas.DrawRegularPolygon(6, hex.X, hex.Y, radius, math.Pi/2)
	canvas.SetColor(hex.Fill.R, hex.Fill.G, hex.Fill.B)
	canvas.Fill()
}

// drawDisc fills a circle.
func drawDisc(canvas Canvas, x, y, radius float64, fill color.RGBA) {
	canvas.DrawRegularPolygon(24, x, y, radius, 0)
	canvas.SetColor(fill.R, fill.G, fill.B)
	canvas.Fill()
}

// fillTriangle fills the triangle with the given corners.
func fillTriangle(canvas Canvas, fill color.RGBA, x1, y1, x2, y2, x3, y3 float64) {
	canvas.DrawTriangle(x1, y1, x2, y2, x3, y3)
	canvas.SetColor(fill.R, fill.G, fill.B)
	canvas.Fill()
}

// strokeSegment draws a line of the given width and color.
func strokeSegment(canvas Canvas, line Line, width float64, r, g, b uint8) {
	canvas.SetLineWidth(width)
	canvas.SetColor(r, g, b)
	canvas.DrawLine(line.X1, line.Y1, line.X2, line.Y2)
	canvas.Stroke()
}

// blendColor linearly interpolates between two colors by t (0 = c1, 1 = c2).
func blendColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		uint8(float64(c1.R) + (float64(c2.R)-float64(c1.R))*t),
		uint8(float64(c1.G) + (float64(c2.G)-float64(c1.G))*t),
		uint8(float64(c1.B) + (float64(c2.B)-float64(c1.B))*t),
		255,
	}
}

// contrastRatio is the WCAG contrast ratio of two colors, from 1 (the same) to 21 (black on white).
func contrastRatio(a, b color.RGBA) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// relativeLuminance is WCAG 2.x's formula (sRGB gamma-decoded, then ITU-R BT.709 luma weights); 0.03928 is WCAG's own value, not a typo for sRGB's 0.04045.
func relativeLuminance(c color.RGBA) float64 {
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}
