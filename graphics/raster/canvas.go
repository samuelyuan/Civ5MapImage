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

// textFace has glyph masks of 0 or 255 only, so text needs no anti-aliasing.
var textFace = basicfont.Face7x13

// glyphOpaque is the mask alpha (of 0xffff) from which a glyph pixel is drawn.
const glyphOpaque = 0x8000

// PalettedCanvas draws into an *image.Paletted; every pixel is a palette entry.
type PalettedCanvas struct {
	*colorTable // palette lookup, shared with sibling (staging) canvases
	img         *image.Paletted
	background  uint8
	origin      image.Point // this canvas's (0, 0) in drawing coordinates

	current   uint8
	lineWidth float64

	// subpaths are in this canvas's pixels until Fill or Stroke consumes them.
	subpaths   []subpath
	crossingXs []float64 // scratch buffer for the x crossings of one scanline

	ids []int32
	id  int32
}

// NewPalettedCanvas returns a canvas over palette, initially black.
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

// Resize blanks the image at the new size and drops any path and origin.
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

// SetOrigin places the canvas's (0, 0) in drawing coordinates, for a staging canvas.
func (c *PalettedCanvas) SetOrigin(origin image.Point) { c.origin = origin }

// local converts drawing coordinates to this canvas's pixels.
func (c *PalettedCanvas) local(x, y float64) Point {
	return Point{x - float64(c.origin.X), y - float64(c.origin.Y)}
}

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
	c.fillSubpaths(c.subpaths)
}

func (c *PalettedCanvas) fillSubpaths(subpaths []subpath) {
	minY, maxY := yRange(subpaths)
	bounds := c.img.Rect
	for y := max(bounds.Min.Y, pixelAtOrAfter(minY)); y < bounds.Max.Y && float64(y)+0.5 < maxY; y++ {
		yc := float64(y) + 0.5
		xs := appendCrossings(c.crossingXs[:0], subpaths, yc)
		for i := 0; i+1 < len(xs); i += 2 {
			x0 := max(bounds.Min.X, pixelAtOrAfter(xs[i]))
			x1 := min(bounds.Max.X, pixelAtOrAfter(xs[i+1]))
			c.fillSpan(y, x0, x1)
		}
		c.crossingXs = xs
	}
}

// fillSpan also records the current id while tracking.
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

// Stroke draws the current path as lines of the current width with square caps, then clears it.
func (c *PalettedCanvas) Stroke() {
	defer c.clearPath()
	half := c.lineWidth / 2
	origin := Point{float64(c.origin.X), float64(c.origin.Y)}
	for _, sp := range c.subpaths {
		for i := 0; i < sp.segmentCount(); i++ {
			a, b := sp.pts[i], sp.pts[(i+1)%len(sp.pts)]
			// Built in drawing coordinates, so a staging canvas gets the very polygon the full canvas does, even where a pixel center sits exactly on an edge.
			line := thickLine(a.plus(origin), b.plus(origin), half)
			for j, p := range line.pts {
				line.pts[j] = p.minus(origin)
			}
			c.fillSubpaths([]subpath{line})
		}
	}
}

// DrawString draws text in textFace, a bitmap font. The pen starts at (x, y) on the baseline; each rune's glyph is a pixel mask placed at the pen rounded to a whole pixel,
// and the pen then advances a fixed width. Nothing is blended, so every pixel stays a palette color. Runes the font lacks are skipped.
func (c *PalettedCanvas) DrawString(text string, x, y float64) {
	at := c.local(x, y)
	pen := fixed.Point26_6{X: floorFixed(at.X), Y: floorFixed(at.Y)}
	for _, r := range text {
		bounds, mask, maskOrigin, advance, ok := textFace.Glyph(pen, r)
		if !ok {
			continue
		}
		c.blitGlyph(bounds, mask, maskOrigin)
		pen.X += advance
	}
}

// floorFixed converts to 26.6 fixed point rounding down, so a negative staging-local position rounds the same as on the full canvas.
func floorFixed(v float64) fixed.Int26_6 { return fixed.Int26_6(math.Floor(v * 64)) }

// blitGlyph sets the pixels of bounds the glyph mask covers at least half to the current color; maskOrigin is where the glyph's top-left pixel sits in mask.
func (c *PalettedCanvas) blitGlyph(bounds image.Rectangle, mask image.Image, maskOrigin image.Point) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := mask.At(maskOrigin.X+x-bounds.Min.X, maskOrigin.Y+y-bounds.Min.Y).RGBA()
			if alpha >= glyphOpaque && image.Pt(x, y).In(c.img.Rect) {
				c.setPixel(x, y, c.current)
			}
		}
	}
}

// MeasureString returns the (w, h) DrawString would occupy.
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

func copyRow(dst *image.Paletted, dx, dy int, src *image.Paletted, sx, sy, n int) {
	copy(dst.Pix[dst.PixOffset(dx, dy):][:n], src.Pix[src.PixOffset(sx, sy):][:n])
}

// setPixel requires local (x, y) in bounds.
func (c *PalettedCanvas) setPixel(x, y int, index uint8) { c.img.Pix[c.img.PixOffset(x, y)] = index }

// IndexAt returns the palette index of the pixel at (x, y) in this canvas's own pixels, which must be in bounds.
func (c *PalettedCanvas) IndexAt(x, y int) uint8 { return c.img.Pix[c.img.PixOffset(x, y)] }

// PaintPixel takes drawing coordinates and ignores pixels this canvas doesn't cover.
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
