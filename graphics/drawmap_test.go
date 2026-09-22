package graphics

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

func TestGetHexEdge(t *testing.T) {
	line := getHexEdge(0, 0, 0, 10)
	angle1 := math.Pi / 6
	wantX1 := 10 * math.Cos(angle1)
	wantY1 := -10 * math.Sin(angle1) // y points down
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

	mr.DrawTerrainTiles(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	ops := canvas.GetOperations()
	// Each tile: fill (DrawRegularPolygon + SetColor + Fill = 3 ops) and outline (SetColor + SetLineWidth + 3 DrawLine + Stroke = 6 ops).
	// Mountains are drawn by DrawMountains, not with the tiles.
	wantOpsCount := 2 * (3 + 6)
	if len(ops) != wantOpsCount {
		t.Fatalf("DrawTerrainTiles() recorded %d ops, want %d: %v", len(ops), wantOpsCount, ops)
	}
}

// All fills come first, then the outlines, so a fill never covers an outline.
func TestDrawTerrainTilesDrawsFillsThenOutlines(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)
	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"},
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{TerrainType: 1, Elevation: 2}, {TerrainType: 0, Elevation: 0}}, // a mountain, then a plain
		},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{},
	}

	mr.DrawTerrainTiles(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	var kinds []string // "fill" or "outline", one per Fill or Stroke
	for _, op := range canvas.GetOperations() {
		switch op {
		case "Stroke()":
			kinds = append(kinds, "outline")
		case "Fill()":
			kinds = append(kinds, "fill")
		}
	}
	want := []string{"fill", "fill", "outline", "outline"}
	if !slices.Equal(kinds, want) {
		t.Errorf("layers = %v, want %v", kinds, want)
	}
}

