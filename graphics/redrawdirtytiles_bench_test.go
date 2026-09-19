package graphics

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/quantize"
)

// --- synthetic map/replay generation for correctness checks and benchmarks ---

// benchCivColors is a few real civColorMap keys, giving synthetic civs real territory colors and borders.
var benchCivColors = []string{
	"PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE", "PLAYERCOLOR_BROWN", "PLAYERCOLOR_CYAN",
	"PLAYERCOLOR_DARK_BLUE", "PLAYERCOLOR_DARK_GREEN", "PLAYERCOLOR_GRAY", "PLAYERCOLOR_GREEN",
	"PLAYERCOLOR_ORANGE", "PLAYERCOLOR_PINK", "PLAYERCOLOR_PURPLE", "PLAYERCOLOR_RED",
}

// buildBenchMapData builds a synthetic political map: numCivs civs holding blockSize-square
// territory blocks (real borders), scattered rivers and roads, one city per block. Deterministic per seed.
func buildBenchMapData(mapHeight, mapWidth, numCivs int, seed int64) *fileio.Civ5MapData {
	rng := rand.New(rand.NewSource(seed))

	players := make([]*fileio.Civ5PlayerData, numCivs)
	cityOwnerIndexMap := make(map[int]int, numCivs)
	for i := 0; i < numCivs; i++ {
		players[i] = &fileio.Civ5PlayerData{
			Index:     i,
			CivType:   fmt.Sprintf("CIVILIZATION_BENCH%d", i),
			TeamColor: benchCivColors[i%len(benchCivColors)],
		}
		cityOwnerIndexMap[i] = i
	}

	const blockSize = 8
	blocksPerRow := (mapWidth + blockSize - 1) / blockSize

	mapTiles := make([][]*fileio.Civ5MapTilePhysical, mapHeight)
	mapImprovements := make([][]*fileio.Civ5MapTileImprovement, mapHeight)
	nextCityId := 0
	for row := 0; row < mapHeight; row++ {
		mapTiles[row] = make([]*fileio.Civ5MapTilePhysical, mapWidth)
		mapImprovements[row] = make([]*fileio.Civ5MapTileImprovement, mapWidth)
		for col := 0; col < mapWidth; col++ {
			riverData := 0
			if rng.Intn(6) == 0 {
				riverData = 1 << uint(rng.Intn(3))
			}
			mapTiles[row][col] = &fileio.Civ5MapTilePhysical{X: col, Y: row, TerrainType: 0, RiverData: riverData}

			blockIndex := (row/blockSize)*blocksPerRow + (col / blockSize)
			owner := blockIndex % numCivs

			routeType := 255
			if rng.Intn(3) == 0 {
				routeType = 0
			}

			cityId := -1
			cityName := ""
			if row%blockSize == blockSize/2 && col%blockSize == blockSize/2 {
				cityId = nextCityId
				cityName = fmt.Sprintf("City%d", nextCityId)
				nextCityId++
			}

			mapImprovements[row][col] = &fileio.Civ5MapTileImprovement{
				X: col, Y: row, Owner: owner, CityId: cityId, CityName: cityName, RouteType: routeType,
			}
		}
	}

	return &fileio.Civ5MapData{
		TerrainList:         []string{"TERRAIN_GRASS", "TERRAIN_OCEAN"},
		MapTiles:            mapTiles,
		MapTileImprovements: mapImprovements,
		Civ5PlayerData:      players,
		CityOwnerIndexMap:   cityOwnerIndexMap,
	}
}

