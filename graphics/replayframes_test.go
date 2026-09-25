package graphics

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// With no anti-aliasing, repainting only the dirty tiles must give exactly a full redraw, every turn.
func TestPalettedCanvasIncrementalRepaintMatchesFullRedrawExactly(t *testing.T) {
	mapData := buildBenchMapData(16, 16, 4, 3)
	turnsEvents := [][]fileio.Civ5ReplayEvent{
		{{Turn: 1, TypeId: fileio.ReplayEventTilesClaimed, CivId: 1, Tiles: []fileio.Civ5ReplayEventTile{{X: 3, Y: 3}}}},
		{{Turn: 2, TypeId: fileio.ReplayEventCityFounded, Text: "Testopolis is founded.", Tiles: []fileio.Civ5ReplayEventTile{{X: 8, Y: 8}}}},
		{{Turn: 3, TypeId: fileio.ReplayEventTilesClaimed, CivId: 2, Tiles: []fileio.Civ5ReplayEventTile{{X: 8, Y: 7}}}},
		{{Turn: 4, TypeId: fileio.ReplayEventTilesClaimed, CivId: 0, Tiles: []fileio.Civ5ReplayEventTile{{X: 1, Y: 1}, {X: 14, Y: 14}}}},
		{{Turn: 5, TypeId: fileio.ReplayEventTilesClaimed, CivId: 3, Tiles: []fileio.Civ5ReplayEventTile{{X: 0, Y: 0}, {X: 15, Y: 15}, {X: 7, Y: 8}}}},
	}
	palette := replayPalette(mapData)
	grid := gridFor(mapData)
	incremental := raster.NewPalettedCanvas(800, 600, palette)

	nextCityId := 0
	var tracker *tileTracker
	for turnIndex, events := range turnsEvents {
		for _, event := range events {
			nextCityId = fileio.ApplyReplayEvent(mapData, event, nextCityId)
		}
		if turnIndex == 0 {
			tracker = newTileTracker(mapData)
			drawPoliticalMapTileMajor(incremental, mapData, grid)
		} else {
			redrawDirtyTiles(incremental, mapData, grid, tracker.takeChanges())
		}

		full := raster.NewPalettedCanvas(800, 600, palette)
		drawPoliticalMapTileMajor(full, mapData, grid)
		if incremental.Image().Bounds() != full.Image().Bounds() {
			t.Fatalf("turn %d: bounds %v != %v", turnIndex, incremental.Image().Bounds(), full.Image().Bounds())
		}
		diff := 0
		var first image.Point
		for y := 0; y < full.Image().Bounds().Dy(); y++ {
			for x := 0; x < full.Image().Bounds().Dx(); x++ {
				if incremental.IndexAt(x, y) != full.IndexAt(x, y) {
					if diff == 0 {
						first = image.Pt(x, y)
					}
					diff++
				}
			}
		}
		if diff != 0 {
			t.Errorf("turn %d: %d pixels differ from a full redraw, first at %v", turnIndex, diff, first)
		}
	}
}

// TestRedrawDirtyTilesMatchesFullRedrawAfterMutation checks that repainting only the dirty tiles matches a full redraw exactly.
func TestRedrawDirtyTilesMatchesFullRedrawAfterMutation(t *testing.T) {
	mapData := buildBenchMapData(12, 12, 4, 7)
	grid := gridFor(mapData)

	canvas := newBenchCanvas(mapData)
	drawPoliticalMapTileMajor(canvas, mapData, grid)

	// Simulate a TilesClaimed event flipping ownership of a small cluster of tiles.
	tracker := newTileTracker(mapData)
	for _, rc := range []fileio.TilePos{{Row: 5, Col: 5}, {Row: 5, Col: 6}, {Row: 6, Col: 5}, {Row: 6, Col: 6}} {
		tile := mapData.MapTileImprovements[rc.Row][rc.Col]
		tile.Owner = (tile.Owner + 1) % 4
	}
	redrawDirtyTiles(canvas, mapData, grid, tracker.takeChanges())

	fresh := newBenchCanvas(mapData)
	drawPoliticalMapTileMajor(fresh, mapData, grid)

	count, first := canvasDiffs(canvas, fresh)
	if count < 0 {
		t.Fatalf("bounds differ: incremental %v, fresh %v", canvas.Image().Bounds(), fresh.Image().Bounds())
	}
	if count != 0 {
		t.Errorf("RedrawDirtyTiles() differs from a full redraw in %d pixels, first at %v", count, first)
	}
}

