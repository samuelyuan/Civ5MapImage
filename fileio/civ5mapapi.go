package fileio

import (
	"fmt"
	"strings"
)

// TilePos is a map tile's position. Build it with field names, so a row and a column can't be swapped.
type TilePos struct{ Row, Col int }

// MapSize is a map's dimensions in tiles. Build it with field names, so a height and a width can't be swapped.
type MapSize struct{ Height, Width int }

// Size returns the map's size, from its physical tiles.
func (m *Civ5MapData) Size() MapSize {
	if len(m.MapTiles) == 0 {
		return MapSize{}
	}
	return MapSize{Height: len(m.MapTiles), Width: len(m.MapTiles[0])}
}

// PhysicalTile returns the physical tile at pos, or nil if pos is off the map.
func (m *Civ5MapData) PhysicalTile(pos TilePos) *Civ5MapTilePhysical {
	if pos.Row < 0 || pos.Row >= len(m.MapTiles) || pos.Col < 0 || pos.Col >= len(m.MapTiles[pos.Row]) {
		return nil
	}
	return m.MapTiles[pos.Row][pos.Col]
}

// TileImprovement returns the improvement record at pos, or nil if pos is off the map or the map has no improvement data.
func (m *Civ5MapData) TileImprovement(pos TilePos) *Civ5MapTileImprovement {
	if pos.Row < 0 || pos.Row >= len(m.MapTileImprovements) || pos.Col < 0 || pos.Col >= len(m.MapTileImprovements[pos.Row]) {
		return nil
	}
	return m.MapTileImprovements[pos.Row][pos.Col]
}

// InMap reports whether p is a tile of a map of the given size.
func (p TilePos) InMap(size MapSize) bool {
	return p.Row >= 0 && p.Col >= 0 && p.Row < size.Height && p.Col < size.Width
}

// Hex grid neighbor offsets, by the parity of the row.
var (
	NeighborOdd  = [6]TilePos{{Col: 1, Row: 1}, {Col: 0, Row: 1}, {Col: -1, Row: 0}, {Col: 0, Row: -1}, {Col: 1, Row: -1}, {Col: 1, Row: 0}}
	NeighborEven = [6]TilePos{{Col: 0, Row: 1}, {Col: -1, Row: 1}, {Col: -1, Row: 0}, {Col: -1, Row: -1}, {Col: 0, Row: -1}, {Col: 1, Row: 0}}
)

// GetNeighbors returns the six neighbors of pos in a fixed order, including ones that fall off the map (see InMap).
func GetNeighbors(pos TilePos) [6]TilePos {
	offsets := NeighborEven
	if pos.Row%2 == 1 {
		offsets = NeighborOdd
	}
	var neighbors [6]TilePos
	for i, offset := range offsets {
		neighbors[i] = TilePos{Row: pos.Row + offset.Row, Col: pos.Col + offset.Col}
	}
	return neighbors
}

func GetTerrainString(mapData *Civ5MapData, pos TilePos) string {
	tile := mapData.PhysicalTile(pos)
	if tile == nil || tile.TerrainType < 0 || tile.TerrainType >= len(mapData.TerrainList) {
		return ""
	}
	return mapData.TerrainList[tile.TerrainType]
}

func IsWaterTile(mapData *Civ5MapData, pos TilePos) bool {
	terrainString := GetTerrainString(mapData, pos)
	return terrainString == "TERRAIN_COAST" || terrainString == "TERRAIN_OCEAN"
}

func TileHasCity(mapData *Civ5MapData, pos TilePos) bool {
	tile := mapData.TileImprovement(pos)
	return tile != nil && tile.CityId != -1
}

func TileHasMountain(mapData *Civ5MapData, pos TilePos) bool {
	tile := mapData.PhysicalTile(pos)
	return tile != nil && tile.Elevation == 2
}

func IsInvalidTileOwner(value int) bool {
	return value == 0xFF || value == 0xFFFF || value == 0xFFFFFFFF || value == -1
}

func GetTileCivName(mapData *Civ5MapData, pos TilePos) string {
	tile := mapData.TileImprovement(pos)
	if tile == nil || IsInvalidTileOwner(tile.Owner) {
		return ""
	}
	civIndex := mapData.CityOwnerIndexMap[tile.Owner]
	if civIndex < len(mapData.Civ5PlayerData) {
		return mapData.Civ5PlayerData[civIndex].CivType
	}
	return ""
}

func GetPoliticalMapTileColor(mapData *Civ5MapData, pos TilePos) string {
	tile := mapData.TileImprovement(pos)
	if tile == nil || IsInvalidTileOwner(tile.Owner) {
		return ""
	}
	civIndex := mapData.CityOwnerIndexMap[tile.Owner]
	if civIndex < len(mapData.Civ5PlayerData) {
		return mapData.Civ5PlayerData[civIndex].TeamColor
	}
	return ""
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
			record := mapData.TileImprovement(tile.Pos())
			record.CityId = nextCityId
			record.CityName = strings.TrimSuffix(event.Text, " is founded.")
			nextCityId += 1
		}
	case ReplayEventTilesClaimed, ReplayEventCityTransferred:
		for _, tile := range event.Tiles {
			mapData.TileImprovement(tile.Pos()).Owner = event.CivId
		}
	case ReplayEventTilesRazed:
		for _, tile := range event.Tiles {
			record := mapData.TileImprovement(tile.Pos())
			record.Owner = -1
			record.CityId = -1
			record.CityName = ""
			// Razed tile becomes a road
			record.RouteType = 2
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
