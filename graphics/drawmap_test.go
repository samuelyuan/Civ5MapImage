package graphics

import (
	"fmt"
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

func TestDrawRiversDrawsEachEdgePresent(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	// bit0 = east, bit1 = southeast, bit2 = southwest
	riverData := uint8(1<<2 | 1<<1 | 1)
	mapData := &fileio.Civ5MapData{
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{RiverData: int(riverData)}},
		},
	}

	mr.DrawRivers(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	// 1 SetColor + 3 edges * (DrawLine + Stroke) = 7 ops
	if len(ops) != 7 {
		t.Fatalf("DrawRivers() recorded %d ops, want 7: %v", len(ops), ops)
	}
}

func TestDrawRiversNoRiverData(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{RiverData: 0}},
		},
	}

	mr.DrawRivers(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	// Only the unconditional SetColor call.
	if len(ops) != 1 {
		t.Fatalf("DrawRivers() recorded %d ops, want 1: %v", len(ops), ops)
	}
}

func newRoadTestMapData(routeType1, routeType2 int) *fileio.Civ5MapData {
	return &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, RouteType: routeType1, CityId: -1},
				{X: 1, Y: 0, RouteType: routeType2, CityId: -1},
			},
		},
	}
}

func TestDrawRoadsConnectsAdjacentTiles(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	// A 1x2 map: tile (0,0) and (1,0) are hex-neighbors of each other, both roads.
	mapData := newRoadTestMapData(0, 0)

	mr.DrawRoads(canvas, mapData, 1, 2)

	ops := canvas.GetOperations()
	// Each tile draws one line to the other: SetLineWidth + SetColor + DrawLine + Stroke = 4 ops each.
	if len(ops) != 8 {
		t.Fatalf("DrawRoads() recorded %d ops, want 8: %v", len(ops), ops)
	}
}

func TestDrawRoadsSkipsTilesWithNoRoute(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	// RouteType 255 means "no route" on both tiles, and neither has a city name.
	mapData := newRoadTestMapData(255, 255)

	mr.DrawRoads(canvas, mapData, 1, 2)

	ops := canvas.GetOperations()
	if len(ops) != 0 {
		t.Fatalf("DrawRoads() recorded %d ops, want 0: %v", len(ops), ops)
	}
}

func TestDrawRoadsNoImprovementsIsNoOp(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{}}

	mr.DrawRoads(canvas, mapData, 1, 2)

	if ops := canvas.GetOperations(); len(ops) != 0 {
		t.Fatalf("DrawRoads() with no improvements recorded %d ops, want 0: %v", len(ops), ops)
	}
}

func newBorderTestMapData(owner1, owner2 int, teamColor1, teamColor2 string) *fileio.Civ5MapData {
	return &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, Owner: owner1, CityId: -1},
				{X: 1, Y: 0, Owner: owner2, CityId: -1},
			},
		},
		Civ5PlayerData: []*fileio.Civ5PlayerData{
			{Index: 0, CivType: "CIVILIZATION_ROME", TeamColor: teamColor1},
			{Index: 1, CivType: "CIVILIZATION_GREECE", TeamColor: teamColor2},
		},
		CityOwnerIndexMap: map[int]int{0: 0, 1: 1},
	}
}