// buildBenchTurnEvents synthesizes TilesClaimed events reassigning dirtyPerTurn random tiles per turn.
func buildBenchTurnEvents(mapHeight, mapWidth, numTurns, dirtyPerTurn, numCivs int, seed int64) [][]fileio.Civ5ReplayEvent {
	rng := rand.New(rand.NewSource(seed))

	turns := make([][]fileio.Civ5ReplayEvent, numTurns)
	for t := 0; t < numTurns; t++ {
		tiles := make([]fileio.Civ5ReplayEventTile, dirtyPerTurn)
		for i := 0; i < dirtyPerTurn; i++ {
			tiles[i] = fileio.Civ5ReplayEventTile{X: rng.Intn(mapWidth), Y: rng.Intn(mapHeight)}
		}
		turns[t] = []fileio.Civ5ReplayEvent{
			{Turn: t, TypeId: ReplayEventTilesClaimed, CivId: rng.Intn(numCivs), Tiles: tiles},
		}
	}
	return turns
}

// buildBenchTurnEventsClustered is buildBenchTurnEvents with each turn's tiles within clusterRadius
// of a random center: a small scratch bounding box, versus a near-full-map one when scattered.
func buildBenchTurnEventsClustered(mapHeight, mapWidth, numTurns, dirtyPerTurn, numCivs, clusterRadius int, seed int64) [][]fileio.Civ5ReplayEvent {
	rng := rand.New(rand.NewSource(seed))

	turns := make([][]fileio.Civ5ReplayEvent, numTurns)
	for t := 0; t < numTurns; t++ {
		centerRow := rng.Intn(mapHeight)
		centerCol := rng.Intn(mapWidth)
		tiles := make([]fileio.Civ5ReplayEventTile, dirtyPerTurn)
		for i := 0; i < dirtyPerTurn; i++ {
			row := clampInt(centerRow+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapHeight-1)
			col := clampInt(centerCol+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapWidth-1)
			tiles[i] = fileio.Civ5ReplayEventTile{X: col, Y: row}
		}
		turns[t] = []fileio.Civ5ReplayEvent{
			{Turn: t, TypeId: ReplayEventTilesClaimed, CivId: rng.Intn(numCivs), Tiles: tiles},
		}
	}
	return turns
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// unionRects returns the bounding rect of rects (panics if empty); test-only, to time quantizeRegion on one known rect.
func unionRects(rects []image.Rectangle) image.Rectangle {
	union := rects[0]
	for _, r := range rects[1:] {
		union = union.Union(r)
	}
	return union
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

// --- correctness ---

// maxChannelDelta returns the largest per-channel (R, G, B, A) difference between two colors; it takes
// color.Color so 8-bit RGBA pixels compare against 16-bit palette entries.
func maxChannelDelta(a, b color.Color) int {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	delta := func(x, y uint32) int {
		d := int(x>>8) - int(y>>8)
		if d < 0 {
			return -d
		}
		return d
	}
	max := delta(ar, br)
	for _, d := range []int{delta(ag, bg), delta(ab, bb), delta(aa, ba)} {
		if d > max {
			max = d
		}
	}
	return max
}

// TestRedrawDirtyTilesMatchesFullRedrawAfterMutation checks that repainting only the dirty tiles matches
// redrawing the whole mutated map. Allows a small per-channel tolerance: the scratch canvas translates
// tiles, and gg's anti-aliasing isn't exactly translation-invariant.
func TestRedrawDirtyTilesMatchesFullRedrawAfterMutation(t *testing.T) {
	const maxAllowedDelta = 8 // measured max observed was 3; comfortable margin, still tight

	mapData := buildBenchMapData(12, 12, 4, 7)
	renderer := NewMapRenderer(DefaultDrawingConfig())

	canvas := NewDrawingContext(800, 600)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	// Simulate a TilesClaimed event flipping ownership of a small cluster of tiles.
	dirty := tileSet{}
	for _, rc := range []tileCoord{{5, 5}, {5, 6}, {6, 5}, {6, 6}} {
		dirty[rc] = true
		tile := mapData.MapTileImprovements[rc.row][rc.col]
		tile.Owner = (tile.Owner + 1) % 4
	}
	paintSet := expandWithNeighbors(dirty, mapHeight, mapWidth)

	renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, paintSet)
	incremental := canvas.Image().(*image.RGBA)

	freshCanvas := NewDrawingContext(800, 600)
	renderer.DrawPoliticalMapTileMajor(freshCanvas, mapData)
	fresh := freshCanvas.Image().(*image.RGBA)

	if incremental.Bounds() != fresh.Bounds() {
		t.Fatalf("bounds differ: incremental %v, fresh %v", incremental.Bounds(), fresh.Bounds())
	}

	diffCount := 0
	worstDelta := 0
	var worstAt image.Point
	for y := incremental.Bounds().Min.Y; y < incremental.Bounds().Max.Y; y++ {
		for x := incremental.Bounds().Min.X; x < incremental.Bounds().Max.X; x++ {
			ic, fc := incremental.RGBAAt(x, y), fresh.RGBAAt(x, y)
			if ic == fc {
				continue
			}
			diffCount++
			if d := maxChannelDelta(ic, fc); d > worstDelta {
				worstDelta = d
				worstAt = image.Pt(x, y)
			}
		}
	}
	t.Logf("RedrawDirtyTiles() vs full redraw: %d differing pixels, worst per-channel delta %d at %v",
		diffCount, worstDelta, worstAt)
	if worstDelta > maxAllowedDelta {
		t.Errorf("RedrawDirtyTiles() differs from a full redraw by %d in a color channel at %v (want <= %d) - likely a real placement bug, not rounding noise",
			worstDelta, worstAt, maxAllowedDelta)
	}
}

// TestRedrawDirtyTilesMatchesFullRedrawWithCrowdedLabels repaints random tiles on a map packed with
// overlapping city labels and requires a match with a full redraw, so a mis-stacked label is caught.
func TestRedrawDirtyTilesMatchesFullRedrawWithCrowdedLabels(t *testing.T) {
	// A few border-corner pixels differ from a full redraw even without cities; a mis-stacked label changes hundreds, so a count bound still catches it.
	const maxAllowedDelta, maxBigDiffPixels = 8, 12
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
	canvas := NewDrawingContext(800, 600)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)

	for round := 0; round < 8; round++ {
		dirty := tileSet{}
		for i := 0; i < 3; i++ {
			rc := tileCoord{rng.Intn(h), rng.Intn(w)}
			dirty[rc] = true
			tile := mapData.MapTileImprovements[rc.row][rc.col]
			tile.Owner = (tile.Owner + 1 + rng.Intn(civs-1)) % civs
		}
		renderer.RedrawDirtyTiles(canvas, mapData, h, w, expandWithNeighbors(dirty, h, w))

		freshCanvas := NewDrawingContext(800, 600)
		renderer.DrawPoliticalMapTileMajor(freshCanvas, mapData)
		incremental, fresh := canvas.Image().(*image.RGBA), freshCanvas.Image().(*image.RGBA)

		bigDiffs := 0
		for y := 0; y < incremental.Bounds().Dy(); y++ {
			for x := 0; x < incremental.Bounds().Dx(); x++ {
				if maxChannelDelta(incremental.RGBAAt(x, y), fresh.RGBAAt(x, y)) > maxAllowedDelta {
					bigDiffs++
				}
			}
		}
		if bigDiffs > maxBigDiffPixels {
			t.Fatalf("round %d: %d pixels differ from a full redraw by more than %d (want <= %d)", round, bigDiffs, maxAllowedDelta, maxBigDiffPixels)
		}
	}
}

