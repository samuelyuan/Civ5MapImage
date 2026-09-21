package graphics

import (
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

func TestRiverEdgesForTileAllEdges(t *testing.T) {
	const cx, cy, radius = 100.0, 50.0, 16.0
	// bit0 = east, bit1 = southeast, bit2 = southwest
	edges := RiverEdgesForTile(0b111, cx, cy, newTileLayout(radius, 100))

	want := []Line{
		getHexEdge(3, cx, cy, radius), // southwest
		getHexEdge(4, cx, cy, radius), // southeast
		getHexEdge(5, cx, cy, radius), // east
	}
	if !reflect.DeepEqual(edges, want) {
		t.Errorf("RiverEdgesForTile(0b111) = %v, want %v", edges, want)
	}
}

func TestRiverEdgesForTileNoRiver(t *testing.T) {
	if edges := RiverEdgesForTile(0, 100, 50, newTileLayout(16, 100)); edges != nil {
		t.Errorf("RiverEdgesForTile(0) = %v, want nil", edges)
	}
}

func TestRiverEdgesForTileSingleBits(t *testing.T) {
	const cx, cy, radius = 100.0, 50.0, 16.0
	tests := []struct {
		name      string
		riverData int
		wantEdge  int
	}{
		{"southwest only", 0b100, 3},
		{"southeast only", 0b010, 4},
		{"east only", 0b001, 5},
	}
	for _, tt := range tests {
		edges := RiverEdgesForTile(tt.riverData, cx, cy, newTileLayout(radius, 100))
		want := []Line{getHexEdge(tt.wantEdge, cx, cy, radius)}
		if !reflect.DeepEqual(edges, want) {
			t.Errorf("%s: RiverEdgesForTile(%b) = %v, want %v", tt.name, tt.riverData, edges, want)
		}
	}
}

// newRoadGeometryTestMap builds a 1x2 map (two side-by-side tiles) for RoadSegmentsForTile tests.
func newRoadGeometryTestMap(routeType0, routeType1 int, cityName1 string) *fileio.Civ5MapData {
	return &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, RouteType: routeType0, CityId: -1},
				{X: 1, Y: 0, RouteType: routeType1, CityId: -1, CityName: cityName1},
			},
		},
	}
}

func TestRoadSegmentsForTileNoRoute(t *testing.T) {
	mapData := newRoadGeometryTestMap(255, 0, "")
	if segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100)); segments != nil {
		t.Errorf("RoadSegmentsForTile() with RouteType 255 = %v, want nil", segments)
	}
}

func TestRoadSegmentsForTileConnectsToNeighborWithRoute(t *testing.T) {
	const radius = 16.0
	mapData := newRoadGeometryTestMap(0, 0, "") // both tiles have a road

	l := newTileLayout(radius, 100)
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, l)
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() = %d segments, want 1: %v", len(segments), segments)
	}

	seg := segments[0]
	x1, y1 := l.center(fileio.TilePos{Row: 0, Col: 0})
	x2, y2 := l.center(fileio.TilePos{Row: 0, Col: 1})
	wantLine := Line{X1: x1, Y1: y1, X2: (x1 + x2) / 2.0, Y2: (y1 + y2) / 2.0}
	if seg.Line != wantLine {
		t.Errorf("segment line = %+v, want %+v", seg.Line, wantLine)
	}
	if seg.LineWidth != 1.0 || seg.R != 51 || seg.G != 51 || seg.B != 51 {
		t.Errorf("segment style = (width=%v, rgb=%d,%d,%d), want road style (1.0, 51,51,51)", seg.LineWidth, seg.R, seg.G, seg.B)
	}
}

func TestRoadSegmentsForTileRailroadStyle(t *testing.T) {
	mapData := newRoadGeometryTestMap(1, 1, "") // railroad
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() = %d segments, want 1", len(segments))
	}
	seg := segments[0]
	if seg.LineWidth != 2.0 || seg.R != 76 || seg.G != 51 || seg.B != 0 {
		t.Errorf("segment style = (width=%v, rgb=%d,%d,%d), want railroad style (2.0, 76,51,0)", seg.LineWidth, seg.R, seg.G, seg.B)
	}
}

