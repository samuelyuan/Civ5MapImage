package graphics

import (
	"cmp"
	"fmt"
	"image"
	"image/gif"
	"os"
	"slices"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

const (
	GIF_DELAY = 100
)

// tileTracker reports which tiles' records changed since it last looked; comparing state can't miss an event's effect.
type tileTracker struct {
	mapData *fileio.Civ5MapData
	prev    []fileio.Civ5MapTileImprovement
}

// newTileTracker starts tracking from mapData's current tile records.
func newTileTracker(mapData *fileio.Civ5MapData) *tileTracker {
	t := &tileTracker{mapData: mapData, prev: make([]fileio.Civ5MapTileImprovement, 0, len(mapData.MapTileImprovements)*len(mapData.MapTileImprovements[0]))}
	for _, row := range mapData.MapTileImprovements {
		for _, tile := range row {
			t.prev = append(t.prev, *tile)
		}
	}
	return t
}

// takeChanges returns the tiles whose record differs from the last call (or from newTileTracker) and makes the current state the new baseline.
func (t *tileTracker) takeChanges() tileSet {
	changed := tileSet{}
	i := 0
	for row, tiles := range t.mapData.MapTileImprovements {
		for col, tile := range tiles {
			if *tile != t.prev[i] {
				changed[tileCoord{row, col}] = true
				t.prev[i] = *tile
			}
			i++
		}
	}
	return changed
}

// clusterGap is the pixel distance under which clusterRects merges rects.
const clusterGap = 8

// clusterRects merges rects within clusterGap into disjoint bounding boxes, so scattered tiles don't become one huge box; empty rects are dropped.
// The result is the same for any input order, sorted by top-left corner (top to bottom, then left to right).
func clusterRects(rects []image.Rectangle) []image.Rectangle {
	var clusters []image.Rectangle // never within clusterGap of each other
	pending := slices.Clone(rects)
	for len(pending) > 0 {
		r := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if r.Empty() {
			continue
		}
		near := func(c image.Rectangle) bool { return r.Inset(-clusterGap).Overlaps(c) }
		if i := slices.IndexFunc(clusters, near); i >= 0 {
			pending = append(pending, r.Union(clusters[i])) // the bigger box may now reach other clusters
			clusters = slices.Delete(clusters, i, i+1)
			continue
		}
		clusters = append(clusters, r)
	}
	slices.SortFunc(clusters, func(a, b image.Rectangle) int {
		return cmp.Or(cmp.Compare(a.Min.Y, b.Min.Y), cmp.Compare(a.Min.X, b.Min.X))
	})
	return clusters
}

// noopRegion is a 1x1 frame, which keeps a turn's delay when nothing on screen changed.
var noopRegion = image.Rect(0, 0, 1, 1)

// frameRegions returns the regions to emit as GIF frames for dirtyRects: their clusters, or noopRegion if none is left.
func frameRegions(dirtyRects []image.Rectangle) []image.Rectangle {
	if regions := clusterRects(dirtyRects); len(regions) > 0 {
		return regions
	}
	return []image.Rectangle{noopRegion}
}

// appendGifFrames repaints the changed tiles and appends one GIF block per changed region (a 1x1 no-op if none); only the last has a delay.
func appendGifFrames(outGif *gif.GIF, renderer *MapRenderer, canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, changed tileSet) {
	dirtyRects := renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, changed)

	regions := frameRegions(dirtyRects)
	for i, rect := range regions {
		delay := 0
		if i == len(regions)-1 {
			delay = GIF_DELAY
		}
		addFrame(outGif, canvas.Snapshot(rect), delay)
	}
}

// addFrame appends frame with the given delay; DisposalNone leaves it on screen under later frames, which only cover their own regions.
func addFrame(outGif *gif.GIF, frame *image.Paletted, delay int) {
	outGif.Image = append(outGif.Image, frame)
	outGif.Delay = append(outGif.Delay, delay)
	outGif.Disposal = append(outGif.Disposal, gif.DisposalNone)
}

// DrawReplay renders a map/replay pair that fileio.PrepareReplay has readied into a GIF at outputFilename.
// maxTurns caps the turns rendered (0 = all). Frames are direct PalettedCanvas copies, with no quantizing.
func DrawReplay(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData, outputFilename string, maxTurns int) error {
	outGif := &gif.GIF{}

	replayTurns := fileio.GroupEventsByTurn(replayData.AllReplayEvents)
	turnNumbers := fileio.GetSortedKeys(replayTurns)
	if maxTurns > 0 && maxTurns < len(turnNumbers) {
		fmt.Printf("Limiting render to the first %d of %d turns (-maxturns)\n", maxTurns, len(turnNumbers))
		turnNumbers = turnNumbers[:maxTurns]
	}

	maxCityId := 0

	config := DefaultDrawingConfig()
	renderer := NewMapRenderer(config)
	canvas := raster.NewPalettedCanvas(800, 600, replayPalette(mapData)) // resized by the renderer

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	var tracker *tileTracker
	for turnIndex, turn := range turnNumbers {
		fmt.Printf("Drawing frame for turn %d...\n", turn)

		for i, event := range replayTurns[turn] {
			fmt.Println("Replay event", i, ":", event)
			maxCityId = fileio.ApplyReplayEvent(mapData, event, maxCityId)
		}

		fmt.Println("Drawing map for turn", turn)

		if turnIndex == 0 {
			tracker = newTileTracker(mapData)
			renderer.DrawPoliticalMapTileMajor(canvas, mapData)
			addFrame(outGif, canvas.Snapshot(canvas.Image().Bounds()), GIF_DELAY)
			continue
		}
		appendGifFrames(outGif, renderer, canvas, mapData, mapHeight, mapWidth, tracker.takeChanges())
	}

	outputFile, err := os.OpenFile(outputFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open output file %q: %w", outputFilename, err)
	}
	defer outputFile.Close()

	if err := gif.EncodeAll(outputFile, outGif); err != nil {
		return fmt.Errorf("failed to encode replay gif to %q: %w", outputFilename, err)
	}

	fmt.Println("Saved replay to", outputFilename)
	return nil
}
