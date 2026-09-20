package graphics

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// newValidReplayFixtures returns a minimal mapData/replayData pair that passes
// fileio.ValidateReplayRenderable, for use as a baseline in the tests below.
func newValidReplayFixtures() (*fileio.Civ5MapData, *fileio.Civ5ReplayData) {
	mapData := &fileio.Civ5MapData{
		TerrainList: []string{"TERRAIN_GRASS"},
		MapTiles: [][]*fileio.Civ5MapTilePhysical{
			{{TerrainType: 0, Elevation: 0}},
		},
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{Owner: -1, CityId: -1, RouteType: 255}},
		},
		CityOwnerIndexMap: map[int]int{},
	}
	replayData := &fileio.Civ5ReplayData{
		IsReplayFile: true,
		PlayerCiv:    "CIVILIZATION_ROME",
		AllCivs:      []fileio.Civ5ReplayCiv{{Name: "CIVILIZATION_ROME"}},
		AllReplayEvents: []fileio.Civ5ReplayEvent{
			{Turn: 1, TypeId: fileio.ReplayEventTilesClaimed, CivId: 0, Tiles: []fileio.Civ5ReplayEventTile{{X: 0, Y: 0}}},
		},
		// Matches the 1x1 map above, as a real .civ5replay file's embedded dimensions would.
		MapWidth:  1,
		MapHeight: 1,
	}
	return mapData, replayData
}

func TestDrawReplaySucceedsAndWritesFile(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()

	outputPath := filepath.Join(t.TempDir(), "replay.gif")
	if err := prepareAndDrawReplay(mapData, replayData, outputPath, 0); err != nil {
		t.Fatalf("DrawReplay() returned error: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("DrawReplay() did not write an output file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("DrawReplay() wrote an empty output file")
	}
}

func TestDrawReplayHandlesNilCityOwnerIndexMap(t *testing.T) {
	mapData, replayData := newValidReplayFixtures()
	// Simulate a map loaded from a bare JSON export where this map was never initialized.
	mapData.CityOwnerIndexMap = nil

	outputPath := filepath.Join(t.TempDir(), "replay.gif")
	if err := prepareAndDrawReplay(mapData, replayData, outputPath, 0); err != nil {
		t.Fatalf("DrawReplay() with nil CityOwnerIndexMap returned error: %v", err)
	}
}

func TestTileTrackerReportsExactlyTheTilesWhoseRecordChanged(t *testing.T) {
	mapData := buildBenchMapData(12, 12, 4, 1)
	tracker := newTileTracker(mapData)

	mapData.MapTileImprovements[3][4].Owner++
	mapData.MapTileImprovements[7][1].CityName = "Renamed"
	mapData.MapTileImprovements[11][11].RouteType = 2

	want := tileSet{{3, 4}: true, {7, 1}: true, {11, 11}: true}
	if got := tracker.takeChanges(); !reflect.DeepEqual(got, want) {
		t.Errorf("tracker.takeChanges() = %v, want %v", got, want)
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

	if got := tracker.takeChanges(); !got[tileCoord{2, 3}] || len(got) != 1 {
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
		if got := tracker.takeChanges(); !got[tileCoord{6, 4}] {
			t.Errorf("%s: tracker.takeChanges() = %v, want it to include (6, 4)", name, got)
		}
	}
}

func TestClusterRects(t *testing.T) {
	tests := []struct {
		name string
		in   []image.Rectangle
		want []image.Rectangle
	}{
		{"nothing", nil, nil},
		{"one rect is unchanged", []image.Rectangle{image.Rect(5, 5, 15, 15)}, []image.Rectangle{image.Rect(5, 5, 15, 15)}},
		{"far apart stay separate",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(100, 100, 110, 110)},
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(100, 100, 110, 110)}},
		{"overlapping merge",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(5, 5, 20, 20)},
			[]image.Rectangle{image.Rect(0, 0, 20, 20)}},
		{"touching merge",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10, 0, 20, 10)},
			[]image.Rectangle{image.Rect(0, 0, 20, 10)}},
		{"just inside the gap merge",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10+clusterGap-1, 0, 30, 10)},
			[]image.Rectangle{image.Rect(0, 0, 30, 10)}},
		{"exactly the gap apart stay separate",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10+clusterGap, 0, 30, 10)},
			[]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(10+clusterGap, 0, 30, 10)}},
		{"empty rects are dropped",
			[]image.Rectangle{image.Rect(0, 0, 10, 10), {}, image.Rect(50, 50, 50, 60)},
			[]image.Rectangle{image.Rect(0, 0, 10, 10)}},
	}
	for _, tt := range tests {
		if got := clusterRects(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("%s: clusterRects(%v) = %v, want %v", tt.name, tt.in, got, tt.want)
		}
	}
}

