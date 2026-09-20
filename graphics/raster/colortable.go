package raster

import "image/color"

// colorTable maps RGB colors to indices in a palette. Scratch canvases share their parent's table, so a color that
// isn't in the palette is counted once no matter which canvas drew it.
type colorTable struct {
	palette color.Palette
	indexOf map[uint32]uint8 // rgbKey -> palette index
	inexact int              // distinct off-palette colors snapped to the nearest palette color
}

func newColorTable(palette color.Palette) *colorTable {
	t := &colorTable{palette: palette, indexOf: make(map[uint32]uint8, len(palette))}
	for i := len(palette) - 1; i >= 0; i-- { // backwards, so a repeated color keeps its first index
		r, g, b, _ := palette[i].RGBA()
		t.indexOf[rgbKey(uint8(r>>8), uint8(g>>8), uint8(b>>8))] = uint8(i)
	}
	return t
}

func rgbKey(r, g, b uint8) uint32 { return uint32(r)<<16 | uint32(g)<<8 | uint32(b) }

// IndexFor returns the palette index of (r, g, b), or of the nearest color if it's not in the palette.
func (t *colorTable) IndexFor(r, g, b uint8) uint8 {
	key := rgbKey(r, g, b)
	if i, ok := t.indexOf[key]; ok {
		return i
	}
	i := uint8(t.palette.Index(color.RGBA{r, g, b, 255}))
	t.indexOf[key] = i
	t.inexact++
	return i
}

// Inexact returns how many distinct colors IndexFor had to snap to the nearest palette color. It is 0 when the palette
// holds every color that was drawn.
func (t *colorTable) Inexact() int { return t.inexact }