// TestRedrawDirtyTilesMatchesFullRedrawWithCrowdedLabels repaints random tiles on a map packed with overlapping labels and requires an exact match.
func TestRedrawDirtyTilesMatchesFullRedrawWithCrowdedLabels(t *testing.T) {
	const h, w, civs = 14, 14, 4
	rng := rand.New(rand.NewSource(11))

	mapData := buildBenchMapData(h, w, civs, 5)
	id := 0
	for row := 1; row < h; row += 2 {
		for col := 0; col < w; col += 2 {
			tile := mapData.MapTileImprovements[row][col]
			tile.CityId = id
			tile.CityName = fmt.Sprintf("Longtown Number %d", id)
			id++
		}
	}

	grid := gridFor(mapData)
	canvas := newBenchCanvas(mapData)
	drawPoliticalMapTileMajor(canvas, mapData, grid)

	for round := 0; round < 8; round++ {
		tracker := newTileTracker(mapData)
		for i := 0; i < 3; i++ {
			rc := fileio.TilePos{Row: rng.Intn(h), Col: rng.Intn(w)}
			tile := mapData.MapTileImprovements[rc.Row][rc.Col]
			tile.Owner = (tile.Owner + 1 + rng.Intn(civs-1)) % civs
		}
		redrawDirtyTiles(canvas, mapData, grid, tracker.takeChanges())

		fresh := newBenchCanvas(mapData)
		drawPoliticalMapTileMajor(fresh, mapData, grid)
		if count, first := canvasDiffs(canvas, fresh); count != 0 {
			t.Fatalf("round %d: %d pixels differ from a full redraw, first at %v", round, count, first)
		}
	}
}

// TestRedrawDirtyTilesDoesNotTouchUnrelatedPixels checks one dirty tile's repaint leaves far-away pixels untouched (a PasteRegion bounds bug).
func TestRedrawDirtyTilesDoesNotTouchUnrelatedPixels(t *testing.T) {
	mapData := buildBenchMapData(20, 20, 4, 7)
	grid := gridFor(mapData)

	canvas := newBenchCanvas(mapData)
	drawPoliticalMapTileMajor(canvas, mapData, grid)
	bounds := canvas.Image().Bounds()
	before := canvas.Snapshot(bounds)

	mapSize := mapData.Size()

	tracker := newTileTracker(mapData)
	tile := mapData.MapTileImprovements[10][10]
	tile.Owner = (tile.Owner + 1) % 4

	redrawDirtyTiles(canvas, mapData, grid, tracker.takeChanges())

	// Tile (1, 1) is 9+ tiles from (10, 10).
	x, y := layoutForMap(mapSize, tileRadius).center(fileio.TilePos{Row: 1, Col: 1})
	cx, cy := int(x), int(y)

	checkRect := image.Rect(cx-8, cy-8, cx+8, cy+8).Intersect(bounds)
	for py := checkRect.Min.Y; py < checkRect.Max.Y; py++ {
		for px := checkRect.Min.X; px < checkRect.Max.X; px++ {
			if was, now := before.ColorIndexAt(px, py), canvas.IndexAt(px, py); was != now {
				t.Fatalf("pixel (%d, %d) near untouched tile (1, 1) changed after RedrawDirtyTiles on tile (10, 10): index %d -> %d",
					px, py, was, now)
			}
		}
	}
}

// --- benchmarks ---
//
// "Medium" (80x52) mirrors a Civ5 Medium map for fast end-to-end runs; "Huge" (128x80) isolates single-call costs.
// Run: go test ./graphics/... -run '^$' -bench . -benchtime=5x

func BenchmarkFullTileMajorRedraw_Huge(b *testing.B) {
	mapData := buildBenchMapData(80, 128, 8, 1)
	grid := gridFor(mapData)
	canvas := newBenchCanvas(mapData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drawPoliticalMapTileMajor(canvas, mapData, grid)
	}
}

