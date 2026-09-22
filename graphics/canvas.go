package graphics

import (
	"image"

	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

var (
	_ Canvas = (*SupersampledCanvas)(nil)
	_ Canvas = (*raster.PalettedCanvas)(nil)
)

// Shapes adds geometric paths to the current path, for Fill or Stroke to paint.
type Shapes interface {
	DrawRegularPolygon(sides int, x, y, radius, rotation float64)
	DrawRectangle(x, y, width, height float64)
	DrawTriangle(x1, y1, x2, y2, x3, y3 float64)
	DrawLine(x1, y1, x2, y2 float64)
}

// Pen sets the color and line width, and paints the current path.
type Pen interface {
	SetColor(r, g, b uint8)
	SetLineWidth(width float64)
	Fill()
	Stroke()
}

// PixelPainter sets single pixels of the final image; PaintPixel takes the index IndexFor returns for a color.
type PixelPainter interface {
	IndexFor(r, g, b uint8) uint8
	PaintPixel(x, y int, index uint8)
}

// TextDrawer draws text in the current color.
type TextDrawer interface {
	DrawString(text string, x, y float64)
	// MeasureString returns the (w, h) DrawString(text, ...) would occupy.
	MeasureString(text string) (w, h float64)
}

// Surface is the image behind a canvas: its size and its final output.
type Surface interface {
	Resize(width, height int)
	Image() image.Image
	SavePNG(filename string) error
}

// Painter is what drawing filled and stroked shapes needs.
type Painter interface {
	Shapes
	Pen
}

// TextPainter is what drawing colored text needs.
type TextPainter interface {
	Pen
	TextDrawer
}

// Canvas is the full drawing surface; functions take the smallest interface that covers what they draw.
type Canvas interface {
	Shapes
	Pen
	PixelPainter
	TextDrawer
	Surface
}
