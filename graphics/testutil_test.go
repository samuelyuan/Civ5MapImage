package graphics

import (
	"fmt"
	"image"
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

// MockCanvas is a Canvas that records operations instead of drawing, for tests.
type MockCanvas struct {
	operations    []string
	width, height int
	indexes       map[[3]uint8]uint8 // the index IndexFor gave each color
}

func NewMockCanvas(width, height int) *MockCanvas {
	return &MockCanvas{
		operations: make([]string, 0),
		width:      width,
		height:     height,
	}
}

func (m *MockCanvas) DrawRegularPolygon(sides int, x, y, radius, rotation float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawRegularPolygon(%d, %.2f, %.2f, %.2f, %.2f)", sides, x, y, radius, rotation))
}

func (m *MockCanvas) DrawRectangle(x, y, width, height float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawRectangle(%.2f, %.2f, %.2f, %.2f)", x, y, width, height))
}

func (m *MockCanvas) DrawTriangle(x1, y1, x2, y2, x3, y3 float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawTriangle(%.2f, %.2f, %.2f, %.2f, %.2f, %.2f)", x1, y1, x2, y2, x3, y3))
}

func (m *MockCanvas) DrawLine(x1, y1, x2, y2 float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawLine(%.2f, %.2f, %.2f, %.2f)", x1, y1, x2, y2))
}

func (m *MockCanvas) SetColor(r, g, b uint8) {
	m.operations = append(m.operations,
		fmt.Sprintf("SetColor(%d, %d, %d)", r, g, b))
}

func (m *MockCanvas) SetLineWidth(width float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("SetLineWidth(%.2f)", width))
}

func (m *MockCanvas) Fill() {
	m.operations = append(m.operations, "Fill()")
}

func (m *MockCanvas) Stroke() {
	m.operations = append(m.operations, "Stroke()")
}

func (m *MockCanvas) Resize(width, height int) {
	m.operations = append(m.operations,
		fmt.Sprintf("Resize(%d, %d)", width, height))
	m.width = width
	m.height = height
}

func (m *MockCanvas) DrawString(text string, x, y float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawString(\"%s\", %.2f, %.2f)", text, x, y))
}

// MeasureString approximates the default font (basicfont.Face7x13: 7px advance, 13px tall).
func (m *MockCanvas) MeasureString(text string) (w, h float64) {
	return float64(len(text)) * 7, 13
}

// IndexFor gives each distinct color the next index, counting from 0.
func (m *MockCanvas) IndexFor(r, g, b uint8) uint8 {
	if m.indexes == nil {
		m.indexes = map[[3]uint8]uint8{}
	}
	key := [3]uint8{r, g, b}
	if _, ok := m.indexes[key]; !ok {
		m.indexes[key] = uint8(len(m.indexes))
	}
	m.operations = append(m.operations, fmt.Sprintf("IndexFor(%d, %d, %d) = %d", r, g, b, m.indexes[key]))
	return m.indexes[key]
}

func (m *MockCanvas) PaintPixel(x, y int, index uint8) {
	m.operations = append(m.operations, fmt.Sprintf("PaintPixel(%d, %d, %d)", x, y, index))
}

func (m *MockCanvas) Image() image.Image {
	// Return a simple 1x1 image for testing
	return image.NewRGBA(image.Rect(0, 0, 1, 1))
}

func (m *MockCanvas) SavePNG(filename string) error {
	m.operations = append(m.operations,
		fmt.Sprintf("SavePNG(\"%s\")", filename))
	return nil
}

// GetOperations returns the recorded operations.
func (m *MockCanvas) GetOperations() []string {
	return m.operations
}
