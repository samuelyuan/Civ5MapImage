package graphics

import (
	"cmp"
	"image"
	"image/color"
	"math"
	"slices"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
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

// A river must show on the lightest and the darkest ground, unlike the teal it replaced, which nearly vanished on many maps.
func TestMapRiverColorContrastsWithBlackAndWhite(t *testing.T) {
	for name, ground := range map[string]color.RGBA{"white": {255, 255, 255, 255}, "black": {0, 0, 0, 255}} {
		if got := contrastRatio(mapRiverColor, ground); got < 3 {
			t.Errorf("contrast of the river color with %s = %.2f, want at least 3", name, got)
		}
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
	l := layoutForMap(size, tileRadius)
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

// grassRow returns a one-row map of grass whose tiles have the given elevations, with an empty improvement record under each.
func grassRow(elevations ...int) *fileio.Civ5MapData {
	mapData := &fileio.Civ5MapData{TerrainList: []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"}}
	mapData.MapTiles = append(mapData.MapTiles, nil)
	mapData.MapTileImprovements = append(mapData.MapTileImprovements, nil)
	for _, elevation := range elevations {
		mapData.MapTiles[0] = append(mapData.MapTiles[0], &fileio.Civ5MapTilePhysical{Elevation: elevation})
		mapData.MapTileImprovements[0] = append(mapData.MapTileImprovements[0], &fileio.Civ5MapTileImprovement{Owner: -1, CityId: -1, RouteType: 255})
	}
	return mapData
}

// fillAt returns the color a tile's ground shows left of its center, clear of the tile's outline, peak and city marker.
func fillAt(c Canvas, size fileio.MapSize, row, col int) color.RGBA {
	x, y := tileCenter(size, row, col)
	return colorAt(c, int(x)-9, int(y))
}

// runsDown returns the distinct colors met in turn going down column x from y0 to y1, ignoring the background.
func runsDown(c *raster.PalettedCanvas, x, y0, y1 int) []color.RGBA {
	var runs []color.RGBA
	for y := y0; y < y1; y++ {
		if c.IndexAt(x, y) == c.Background() {
			continue
		}
		if col := colorAt(c, x, y); len(runs) == 0 || runs[len(runs)-1] != col {
			runs = append(runs, col)
		}
	}
	return runs
}

func TestPhysicalMapFillsTilesWithTheirTerrainColor(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	for terrain, name := range []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"} {
		canvas := newCanvas(1, 1)
		DrawPhysicalMap(canvas, newTerritoryTestMapData(terrain, -1, "", ""))
		if got, want := fillAt(canvas, size, 0, 0), GetPhysicalMapTileColor(name); got != want {
			t.Errorf("%s tile fill = %v, want %v", name, got, want)
		}
	}
}

func TestPoliticalMapFillsTiles(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	black := shadesFor(civColorMap["PLAYERCOLOR_BLACK"], false).fill
	for _, tt := range []struct {
		name    string
		mapData *fileio.Civ5MapData
		want    color.RGBA
	}{
		{"water", newTerritoryTestMapData(1, -1, "", ""), GetPhysicalMapTileColor("TERRAIN_OCEAN")},
		{"unowned land", newTerritoryTestMapData(0, -1, "", ""), GetPhysicalMapTileColor("TERRAIN_GRASS")},
		{"land owned by a civ", newTerritoryTestMapData(0, 0, "PLAYERCOLOR_BLACK", "CIVILIZATION_ROME"), black},
		{"land owned by a civ with an unrecognized color", newTerritoryTestMapData(0, 0, "PLAYERCOLOR_DOES_NOT_EXIST", "CIVILIZATION_ROME"), color.RGBA{0, 0, 0, 255}},
	} {
		canvas := newCanvas(1, 1)
		DrawPoliticalMap(canvas, tt.mapData)
		if got := fillAt(canvas, size, 0, 0); got != tt.want {
			t.Errorf("%s: tile fill = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A mountain tile's ground is darkened toward the mountain color; other tiles keep theirs.
func TestMountainTilesAreTintedTowardTheMountainGround(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 2}
	canvas := newCanvas(1, 1)
	DrawPhysicalMap(canvas, grassRow(0, 2))

	plain := GetPhysicalMapTileColor("TERRAIN_GRASS")
	if got := fillAt(canvas, size, 0, 0); got != plain {
		t.Errorf("plain tile fill = %v, want %v", got, plain)
	}
	if got, want := fillAt(canvas, size, 0, 1), mountainTileColor(plain); got != want || want == plain {
		t.Errorf("mountain tile fill = %v, want the tinted %v (plain is %v)", got, want, plain)
	}
}

// All fills go down before any outline, so a fill never covers an outline: the outline on the edge two tiles share is the owner's, not the neighbor's fill.
func TestTileOutlinesAreNotCoveredByANeighborsFill(t *testing.T) {
	mapData := grassRow(0, 0)
	mapData.MapTiles[0][1].TerrainType = 1 // ocean beside the grass
	canvas := newCanvas(1, 1)
	DrawPhysicalMap(canvas, mapData)

	x, y := tileCenter(fileio.MapSize{Height: 1, Width: 2}, 0, 0)
	mx, my := edgeMidpoint(x, y, 5) // the east edge, shared with the ocean tile
	if want := tileOutlineColor(GetPhysicalMapTileColor("TERRAIN_GRASS")); !nearPoint(canvas, mx, my, want) {
		t.Errorf("no outline pixel %v near the shared edge (%.1f, %.1f)", want, mx, my)
	}
}

// Each mountain tile gets a peak, and tiles without one get nothing.
func TestDrawMountainsDrawsAPeakOnEveryMountainTile(t *testing.T) {
	mountains := []bool{true, false, false, true}
	size := fileio.MapSize{Height: 1, Width: len(mountains)}
	canvas := newMapCanvas(size)

	drawMountains(canvas, terrainWithMountains([][]bool{mountains}), size)

	for col, isMountain := range mountains {
		x, y := tileCenter(size, 0, col)
		around := image.Rect(int(x)-10, int(y)-14, int(x)+11, int(y)+10)
		if got := countColor(canvas, around, peakOutlineColor) > 0; got != isMountain {
			t.Errorf("tile %d: peak drawn = %v, want %v", col, got, isMountain)
		}
	}
}

// A peak is an outline under a lit and a shaded face that meet at its ridge, with snow on the top of each face.
func TestDrawPeakDrawsOutlinedFacesWithSnow(t *testing.T) {
	canvas := newCanvas(100, 100)

	drawPeak(canvas, peakAt(50, 50, 16, 1)) // ridge at x=50, ground line at y=56.4, top at y=40.4

	for _, tt := range []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{"lit face, left of the ridge", 47, 52, peakLitColor},
		{"shaded face, right of it", 53, 52, peakShadeColor},
		{"snow on the lit face", 49, 43, peakSnowLitColor},
		{"snow on the shaded face", 50, 43, peakSnowShade},
		{"outline below the faces", 50, 56, peakOutlineColor},
	} {
		if got := colorAt(canvas, tt.x, tt.y); got != tt.want {
			t.Errorf("%s: pixel (%d, %d) = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}
}

func TestDrawRiversDrawsEachOwnedEdgePresent(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	x, y := tileCenter(size, 0, 0)
	edges := map[string]int{"southwest": 3, "southeast": 4, "east": 5} // the three edges a tile owns
	for name, tt := range map[string]struct{ bit, edge int }{"east": {1, 5}, "southeast": {2, 4}, "southwest": {4, 3}} {
		mapData := &fileio.Civ5MapData{MapTiles: [][]*fileio.Civ5MapTilePhysical{{{RiverData: tt.bit}}}}
		canvas := newMapCanvas(size)

		drawRivers(canvas, mapData, size)

		for other, edge := range edges {
			mx, my := edgeMidpoint(x, y, edge)
			if got, want := nearPoint(canvas, mx, my, mapRiverColor), edge == tt.edge; got != want {
				t.Errorf("river on the %s edge: river near the %s edge = %v, want %v", name, other, got, want)
			}
		}
	}
}

// Rivers on maps are bluer than the replay's, which stay the thin teal ones.
func TestMapAndReplayRiversUseTheirOwnColors(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	mapData := &fileio.Civ5MapData{MapTiles: [][]*fileio.Civ5MapTilePhysical{{{RiverData: 1}}}}
	x, y := tileCenter(size, 0, 0)
	mx, my := edgeMidpoint(x, y, 5)

	onMap := newMapCanvas(size)
	drawRivers(onMap, mapData, size)
	inReplay := newMapCanvas(size)
	drawRiverTile(inReplay, layoutForMap(size, tileRadius), mapData, fileio.TilePos{}, riverColor, 1.0)

	if !nearPoint(onMap, mx, my, mapRiverColor) || nearPoint(onMap, mx, my, riverColor) {
		t.Error("the map's river should be mapRiverColor and not the replay's")
	}
	if !nearPoint(inReplay, mx, my, riverColor) || nearPoint(inReplay, mx, my, mapRiverColor) {
		t.Error("the replay's river should be riverColor and not the map's")
	}
}

func TestDrawRiversNoRiverData(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	canvas := newMapCanvas(size)

	drawRivers(canvas, &fileio.Civ5MapData{MapTiles: [][]*fileio.Civ5MapTilePhysical{{{RiverData: 0}}}}, size)

	if n := drawnPixels(canvas); n != 0 {
		t.Errorf("a tile with no river drew %d pixels", n)
	}
}

// A route is its line between a lighter casing on each side.
func TestDrawRoadsConnectsAdjacentTilesWithACasedLine(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 2}
	canvas := newMapCanvas(size)

	drawRoads(canvas, newRoadTestMapData(0, 0), size)

	x0, y0 := tileCenter(size, 0, 0)
	x1, _ := tileCenter(size, 0, 1)
	want := []color.RGBA{roadCasingColor, roadColor, roadCasingColor}
	for _, x := range []int{int(x0) + 6, int(x1) - 6} { // the half of the road from each tile
		if got := runsDown(canvas, x, int(y0)-8, int(y0)+8); !slices.Equal(got, want) {
			t.Errorf("column %d, going down: %v, want casing, road, casing %v", x, got, want)
		}
	}
}

// Every casing goes down before any line, so a casing never covers another route's line, and a railroad (wider) is drawn over a road.
func TestDrawRoadsDrawsCasingsThenLinesWithWiderLinesLast(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 2}
	canvas := newMapCanvas(size)

	drawRoads(canvas, newRoadTestMapData(1, 0), size) // tile (0,0) is a railroad, tile (0,1) a road

	x0, y0 := tileCenter(size, 0, 0)
	x1, _ := tileCenter(size, 0, 1)
	if got, want := runsDown(canvas, int(x0)+6, int(y0)-8, int(y0)+8), []color.RGBA{railroadCasingColor, railroadColor, railroadCasingColor}; !slices.Equal(got, want) {
		t.Errorf("railroad half, going down: %v, want %v", got, want)
	}
	if got, want := runsDown(canvas, int(x1)-6, int(y0)-8, int(y0)+8), []color.RGBA{roadCasingColor, roadColor, roadCasingColor}; !slices.Equal(got, want) {
		t.Errorf("road half, going down: %v, want %v", got, want)
	}
	join := int((x0 + x1) / 2) // where the two halves meet, so the railroad's casing and the road's overlap
	for _, y := range []int{int(y0) - 1, int(y0)} {
		if got := colorAt(canvas, join, y); got != railroadColor {
			t.Errorf("pixel (%d, %d) where the routes meet = %v, want the railroad line %v over both casings and the road", join, y, got, railroadColor)
		}
	}
}

func TestDrawRoadsDrawsNothingWithoutARoute(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 2}
	for name, mapData := range map[string]*fileio.Civ5MapData{
		"no route on either tile": newRoadTestMapData(255, 255),
		"no improvement data":     {MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{}},
	} {
		canvas := newMapCanvas(size)
		drawRoads(canvas, mapData, size)
		if n := drawnPixels(canvas); n != 0 {
			t.Errorf("%s: drew %d pixels, want none", name, n)
		}
	}
}

// Each tile paints, in its own color, only pixels of its own hex that lie within borderReach of the other tile.
func TestDrawBordersPaintsEachOwnersSideOfTheBoundary(t *testing.T) {
	mapData := newBorderTestMapData(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")
	size := fileio.MapSize{Height: 1, Width: 2}
	canvas := newMapCanvas(size)

	drawBorders(canvas, mapData, size)

	grid := buildTileGrid(size, tileRadius)
	painted := map[int32]int{} // tile id -> pixels painted for it
	for y := 0; y < grid.height; y++ {
		for x := 0; x < grid.width; x++ {
			if canvas.IndexAt(x, y) == canvas.Background() {
				continue
			}
			id := grid.at(x, y)
			if id < 0 {
				t.Fatalf("pixel (%d, %d) is painted but belongs to no tile", x, y)
			}
			if want := tileBorderColor(mapData, grid.tilePos(id)); colorAt(canvas, x, y) != want {
				t.Fatalf("pixel (%d, %d) of tile %d is %v, want that tile's border color %v", x, y, id, colorAt(canvas, x, y), want)
			}
			near := false
			for _, p := range borderProbes {
				near = near || raster.IsOther(grid.at(x+p.X, y+p.Y), id)
			}
			if !near {
				t.Fatalf("pixel (%d, %d) painted for tile %d is not within %d px of the other tile", x, y, id, borderReach)
			}
			painted[id]++
		}
	}
	if painted[0] == 0 || painted[1] == 0 {
		t.Errorf("painted pixels per tile = %v, want both tiles to have a border", painted)
	}
}

// Territory is a political idea: the physical map shows terrain only, even where two civs own neighboring tiles.
func TestPhysicalMapDrawsNoTerritoryBorders(t *testing.T) {
	mapData := newBorderTestMapData(0, 1, "PLAYERCOLOR_RED", "PLAYERCOLOR_BLUE")
	mapData.TerrainList = []string{"TERRAIN_GRASS"}
	mapData.MapTiles = [][]*fileio.Civ5MapTilePhysical{{{}, {}}}
	canvas := newCanvas(1, 1)

	DrawPhysicalMap(canvas, mapData)

	for owner := range 2 {
		border := tileBorderColor(mapData, fileio.TilePos{Row: 0, Col: owner})
		if n := countColor(canvas, wholeCanvas(canvas), border); n != 0 {
			t.Errorf("the physical map has %d pixels in owner %d's border color %v", n, owner, border)
		}
	}
	canvas = newCanvas(1, 1)
	DrawPoliticalMap(canvas, mapData)
	if n := countColor(canvas, wholeCanvas(canvas), tileBorderColor(mapData, fileio.TilePos{})); n == 0 {
		t.Error("the political map of the same tiles has no border pixels, so this test proves nothing")
	}
}

func TestDrawBordersDrawsNothingWithoutADifferentOwner(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 2}
	for name, mapData := range map[string]*fileio.Civ5MapData{
		"same owner":    newBorderTestMapData(0, 0, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK"),
		"invalid owner": newBorderTestMapData(-1, -1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLACK"),
	} {
		canvas := newMapCanvas(size)
		drawBorders(canvas, mapData, size)
		if n := drawnPixels(canvas); n != 0 {
			t.Errorf("%s: drew %d pixels, want none", name, n)
		}
	}
}

// Every city name has a halo: its text repeated one pixel up, down, left and right in the halo color, so each text pixel is ringed by halo or more text.
func TestDrawHaloedLabelRingsTheLabelInItsHalo(t *testing.T) {
	canvas := newCanvas(100, 40)
	text := color.RGBA{30, 30, 30, 255} // dark text, so a light halo

	drawHaloedLabel(canvas, ColoredText{Text: "Rome", X: 10, Y: 20, R: text.R, G: text.G, B: text.B})

	textPixels := 0
	for y := 1; y < 39; y++ {
		for x := 1; x < 99; x++ {
			if colorAt(canvas, x, y) != text {
				continue
			}
			textPixels++
			for _, n := range []image.Point{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				if got := colorAt(canvas, n.X, n.Y); got != text && got != labelHaloLight {
					t.Fatalf("pixel (%d, %d) beside the text pixel (%d, %d) is %v, want halo %v or text", n.X, n.Y, x, y, got, labelHaloLight)
				}
			}
		}
	}
	if textPixels == 0 {
		t.Error("no text was drawn")
	}
}

// physicalNames draws the city names of the physical map's style for mapData on canvas.
func physicalNames(canvas Canvas, mapData *fileio.Civ5MapData, size fileio.MapSize) {
	l := layoutForMap(size, tileRadius)
	drawCityNames(canvas, mapData, size, l, physicalStyle(mapData, l).label)
}

// politicalNames draws the city names of the political map's style for mapData on canvas.
func politicalNames(canvas Canvas, mapData *fileio.Civ5MapData, size fileio.MapSize) {
	l := layoutForMap(size, tileRadius)
	drawCityNames(canvas, mapData, size, l, politicalStyle(mapData, l).label)
}

var white = color.RGBA{255, 255, 255, 255}

// wholeCanvas returns the rectangle of all of c.
func wholeCanvas(c Canvas) image.Rectangle {
	return image.Rect(0, 0, c.Image().Bounds().Dx(), c.Image().Bounds().Dy())
}

// Physical labels are white over a dark halo even on a civ's city; only the political map colors them by civ.
func TestPhysicalCityNamesAreWhiteOverADarkHaloWhoeverOwnsTheCity(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	owned := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityName: "Rome", Owner: 0, CityId: -1}}},
		Civ5PlayerData:      []*fileio.Civ5PlayerData{{Index: 0, CivType: "CIVILIZATION_ROME", TeamColor: "PLAYERCOLOR_RED"}},
		CityOwnerIndexMap:   map[int]int{0: 0},
	}
	civText := PoliticalCityNameLabel(owned, fileio.TilePos{}, layoutForMap(size, tileRadius))
	if civText.R == 255 && civText.G == 255 && civText.B == 255 {
		t.Fatal("test setup: the civ's political label should not be white")
	}
	canvas := newMapCanvas(size)

	physicalNames(canvas, owned, size)

	if countColor(canvas, wholeCanvas(canvas), white) == 0 || countColor(canvas, wholeCanvas(canvas), labelHaloDark) == 0 {
		t.Error("want white text over a dark halo")
	}
	if n := countColor(canvas, wholeCanvas(canvas), color.RGBA{civText.R, civText.G, civText.B, 255}); n != 0 {
		t.Errorf("%d pixels in the civ's label color, want none on the physical map", n)
	}
}

// Tiles with no city draw nothing at all: no halo, no empty label.
func TestCityNamesAreDrawnOnlyForCities(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 3}
	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{}, {CityName: "Rome"}, {}}}}
	canvas := newMapCanvas(size)

	physicalNames(canvas, mapData, size)

	label := PhysicalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 1}, layoutForMap(size, tileRadius))
	if inside, all := countColor(canvas, labelBox(label), white), countColor(canvas, wholeCanvas(canvas), white); inside == 0 || inside != all {
		t.Errorf("%d white pixels in Rome's label box, %d on the whole canvas, want some, and all of them in the box", inside, all)
	}
}

