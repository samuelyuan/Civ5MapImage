// Package raster draws shapes and text onto an indexed image without anti-aliasing, so every pixel is exactly a palette color.
package raster

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// textFace is the DrawString font; its glyph masks are 0 or 255, so no anti-aliasing is needed.
var textFace = basicfont.Face7x13

// glyphOpaque is the mask alpha (of 0xffff) from which a glyph pixel is drawn.
const glyphOpaque = 0x8000

// PalettedCanvas draws without anti-aliasing into an *image.Paletted, so every pixel is a palette entry.
type PalettedCanvas struct {
	*colorTable // palette lookup, shared with sibling (staging) canvases
	img         *image.Paletted
	background  uint8       // the index the canvas is cleared to
	origin      image.Point // this canvas's (0, 0) in the coordinates drawing calls use

	// Pen state.
	current   uint8 // index of the current color
	lineWidth float64

	// The path the Draw* calls build, in this canvas's pixels, until Fill or Stroke consumes it.
	subpaths []subpath
	edgeXs   []float64 // Fill's scratch space for scanline crossings

	// Region tracking, see TrackIDs.
	ids []int32
	id  int32
}

// NewPalettedCanvas returns a width x height canvas over palette, initially black.
func NewPalettedCanvas(width, height int, palette color.Palette) *PalettedCanvas {
	return newPalettedCanvas(width, height, newColorTable(palette))
}

// NewGrowingPalettedCanvas returns a canvas whose palette starts as black and gains each color drawn, up to 256. Read it via Palette or Image.
func NewGrowingPalettedCanvas(width, height int) *PalettedCanvas {
	table := newColorTable(nil)
	table.grow = true
	return newPalettedCanvas(width, height, table)
}

func newPalettedCanvas(width, height int, table *colorTable) *PalettedCanvas {
	c := &PalettedCanvas{colorTable: table, lineWidth: 1}
	c.background = c.IndexFor(0, 0, 0)
	c.Resize(width, height)
	return c
}

// NewSibling returns a blank canvas sharing c's palette and color lookup.
func (c *PalettedCanvas) NewSibling(width, height int) *PalettedCanvas {
	s := &PalettedCanvas{colorTable: c.colorTable, background: c.background, lineWidth: 1}
	s.Resize(width, height)
	return s
}

// Resize replaces the image with a blank one of the given size and drops any path and origin.
func (c *PalettedCanvas) Resize(width, height int) {
	c.img = image.NewPaletted(image.Rect(0, 0, width, height), c.palette)
	if c.background != 0 {
		for i := range c.img.Pix {
			c.img.Pix[i] = c.background
		}
	}
	c.origin = image.Point{}
	c.clearPath()
	if c.ids != nil {
		c.TrackIDs()
	}
}

// SetOrigin sets the drawing coordinates of the canvas's (0, 0), for a staging canvas.
func (c *PalettedCanvas) SetOrigin(origin image.Point) { c.origin = origin }

// local converts drawing coordinates to this canvas's pixels.
func (c *PalettedCanvas) local(x, y float64) Point {
	return Point{x - float64(c.origin.X), y - float64(c.origin.Y)}
}

// Background returns the palette index the canvas is cleared to.
func (c *PalettedCanvas) Background() uint8 { return c.background }

// TrackIDs makes Fill record the SetID region per pixel (-1 elsewhere).
func (c *PalettedCanvas) TrackIDs() {
	c.ids = make([]int32, c.img.Rect.Dx()*c.img.Rect.Dy())
	for i := range c.ids {
		c.ids[i] = -1
	}
}

// SetID sets the region id Fill records.
func (c *PalettedCanvas) SetID(id int32) { c.id = id }

// IDs returns the per-pixel region ids (row-major, -1 unfilled) while tracking; the canvas's own slice.
func (c *PalettedCanvas) IDs() []int32 { return c.ids }

func (c *PalettedCanvas) SetColor(r, g, b uint8)     { c.current = c.IndexFor(r, g, b) }
func (c *PalettedCanvas) SetLineWidth(width float64) { c.lineWidth = width }

func (c *PalettedCanvas) DrawRegularPolygon(sides int, x, y, radius, rotation float64) {
	pts := RegularPolygon(sides, x, y, radius, rotation)
	for i, p := range pts {
		pts[i] = c.local(p.X, p.Y)
	}
	c.subpaths = append(c.subpaths, subpath{pts: pts, closed: true})
}

func (c *PalettedCanvas) DrawRectangle(x, y, width, height float64) {
	c.subpaths = append(c.subpaths, subpath{closed: true, pts: []Point{
		c.local(x, y), c.local(x+width, y),
		c.local(x+width, y+height), c.local(x, y+height),
	}})
}

func (c *PalettedCanvas) DrawTriangle(x1, y1, x2, y2, x3, y3 float64) {
	c.subpaths = append(c.subpaths, subpath{closed: true, pts: []Point{c.local(x1, y1), c.local(x2, y2), c.local(x3, y3)}})
}

func (c *PalettedCanvas) DrawLine(x1, y1, x2, y2 float64) {
	c.subpaths = append(c.subpaths, subpath{pts: []Point{c.local(x1, y1), c.local(x2, y2)}})
}

func (c *PalettedCanvas) clearPath() { c.subpaths = c.subpaths[:0] }

// Fill fills the current path (even-odd, sampled at pixel centers) and clears it.
func (c *PalettedCanvas) Fill() {
	defer c.clearPath()
	minY, maxY := yRange(c.subpaths)
	bounds := c.img.Rect
	for y := max(bounds.Min.Y, pixelAtOrAfter(minY)); y < bounds.Max.Y && float64(y)+0.5 < maxY; y++ {
		yc := float64(y) + 0.5
		xs := appendCrossings(c.edgeXs[:0], c.subpaths, yc)
		for i := 0; i+1 < len(xs); i += 2 {
			x0 := max(bounds.Min.X, pixelAtOrAfter(xs[i]))
			x1 := min(bounds.Max.X, pixelAtOrAfter(xs[i+1]))
			c.fillSpan(y, x0, x1)
		}
		c.edgeXs = xs
	}
}