// BenchmarkNewSiblingAllocation_Huge isolates allocating and inverting a full-map-sized sibling (staging) canvas.
func BenchmarkNewSiblingAllocation_Huge(b *testing.B) {
	mapHeight, mapWidth := 80, 128
	base := newBenchCanvas(buildBenchMapData(mapHeight, mapWidth, 8, 1))
	radius := tileRadius
	maxImageWidth, maxImageHeight := imageSize(fileio.MapSize{Height: mapHeight, Width: mapWidth}, radius)
	w, h := int(maxImageWidth), int(maxImageHeight)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sibling := base.NewSibling(w, h)
		_ = sibling.Image()
	}
}

func benchmarkRedrawDirtyTiles(b *testing.B, mapData *fileio.Civ5MapData, dirtyCount int) {
	grid := gridFor(mapData)
	canvas := newBenchCanvas(mapData)
	drawPoliticalMapTileMajor(canvas, mapData, grid) // establish the base frame once

	mapSize := mapData.Size()

	rng := rand.New(rand.NewSource(2))
	dirty := tileChanges{}
	for len(dirty) < dirtyCount {
		rc := fileio.TilePos{Row: rng.Intn(mapSize.Height), Col: rng.Intn(mapSize.Width)}
		dirty[rc] = *mapData.MapTileImprovements[rc.Row][rc.Col]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		redrawDirtyTiles(canvas, mapData, grid, dirty)
	}
}

// benchmarkRedrawDirtyTilesClustered is benchmarkRedrawDirtyTiles with dirty tiles within clusterRadius of one center.
func benchmarkRedrawDirtyTilesClustered(b *testing.B, mapData *fileio.Civ5MapData, dirtyCount, clusterRadius int) {
	grid := gridFor(mapData)
	canvas := newBenchCanvas(mapData)
	drawPoliticalMapTileMajor(canvas, mapData, grid)

	mapSize := mapData.Size()

	rng := rand.New(rand.NewSource(2))
	centerRow, centerCol := rng.Intn(mapSize.Height), rng.Intn(mapSize.Width)
	dirty := tileChanges{}
	for len(dirty) < dirtyCount {
		row := clampInt(centerRow+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapSize.Height-1)
		col := clampInt(centerCol+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapSize.Width-1)
		dirty[fileio.TilePos{Row: row, Col: col}] = *mapData.MapTileImprovements[row][col]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		redrawDirtyTiles(canvas, mapData, grid, dirty)
	}
}

func BenchmarkRedrawDirtyTiles_5_Huge(b *testing.B) {
	benchmarkRedrawDirtyTiles(b, buildBenchMapData(80, 128, 8, 1), 5)
}

func BenchmarkRedrawDirtyTiles_20_Huge(b *testing.B) {
	benchmarkRedrawDirtyTiles(b, buildBenchMapData(80, 128, 8, 1), 20)
}

func BenchmarkRedrawDirtyTiles_100_Huge(b *testing.B) {
	benchmarkRedrawDirtyTiles(b, buildBenchMapData(80, 128, 8, 1), 100)
}

func BenchmarkRedrawDirtyTiles_20_Clustered_Huge(b *testing.B) {
	benchmarkRedrawDirtyTilesClustered(b, buildBenchMapData(80, 128, 8, 1), 20, 5)
}

