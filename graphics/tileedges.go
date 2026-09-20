package graphics

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/samuelyuan/Civ5MapImage/fileio"
	"github.com/samuelyuan/Civ5MapImage/graphics/raster"
)

// borderReach is how many pixels of border each side of a boundary between two owners gets.
const borderReach = 2

// tileGrid maps every canvas pixel to the tile whose hex fills it; built once with the canvas's own fill rule.
type tileGrid struct {
	width, height, mapHeight, mapWidth int
	layout                             tileLayout
	ids                                []int32       // row*mapWidth+col of the tile covering the pixel, or -1 outside every hex
	outline                            [][]gridPixel // per tile: its pixels whose right or lower neighbor is another tile
	rim                                [][]rimPixel  // per tile: its pixels within borderReach of another tile
}

type gridPixel struct{ x, y int32 }

// neighborMask is a set of a tile's six neighbors; bit j is the j-th neighbor in GetNeighbors order.
type neighborMask uint8

func (m *neighborMask) add(j int)                     { *m |= 1 << j }
func (m neighborMask) intersects(o neighborMask) bool { return m&o != 0 }

// rimPixel is a pixel near another tile; reach holds the neighbors within borderReach.
type rimPixel struct {
	x, y  int32
	reach neighborMask
}

// neighbor is a tile's neighbor; onMap is false where it falls off the map.
type neighbor struct {
	row, col int
	onMap    bool
}

func (g *tileGrid) at(x, y int) int32 {
	if x < 0 || y < 0 || x >= g.width || y >= g.height {
		return -1
	}
	return g.ids[y*g.width+x]
}

// tileID is the id of tile (row, col) in ids, outline and rim.
func (g *tileGrid) tileID(row, col int) int32 { return int32(row*g.mapWidth + col) }

// hasTile reports whether (row, col) is a tile on the map.
func (g *tileGrid) hasTile(row, col int) bool {
	return row >= 0 && col >= 0 && row < g.mapHeight && col < g.mapWidth
}

// neighbors returns tile (row, col)'s neighbors in GetNeighbors order.
func (g *tileGrid) neighbors(row, col int) (ns [6]neighbor) {
	for j, n := range fileio.GetNeighbors(col, row) { // GetNeighbors takes and returns (col, row)
		ns[j] = neighbor{row: n[1], col: n[0], onMap: g.hasTile(n[1], n[0])}
	}
	return ns
}

// neighborIDs returns the ids of tile (row, col)'s neighbors in GetNeighbors order, -1 where off the map.
func (g *tileGrid) neighborIDs(row, col int) (ids [6]int32) {
	for j, n := range g.neighbors(row, col) {
		ids[j] = -1
		if n.onMap {
			ids[j] = g.tileID(n.row, n.col)
		}
	}
	return ids
}

// tileBounds returns the pixel rect that holds tile (row, col)'s hex.
func (g *tileGrid) tileBounds(row, col int) image.Rectangle {
	radius := g.layout.radius
	x, y := g.layout.center(row, col)
	return image.Rect(int(x-radius)-1, int(y-radius)-1, int(x+radius)+2, int(y+radius)+2).Intersect(image.Rect(0, 0, g.width, g.height))
}

// buildTileGrid fills every hex on a throwaway canvas that records which tile covers each pixel.
func buildTileGrid(mapHeight, mapWidth int, radius float64) *tileGrid {
	w, h := GetImagePosition(mapHeight, mapWidth, radius)
	canvas := raster.NewPalettedCanvas(int(w), int(h), color.Palette{color.RGBA{0, 0, 0, 255}})
	canvas.TrackIDs()
	layout := pixelLayout(radius, int(h))
	for row := 0; row < mapHeight; row++ {
		for col := 0; col < mapWidth; col++ {
			x, y := layout.center(row, col)
			canvas.SetID(int32(row*mapWidth + col))
			canvas.DrawRegularPolygon(6, x, y, radius, math.Pi/2)
			canvas.Fill()
		}
	}
	g := &tileGrid{width: int(w), height: int(h), mapHeight: mapHeight, mapWidth: mapWidth, layout: layout, ids: canvas.IDs()}
	g.buildEdgePixels()
	return g
}

// buildEdgePixels lists each tile's possible outline and border pixels, so a turn walks short lists; rows build in parallel.
func (g *tileGrid) buildEdgePixels() {
	g.outline = make([][]gridPixel, g.mapHeight*g.mapWidth)
	g.rim = make([][]rimPixel, g.mapHeight*g.mapWidth)
	var wg sync.WaitGroup
	for row := 0; row < g.mapHeight; row++ {
		wg.Go(func() {
			for col := 0; col < g.mapWidth; col++ {
				g.buildTileEdgePixels(row, col)
			}
		})
	}
	wg.Wait()
}

