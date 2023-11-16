package graphics

import (
	"fmt"
	"image"
	"image/gif"
	"os"
	"sort"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/quantize"
)

const (
	GIF_DELAY = 100
)

func DrawReplay(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData, outputFilename string) {
	outGif := &gif.GIF{}

	replayTurns := fileio.GroupEventsByTurn(replayData.AllReplayEvents)
	turnNumbers := make([]int, 0)
	for turn := range replayTurns {
		turnNumbers = append(turnNumbers, turn)
	}
	sort.Ints(turnNumbers)

	// set civ color and civ index map before loading replay turns
	fmt.Println("Player Civ:", replayData.PlayerCiv)
	for i := 0; i < len(replayData.AllCivs); i++ {
		fmt.Println("Index", i, ", civ data:", replayData.AllCivs[i])
		mapData.CityOwnerIndexMap[i] = i
	}

	if len(mapData.Civ5PlayerData) == 0 {
		for i := 0; i < len(replayData.AllCivs); i++ {
			civName := replayData.AllCivs[i].Name
			civName = strings.ReplaceAll(civName, " ", "")

			mapData.Civ5PlayerData = append(mapData.Civ5PlayerData, &fileio.Civ5PlayerData{
				Index:     i,
				CivType:   fmt.Sprintf("CIVILIZATION_%s", strings.ToUpper(civName)),
				TeamColor: fmt.Sprintf("PLAYERCOLOR_%s", strings.ToUpper(civName)),
			},
			)
		}
	} else {
		indexPlayerCivilization := -1

		for i := 0; i < len(mapData.Civ5PlayerData); i++ {
			if mapData.Civ5PlayerData[i].CivType == replayData.PlayerCiv {
				indexPlayerCivilization = i
				break
			}
		}

		// The replay file sets the player's civId to 0, but the original civId is usually a different value
		// Swap values to ensure the correct color is assigned
		fmt.Println("Player civilization index:", indexPlayerCivilization)
		if indexPlayerCivilization != -1 {
			temp := mapData.Civ5PlayerData[0]
			mapData.Civ5PlayerData[0] = mapData.Civ5PlayerData[indexPlayerCivilization]
			mapData.Civ5PlayerData[indexPlayerCivilization] = temp
		}
	}

	maxCityId := 0
	for _, turn := range turnNumbers {
		fmt.Println(fmt.Sprintf("Drawing frame for turn %d...", turn))

		for i, event := range replayTurns[turn] {
			fmt.Println("Replay event", i, ":", event)

			if event.TypeId == 1 {
				// City founded
				// Set city id
				for _, tile := range event.Tiles {
					mapData.MapTileImprovements[tile.Y][tile.X].CityId = maxCityId
					mapData.MapTileImprovements[tile.Y][tile.X].CityName = event.Text[0 : len(event.Text)-len(" is founded.")]
					maxCityId += 1
				}
			} else if event.TypeId == 2 {
				// Tiles claimed
				// Change owner to new civ id
				for _, tile := range event.Tiles {
					mapData.MapTileImprovements[tile.Y][tile.X].Owner = event.CivId
				}
			} else if event.TypeId == 3 {
				// City transferred to another civ
				for _, tile := range event.Tiles {
					mapData.MapTileImprovements[tile.Y][tile.X].Owner = event.CivId
				}
			}
		}

		mapImage := DrawPoliticalMap(mapData)
		bounds := mapImage.Bounds()

		palettedImage := image.NewPaletted(bounds, nil)
		quantizer := quantize.MedianCutQuantizer{NumColor: 256}
		quantizer.Quantize(palettedImage, bounds, mapImage, image.ZP)

		outGif.Image = append(outGif.Image, palettedImage)
		outGif.Delay = append(outGif.Delay, GIF_DELAY)
	}

	outputFile, _ := os.OpenFile(outputFilename, os.O_WRONLY|os.O_CREATE, 0600)
	defer outputFile.Close()
	gif.EncodeAll(outputFile, outGif)

	fmt.Println("Saved replay to", outputFilename)
}