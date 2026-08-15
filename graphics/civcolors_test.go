package graphics

import (
	"image/color"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

func TestConvertFractionToRGBA(t *testing.T) {
	got := convertFractionToRGBA(1.0, 0.5, 0.0)
	want := color.RGBA{255, 128, 0, 255}
	if got != want {
		t.Errorf("convertFractionToRGBA(1.0, 0.5, 0.0) = %v, want %v", got, want)
	}
}

func TestConvertCivColorInfoToRGBAConstant(t *testing.T) {
	info := fileio.CivColorInfo{Model: "constant", ColorConstant: "COLOR_WHITE"}
	got := convertCivColorInfoToRGBA(info)
	want := colorMap["COLOR_WHITE"]
	if got != want {
		t.Errorf("convertCivColorInfoToRGBA(constant) = %v, want %v", got, want)
	}
}

func TestConvertCivColorInfoToRGBAFraction(t *testing.T) {
	info := fileio.CivColorInfo{Model: "fraction", Red: 1.0, Green: 0.0, Blue: 0.0}
	got := convertCivColorInfoToRGBA(info)
	want := color.RGBA{255, 0, 0, 255}
	if got != want {
		t.Errorf("convertCivColorInfoToRGBA(fraction) = %v, want %v", got, want)
	}
}

func TestConvertCivColorInfoToRGBARgb255(t *testing.T) {
	info := fileio.CivColorInfo{Model: "rgb255", Red: 10, Green: 20, Blue: 30}
	got := convertCivColorInfoToRGBA(info)
	want := color.RGBA{10, 20, 30, 255}
	if got != want {
		t.Errorf("convertCivColorInfoToRGBA(rgb255) = %v, want %v", got, want)
	}
}

func TestConvertCivColorInfoToRGBAUnknownModel(t *testing.T) {
	info := fileio.CivColorInfo{Model: "bogus"}
	got := convertCivColorInfoToRGBA(info)
	want := color.RGBA{0, 0, 0, 255}
	if got != want {
		t.Errorf("convertCivColorInfoToRGBA(unknown) = %v, want %v", got, want)
	}
}

func TestOverrideColorMap(t *testing.T) {
	const key = "PLAYERCOLOR_TEST_OVERRIDE"
	overrides := []fileio.CivColorOverride{
		{
			CivKey:     key,
			OuterColor: fileio.CivColorInfo{Model: "rgb255", Red: 1, Green: 2, Blue: 3},
			InnerColor: fileio.CivColorInfo{Model: "rgb255", Red: 4, Green: 5, Blue: 6},
		},
	}

	OverrideColorMap(overrides)
	defer delete(civColorMap, key)

	got, ok := civColorMap[key]
	if !ok {
		t.Fatalf("OverrideColorMap() did not add key %q", key)
	}
	wantOuter := color.RGBA{1, 2, 3, 255}
	wantInner := color.RGBA{4, 5, 6, 255}
	if got.OuterColor != wantOuter {
		t.Errorf("OverrideColorMap() OuterColor = %v, want %v", got.OuterColor, wantOuter)
	}
	if got.InnerColor != wantInner {
		t.Errorf("OverrideColorMap() InnerColor = %v, want %v", got.InnerColor, wantInner)
	}
	if got.TextColor != wantInner {
		t.Errorf("OverrideColorMap() TextColor = %v, want %v (mirrors inner)", got.TextColor, wantInner)
	}
}

func TestKnownColorMapEntriesExist(t *testing.T) {
	// A handful of well-known keys that the rest of the package relies on.
	keys := []string{"COLOR_WHITE", "COLOR_BLACK", "COLOR_CLEAR"}
	for _, key := range keys {
		if _, ok := colorMap[key]; !ok {
			t.Errorf("colorMap missing expected key %q", key)
		}
	}
}

func TestKnownCivColorMapEntriesExist(t *testing.T) {
	if _, ok := civColorMap["PLAYERCOLOR_BLACK"]; !ok {
		t.Errorf("civColorMap missing expected key %q", "PLAYERCOLOR_BLACK")
	}
}
