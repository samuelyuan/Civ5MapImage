package graphics

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// bordersOnlyMapData is a multi-owner map with no roads, rivers or cities.
func bordersOnlyMapData(size, civs int) *fileio.Civ5MapData {
	mapData := flatMapData(size)
	for row := range mapData.MapTileImprovements {
		for col := range mapData.MapTileImprovements[row] {
			owner := ((row/3)*5 + col/3) % (civs + 1)
			if owner == civs {
				owner = -1 // unowned
			}
			mapData.MapTileImprovements[row][col].Owner = owner
		}
	}
	mapData.Civ5PlayerData = buildBenchMapData(size, size, civs, 1).Civ5PlayerData
	mapData.CityOwnerIndexMap = map[int]int{}
	for i := 0; i < civs; i++ {
		mapData.CityOwnerIndexMap[i] = i
	}
	return mapData
}

// Every pixel within borderReach of a different owner must be in its own owner's border color; unowned tiles get none.
func TestTileBordersMatchTheRuleWithNoGaps(t *testing.T) {
	const size, civs = 18, 4
	mapData := bordersOnlyMapData(size, civs)
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	grid := renderer.tileGridFor(fileio.MapSize{Height: size, Width: size})

	ownerOf := func(id int32) int { return mapData.MapTileImprovements[int(id)/size][int(id)%size].Owner }
	indexOf := func(c color.RGBA) uint8 { return canvas.IndexFor(c.R, c.G, c.B) }

	expected, gaps, strays := 0, 0, 0
	for y := 0; y < grid.height; y++ {
		for x := 0; x < grid.width; x++ {
			id := grid.at(x, y)
			if id < 0 {
				continue
			}
			row, col := int(id)/size, int(id)%size
			owner := ownerOf(id)
			if fileio.IsInvalidTileOwner(owner) {
				continue
			}
			near := false
			for _, p := range borderProbes {
				if other := grid.at(x+p[0], y+p[1]); other >= 0 && ownerOf(other) != owner {
					near = true
					break
				}
			}
			border := indexOf(tileBorderColor(mapData, fileio.TilePos{Row: row, Col: col}))
			got := canvas.IndexAt(x, y)
			switch {
			case near:
				expected++
				if got != border {
					gaps++
				}
			default:
				hex, _ := PoliticalHexTile(mapData, fileio.TilePos{Row: row, Col: col}, mapLayout(renderer.config.Radius))
				fill := indexOf(color.RGBA{hex.R, hex.G, hex.B, 255})
				outline := indexOf(tileOutlineColor(color.RGBA{hex.R, hex.G, hex.B, 255}))
				if got == border && border != fill && border != outline {
					strays++
				}
			}
		}
	}
	if expected == 0 {
		t.Fatal("no border pixels expected; the test map has no boundaries")
	}
	if gaps != 0 {
		t.Errorf("%d of %d pixels near a boundary are not their owner's border color", gaps, expected)
	}
	if strays != 0 {
		t.Errorf("%d pixels far from any boundary are drawn in their border color", strays)
	}
}

// flatMapData is one unowned grass map with nothing on it.
func flatMapData(size int) *fileio.Civ5MapData {
	mapData := buildBenchMapData(size, size, 1, 1)
	for row := range mapData.MapTileImprovements {
		for col := range mapData.MapTileImprovements[row] {
			tile := mapData.MapTileImprovements[row][col]
			tile.Owner, tile.CityId, tile.CityName, tile.RouteType = -1, -1, "", 255
			mapData.MapTiles[row][col].RiverData = 0
		}
	}
	return mapData
}