func TestDrawBordersDrawsLineBetweenDifferentOwners(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newBorderTestMapData(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")

	mr.DrawBorders(canvas, mapData, 1, 2)

	ops := canvas.GetOperations()
	// Each tile draws one border edge to the other: SetColor + SetLineWidth + DrawLine + Stroke = 4 ops each,
	// plus a trailing SetLineWidth(1.0) reset at the end of the function.
	if len(ops) != 9 {
		t.Fatalf("DrawBorders() recorded %d ops, want 9: %v", len(ops), ops)
	}
	if last := ops[len(ops)-1]; last != "SetLineWidth(1.00)" {
		t.Errorf("DrawBorders() last op = %q, want SetLineWidth(1.00)", last)
	}
}

func TestDrawBordersSameOwnerDrawsNothing(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newBorderTestMapData(0, 0, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK")

	mr.DrawBorders(canvas, mapData, 1, 2)

	ops := canvas.GetOperations()
	// No border edges drawn, but the trailing SetLineWidth reset still happens.
	if len(ops) != 1 || ops[0] != "SetLineWidth(1.00)" {
		t.Fatalf("DrawBorders() with same owner recorded %v, want just [SetLineWidth(1.00)]", ops)
	}
}

func TestDrawBordersInvalidOwnerSkipsTile(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newBorderTestMapData(-1, -1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK")

	mr.DrawBorders(canvas, mapData, 1, 2)

	ops := canvas.GetOperations()
	if len(ops) != 1 || ops[0] != "SetLineWidth(1.00)" {
		t.Fatalf("DrawBorders() with invalid owners recorded %v, want just [SetLineWidth(1.00)]", ops)
	}
}

func TestDrawPhysicalCityNames(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{CityName: "Rome"}},
		},
	}

	mr.DrawPhysicalCityNames(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	if len(ops) != 2 {
		t.Fatalf("DrawPhysicalCityNames() recorded %d ops, want 2: %v", len(ops), ops)
	}
	if ops[0] != "SetColor(255, 255, 255)" {
		t.Errorf("DrawPhysicalCityNames() color op = %q, want white", ops[0])
	}
	if want := `DrawString("Rome"`; len(ops[1]) < len(want) || ops[1][:len(want)] != want {
		t.Errorf("DrawPhysicalCityNames() draw op = %q, want prefix %q", ops[1], want)
	}
}

func TestDrawPhysicalCityNamesNoImprovementsIsNoOp(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{}}

	mr.DrawPhysicalCityNames(canvas, mapData, 1, 1)

	if ops := canvas.GetOperations(); len(ops) != 0 {
		t.Fatalf("DrawPhysicalCityNames() with no improvements recorded %d ops, want 0: %v", len(ops), ops)
	}
}

func TestDrawPoliticalCityNamesKnownColor(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{CityName: "Rome", Owner: 0, CityId: -1}},
		},
		Civ5PlayerData:    []*fileio.Civ5PlayerData{{Index: 0, CivType: "CIVILIZATION_ROME", TeamColor: "PLAYERCOLOR_BLACK"}},
		CityOwnerIndexMap: map[int]int{0: 0},
	}

	mr.DrawPoliticalCityNames(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	if len(ops) != 2 {
		t.Fatalf("DrawPoliticalCityNames() recorded %d ops, want 2: %v", len(ops), ops)
	}
	if ops[0] == "SetColor(255, 255, 255)" {
		t.Errorf("DrawPoliticalCityNames() should use the civ color, not the white fallback: %q", ops[0])
	}
}

func TestDrawPoliticalCityNamesUnknownColorFallsBackToWhite(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{CityName: "Rome", Owner: -1, CityId: -1}},
		},
	}

	mr.DrawPoliticalCityNames(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	if len(ops) != 2 || ops[0] != "SetColor(255, 255, 255)" {
		t.Fatalf("DrawPoliticalCityNames() with unowned tile = %v, want white fallback color first", ops)
	}
}

func newTerritoryTestMapData(terrainType, owner int, teamColor, civType string) *fileio.Civ5MapData {
	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"},
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{TerrainType: terrainType, Elevation: 0}},
		},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{Owner: owner, CityId: -1}},
		},
	}
	if civType != "" {
		mapData.Civ5PlayerData = []*fileio.Civ5PlayerData{{Index: 0, CivType: civType, TeamColor: teamColor}}
		mapData.CityOwnerIndexMap = map[int]int{owner: 0}
	}
	return mapData
}

