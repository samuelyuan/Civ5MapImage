package graphics

import (
	"fmt"
	"image/gif"
	"os"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

const (
	GIF_DELAY = 100
)

// progressInterval is how many turns pass between progress lines.
const progressInterval = 10

// DrawReplay renders a map/replay pair readied by fileio.PrepareReplay into a GIF at outputFilename; maxTurns 0 renders all turns.
func DrawReplay(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData, outputFilename string, maxTurns int) error {
	outGif := &gif.GIF{}

	replayTurns := fileio.GroupEventsByTurn(replayData.AllReplayEvents)
	turnNumbers := fileio.GetSortedKeys(replayTurns)
	if maxTurns > 0 && maxTurns < len(turnNumbers) {
		fmt.Printf("Limiting render to the first %d of %d turns (-maxturns)\n", maxTurns, len(turnNumbers))
		turnNumbers = turnNumbers[:maxTurns]
	}

	maxCityId := 0

	canvas := raster.NewPalettedCanvas(800, 600, replayPalette(mapData)) // resized by drawPoliticalMapTileMajor

	grid := buildTileGrid(mapData.Size(), tileRadius) // the map size never changes during a replay

	var tracker *tileTracker
	for turnIndex, turn := range turnNumbers {
		if turnIndex%progressInterval == 0 || turnIndex == len(turnNumbers)-1 {
			fmt.Printf("Drawing turn %d (%d of %d)...\n", turn, turnIndex+1, len(turnNumbers))
		}

		for _, event := range replayTurns[turn] {
			maxCityId = fileio.ApplyReplayEvent(mapData, event, maxCityId)
		}

		if turnIndex == 0 {
			tracker = newTileTracker(mapData)
			drawPoliticalMapTileMajor(canvas, mapData, grid)
			addFrame(outGif, canvas.Snapshot(canvas.Image().Bounds()), GIF_DELAY)
			continue
		}
		appendGifFrames(outGif, canvas, mapData, grid, tracker.takeChanges())
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
