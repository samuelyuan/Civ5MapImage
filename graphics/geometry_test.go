package graphics

import (
	"reflect"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

func TestInvertedRow(t *testing.T) {
	tests := []struct {
		mapHeight, row, want int
	}{
		{10, 0, 10},
		{10, 10, 0},
		{10, 4, 6},
		{1, 0, 1},
		{1, 1, 0},
	}
	for _, tt := range tests {
		if got := InvertedRow(tt.mapHeight, tt.row); got != tt.want {
			t.Errorf("InvertedRow(%d, %d) = %d, want %d", tt.mapHeight, tt.row, got, tt.want)
		}
	}
}

func TestRiverEdgesForTileAllEdges(t *testing.T) {
	const cx, cy, radius = 100.0, 50.0, 16.0
	// bit0 = east, bit1 = southeast, bit2 = southwest
	edges := RiverEdgesForTile(0b111, cx, cy, radius)

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
	if edges := RiverEdgesForTile(0, 100, 50, 16); edges != nil {
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
		edges := RiverEdgesForTile(tt.riverData, cx, cy, radius)
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
	if segments := RoadSegmentsForTile(mapData, 1, 2, 0, 0, 16.0); segments != nil {
		t.Errorf("RoadSegmentsForTile() with RouteType 255 = %v, want nil", segments)
	}
}

func TestRoadSegmentsForTileConnectsToNeighborWithRoute(t *testing.T) {
	const radius = 16.0
	mapData := newRoadGeometryTestMap(0, 0, "") // both tiles have a road

	segments := RoadSegmentsForTile(mapData, 1, 2, 0, 0, radius)
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() = %d segments, want 1: %v", len(segments), segments)
	}

	seg := segments[0]
	x1, y1 := fileio.GetImagePosition(0, 0, radius)
	x2, y2 := fileio.GetImagePosition(0, 1, radius)
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
	segments := RoadSegmentsForTile(mapData, 1, 2, 0, 0, 16.0)
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
	segments := RoadSegmentsForTile(mapData, 1, 2, 0, 0, 16.0)
	if len(segments) != 1 {
		t.Fatalf("RoadSegmentsForTile() to city with no route = %d segments, want 1", len(segments))
	}
}

func TestRoadSegmentsForTileSkipsDisconnectedNeighbor(t *testing.T) {
	// Neighbor has no route and no city -- should not connect.
	mapData := newRoadGeometryTestMap(0, 255, "")
	segments := RoadSegmentsForTile(mapData, 1, 2, 0, 0, 16.0)
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
	segments := RoadSegmentsForTile(mapData, 1, 1, 0, 0, 16.0)
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
	if segments := BorderSegmentsForTile(mapData, 1, 2, 0, 0, 16.0); segments != nil {
		t.Errorf("BorderSegmentsForTile() with invalid owner = %v, want nil", segments)
	}
}

func TestBorderSegmentsForTileSameOwnerNoBorder(t *testing.T) {
	mapData := newBorderGeometryTestMap(0, 0, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK")
	if segments := BorderSegmentsForTile(mapData, 1, 2, 0, 0, 16.0); len(segments) != 0 {
		t.Errorf("BorderSegmentsForTile() with same owner = %v, want empty", segments)
	}
}

func TestBorderSegmentsForTileDifferentOwnerDrawsBorder(t *testing.T) {
	const radius = 16.0
	mapData := newBorderGeometryTestMap(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")

	segments := BorderSegmentsForTile(mapData, 1, 2, 0, 0, radius)
	if len(segments) != 1 {
		t.Fatalf("BorderSegmentsForTile() = %d segments, want 1: %v", len(segments), segments)
	}

	x1, y1 := fileio.GetImagePosition(0, 0, radius)
	renderColor := civColorMap["PLAYERCOLOR_BLACK"]
	wantLine := getHexEdge(5, x1, y1, radius-1) // neighbor (1,0) is edge index 5 for an even row
	if segments[0].Line != wantLine {
		t.Errorf("segment line = %+v, want %+v", segments[0].Line, wantLine)
	}
	if segments[0].R != renderColor.InnerColor.R || segments[0].G != renderColor.InnerColor.G || segments[0].B != renderColor.InnerColor.B {
		t.Errorf("segment color = (%d,%d,%d), want inner color %+v", segments[0].R, segments[0].G, segments[0].B, renderColor.InnerColor)
	}
}

func TestBorderSegmentsForTileUnknownColorFallsBackToWhite(t *testing.T) {
	mapData := newBorderGeometryTestMap(0, 1, "PLAYERCOLOR_DOES_NOT_EXIST", "PLAYERCOLOR_ALSO_MISSING")
	segments := BorderSegmentsForTile(mapData, 1, 2, 0, 0, 16.0)
	if len(segments) != 1 {
		t.Fatalf("BorderSegmentsForTile() = %d segments, want 1", len(segments))
	}
	if segments[0].R != 255 || segments[0].G != 255 || segments[0].B != 255 {
		t.Errorf("segment color = (%d,%d,%d), want white fallback (255,255,255)", segments[0].R, segments[0].G, segments[0].B)
	}
}
