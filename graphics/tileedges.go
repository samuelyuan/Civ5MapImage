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
// It is built for one map size, and the replay functions take it beside the map data and use its size throughout, so it must be built for that map data's size.
type tileGrid struct {
	width, height int // pixels
	mapSize       fileio.MapSize
	layout        tileLayout
	ids           []int32       // row*mapSize.Width+col of the tile covering the pixel, or -1 outside every hex
	outline       [][]gridPixel // per tile: its pixels whose right or lower neighbor is another tile
	rim           [][]gridPixel // per tile: its pixels within borderReach of another tile
}

type gridPixel = raster.Pixel

// at returns the id of the tile covering pixel (x, y), or -1 outside every hex and off the canvas.
func (g *tileGrid) at(x, y int) int32 {
	if x < 0 || y < 0 || x >= g.width || y >= g.height {
		return -1
	}
	return g.ids[y*g.width+x]
}

// tileID is the id of the tile at pos in ids, outline and rim.
func (g *tileGrid) tileID(pos fileio.TilePos) int32 { return int32(pos.Row*g.mapSize.Width + pos.Col) }

// tilePos is the tile with the given id.
func (g *tileGrid) tilePos(id int32) fileio.TilePos {
	return fileio.TilePos{Row: int(id) / g.mapSize.Width, Col: int(id) % g.mapSize.Width}
}

// tilesIn returns the tiles with a pixel in rect.
func (g *tileGrid) tilesIn(rect image.Rectangle) tileSet {
	tiles := tileSet{}
	last := int32(-1)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if id := g.at(x, y); id >= 0 && id != last {
				tiles[g.tilePos(id)] = true
				last = id
			}
		}
	}
	return tiles
}

// tileBounds returns the pixel rect that holds the hex of the tile at pos.
func (g *tileGrid) tileBounds(pos fileio.TilePos) image.Rectangle {
	radius := g.layout.radius
	x, y := g.layout.center(pos)
	return image.Rect(int(x-radius)-1, int(y-radius)-1, int(x+radius)+2, int(y+radius)+2).Intersect(image.Rect(0, 0, g.width, g.height))
}

// buildTileGrid fills every hex on a throwaway canvas that records which tile covers each pixel.
func buildTileGrid(mapSize fileio.MapSize, radius float64) *tileGrid {
	w, h := imageSize(mapSize, radius)
	canvas := raster.NewPalettedCanvas(int(w), int(h), color.Palette{color.RGBA{0, 0, 0, 255}})
	canvas.TrackIDs()
	layout := layoutForMap(mapSize, radius)
	for row := 0; row < mapSize.Height; row++ {
		for col := 0; col < mapSize.Width; col++ {
			x, y := layout.center(fileio.TilePos{Row: row, Col: col})
			canvas.SetID(int32(row*mapSize.Width + col))
			canvas.DrawRegularPolygon(6, x, y, radius, math.Pi/2)
			canvas.Fill()
		}
	}
	g := &tileGrid{width: int(w), height: int(h), mapSize: mapSize, layout: layout, ids: canvas.IDs()}
	g.buildEdgePixels()
	return g
}

// buildEdgePixels lists each tile's possible outline and border pixels, built in parallel by row.
func (g *tileGrid) buildEdgePixels() {
	g.outline = make([][]gridPixel, g.mapSize.Height*g.mapSize.Width)
	g.rim = make([][]gridPixel, g.mapSize.Height*g.mapSize.Width)
	var wg sync.WaitGroup
	for row := 0; row < g.mapSize.Height; row++ {
		wg.Go(func() {
			for col := 0; col < g.mapSize.Width; col++ {
				g.buildTileEdgePixels(fileio.TilePos{Row: row, Col: col})
			}
		})
	}
	wg.Wait()
}

// buildTileEdgePixels lists the outline and rim pixels of the tile at pos.
func (g *tileGrid) buildTileEdgePixels(pos fileio.TilePos) {
	id := g.tileID(pos)
	outline := make([]gridPixel, 0, 64)
	rim := make([]gridPixel, 0, 192)
	bounds := g.tileBounds(pos)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if g.at(x, y) != id {
				continue
			}
			if raster.IsOutline(g.at, x, y, id) {
				outline = append(outline, gridPixel{X: int32(x), Y: int32(y)})
			}
			if raster.NearOther(g.at, borderProbes, x, y, id) {
				rim = append(rim, gridPixel{X: int32(x), Y: int32(y)})
			}
		}
	}
	g.outline[id], g.rim[id] = outline, rim
}

// borderProbes are the offsets within borderReach (Manhattan distance) of a pixel.
var borderProbes = raster.BorderProbes(borderReach)

// drawTileOutlines draws a 1px outline on each pixel whose right or lower neighbor is another tile.
func drawTileOutlines(canvas *raster.PalettedCanvas, mapData *fileio.Civ5MapData, grid *tileGrid, tiles []fileio.TilePos) {
	for _, pos := range tiles {
		hex, _ := PoliticalHexTile(mapData, pos, grid.layout)
		outline := tileOutlineColor(hex.Fill)
		index := canvas.IndexFor(outline.R, outline.G, outline.B)
		for _, p := range grid.outline[grid.tileID(pos)] {
			canvas.PaintPixel(int(p.X), int(p.Y), index)
		}
	}
}

// drawTileBorders draws every pixel within borderReach of a different owner in its own owner's border color.
func drawTileBorders(canvas Canvas, mapData *fileio.Civ5MapData, grid *tileGrid, tiles []fileio.TilePos) {
	if len(mapData.MapTileImprovements) == 0 {
		return
	}
	for _, pos := range tiles {
		owner := mapData.TileImprovement(pos).Owner
		if fileio.IsInvalidTileOwner(owner) || !bordersOtherOwner(mapData, grid, pos, owner) {
			continue
		}
		id := grid.tileID(pos)
		border := tileBorderColor(mapData, pos)
		index := canvas.IndexFor(border.R, border.G, border.B)
		for _, p := range grid.rim[id] {
			if nearOtherOwner(mapData, grid, p, id, owner) {
				canvas.PaintPixel(int(p.X), int(p.Y), index)
			}
		}
	}
}

// nearOtherOwner reports whether pixel p of tile id, owned by owner, is within borderReach of a tile with a different owner.
func nearOtherOwner(mapData *fileio.Civ5MapData, grid *tileGrid, p gridPixel, id int32, owner int) bool {
	for _, offset := range borderProbes {
		other := grid.at(int(p.X)+offset.X, int(p.Y)+offset.Y)
		if raster.IsOther(other, id) && mapData.TileImprovement(grid.tilePos(other)).Owner != owner {
			return true
		}
	}
	return false
}

// bordersOtherOwner reports whether a neighbor of tile pos exists and has a different owner.
func bordersOtherOwner(mapData *fileio.Civ5MapData, grid *tileGrid, pos fileio.TilePos, owner int) bool {
	for _, n := range fileio.GetNeighbors(pos) {
		if n.InMap(grid.mapSize) && mapData.TileImprovement(n).Owner != owner {
			return true
		}
	}
	return false
}
