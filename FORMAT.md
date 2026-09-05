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
  + [Free-list allocator](#free-list-allocator)
  + [Unit data format](#unit-data-format)
    - [Owner encoding](#owner-encoding)
    - [Unit name array](#unit-name-array)
  + [City format](#city-format)
  + [Team relationship data](#team-relationship-data)
  + [City-state influence data](#city-state-influence-data)
  + [Team visibility data](#team-visibility-data)
  + [Team format](#team-format)
  + [Player format](#player-format)
  + [Map tile improvement properties](#map-tile-improvement-properties)
  + [Map tile improvement data](#map-tile-improvement-data)
* [Replay File Format](#replay-file-format)
  + [DLC Array Element](#dlc-array-element)
  + [Mods Array Element](#mods-array-element)
  + [Header Continued](#header-continued)
  + [Civ Names](#civ-names)
  + [Civ Dataset Names](#civ-dataset-names)
  + [Civ Dataset Values](#civ-dataset-values)
  + [Replay Events](#replay-events)
    - [Replay Event format](#replay-event-format)
  + [Tiles](#tiles)
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

String lists follow the header, sized per the header, with items null-separated.

### Geography list data

| Type | Size | Description |
| ---- | ---- | ----------- |
| String list | TerrainDataSize bytes | Terrain (e.g. TERRAIN_GRASS) |
| String list | FeatureTerrainDataSize bytes | Feature terrain (e.g. FEATURE_ICE) |
| String list | FeatureWonderDataSize bytes | Feature wonders(e.g. FEATURE_CRATER, FEATURE_FUJI) |
| String list | ResourceDataSize bytes | Resources (e.g. RESOURCE_IRON) |
| String | ModDataSize bytes | Mod data |
| String | MapNameLength bytes | Map name |
| String | MapDescriptionLength bytes | Map description |
| uint32 | 4 bytes | WorldSizeLength (Only if version >= 11) |
| String | WorldSizeLength bytes | WorldSize (Only if version >= 11) |

### Map geography

Rows are stored bottom-to-top: array row 0 is the bottom row on screen, and the array's last row is the top row on screen.

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
| byte[64] | 64 bytes | GameSpeed: null-terminated string (e.g. GAMESPEED_STANDARD) |
| uint32 | 4 bytes | StartingTurn |
| uint32 | 4 bytes | MaxTurns |
| uint32 | 4 bytes | TargetScore: score-victory target |
| int32 | 4 bytes | StartYear |
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

### Free-list allocator

The Unit data array, Unit name array, and City array are each preceded by a free-list head, not a record count. The record array has a fixed, power-of-2 capacity, often well above the records in use. Unused slots are threaded into a singly-linked list via their leading bytes (next index, -1 = end); every other field keeps its stale value.

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32 | 4 bytes | Free-list head index (-1 if none) |

### Unit data format

In version 11, the sizeof this struct is 48 bytes.

In version 12, the sizeof this struct is 84 bytes.

Preceded by the free-list head; an unused slot's leading 2 bytes hold the link instead of a real stacking value.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint16 | 2 bytes | Index of another unit sharing this unit's tile (0xFFFF if not stacked with anything) |
| uint16 | 2 bytes | NameIndex: index into the unit name array (0xFFFF if the unit has no custom name) |
| uint32 | 4 bytes | Experience |
| uint32 | 4 bytes | Health (100% health is 100000) |
| uint8 (version 11) or uint32 (version 12) | 1 byte for version 11, 4 bytes for version 12 | Unit type |
| uint8 | 1 byte | Owner |
| uint8 | 1 byte | Facing direction |
| uint8 | 1 byte | Status (low 3 bits: garrisoned(4), embarked(2), fortified(1)) |
| byte | 1 byte | Unknown (only for version 12) |
| byte[] | 32 bytes for version 11, 64 bytes for version 12 | Promotion data (bitset, one bit per index in the promotion type list; bit N set means the unit has promotion N) |

#### Facing direction encoding

| Raw value | Direction |
| --- | --- |
| 0 | Random Direction |
| 1 | Northeast |
| 2 | East |
| 3 | Southeast |
| 4 | Southwest |
| 5 | West |
| 6 | Northwest |

#### Owner encoding

The Owner byte is used the same way for units, cities, and map tile improvement data:

* 0-31: index into the major civilization player list
* 32-95: city-state; the actual city-state index is the value minus 32
* 96 (units only): Barbarian unit - a sentinel value, not an index into any player/city-state list

#### Unit name array

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32 | 4 bytes | Free-list head index |
| byte[64][] | (UnitNameDataSize - 4) bytes | Fixed-size null-terminated name records. A unit's `NameIndex` is the 0-based index of its record here |

### City format

In version 11, the sizeof this struct is 104 bytes.

In version 12, the sizeof this struct is 136 bytes.

Preceded by the free-list head; an unused slot's leading Name bytes hold the link instead, so a freed city's name typically reads empty.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[64] | 64 bytes | City name |
| uint8 | 1 byte | Owner |
| uint8 | 1 byte | Settings |
| uint16 | 2 bytes | Population |
| uint32 | 4 bytes | Health (100% health is 100000) |
| byte[] | 32 bytes for version 11, 64 bytes for version 12 | Building data (bitset, one bit per index in the building type list; bit N set means the city has building N) |

### Team relationship data

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[] | `5 * ceil(TeamCount*(TeamCount-1)/2 / 8)` bytes | Team diplomacy relationship bitsets: `IN_CONTACT`, `AT_WAR`, `PERMANENT_WAR_OR_PEACE`, `OPEN_BORDERS`, `DEFENSIVE_PACT`, one triangular (no self-pairs) bitset per type, in that order |

Within a type's bitset, the bit for team pair `(i, j)` with `i > j` is at index `i*(i-1)/2 + j`.

### City-state influence data

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32[] | `PlayerCount * 64 * 4` bytes | Each major civ player's influence value with every city-state; row-major `[player][cityState]` with a fixed row stride of 64 entries per player, regardless of the actual city-state count |

### Team visibility data

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[] | `ceil(Width*Height*TeamCount / 8)` bytes | Per-team, per-tile visibility/exploration (fog-of-war) bitset: one full `Width*Height`-bit tile bitmap per team |

### Team format

The sizeof this struct is 64 bytes. The team name is usually the default value, e.g. Team 1.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[64] | 64 bytes | Team name |

### Player format

The sizeof this struct is 436 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[32] | 32 bytes | Policies (bitset, one bit per index in the policy type list; bit N set means the player has policy N) |
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
| byte[4] | 4 bytes | Game name, should always be "CIV5" |
| uint32 | 4 bytes | UnknownUint1 |
| varstring | var bytes | Game version |
| varstring | var bytes | Game build |
| uint32 | 4 bytes | Current turn number |
| byte[1] | 1 bytes | UnknownByte1 |
| varstring | var bytes | Player civ |
| varstring | var bytes | Difficulty |
| varstring | var bytes | Era start |
| varstring | var bytes | Era end |
| varstring | var bytes | Game speed |
| varstring | var bytes | World size |
| varstring | var bytes | Map filename |

### DLC Array Element

Array size is uint32 followed by list of elements.

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[16] | 16 bytes | DLC id |
| byte[4] | 4 bytes | DLC enabled |
| varstring | var bytes | DLC name |

### Mods Array Element

Array size is uint32 followed by list of elements.

| Type | Size | Description |
| ---- | ---- | ----------- |
| varstring | var bytes | Mod id |
| byte[4] | 4 bytes | Mod version |
| varstring | var bytes | Mod name |

### Header Continued

| Type | Size | Description |
| ---- | ---- | ----------- |
| varstring | var bytes | Civ name |
| varstring | var bytes | Leader name |
| varstring | var bytes | Player color |
| uint32 | 4 bytes | Replay version |
| uint32 | 4 bytes | Active player index |
| varstring | var bytes | Map filename 2 |
| uint32 | 4 bytes | WorldSize: 0=WORLDSIZE_DUEL, 1=WORLDSIZE_TINY, 2=WORLDSIZE_SMALL, 3=WORLDSIZE_STANDARD, 4=WORLDSIZE_LARGE, 5=WORLDSIZE_HUGE |
| uint32 | 4 bytes | Climate: 0=CLIMATE_TEMPERATE, 1=CLIMATE_TROPICAL, 2=CLIMATE_ARID, 3=CLIMATE_ROCKY, 4=CLIMATE_COLD |
| uint32 | 4 bytes | SeaLevel: 0=SEALEVEL_LOW, 1=SEALEVEL_MEDIUM, 2=SEALEVEL_HIGH |
| uint32 | 4 bytes | Era: 0=ERA_ANCIENT, 1=ERA_CLASSICAL, 2=ERA_MEDIEVAL, 3=ERA_RENAISSANCE, 4=ERA_INDUSTRIAL, 5=ERA_MODERN, 6=ERA_POSTMODERN, 7=ERA_FUTURE |
| uint32 | 4 bytes | GameSpeed: 0=GAMESPEED_MARATHON, 1=GAMESPEED_EPIC, 2=GAMESPEED_STANDARD, 3=GAMESPEED_QUICK |
| uint32 | 4 bytes | GameOptionCount |
| uint32[] | (GameOptionCount * 4) bytes | names of the game options enabled at game setup: 0=GAMEOPTION_NO_CITY_RAZING, 1=GAMEOPTION_NO_BARBARIANS, 2=GAMEOPTION_RAGING_BARBARIANS, 3=GAMEOPTION_ALWAYS_WAR, 4=GAMEOPTION_ALWAYS_PEACE, 5=GAMEOPTION_ONE_CITY_CHALLENGE, 6=GAMEOPTION_NO_CHANGING_WAR_PEACE, 7=GAMEOPTION_NEW_RANDOM_SEED, 8=GAMEOPTION_LOCK_MODS, 9=GAMEOPTION_COMPLETE_KILLS, 10=GAMEOPTION_NO_GOODY_HUTS, 11=GAMEOPTION_RANDOM_PERSONALITIES, 12=GAMEOPTION_POLICY_SAVING, 13=GAMEOPTION_PROMOTION_SAVING, 14=GAMEOPTION_END_TURN_TIMER_ENABLED, 15=GAMEOPTION_QUICK_COMBAT, 16=GAMEOPTION_DISABLE_START_BIAS, 17=GAMEOPTION_NO_SCIENCE, 18=GAMEOPTION_NO_POLICIES, 19=GAMEOPTION_NO_HAPPINESS, 20=GAMEOPTION_NO_TUTORIAL, 21=GAMEOPTION_NO_RELIGION |
| uint32 | 4 bytes | VictoryTypeCount |
| uint32[] | (VictoryTypeCount * 4) bytes | names of the victory conditions enabled at game setup: 0=VICTORY_TIME, 1=VICTORY_SPACE_RACE, 2=VICTORY_DOMINATION, 3=VICTORY_CULTURAL, 4=VICTORY_DIPLOMATIC |
| uint32 | 4 bytes | VictoryAchieved: victory type PlayerCiv won, or 0xFFFFFFFF if not (ongoing, resigned, or someone else won) |
| uint8 | 1 byte | UnknownByte2 |
| uint32 | 4 bytes | Start turn |
| int32 | 4 bytes | Start year |
| uint32 | 4 bytes | End turn |
| varstring | varstring bytes | End year |
| uint32 | 4 bytes | Zero start year |
| uint32 | 4 bytes | Zero end year |

### Civ Names

The civ names are stored in an array.

Array size is uint32.

Array element format

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Civilization index |
| uint32 | 4 bytes | Leader type index |
| uint32 | 4 bytes | Player color index |
| uint32 | 4 bytes | Difficulty: 0=HANDICAP_SETTLER, 1=HANDICAP_CHIEFTAIN, 2=HANDICAP_WARLORD, 3=HANDICAP_PRINCE, 4=HANDICAP_KING, 5=HANDICAP_EMPEROR, 6=HANDICAP_IMMORTAL, 7=HANDICAP_DEITY |
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

3D array `datasetValues[civIndex][datasetNameIndex]` of (Turn, Value) pairs.

### Replay Events

The number of events is a uint32.

#### Replay Event format

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
| uint32 | 4 bytes | Map entry count |
| uint32 | 4 bytes | Turn key - always equals this file's own "End turn" |
| uint8 | 1 byte | PlotType: 0=PLOT_MOUNTAIN, 1=PLOT_HILLS, 2=PLOT_LAND, 3=PLOT_OCEAN - not the same field as `.civ5map`'s Elevation |
| uint8 | 1 byte | TerrainType |
| uint8 | 1 byte | Feature |
| uint8 | 1 byte | River bits (bit 0 unused. 8 (>>3) is southwest edge, 4 (>>2) is east edge, 2 (>>1) is southeast edge) - different bit layout from `.civ5map`'s RiverData |

## Save File Format

Most of the save file is compressed and the header begins with 0x789C, which is a ZLIB header.
