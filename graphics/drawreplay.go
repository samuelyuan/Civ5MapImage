package graphics

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/quantize"
)

const (
	GIF_DELAY = 100

	// replayNoTileSentinel (0xFFFF) is the (X, Y) value a locationless event carries instead of a
	// real tile - see replayEventCanBeLocationless.
	replayNoTileSentinel = 65535
)

// Replay event type ids, as encoded in Civ5ReplayEvent.TypeId. applyReplayEvent below only acts on
// CityFounded/TilesClaimed/CityTransferred/TilesRazed; the rest are left unhandled.
const (
	ReplayEventNotification    = 0
	ReplayEventCityFounded     = 1
	ReplayEventTilesClaimed    = 2
	ReplayEventCityTransferred = 3
	ReplayEventTilesRazed      = 4
	ReplayEventReligionFounded = 5 // "X has founded the new religion Y in the holy city of Z." - real tile (Z).
	ReplayEventPantheonFounded = 6 // "X has started worshipping a pantheon of gods..." - civ-level, no tile.
)

// replayEventCanBeLocationless reports whether typeId can carry replayNoTileSentinel instead of a
// real tile. Every other TypeId always has a real, in-bounds tile.
func replayEventCanBeLocationless(typeId int) bool {
	return typeId == ReplayEventNotification || typeId == ReplayEventPantheonFounded
}

// Helper function to setup civ player data from replay
func setupCivPlayerData(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData) {
	if len(mapData.Civ5PlayerData) == 0 || !replayData.IsReplayFile {
		fmt.Println("Rebuilding civ player data from replay file...")
		mapData.Civ5PlayerData = make([]*fileio.Civ5PlayerData, 0)
		for i := 0; i < len(replayData.AllCivs); i++ {
			civName := replayData.AllCivs[i].Name

			if strings.Contains(civName, "CIVILIZATION") || strings.Contains(civName, "MINOR_CIV") {
				mapData.Civ5PlayerData = append(mapData.Civ5PlayerData, &fileio.Civ5PlayerData{
					Index:     i,
					CivType:   civName,
					TeamColor: replayData.AllCivs[i].LongName,
				})
			} else {
				civName = strings.ReplaceAll(civName, " ", "")
				mapData.Civ5PlayerData = append(mapData.Civ5PlayerData, &fileio.Civ5PlayerData{
					Index:     i,
					CivType:   fmt.Sprintf("CIVILIZATION_%s", strings.ToUpper(civName)),
					TeamColor: fmt.Sprintf("PLAYERCOLOR_%s", strings.ToUpper(civName)),
				})
			}
		}
	} else {
		// Swap player civilization to index 0
		indexPlayerCivilization := -1
		for i := 0; i < len(mapData.Civ5PlayerData); i++ {
			if mapData.Civ5PlayerData[i].CivType == replayData.PlayerCiv {
				indexPlayerCivilization = i
				break
			}
		}

		fmt.Println("Player civilization index:", indexPlayerCivilization)
		if indexPlayerCivilization != -1 {
			temp := mapData.Civ5PlayerData[0]
			mapData.Civ5PlayerData[0] = mapData.Civ5PlayerData[indexPlayerCivilization]
			mapData.Civ5PlayerData[indexPlayerCivilization] = temp
		}
	}
}