func TestCityNamesDrawNothingWithoutImprovementData(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	canvas := newMapCanvas(size)

	physicalNames(canvas, &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{}}, size)

	if n := drawnPixels(canvas); n != 0 {
		t.Errorf("drew %d pixels, want none", n)
	}
}

func TestPoliticalCityNamesUseTheCivColorOrFallBackToWhite(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	known := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityName: "Rome", Owner: 0, CityId: -1}}},
		Civ5PlayerData:      []*fileio.Civ5PlayerData{{Index: 0, CivType: "CIVILIZATION_ROME", TeamColor: "PLAYERCOLOR_BLACK"}},
		CityOwnerIndexMap:   map[int]int{0: 0},
	}
	unowned := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityName: "Rome", Owner: -1, CityId: -1}}}}

	for name, tt := range map[string]struct {
		mapData *fileio.Civ5MapData
		text    color.RGBA
	}{
		"a civ's color": {known, markerColor(civColorMap["PLAYERCOLOR_BLACK"].InnerColor)},
		"unowned":       {unowned, white},
	} {
		canvas := newMapCanvas(size)
		politicalNames(canvas, tt.mapData, size)
		if countColor(canvas, wholeCanvas(canvas), tt.text) == 0 {
			t.Errorf("%s: no text pixel in %v", name, tt.text)
		}
		if halo := labelHaloColor(tt.text); countColor(canvas, wholeCanvas(canvas), halo) == 0 {
			t.Errorf("%s: no halo pixel in %v", name, halo)
		}
	}
}