func TestRoadSegmentsForTileConnectsToCityWithNoRoute(t *testing.T) {
	// Neighbor has RouteType 255 (no route) but has a city name -- should still connect.
	mapData := newRoadGeometryTestMap(0, 255, "Rome")
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() to city with no route = %d segments, want 1", len(segments))
	}
}

func TestRoadSegmentsForTileSkipsDisconnectedNeighbor(t *testing.T) {
	// Neighbor has no route and no city -- should not connect.
	mapData := newRoadGeometryTestMap(0, 255, "")
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if len(segments) != 0 {
		t.Errorf("RoadSegmentsForTile() to disconnected neighbor = %d segments, want 0", len(segments))
	}
}

func TestRoadSegmentsForTileSkipsOutOfBoundsNeighbor(t *testing.T) {
	// A 1x1 map has no valid neighbors at all.
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{X: 0, Y: 0, RouteType: 0, CityId: -1}},
		},
	}
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 1}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if len(segments) != 0 {
		t.Errorf("RoadSegmentsForTile() on 1x1 map = %d segments, want 0", len(segments))
	}
}

// newBorderGeometryTestMap builds a 1x2 map for BorderSegmentsForTile tests.
func newBorderGeometryTestMap(owner0, owner1 int, teamColor0, teamColor1 string) *fileio.Civ5MapData {
	return &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{
				{X: 0, Y: 0, Owner: owner0, CityId: -1},
				{X: 1, Y: 0, Owner: owner1, CityId: -1},
			},
		},
		Civ5PlayerData: []*fileio.Civ5PlayerData{
			{Index: 0, CivType: "CIVILIZATION_ROME", TeamColor: teamColor0},
			{Index: 1, CivType: "CIVILIZATION_GREECE", TeamColor: teamColor1},
		},
		CityOwnerIndexMap: map[int]int{0: 0, 1: 1},
	}
}

func TestBorderSegmentsForTileInvalidOwner(t *testing.T) {
	mapData := newBorderGeometryTestMap(-1, -1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK")
	if segments := BorderSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100)); segments != nil {
		t.Errorf("BorderSegmentsForTile() with invalid owner = %v, want nil", segments)
	}
}

func TestBorderSegmentsForTileSameOwnerNoBorder(t *testing.T) {
	mapData := newBorderGeometryTestMap(0, 0, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK")
	if segments := BorderSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100)); len(segments) != 0 {
		t.Errorf("BorderSegmentsForTile() with same owner = %v, want empty", segments)
	}
}

func TestBorderSegmentsForTileDifferentOwnerDrawsBorder(t *testing.T) {
	const radius = 16.0
	mapData := newBorderGeometryTestMap(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")

	l := newTileLayout(radius, 100)
	segments := BorderSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, l)
	if len(segments) != 1 {
		t.Fatalf("BorderSegmentsForTile() = %d segments, want 1: %v", len(segments), segments)
	}

	x1, y1 := l.center(fileio.TilePos{Row: 0, Col: 0})
	renderColor := civColorMap["PLAYERCOLOR_BLACK"]
	wantLine := getHexEdge(5, x1, y1, radius-1) // neighbor (1,0) is edge index 5 for an even row
	if segments[0].Line != wantLine {
		t.Errorf("segment line = %+v, want %+v", segments[0].Line, wantLine)
	}
	if segments[0].R != renderColor.InnerColor.R || segments[0].G != renderColor.InnerColor.G || segments[0].B != renderColor.InnerColor.B {
		t.Errorf("segment color = (%d,%d,%d), want inner color %+v", segments[0].R, segments[0].G, segments[0].B, renderColor.InnerColor)
	}
	if segments[0].LineWidth != BorderLineWidth {
		t.Errorf("segment LineWidth = %v, want %v", segments[0].LineWidth, BorderLineWidth)
	}
}

