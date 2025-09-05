package fileio

import (
	"image/color"
	"math"
)

// Hex grid utility functions
var (
	NeighborOdd  = [6][2]int{{1, 1}, {0, 1}, {-1, 0}, {0, -1}, {1, -1}, {1, 0}}
	NeighborEven = [6][2]int{{0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, 0}}
)

func GetNeighbors(x int, y int) [6][2]int {
	var offset [6][2]int
	if y%2 == 1 {
		offset = NeighborOdd
	} else {
		offset = NeighborEven
	}

	neighbors := [6][2]int{}
	for i := 0; i < 6; i++ {
		newX := x + offset[i][0]
		newY := y + offset[i][1]
		neighbors[i][0] = newX
		neighbors[i][1] = newY
	}
	return neighbors
}

func GetImagePosition(i int, j int, radius float64) (float64, float64) {
	angle := math.Pi / 6

	x := (radius * 1.5) + float64(j)*(2*radius*math.Cos(angle))
	y := radius + float64(i)*radius*(1+math.Sin(angle))
	if i%2 == 1 {
		x += radius * math.Cos(angle)
	}
	return x, y
}

func GetPhysicalMapTileColor(terrainString string) color.RGBA {
	switch terrainString {
	case "TERRAIN_GRASS":
		return color.RGBA{105, 125, 54, 255}
	case "TERRAIN_PLAINS":
		return color.RGBA{127, 121, 71, 255}
	case "TERRAIN_DESERT":
		return color.RGBA{200, 200, 164, 255}
	case "TERRAIN_TUNDRA":
		return color.RGBA{118, 123, 117, 255}
	case "TERRAIN_SNOW":
		return color.RGBA{238, 249, 255, 255}
	case "TERRAIN_COAST":
		return color.RGBA{95, 149, 149, 255}
	case "TERRAIN_OCEAN":
		return color.RGBA{47, 74, 93, 255}
	}

	// default
	return color.RGBA{0, 0, 0, 255}
}

func GetTerrainString(mapData *Civ5MapData, row int, column int) string {
	// Check bounds to prevent panic
	if row < 0 || row >= len(mapData.MapTiles) {
		return ""
	}
	if column < 0 || column >= len(mapData.MapTiles[row]) {
		return ""
	}
	terrainType := mapData.MapTiles[row][column].TerrainType
	if terrainType < 0 || terrainType >= len(mapData.TerrainList) {
		return ""
	}
	return mapData.TerrainList[terrainType]
}

func IsWaterTile(mapData *Civ5MapData, row int, column int) bool {
	terrainString := GetTerrainString(mapData, row, column)
	return terrainString == "TERRAIN_COAST" || terrainString == "TERRAIN_OCEAN"
}

func TileHasCity(mapData *Civ5MapData, row int, column int) bool {
	// Check bounds to prevent panic
	if row < 0 || row >= len(mapData.MapTileImprovements) {
		return false
	}
	if column < 0 || column >= len(mapData.MapTileImprovements[row]) {
		return false
	}
	return mapData.MapTileImprovements[row][column].CityId != -1
}

func TileHasMountain(mapData *Civ5MapData, row int, column int) bool {
	// Check bounds to prevent panic
	if row < 0 || row >= len(mapData.MapTiles) {
		return false
	}
	if column < 0 || column >= len(mapData.MapTiles[row]) {
		return false
	}
	return mapData.MapTiles[row][column].Elevation == 2
}

func IsInvalidTileOwner(value int) bool {
	return value == 0xFF || value == 0xFFFF || value == 0xFFFFFFFF || value == -1
}

func GetTileCivName(mapData *Civ5MapData, row int, column int) string {
	// Check bounds to prevent panic
	if row < 0 || row >= len(mapData.MapTileImprovements) {
		return ""
	}
	if column < 0 || column >= len(mapData.MapTileImprovements[row]) {
		return ""
	}
	tileOwner := mapData.MapTileImprovements[row][column].Owner
	if IsInvalidTileOwner(tileOwner) {
		return ""
	}
	civIndex := mapData.CityOwnerIndexMap[tileOwner]
	if civIndex < len(mapData.Civ5PlayerData) {
		return mapData.Civ5PlayerData[civIndex].CivType
	}
	return ""
}

func GetPoliticalMapTileColor(mapData *Civ5MapData, row int, column int) string {
	// Check bounds to prevent panic
	if row < 0 || row >= len(mapData.MapTileImprovements) {
		return ""
	}
	if column < 0 || column >= len(mapData.MapTileImprovements[row]) {
		return ""
	}
	tileOwner := mapData.MapTileImprovements[row][column].Owner
	if IsInvalidTileOwner(tileOwner) {
		return ""
	}
	civIndex := mapData.CityOwnerIndexMap[tileOwner]
	tileColor := ""
	if civIndex < len(mapData.Civ5PlayerData) {
		tileColor = mapData.Civ5PlayerData[civIndex].TeamColor
	}
	return tileColor
}