// A halo goes on every label, whatever the ground: each of two cities gets its own.
func TestCityNamesHaloEveryLabel(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 2}
	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityName: "Rome", Owner: -1, CityId: -1}, {CityName: "Kyiv", Owner: -1, CityId: -1}}}}
	canvas := newMapCanvas(size)

	politicalNames(canvas, mapData, size)

	l := layoutForMap(size, tileRadius)
	for col, name := range []string{"Rome", "Kyiv"} {
		box := labelBox(PoliticalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: col}, l))
		if countColor(canvas, box, white) == 0 || countColor(canvas, box, labelHaloDark) == 0 {
			t.Errorf("%s has no white text over a dark halo in %v", name, box)
		}
	}
}

// A city is a circle on a slightly larger one in the halo color, both at the tile's center.
func TestDrawCityMarkersDrawsAnOutlinedCircleOnEveryCity(t *testing.T) {
	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: 0}, {CityId: -1}, {CityId: 1}}}}
	size := fileio.MapSize{Height: 1, Width: 3}
	canvas := newMapCanvas(size)
	cityColor := color.RGBA{200, 100, 50, 255}

	drawCityMarkers(canvas, mapData, size, func(fileio.TilePos) color.RGBA { return cityColor })

	fill := markerColor(cityColor)
	halo := labelHaloColor(fill)
	for col, hasCity := range []bool{true, false, true} {
		x, y := tileCenter(size, 0, col)
		around := image.Rect(int(x)-7, int(y)-7, int(x)+8, int(y)+8)
		if got := colorAt(canvas, int(x), int(y)) == fill; got != hasCity {
			t.Errorf("tile %d: circle at its center = %v, want %v", col, got, hasCity)
		}
		if got := countColor(canvas, around, halo) > 0; got != hasCity {
			t.Errorf("tile %d: halo ring = %v, want %v", col, got, hasCity)
		}
	}
}