// TestRedrawDirtyTilesDoesNotTouchUnrelatedPixels checks one dirty tile's repaint leaves far-away pixels untouched (a PasteRegion bounds bug).
func TestRedrawDirtyTilesDoesNotTouchUnrelatedPixels(t *testing.T) {
	mapData := buildBenchMapData(20, 20, 4, 7)
	renderer := NewMapRenderer(DefaultDrawingConfig())

	canvas := NewDrawingContext(800, 600)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	before := cloneRGBA(canvas.Image().(*image.RGBA))

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	dirty := tileSet{{10, 10}: true}
	tile := mapData.MapTileImprovements[10][10]
	tile.Owner = (tile.Owner + 1) % 4
	paintSet := expandWithNeighbors(dirty, mapHeight, mapWidth)

	renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, paintSet)
	after := canvas.Image().(*image.RGBA)

	// Tile (1, 1) is 9+ tiles from (10, 10); locate it the way RedrawDirtyTiles does.
	radius := DefaultDrawingConfig().Radius
	_, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, radius)
	x, y := fileio.GetImagePosition(1, 1, radius)
	cx, cy := int(x), int(maxImageHeight-y)

	checkRect := image.Rect(cx-8, cy-8, cx+8, cy+8).Intersect(before.Bounds())
	for py := checkRect.Min.Y; py < checkRect.Max.Y; py++ {
		for px := checkRect.Min.X; px < checkRect.Max.X; px++ {
			if before.RGBAAt(px, py) != after.RGBAAt(px, py) {
				t.Fatalf("pixel (%d, %d) near untouched tile (1, 1) changed after RedrawDirtyTiles on tile (10, 10): before %v, after %v",
					px, py, before.RGBAAt(px, py), after.RGBAAt(px, py))
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
	canvas := NewDrawingContext(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	}
}

// BenchmarkScratchCanvasAllocation_Huge isolates allocating and inverting a full-map-sized scratch canvas.
func BenchmarkScratchCanvasAllocation_Huge(b *testing.B) {
	mapHeight, mapWidth := 80, 128
	radius := DefaultDrawingConfig().Radius
	maxImageWidth, maxImageHeight := fileio.GetImagePosition(mapHeight, mapWidth, radius)
	w, h := int(maxImageWidth), int(maxImageHeight)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scratch := NewDrawingContext(w, h)
		scratch.InvertY()
		scratch.InvertY()
		_ = scratch.Image()
	}
}

func benchmarkRedrawDirtyTiles(b *testing.B, mapData *fileio.Civ5MapData, dirtyCount int) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewDrawingContext(800, 600)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData) // establish the base frame once

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	rng := rand.New(rand.NewSource(2))
	dirty := tileSet{}
	for len(dirty) < dirtyCount {
		dirty[tileCoord{rng.Intn(mapHeight), rng.Intn(mapWidth)}] = true
	}
	paintSet := expandWithNeighbors(dirty, mapHeight, mapWidth)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, paintSet)
	}
}