// Merging two rects grows their box, and the bigger box can reach a third rect that neither was near alone; the input order must not matter.
func TestClusterRectsMergesChainsInAnyOrder(t *testing.T) {
	tall := image.Rect(0, 0, 10, 50)
	wide := image.Rect(15, 0, 60, 10)    // near tall, so they merge into (0, 0, 60, 50)
	inside := image.Rect(45, 30, 55, 40) // far from both alone, but inside their merged box
	want := []image.Rectangle{image.Rect(0, 0, 60, 50)}
	orders := [][]image.Rectangle{{tall, wide, inside}, {inside, tall, wide}, {wide, inside, tall}, {inside, wide, tall}}
	for _, in := range orders {
		if got := clusterRects(in); !slices.Equal(got, want) {
			t.Errorf("clusterRects(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestFrameRegionsAlwaysYieldsAFrame(t *testing.T) {
	noop := []image.Rectangle{noopRegion}
	for name, in := range map[string][]image.Rectangle{
		"no rects":   nil,
		"only empty": {{}, image.Rect(5, 5, 5, 9)},
	} {
		if got := frameRegions(in); !slices.Equal(got, noop) {
			t.Errorf("%s: frameRegions() = %v, want the no-op frame %v", name, got, noop)
		}
	}
	real := []image.Rectangle{image.Rect(0, 0, 10, 10), {}, image.Rect(100, 0, 110, 10)}
	if got, want := frameRegions(real), clusterRects(real); !slices.Equal(got, want) {
		t.Errorf("frameRegions(%v) = %v, want the clusters %v", real, got, want)
	}
}

// For any input, every rect is covered by some cluster, no two clusters are within the gap of each other, and the input order doesn't matter.
func TestClusterRectsCoversInputWithSeparatedClusters(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 200; trial++ {
		var in []image.Rectangle
		for i, n := 0, 1+rng.Intn(60); i < n; i++ {
			x, y := rng.Intn(400), rng.Intn(400)
			in = append(in, image.Rect(x, y, x+1+rng.Intn(40), y+1+rng.Intn(40)))
		}
		clusters := clusterRects(in)
		reversed := slices.Clone(in)
		slices.Reverse(reversed)
		if again := clusterRects(reversed); !slices.Equal(clusters, again) {
			t.Fatalf("trial %d: reversing the input gave %v, want %v", trial, again, clusters)
		}
		for _, r := range in {
			if !slices.ContainsFunc(clusters, func(c image.Rectangle) bool { return r.In(c) }) {
				t.Fatalf("trial %d: %v is in no cluster of %v", trial, r, clusters)
			}
		}
		for i, a := range clusters {
			for _, b := range clusters[i+1:] {
				if a.Inset(-clusterGap).Overlaps(b) {
					t.Fatalf("trial %d: clusters %v and %v are within the gap", trial, a, b)
				}
			}
		}
	}
}

// DrawReplay's output must be byte-identical run to run, which needs paintTileRects to sort its rects.
func TestDrawReplayOutputIsDeterministic(t *testing.T) {
	const mapHeight, mapWidth, turns, dirtyPerTurn, civs = 30, 40, 10, 15, 8
	var first []byte
	for run := 0; run < 6; run++ {
		mapData := buildBenchMapData(mapHeight, mapWidth, civs, 1)
		turnsEvents := buildBenchTurnEvents(mapHeight, mapWidth, turns, dirtyPerTurn, civs, 99) // scattered: several regions per turn
		path := filepath.Join(t.TempDir(), "replay.gif")
		if err := prepareAndDrawReplay(mapData, buildBenchReplayData(mapData, turnsEvents), path, 0); err != nil {
			t.Fatalf("DrawReplay() = %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			first = got
		} else if !bytes.Equal(first, got) {
			t.Fatalf("run %d produced a different GIF (%d vs %d bytes)", run, len(got), len(first))
		}
	}
}

// runReplayFrames is DrawReplay's per-turn loop minus GIF encoding: later turns repaint and snapshot only the dirty region.
func runReplayFrames(renderer *MapRenderer, mapData *fileio.Civ5MapData, turnsEvents [][]fileio.Civ5ReplayEvent) {
	canvas := newBenchCanvas(mapData)
	nextCityId := 0
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])
	var tracker *tileTracker
	for turnIndex, events := range turnsEvents {
		for _, event := range events {
			nextCityId = fileio.ApplyReplayEvent(mapData, event, nextCityId)
		}
		if turnIndex == 0 {
			tracker = newTileTracker(mapData)
			renderer.DrawPoliticalMapTileMajor(canvas, mapData)
			canvas.Snapshot(canvas.Image().Bounds())
		} else if dirtyRects := renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, tracker.takeChanges()); len(dirtyRects) > 0 {
			canvas.Snapshot(unionRects(dirtyRects))
		}
	}
}

const (
	benchEndToEndTurns      = 40
	benchEndToEndDirtyTiles = 15
	benchEndToEndCivs       = 8
)

func BenchmarkDrawReplayFrames_Medium(b *testing.B) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEvents(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 99)
		b.StartTimer()

		runReplayFrames(renderer, mapData, turnsEvents)
	}
}

