package graphics

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// mapSupersample is the factor a map is drawn larger before averaging down; raster.Downsample2x assumes 2.
const mapSupersample = 2

// SupersampledCanvas draws in final-image coordinates on a mapSupersample-times larger canvas and averages blocks down to opaque pixels; text is drawn afterward at final size.
type SupersampledCanvas struct {
	big   *raster.PalettedCanvas
	text  []textOp
	color color.RGBA // the current color, for text
	final *image.RGBA
}

// textOp is a DrawString call, kept until the shapes are averaged down.
type textOp struct {
	text string
	x, y float64
	c    color.RGBA
}

// NewMapCanvas returns a canvas whose palette grows to the colors drawn on it.
func NewMapCanvas() *SupersampledCanvas {
	return &SupersampledCanvas{big: raster.NewGrowingPalettedCanvas(1, 1)}
}

func (c *SupersampledCanvas) DrawRegularPolygon(sides int, x, y, radius, rotation float64) {
	c.big.DrawRegularPolygon(sides, x*mapSupersample, y*mapSupersample, radius*mapSupersample, rotation)
}

func (c *SupersampledCanvas) DrawRectangle(x, y, width, height float64) {
	c.big.DrawRectangle(x*mapSupersample, y*mapSupersample, width*mapSupersample, height*mapSupersample)
}

func (c *SupersampledCanvas) DrawTriangle(x1, y1, x2, y2, x3, y3 float64) {
	c.big.DrawTriangle(x1*mapSupersample, y1*mapSupersample, x2*mapSupersample, y2*mapSupersample, x3*mapSupersample, y3*mapSupersample)
}

func (c *SupersampledCanvas) DrawLine(x1, y1, x2, y2 float64) {
	c.big.DrawLine(x1*mapSupersample, y1*mapSupersample, x2*mapSupersample, y2*mapSupersample)
}

func (c *SupersampledCanvas) SetColor(r, g, b uint8) {
	c.color = color.RGBA{r, g, b, 255}
	c.big.SetColor(r, g, b)
}

func (c *SupersampledCanvas) SetLineWidth(width float64) { c.big.SetLineWidth(width * mapSupersample) }

func (c *SupersampledCanvas) Fill() {
	c.final = nil
	c.big.Fill()
}

func (c *SupersampledCanvas) Stroke() {
	c.final = nil
	c.big.Stroke()
}

func (c *SupersampledCanvas) IndexFor(r, g, b uint8) uint8 { return c.big.IndexFor(r, g, b) }

// PaintPixel sets the pixel of the final image at (x, y), which is a block of the big canvas.
func (c *SupersampledCanvas) PaintPixel(x, y int, index uint8) {
	c.final = nil
	for dy := 0; dy < mapSupersample; dy++ {
		for dx := 0; dx < mapSupersample; dx++ {
			c.big.PaintPixel(x*mapSupersample+dx, y*mapSupersample+dy, index)
		}
	}
}

// Resize replaces the image with a blank width x height one.
func (c *SupersampledCanvas) Resize(width, height int) {
	c.big.Resize(width*mapSupersample, height*mapSupersample)
	c.text = nil
	c.final = nil
}

// DrawString queues text drawn over all shapes at final size, unscaled and unblended.
func (c *SupersampledCanvas) DrawString(text string, x, y float64) {
	c.final = nil
	c.text = append(c.text, textOp{text, x, y, c.color})
}

func (c *SupersampledCanvas) MeasureString(text string) (w, h float64) {
	return c.big.MeasureString(text)
}

// Image returns the opaque final image: the shapes averaged down, with the text over them.
func (c *SupersampledCanvas) Image() image.Image {
	if c.final == nil {
		c.final = c.render()
	}
	return c.final
}

func (c *SupersampledCanvas) SavePNG(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	// BestSpeed: about a third faster than the default for a file only 6% larger.
	return (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(f, c.Image())
}

// render downsamples the big canvas 2x2 to a pixel, then draws the text.
func (c *SupersampledCanvas) render() *image.RGBA {
	dst := raster.Downsample2x(c.big.Image().(*image.Paletted))
	for _, t := range c.text {
		d := font.Drawer{Dst: dst, Src: image.NewUniform(t.c), Face: basicfont.Face7x13,
			Dot: fixed.Point26_6{X: fixed.Int26_6(math.Floor(t.x * 64)), Y: fixed.Int26_6(math.Floor(t.y * 64))}}
		d.DrawString(t.text)
	}
	return dst
}