// ValidateReplayCompatibility checks that mapData and replayData are consistent enough to
// render a replay animation, returning a descriptive error if not. Calling this up front lets
// DrawReplay fail fast with a clear message instead of panicking partway through a potentially
// long render loop (e.g. because the replay was generated from a different map).
func ValidateReplayCompatibility(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData) error {
	if mapData == nil {
		return fmt.Errorf("map data is nil")
	}
	if replayData == nil {
		return fmt.Errorf("replay data is nil")
	}
	if len(mapData.MapTiles) == 0 || len(mapData.MapTiles[0]) == 0 {
		return fmt.Errorf("map has no tiles to draw")
	}
	if len(mapData.MapTileImprovements) == 0 {
		return fmt.Errorf("map is missing tile improvement data required to render a replay (was it exported without game data?)")
	}
	if len(replayData.AllReplayEvents) == 0 {
		return fmt.Errorf("replay has no events to animate")
	}

	mapHeight := len(mapData.MapTileImprovements)
	mapWidth := len(mapData.MapTileImprovements[0])

	// 0 for replays that don't carry their own dimensions (e.g. one converted from a .civ5save).
	if replayData.MapWidth > 0 && replayData.MapHeight > 0 {
		if replayData.MapWidth != mapWidth || replayData.MapHeight != mapHeight {
			return fmt.Errorf("replay was recorded on a %dx%d map, but the provided map is %dx%d; make sure -map points to the matching .Civ5Map file",
				replayData.MapWidth, replayData.MapHeight, mapWidth, mapHeight)
		}
	}

	fmt.Println("\n=== Validating replay events ===")
	eventTypeCounts := map[int]int{}
	for _, event := range replayData.AllReplayEvents {
		eventTypeCounts[event.TypeId]++
	}
	fmt.Printf("Validated %d/%d replay events\n", len(replayData.AllReplayEvents), len(replayData.AllReplayEvents))
	for _, typeId := range fileio.GetSortedKeys(eventTypeCounts) {
		fmt.Printf("  %d of TypeId %d (%s)\n", eventTypeCounts[typeId], typeId, replayEventTypeName(typeId))
	}

	for _, event := range replayData.AllReplayEvents {
		for _, tile := range event.Tiles {
			if replayEventCanBeLocationless(event.TypeId) && tile.X == replayNoTileSentinel && tile.Y == replayNoTileSentinel {
				continue
			}
			if tile.Y < 0 || tile.Y >= mapHeight || tile.X < 0 || tile.X >= mapWidth {
				return fmt.Errorf("replay event on turn %d (TypeId %d) references tile (%d, %d), which is outside the map bounds (%dx%d); the replay may not match this map",
					event.Turn, event.TypeId, tile.X, tile.Y, mapWidth, mapHeight)
			}
		}
	}
	fmt.Println()

	return nil
}

func replayEventTypeName(typeId int) string {
	switch typeId {
	case ReplayEventNotification:
		return "notification"
	case ReplayEventCityFounded:
		return "CityFounded"
	case ReplayEventTilesClaimed:
		return "TilesClaimed"
	case ReplayEventCityTransferred:
		return "CityTransferred"
	case ReplayEventTilesRazed:
		return "TilesRazed"
	case ReplayEventReligionFounded:
		return "ReligionFounded"
	case ReplayEventPantheonFounded:
		return "PantheonFounded"
	default:
		return "unhandled"
	}
}

// applyReplayEvent applies a single replay event's effect to the map tile improvements,
// mutating mapData in place. nextCityId is the city id to assign if this event founds a
// new city; the (possibly incremented) next available city id is returned so the caller
// can thread it into the next call.
func applyReplayEvent(mapData *fileio.Civ5MapData, event fileio.Civ5ReplayEvent, nextCityId int) int {
	switch event.TypeId {
	case ReplayEventCityFounded:
		// Set city id
		for _, tile := range event.Tiles {
			mapData.MapTileImprovements[tile.Y][tile.X].CityId = nextCityId
			mapData.MapTileImprovements[tile.Y][tile.X].CityName = strings.TrimSuffix(event.Text, " is founded.")
			nextCityId += 1
		}
	case ReplayEventTilesClaimed, ReplayEventCityTransferred:
		// Change owner to new civ id
		for _, tile := range event.Tiles {
			mapData.MapTileImprovements[tile.Y][tile.X].Owner = event.CivId
		}
	case ReplayEventTilesRazed:
		for _, tile := range event.Tiles {
			// Remove city from map
			mapData.MapTileImprovements[tile.Y][tile.X].Owner = -1
			mapData.MapTileImprovements[tile.Y][tile.X].CityId = -1
			mapData.MapTileImprovements[tile.Y][tile.X].CityName = ""
			// Set razed city tile to road
			mapData.MapTileImprovements[tile.Y][tile.X].RouteType = 2
		}
	}
	return nextCityId
}

// resetCityOwnerIndexMap rebuilds mapData.CityOwnerIndexMap as an identity mapping over the
// replay's civs (civ index i maps to itself). This differs from how map/save loading builds
// the same field (buildCityOwnerMaps maps a raw file owner slot to a compact player array
// index) because replay events already reference civs using the replay's own AllCivs ordering,
// so no remapping is needed here — just an identity table sized to match. This also guards
// against mapData.CityOwnerIndexMap being nil (e.g. a map round-tripped through JSON without
// game data), which would otherwise panic on the first write below.
func resetCityOwnerIndexMap(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData) {
	if mapData.CityOwnerIndexMap == nil {
		mapData.CityOwnerIndexMap = make(map[int]int)
	}
	for i := 0; i < len(replayData.AllCivs); i++ {
		fmt.Println("Index", i, ", civ data:", replayData.AllCivs[i])
		mapData.CityOwnerIndexMap[i] = i
	}
}

