package graphics

import (
	"image/color"
	"math"
	"testing"
)

func TestMarkerColorLightensTowardWhite(t *testing.T) {
	got := markerColor(color.RGBA{0, 0, 0, 255})
	want := blendColor(color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}, 0.2)
	if got != want {
		t.Errorf("markerColor() = %v, want %v", got, want)
	}
}

func TestBlendColor(t *testing.T) {
	c1 := color.RGBA{0, 0, 0, 255}
	c2 := color.RGBA{100, 200, 50, 255}

	got := blendColor(c1, c2, 0.5)
	want := color.RGBA{50, 100, 25, 255}
	if got != want {
		t.Errorf("blendColor(midpoint) = %v, want %v", got, want)
	}

	gotStart := blendColor(c1, c2, 0.0)
	if gotStart != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("blendColor(t=0) = %v, want c1", gotStart)
	}
}

func TestContrastRatio(t *testing.T) {
	black, white := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}
	if got := contrastRatio(black, white); math.Abs(got-21) > 0.01 {
		t.Errorf("contrastRatio(black, white) = %.2f, want 21", got)
	}
	if got := contrastRatio(white, black); math.Abs(got-21) > 0.01 {
		t.Errorf("contrastRatio(white, black) = %.2f, want the same 21 in either order", got)
	}
	if got := contrastRatio(white, white); math.Abs(got-1) > 1e-9 {
		t.Errorf("contrastRatio(white, white) = %.2f, want 1", got)
	}
	// A published WCAG value: #767676 on white is 4.54 to 1, the darkest grey that passes the 4.5 minimum.
	if got := contrastRatio(color.RGBA{118, 118, 118, 255}, white); math.Abs(got-4.54) > 0.01 {
		t.Errorf("contrastRatio(#767676, white) = %.3f, want 4.54", got)
	}
}