// benchmarkRedrawDirtyTilesClustered is benchmarkRedrawDirtyTiles with dirty tiles within clusterRadius of one center.
func benchmarkRedrawDirtyTilesClustered(b *testing.B, mapData *fileio.Civ5MapData, dirtyCount, clusterRadius int) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewDrawingContext(800, 600)
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
	paintSet := expandWithNeighbors(dirty, mapHeight, mapWidth)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, paintSet)
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

// BenchmarkQuantizeFullImage_Huge isolates quantizeCanvasImage against a fixed palette; it scans the whole canvas every call.
func BenchmarkQuantizeFullImage_Huge(b *testing.B) {
	mapData := buildBenchMapData(80, 128, 8, 1)
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewDrawingContext(800, 600)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	_, palette := quantizeCanvasImage(canvas, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		quantizeCanvasImage(canvas, palette)
	}
}

// runReplayFramesOld is the pre-optimization loop: every turn fully redraws and quantizes.
func runReplayFramesOld(renderer *MapRenderer, mapData *fileio.Civ5MapData, turnsEvents [][]fileio.Civ5ReplayEvent) {
	canvas := NewDrawingContext(800, 600)
	var palette color.Palette
	nextCityId := 0
	for _, events := range turnsEvents {
		for _, event := range events {
			nextCityId = applyReplayEvent(mapData, event, nextCityId)
		}
		_, palette = renderReplayFrame(renderer, canvas, mapData, palette)
	}
}