// A label wider than the repainted tiles must still be fully erased, drawn or recolored when its city changes.
func TestRedrawDirtyTilesMatchesFullRedrawWhenLongLabelsChange(t *testing.T) {
	const size = 16
	tile := func(x, y int) []fileio.Civ5ReplayEventTile { return []fileio.Civ5ReplayEventTile{{X: x, Y: y}} }
	scenarios := []struct {
		name   string
		before map[fileio.TilePos]string // city names on the map before the event
		event  fileio.Civ5ReplayEvent
	}{
		{"razed", map[fileio.TilePos]string{{Row: 8, Col: 8}: "%s"},
			fileio.Civ5ReplayEvent{TypeId: fileio.ReplayEventTilesRazed, Tiles: tile(8, 8)}},
		{"founded", nil,
			fileio.Civ5ReplayEvent{TypeId: fileio.ReplayEventCityFounded, Text: "%s is founded.", Tiles: tile(8, 8)}},
		{"recolored", map[fileio.TilePos]string{{Row: 8, Col: 8}: "%s"},
			fileio.Civ5ReplayEvent{TypeId: fileio.ReplayEventTilesClaimed, CivId: 3, Tiles: tile(8, 8)}},
		{"razed beside an overlapping label", map[fileio.TilePos]string{{Row: 8, Col: 8}: "%s", {Row: 8, Col: 9}: "Neighbor of %s"},
			fileio.Civ5ReplayEvent{TypeId: fileio.ReplayEventTilesRazed, Tiles: tile(8, 8)}},
	}
	for _, sc := range scenarios {
		for _, name := range []string{"Roma", "Constantinople", "Saint Petersburg Upon Neva"} {
			mapData := buildBenchMapData(size, size, 4, 3)
			for rc, pattern := range sc.before {
				record := mapData.MapTileImprovements[rc.Row][rc.Col]
				record.CityName, record.CityId = fmt.Sprintf(pattern, name), 77
			}
			event := sc.event
			event.Text = strings.Replace(event.Text, "%s", name, 1)

			grid := gridFor(mapData)
			canvas := newBenchCanvas(mapData)
			drawPoliticalMapTileMajor(canvas, mapData, grid)
			tracker := newTileTracker(mapData)
			fileio.ApplyReplayEvent(mapData, event, 100)
			redrawDirtyTiles(canvas, mapData, grid, tracker.takeChanges())

			full := newBenchCanvas(mapData)
			drawPoliticalMapTileMajor(full, mapData, grid)
			if diffs, first := canvasDiffs(canvas, full); diffs != 0 {
				t.Errorf("%s, %q (%d chars): %d pixels differ from a full redraw, first at %v", sc.name, name, len(name), diffs, first)
			}
		}
	}
}

// Random founding, razing, claiming and transferring of cities with names of all lengths must never make a repaint differ from a full redraw.
func TestRedrawDirtyTilesMatchesFullRedrawOverRandomCityEvents(t *testing.T) {
	const size, turns = 14, 12
	names := []string{"Rome", "Kyiv", "Constantinople", "Saint Petersburg", "Ulaanbaatar", "Minneapolis-Saint Paul Upon The Great River", "Al"}
	for seed := int64(1); seed <= 6; seed++ {
		rng := rand.New(rand.NewSource(seed))
		mapData := buildBenchMapData(size, size, 4, seed)
		grid := gridFor(mapData)
		canvas := newBenchCanvas(mapData)
		for i := 0; i < 20; i++ { // start with a scatter of named cities, some adjacent so their labels overlap
			record := mapData.MapTileImprovements[rng.Intn(size)][rng.Intn(size)]
			record.CityName, record.CityId = names[rng.Intn(len(names))], 100+i
		}
		drawPoliticalMapTileMajor(canvas, mapData, grid)
		tracker := newTileTracker(mapData)

		nextCityId := 500
		for turn := 1; turn <= turns; turn++ {
			for i, n := 0, 1+rng.Intn(4); i < n; i++ {
				tiles := []fileio.Civ5ReplayEventTile{{X: rng.Intn(size), Y: rng.Intn(size)}}
				event := fileio.Civ5ReplayEvent{Turn: turn, CivId: rng.Intn(4), Tiles: tiles}
				switch rng.Intn(4) {
				case 0:
					event.TypeId, event.Text = fileio.ReplayEventCityFounded, names[rng.Intn(len(names))]+" is founded."
				case 1:
					event.TypeId = fileio.ReplayEventTilesRazed
				case 2:
					event.TypeId = fileio.ReplayEventTilesClaimed
				default:
					event.TypeId = fileio.ReplayEventCityTransferred
				}
				nextCityId = fileio.ApplyReplayEvent(mapData, event, nextCityId)
			}
			redrawDirtyTiles(canvas, mapData, grid, tracker.takeChanges())

			full := newBenchCanvas(mapData)
			drawPoliticalMapTileMajor(full, mapData, grid)
			if diffs, first := canvasDiffs(canvas, full); diffs != 0 {
				t.Fatalf("seed %d turn %d: %d pixels differ from a full redraw, first at %v", seed, turn, diffs, first)
			}
		}
	}
}