func TestBorderSegmentsForTileUnknownColorFallsBackToWhite(t *testing.T) {
	mapData := newBorderGeometryTestMap(0, 1, "PLAYERCOLOR_DOES_NOT_EXIST", "PLAYERCOLOR_ALSO_MISSING")
	segments := BorderSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if len(segments) != 1 {
		t.Fatalf("BorderSegmentsForTile() = %d segments, want 1", len(segments))
	}
	if segments[0].R != 255 || segments[0].G != 255 || segments[0].B != 255 {
		t.Errorf("segment color = (%d,%d,%d), want white fallback (255,255,255)", segments[0].R, segments[0].G, segments[0].B)
	}
}

// newLabelGeometryTestMap builds a 1x1 map with a single named city tile for label tests.
func newLabelGeometryTestMap(cityName string, owner int, teamColor, civType string) *fileio.Civ5MapData {
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{X: 0, Y: 0, CityId: 0, CityName: cityName, Owner: owner}},
		},
	}
	if civType != "" {
		mapData.Civ5PlayerData = []*fileio.Civ5PlayerData{{Index: 0, CivType: civType, TeamColor: teamColor}}
		mapData.CityOwnerIndexMap = map[int]int{owner: 0}
	}
	return mapData
}

func TestPhysicalCityNameLabel(t *testing.T) {
	const radius = 16.0
	mapData := newLabelGeometryTestMap("Rome", -1, "", "")

	label := PhysicalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(radius, 100))

	// Tile (0, 0) is centered at (24, 84) on a canvas 100 tall; the name is centered above it: 6 px per character, 4 characters.
	wantX, wantY := 24.0-12.0, 84.0-8.0
	if label.Text != "Rome" || label.X != wantX || label.Y != wantY {
		t.Errorf("PhysicalCityNameLabel() = %+v, want {Text:Rome X:%v Y:%v}", label, wantX, wantY)
	}
	if label.R != 255 || label.G != 255 || label.B != 255 {
		t.Errorf("PhysicalCityNameLabel() color = (%d,%d,%d), want white", label.R, label.G, label.B)
	}
}

func TestPhysicalCityNameLabelTrimsNullByte(t *testing.T) {
	mapData := newLabelGeometryTestMap("Rome\x00garbage", -1, "", "")
	label := PhysicalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if label.Text != "Rome" {
		t.Errorf("PhysicalCityNameLabel().Text = %q, want %q", label.Text, "Rome")
	}
}

func TestPoliticalCityNameLabelKnownColor(t *testing.T) {
	const radius = 16.0
	mapData := newLabelGeometryTestMap("Rome", 0, "PLAYERCOLOR_BLACK", "CIVILIZATION_ROME")

	label := PoliticalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(radius, 100))

	renderColor := civColorMap["PLAYERCOLOR_BLACK"]
	wantColor := blendColor(renderColor.InnerColor, color.RGBA{255, 255, 255, 255}, 0.2)
	if label.R != wantColor.R || label.G != wantColor.G || label.B != wantColor.B {
		t.Errorf("PoliticalCityNameLabel() color = %d,%d,%d, want fully blended %d,%d,%d",
			label.R, label.G, label.B, wantColor.R, wantColor.G, wantColor.B)
	}
}

func TestPoliticalCityNameLabelUnknownColorFallsBackToWhite(t *testing.T) {
	mapData := newLabelGeometryTestMap("Rome", -1, "", "")
	label := PoliticalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if label.R != 255 || label.G != 255 || label.B != 255 {
		t.Errorf("PoliticalCityNameLabel() color = (%d,%d,%d), want white fallback", label.R, label.G, label.B)
	}
}

