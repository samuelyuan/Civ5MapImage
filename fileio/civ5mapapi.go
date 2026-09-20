package fileio

import (
	"fmt"
	"strings"
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

// SetupCivPlayerData builds or reorders mapData's civ player data from the replay.
func SetupCivPlayerData(mapData *Civ5MapData, replayData *Civ5ReplayData) {
	if len(mapData.Civ5PlayerData) == 0 || !replayData.IsReplayFile {
		fmt.Println("Rebuilding civ player data from replay file...")
		mapData.Civ5PlayerData = make([]*Civ5PlayerData, 0)
		for i := 0; i < len(replayData.AllCivs); i++ {
			civName := replayData.AllCivs[i].Name

			if strings.Contains(civName, "CIVILIZATION") || strings.Contains(civName, "MINOR_CIV") {
				mapData.Civ5PlayerData = append(mapData.Civ5PlayerData, &Civ5PlayerData{
					Index:     i,
					CivType:   civName,
					TeamColor: replayData.AllCivs[i].LongName,
				})
			} else {
				civName = strings.ReplaceAll(civName, " ", "")
				mapData.Civ5PlayerData = append(mapData.Civ5PlayerData, &Civ5PlayerData{
					Index:     i,
					CivType:   fmt.Sprintf("CIVILIZATION_%s", strings.ToUpper(civName)),
					TeamColor: fmt.Sprintf("PLAYERCOLOR_%s", strings.ToUpper(civName)),
				})
			}
		}
	} else {
		// Swap player civilization to index 0
		indexPlayerCivilization := -1
		for i := 0; i < len(mapData.Civ5PlayerData); i++ {
			if mapData.Civ5PlayerData[i].CivType == replayData.PlayerCiv {
				indexPlayerCivilization = i
				break
			}
		}

		fmt.Println("Player civilization index:", indexPlayerCivilization)
		if indexPlayerCivilization != -1 {
			temp := mapData.Civ5PlayerData[0]
			mapData.Civ5PlayerData[0] = mapData.Civ5PlayerData[indexPlayerCivilization]
			mapData.Civ5PlayerData[indexPlayerCivilization] = temp
		}
	}
}

// ApplyReplayEvent applies one event to mapData's tile improvements and returns the next free city id.
func ApplyReplayEvent(mapData *Civ5MapData, event Civ5ReplayEvent, nextCityId int) int {
	switch event.TypeId {
	case ReplayEventCityFounded:
		for _, tile := range event.Tiles {
			mapData.MapTileImprovements[tile.Y][tile.X].CityId = nextCityId
			mapData.MapTileImprovements[tile.Y][tile.X].CityName = strings.TrimSuffix(event.Text, " is founded.")
			nextCityId += 1
		}
	case ReplayEventTilesClaimed, ReplayEventCityTransferred:
		for _, tile := range event.Tiles {
			mapData.MapTileImprovements[tile.Y][tile.X].Owner = event.CivId
		}
	case ReplayEventTilesRazed:
		for _, tile := range event.Tiles {
			mapData.MapTileImprovements[tile.Y][tile.X].Owner = -1
			mapData.MapTileImprovements[tile.Y][tile.X].CityId = -1
			mapData.MapTileImprovements[tile.Y][tile.X].CityName = ""
			// Razed tile becomes a road
			mapData.MapTileImprovements[tile.Y][tile.X].RouteType = 2
		}
	}
	return nextCityId
}

// ResetCityOwnerIndexMap makes mapData.CityOwnerIndexMap an identity map over the replay's civs, since events already use AllCivs indices.
func ResetCityOwnerIndexMap(mapData *Civ5MapData, replayData *Civ5ReplayData) {
	if mapData.CityOwnerIndexMap == nil {
		mapData.CityOwnerIndexMap = make(map[int]int)
	}
	for i := 0; i < len(replayData.AllCivs); i++ {
		fmt.Println("Index", i, ", civ data:", replayData.AllCivs[i])
		mapData.CityOwnerIndexMap[i] = i
	}
}

// PrepareReplay validates that replayData fits mapData and readies mapData's civ data for rendering it; DrawReplay expects this to have run.
func PrepareReplay(mapData *Civ5MapData, replayData *Civ5ReplayData) error {
	if err := ValidateReplayRenderable(mapData, replayData); err != nil {
		return fmt.Errorf("replay is not compatible with map: %w", err)
	}
	fmt.Println("Player Civ:", replayData.PlayerCiv)
	ResetCityOwnerIndexMap(mapData, replayData)
	SetupCivPlayerData(mapData, replayData)
	return nil
}