// BenchmarkDrawReplayFrames_Clustered_Medium is the Medium benchmark with clustered dirty tiles, the realistic turn shape.
func BenchmarkDrawReplayFrames_Clustered_Medium(b *testing.B) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEventsClustered(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 5, 99)
		b.StartTimer()

		runReplayFrames(renderer, mapData, turnsEvents)
	}
}

// buildBenchReplayData flattens turnsEvents into a Civ5ReplayData DrawReplay can consume directly.
func buildBenchReplayData(mapData *fileio.Civ5MapData, turnsEvents [][]fileio.Civ5ReplayEvent) *fileio.Civ5ReplayData {
	var events []fileio.Civ5ReplayEvent
	for _, turn := range turnsEvents {
		events = append(events, turn...)
	}
	civs := make([]fileio.Civ5ReplayCiv, len(mapData.Civ5PlayerData))
	for i, p := range mapData.Civ5PlayerData {
		civs[i] = fileio.Civ5ReplayCiv{Name: p.CivType, LongName: p.TeamColor}
	}
	return &fileio.Civ5ReplayData{
		IsReplayFile:    true,
		PlayerCiv:       civs[0].Name,
		AllCivs:         civs,
		AllReplayEvents: events,
		MapWidth:        len(mapData.MapTiles[0]),
		MapHeight:       len(mapData.MapTiles),
	}
}

// BenchmarkDrawReplayEndToEnd_Clustered_Medium runs the real DrawReplay, GIF encoding and file write included.
func BenchmarkDrawReplayEndToEnd_Clustered_Medium(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEventsClustered(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 5, 99)
		replayData := buildBenchReplayData(mapData, turnsEvents)
		outputPath := fmt.Sprintf("%s/replay-%d.gif", dir, i)
		b.StartTimer()

		if err := prepareAndDrawReplay(mapData, replayData, outputPath, 0); err != nil {
			b.Fatalf("DrawReplay() = %v", err)
		}
	}
}