// Every pixel pair straddling two tiles must have an outline pixel on one side.
func TestTileOutlinesLeaveNoGapsBetweenTiles(t *testing.T) {
	const size = 14
	mapData := flatMapData(size)
	renderer := NewMapRenderer(DefaultDrawingConfig())
	canvas := newBenchCanvas(mapData)
	renderer.DrawPoliticalMapTileMajor(canvas, mapData)
	grid := renderer.tileGridFor(fileio.MapSize{Height: size, Width: size})

	hex, _ := PoliticalHexTile(mapData, fileio.TilePos{Row: 0, Col: 0}, mapLayout(renderer.config.Radius))
	outline := tileOutlineColor(color.RGBA{hex.R, hex.G, hex.B, 255})
	outlineIndex := canvas.IndexFor(outline.R, outline.G, outline.B)

	bounds := canvas.Image().Bounds()
	window := image.Rect(bounds.Dx()/4, bounds.Dy()/4, bounds.Dx()*3/4, bounds.Dy()*3/4)
	checked, gaps := 0, 0
	isOutline := func(x, y int) bool { return canvas.IndexAt(x, y) == outlineIndex }
	for y := window.Min.Y; y < window.Max.Y; y++ {
		for x := window.Min.X; x < window.Max.X; x++ {
			for _, q := range [][2]int{{x + 1, y}, {x, y + 1}} {
				a, b := grid.at(x, y), grid.at(q[0], q[1])
				if a < 0 || b < 0 || a == b {
					continue
				}
				checked++
				if !isOutline(x, y) && !isOutline(q[0], q[1]) {
					gaps++
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no tile boundaries checked")
	}
	if gaps != 0 {
		t.Errorf("%d of %d pixel pairs across a tile boundary have no outline pixel on either side", gaps, checked)
	}
}

// The outline network must be one connected piece, with no notches at hex corners.
func TestTileOutlineIsConnectedAcrossHexCorners(t *testing.T) {
	const size = 14
	mapData := flatMapData(size)
	canvas := newBenchCanvas(mapData)
	NewMapRenderer(DefaultDrawingConfig()).DrawPoliticalMapTileMajor(canvas, mapData)

	// Look at the middle of the map only, away from the edge tiles' missing neighbors.
	bounds := canvas.Image().Bounds()
	window := image.Rect(bounds.Dx()/4, bounds.Dy()/4, bounds.Dx()*3/4, bounds.Dy()*3/4)
	indexCounts := map[uint8]int{}
	for y := window.Min.Y; y < window.Max.Y; y++ {
		for x := window.Min.X; x < window.Max.X; x++ {
			indexCounts[canvas.IndexAt(x, y)]++
		}
	}
	fill, fillCount := uint8(0), 0
	for index, count := range indexCounts {
		if count > fillCount {
			fill, fillCount = index, count
		}
	}

	outline := map[image.Point]bool{}
	for y := window.Min.Y; y < window.Max.Y; y++ {
		for x := window.Min.X; x < window.Max.X; x++ {
			if canvas.IndexAt(x, y) != fill {
				outline[image.Pt(x, y)] = true
			}
		}
	}
	if len(outline) == 0 {
		t.Fatal("no outline pixels drawn")
	}

	seen := map[image.Point]bool{}
	components := 0
	for start := range outline {
		if seen[start] {
			continue
		}
		components++
		seen[start] = true
		queue := []image.Point{start}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if n := image.Pt(p.X+dx, p.Y+dy); outline[n] && !seen[n] {
						seen[n] = true
						queue = append(queue, n)
					}
				}
			}
		}
	}
	if components != 1 {
		t.Errorf("outline splits into %d disconnected pieces, want 1", components)
	}
}

func testPalette(n int) color.Palette {
	palette := color.Palette{color.RGBA{0, 0, 0, 255}}
	for i := 1; i < n; i++ {
		palette = append(palette, color.RGBA{uint8(i * 3), uint8(255 - i*2), uint8(i * 5), 255})
	}
	return palette
}