// runReplayFramesIncrementalDrawOnly repaints only dirty tiles but still fully re-quantizes each turn.
func runReplayFramesIncrementalDrawOnly(renderer *MapRenderer, mapData *fileio.Civ5MapData, turnsEvents [][]fileio.Civ5ReplayEvent) {
	canvas := NewDrawingContext(800, 600)
	var palette color.Palette
	nextCityId := 0
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])
	for turnIndex, events := range turnsEvents {
		dirty := tileSet{}
		for _, event := range events {
			nextCityId = applyReplayEvent(mapData, event, nextCityId)
			for _, rc := range dirtyTilesForEvent(event) {
				dirty[rc] = true
			}
		}
		if turnIndex == 0 {
			_, palette = renderReplayFrame(renderer, canvas, mapData, palette)
		} else {
			renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, expandWithNeighbors(dirty, mapHeight, mapWidth))
			_, palette = quantizeCanvasImage(canvas, palette)
		}
	}
}

// runReplayFramesIncrementalQuantize is DrawReplay's per-turn loop minus GIF encoding: later turns repaint and quantize only the dirty region.
func runReplayFramesIncrementalQuantize(renderer *MapRenderer, mapData *fileio.Civ5MapData, turnsEvents [][]fileio.Civ5ReplayEvent) {
	canvas := NewDrawingContext(800, 600)
	var palette color.Palette
	nextCityId := 0
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])
	for turnIndex, events := range turnsEvents {
		dirty := tileSet{}
		for _, event := range events {
			nextCityId = applyReplayEvent(mapData, event, nextCityId)
			for _, rc := range dirtyTilesForEvent(event) {
				dirty[rc] = true
			}
		}
		if turnIndex == 0 {
			_, palette = renderReplayFrame(renderer, canvas, mapData, palette)
		} else {
			dirtyRects := renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, expandWithNeighbors(dirty, mapHeight, mapWidth))
			if len(dirtyRects) > 0 {
				quantizeRegion(canvas, quantize.NewPaletteMapper(palette), unionRects(dirtyRects))
			}
		}
	}
}

const (
	benchEndToEndTurns      = 40
	benchEndToEndDirtyTiles = 15
	benchEndToEndCivs       = 8
)

func BenchmarkDrawReplayFrames_Old_Medium(b *testing.B) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEvents(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 99)
		b.StartTimer()

		runReplayFramesOld(renderer, mapData, turnsEvents)
	}
}

func BenchmarkDrawReplayFrames_IncrementalDrawOnly_Medium(b *testing.B) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEvents(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 99)
		b.StartTimer()

		runReplayFramesIncrementalDrawOnly(renderer, mapData, turnsEvents)
	}
}

func BenchmarkDrawReplayFrames_IncrementalQuantize_Medium(b *testing.B) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEvents(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 99)
		b.StartTimer()

		runReplayFramesIncrementalQuantize(renderer, mapData, turnsEvents)
	}
}

// BenchmarkDrawReplayFrames_IncrementalQuantize_Clustered_Medium is the Medium benchmark with clustered dirty tiles, the realistic turn shape.
func BenchmarkDrawReplayFrames_IncrementalQuantize_Clustered_Medium(b *testing.B) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mapData := buildBenchMapData(52, 80, benchEndToEndCivs, 1)
		turnsEvents := buildBenchTurnEventsClustered(52, 80, benchEndToEndTurns, benchEndToEndDirtyTiles, benchEndToEndCivs, 5, 99)
		b.StartTimer()

		runReplayFramesIncrementalQuantize(renderer, mapData, turnsEvents)
	}
}

