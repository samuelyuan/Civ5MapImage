package graphics

import (
	"image/color"
	"testing"
)

func paletteContains(palette color.Palette, c color.RGBA) bool {
	for _, p := range palette {
		r, g, b, _ := p.RGBA()
		if uint8(r>>8) == c.R && uint8(g>>8) == c.G && uint8(b>>8) == c.B {
			return true
		}
	}
	return false
}

// TestReplayPaletteCoversEveryDrawnColor guards against drawn colors missing from the palette.
func TestReplayPaletteCoversEveryDrawnColor(t *testing.T) {
	mapData := buildBenchMapData(24, 24, 4, 1)
	mapData.Civ5PlayerData[1].CivType = "CIVILIZATION_MINOR_TEST"

	canvas := newBenchCanvas(mapData)
	renderer := NewMapRenderer(DefaultDrawingConfig())
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	renderer.DrawMountain(canvas, mapLayout(renderer.config.Radius), 0, 0)

	if canvas.Inexact() != 0 {
		t.Errorf("the renderer drew %d colors that aren't in replayPalette", canvas.Inexact())
	}
}

func TestReplayPaletteHasNoDuplicatesFitsAGIFAndKeepsBackground(t *testing.T) {
	mapData := buildBenchMapData(8, 8, 6, 1)
	palette := replayPalette(mapData)
	if len(palette) == 0 || len(palette) > 256 {
		t.Fatalf("palette has %d entries, want 1..256", len(palette))
	}
	seen := map[color.Color]bool{}
	for _, c := range palette {
		if seen[c] {
			t.Errorf("palette has duplicate entry %v", c)
		}
		seen[c] = true
	}
	if !paletteContains(palette, color.RGBA{0, 0, 0, 255}) {
		t.Error("palette is missing the black canvas background")
	}
}

func TestGetPhysicalMapTileColor(t *testing.T) {
	tests := []struct {
		terrain string
		want    color.RGBA
	}{
		{"TERRAIN_GRASS", color.RGBA{105, 125, 54, 255}},
		{"TERRAIN_OCEAN", color.RGBA{47, 74, 93, 255}},
		{"TERRAIN_UNKNOWN", color.RGBA{0, 0, 0, 255}},
		{"", color.RGBA{0, 0, 0, 255}},
	}
	for _, tt := range tests {
		if got := GetPhysicalMapTileColor(tt.terrain); got != tt.want {
			t.Errorf("GetPhysicalMapTileColor(%q) = %v, want %v", tt.terrain, got, tt.want)
		}
	}
}
