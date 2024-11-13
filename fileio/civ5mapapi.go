package fileio

func GetTerrainString(mapData *Civ5MapData, row int, column int) string {
	terrainType := mapData.MapTiles[row][column].TerrainType
	return mapData.TerrainList[terrainType]
}

func IsWaterTile(mapData *Civ5MapData, row int, column int) bool {
  terrainString := GetTerrainString(mapData, row, column)
  return terrainString == "TERRAIN_COAST" || terrainString == "TERRAIN_OCEAN"
}

func TileHasCity(mapData *Civ5MapData, row int, column int) bool {
  return mapData.MapTileImprovements[row][column].CityId != -1
}

func TileHasMountain(mapData *Civ5MapData, row int, column int) bool {
  return mapData.MapTiles[row][column].Elevation == 2
}

func GetPoliticalMapTileColor(mapData *fileio.Civ5MapData, row int, column int) string {
	tileOwner := mapData.MapTileImprovements[row][column].Owner
	if tileOwner == 0xFF {
		return ""
	}
	civIndex := mapData.CityOwnerIndexMap[tileOwner]
	tileColor := ""
	if civIndex < len(mapData.Civ5PlayerData) {
		tileColor = mapData.Civ5PlayerData[civIndex].TeamColor
	}
	return tileColor
}