// dirtyTilesForEvent returns the (row, col) tiles applyReplayEvent mutates for event; keep in sync with its switch.
func dirtyTilesForEvent(event fileio.Civ5ReplayEvent) []tileCoord {
	switch event.TypeId {
	case ReplayEventCityFounded, ReplayEventTilesClaimed, ReplayEventCityTransferred, ReplayEventTilesRazed:
		tiles := make([]tileCoord, len(event.Tiles))
		for i, tile := range event.Tiles {
			tiles[i] = tileCoord{tile.Y, tile.X}
		}
		return tiles
	default:
		return nil
	}
}

// expandWithNeighbors adds each dirty tile's in-bounds hex neighbors: a tile's border/road depends
// on its neighbors' state, so they must be repainted too.
func expandWithNeighbors(dirty tileSet, mapHeight, mapWidth int) tileSet {
	expanded := make(tileSet, len(dirty)*3)
	for rc := range dirty {
		expanded[rc] = true
		for _, n := range fileio.GetNeighbors(rc.col, rc.row) {
			nx, ny := n[0], n[1]
			if nx >= 0 && ny >= 0 && nx < mapWidth && ny < mapHeight {
				expanded[tileCoord{ny, nx}] = true
			}
		}
	}
	return expanded
}

// quantizeCanvasImage quantizes canvas into a paletted GIF frame. Pass nil palette to compute a fresh
// one; pass the returned palette back on later calls so the animation doesn't flicker between palettes.
func quantizeCanvasImage(canvas Canvas, palette color.Palette) (*image.Paletted, color.Palette) {
	mapImage := canvasRGBA(canvas)
	bounds := mapImage.Bounds()
	if palette == nil {
		palette = quantize.BuildPalette(mapImage, bounds, 256)
	}
	palettedImage := image.NewPaletted(bounds, palette)
	quantize.NewPaletteMapper(palette).Fill(palettedImage, bounds, mapImage, bounds.Min)
	return palettedImage, palette
}

// canvasRGBA returns the canvas's pixels; the quantizer reads *image.RGBA directly.
func canvasRGBA(canvas Canvas) *image.RGBA {
	return canvas.Image().(*image.RGBA)
}

// clusterGap (pixels): rects closer than this merge in clusterRects - enough to bridge adjacent tiles, not distant clusters.
const clusterGap = 8

// clusterRects merges rects within clusterGap of each other into disjoint bounding boxes, so
// scattered dirty tiles don't collapse into one box spanning everything in between.
func clusterRects(rects []image.Rectangle) []image.Rectangle {
	clusters := append([]image.Rectangle(nil), rects...)
	for {
		merged := false
		for i := 0; i < len(clusters) && !merged; i++ {
			for j := i + 1; j < len(clusters); j++ {
				if clusters[i].Inset(-clusterGap).Overlaps(clusters[j]) {
					clusters[i] = clusters[i].Union(clusters[j])
					clusters = append(clusters[:j], clusters[j+1:]...)
					merged = true
					break
				}
			}
		}
		if !merged {
			return clusters
		}
	}
}

// quantizeRegion quantizes just rect of canvas into a small *image.Paletted positioned at rect -
// a GIF frame that, with DisposalNone, leaves everything outside rect as earlier frames drew it.
func quantizeRegion(canvas Canvas, mapper *quantize.PaletteMapper, rect image.Rectangle) *image.Paletted {
	dst := image.NewPaletted(rect, mapper.Palette())
	mapper.Fill(dst, rect, canvasRGBA(canvas), rect.Min)
	return dst
}

// renderReplayFrame fully redraws mapData (tile-major, matching RedrawDirtyTiles's rendering) and
// quantizes it. Only the first frame needs this; later frames repaint just what changed.
func renderReplayFrame(renderer *MapRenderer, canvas Canvas, mapData *fileio.Civ5MapData, palette color.Palette) (*image.Paletted, color.Palette) {
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	return quantizeCanvasImage(canvas, palette)
}