// A marker's outline is dark around a light circle and light around a dark one, like the labels' halos.
func TestDrawCityMarkersOutlineContrastsWithTheCircle(t *testing.T) {
	mapData := &fileio.Civ5MapData{MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: 0}}}}
	size := fileio.MapSize{Height: 1, Width: 1}
	for name, tt := range map[string]struct{ city, outline color.RGBA }{
		"white": {white, labelHaloDark},
		"black": {color.RGBA{0, 0, 0, 255}, labelHaloLight},
	} {
		canvas := newMapCanvas(size)
		drawCityMarkers(canvas, mapData, size, func(fileio.TilePos) color.RGBA { return tt.city })
		x, y := tileCenter(size, 0, 0)
		if countColor(canvas, image.Rect(int(x)-7, int(y)-7, int(x)+8, int(y)+8), tt.outline) == 0 {
			t.Errorf("%s city: no outline pixel in %v", name, tt.outline)
		}
	}
}

// On the physical map every city marker is white over a dark outline, whoever owns the city; on the political map it takes the civ's color.
func TestPhysicalMapCityMarkersAreWhiteAndPoliticalOnesTakeTheCivColor(t *testing.T) {
	mapData := newTerritoryTestMapData(0, 0, "PLAYERCOLOR_RED", "CIVILIZATION_ROME")
	mapData.MapTileImprovements[0][0].CityId = 0
	size := fileio.MapSize{Height: 1, Width: 1}
	x, y := tileCenter(size, 0, 0)
	around := image.Rect(int(x)-7, int(y)-7, int(x)+8, int(y)+8)

	physical := newCanvas(1, 1)
	DrawPhysicalMap(physical, mapData)
	if got := colorAt(physical, int(x), int(y)); got != white {
		t.Errorf("physical marker = %v, want white", got)
	}
	if countColor(physical, around, labelHaloDark) == 0 {
		t.Error("physical marker has no dark outline")
	}

	political := newCanvas(1, 1)
	DrawPoliticalMap(political, mapData)
	if want := markerColor(civColorMap["PLAYERCOLOR_RED"].InnerColor); colorAt(political, int(x), int(y)) == white || countColor(political, around, want) == 0 {
		t.Errorf("political marker at (%d, %d) is %v, want the civ's marker color %v", int(x), int(y), colorAt(political, int(x), int(y)), want)
	}
}