func (g *tileGrid) buildTileEdgePixels(row, col int) {
	id := g.tileID(row, col)
	neighbors := g.neighborIDs(row, col)
	outline := make([]gridPixel, 0, 64)
	rim := make([]rimPixel, 0, 192)
	bounds := g.tileBounds(row, col)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if g.at(x, y) != id {
				continue
			}
			if g.isOutline(x, y, id) {
				outline = append(outline, gridPixel{int32(x), int32(y)})
			}
			if reach := g.rimReach(x, y, id, neighbors); reach != 0 {
				rim = append(rim, rimPixel{int32(x), int32(y), reach})
			}
		}
	}
	g.outline[id], g.rim[id] = outline, rim
}

// isOutline reports whether pixel (x, y) of tile id has a right or lower neighbor in another tile.
func (g *tileGrid) isOutline(x, y int, id int32) bool {
	right, down := g.at(x+1, y), g.at(x, y+1)
	return (right >= 0 && right != id) || (down >= 0 && down != id)
}

// rimReach returns rimPixel.reach for pixel (x, y) of tile id, given its neighbors' ids.
func (g *tileGrid) rimReach(x, y int, id int32, neighbors [6]int32) (reach neighborMask) {
	for _, p := range borderProbes {
		other := g.at(x+p[0], y+p[1])
		if other < 0 || other == id {
			continue
		}
		for j, n := range neighbors {
			if n == other {
				reach.add(j)
			}
		}
	}
	return reach
}

// borderProbes are the offsets within borderReach (Manhattan distance) of a pixel.
var borderProbes = func() [][2]int {
	var probes [][2]int
	for dy := -borderReach; dy <= borderReach; dy++ {
		for dx := -borderReach; dx <= borderReach; dx++ {
			if d := max(dx, -dx) + max(dy, -dy); d > 0 && d <= borderReach {
				probes = append(probes, [2]int{dx, dy})
			}
		}
	}
	return probes
}()

// tileGridFor returns the renderer's tile grid for a map of this size, building it on first use.
func (mr *MapRenderer) tileGridFor(mapHeight, mapWidth int) *tileGrid {
	if mr.grid == nil || mr.grid.mapHeight != mapHeight || mr.grid.mapWidth != mapWidth {
		mr.grid = buildTileGrid(mapHeight, mapWidth, mr.config.Radius)
	}
	return mr.grid
}

// drawTileOutlines draws a 1px outline on each pixel whose right or lower neighbor is another tile.
func (mr *MapRenderer) drawTileOutlines(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, grid *tileGrid, tiles []tileCoord) {
	for _, rc := range tiles {
		hex, _ := PoliticalHexTile(mapData, rc.row, rc.col, grid.layout)
		outline := tileOutlineColor(color.RGBA{hex.R, hex.G, hex.B, 255})
		index := canvas.IndexFor(outline.R, outline.G, outline.B)
		for _, p := range grid.outline[grid.tileID(rc.row, rc.col)] {
			canvas.PaintPixel(int(p.x), int(p.y), index)
		}
	}
}

// drawTileBorders draws every pixel within borderReach of a different owner in its own owner's border color.
func (mr *MapRenderer) drawTileBorders(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, grid *tileGrid, tiles []tileCoord) {
	if len(mapData.MapTileImprovements) == 0 {
		return
	}
	for _, rc := range tiles {
		owner := mapData.MapTileImprovements[rc.row][rc.col].Owner
		if fileio.IsInvalidTileOwner(owner) {
			continue
		}
		differs := differingNeighbors(mapData, grid, rc, owner)
		if differs == 0 {
			continue
		}
		border := tileBorderColor(mapData, rc.row, rc.col)
		index := canvas.IndexFor(border.R, border.G, border.B)
		for _, p := range grid.rim[grid.tileID(rc.row, rc.col)] {
			if p.reach.intersects(differs) {
				canvas.PaintPixel(int(p.x), int(p.y), index)
			}
		}
	}
}

// differingNeighbors returns the neighbors of tile rc that exist and have a different owner.
func differingNeighbors(mapData *fileio.Civ5MapData, grid *tileGrid, rc tileCoord, owner int) (differs neighborMask) {
	for j, n := range grid.neighbors(rc.row, rc.col) {
		if n.onMap && mapData.MapTileImprovements[n.row][n.col].Owner != owner {
			differs.add(j)
		}
	}
	return differs
}
