// Package quantize reduces an opaque RGBA image to a small palette and maps pixels onto it.
// Alpha is ignored throughout.
package quantize

import (
	"image"
	"image/color"
	"sort"
)

// rgb is a color with 16 bits per channel.
type rgb [3]int

// colorCount is a distinct color and the number of pixels that have it.
type colorCount struct {
	color rgb
	count int
}

// colorBox is a set of distinct colors plus the bounding box of their values.
type colorBox struct {
	colors   []colorCount
	min, max rgb
}

func newColorBox(colors []colorCount) colorBox {
	box := colorBox{colors: colors, min: colors[0].color, max: colors[0].color}
	for _, cc := range colors[1:] {
		for ch, v := range cc.color {
			if v < box.min[ch] {
				box.min[ch] = v
			}
			if v > box.max[ch] {
				box.max[ch] = v
			}
		}
	}
	return box
}

// widestChannel returns the channel with the largest value range, and that range.
func (b colorBox) widestChannel() (channel, span int) {
	for ch := range b.min {
		if s := b.max[ch] - b.min[ch]; ch == 0 || s > span {
			channel, span = ch, s
		}
	}
	return channel, span
}

// lessOnChannel orders colors by channel first, then the remaining channels, giving a total order.
func lessOnChannel(a, b rgb, channel int) bool {
	for i := range a {
		ch := (channel + i) % len(a)
		if a[ch] != b[ch] {
			return a[ch] < b[ch]
		}
	}
	return false
}

// split sorts the box along its widest channel and cuts it where the cumulative pixel count reaches
// half, so heavily used colors (not merely distinct ones) drive the cut.
func (b colorBox) split() (colorBox, colorBox) {
	channel, _ := b.widestChannel()
	colors := b.colors
	sort.Slice(colors, func(i, j int) bool { return lessOnChannel(colors[i].color, colors[j].color, channel) })

	total := 0
	for _, cc := range colors {
		total += cc.count
	}
	cut, seen := 1, 0
	for i, cc := range colors {
		seen += cc.count
		if seen*2 >= total {
			cut = i + 1
			break
		}
	}
	if cut >= len(colors) {
		cut = len(colors) - 1
	}
	return newColorBox(colors[:cut]), newColorBox(colors[cut:])
}

// average returns the pixel-weighted mean color of the box.
func (b colorBox) average() color.Color {
	var sum rgb
	pixels := 0
	for _, cc := range b.colors {
		for ch, v := range cc.color {
			sum[ch] += v * cc.count
		}
		pixels += cc.count
	}
	return opaque(rgb{sum[0] / pixels, sum[1] / pixels, sum[2] / pixels})
}

func opaque(c rgb) color.RGBA64 {
	return color.RGBA64{R: uint16(c[0]), G: uint16(c[1]), B: uint16(c[2]), A: 0xFFFF}
}

// BuildPalette returns at most maxColors colors representing src's pixels in r (clipped to src's
// bounds). If src has no more distinct colors than that, they are used exactly; otherwise median
// cut weights each color by its pixel count, so a color covering most of the image isn't spread
// over many entries. The result is deterministic.
func BuildPalette(src *image.RGBA, r image.Rectangle, maxColors int) color.Palette {
	r = r.Intersect(src.Bounds())
	if r.Empty() || maxColors <= 0 {
		return color.Palette{}
	}
	colors := countColors(src, r)
	if len(colors) <= maxColors {
		palette := make(color.Palette, len(colors))
		for i, cc := range colors {
			palette[i] = opaque(cc.color)
		}
		return palette
	}

	boxes := []colorBox{newColorBox(colors)}
	for len(boxes) < maxColors {
		widest, widestSpan := -1, -1
		for i, box := range boxes {
			if len(box.colors) < 2 {
				continue
			}
			if _, span := box.widestChannel(); span > widestSpan {
				widest, widestSpan = i, span
			}
		}
		if widest < 0 {
			break
		}
		lo, hi := boxes[widest].split()
		boxes[widest] = lo
		boxes = append(boxes, hi)
	}

	palette := make(color.Palette, len(boxes))
	for i, box := range boxes {
		palette[i] = box.average()
	}
	return palette
}

// countColors counts pixels per distinct color in r, sorted for determinism.
func countColors(src *image.RGBA, r image.Rectangle) []colorCount {
	counts := make(map[uint32]int)
	// Neighboring pixels usually match, so tally runs before touching the map.
	var runKey uint32
	run := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			key := colorKey(src.RGBAAt(x, y))
			if run > 0 && key == runKey {
				run++
				continue
			}
			if run > 0 {
				counts[runKey] += run
			}
			runKey, run = key, 1
		}
	}
	if run > 0 {
		counts[runKey] += run
	}

	colors := make([]colorCount, 0, len(counts))
	for key, n := range counts {
		// An 8-bit channel v widens to 16 bits as v*0x101.
		colors = append(colors, colorCount{
			color: rgb{int(key>>16&0xFF) * 0x101, int(key>>8&0xFF) * 0x101, int(key&0xFF) * 0x101},
			count: n,
		})
	}
	sort.Slice(colors, func(i, j int) bool { return lessOnChannel(colors[i].color, colors[j].color, 0) })
	return colors
}

// colorKey packs a pixel's RGB into 24 bits.
func colorKey(p color.RGBA) uint32 {
	return uint32(p.R)<<16 | uint32(p.G)<<8 | uint32(p.B)
}

// PaletteMapper resolves pixels to indices of a fixed palette, remembering each color's index
// across Fill calls. Not safe for concurrent use.
type PaletteMapper struct {
	palette color.Palette
	cache   map[uint32]uint8
}

func NewPaletteMapper(palette color.Palette) *PaletteMapper {
	return &PaletteMapper{palette: palette, cache: make(map[uint32]uint8)}
}

func (m *PaletteMapper) Palette() color.Palette { return m.palette }

// Fill sets dst.Palette and copies src's pixels in r into dst starting at sp, as palette indices.
// r must lie within src's bounds.
func (m *PaletteMapper) Fill(dst *image.Paletted, r image.Rectangle, src *image.RGBA, sp image.Point) {
	dst.Palette = m.palette
	// Neighboring pixels are usually the same color, so remember the last lookup. Keys are 24-bit,
	// so all-ones never matches a real color.
	lastKey, lastIndex := ^uint32(0), uint8(0)
	for dy := 0; dy < r.Dy(); dy++ {
		for dx := 0; dx < r.Dx(); dx++ {
			p := src.RGBAAt(r.Min.X+dx, r.Min.Y+dy)
			key := colorKey(p)
			if key != lastKey {
				index, ok := m.cache[key]
				if !ok {
					index = uint8(m.palette.Index(p))
					m.cache[key] = index
				}
				lastKey, lastIndex = key, index
			}
			dst.SetColorIndex(sp.X+dx, sp.Y+dy, lastIndex)
		}
	}
}
