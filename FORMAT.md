# Civ5MapImage File Format Documentation

This document describes the technical file formats used by Civilization 5 for maps, replays, and save files.

## Table of Contents

* [Map File Format](#map-file-format)
  + [Header](#header)
  + [Geography list data](#geography-list-data)
  + [Map geography](#map-geography)
  + [Map tile data](#map-tile-data)
  + [Game description header](#game-description-header)
  + [Game description data](#game-description-data)
  + [Unit data format](#unit-data-format)
  + [City format](#city-format)
  + [Unknown block](#unknown-block)
  + [Team format](#team-format)
  + [Player format](#player-format)
  + [Map tile improvement properties](#map-tile-improvement-properties)
  + [Map tile improvement data](#map-tile-improvement-data)
* [Replay File Format](#replay-file-format)
* [Save File Format](#save-file-format)

## Map File Format

This file format covers .civ5map files, which stores the map data. All data is stored in little endian.

### Header

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint8 | 1 byte | ScenarioVersion (The leftmost 4 bits are for scenario. The rightmost 4 bits are for version, which is set to 12 for newer files.) |
| uint32 | 4 bytes | Map width |
| uint32 | 4 bytes | Map height |
| uint8 | 1 byte | Number of players |
| uint8[4] | 4 bytes | Settings (hasWorldWrap, hasRandomResources, hasRandomGoodies) |
| uint32 | 4 bytes | TerrainDataSize (Length of terrain list) |
| uint32 | 4 bytes | FeatureTerrainDataSize (Length of feature terrain list) |
| uint32 | 4 bytes | FeatureWonderDataSize (Length of feature wonder list) |
| uint32 | 4 bytes | ResourceDataSize  (Length of resource list) |
| uint32 | 4 bytes | ModDataSize |
| uint32 | 4 bytes | MapNameLength |
| uint32 | 4 bytes | MapDescriptionLength |

Following the header, is a list of strings whose size is determined in the header. Each string list will have a zero byte to split items.

### Geography list data

| Type | Size | Description |
| ---- | ---- | ----------- |
| String list | TerrainDataSize bytes | Terrain (e.g. TERRAIN_GRASS) |
| String list | FeatureTerrainDataSize bytes | Feature terrain (e.g. FEATURE_ICE) |
| String list | FeatureWonderDataSize bytes | Feature wonders(e.g. FEATURE_CRATER, FEATURE_FUJI) |
| String list | ResourceDataSize bytes | Resources (e.g. RESOURCE_IRON) |
| String | ModDataSize bytes | modData |
| String | MapNameLength bytes | Map name |
| String | MapDescriptionLength bytes | Map description |
| uint32 | 4 bytes | WorldSizeLength (Only if version >= 11) |
| String | WorldSizeLength bytes | WorldSize (Only if version >= 11) |

### Map geography

The map data is inverted, which means that the bottommost row rendered on the screen is stored on the top row of the array and the topmost row rendered on the screen is stored on the last row of the array.

| Type | Size | Description |
| ---- | ---- | ----------- |
| MapTile[Height][Width] | (Height * Width * 8) bytes | Map geography |

### Map tile data

The size of this struct is 8 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint8 | 1 byte | TerrainType (index in terrain list) |
| uint8 | 1 byte | ResourceType (index in resource list, 0xFF if none) |
| uint8 | 1 byte | FeatureTerrainType (index in feature terrain list, 0xFF if none) |
| uint8 | 1 byte | RiverData (The low 3 bits means the tile border has a river. Only 3 edges needs to be marked per tile. 4 (>>2) is southwest edge, 2 (>>1) is southeast edge, 1 (>>0) is eastern edge) |
| uint8 | 1 byte | Elevation (0 = flat, 1 = hills, 2 = mountain) |
| uint8 | 1 byte | Continent (0 = none, 1 = Americas, 2 = Asia, 3 = Africa, 4 = Europe) |
| uint8 | 1 byte | FeatureWonderType (index in feature wonder list, 0xFF if none) |
| uint8 | 1 byte | ResourceAmount |

### Game description header

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[68] | 68 bytes | Unknown, seems related to GameSpeed |
| uint32 | 4 bytes | MaxTurns |
| byte[4] | 4 bytes | Unknown |
| uint32 | 4 bytes | StartYear |
| uint8 | 1 byte | PlayerCount (Number of playable civs) |
| uint8 | 1 byte | CityStateCount |
| uint8 | 1 byte | TeamCount (should be the sum of PlayerCount and CityStateCount) |
| byte | 1 byte | Unknown |
| uint32 | 4 bytes | ImprovementDataSize |
| uint32 | 4 bytes | UnitTypeDataSize |
| uint32 | 4 bytes | TechTypeDataSize |
| uint32 | 4 bytes | PolicyTypeDataSize |
| uint32 | 4 bytes | BuildingTypeDataSize |
| uint32 | 4 bytes | PromotionTypeDataSize |
| uint32 | 4 bytes | UnitDataSize |
| uint32 | 4 bytes | UnitNameDataSize |
| uint32 | 4 bytes | CityDataSize |
| uint32 | 4 bytes | VictoryDataSize (Only if version >= 11) |
| uint32 | 4 bytes | GameOptionDataSize (Only if version >= 11) |

### Game description data

| Type | Size | Description |
| ---- | ---- | ----------- |
| String list | ImprovementDataSize bytes | Improvements (e.g. IMPROVEMENT_FARM) |
| String list | UnitTypeDataSize bytes | Unit types (e.g. UNIT_SETTLER) |
| String list | TechTypeDataSize bytes | Tech types (e.g. TECH_AGRICULTURE) |
| String list | PolicyTypeDataSize bytes | Policy types (e.g. POLICY_LIBERTY) |
| String list | BuildingTypeDataSize bytes | Building types (e.g. BUILDING_STADIUM) |
| String list | PromotionTypeDataSize bytes | Promotion types (e.g. PROMOTION_DRILL_1) |
| Unit data array | UnitDataSize bytes | Unit data |
| Unit name array | UnitNameDataSize bytes | Unit names |
| City array | CityDataSize bytes | City information |
| String list | VictoryDataSize bytes | Victory types (e.g. VICTORY_CULTURAL) |
| String list | GameOptionDataSize bytes | Game options (e.g. GAMEOPTION_NO_CITY_RAZING) |

### Unit data format

In version 11, the sizeof this struct is 48 bytes.

In version 12, the sizeof this struct is 84 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[2] | 2 bytes | Unknown |
| uint16 | 2 bytes | Index to custom unit name data |
| uint32 | 4 bytes | Experience |
| uint32 | 4 bytes | Health (100% health is 100000) |
| uint8 (version 11) or uint32 (version 12) | 1 byte for version 11, 4 bytes for version 12 | Unit type |
| uint8 | 1 byte | Owner |
| uint8 | 1 byte | Facing direction |
| uint8 | 1 byte | Status (The low 3 bits are used. 4 (>>2) is garrisoned, 2 (>>1) is embarked, 1 (>>0) is fortified) |
| byte | 1 byte | Unknown (Only for version 12)|
| byte[] | 32 bytes for version 11, 64 bytes for version 12 | Promotion data |

### City format

In version 11, the sizeof this struct is 104 bytes.

In version 12, the sizeof this struct is 136 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[64] | 64 bytes | City name |
| uint8 | 1 byte | Owner |
| uint8 | 1 byte | Settings |
| uint16 | 2 bytes | Population |
| uint32 | 4 bytes | Health (100% health is 100000) |
| byte[] | 32 bytes for version 11, 64 bytes for version 12 | Building data |

### Unknown block

There is a section between the city data and team data that doesn't seem to be used anywhere, except for padding. The sizeof this block is unknown, but this block size increases as the number of civs increases.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[] | Unknown bytes | This block doesn't seem to correspond to anything in the game |

### Team format

The sizeof this struct is 64 bytes. The team name is usually the default value, e.g. Team 1.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[64] | 64 bytes | Team name |

### Player format

The sizeof this struct is 436 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[32] | 32 bytes | Policies |
| byte[64] | 64 bytes | Leader name (override leader name) |
| byte[64] | 64 bytes | Civ name (override civ name) |
| byte[64] | 64 bytes | Civ type (default civ name) |
| byte[64] | 64 bytes | Team color |
| byte[64] | 64 bytes | Era |
| byte[64] | 64 bytes | Handicap |
| uint32 | 4 bytes | Culture |
| uint32 | 4 bytes | Gold |
| uint32 | 4 bytes | Start position X |
| uint32 | 4 bytes | Start position Y |
| uint8 | 1 byte | Team |
| uint8 | 1 byte | Playable |
| byte[2] | 2 bytes | Unknown |

### Map tile improvement properties

This block is always placed at the end of a file.

| Type | Size | Description |
| ---- | ---- | ----------- |
| MapTileImprovement[Height][Width] | (Height * Width * 8) bytes | 2D array of map tile improvements |

### Map tile improvement data

The size of this struct is 8 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint16 | 2 bytes | City id (0xFFFF if none) |
| uint16 | 2 bytes | Unit id (0xFFFF if none) |
| uint8 | 1 byte | Owner |
| uint8 | 1 byte | Improvement |
| uint8 | 1 byte | RouteType (0 = road, 1 = railroad, 0xFF = none) |
| uint8 | 1 byte | RouteOwner |

## Replay File Format

The replay files store a list of civilizations, events, and datasets for different statistics like gold per turn.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[4] | 4 bytes | Game name |
| uint32 | 4 bytes | unknownBlock1 |
| varstring | var bytes | Game version |
| varstring | var bytes | Game build |
| uint32 | 4 bytes | Current turn number |
| byte[1] | 1 bytes | unknownBlock2 |
| varstring | var bytes | Player civ |
| varstring | var bytes | Difficulty |
| varstring | var bytes | Era start |
| varstring | var bytes | Era end |
| varstring | var bytes | Game speed |
| varstring | var bytes | World size |
| varstring | var bytes | Map filename |

DLC Array Element

Array size is uint32 followed by list of elements.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[16] | 16 bytes | dlcId |
| byte[4] | 4 bytes | dlcEnabled |
| varstring | var bytes | dlcName |

Mods Array Element

Array size is uint32 followed by list of elements.

| Type | Size | Description |
| ---- | ---- | ----------- |
| varstring | var bytes | modId |
| byte[4] | 4 bytes | modVersion |
| varstring | var bytes | modName |

Header Continued

| Type | Size | Description |
| ---- | ---- | ----------- |
| varstring | var bytes | Civ name |
| varstring | var bytes | Leader name |
| varstring | var bytes | Player color |
| byte[8] | 8 bytes | unknownBlock5 |
| varstring | var bytes | mapFilename2 |

Unknown Block

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | unknownVersion |
| uint32[4] | 16 bytes | Unknown array |
| uint32 | 4 bytes | unknownCount1 |
| uint32[] | (unknownCount1 * 4) bytes | Unknown array |
| uint32 | 4 bytes | unknownCount2 |
| uint32[] | (unknownCount2 + 1) bytes | Unknown array |
| uint8 | 1 byte | Unknown |

Header Continued

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Start turn |
| int32 | 4 bytes | Start year |
| uint32 | 4 bytes | End turn |
| varstring | varstring bytes | End year |
| uint32 | 4 bytes | zeroStartYear |
| uint32 | 4 bytes | zeroEndYear |

### Civ Names

The civ names are stored in an array.

Array size is uint32.

Array element format

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32[4] | 4*4 bytes | Unknown |
| varstring | var bytes | Leader name |
| varstring | var bytes | Civ long name |
| varstring | var bytes | Civ name |
| varstring | var bytes | Civ demonym |

### Civ Dataset Names

Array size is uint32.

Array element format

| Type | Size | Description |
| ---- | ---- | ----------- |
| varstring | var bytes | Dataset name |

### Civ Dataset Values

This is a 3D array. The first level is divided by civilization and the second level is divided by category. To get a list of dataset values, you have to call datasetValues[civIndex][datasetNameIndex]. Each dataset value is represented by a Turn and Value pair.

### Replay Events

The number of events is a uint32.

Replay Event format

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Turn |
| uint32 | 4 bytes | Type id |
| uint32 | 4 bytes | Number tiles |
| tile array | numTIles * 4 bytes | Tile array, which contains uint16 for x and uint16 for y |
| uint32 | 4 bytes | Civilization Id |
| varstring | varstring bytes | Event text |

### Tiles

Tile data contains information about the physical map.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | unknownVariable1 |
| uint32 | 4 bytes | unknownVariable2 |
| uint8 | 1 byte | Elevation id |
| uint8 | 1 byte | Type id |
| uint8 | 1 byte | Feature id |
| uint8 | 1 byte | unknownVariable3 |

## Save File Format

Most of the save file is compressed and the header begins with 0x789C, which is a ZLIB header.