// Each mountain tile's peak is drawn, five triangles of three ops each, and nothing else.
func TestDrawMountainsDrawsAPeakOnEveryMountainTile(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mr.DrawMountains(canvas, terrainWithMountains([][]bool{{true, false, false, true}}), fileio.MapSize{Height: 1, Width: 4})

	ops := canvas.GetOperations()
	if len(ops) != 2*5*3 {
		t.Fatalf("DrawMountains() recorded %d ops, want 30: %v", len(ops), ops)
	}
	for i := 0; i < len(ops); i += 3 {
		if !strings.HasPrefix(ops[i], "DrawTriangle(") || !strings.HasPrefix(ops[i+1], "SetColor(") || ops[i+2] != "Fill()" {
			t.Fatalf("ops %d..%d = %v, want a filled triangle", i, i+2, ops[i:i+3])
		}
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

	mr.DrawRivers(canvas, mapData, fileio.MapSize{Height: 1, Width: 1})

	ops := canvas.GetOperations()
	// 1 SetColor + 1 SetLineWidth + 3 edges * (DrawLine + Stroke) = 8 ops
	if len(ops) != 8 {
		t.Fatalf("DrawRivers() recorded %d ops, want 8: %v", len(ops), ops)
	}
}

// Rivers on maps are bluer and half again as wide as a tile outline; the replay's stay the thin teal ones.
func TestDrawRiversUsesTheMapRiverColorAndWidth(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	mapData := &fileio.Civ5MapData{MapTiles: [][]*fileio.Civ5MapTilePhysical{{{RiverData: 1}}}}
	size := fileio.MapSize{Height: 1, Width: 1}

	canvas := NewMockCanvas(200, 200)
	mr.DrawRivers(canvas, mapData, size)
	if got, want := canvas.GetOperations()[:2], []string{"SetColor(84, 148, 210)", "SetLineWidth(1.50)"}; !slices.Equal(got, want) {
		t.Errorf("map river style = %v, want %v", got, want)
	}

	replay := NewMockCanvas(200, 200)
	mr.drawRiverTile(replay, layoutForMap(size, mr.config.Radius), mapData, fileio.TilePos{}, riverColor, 1.0)
	if got, want := replay.GetOperations()[:2], []string{"SetColor(95, 150, 148)", "SetLineWidth(1.00)"}; !slices.Equal(got, want) {
		t.Errorf("replay river style = %v, want %v", got, want)
	}
}

// A river must show on the lightest and the darkest ground, unlike the teal it replaced, which nearly vanished on many maps.
func TestMapRiverColorContrastsWithBlackAndWhite(t *testing.T) {
	for name, ground := range map[string]color.RGBA{"white": {255, 255, 255, 255}, "black": {0, 0, 0, 255}} {
		if got := contrastRatio(mapRiverColor, ground); got < 3 {
			t.Errorf("contrast of the river color with %s = %.2f, want at least 3", name, got)
		}
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

	mr.DrawRivers(canvas, mapData, fileio.MapSize{Height: 1, Width: 1})

	ops := canvas.GetOperations()
	// Only the unconditional SetColor + SetLineWidth calls.
	if len(ops) != 2 {
		t.Fatalf("DrawRivers() recorded %d ops, want 2: %v", len(ops), ops)
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

	mr.DrawRoads(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	ops := canvas.GetOperations()
	// Each tile draws one line to the other, and a casing under it: SetLineWidth + SetColor + DrawLine + Stroke = 4 ops each.
	if len(ops) != 16 {
		t.Fatalf("DrawRoads() recorded %d ops, want 16: %v", len(ops), ops)
	}
}

// Every casing goes down before any line, so a casing never covers another route's line, and a railroad (wider) is drawn over a road.
func TestDrawRoadsDrawsAllCasingsThenLinesWithWiderLinesLast(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)
	mapData := newRoadTestMapData(1, 0) // tile (0,0) is a railroad, tile (1,0) a road

	mr.DrawRoads(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	var strokes []string // each stroke's width and color, in drawing order
	for _, op := range canvas.GetOperations() {
		if strings.HasPrefix(op, "SetLineWidth") || strings.HasPrefix(op, "SetColor") {
			strokes = append(strokes, op)
		}
	}
	want := []string{
		"SetLineWidth(4.00)", "SetColor(190, 155, 100)", // casings: the railroad's warm one,
		"SetLineWidth(3.00)", "SetColor(150, 150, 150)", // then the road's grey one, in tile order
		"SetLineWidth(1.00)", "SetColor(51, 51, 51)", // lines: the road,
		"SetLineWidth(2.00)", "SetColor(76, 51, 0)", // then the railroad over it
	}
	if !slices.Equal(strokes, want) {
		t.Errorf("widths and colors in drawing order = %v, want %v", strokes, want)
	}
}

func TestDrawRoadsSkipsTilesWithNoRoute(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	// RouteType 255 means "no route" on both tiles, and neither has a city name.
	mapData := newRoadTestMapData(255, 255)

	mr.DrawRoads(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	ops := canvas.GetOperations()
	if len(ops) != 0 {
		t.Fatalf("DrawRoads() recorded %d ops, want 0: %v", len(ops), ops)
	}
}

func TestDrawRoadsNoImprovementsIsNoOp(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{}}

	mr.DrawRoads(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

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

// Each tile paints, in its own color, only pixels of its own hex that lie within borderReach of the other tile.
func TestDrawBordersPaintsEachOwnersSideOfTheBoundary(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)
	mapData := newBorderTestMapData(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")
	mapSize := fileio.MapSize{Height: 1, Width: 2}

	mr.DrawBorders(canvas, mapData, mapSize)

	grid := mr.tileGridFor(mapSize)
	painted := map[int]int{} // tile id -> pixels painted for it
	index := -1
	for _, op := range canvas.GetOperations() {
		var r, g, b, x, y, i int
		if n, _ := fmt.Sscanf(op, "IndexFor(%d, %d, %d) = %d", &r, &g, &b, &i); n == 4 {
			index = i
			continue
		}
		if n, _ := fmt.Sscanf(op, "PaintPixel(%d, %d, %d)", &x, &y, &i); n != 3 {
			continue
		}
		if int32(index) != grid.at(x, y) {
			t.Fatalf("pixel (%d, %d) belongs to tile %d but was painted for tile %d", x, y, grid.at(x, y), index)
		}
		near := false
		for _, p := range borderProbes {
			if other := grid.at(x+p[0], y+p[1]); other >= 0 && other != int32(index) {
				near = true
			}
		}
		if !near {
			t.Fatalf("pixel (%d, %d) painted for tile %d is not within %d px of the other tile", x, y, index, borderReach)
		}
		painted[index]++
	}
	if painted[0] == 0 || painted[1] == 0 {
		t.Errorf("painted pixels per tile = %v, want both tiles to have a border", painted)
	}
}

func TestDrawBordersSameOwnerDrawsNothing(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newBorderTestMapData(0, 0, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK")

	mr.DrawBorders(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	if ops := canvas.GetOperations(); len(ops) != 0 {
		t.Fatalf("DrawBorders() with same owner recorded %v, want no ops", ops)
	}
}

func TestDrawBordersInvalidOwnerSkipsTile(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newBorderTestMapData(-1, -1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK")

	mr.DrawBorders(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	if ops := canvas.GetOperations(); len(ops) != 0 {
		t.Fatalf("DrawBorders() with invalid owners recorded %v, want no ops", ops)
	}
}

// Every city name has a halo: its text repeated one pixel up, down, left and right in the halo color, then the name itself over it.
func haloOps(label ColoredText, halo string) []string {
	draw := func(dx, dy float64) string {
		return fmt.Sprintf(`DrawString("%s", %.2f, %.2f)`, label.Text, label.X+dx, label.Y+dy)
	}
	return []string{
		halo, draw(-1, 0), draw(1, 0), draw(0, -1), draw(0, 1),
		fmt.Sprintf("SetColor(%d, %d, %d)", label.R, label.G, label.B), draw(0, 0),
	}
}

func TestLabelHaloColorIsWhicheverOfDarkAndLightContrastsMoreWithTheText(t *testing.T) {
	tests := []struct {
		text color.RGBA
		want color.RGBA
	}{
		{color.RGBA{255, 255, 255, 255}, labelHaloDark},
		{color.RGBA{200, 200, 200, 255}, labelHaloDark},
		{color.RGBA{128, 128, 128, 255}, labelHaloDark}, // a mid grey: 5.3 against the dark halo, 3.9 against the light one
		{color.RGBA{30, 30, 30, 255}, labelHaloLight},
		{color.RGBA{0, 0, 0, 255}, labelHaloLight},
		{color.RGBA{76, 51, 0, 255}, labelHaloLight},
	}
	for _, tt := range tests {
		if got := labelHaloColor(tt.text); got != tt.want {
			t.Errorf("labelHaloColor(%v) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestDrawHaloedLabelDrawsTheHaloThenTheLabel(t *testing.T) {
	canvas := NewMockCanvas(200, 200)
	label := ColoredText{Text: "Rome", X: 10, Y: 20, R: 30, G: 30, B: 30} // dark text, so a light halo

	drawHaloedLabel(canvas, label)

	if want := haloOps(label, "SetColor(244, 244, 244)"); !slices.Equal(canvas.GetOperations(), want) {
		t.Errorf("ops = %v, want %v", canvas.GetOperations(), want)
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
	size := fileio.MapSize{Height: 1, Width: 1}

	mr.DrawPhysicalCityNames(canvas, mapData, size)

	label := PhysicalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, layoutForMap(size, mr.config.Radius))
	if want := haloOps(label, "SetColor(18, 18, 18)"); !slices.Equal(canvas.GetOperations(), want) { // white text, so a dark halo
		t.Errorf("ops = %v, want %v", canvas.GetOperations(), want)
	}
}

// Tiles with no city draw nothing at all: no halo, no empty label.
func TestDrawPhysicalCityNamesSkipsTilesWithNoCity(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{}, {CityName: "Rome"}, {}},
		},
	}

	mr.DrawPhysicalCityNames(canvas, mapData, fileio.MapSize{Height: 1, Width: 3})

	if got := len(canvas.GetOperations()); got != 7 {
		t.Errorf("recorded %d ops, want the 7 of one haloed label: %v", got, canvas.GetOperations())
	}
}

func TestDrawPhysicalCityNamesNoImprovementsIsNoOp(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{}}

	mr.DrawPhysicalCityNames(canvas, mapData, fileio.MapSize{Height: 1, Width: 1})

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

	mr.DrawPoliticalCityNames(canvas, mapData, fileio.MapSize{Height: 1, Width: 1}, newTileLayout(mr.config.Radius, 100))

	ops := canvas.GetOperations()
	if len(ops) != 7 {
		t.Fatalf("DrawPoliticalCityNames() recorded %d ops, want 7 (halo and label): %v", len(ops), ops)
	}
	if ops[5] == "SetColor(255, 255, 255)" {
		t.Errorf("DrawPoliticalCityNames() should use the civ color, not the white fallback: %q", ops[5])
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

	mr.DrawPoliticalCityNames(canvas, mapData, fileio.MapSize{Height: 1, Width: 1}, newTileLayout(mr.config.Radius, 100))

	ops := canvas.GetOperations()
	if len(ops) != 7 || ops[0] != "SetColor(18, 18, 18)" || ops[5] != "SetColor(255, 255, 255)" {
		t.Fatalf("DrawPoliticalCityNames() with unowned tile = %v, want a dark halo, then the white fallback label", ops)
	}
}

// A halo goes on every label, whatever the ground: each of two cities gets its own halo and label.
func TestDrawPoliticalCityNamesHalosEveryLabel(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{CityName: "Rome", Owner: -1, CityId: -1}, {CityName: "Kyiv", Owner: -1, CityId: -1}},
		},
	}

	mr.DrawPoliticalCityNames(canvas, mapData, fileio.MapSize{Height: 1, Width: 2}, newTileLayout(mr.config.Radius, 100))

	if got := len(canvas.GetOperations()); got != 14 {
		t.Errorf("recorded %d ops, want 14: a halo and a label for each of two cities", got)
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

	mr.DrawTerritoryTiles(canvas, mapData, fileio.MapSize{Height: 1, Width: 1})

	ops := canvas.GetOperations()
	// Fill and outline, no mountain, no city.
	if len(ops) != 9 {
		t.Fatalf("DrawTerritoryTiles() water tile recorded %d ops, want 9 (fill + outline): %v", len(ops), ops)
	}
	oceanColor := GetPhysicalMapTileColor("TERRAIN_OCEAN")
	wantColorOp := fmt.Sprintf("SetColor(%d, %d, %d)", oceanColor.R, oceanColor.G, oceanColor.B)
	if ops[1] != wantColorOp {
		t.Errorf("DrawTerritoryTiles() water color op = %q, want %q", ops[1], wantColorOp)
	}
}

func TestDrawTerritoryTilesUnownedLandUsesTerrainColor(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newTerritoryTestMapData(0 /* TERRAIN_GRASS */, -1, "", "")

	mr.DrawTerritoryTiles(canvas, mapData, fileio.MapSize{Height: 1, Width: 1})

	ops := canvas.GetOperations()
	if len(ops) != 9 {
		t.Fatalf("DrawTerritoryTiles() unowned land recorded %d ops, want 9 (fill + outline): %v", len(ops), ops)
	}
}

// Cities are not drawn with the tiles: DrawCityMarkers draws them later, over the routes.
func TestDrawTerritoryTilesLeavesCitiesToDrawCityMarkers(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_BLACK", "CIVILIZATION_ROME")
	mapData.MapTileImprovements[0][0].CityId = 0 // has a city

	mr.DrawTerritoryTiles(canvas, mapData, fileio.MapSize{Height: 1, Width: 1})

	ops := canvas.GetOperations()
	if len(ops) != 9 { // the tile's fill and outline only
		t.Fatalf("DrawTerritoryTiles() owned major civ with city recorded %d ops, want 9 (fill + outline): %v", len(ops), ops)
	}
}

// A city is a circle on a slightly larger one in the halo color, both at the tile's center.
func TestDrawCityMarkersDrawsAnOutlinedCircleOnEveryCity(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{CityId: 0}, {CityId: -1}, {CityId: 1}},
		},
	}
	size := fileio.MapSize{Height: 1, Width: 3}
	l := layoutForMap(size, mr.config.Radius)
	cityColor := color.RGBA{200, 100, 50, 255}

	mr.DrawCityMarkers(canvas, mapData, size, func(fileio.TilePos) color.RGBA { return cityColor })

	fill := markerColor(cityColor)
	halo := labelHaloColor(fill)
	radius := mr.config.Radius * cityMarkerRadius
	var want []string
	for _, col := range []int{0, 2} { // the tile between them has no city
		x, y := l.center(fileio.TilePos{Row: 0, Col: col})
		want = append(want,
			fmt.Sprintf("DrawRegularPolygon(24, %.2f, %.2f, %.2f, 0.00)", x, y, radius+1), fmt.Sprintf("SetColor(%d, %d, %d)", halo.R, halo.G, halo.B), "Fill()",
			fmt.Sprintf("DrawRegularPolygon(24, %.2f, %.2f, %.2f, 0.00)", x, y, radius), fmt.Sprintf("SetColor(%d, %d, %d)", fill.R, fill.G, fill.B), "Fill()")
	}
	if got := canvas.GetOperations(); !slices.Equal(got, want) {
		t.Errorf("ops = %v, want %v", got, want)
	}
}

// A marker's outline is dark around a light circle and light around a dark one, like the labels' halos.
func TestDrawCityMarkersOutlineContrastsWithTheCircle(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: 0}}}}
	size := fileio.MapSize{Height: 1, Width: 1}
	for _, tt := range []struct {
		name string
		city color.RGBA
		want string
	}{
		{"white", color.RGBA{255, 255, 255, 255}, "SetColor(18, 18, 18)"},
		{"black", color.RGBA{0, 0, 0, 255}, "SetColor(244, 244, 244)"},
	} {
		canvas := NewMockCanvas(200, 200)
		mr.DrawCityMarkers(canvas, mapData, size, func(fileio.TilePos) color.RGBA { return tt.city })
		if got := canvas.GetOperations()[1]; got != tt.want {
			t.Errorf("%s city: outline op = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// On the physical map every city marker is white, so its outline is the dark one.
func TestDrawPhysicalMapCityMarkersAreWhite(t *testing.T) {
	mapData := newTerritoryTestMapData(0, -1, "", "")
	mapData.MapTileImprovements[0][0].CityId = 0
	canvas := NewMockCanvas(400, 200)

	NewMapRenderer(DefaultDrawingConfig()).DrawPhysicalMap(canvas, mapData)

	ops := canvas.GetOperations()
	for i, op := range ops {
		if strings.HasPrefix(op, "DrawRegularPolygon(24,") {
			if ops[i+1] != "SetColor(18, 18, 18)" || ops[i+4] != "SetColor(255, 255, 255)" {
				t.Errorf("marker colors = %q then %q, want a dark outline, then white", ops[i+1], ops[i+4])
			}
			return
		}
	}
	t.Error("no city marker was drawn")
}

// The routes all lead to a city's center, so markers go on after every route and nothing draws over one.
func TestMapsDrawCityMarkersAfterTheRoutes(t *testing.T) {
	newMap := func() *fileio.Civ5MapData {
		mapData := newTerritoryTestMapData(0, -1, "", "")
		mapData.MapTiles = [][]*fileio.Civ5MapTilePhysical{{{TerrainType: 0}, {TerrainType: 0}}}
		mapData.MapTileImprovements = [][]*fileio.Civ5MapTileImprovement{
			{{CityId: 0, CityName: "Rome", Owner: -1, RouteType: 1}, {CityId: -1, Owner: -1, RouteType: 1}},
		}
		return mapData
	}
	for name, drawMap := range map[string]func(*MapRenderer, Canvas, *fileio.Civ5MapData) image.Image{
		"political": (*MapRenderer).DrawPoliticalMap,
		"physical":  (*MapRenderer).DrawPhysicalMap,
	} {
		canvas := NewMockCanvas(400, 200)
		drawMap(NewMapRenderer(DefaultDrawingConfig()), canvas, newMap())

		lastStroke, firstMarker, railroad := -1, -1, -1
		for i, op := range canvas.GetOperations() {
			if op == "Stroke()" {
				lastStroke = i
			}
			if op == "SetColor(76, 51, 0)" && railroad < 0 { // the railroad's line
				railroad = i
			}
			if strings.HasPrefix(op, "DrawRegularPolygon(24,") && firstMarker < 0 {
				firstMarker = i
			}
		}
		if railroad < 0 || firstMarker < railroad {
			t.Errorf("%s map: railroad drawn at op %d, first city marker at op %d, want the railroad drawn, and before the marker", name, railroad, firstMarker)
		}
		if lastStroke < 0 || firstMarker < 0 || firstMarker < lastStroke {
			t.Errorf("%s map: first city marker at op %d, last stroke at op %d, want the markers after every stroke", name, firstMarker, lastStroke)
		}
	}
}

func TestDrawTerritoryTilesOwnedUnknownColorFallsBackToBlack(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)

	// Owned by a civ whose team color isn't a recognized key in civColorMap.
	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_DOES_NOT_EXIST", "CIVILIZATION_ROME")

	mr.DrawTerritoryTiles(canvas, mapData, fileio.MapSize{Height: 1, Width: 1})

	ops := canvas.GetOperations()
	if len(ops) != 9 {
		t.Fatalf("DrawTerritoryTiles() unknown owner color recorded %d ops, want 9 (fill + outline): %v", len(ops), ops)
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

func TestDrawPhysicalMapResizesCanvas(t *testing.T) {
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
}

func TestDrawPoliticalMapResizesCanvas(t *testing.T) {
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

// A mountain tile's ground is darkened toward the mountain color; other tiles keep theirs.
func TestDrawTerrainTilesTintsMountainTiles(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(200, 200)
	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS"},
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{TerrainType: 0, Elevation: 0}, {TerrainType: 0, Elevation: 2}},
		},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{},
	}
	l := layoutForMap(fileio.MapSize{Height: 1, Width: 2}, mr.config.Radius)
	plain := PhysicalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, l)
	plainFill := color.RGBA{plain.R, plain.G, plain.B, 255}
	tinted := mountainTileColor(plainFill)

	mr.DrawTerrainTiles(canvas, mapData, fileio.MapSize{Height: 1, Width: 2})

	ops := canvas.GetOperations()
	if want := fmt.Sprintf("SetColor(%d, %d, %d)", plainFill.R, plainFill.G, plainFill.B); ops[1] != want {
		t.Errorf("plain tile fill = %q, want %q", ops[1], want)
	}
	if want := fmt.Sprintf("SetColor(%d, %d, %d)", tinted.R, tinted.G, tinted.B); ops[4] != want {
		t.Errorf("mountain tile fill = %q, want %q", ops[4], want)
	}
	if tinted == plainFill {
		t.Error("mountainTileColor() left the ground unchanged")
	}
}

// A peak is an outline triangle under a lit and a shaded face that meet at its ridge, with snow on the top of each face.
func TestDrawPeakDrawsOutlinedFacesWithSnow(t *testing.T) {
	canvas := NewMockCanvas(100, 100)

	drawPeak(canvas, peakAt(50, 50, 16, 1))

	ops := canvas.GetOperations()
	want := []string{
		"DrawTriangle(50.00, 38.40, 39.70, 57.40, 60.30, 57.40)", // outline, a little larger than the faces
		"SetColor(40, 38, 36)", "Fill()",
		"DrawTriangle(50.00, 40.40, 41.20, 56.40, 50.00, 56.40)", // lit face, left of the ridge
		"SetColor(132, 132, 124)", "Fill()",
		"DrawTriangle(50.00, 40.40, 50.00, 56.40, 58.80, 56.40)", // shaded face, right of it
		"SetColor(70, 70, 66)", "Fill()",
		"DrawTriangle(50.00, 40.40, 47.36, 45.20, 50.00, 45.20)", // snow on the lit face
		"SetColor(250, 252, 255)", "Fill()",
		"DrawTriangle(50.00, 40.40, 50.00, 45.20, 52.64, 45.20)", // snow on the shaded face
		"SetColor(190, 200, 216)", "Fill()",
	}
	if !slices.Equal(ops, want) {
		t.Errorf("drawPeak() ops =\n%v\nwant\n%v", ops, want)
	}
}

// terrainWithMountains returns a map of grass with mountains where mountains says.
func terrainWithMountains(mountains [][]bool) *fileio.Civ5MapData {
	mapData := &fileio.Civ5MapData{
		TerrainList:         []string{"TERRAIN_GRASS"},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{},
	}
	for _, row := range mountains {
		var tiles []*fileio.Civ5MapTilePhysical
		for _, mountain := range row {
			elevation := 0
			if mountain {
				elevation = 2
			}
			tiles = append(tiles, &fileio.Civ5MapTilePhysical{TerrainType: 0, Elevation: elevation})
		}
		mapData.MapTiles = append(mapData.MapTiles, tiles)
	}
	return mapData
}

// rangePeaksOf returns rangePeaks of the terrain and the layout it was placed with.
func rangePeaksOf(mountains [][]bool) ([]peak, tileLayout) {
	size := fileio.MapSize{Height: len(mountains), Width: len(mountains[0])}
	l := layoutForMap(size, DefaultDrawingConfig().Radius)
	return rangePeaks(terrainWithMountains(mountains), size, l), l
}

func centerOf(l tileLayout, row, col int) (x, y float64) {
	return l.center(fileio.TilePos{Row: row, Col: col})
}

// Three mutually adjacent mountains get one peak in the middle of them, and none for the pairs among them.
func TestRangePeaksPutOnePeakInTheMiddleOfThreeMountains(t *testing.T) {
	peaks, l := rangePeaksOf([][]bool{{true, true}, {true, false}}) // (0,0), (0,1) and (1,0) touch each other
	x0, y0 := centerOf(l, 0, 0)
	x1, y1 := centerOf(l, 0, 1)
	x2, y2 := centerOf(l, 1, 0)

	if len(peaks) != 4 {
		t.Fatalf("got %d peaks, want the three tiles' and one in the middle: %v", len(peaks), peaks)
	}
	want := peakAt((x0+x1+x2)/3, (y0+y1+y2)/3, l.radius, trioPeakScale)
	if !slices.Contains(peaks, want) {
		t.Errorf("no middle peak %v in %v", want, peaks)
	}
}

// Two adjacent mountains get a smaller peak between them, whichever way they touch.
func TestRangePeaksFillTheGapBetweenAPairInAnyDirection(t *testing.T) {
	for name, tc := range map[string]struct {
		mountains              [][]bool
		row1, col1, row2, col2 int
	}{
		"east-west": {[][]bool{{true, true}, {false, false}}, 0, 0, 0, 1},
		"below":     {[][]bool{{true, false}, {true, false}}, 0, 0, 1, 0},
		"diagonal":  {[][]bool{{false, true}, {true, false}}, 0, 1, 1, 0},
	} {
		peaks, l := rangePeaksOf(tc.mountains)
		x1, y1 := centerOf(l, tc.row1, tc.col1)
		x2, y2 := centerOf(l, tc.row2, tc.col2)
		want := peakAt((x1+x2)/2, (y1+y2)/2, l.radius, gapPeakScale)

		if len(peaks) != 3 {
			t.Errorf("%s: got %d peaks, want two tiles' and one between: %v", name, len(peaks), peaks)
		}
		if !slices.Contains(peaks, want) {
			t.Errorf("%s: no gap peak %v in %v", name, want, peaks)
		}
	}
}

func TestRangePeaksLeaveLoneAndDistantMountainsAlone(t *testing.T) {
	for name, mountains := range map[string][][]bool{
		"alone":   {{true, false, false}},
		"apart":   {{true, false, true}},
		"no ring": {{true, false}, {false, false}, {false, true}},
	} {
		want := 0
		for _, row := range mountains {
			for _, m := range row {
				if m {
					want++
				}
			}
		}
		if peaks, _ := rangePeaksOf(mountains); len(peaks) != want {
			t.Errorf("%s: got %d peaks, want %d (the tiles')", name, len(peaks), want)
		}
	}
}

// Counting the threes and pairs among all tiles directly gives the number of peaks a cluster of mountains should get.
func TestRangePeaksCountMatchesThreesAndPairsOfAnyCluster(t *testing.T) {
	mountains := [][]bool{
		{true, true, true, false, true},
		{true, true, false, true, true},
		{false, true, true, true, false},
		{true, false, true, true, true},
	}
	adjacent := func(a, b fileio.TilePos) bool { n := fileio.GetNeighbors(a); return slices.Contains(n[:], b) }
	var tiles []fileio.TilePos
	for row, cols := range mountains {
		for col, m := range cols {
			if m {
				tiles = append(tiles, fileio.TilePos{Row: row, Col: col})
			}
		}
	}
	trios, pairs := 0, 0
	inTrio := map[[2]fileio.TilePos]bool{}
	for i, a := range tiles {
		for j := i + 1; j < len(tiles); j++ {
			for k := j + 1; k < len(tiles); k++ {
				if b, c := tiles[j], tiles[k]; adjacent(a, b) && adjacent(a, c) && adjacent(b, c) {
					trios++
					inTrio[[2]fileio.TilePos{a, b}], inTrio[[2]fileio.TilePos{a, c}], inTrio[[2]fileio.TilePos{b, c}] = true, true, true
				}
			}
		}
	}
	for i, a := range tiles {
		for _, b := range tiles[i+1:] {
			if adjacent(a, b) && !inTrio[[2]fileio.TilePos{a, b}] {
				pairs++
			}
		}
	}

	peaks, _ := rangePeaksOf(mountains)

	if want := len(tiles) + trios + pairs; len(peaks) != want || trios == 0 || pairs == 0 {
		t.Errorf("got %d peaks, want %d tiles + %d threes + %d pairs (a cluster with both)", len(peaks), len(tiles), trios, pairs)
	}
}

// A peak lower on the map is nearer, so it comes after the ones behind it.
func TestRangePeaksAreBackToFront(t *testing.T) {
	peaks, _ := rangePeaksOf([][]bool{{true, true, true}, {true, true, true}, {true, true, true}})

	if !slices.IsSortedFunc(peaks, func(a, b peak) int { return cmp.Compare(a.base, b.base) }) {
		t.Errorf("peaks are not in order of their ground line: %v", peaks)
	}
}

// Borders and rivers run along tile edges, close to the peaks, so the peaks go on after both and nothing covers one; the routes cross them.
func TestPoliticalMapDrawsMountainsAfterBordersAndRiversAndBeforeRoutes(t *testing.T) {
	mapData := newBorderTestMapData(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")
	mapData.TerrainList = []string{"TERRAIN_GRASS"}
	mapData.MapTiles = [][]*fileio.Civ5MapTilePhysical{
		{{TerrainType: 0, Elevation: 2, RiverData: 1}, {TerrainType: 0, Elevation: 2}},
	}
	for _, tile := range mapData.MapTileImprovements[0] {
		tile.RouteType = 1
	}
	riverColorOp := fmt.Sprintf("SetColor(%d, %d, %d)", mapRiverColor.R, mapRiverColor.G, mapRiverColor.B)
	railroadColorOp := fmt.Sprintf("SetColor(%d, %d, %d)", railroadColor.R, railroadColor.G, railroadColor.B)

	canvas := NewMockCanvas(400, 200)
	NewMapRenderer(DefaultDrawingConfig()).DrawPoliticalMap(canvas, mapData)

	lastBorder, lastRiver, firstPeak, lastPeak, firstRailroad := -1, -1, -1, -1, -1
	for i, op := range canvas.GetOperations() {
		switch {
		case strings.HasPrefix(op, "PaintPixel("):
			lastBorder = i
		case op == riverColorOp:
			lastRiver = i
		case strings.HasPrefix(op, "DrawTriangle("):
			if firstPeak < 0 {
				firstPeak = i
			}
			lastPeak = i
		case op == railroadColorOp && firstRailroad < 0:
			firstRailroad = i
		}
	}
	if lastBorder < 0 || lastRiver < 0 || firstPeak < 0 || firstRailroad < 0 {
		t.Fatalf("border at op %d, river at %d, peak at %d, railroad at %d: want all drawn", lastBorder, lastRiver, firstPeak, firstRailroad)
	}
	if firstPeak < lastBorder || firstPeak < lastRiver {
		t.Errorf("first peak at op %d, last border at %d, last river at %d, want the peaks after both", firstPeak, lastBorder, lastRiver)
	}
	if lastPeak > firstRailroad {
		t.Errorf("last peak at op %d, first railroad at %d, want the routes over the peaks", lastPeak, firstRailroad)
	}
}

func TestPhysicalMapDrawsMountainsAfterRiversAndBeforeRoutes(t *testing.T) {
	mapData := terrainWithMountains([][]bool{{true, true}})
	mapData.MapTiles[0][0].RiverData = 1
	mapData.MapTileImprovements = [][]*fileio.Civ5MapTileImprovement{{{CityId: -1, Owner: -1, RouteType: 1}, {CityId: -1, Owner: -1, RouteType: 1}}}
	riverColorOp := fmt.Sprintf("SetColor(%d, %d, %d)", mapRiverColor.R, mapRiverColor.G, mapRiverColor.B)
	railroadColorOp := fmt.Sprintf("SetColor(%d, %d, %d)", railroadColor.R, railroadColor.G, railroadColor.B)

	canvas := NewMockCanvas(400, 200)
	NewMapRenderer(DefaultDrawingConfig()).DrawPhysicalMap(canvas, mapData)

	lastRiver, firstPeak, lastPeak, firstRailroad := -1, -1, -1, -1
	for i, op := range canvas.GetOperations() {
		switch {
		case op == riverColorOp:
			lastRiver = i
		case strings.HasPrefix(op, "DrawTriangle("):
			if firstPeak < 0 {
				firstPeak = i
			}
			lastPeak = i
		case op == railroadColorOp && firstRailroad < 0:
			firstRailroad = i
		}
	}
	if lastRiver < 0 || firstPeak < 0 || firstRailroad < 0 {
		t.Fatalf("river at op %d, peak at %d, railroad at %d: want all drawn", lastRiver, firstPeak, firstRailroad)
	}
	if firstPeak < lastRiver || lastPeak > firstRailroad {
		t.Errorf("peaks at ops %d..%d, last river at %d, first railroad at %d, want the peaks between them", firstPeak, lastPeak, lastRiver, firstRailroad)
	}
}
