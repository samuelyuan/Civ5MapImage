package raster

import "image/color"

// colorTable maps RGB colors to palette indices; sibling canvases share it.
type colorTable struct {
	palette color.Palette
	indexOf map[uint32]uint8
	grow    bool // add off-palette colors to the palette, up to 256, instead of snapping
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

// IndexFor returns the palette index of (r, g, b): added if the table grows and has room, else snapped to the nearest color.
func (t *colorTable) IndexFor(r, g, b uint8) uint8 {
	key := rgbKey(r, g, b)
	if i, ok := t.indexOf[key]; ok {
		return i
	}
	if t.grow && len(t.palette) < 256 {
		t.palette = append(t.palette, color.RGBA{r, g, b, 255})
		i := uint8(len(t.palette) - 1)
		t.indexOf[key] = i
		return i
	}
	i := uint8(t.palette.Index(color.RGBA{r, g, b, 255}))
	t.indexOf[key] = i
	return i
}