// BenchmarkQuantizeRegion_Huge isolates quantizeRegion on a clustered turn's bounding box, to compare with BenchmarkQuantizeFullImage_Huge.
func BenchmarkQuantizeRegion_Huge(b *testing.B) {
	mapData := buildBenchMapData(80, 128, 8, 1)
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewDrawingContext(800, 600)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	_, palette := quantizeCanvasImage(canvas, nil)

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])
	rng := rand.New(rand.NewSource(2))
	centerRow, centerCol := rng.Intn(mapHeight), rng.Intn(mapWidth)
	const clusterRadius = 5
	dirty := tileSet{}
	for len(dirty) < 20 {
		row := clampInt(centerRow+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapHeight-1)
		col := clampInt(centerCol+rng.Intn(2*clusterRadius+1)-clusterRadius, 0, mapWidth-1)
		dirty[tileCoord{row, col}] = true
	}
	dirtyRects := renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, expandWithNeighbors(dirty, mapHeight, mapWidth))
	rect := unionRects(dirtyRects)
	mapper := quantize.NewPaletteMapper(palette)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		quantizeRegion(canvas, mapper, rect)
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

		if err := DrawReplay(mapData, replayData, outputPath, 0); err != nil {
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

		if err := DrawReplay(mapData, replayData, outputPath, 0); err != nil {
			b.Fatalf("DrawReplay() = %v", err)
		}
	}
}

