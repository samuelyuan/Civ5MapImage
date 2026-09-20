package graphics

import (
	"fmt"
	"image"
	"math/rand"
	"slices"
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
	renderer := NewMapRenderer(DefaultDrawingConfig())
	incremental := raster.NewPalettedCanvas(800, 600, palette)
	mapHeight, mapWidth := len(mapData.MapTiles), len(mapData.MapTiles[0])

	nextCityId := 0
	var tracker *tileTracker
	for turnIndex, events := range turnsEvents {
		for _, event := range events {
			nextCityId = fileio.ApplyReplayEvent(mapData, event, nextCityId)
		}
		if turnIndex == 0 {
			tracker = newTileTracker(mapData)
			renderer.DrawPoliticalMapTileMajor(incremental, mapData)
		} else {
			renderer.RedrawDirtyTiles(incremental, mapData, mapHeight, mapWidth, tracker.takeChanges())
		}

		full := raster.NewPalettedCanvas(800, 600, palette)
		renderer.DrawPoliticalMapTileMajor(full, mapData)
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
	renderer := NewMapRenderer(DefaultDrawingConfig())

	canvas := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	// Simulate a TilesClaimed event flipping ownership of a small cluster of tiles.
	tracker := newTileTracker(mapData)
	for _, rc := range []tileCoord{{5, 5}, {5, 6}, {6, 5}, {6, 6}} {
		tile := mapData.MapTileImprovements[rc.row][rc.col]
		tile.Owner = (tile.Owner + 1) % 4
	}
	renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, tracker.takeChanges())

	fresh := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(fresh, mapData)

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

	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)

	for round := 0; round < 8; round++ {
		tracker := newTileTracker(mapData)
		for i := 0; i < 3; i++ {
			rc := tileCoord{rng.Intn(h), rng.Intn(w)}
			tile := mapData.MapTileImprovements[rc.row][rc.col]
			tile.Owner = (tile.Owner + 1 + rng.Intn(civs-1)) % civs
		}
		renderer.RedrawDirtyTiles(canvas, mapData, h, w, tracker.takeChanges())

		fresh := newBenchCanvas(mapData)
		renderer.DrawPoliticalMapTileMajor(fresh, mapData)
		if count, first := canvasDiffs(canvas, fresh); count != 0 {
			t.Fatalf("round %d: %d pixels differ from a full redraw, first at %v", round, count, first)
		}
	}
}

// TestRedrawDirtyTilesDoesNotTouchUnrelatedPixels checks one dirty tile's repaint leaves far-away pixels untouched (a PasteRegion bounds bug).
func TestRedrawDirtyTilesDoesNotTouchUnrelatedPixels(t *testing.T) {
	mapData := buildBenchMapData(20, 20, 4, 7)
	renderer := NewMapRenderer(DefaultDrawingConfig())

	canvas := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	bounds := canvas.Image().Bounds()
	before := canvas.Snapshot(bounds)

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	tracker := newTileTracker(mapData)
	tile := mapData.MapTileImprovements[10][10]
	tile.Owner = (tile.Owner + 1) % 4

	renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, tracker.takeChanges())

	// Tile (1, 1) is 9+ tiles from (10, 10).
	x, y := pixelLayout(DefaultDrawingConfig().Radius, bounds.Dy()).center(1, 1)
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
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := newBenchCanvas(mapData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	}
}

// BenchmarkScratchCanvasAllocation_Huge isolates allocating and inverting a full-map-sized scratch canvas.
func BenchmarkScratchCanvasAllocation_Huge(b *testing.B) {
	mapHeight, mapWidth := 80, 128
	base := newBenchCanvas(buildBenchMapData(mapHeight, mapWidth, 8, 1))
	radius := DefaultDrawingConfig().Radius
	maxImageWidth, maxImageHeight := GetImagePosition(mapHeight, mapWidth, radius)
	w, h := int(maxImageWidth), int(maxImageHeight)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scratch := base.NewSibling(w, h)
		_ = scratch.Image()
	}
}

func benchmarkRedrawDirtyTiles(b *testing.B, mapData *fileio.Civ5MapData, dirtyCount int) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData) // establish the base frame once

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	rng := rand.New(rand.NewSource(2))
	dirty := tileSet{}
	for len(dirty) < dirtyCount {
		dirty[tileCoord{rng.Intn(mapHeight), rng.Intn(mapWidth)}] = true
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, dirty)
	}
}

// benchmarkRedrawDirtyTilesClustered is benchmarkRedrawDirtyTiles with dirty tiles within clusterRadius of one center.
func benchmarkRedrawDirtyTilesClustered(b *testing.B, mapData *fileio.Civ5MapData, dirtyCount, clusterRadius int) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	rng := rand.New(rand.NewSource(2))
	centerRow, centerCol := rng.Intn(mapHeight), rng.Intn(mapWidth)
	dirty := tileSet{}
	for len(dirty) < dirtyCount {
		row := clampInt(centerRow+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapHeight-1)
		col := clampInt(centerCol+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapWidth-1)
		dirty[tileCoord{row, col}] = true
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, dirty)
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

func TestLabelsToRestamp(t *testing.T) {
	label := func(row, col int, rect image.Rectangle) cityLabel { return cityLabel{row: row, col: col, rect: rect} }
	// a, b and c overlap in a chain; far touches nothing; under sits beneath a pasted rect.
	a := label(0, 0, image.Rect(0, 0, 20, 13))
	b := label(0, 1, image.Rect(15, 0, 40, 13))
	c := label(0, 2, image.Rect(35, 0, 60, 13))
	far := label(5, 5, image.Rect(500, 500, 520, 513))
	under := label(6, 6, image.Rect(200, 200, 230, 213))
	labels := []cityLabel{a, b, c, far, under}

	tests := []struct {
		name        string
		paintSet    tileSet
		pasted      []image.Rectangle
		wantRestamp []bool
		wantChanged []image.Rectangle
	}{
		{"nothing painted or pasted", tileSet{}, nil, []bool{false, false, false, false, false}, nil},
		{"a label on a painted tile pulls in the labels chained to it",
			tileSet{{0, 0}: true}, nil, []bool{true, true, true, false, false}, []image.Rectangle{a.rect}},
		{"a pasted rect over a label restamps it but doesn't count as changed",
			tileSet{}, []image.Rectangle{image.Rect(210, 205, 260, 260)}, []bool{false, false, false, false, true}, nil},
		{"a pasted rect over the middle of a chain pulls in both ends",
			tileSet{}, []image.Rectangle{image.Rect(30, 5, 32, 8)}, []bool{true, true, true, false, false}, nil},
		{"a painted tile with no label changes nothing", tileSet{{9, 9}: true}, nil, []bool{false, false, false, false, false}, nil},
	}
	for _, tt := range tests {
		gotRestamp, gotChanged := labelsToRestamp(labels, tt.paintSet, tt.pasted)
		if !slices.Equal(gotRestamp, tt.wantRestamp) {
			t.Errorf("%s: restamp = %v, want %v", tt.name, gotRestamp, tt.wantRestamp)
		}
		if !slices.Equal(gotChanged, tt.wantChanged) {
			t.Errorf("%s: changed = %v, want %v", tt.name, gotChanged, tt.wantChanged)
		}
	}
}