func TestDrawTerritoryTilesWaterTileUsesTerrainColor(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newTerritoryTestMapData(1 /* TERRAIN_OCEAN */, -1, "", "")

	mr.DrawTerritoryTiles(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	// DrawRegularPolygon + SetColor + Fill, no mountain, no city.
	if len(ops) != 3 {
		t.Fatalf("DrawTerritoryTiles() water tile recorded %d ops, want 3: %v", len(ops), ops)
	}
	oceanColor := fileio.GetPhysicalMapTileColor("TERRAIN_OCEAN")
	wantColorOp := fmt.Sprintf("SetColor(%d, %d, %d)", oceanColor.R, oceanColor.G, oceanColor.B)
	if ops[1] != wantColorOp {
		t.Errorf("DrawTerritoryTiles() water color op = %q, want %q", ops[1], wantColorOp)
	}
}

func TestDrawTerritoryTilesUnownedLandUsesTerrainColor(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newTerritoryTestMapData(0 /* TERRAIN_GRASS */, -1, "", "")

	mr.DrawTerritoryTiles(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	if len(ops) != 3 {
		t.Fatalf("DrawTerritoryTiles() unowned land recorded %d ops, want 3: %v", len(ops), ops)
	}
}

func TestDrawTerritoryTilesOwnedMajorCivDrawsCityIcon(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_BLACK", "CIVILIZATION_ROME")
	mapData.MapTileImprovements[0][0].CityId = 0 // has a city

	mr.DrawTerritoryTiles(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	// Tile: DrawRegularPolygon + SetColor + Fill (3), plus city icon: DrawRectangle + SetColor + Fill (3).
	if len(ops) != 6 {
		t.Fatalf("DrawTerritoryTiles() owned major civ with city recorded %d ops, want 6: %v", len(ops), ops)
	}
}

func TestDrawTerritoryTilesOwnedUnknownColorFallsBackToBlack(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	// Owned by a civ whose team color isn't a recognized key in civColorMap.
	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_DOES_NOT_EXIST", "CIVILIZATION_ROME")

	mr.DrawTerritoryTiles(canvas, mapData, 1, 1)

	ops := canvas.GetOperations()
	if len(ops) != 3 {
		t.Fatalf("DrawTerritoryTiles() unknown owner color recorded %d ops, want 3: %v", len(ops), ops)
	}
	if ops[1] != "SetColor(0, 0, 0)" {
		t.Errorf("DrawTerritoryTiles() unknown owner color op = %q, want SetColor(0, 0, 0)", ops[1])
	}
}

func newFullMapDataForRender() *fileio.Civ5MapData {
	return &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"},
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{TerrainType: 0, Elevation: 0}},
		},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{Owner: -1, CityId: -1, RouteType: 255}},
		},
	}
}

func TestDrawPhysicalMapResizesAndInvertsCanvas(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(1, 1)
	mapData := newFullMapDataForRender()

	img := mr.DrawPhysicalMap(canvas, mapData)

	if img == nil {
		t.Fatal("DrawPhysicalMap() returned a nil image")
	}
	ops := canvas.GetOperations()
	if ops[0][:6] != "Resize" {
		t.Errorf("DrawPhysicalMap() first op = %q, want a Resize call", ops[0])
	}
	invertCount := 0
	for _, op := range ops {
		if op == "InvertY()" {
			invertCount++
		}
	}
	if invertCount != 2 {
		t.Errorf("DrawPhysicalMap() called InvertY() %d times, want 2 (once each direction)", invertCount)
	}
}

func TestDrawPoliticalMapResizesAndInvertsCanvas(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(1, 1)
	mapData := newFullMapDataForRender()

	img := mr.DrawPoliticalMap(canvas, mapData)

	if img == nil {
		t.Fatal("DrawPoliticalMap() returned a nil image")
	}
	ops := canvas.GetOperations()
	if ops[0][:6] != "Resize" {
		t.Errorf("DrawPoliticalMap() first op = %q, want a Resize call", ops[0])
	}
}

func TestSaveImage(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(1, 1)

	if err := mr.SaveImage(canvas, "output.png"); err != nil {
		t.Fatalf("SaveImage() returned error: %v", err)
	}

	ops := canvas.GetOperations()
	if len(ops) != 1 || ops[0] != `SavePNG("output.png")` {
		t.Errorf("SaveImage() ops = %v, want [SavePNG(\"output.png\")]", ops)
	}
}