// Adjacent hexes must tile the plane: every interior pixel belongs to the hex whose center is nearest.
func TestPalettedCanvasHexTilingHasNoGapsOrOverlaps(t *testing.T) {
	const rows, cols, radius = 10, 10, 16.0
	palette := testPalette(rows*cols + 1)
	width, height := imageSize(fileio.MapSize{Height: rows, Width: cols}, radius)
	canvas := raster.NewPalettedCanvas(int(width), int(height), palette)
	layout := pixelLayout(radius, int(height))

	type center struct{ x, y float64 }
	centers := make([]center, 0, rows*cols)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x, y := layout.center(fileio.TilePos{Row: row, Col: col})
			centers = append(centers, center{x, y})
			r, g, b, _ := palette[len(centers)].RGBA()
			canvas.SetColor(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			canvas.DrawRegularPolygon(6, x, y, radius, math.Pi/2)
			canvas.Fill()
		}
	}

	checked := 0
	for y := int(3 * radius); y < canvas.Image().Bounds().Dy()-int(3*radius); y++ {
		for x := int(3 * radius); x < canvas.Image().Bounds().Dx()-int(3*radius); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			best, bestD, secondD := -1, math.Inf(1), math.Inf(1)
			for i, c := range centers {
				d := math.Hypot(px-c.x, py-c.y)
				if d < bestD {
					best, bestD, secondD = i, d, bestD
				} else if d < secondD {
					secondD = d
				}
			}
			if secondD-bestD < 1e-6 {
				continue
			}
			checked++
			if got, want := canvas.IndexAt(x, y), uint8(best+1); got != want {
				t.Fatalf("pixel (%d, %d): index %d, want %d (hex %d)", x, y, got, want, best)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no pixels checked")
	}
}

// On a map that isn't square, a tile's id, the pixel at its center and the tile found under that pixel must all agree,
// which fails if the id is built or decoded with the height where the width belongs.
func TestTileGridIDsAgreeWithTileIDOnANonSquareMap(t *testing.T) {
	grid := buildTileGrid(fileio.MapSize{Height: 5, Width: 9}, 16)
	for row := 0; row < grid.mapSize.Height; row++ {
		for col := 0; col < grid.mapSize.Width; col++ {
			pos := fileio.TilePos{Row: row, Col: col}
			x, y := grid.layout.center(pos)
			px, py := int(x), int(y)
			if got, want := grid.at(px, py), grid.tileID(pos); got != want {
				t.Fatalf("the pixel at the center of %+v holds id %d, want tileID %d", pos, got, want)
			}
			if got := grid.tilesIn(image.Rect(px, py, px+1, py+1)); len(got) != 1 || !got[pos] {
				t.Fatalf("tilesIn at the center of %+v = %v, want just that tile", pos, got)
			}
		}
	}
}

func TestTileGridForRebuildsOnlyWhenTheMapSizeChanges(t *testing.T) {
	renderer := NewMapRenderer(DefaultDrawingConfig())
	wide := fileio.MapSize{Height: 4, Width: 8}
	first := renderer.tileGridFor(fileio.MapSize{Height: 4, Width: 6})
	if again := renderer.tileGridFor(fileio.MapSize{Height: 4, Width: 6}); again != first {
		t.Error("the same size rebuilt the grid, want the cached one")
	}
	if grid := renderer.tileGridFor(wide); grid == first || grid.mapSize != wide {
		t.Errorf("a wider map of the same height gave grid size %+v (rebuilt: %v), want %+v rebuilt", grid.mapSize, grid != first, wide)
	}
	taller := fileio.MapSize{Height: 6, Width: 4}
	renderer.tileGridFor(fileio.MapSize{Height: 4, Width: 4})
	if grid := renderer.tileGridFor(taller); grid.mapSize != taller {
		t.Errorf("a taller map of the same width gave grid size %+v, want %+v", grid.mapSize, taller)
	}
	tall := fileio.MapSize{Height: 8, Width: 4}
	if grid := renderer.tileGridFor(tall); grid.mapSize != tall {
		t.Errorf("the next taller size gave grid size %+v, want %+v", grid.mapSize, tall)
	}
}