func TestPhysicalHexTile(t *testing.T) {
	const radius = 16.0
	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_OCEAN"},
		MapTiles:    [][]*fileio.Civ5MapTilePhysical{{{TerrainType: 0}}},
	}

	l := newTileLayout(radius, 100)
	hex := PhysicalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, l)

	wantX, wantY := l.center(fileio.TilePos{Row: 0, Col: 0})
	oceanColor := GetPhysicalMapTileColor("TERRAIN_OCEAN")
	if hex.X != wantX || hex.Y != wantY {
		t.Errorf("PhysicalHexTile() position = (%v,%v), want (%v,%v)", hex.X, hex.Y, wantX, wantY)
	}
	if hex.R != oceanColor.R || hex.G != oceanColor.G || hex.B != oceanColor.B {
		t.Errorf("PhysicalHexTile() color = (%d,%d,%d), want ocean color %+v", hex.R, hex.G, hex.B, oceanColor)
	}
}

func TestPoliticalHexTileWater(t *testing.T) {
	mapData := newTerritoryTestMapData(1 /* TERRAIN_OCEAN */, -1, "", "")
	hex, cityColor := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))

	oceanColor := GetPhysicalMapTileColor("TERRAIN_OCEAN")
	if hex.R != oceanColor.R || hex.G != oceanColor.G || hex.B != oceanColor.B {
		t.Errorf("PoliticalHexTile() water color = (%d,%d,%d), want ocean color %+v", hex.R, hex.G, hex.B, oceanColor)
	}
	if cityColor != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("PoliticalHexTile() cityColor = %+v, want white", cityColor)
	}
}

func TestPoliticalHexTileUnownedLand(t *testing.T) {
	mapData := newTerritoryTestMapData(0 /* TERRAIN_GRASS */, -1, "", "")
	hex, _ := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))

	grassColor := GetPhysicalMapTileColor("TERRAIN_GRASS")
	if hex.R != grassColor.R || hex.G != grassColor.G || hex.B != grassColor.B {
		t.Errorf("PoliticalHexTile() unowned color = (%d,%d,%d), want grass color %+v", hex.R, hex.G, hex.B, grassColor)
	}
}

func TestPoliticalHexTileOwnedKnownColor(t *testing.T) {
	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_BLACK", "CIVILIZATION_ROME")
	hex, cityColor := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))

	renderColor := civColorMap["PLAYERCOLOR_BLACK"]
	wantBackground := blendColor(renderColor.OuterColor, color.RGBA{255, 255, 255, 255}, 0.2)
	if hex.R != wantBackground.R || hex.G != wantBackground.G || hex.B != wantBackground.B {
		t.Errorf("PoliticalHexTile() background = (%d,%d,%d), want %+v", hex.R, hex.G, hex.B, wantBackground)
	}
	if cityColor != renderColor.InnerColor {
		t.Errorf("PoliticalHexTile() cityColor = %+v, want inner color %+v", cityColor, renderColor.InnerColor)
	}
}

func TestPoliticalHexTileOwnedUnknownColor(t *testing.T) {
	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_DOES_NOT_EXIST", "CIVILIZATION_ROME")
	hex, _ := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if hex.R != 0 || hex.G != 0 || hex.B != 0 {
		t.Errorf("PoliticalHexTile() unknown-owner color = (%d,%d,%d), want black", hex.R, hex.G, hex.B)
	}
}

func TestTileEntitiesMountain(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 2}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: -1}}},
	}
	entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100), color.RGBA{255, 255, 255, 255})
	if len(entities) != 1 || entities[0].Type != EntityMountain {
		t.Fatalf("TileEntities() = %+v, want a single EntityMountain", entities)
	}
}

func TestTileEntitiesCity(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 0}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: 0}}},
	}
	cityColor := color.RGBA{10, 20, 30, 255}
	entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100), cityColor)
	if len(entities) != 1 || entities[0].Type != EntityCity {
		t.Fatalf("TileEntities() = %+v, want a single EntityCity", entities)
	}
	if entities[0].R != cityColor.R || entities[0].G != cityColor.G || entities[0].B != cityColor.B {
		t.Errorf("TileEntities() city color = (%d,%d,%d), want %+v", entities[0].R, entities[0].G, entities[0].B, cityColor)
	}
}