// TestPlanRedrawCoversChangedTileAndNeighborsWithinTilesToDraw checks the plan for one changed tile without drawing anything.
func TestPlanRedrawCoversChangedTileAndNeighborsWithinTilesToDraw(t *testing.T) {
	mapData := buildBenchMapData(12, 12, 4, 7)
	grid := gridFor(mapData)
	canvas := newBenchCanvas(mapData)
	drawPoliticalMapTileMajor(canvas, mapData, grid)
	mapSize := mapData.Size()

	pos := fileio.TilePos{Row: 5, Col: 5}
	changed := tileChanges{pos: *mapData.TileImprovement(pos)}
	plan := planRedraw(canvas, grid, changed, nil)

	drawn := tileSet{}
	for _, rc := range plan.tilesToDraw {
		drawn[rc] = true
	}
	// Every tile within two steps of the changed tile is drawn: its neighbors are repainted, and the ring around them fills their rects.
	want := withNeighbors(withNeighbors(tileSet{pos: true}, mapSize), mapSize)
	for rc := range want {
		if !drawn[rc] {
			t.Errorf("tilesToDraw is missing %v", rc)
		}
	}
	if !slices.IsSortedFunc(plan.tilesToDraw, func(a, b fileio.TilePos) int {
		return cmp.Or(cmp.Compare(a.Row, b.Row), cmp.Compare(a.Col, b.Col))
	}) {
		t.Errorf("tilesToDraw is not in row-major order: %v", plan.tilesToDraw)
	}

	// One rect per repainted tile (the tile and its six neighbors); no labels here.
	if got := len(plan.dirtyRects); got != 7 {
		t.Errorf("len(dirtyRects) = %d, want 7", got)
	}
	bounds := image.Rect(0, 0, grid.width, grid.height)
	for _, rect := range plan.dirtyRects {
		if !rect.In(bounds) {
			t.Errorf("paste rect %v is outside the canvas %v", rect, bounds)
		}
	}
	if !slices.ContainsFunc(plan.dirtyRects, grid.tileBounds(pos).Overlaps) {
		t.Errorf("no paste rect covers the changed tile %v", pos)
	}
}

func TestTileTrackerReportsExactlyTheTilesWhoseRecordChanged(t *testing.T) {
	mapData := buildBenchMapData(12, 12, 4, 1)
	tracker := newTileTracker(mapData)

	mapData.MapTileImprovements[3][4].Owner++
	oldName := mapData.MapTileImprovements[7][1].CityName
	mapData.MapTileImprovements[7][1].CityName = "Renamed"
	mapData.MapTileImprovements[11][11].RouteType = 2

	want := tileSet{{Row: 3, Col: 4}: true, {Row: 7, Col: 1}: true, {Row: 11, Col: 11}: true}
	got := tracker.takeChanges()
	if !reflect.DeepEqual(got.tiles(), want) {
		t.Errorf("tracker.takeChanges() = %v, want %v", got.tiles(), want)
	}
	if got[fileio.TilePos{Row: 7, Col: 1}].CityName != oldName {
		t.Errorf("the change to (7, 1) reports the earlier city name %q, want %q", got[fileio.TilePos{Row: 7, Col: 1}].CityName, oldName)
	}
	if got := tracker.takeChanges(); len(got) != 0 {
		t.Errorf("a second call reported %v; the tracker should be up to date", got)
	}
}

// A change to any field is caught, including ones no event touches today: nothing here lists event types.
func TestTileTrackerNeedsNoKnowledgeOfEvents(t *testing.T) {
	mapData := buildBenchMapData(6, 6, 2, 1)
	tracker := newTileTracker(mapData)

	tile := mapData.MapTileImprovements[2][3]
	tile.Improvement, tile.UnitId, tile.RouteOwner = 5, 9, 1

	if got := tracker.takeChanges().tiles(); !got[fileio.TilePos{Row: 2, Col: 3}] || len(got) != 1 {
		t.Errorf("tracker.takeChanges() = %v, want just (2, 3)", got)
	}
}

// An event that doesn't change anything, such as claiming a tile its civ already owns, repaints nothing.
func TestTileTrackerIgnoresEventsThatChangeNothing(t *testing.T) {
	mapData := buildBenchMapData(12, 12, 4, 1)
	tracker := newTileTracker(mapData)

	owner := mapData.MapTileImprovements[5][5].Owner
	fileio.ApplyReplayEvent(mapData, fileio.Civ5ReplayEvent{
		TypeId: fileio.ReplayEventTilesClaimed, CivId: owner, Tiles: []fileio.Civ5ReplayEventTile{{X: 5, Y: 5}},
	}, 0)

	if got := tracker.takeChanges(); len(got) != 0 {
		t.Errorf("tracker.takeChanges() = %v after a no-op claim, want nothing", got)
	}
}

