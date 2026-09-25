package graphics

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// benchCivColors is a few real civColorMap keys, giving synthetic civs real territory colors and borders.
var benchCivColors = []string{
	"PLAYERCOLOR_BLACK", "PLAYERCOLOR_BLUE", "PLAYERCOLOR_BROWN", "PLAYERCOLOR_CYAN",
	"PLAYERCOLOR_DARK_BLUE", "PLAYERCOLOR_DARK_GREEN", "PLAYERCOLOR_GRAY", "PLAYERCOLOR_GREEN",
	"PLAYERCOLOR_ORANGE", "PLAYERCOLOR_PINK", "PLAYERCOLOR_PURPLE", "PLAYERCOLOR_RED",
}

// buildBenchMapData builds a deterministic synthetic political map: civs holding square territory blocks, rivers, roads and a city per block.
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
			{Turn: t, TypeId: fileio.ReplayEventTilesClaimed, CivId: rng.Intn(numCivs), Tiles: tiles},
		}
	}
	return turns
}

// buildBenchTurnEventsClustered is buildBenchTurnEvents with each turn's tiles within clusterRadius of one center.
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
			{Turn: t, TypeId: fileio.ReplayEventTilesClaimed, CivId: rng.Intn(numCivs), Tiles: tiles},
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

// unionRects returns the bounding rect of rects (panics if empty); test-only, to snapshot one region per turn.
func unionRects(rects []image.Rectangle) image.Rectangle {
	union := rects[0]
	for _, r := range rects[1:] {
		union = union.Union(r)
	}
	return union
}

// newBenchCanvas returns a canvas over mapData's exact replay palette, as DrawReplay uses.
func newBenchCanvas(mapData *fileio.Civ5MapData) *raster.PalettedCanvas {
	return raster.NewPalettedCanvas(800, 600, replayPalette(mapData))
}

// canvasDiffs counts the pixels where a and b differ and returns the first, or (-1, _) if their bounds differ.
func canvasDiffs(a, b *raster.PalettedCanvas) (count int, first image.Point) {
	if a.Image().Bounds() != b.Image().Bounds() {
		return -1, image.Point{}
	}
	for y := a.Image().Bounds().Min.Y; y < a.Image().Bounds().Max.Y; y++ {
		for x := a.Image().Bounds().Min.X; x < a.Image().Bounds().Max.X; x++ {
			if a.IndexAt(x, y) != b.IndexAt(x, y) {
				if count == 0 {
					first = image.Pt(x, y)
				}
				count++
			}
		}
	}
	return count, first
}

// --- correctness ---

// prepareAndDrawReplay runs fileio.PrepareReplay, which DrawReplay expects, then DrawReplay.
func prepareAndDrawReplay(mapData *fileio.Civ5MapData, replayData *fileio.Civ5ReplayData, outputFilename string, maxTurns int) error {
	if err := fileio.PrepareReplay(mapData, replayData); err != nil {
		return err
	}
	return DrawReplay(mapData, replayData, outputFilename, maxTurns)
}

// newCanvas returns a width x height canvas that draws exact palette colors and grows its palette as colors are used.
func newCanvas(width, height int) *raster.PalettedCanvas {
	c := raster.NewGrowingPalettedCanvas(1, 1)
	c.Resize(width, height)
	return c
}

// newMapCanvas returns a canvas the size of a map of the given size, as the map renderers make one.
func newMapCanvas(size fileio.MapSize) *raster.PalettedCanvas {
	w, h := imageSize(size, tileRadius)
	return newCanvas(int(w), int(h))
}

// colorAt returns the color of pixel (x, y).
func colorAt(c Canvas, x, y int) color.RGBA {
	r, g, b, _ := c.Image().At(x, y).RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
}

// countColor returns how many pixels of rect are want.
func countColor(c Canvas, rect image.Rectangle, want color.RGBA) int {
	n := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if colorAt(c, x, y) == want {
				n++
			}
		}
	}
	return n
}

// nearPoint reports whether a pixel within one of (x, y) is want.
func nearPoint(c Canvas, x, y float64, want color.RGBA) bool {
	return countColor(c, image.Rect(int(x)-1, int(y)-1, int(x)+2, int(y)+2), want) > 0
}

// drawnPixels returns how many pixels of c have been drawn on, that is differ from its background.
func drawnPixels(c *raster.PalettedCanvas) int {
	n := 0
	for y := 0; y < c.Image().Bounds().Dy(); y++ {
		for x := 0; x < c.Image().Bounds().Dx(); x++ {
			if c.IndexAt(x, y) != c.Background() {
				n++
			}
		}
	}
	return n
}

// tileCenter returns the center of the tile at (row, col) of a map of the given size.
func tileCenter(size fileio.MapSize, row, col int) (x, y float64) {
	return layoutForMap(size, tileRadius).center(fileio.TilePos{Row: row, Col: col})
}

// edgeMidpoint returns the middle of a hex's edge (see getHexEdge) centered on (x, y).
func edgeMidpoint(x, y float64, edge int) (mx, my float64) {
	line := getHexEdge(edge, x, y, tileRadius)
	return (line.X1 + line.X2) / 2, (line.Y1 + line.Y2) / 2
}

// gridFor returns the tile grid for the size of mapData.
func gridFor(mapData *fileio.Civ5MapData) *tileGrid { return buildTileGrid(mapData.Size(), tileRadius) }