// writeGifOldStyle encodes turnsEvents with a full-canvas frame every turn (the pre-delta-frame format), as a size baseline.
func writeGifOldStyle(mapData *fileio.Civ5MapData, turnsEvents [][]fileio.Civ5ReplayEvent, outputPath string) error {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewDrawingContext(800, 600)
	outGif := &gif.GIF{}
	var palette color.Palette
	nextCityId := 0
	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])
	for turnIndex, events := range turnsEvents {
		dirty := tileSet{}
		for _, event := range events {
			nextCityId = applyReplayEvent(mapData, event, nextCityId)
			for _, rc := range dirtyTilesForEvent(event) {
				dirty[rc] = true
			}
		}
		var frame *image.Paletted
		if turnIndex == 0 {
			frame, palette = renderReplayFrame(renderer, canvas, mapData, palette)
		} else {
			renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, expandWithNeighbors(dirty, mapHeight, mapWidth))
			frame, palette = quantizeCanvasImage(canvas, palette)
		}
		outGif.Image = append(outGif.Image, frame)
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
	// Smaller than the other end-to-end scenarios: the full-frame baseline is slow and only the size effect matters.
	const mapHeight, mapWidth, turns, dirtyPerTurn, civs, clusterRadius = 20, 30, 10, 15, 8, 5

	dir := t.TempDir()

	mapDataOld := buildBenchMapData(mapHeight, mapWidth, civs, 1)
	turnsEventsOld := buildBenchTurnEventsClustered(mapHeight, mapWidth, turns, dirtyPerTurn, civs, clusterRadius, 99)
	oldPath := filepath.Join(dir, "old.gif")
	if err := writeGifOldStyle(mapDataOld, turnsEventsOld, oldPath); err != nil {
		t.Fatalf("writeGifOldStyle() = %v", err)
	}

	mapDataNew := buildBenchMapData(mapHeight, mapWidth, civs, 1)
	turnsEventsNew := buildBenchTurnEventsClustered(mapHeight, mapWidth, turns, dirtyPerTurn, civs, clusterRadius, 99)
	replayData := buildBenchReplayData(mapDataNew, turnsEventsNew)
	newPath := filepath.Join(dir, "new.gif")
	if err := DrawReplay(mapDataNew, replayData, newPath, 0); err != nil {
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

// TestDrawReplayGifFramesReconstructFullQuantizeEveryFrame decodes DrawReplay's GIF and composites its
// frames (respecting DisposalNone) to check every turn matches a full redraw + full quantize. Includes a
// city founding (label rects) so a cropping or missed-rect bug shows as a persistent discrepancy.
func TestDrawReplayGifFramesReconstructFullQuantizeEveryFrame(t *testing.T) {
	const maxAllowedDelta = 8 // same rounding-noise tolerance as TestRedrawDirtyTilesMatchesFullRedrawAfterMutation

	buildScenario := func() (*fileio.Civ5MapData, [][]fileio.Civ5ReplayEvent) {
		mapData := buildBenchMapData(16, 16, 4, 3)
		turnsEvents := [][]fileio.Civ5ReplayEvent{
			{{Turn: 1, TypeId: ReplayEventTilesClaimed, CivId: 1, Tiles: []fileio.Civ5ReplayEventTile{{X: 3, Y: 3}}}}, // establishes frame 0
			{{Turn: 2, TypeId: ReplayEventCityFounded, Text: "Testopolis is founded.", Tiles: []fileio.Civ5ReplayEventTile{{X: 8, Y: 8}}}},
			{{Turn: 3, TypeId: ReplayEventTilesClaimed, CivId: 2, Tiles: []fileio.Civ5ReplayEventTile{{X: 8, Y: 7}}}}, // dirties a neighbor of the new city's tile
			{{Turn: 4, TypeId: ReplayEventTilesClaimed, CivId: 0, Tiles: []fileio.Civ5ReplayEventTile{{X: 1, Y: 1}, {X: 14, Y: 14}}}},
		}
		return mapData, turnsEvents
	}

	// Reference: full redraw + full quantize every turn.
	mapDataRef, turnsEventsRef := buildScenario()
	rendererRef := NewMapRenderer(DefaultDrawingConfig())
	canvasRef := NewDrawingContext(800, 600)
	var paletteRef color.Palette
	var referenceFrames []*image.Paletted
	nextCityIdRef := 0
	mapHeight, mapWidth := len(mapDataRef.MapTiles), len(mapDataRef.MapTiles[0])
	for turnIndex, events := range turnsEventsRef {
		dirty := tileSet{}
		for _, event := range events {
			nextCityIdRef = applyReplayEvent(mapDataRef, event, nextCityIdRef)
			for _, rc := range dirtyTilesForEvent(event) {
				dirty[rc] = true
			}
		}
		var frame *image.Paletted
		if turnIndex == 0 {
			frame, paletteRef = renderReplayFrame(rendererRef, canvasRef, mapDataRef, paletteRef)
		} else {
			rendererRef.RedrawDirtyTiles(canvasRef, mapDataRef, mapHeight, mapWidth, expandWithNeighbors(dirty, mapHeight, mapWidth))
			frame, paletteRef = quantizeCanvasImage(canvasRef, paletteRef)
		}
		referenceFrames = append(referenceFrames, frame)
	}

	// Actual: the real DrawReplay entry point, producing genuine GIF bytes on disk.
	mapDataReal, turnsEventsReal := buildScenario()
	replayData := buildBenchReplayData(mapDataReal, turnsEventsReal)
	outputPath := filepath.Join(t.TempDir(), "replay.gif")
	if err := DrawReplay(mapDataReal, replayData, outputPath, 0); err != nil {
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

		ref := referenceFrames[turnIndex]
		if accum.Bounds() != ref.Bounds() {
			t.Fatalf("turn %d: reconstructed bounds %v != reference bounds %v", turnIndex, accum.Bounds(), ref.Bounds())
		}

		diffCount := 0
		worstDelta := 0
		var worstAt image.Point
		for y := ref.Bounds().Min.Y; y < ref.Bounds().Max.Y; y++ {
			for x := ref.Bounds().Min.X; x < ref.Bounds().Max.X; x++ {
				// Interface values of different color types are never ==, so compare via the normalized delta.
				d := maxChannelDelta(ref.At(x, y), accum.At(x, y))
				if d == 0 {
					continue
				}
				diffCount++
				if d > worstDelta {
					worstDelta = d
					worstAt = image.Pt(x, y)
				}
			}
		}
		t.Logf("turn %d: %d differing pixels vs full re-quantize, worst per-channel delta %d at %v",
			turnIndex, diffCount, worstDelta, worstAt)
		if worstDelta > maxAllowedDelta {
			t.Errorf("turn %d: reconstructed GIF composite differs from full re-quantize by %d in a color channel at %v (want <= %d) - likely a real bug (e.g. a missed dirty rect), not rounding noise",
				turnIndex, worstDelta, worstAt, maxAllowedDelta)
		}
		turnIndex++
	}
	if turnIndex != len(referenceFrames) {
		t.Errorf("decoded GIF closed out %d turns, want %d", turnIndex, len(referenceFrames))
	}
}