// BenchmarkDrawReplayEndToEnd_Scattered_Medium is the clustered benchmark with map-wide scattered dirty tiles: the worst case for cropping.
func BenchmarkDrawReplayEndToEnd_Scattered_Medium(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEvents(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 99)
		replayData := buildBenchReplayData(mapData, turnsEvents)
		outputPath := fmt.Sprintf("%s/replay-%d.gif", dir, i)
		b.StartTimer()

		if err := prepareAndDrawReplay(mapData, replayData, outputPath, 0); err != nil {
			b.Fatalf("DrawReplay() = %v", err)
		}
	}
}

// writeGifFullFrames encodes turnsEvents with a full-canvas frame every turn (the pre-delta-frame format), as a size baseline.
func writeGifFullFrames(mapData *fileio.Civ5MapData, turnsEvents [][]fileio.Civ5ReplayEvent, outputPath string) error {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := newBenchCanvas(mapData)
	outGif := &gif.GIF{}
	nextCityId := 0
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])
	var tracker *tileTracker
	for turnIndex, events := range turnsEvents {
		for _, event := range events {
			nextCityId = fileio.ApplyReplayEvent(mapData, event, nextCityId)
		}
		if turnIndex == 0 {
			tracker = newTileTracker(mapData)
			renderer.DrawPoliticalMapTileMajor(canvas, mapData)
		} else {
			renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, tracker.takeChanges())
		}
		outGif.Image = append(outGif.Image, canvas.Snapshot(canvas.Image().Bounds()))
		outGif.Delay = append(outGif.Delay, GIF_DELAY)
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.EncodeAll(f, outGif)
}

// TestDrawReplayOutputSizeVsFullFrameEveryTurn logs how much smaller DrawReplay's GIF is than a full-frame-every-turn one; sizes are data-dependent, so it asserts no number.
func TestDrawReplayOutputSizeVsFullFrameEveryTurn(t *testing.T) {
	const mapHeight, mapWidth, turns, dirtyPerTurn, civs, clusterRadius = 20, 30, 10, 15, 8, 5

	dir := t.TempDir()

	mapDataOld := buildBenchMapData(mapHeight, mapWidth, civs, 1)
	turnsEventsOld := buildBenchTurnEventsClustered(mapHeight, mapWidth, turns, dirtyPerTurn, civs, clusterRadius, 99)
	oldPath := filepath.Join(dir, "old.gif")
	if err := writeGifFullFrames(mapDataOld, turnsEventsOld, oldPath); err != nil {
		t.Fatalf("writeGifFullFrames() = %v", err)
	}

	mapDataNew := buildBenchMapData(mapHeight, mapWidth, civs, 1)
	turnsEventsNew := buildBenchTurnEventsClustered(mapHeight, mapWidth, turns, dirtyPerTurn, civs, clusterRadius, 99)
	replayData := buildBenchReplayData(mapDataNew, turnsEventsNew)
	newPath := filepath.Join(dir, "new.gif")
	if err := prepareAndDrawReplay(mapDataNew, replayData, newPath, 0); err != nil {
		t.Fatalf("DrawReplay() = %v", err)
	}

	oldInfo, err := os.Stat(oldPath)
	if err != nil {
		t.Fatalf("stat old.gif: %v", err)
	}
	newInfo, err := os.Stat(newPath)
	if err != nil {
		t.Fatalf("stat new.gif: %v", err)
	}
	t.Logf("full-frame-every-turn: %d bytes; cropped-delta-frame: %d bytes (%.1f%% of original)",
		oldInfo.Size(), newInfo.Size(), 100*float64(newInfo.Size())/float64(oldInfo.Size()))
}

