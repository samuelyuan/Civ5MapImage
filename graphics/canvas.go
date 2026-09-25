package graphics

import (
	"image"

	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

var (
	_ Canvas = (*SupersampledCanvas)(nil)
	_ Canvas = (*raster.PalettedCanvas)(nil)
)

// Canvas is what the renderers draw on. The Draw* shape calls add to a path that Fill or Stroke paints in the current color, text is drawn in the current color, and single pixels are set by palette index.
type Canvas interface {
	DrawRegularPolygon(sides int, x, y, radius, rotation float64)
	DrawRectangle(x, y, width, height float64)
	DrawTriangle(x1, y1, x2, y2, x3, y3 float64)
	DrawLine(x1, y1, x2, y2 float64)
	SetColor(r, g, b uint8)
	SetLineWidth(width float64)
	Fill()
	Stroke()

	IndexFor(r, g, b uint8) uint8 // the palette index PaintPixel takes for a color
	PaintPixel(x, y int, index uint8)

	DrawString(text string, x, y float64)
	MeasureString(text string) (w, h float64) // the (w, h) DrawString would occupy

	Resize(width, height int)
	Image() image.Image
	SavePNG(filename string) error
}