func TestTileEntitiesMountainAndCity(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 2}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: 0}}},
	}
	entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100), color.RGBA{255, 255, 255, 255})
	if len(entities) != 2 || entities[0].Type != EntityMountain || entities[1].Type != EntityCity {
		t.Fatalf("TileEntities() = %+v, want [EntityMountain, EntityCity] in that order", entities)
	}
}

func TestTileEntitiesNone(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 0}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: -1}}},
	}
	if entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100), color.RGBA{255, 255, 255, 255}); entities != nil {
		t.Errorf("TileEntities() = %v, want nil", entities)
	}
}

func TestTileEntitiesNoImprovementDataIsSafe(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 0}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{},
	}
	if entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100), color.RGBA{255, 255, 255, 255}); entities != nil {
		t.Errorf("TileEntities() with no improvement data = %v, want nil", entities)
	}
}

func TestGetHexEdgeJoinsConsecutiveVertices(t *testing.T) {
	for i := 0; i < 6; i++ {
		edge := getHexEdge(i, 5, 7, 16)
		x1, y1 := hexVertex(i, 5, 7, 16)
		x2, y2 := hexVertex(i+1, 5, 7, 16)
		if edge.X1 != x1 || edge.Y1 != y1 || edge.X2 != x2 || edge.Y2 != y2 {
			t.Errorf("edge %d = %+v, want (%v, %v) to (%v, %v)", i, edge, x1, y1, x2, y2)
		}
	}
}

// The hexagon the canvas draws (rotation pi/2, pointy-top) has the same vertices as hexVertex, so the tile grid,
// the fills and the edge math all describe one shape.
func TestHexagonPolygonMatchesHexVertex(t *testing.T) {
	hex := raster.RegularPolygon(6, 30, 40, 16, math.Pi/2)
	for i, p := range hex {
		found := false
		for j := 0; j < 6; j++ {
			x, y := hexVertex(j, 30, 40, 16)
			found = found || (math.Abs(p.X-x) < 1e-9 && math.Abs(p.Y-y) < 1e-9)
		}
		if !found {
			t.Errorf("polygon vertex %d = %v is not one of hexVertex's six", i, p)
		}
	}
}

// center flips the map file's y-up position against the canvas height, so tiles land in y-down pixels.
func TestCenterFlipsTheMapFilesYAxisAgainstTheCanvasHeight(t *testing.T) {
	const radius, canvasHeight = 16.0, 400
	l := newTileLayout(radius, canvasHeight)
	for _, pos := range []fileio.TilePos{{Row: 0, Col: 0}, {Row: 1, Col: 0}, {Row: 3, Col: 4}, {Row: 8, Col: 11}} {
		x, y := GetImagePosition(pos, radius)
		cx, cy := l.center(pos)
		if cx != x || cy != canvasHeight-y {
			t.Errorf("tile %+v: center (%v, %v), want (%v, %v)", pos, cx, cy, x, canvasHeight-y)
		}
	}
}

// A mountain's apex points up on screen and a city icon reaches further above its center than below it.
func TestMarksPointUp(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	l := newTileLayout(16, 100)

	canvas := NewMockCanvas(100, 100)
	mr.DrawMountain(canvas, l, 10, 20)
	got := canvas.GetOperations()
	if got[0] != "DrawRegularPolygon(3, 10.00, 20.00, 16.00, 0.00)" || got[3] != "DrawRegularPolygon(3, 10.00, 12.00, 8.00, 0.00)" {
		t.Errorf("mountain = %v, want the base at the tile center and the peak 8 above it", got)
	}

	canvas = NewMockCanvas(100, 100)
	mr.DrawCityIcon(canvas, l, 10, 20, mountainBaseColor)
	if got, want := canvas.GetOperations()[0], "DrawRectangle(6.80, 15.20, 8.00, 8.00)"; got != want {
		t.Errorf("city icon = %q, want %q (top 0.3r above the center)", got, want)
	}
}