// TestDrawReplayGifFramesReconstructFullRedrawEveryFrame composites DrawReplay's GIF frames and checks every turn matches a full redraw exactly.
func TestDrawReplayGifFramesReconstructFullRedrawEveryFrame(t *testing.T) {
	buildScenario := func() (*fileio.Civ5MapData, [][]fileio.Civ5ReplayEvent) {
		mapData := buildBenchMapData(16, 16, 4, 3)
		turnsEvents := [][]fileio.Civ5ReplayEvent{
			{{Turn: 1, TypeId: fileio.ReplayEventTilesClaimed, CivId: 1, Tiles: []fileio.Civ5ReplayEventTile{{X: 3, Y: 3}}}}, // establishes frame 0
			{{Turn: 2, TypeId: fileio.ReplayEventCityFounded, Text: "Testopolis is founded.", Tiles: []fileio.Civ5ReplayEventTile{{X: 8, Y: 8}}}},
			{{Turn: 3, TypeId: fileio.ReplayEventTilesClaimed, CivId: 2, Tiles: []fileio.Civ5ReplayEventTile{{X: 8, Y: 7}}}}, // dirties a neighbor of the new city's tile
			{{Turn: 4, TypeId: fileio.ReplayEventTilesClaimed, CivId: 0, Tiles: []fileio.Civ5ReplayEventTile{{X: 1, Y: 1}, {X: 14, Y: 14}}}},
		}
		return mapData, turnsEvents
	}

	// Reference: a fresh full redraw every turn.
	mapDataRef, turnsEventsRef := buildScenario()
	rendererRef := NewMapRenderer(DefaultDrawingConfig())
	paletteRef := replayPalette(mapDataRef)
	var referenceFrames []*raster.PalettedCanvas
	nextCityIdRef := 0
	for _, events := range turnsEventsRef {
		for _, event := range events {
			nextCityIdRef = fileio.ApplyReplayEvent(mapDataRef, event, nextCityIdRef)
		}
		frame := raster.NewPalettedCanvas(800, 600, paletteRef)
		rendererRef.DrawPoliticalMapTileMajor(frame, mapDataRef)
		referenceFrames = append(referenceFrames, frame)
	}

	// Actual: the real DrawReplay entry point, producing genuine GIF bytes on disk.
	mapDataReal, turnsEventsReal := buildScenario()
	replayData := buildBenchReplayData(mapDataReal, turnsEventsReal)
	outputPath := filepath.Join(t.TempDir(), "replay.gif")
	if err := prepareAndDrawReplay(mapDataReal, replayData, outputPath, 0); err != nil {
		t.Fatalf("DrawReplay() = %v", err)
	}

	f, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("open output gif: %v", err)
	}
	defer f.Close()
	decoded, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("gif.DecodeAll() = %v", err)
	}
	// A turn spans several GIF blocks; all but the last have delay 0, so a positive delay ends a turn.
	accum := image.NewRGBA(image.Rect(0, 0, decoded.Config.Width, decoded.Config.Height))
	turnIndex := 0
	for i, frame := range decoded.Image {
		draw.Draw(accum, frame.Bounds(), frame, frame.Bounds().Min, draw.Src)
		if decoded.Delay[i] == 0 {
			continue // more blocks belong to this same turn
		}
		if turnIndex >= len(referenceFrames) {
			t.Fatalf("decoded GIF has more turns (index %d) than expected (%d)", turnIndex, len(referenceFrames))
		}

		ref := referenceFrames[turnIndex].Image()
		if accum.Bounds() != ref.Bounds() {
			t.Fatalf("turn %d: reconstructed bounds %v != reference bounds %v", turnIndex, accum.Bounds(), ref.Bounds())
		}

		diffCount := 0
		var firstDiff image.Point
		for y := ref.Bounds().Min.Y; y < ref.Bounds().Max.Y; y++ {
			for x := ref.Bounds().Min.X; x < ref.Bounds().Max.X; x++ {
				if color.RGBAModel.Convert(ref.At(x, y)) != accum.RGBAAt(x, y) {
					if diffCount == 0 {
						firstDiff = image.Pt(x, y)
					}
					diffCount++
				}
			}
		}
		if diffCount != 0 {
			t.Errorf("turn %d: reconstructed GIF composite differs from a full redraw in %d pixels, first at %v - likely a real bug (e.g. a missed dirty rect)",
				turnIndex, diffCount, firstDiff)
		}
		turnIndex++
	}
	if turnIndex != len(referenceFrames) {
		t.Errorf("decoded GIF closed out %d turns, want %d", turnIndex, len(referenceFrames))
	}
}