// Every event type fileio.ApplyReplayEvent handles must show up as changed tiles, or the renderer would skip them.
func TestTileTrackerSeesEveryHandledEventType(t *testing.T) {
	tile := []fileio.Civ5ReplayEventTile{{X: 4, Y: 6}}
	events := map[string]fileio.Civ5ReplayEvent{
		"city founded":     {TypeId: fileio.ReplayEventCityFounded, Text: "Testopolis is founded.", Tiles: tile},
		"tiles claimed":    {TypeId: fileio.ReplayEventTilesClaimed, CivId: 3, Tiles: tile},
		"city transferred": {TypeId: fileio.ReplayEventCityTransferred, CivId: 3, Tiles: tile},
		"tiles razed":      {TypeId: fileio.ReplayEventTilesRazed, Tiles: tile},
	}
	for name, event := range events {
		mapData := buildBenchMapData(12, 12, 4, 1)
		mapData.MapTileImprovements[6][4].Owner = 0 // not already owned by civ 3
		tracker := newTileTracker(mapData)
		fileio.ApplyReplayEvent(mapData, event, 100)
		if got := tracker.takeChanges().tiles(); !got[fileio.TilePos{Row: 6, Col: 4}] {
			t.Errorf("%s: tracker.takeChanges() = %v, want it to include (6, 4)", name, got)
		}
	}
}

func TestFrameRegionsAlwaysYieldsAFrame(t *testing.T) {
	noop := []image.Rectangle{noopRect}
	for name, in := range map[string][]image.Rectangle{
		"no rects":   nil,
		"only empty": {{}, image.Rect(5, 5, 5, 9)},
	} {
		if got := gifFrameRects(in); !slices.Equal(got, noop) {
			t.Errorf("%s: gifFrameRects() = %v, want the no-op frame %v", name, got, noop)
		}
	}
	real := []image.Rectangle{image.Rect(0, 0, 10, 10), {}, image.Rect(100, 0, 110, 10)}
	if got, want := gifFrameRects(real), raster.MergeOverlappingRects(real); !slices.Equal(got, want) {
		t.Errorf("gifFrameRects(%v) = %v, want the clusters %v", real, got, want)
	}
}

func TestTileRepaintRectsAreRowMajor(t *testing.T) {
	l := newTileLayout(16, 1)
	bounds := image.Rect(0, 0, 2000, 2000)
	rects := tileRepaintRects(tileSet{{Row: 2, Col: 3}: true, {Row: 0, Col: 1}: true, {Row: 2, Col: 0}: true}, bounds, l)

	wantOrder := []fileio.TilePos{{Row: 0, Col: 1}, {Row: 2, Col: 0}, {Row: 2, Col: 3}}
	if len(rects) != len(wantOrder) {
		t.Fatalf("got %d rects, want %d", len(rects), len(wantOrder))
	}
	for i, rc := range wantOrder {
		if want := l.repaintRect(rc, bounds); rects[i] != want {
			t.Errorf("rect %d = %v, want %v for tile %v", i, rects[i], want, rc)
		}
	}
	if got := tileRepaintRects(tileSet{}, bounds, l); len(got) != 0 {
		t.Errorf("an empty paint set gave %v, want nothing", got)
	}
}

// The replay keeps its thin river, one pixel wide, not the map's two-pixel one: a river edge is a tile radius long, so it paints about that many pixels.
func TestReplayRiversAreOnePixelWide(t *testing.T) {
	mapData := terrainWithMountains([][]bool{{false}})
	mapData.MapTiles[0][0].RiverData = 1 // the east edge
	mapData.MapTileImprovements = [][]*fileio.Civ5MapTileImprovement{{{CityId: -1, Owner: -1, RouteType: 255}}}
	canvas := newBenchCanvas(mapData)

	paintTiles(canvas, mapData, gridFor(mapData), []fileio.TilePos{{Row: 0, Col: 0}})

	river := canvas.IndexFor(riverColor.R, riverColor.G, riverColor.B)
	painted := 0
	bounds := canvas.Image().Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if canvas.IndexAt(x, y) == river {
				painted++
			}
		}
	}
	if radius := int(tileRadius); painted < radius/2 || painted > radius*3/2 {
		t.Errorf("the river painted %d pixels, want about %d (a tile radius, one pixel wide)", painted, radius)
	}
}