// The routes all lead to a city's center, so markers go on after every route and nothing draws over one.
func TestMapsDrawCityMarkersOverTheRoutes(t *testing.T) {
	newMap := func() *fileio.Civ5MapData {
		mapData := newTerritoryTestMapData(0, -1, "", "")
		mapData.MapTiles = [][]*fileio.Civ5MapTilePhysical{{{TerrainType: 0}, {TerrainType: 0}}}
		mapData.MapTileImprovements = [][]*fileio.Civ5MapTileImprovement{
			{{CityId: 0, CityName: "Rome", Owner: -1, RouteType: 1}, {CityId: -1, Owner: -1, RouteType: 1}},
		}
		return mapData
	}
	size := fileio.MapSize{Height: 1, Width: 2}
	for name, drawMap := range map[string]func(Canvas, *fileio.Civ5MapData) image.Image{
		"political": DrawPoliticalMap,
		"physical":  DrawPhysicalMap,
	} {
		canvas := newCanvas(1, 1)
		drawMap(canvas, newMap())

		x0, y0 := tileCenter(size, 0, 0)
		x1, _ := tileCenter(size, 0, 1)
		if got := colorAt(canvas, int(x0), int(y0)); got == railroadColor || got == railroadCasingColor {
			t.Errorf("%s map: the city's center is %v, a route color, want the marker on top", name, got)
		}
		if !nearPoint(canvas, (x0+x1)/2, y0, railroadColor) {
			t.Errorf("%s map: no railroad between the two tiles", name)
		}
	}
}

