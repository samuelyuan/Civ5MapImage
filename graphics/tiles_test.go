package graphics

import (
	"image/color"
	"math"
	"reflect"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// A tile's border color is its civ's border shade.
func TestTileBorderColorIsTheCivsBorderShade(t *testing.T) {
	mapData := newBorderTestMapData(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")
	renderColor := civColorMap["PLAYERCOLOR_BLACK"]
	want := shadesFor(renderColor, false).border
	if got := tileBorderColor(mapData, fileio.TilePos{Row: 0, Col: 0}); got != want {
		t.Errorf("tileBorderColor() = %v, want the civ's border shade %v", got, want)
	}
}

func TestTileBorderColorUnknownColorFallsBackToWhite(t *testing.T) {
	mapData := newBorderTestMapData(0, 1, "PLAYERCOLOR_DOES_NOT_EXIST", "PLAYERCOLOR_ALSO_MISSING")
	if got, want := tileBorderColor(mapData, fileio.TilePos{Row: 0, Col: 0}), (color.RGBA{255, 255, 255, 255}); got != want {
		t.Errorf("tileBorderColor() = %v, want the white fallback %v", got, want)
	}
}

func TestPhysicalHexTile(t *testing.T) {
	const radius = 16.0
	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_OCEAN"},
		MapTiles:    [][]*fileio.Civ5MapTilePhysical{{{TerrainType: 0}}},
	}

	l := newTileLayout(radius, 1)
	hex := PhysicalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, l)

	wantX, wantY := l.center(fileio.TilePos{Row: 0, Col: 0})
	oceanColor := GetPhysicalMapTileColor("TERRAIN_OCEAN")
	if hex.X != wantX || hex.Y != wantY {
		t.Errorf("PhysicalHexTile() position = (%v,%v), want (%v,%v)", hex.X, hex.Y, wantX, wantY)
	}
	if hex.Fill != oceanColor {
		t.Errorf("PhysicalHexTile() color = %+v, want ocean color %+v", hex.Fill, oceanColor)
	}
}

func TestPoliticalHexTileWater(t *testing.T) {
	mapData := newTerritoryTestMapData(1 /* TERRAIN_OCEAN */, -1, "", "")
	hex, cityColor := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))

	oceanColor := GetPhysicalMapTileColor("TERRAIN_OCEAN")
	if hex.Fill != oceanColor {
		t.Errorf("PoliticalHexTile() water color = %+v, want ocean color %+v", hex.Fill, oceanColor)
	}
	if cityColor != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("PoliticalHexTile() cityColor = %+v, want white", cityColor)
	}
}

func TestPoliticalHexTileUnownedLand(t *testing.T) {
	mapData := newTerritoryTestMapData(0 /* TERRAIN_GRASS */, -1, "", "")
	hex, _ := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))

	grassColor := GetPhysicalMapTileColor("TERRAIN_GRASS")
	if hex.Fill != grassColor {
		t.Errorf("PoliticalHexTile() unowned color = %+v, want grass color %+v", hex.Fill, grassColor)
	}
}

func TestPoliticalHexTileOwnedKnownColor(t *testing.T) {
	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_BLACK", "CIVILIZATION_ROME")
	hex, cityColor := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))

	renderColor := civColorMap["PLAYERCOLOR_BLACK"]
	wantBackground := blendColor(renderColor.OuterColor, color.RGBA{255, 255, 255, 255}, 0.2)
	if hex.Fill != wantBackground {
		t.Errorf("PoliticalHexTile() background = %+v, want %+v", hex.Fill, wantBackground)
	}
	if cityColor != renderColor.InnerColor {
		t.Errorf("PoliticalHexTile() cityColor = %+v, want inner color %+v", cityColor, renderColor.InnerColor)
	}
}

func TestPoliticalHexTileOwnedUnknownColor(t *testing.T) {
	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_DOES_NOT_EXIST", "CIVILIZATION_ROME")
	hex, _ := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))
	if hex.Fill != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("PoliticalHexTile() unknown-owner color = %+v, want black", hex.Fill)
	}
}

func TestRiverEdgesForTileAllEdges(t *testing.T) {
	const cx, cy, radius = 100.0, 50.0, 16.0
	// bit0 = east, bit1 = southeast, bit2 = southwest
	edges := RiverEdgesForTile(0b111, cx, cy, newTileLayout(radius, 1))

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
	if edges := RiverEdgesForTile(0, 100, 50, newTileLayout(16, 1)); edges != nil {
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
		edges := RiverEdgesForTile(tt.riverData, cx, cy, newTileLayout(radius, 1))
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
	if segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1)); segments != nil {
		t.Errorf("RoadSegmentsForTile() with RouteType 255 = %v, want nil", segments)
	}
}

func TestRoadSegmentsForTileConnectsToNeighborWithRoute(t *testing.T) {
	const radius = 16.0
	mapData := newRoadGeometryTestMap(0, 0, "") // both tiles have a road

	l := newTileLayout(radius, 1)
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
	if seg.CasingWidth != 3 || (color.RGBA{seg.CasingR, seg.CasingG, seg.CasingB, 255}) != roadCasingColor {
		t.Errorf("road casing = (width=%v, rgb=%d,%d,%d), want a grey casing 3 wide", seg.CasingWidth, seg.CasingR, seg.CasingG, seg.CasingB)
	}
}