func TestRepaintRectCoversTheHexPlusPad(t *testing.T) {
	const radius = 16.0
	l := newTileLayout(radius, 10)
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
	l := newTileLayout(16, 1) // tile (0, 0) is centered at (24, 24)
	if got, want := l.repaintRect(fileio.TilePos{Row: 0, Col: 0}, image.Rect(0, 0, 30, 30)), image.Rect(6, 6, 30, 30); got != want {
		t.Errorf("repaintRect clipped = %v, want %v", got, want)
	}
	if got := l.repaintRect(fileio.TilePos{Row: 20, Col: 20}, image.Rect(0, 0, 30, 30)); !got.Empty() {
		t.Errorf("repaintRect of a tile outside the bounds = %v, want empty", got)
	}
}

func TestTileEntitiesMountain(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 2}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: -1}}},
	}
	entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1), color.RGBA{255, 255, 255, 255})
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
	entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1), cityColor)
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
	entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1), color.RGBA{255, 255, 255, 255})
	if len(entities) != 2 || entities[0].Type != EntityMountain || entities[1].Type != EntityCity {
		t.Fatalf("TileEntities() = %+v, want [EntityMountain, EntityCity] in that order", entities)
	}
}

func TestTileEntitiesNone(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 0}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{{{CityId: -1}}},
	}
	if entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1), color.RGBA{255, 255, 255, 255}); entities != nil {
		t.Errorf("TileEntities() = %v, want nil", entities)
	}
}

func TestTileEntitiesNoImprovementDataIsSafe(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTiles:            [][]*fileio.Civ5MapTilePhysical{{{Elevation: 0}}},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{},
	}
	if entities := TileEntities(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 1), color.RGBA{255, 255, 255, 255}); entities != nil {
		t.Errorf("TileEntities() with no improvement data = %v, want nil", entities)
	}
}

// A mountain is a small peak on a larger base, both pointing up on screen from the tile center.
func TestDrawMountainPointsUp(t *testing.T) {
	canvas := newCanvas(60, 60)
	background := color.RGBA{0, 0, 0, 255}

	drawMountain(canvas, newTileLayout(16, 1), 10, 20) // the base is a triangle of radius 16 about (10, 20), the peak one of radius 8 about (10, 12)

	for _, tt := range []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{"above the apex", 10, 2, background},
		{"near the apex, in the peak", 10, 6, mountainPeakColor},
		{"in the peak", 10, 12, mountainPeakColor},
		{"below the peak, in the base", 10, 20, mountainBaseColor},
		{"near the bottom of the base", 10, 27, mountainBaseColor},
		{"below the base", 10, 29, background},
	} {
		if got := colorAt(canvas, tt.x, tt.y); got != tt.want {
			t.Errorf("%s: pixel (%d, %d) = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}
}

// A city icon is a square reaching 0.3 of a radius above the tile center and 0.2 below it.
func TestDrawCityIconReachesFurtherAboveTheCenterThanBelow(t *testing.T) {
	canvas := newCanvas(60, 60)
	city := color.RGBA{200, 100, 50, 255}
	icon, background := markerColor(city), color.RGBA{0, 0, 0, 255}

	drawCityIcon(canvas, newTileLayout(16, 1), 10, 20, city) // x from 6.8 to 14.8, y from 15.2 to 23.2

	for _, tt := range []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{"just above the top", 10, 14, background},
		{"top row", 10, 15, icon},
		{"bottom row", 10, 22, icon},
		{"just below the bottom", 10, 23, background},
		{"left column", 7, 18, icon},
		{"left of it", 6, 18, background},
		{"right column", 14, 18, icon},
		{"right of it", 15, 18, background},
	} {
		if got := colorAt(canvas, tt.x, tt.y); got != tt.want {
			t.Errorf("%s: pixel (%d, %d) = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}
}
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

	canvas := raster.NewGrowingPalettedCanvas(1, 1) // its palette ends up holding every color drawn
	grid := gridFor(mapData)
	drawPoliticalMapTileMajor(canvas, mapData, grid)
	drawMountain(canvas, newTileLayout(tileRadius, 1), 0, 0)

	palette := replayPalette(mapData)
	for _, drawn := range canvas.Palette() {
		r, g, b, _ := drawn.RGBA()
		if c := (color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}); !paletteContains(palette, c) {
			t.Errorf("the renderer drew %v, which is not in replayPalette", c)
		}
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