// appendFirstGifFrame appends turn 0's full-size frame and returns the palette it fixes for all later frames.
func appendFirstGifFrame(outGif *gif.GIF, renderer *MapRenderer, canvas Canvas, mapData *fileio.Civ5MapData) color.Palette {
	palettedImage, palette := renderReplayFrame(renderer, canvas, mapData, nil)
	outGif.Image = append(outGif.Image, palettedImage)
	outGif.Delay = append(outGif.Delay, GIF_DELAY)
	outGif.Disposal = append(outGif.Disposal, gif.DisposalNone)
	return palette
}

// appendGifFrames repaints dirty's tiles and appends one GIF block per changed region (or one 1x1
// no-op if nothing changed). All blocks but the last have delay 0, so a turn advances the
// animation by exactly GIF_DELAY.
func appendGifFrames(outGif *gif.GIF, renderer *MapRenderer, canvas Canvas, mapData *fileio.Civ5MapData, mapHeight, mapWidth int, mapper *quantize.PaletteMapper, dirty tileSet) {
	dirtyRects := renderer.RedrawDirtyTiles(canvas, mapData, mapHeight, mapWidth, expandWithNeighbors(dirty, mapHeight, mapWidth))

	regions := []image.Rectangle{image.Rect(0, 0, 1, 1)}
	if len(dirtyRects) > 0 {
		regions = clusterRects(dirtyRects)
	}

	for i, rect := range regions {
		outGif.Image = append(outGif.Image, quantizeRegion(canvas, mapper, rect))
		delay := 0
		if i == len(regions)-1 {
			delay = GIF_DELAY
		}
		outGif.Delay = append(outGif.Delay, delay)
		outGif.Disposal = append(outGif.Disposal, gif.DisposalNone)
	}
}

// DrawReplay renders the given map/replay pair into an animated GIF at outputFilename.
// It returns an error (rather than panicking) if the map and replay are incompatible, or if
// the output file cannot be written. maxTurns caps how many turns are rendered (0 = all) - use
// it for a quick preview/test render instead of waiting on a full, possibly long, animation.
func DrawReplay(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData, outputFilename string, maxTurns int) error {
	if err := ValidateReplayCompatibility(mapData, replayData); err != nil {
		return fmt.Errorf("replay is not compatible with map: %w", err)
	}

	outGif := &gif.GIF{}

	replayTurns := fileio.GroupEventsByTurn(replayData.AllReplayEvents)
	turnNumbers := fileio.GetSortedKeys(replayTurns)
	if maxTurns > 0 && maxTurns < len(turnNumbers) {
		fmt.Printf("Limiting render to the first %d of %d turns (-maxturns)\n", maxTurns, len(turnNumbers))
		turnNumbers = turnNumbers[:maxTurns]
	}

	// Setup civ data and player mapping
	fmt.Println("Player Civ:", replayData.PlayerCiv)
	resetCityOwnerIndexMap(mapData, replayData)
	setupCivPlayerData(mapData, replayData)

	maxCityId := 0
	var mapPalette color.Palette
	var paletteMapper *quantize.PaletteMapper

	// Initialize canvas and renderer once outside the loop
	config := DefaultDrawingConfig()
	renderer := NewMapRenderer(config)
	canvas := NewDrawingContext(800, 600) // Will be resized by renderer

	mapHeight := len(mapData.MapTiles)
	mapWidth := len(mapData.MapTiles[0])

	for turnIndex, turn := range turnNumbers {
		fmt.Printf("Drawing frame for turn %d...\n", turn)

		dirty := tileSet{}
		for i, event := range replayTurns[turn] {
			fmt.Println("Replay event", i, ":", event)
			maxCityId = applyReplayEvent(mapData, event, maxCityId)
			for _, rc := range dirtyTilesForEvent(event) {
				dirty[rc] = true
			}
		}

		fmt.Println("Drawing map for turn", turn)

		if turnIndex == 0 {
			mapPalette = appendFirstGifFrame(outGif, renderer, canvas, mapData)
			paletteMapper = quantize.NewPaletteMapper(mapPalette)
			continue
		}
		appendGifFrames(outGif, renderer, canvas, mapData, mapHeight, mapWidth, paletteMapper, dirty)
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