// fillSpan paints pixels [x0, x1) of row y with the current color, and the current id while tracking.
func (c *PalettedCanvas) fillSpan(y, x0, x1 int) {
	row := c.img.Pix[c.img.PixOffset(0, y):]
	for x := x0; x < x1; x++ {
		row[x] = c.current
	}
	if c.ids != nil {
		idRow := c.ids[y*c.img.Rect.Dx():]
		for x := x0; x < x1; x++ {
			idRow[x] = c.id
		}
	}
}

// Stroke draws the current path with round caps, then clears it.
func (c *PalettedCanvas) Stroke() {
	defer c.clearPath()
	half := c.lineWidth / 2
	for _, sp := range c.subpaths {
		for i := 0; i < sp.segmentCount(); i++ {
			c.strokeSegment(newSegment(sp.pts[i], sp.pts[(i+1)%len(sp.pts)]), half)
		}
	}
}

func (c *PalettedCanvas) strokeSegment(seg segment, half float64) {
	r := seg.pixelBounds(half).Intersect(c.img.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		lo, hi, ok := seg.rowSpan(float64(y)+0.5, half)
		if !ok {
			continue
		}
		// One pixel of slack each side keeps rounding in rowSpan from ever cutting off a pixel distSq accepts.
		for x := max(r.Min.X, int(math.Floor(lo))-1); x < min(r.Max.X, int(math.Ceil(hi))+2); x++ {
			if seg.distSq(float64(x)+0.5, float64(y)+0.5) <= half*half {
				c.setPixel(x, y, c.current)
			}
		}
	}
}

// DrawString draws text in textFace, thresholding glyph masks to solid pixels.
func (c *PalettedCanvas) DrawString(text string, x, y float64) {
	at := c.local(x, y)
	dot := fixed.Point26_6{X: fixed.Int26_6(at.X * 64), Y: fixed.Int26_6(at.Y * 64)}
	for _, r := range text {
		dr, mask, maskp, advance, ok := textFace.Glyph(dot, r)
		if !ok {
			continue
		}
		for gy := 0; gy < dr.Dy(); gy++ {
			for gx := 0; gx < dr.Dx(); gx++ {
				if _, _, _, a := mask.At(maskp.X+gx, maskp.Y+gy).RGBA(); a < glyphOpaque {
					continue
				}
				if p := image.Pt(dr.Min.X+gx, dr.Min.Y+gy); p.In(c.img.Rect) {
					c.setPixel(p.X, p.Y, c.current)
				}
			}
		}
		dot.X += advance
	}
}

// MeasureString returns the (w, h) DrawString(text, ...) would occupy.
func (c *PalettedCanvas) MeasureString(text string) (w, h float64) {
	d := &font.Drawer{Face: textFace}
	return float64(d.MeasureString(text) >> 6), float64(textFace.Height)
}

// PasteRegion copies rect from src (read at srcPoint), as raw indices when src is a same-palette *image.Paletted.
func (c *PalettedCanvas) PasteRegion(src image.Image, rect image.Rectangle, srcPoint image.Point) {
	sp, ok := src.(*image.Paletted)
	if !ok || len(sp.Palette) != len(c.palette) {
		draw.Draw(c.img, rect, src, srcPoint, draw.Src)
		return
	}
	for y := 0; y < rect.Dy(); y++ {
		copyRow(c.img, rect.Min.X, rect.Min.Y+y, sp, srcPoint.X, srcPoint.Y+y, rect.Dx())
	}
}

// Snapshot returns a copy of rect as a GIF frame positioned at rect.
func (c *PalettedCanvas) Snapshot(rect image.Rectangle) *image.Paletted {
	frame := image.NewPaletted(rect, c.palette)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		copyRow(frame, rect.Min.X, y, c.img, rect.Min.X, y, rect.Dx())
	}
	return frame
}

// copyRow copies n palette indices from src starting at (sx, sy) to dst starting at (dx, dy).
func copyRow(dst *image.Paletted, dx, dy int, src *image.Paletted, sx, sy, n int) {
	copy(dst.Pix[dst.PixOffset(dx, dy):][:n], src.Pix[src.PixOffset(sx, sy):][:n])
}

// setPixel sets the in-bounds pixel at local (x, y) to a palette index.
func (c *PalettedCanvas) setPixel(x, y int, index uint8) { c.img.Pix[c.img.PixOffset(x, y)] = index }

// IndexAt returns the palette index of the pixel at (x, y) in this canvas's own pixels, which must be in bounds.
func (c *PalettedCanvas) IndexAt(x, y int) uint8 { return c.img.Pix[c.img.PixOffset(x, y)] }

// PaintPixel sets the pixel at drawing coordinates (x, y) to a palette index, if this canvas covers it.
func (c *PalettedCanvas) PaintPixel(x, y int, index uint8) {
	if pt := image.Pt(x, y).Sub(c.origin); pt.In(c.img.Rect) {
		c.setPixel(pt.X, pt.Y, index)
	}
}

func (c *PalettedCanvas) Image() image.Image {
	c.img.Palette = c.palette // a growing palette may have gained colors since the image was made
	return c.img
}

// Palette returns the canvas's palette, which a growing canvas extends as colors are drawn.
func (c *PalettedCanvas) Palette() color.Palette { return c.palette }

func (c *PalettedCanvas) SavePNG(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.img)
}
