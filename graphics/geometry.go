package graphics

import (
	"image/color"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// This file holds pure geometry calculations (which lines/positions to draw) with no Canvas
// dependency, so they're testable via plain value assertions. drawmap.go owns when to draw them
// (loop structure, InvertY() timing, canvas call sequencing).

// InvertedRow mirrors a row against mapHeight, for city name labels only: they're drawn after a
// second InvertY() cancels the first back to identity (InvertY()'s transform mirrors text
// glyphs, not just repositions them), so label math must supply this inversion itself instead of
// relying on the canvas transform like every other draw call does.
func InvertedRow(mapHeight, row int) int {
	return mapHeight - row
}

// RiverEdgesForTile decodes a RiverData bitmask into hex edge lines. Only southwest/southeast/
// east are produced; the other three edges belong to the neighboring tile's own RiverData.
func RiverEdgesForTile(riverData int, centerX, centerY, radius float64) []Line {
	var edges []Line
	if (riverData>>2)&1 != 0 { // Southwest (edge 3)
		edges = append(edges, getHexEdge(3, centerX, centerY, radius))
	}
	if (riverData>>1)&1 != 0 { // Southeast (edge 4)
		edges = append(edges, getHexEdge(4, centerX, centerY, radius))
	}
	if riverData&1 != 0 { // East (edge 5)
		edges = append(edges, getHexEdge(5, centerX, centerY, radius))
	}
	return edges
}

// RoadSegment is one road/railroad line (tile center to the midpoint toward its neighbor, i.e.
// the shared tile border) plus the width/color its route type draws with.
type RoadSegment struct {
	Line      Line
	LineWidth float64
	R, G, B   uint8
}

// RoadSegmentsForTile returns segments from tile (row, col) to each connected neighbor, or nil
// if the tile has no route (RouteType 255). A neighbor is connected if it has a route too, or a
// city (roads visibly terminate at cities even without a route type set).
func RoadSegmentsForTile(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) []RoadSegment {
	routeType := mapData.MapTileImprovements[row][col].RouteType
	if routeType == 255 {
		return nil
	}

	x1, y1 := fileio.GetImagePosition(row, col, radius)

	var segments []RoadSegment
	neighbors := fileio.GetNeighbors(col, row)
	for n := 0; n < len(neighbors); n++ {
		newX := neighbors[n][0]
		newY := neighbors[n][1]
		if newX < 0 || newY < 0 || newX >= mapWidth || newY >= mapHeight {
			continue
		}

		neighborTile := mapData.MapTileImprovements[newY][newX]
		if neighborTile.RouteType == 255 && neighborTile.CityName == "" {
			continue
		}

		x2, y2 := fileio.GetImagePosition(newY, newX, radius)

		var lineWidth float64
		var r, g, b uint8
		switch routeType {
		case 1: // Railroad
			lineWidth, r, g, b = 2.0, 76, 51, 0
		case 0: // Road
			lineWidth, r, g, b = 1.0, 51, 51, 51
		default: // Unknown
			lineWidth, r, g, b = 1.0, 0, 0, 0
		}

		// Draw only up to the midpoint, which is the shared tile border.
		borderX := (x1 + x2) / 2.0
		borderY := (y1 + y2) / 2.0

		segments = append(segments, RoadSegment{
			Line:      Line{X1: x1, Y1: y1, X2: borderX, Y2: borderY},
			LineWidth: lineWidth,
			R:         r,
			G:         g,
			B:         b,
		})
	}
	return segments
}

// BorderSegment is one territory border line and its color.
type BorderSegment struct {
	Line    Line
	R, G, B uint8
}

// BorderSegmentsForTile returns border lines around tile (row, col) against neighbors with a
// different owner, colored with the owning civ's border color (white if unrecognized). Returns
// nil if the tile has no valid owner.
func BorderSegmentsForTile(mapData *fileio.Civ5MapData, mapHeight, mapWidth, row, col int, radius float64) []BorderSegment {
	currentTileOwner := mapData.MapTileImprovements[row][col].Owner
	if fileio.IsInvalidTileOwner(currentTileOwner) {
		return nil
	}

	x1, y1 := fileio.GetImagePosition(row, col, radius)

	tileColor := fileio.GetPoliticalMapTileColor(mapData, row, col)
	renderColor, ok := civColorMap[tileColor]
	borderColor := color.RGBA{255, 255, 255, 255}
	if ok {
		if strings.Contains(fileio.GetTileCivName(mapData, row, col), "MINOR") {
			// Invert city state colors, matching DrawTerritoryTiles' convention.
			borderColor = renderColor.OuterColor
		} else {
			borderColor = renderColor.InnerColor
		}
	}

	var segments []BorderSegment
	neighbors := fileio.GetNeighbors(col, row)
	for n := 0; n < len(neighbors); n++ {
		newX := neighbors[n][0]
		newY := neighbors[n][1]
		if newX < 0 || newY < 0 || newX >= mapWidth || newY >= mapHeight {
			continue
		}

		otherTileOwner := mapData.MapTileImprovements[newY][newX].Owner
		if currentTileOwner == otherTileOwner {
			continue
		}

		line := getHexEdge(n, x1, y1, radius-1)
		segments = append(segments, BorderSegment{
			Line: line,
			R:    borderColor.R,
			G:    borderColor.G,
			B:    borderColor.B,
		})
	}
	return segments
}