func TestRoadSegmentsForTileRailroadStyle(t *testing.T) {
	mapData := newRoadGeometryTestMap(1, 1, "") // railroad
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() = %d segments, want 1", len(segments))
	}
	seg := segments[0]
	if seg.LineWidth != 2.0 || seg.R != 76 || seg.G != 51 || seg.B != 0 {
		t.Errorf("segment style = (width=%v, rgb=%d,%d,%d), want railroad style (2.0, 76,51,0)", seg.LineWidth, seg.R, seg.G, seg.B)
	}
	if seg.CasingWidth != 4 || (color.RGBA{seg.CasingR, seg.CasingG, seg.CasingB, 255}) != railroadCasingColor {
		t.Errorf("railroad casing = (width=%v, rgb=%d,%d,%d), want a warm casing 4 wide", seg.CasingWidth, seg.CasingR, seg.CasingG, seg.CasingB)
	}
}

func TestRoadSegmentsForTileUnknownRouteTypeIsAThinBlackLineWithARoadCasing(t *testing.T) {
	mapData := newRoadGeometryTestMap(2, 2, "")
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() = %d segments, want 1", len(segments))
	}
	seg := segments[0]
	if seg.LineWidth != 1.0 || (color.RGBA{seg.R, seg.G, seg.B, 255}) != unknownRouteColor {
		t.Errorf("unknown route line = (width=%v, rgb=%d,%d,%d), want 1 wide in unknownRouteColor", seg.LineWidth, seg.R, seg.G, seg.B)
	}
	if seg.CasingWidth != 3 || (color.RGBA{seg.CasingR, seg.CasingG, seg.CasingB, 255}) != roadCasingColor {
		t.Errorf("unknown route casing = (width=%v, rgb=%d,%d,%d), want a road casing 3 wide", seg.CasingWidth, seg.CasingR, seg.CasingG, seg.CasingB)
	}
}

func TestRoadSegmentsForTileConnectsToCityWithNoRoute(t *testing.T) {
	// Neighbor has RouteType 255 (no route) but has a city name -- should still connect.
	mapData := newRoadGeometryTestMap(0, 255, "Rome")
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() to city with no route = %d segments, want 1", len(segments))
	}
}

func TestRoadSegmentsForTileSkipsDisconnectedNeighbor(t *testing.T) {
	// Neighbor has no route and no city -- should not connect.
	mapData := newRoadGeometryTestMap(0, 255, "")
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 2}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))
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
	segments := RoadSegmentsForTile(mapData, fileio.MapSize{Height: 1, Width: 1}, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1))
	if len(segments) != 0 {
		t.Errorf("RoadSegmentsForTile() on 1x1 map = %d segments, want 0", len(segments))
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

// Row 0 is the bottom row of the map, so rows count upward on screen: a higher row has a smaller y.
func TestCenterDrawsRowZeroLowestAndShiftsOddRows(t *testing.T) {
	l := newTileLayout(16, 4)
	x0, y0 := l.center(fileio.TilePos{Row: 0, Col: 0})
	if x0 != 24 || y0 != 96 {
		t.Errorf("tile (0, 0) center = (%v, %v), want (24, 96): 1.5 radii from the left, 4 row heights down", x0, y0)
	}
	x1, y1 := l.center(fileio.TilePos{Row: 1, Col: 0})
	if y1 != y0-24 {
		t.Errorf("tile (1, 0) y = %v, want one row height (24) above tile (0, 0)", y1)
	}
	if want := x0 + 16*math.Cos(math.Pi/6); math.Abs(x1-want) > 1e-9 {
		t.Errorf("odd row x = %v, want the even row's %v shifted right by half a tile (%v)", x1, x0, want)
	}
	if x2, _ := l.center(fileio.TilePos{Row: 0, Col: 1}); math.Abs(x2-x0-2*16*math.Cos(math.Pi/6)) > 1e-9 {
		t.Errorf("neighbouring column x = %v, want one tile width (%v) from %v", x2, 2*16*math.Cos(math.Pi/6), x0)
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
}

// Every hex of a map lies inside the image imageSize returns, with no room to spare at the bottom.
func TestImageSizeHoldsEveryHex(t *testing.T) {
	const radius = 16.0
	for _, size := range []fileio.MapSize{{Height: 1, Width: 1}, {Height: 5, Width: 7}, {Height: 6, Width: 4}} {
		w, h := imageSize(size, radius)
		l := layoutForMap(size, radius)
		minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, pos := range allTiles(size) {
			cx, cy := l.center(pos)
			for v := 0; v < 6; v++ {
				x, y := hexVertex(v, cx, cy, radius)
				minX, minY, maxX, maxY = math.Min(minX, x), math.Min(minY, y), math.Max(maxX, x), math.Max(maxY, y)
			}
		}
		if minX < 0 || minY < 0 || maxX > w+1e-9 || maxY > h+1e-9 {
			t.Errorf("%+v map: hexes span (%.1f, %.1f)-(%.1f, %.1f), outside the %.1f x %.1f image", size, minX, minY, maxX, maxY, w, h)
		}
		if math.Abs(maxY-h) > 1e-9 {
			t.Errorf("%+v map: lowest vertex is at y=%v, want the image height %v", size, maxY, h)
		}
	}
}
