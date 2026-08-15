package graphics

import (
	"image/color"
	"math"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

func TestInterpolateColor(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	c1 := color.RGBA{0, 0, 0, 255}
	c2 := color.RGBA{100, 200, 50, 255}

	got := mr.InterpolateColor(c1, c2, 0.5)
	want := color.RGBA{50, 100, 25, 255}
	if got != want {
		t.Errorf("InterpolateColor(midpoint) = %v, want %v", got, want)
	}

	gotStart := mr.InterpolateColor(c1, c2, 0.0)
	if gotStart != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("InterpolateColor(t=0) = %v, want c1", gotStart)
	}
}

func TestGetNewCityColor(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	got := mr.GetNewCityColor(color.RGBA{0, 0, 0, 255})
	want := mr.InterpolateColor(color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}, 0.2)
	if got != want {
		t.Errorf("GetNewCityColor() = %v, want %v", got, want)
	}
}

func TestDrawMountain(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(100, 100)

	mr.DrawMountain(canvas, 10, 20)

	ops := canvas.GetOperations()
	// Expect base triangle + color + fill, then peak triangle + color + fill = 6 ops
	if len(ops) != 6 {
		t.Fatalf("DrawMountain() recorded %d ops, want 6: %v", len(ops), ops)
	}
	if ops[0] != "DrawRegularPolygon(3, 10.00, 20.00, 16.00, 3.14)" {
		t.Errorf("DrawMountain() base polygon op = %q", ops[0])
	}
	if ops[3] != "DrawRegularPolygon(3, 10.00, 28.00, 8.00, 3.14)" {
		t.Errorf("DrawMountain() peak polygon op = %q", ops[3])
	}
}

func TestDrawCityIcon(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(100, 100)

	mr.DrawCityIcon(canvas, 10, 20, color.RGBA{0, 0, 0, 255})

	ops := canvas.GetOperations()
	if len(ops) != 3 {
		t.Fatalf("DrawCityIcon() recorded %d ops, want 3: %v", len(ops), ops)
	}
	if ops[len(ops)-1] != "Fill()" {
		t.Errorf("DrawCityIcon() last op = %q, want Fill()", ops[len(ops)-1])
	}
}

func TestGetHexEdge(t *testing.T) {
	line := getHexEdge(0, 0, 0, 10)
	angle1 := math.Pi / 6
	wantX1 := 10 * math.Cos(angle1)
	wantY1 := 10 * math.Sin(angle1)
	if math.Abs(line.X1-wantX1) > 1e-9 || math.Abs(line.Y1-wantY1) > 1e-9 {
		t.Errorf("getHexEdge(0) start = (%v, %v), want (%v, %v)", line.X1, line.Y1, wantX1, wantY1)
	}
}

func TestDrawTerrainTilesDrawsExpectedShapes(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"},
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{
				{TerrainType: 0, Elevation: 0},
				{TerrainType: 1, Elevation: 2}, // mountain
			},
		},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{},
	}

	mr.DrawTerrainTiles(canvas, mapData, 1, 2)

	ops := canvas.GetOperations()
	// Tile 1 (no mountain): DrawRegularPolygon + SetColor + Fill = 3 ops
	// Tile 2 (mountain): DrawRegularPolygon + SetColor + Fill + DrawMountain(6 ops) = 9 ops
	wantOpsCount := 3 + 9
	if len(ops) != wantOpsCount {
		t.Fatalf("DrawTerrainTiles() recorded %d ops, want %d: %v", len(ops), wantOpsCount, ops)
	}
}