// Every peak sits over the borders and rivers along the tile edges, and the routes cross the peaks: on the edge two mountain tiles share, a pair's small peak covers the border and the river, and the railroad covers the peak.
func TestMountainsCoverBordersAndRiversAndRoutesCrossThem(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 2}
	political := newBorderTestMapData(0, 1, "PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE")
	political.TerrainList = []string{"TERRAIN_GRASS"}
	political.MapTiles = [][]*fileio.Civ5MapTilePhysical{{{Elevation: 2, RiverData: 1}, {Elevation: 2}}}
	physical := terrainWithMountains([][]bool{{true, true}})
	physical.MapTiles[0][0].RiverData = 1
	physical.MapTileImprovements = [][]*fileio.Civ5MapTileImprovement{{{CityId: -1, Owner: -1}, {CityId: -1, Owner: -1}}}
	for _, mapData := range []*fileio.Civ5MapData{political, physical} {
		for _, tile := range mapData.MapTileImprovements[0] {
			tile.RouteType = 1
		}
	}
	for name, tt := range map[string]struct {
		mapData *fileio.Civ5MapData
		draw    func(Canvas, *fileio.Civ5MapData) image.Image
	}{
		"political": {political, DrawPoliticalMap},
		"physical":  {physical, DrawPhysicalMap},
	} {
		canvas := newCanvas(1, 1)
		tt.draw(canvas, tt.mapData)

		x0, y0 := tileCenter(size, 0, 0)
		x1, _ := tileCenter(size, 0, 1)
		mid := (x0 + x1) / 2 // the shared edge, where the river runs, the border lies and the pair's peak stands
		pair := peakAt(mid, y0, tileRadius, gapPeakScale)
		lowOnPeak := int(pair.base) - 2
		// The border band (2 px each side of the edge) spans x = mid-2..mid+2 and the river (1.5 px wide) the two pixels either side of mid, so
		// pixels left of the ridge are the lit face and pixels right of it the shaded face only if the peak went on after both.
		for _, x := range []int{int(mid) - 1, int(mid)} {
			if got := colorAt(canvas, x, lowOnPeak); got != peakLitColor {
				t.Errorf("%s map: pixel (%d, %d) left of the ridge = %v, want the lit face %v over the border and river", name, x, lowOnPeak, got, peakLitColor)
			}
		}
		for _, x := range []int{int(mid) + 1, int(mid) + 2} {
			if got := colorAt(canvas, x, lowOnPeak); got != peakShadeColor {
				t.Errorf("%s map: pixel (%d, %d) right of the ridge = %v, want the shaded face %v over the border and river", name, x, lowOnPeak, got, peakShadeColor)
			}
		}
		if got := colorAt(canvas, int(mid)-1, int(y0)-1); got != railroadColor {
			t.Errorf("%s map: the railroad across the peak = %v, want %v", name, got, railroadColor)
		}
	}
}

func TestMapsResizeTheCanvasToTheMap(t *testing.T) {
	size := fileio.MapSize{Height: 1, Width: 1}
	w, h := imageSize(size, tileRadius)
	for name, drawMap := range map[string]func(Canvas, *fileio.Civ5MapData) image.Image{
		"political": DrawPoliticalMap,
		"physical":  DrawPhysicalMap,
	} {
		canvas := newCanvas(1, 1)
		if img := drawMap(canvas, newFullMapDataForRender()); img.Bounds() != image.Rect(0, 0, int(w), int(h)) {
			t.Errorf("%s map: image bounds = %v, want %v", name, img.Bounds(), image.Rect(0, 0, int(w), int(h)))
		}
	}
}