// A label is centered on its own tile at any row and map height.
func TestLabelIsCenteredOnItsTile(t *testing.T) {
	const radius, rows = 16.0, 7
	l := newTileLayout(radius, 400)
	for row := 0; row < rows; row++ {
		pos := fileio.TilePos{Row: row, Col: 3}
		x, y := l.center(pos)
		labelX, labelY := cityLabelPosition(l, pos, "Rome")
		if labelX != x-12 || labelY != y-radius/2 {
			t.Errorf("row %d: label at (%v, %v), want (%v, %v)", row, labelX, labelY, x-12, y-radius/2)
		}
	}
}

func TestGetImagePosition(t *testing.T) {
	x, y := GetImagePosition(fileio.TilePos{Row: 0, Col: 0}, 16.0)
	angle := math.Pi / 6
	wantX := 16.0 * 1.5
	wantY := 16.0
	if math.Abs(x-wantX) > 1e-9 || math.Abs(y-wantY) > 1e-9 {
		t.Errorf("GetImagePosition(row 0, col 0, 16.0) = (%v, %v), want (%v, %v)", x, y, wantX, wantY)
	}

	// Odd row should shift x by radius*cos(angle)
	xOdd, _ := GetImagePosition(fileio.TilePos{Row: 1, Col: 0}, 16.0)
	wantXOdd := wantX + 16.0*math.Cos(angle)
	if math.Abs(xOdd-wantXOdd) > 1e-9 {
		t.Errorf("GetImagePosition(row 1, col 0, 16.0) x = %v, want %v", xOdd, wantXOdd)
	}
}

func TestRepaintRectCoversTheHexPlusPad(t *testing.T) {
	const radius = 16.0
	l := newTileLayout(radius, 600)
	bounds := image.Rect(0, 0, 2000, 2000)
	rect := l.repaintRect(fileio.TilePos{Row: 3, Col: 4}, bounds)
	x, y := l.center(fileio.TilePos{Row: 3, Col: 4})
	for i := 0; i < 6; i++ {
		vx, vy := hexVertex(i, x, y, radius)
		if vx-repaintRectPad < float64(rect.Min.X) || vx+repaintRectPad > float64(rect.Max.X) ||
			vy-repaintRectPad < float64(rect.Min.Y) || vy+repaintRectPad > float64(rect.Max.Y) {
			t.Errorf("vertex %d (%.2f, %.2f) plus %v of padding is outside %v", i, vx, vy, repaintRectPad, rect)
		}
	}
}

func TestRepaintRectIsClippedToBounds(t *testing.T) {
	l := newTileLayout(16, 32) // tile (0, 0) is centered at (24, 16)
	if got, want := l.repaintRect(fileio.TilePos{Row: 0, Col: 0}, image.Rect(0, 0, 30, 30)), image.Rect(6, 0, 30, 30); got != want {
		t.Errorf("repaintRect clipped = %v, want %v", got, want)
	}
	if got := l.repaintRect(fileio.TilePos{Row: 20, Col: 20}, image.Rect(0, 0, 30, 30)); !got.Empty() {
		t.Errorf("repaintRect of a tile outside the bounds = %v, want empty", got)
	}
}

// The image is wider than tall for a wide map and taller than wide for a tall one, so height and width can't be swapped.
func TestImageSizeFollowsTheMapsHeightAndWidth(t *testing.T) {
	const radius = 16.0
	wideW, wideH := imageSize(fileio.MapSize{Height: 10, Width: 40}, radius)
	tallW, tallH := imageSize(fileio.MapSize{Height: 40, Width: 10}, radius)
	if wideW <= wideH {
		t.Errorf("a 40 wide x 10 high map is %.1f x %.1f, want wider than tall", wideW, wideH)
	}
	if tallW >= tallH {
		t.Errorf("a 10 wide x 40 high map is %.1f x %.1f, want taller than wide", tallW, tallH)
	}
	x, y := GetImagePosition(fileio.TilePos{Row: 10, Col: 40}, radius)
	if wideW != x || wideH != y {
		t.Errorf("imageSize(10, 40) = (%v, %v), want the position just past the far corner (%v, %v)", wideW, wideH, x, y)
	}
}
