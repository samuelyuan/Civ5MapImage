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
  + [Player & Map Info](#player--map-info)
  + [Civ Roster](#civ-roster)
  + [Leaders & Civ Arrays](#leaders--civ-arrays)
  + [Climate Section](#climate-section)
  + [Game Name & Turn Info](#game-name--turn-info)
  + [Leader Array 2 & Player Setup](#leader-array-2--player-setup)
  + [Minor Civ Names](#minor-civ-names)
  + [Player Arrays & Colors](#player-arrays--colors)
  + [Sea Level](#sea-level)
  + [Additional Pre-Game Fields](#additional-pre-game-fields)
  + [Turn Speed Data](#turn-speed-data)
  + [World Size Data](#world-size-data)
  + [Game Options](#game-options)
  + [Compressed Block](#compressed-block)
  + [Decompressed Header](#decompressed-header)
  + [Version-Dependent Unit Data](#version-dependent-unit-data)
  + [Great Person](#great-person)
  + [Votes](#votes)
  + [Random State](#random-state)
  + [Save File Replay Events](#save-file-replay-events)
  + [Session & Barbarian State](#session--barbarian-state)
  + [Game Deals](#game-deals)
    - [Deal Element](#deal-element)
    - [Traded Item Element](#traded-item-element)
  + [Game Religions](#game-religions)
    - [Religion Element](#religion-element)
    - [Religion Beliefs Element](#religion-beliefs-element)
  + [Game Culture](#game-culture)
    - [Great Work Element](#great-work-element)
  + [Game Leagues](#game-leagues)
    - [League Element](#league-element)
    - [League Member Element](#league-member-element)
    - [League Project Element](#league-project-element)
    - [Resolution Element](#resolution-element)
    - [Resolution Effects Element](#resolution-effects-element)
    - [Voter/Proposer Decision Elements](#voterproposer-decision-elements)
  + [Game Trade](#game-trade)
    - [Trade Connection Element](#trade-connection-element)
  + [Embedded Database](#embedded-database)
  + [Map Header](#map-header)

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

### Player & Map Info

| Type | Size | Description |
| ---- | ---- | ----------- |
| varstring | var bytes | Player civ name |
| varstring | var bytes | Player leader name |
| varstring | var bytes | Player color |
| byte[16] | 16 bytes | UnknownBytes1 |
| varstring | var bytes | Version |
| byte[16] | 16 bytes | UnknownBytes2 |
| uint32 | 4 bytes | UnknownUint1 |
| uint32 | 4 bytes | Slot hints version (loadSlotHints/loadSlotsHelper's own uiVersion) |
| uint32 | 4 bytes | UnknownUint3 |
| uint32 | 4 bytes | UnknownUint4 |
| varstring | var bytes | Map filename 2 |
| uint32 | 4 bytes | Civilization index array count |
| int32[] | (count * 4) bytes | Civilization index per player slot |
| uint32 | 4 bytes | Nickname array count |
| varstring[] | var bytes | Nickname per player slot |
| uint32 | 4 bytes | Slot status array count |
| uint32[] | (count * 4) bytes | Slot status per player slot: 0=OPEN, 1=COMPUTER, 2=CLOSED, 3=TAKEN, 4=OBSERVER |
| uint32 | 4 bytes | Slot claim array count |
| uint32[] | (count * 4) bytes | Slot claim per player slot: 0=UNASSIGNED, 1=RESERVED, 2=ASSIGNED |
| uint32 | 4 bytes | Team type array count |
| uint32[] | (count * 4) bytes | Team index per player slot |
| uint32 | 4 bytes | Handicap array count |
| uint32[] | (count * 4) bytes | Handicap index per player slot |

### Civ Roster

Absent entirely (zero bytes) when slot hints version < 3 - see Player & Map Info above.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Civilization key array count (only if slot hints version >= 3) |
| varstring[] | var bytes | Civilization key per player slot (only if slot hints version >= 3) |

### Leaders & Civ Arrays

Leader key array is absent entirely (zero bytes) when slot hints version < 3 - see Player & Map Info
above.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Leader key array count (only if slot hints version >= 3) |
| varstring[] | var bytes | Leader key per player slot (only if slot hints version >= 3) |
| uint32 | 4 bytes | Pre-game format marker (0 if the block below is absent) |
| byte[12] | 12 bytes | UnknownBytes1 (only if pre-game format marker != 0) |
| varstring | var bytes | Computer username |
| uint32 | 4 bytes | UnknownInt array count |
| int32[] | (count * 4) bytes | UnknownInt array elements |
| byte[53] | 53 bytes | UnknownBytes2 |
| uint32 | 4 bytes | UnknownUint1 array count |
| uint32[] | (count * 4) bytes | UnknownUint1 array elements |
| uint32 | 4 bytes | Civ array count |
| varstring[] | var bytes | Civ names (again) |
| uint32 | 4 bytes | UnknownUint2 array count |
| uint32[] | (count * 4) bytes | UnknownUint2 array elements |
| uint32 | 4 bytes | Civ array 2 count |
| varstring[] | var bytes | Civ array 2 strings |
| uint32 | 4 bytes | Extra array count (only if slot hints version < 3) |
| uint32[] | (count * 4) bytes | Extra array elements (only if slot hints version < 3) |

### Climate Section

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Climate enum |
| uint32 | 4 bytes | Climate info id |
| uint32 | 4 bytes | Climate info civilopedia |
| varstring | var bytes | Climate display name |
| uint32 | 4 bytes | Climate help |
| uint32 | 4 bytes | Climate disabled help |
| uint32 | 4 bytes | Climate strategy |
| varstring | var bytes | Climate type (e.g. CLIMATE_TEMPERATE) |
| varstring | var bytes | Climate text key |
| varstring | var bytes | Climate display name 2 |
| int32 | 4 bytes | Desert percent change |
| uint32 | 4 bytes | Jungle latitude |
| uint32 | 4 bytes | Hill range |
| uint32 | 4 bytes | Mountain percent |
| float32 | 4 bytes | Snow latitude change |
| float32 | 4 bytes | Tundra latitude change |
| float32 | 4 bytes | Grass latitude change |
| float32 | 4 bytes | Desert bottom latitude change |
| float32 | 4 bytes | Desert top latitude change |
| float32 | 4 bytes | Ice latitude |
| float32 | 4 bytes | Rand ice latitude |

### Game Name & Turn Info

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Era enum |
| uint32 | 4 bytes | Email address array count (MAX_PLAYERS) |
| varstring[] | var bytes | Email addresses |
| float32 | 4 bytes | End turn timer length |
| uint32 | 4 bytes | Flag decal array count |
| varstring[] | var bytes | Flag decals |
| uint32 | 4 bytes | Deprecated force-controls count |
| byte[7] | 7 bytes | Deprecated force-controls flags |
| int32 | 4 bytes | Game mode enum |
| varstring | var bytes | Game name |
| uint32 | 4 bytes | Game speed enum |
| uint8 | 1 byte | Game started (bool) |
| uint32 | 4 bytes | Current turn number |
| int32 | 4 bytes | Legacy game-type dummy int (unused, uiVersion==0 branch) |
| uint8 | 1 byte | Network multiplayer game (bool) |
| uint32 | 4 bytes | Game update time |
| uint32 | 4 bytes | Handicap array count (MAX_PLAYERS) |
| uint32[] | (count * 4) bytes | Handicap array elements (HandicapTypes per player; element 0 matches the header's own difficulty string) |

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Tracked-player handicap array count (only if pre-game format marker != 4) |
| int32[] | (count * 4) bytes | Tracked-player handicap array elements (only if pre-game format marker != 4) |
| byte[2] | 2 bytes | UnknownBytes3 (only if pre-game format marker != 4) |
| byte[2] | 2 bytes | Reserved bytes, present instead of the above 2 rows when pre-game format marker == 4 |

### Leader Array 2 & Player Setup

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Leader name array count |
| varstring[] | var bytes | Leader names |
| uint32 | 4 bytes | Padding marker |
| byte[] | (marker + 1) * 4 bytes | UnknownBytes1 (only if marker != 0) |
| varstring | var bytes | Computer username 2 |
| byte[7] | 7 bytes | UnknownId4 |
| varstring | var bytes | Map filename 3 |
| uint32 | 4 bytes | Max city elimination |
| uint32 | 4 bytes | Max turns |
| uint32 | 4 bytes | Num minor civs |

### Minor Civ Names

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Minor civ names array count (only as many entries as real city-states in the game, not MAX_PLAYERS) |
| varstring[] | var bytes | Minor civ (city-state) names — patches matching entries into the civ roster |
| uint32 | 4 bytes | Minor nation civ array count (MAX_PLAYERS) |
| uint8[] | count bytes | Minor nation civ array elements (bool) |
| uint8 | 1 byte | Dummy value (bool) |
| uint32 | 4 bytes | Multiplayer options array count |
| uint8[] | count bytes | Multiplayer options array elements (bool): 0=SIMULTANEOUS_TURNS, 1=TAKEOVER_AI, 2=SHUFFLE_TEAMS, 3=ANONYMOUS |

### Player Arrays & Colors

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Net ID array count (MAX_PLAYERS) |
| int32[] | (count * 4) bytes | Net ID array elements (-1 = no network ID, expected for non-multiplayer saves) |
| uint32 | 4 bytes | Nickname array 2 count (MAX_PLAYERS; a second, independent nickname array - see Player & Map Info's own Nickname array) |
| varstring[] | var bytes | Nickname array 2 elements |
| int32 | 4 bytes | Num victory infos (the ruleset's own victory type count) |
| int32 | 4 bytes | Pitboss turn time |
| uint32 | 4 bytes | Playable civs array count (MAX_PLAYERS) |
| uint8[] | count bytes | Playable civs array elements (bool) |
| uint32 | 4 bytes | Player color array count (only as many entries as real players in the game, not MAX_PLAYERS) |
| varstring[] | var bytes | Player colors |
| uint8 | 1 byte | Private game (bool) |
| uint8 | 1 byte | Quick combat (bool) |
| uint8 | 1 byte | Quick combat default (bool) |
| int32 | 4 bytes | Quick handicap (HandicapTypes; -1 = NO_HANDICAP) |
| uint8 | 1 byte | Quickstart (bool) |
| uint8 | 1 byte | Random world size (bool) |
| uint8 | 1 byte | Random map script (bool) |
| uint32 | 4 bytes | Ready players array count (MAX_PLAYERS) |
| uint8[] | count bytes | Ready players array elements (bool) |

### Sea Level

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Sea level enum |
| uint32 | 4 bytes | Sea level info id |
| uint32 | 4 bytes | Sea level info civilopedia |
| varstring | var bytes | Sea level display name |
| uint32 | 4 bytes | Sea level help |
| uint32 | 4 bytes | Sea level disabled help |
| uint32 | 4 bytes | Sea level strategy |
| varstring | var bytes | Sea level type |
| varstring | var bytes | Sea level text key |
| varstring | var bytes | Sea level display name 2 |
| int32 | 4 bytes | Sea level change |
| uint8 | 1 byte | Dummy value 2 |

### Additional Pre-Game Fields

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Slot claim array 2 count (a second, independent slot-claim array - see Player & Map Info's own Slot claim array) |
| uint32[] | (count * 4) bytes | Slot claim array 2 elements: 0=UNASSIGNED, 1=RESERVED, 2=ASSIGNED |
| uint32 | 4 bytes | Slot status array 2 count (a second, independent slot-status array - see Player & Map Info's own Slot status array) |
| uint32[] | (count * 4) bytes | Slot status array 2 elements: 0=OPEN, 1=COMPUTER, 2=CLOSED, 3=TAKEN, 4=OBSERVER |
| varstring | var bytes | SMTP host (typically empty) |
| uint32 | 4 bytes | Sync random seed |
| int32 | 4 bytes | Target score |
| uint32 | 4 bytes | Team type array 2 count (a second, independent team-type array - see Player & Map Info's own Team type array) |
| uint32[] | (count * 4) bytes | Team type array 2 elements |
| uint8 | 1 byte | Transferred map (bool) |

### Turn Speed Data

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Turn timer id |
| uint32 | 4 bytes | Turn timer civilopedia |
| varstring | var bytes | Turn timer display name |
| uint32 | 4 bytes | Turn timer help |
| uint32 | 4 bytes | Turn timer disabled help |
| uint32 | 4 bytes | Turn timer strategy |
| varstring | var bytes | Turn timer type (e.g. TURNTIMER_SNAIL) |
| varstring | var bytes | Turn timer text key |
| varstring | var bytes | Turn timer display name 2 |
| uint32 | 4 bytes | Turn timer base time |
| uint32 | 4 bytes | Turn timer city bonus |
| uint32 | 4 bytes | Turn timer unit bonus |
| uint32 | 4 bytes | Turn timer first-turn multiplayer |
| uint32 | 4 bytes | Turn timer type enum |
| uint8 | 1 byte | Turn timer city screen blocked |
| uint32 | 4 bytes | Victory flags array count (one per victory condition - 5 in vanilla/BNW) |
| uint8[] | count bytes | Victory flags |
| uint32 | 4 bytes | White flag array count |
| uint8[] | count bytes | White flag array elements (bool) |

### World Size Data

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Serialization version (typically 2) |
| uint32 | 4 bytes | World info id |
| uint32 | 4 bytes | World info civilopedia (v2 only) |
| varstring | var bytes | World size display name |
| varstring | var bytes | World size help text |
| uint32 | 4 bytes | World size disabled help |
| uint32 | 4 bytes | World size strategy |
| varstring | var bytes | World size type (e.g. WORLDSIZE_STANDARD) |
| varstring | var bytes | World size text key |
| varstring | var bytes | World size display name 2 |
| uint32 | 4 bytes | Default players |
| uint32 | 4 bytes | Default minor civs |
| uint32 | 4 bytes | Fog tiles per barbarian camp |
| uint32 | 4 bytes | Num natural wonders |
| uint32 | 4 bytes | Unit name modifier |
| uint32 | 4 bytes | Target num cities |
| uint32 | 4 bytes | Num free building resources |
| uint32 | 4 bytes | Building class prereq modifier |
| int32 | 4 bytes | Max conscript modifier |
| uint32 | 4 bytes | Grid width |
| uint32 | 4 bytes | Grid height |
| uint32 | 4 bytes | Max active religions (v2 only) |
| int32 | 4 bytes | Terrain grain change |
| int32 | 4 bytes | Feature grain change |
| uint32 | 4 bytes | Research percent |
| uint32 | 4 bytes | Advanced start points mod |
| uint32 | 4 bytes | Num cities unhappiness percent |
| uint32 | 4 bytes | Num cities policy cost mod |
| uint32 | 4 bytes | Num cities tech cost mod |
| uint32 | 4 bytes | World size enum (v2 only) |

### Game Options

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Game option array count |
| varstring | var bytes | Game option name (repeated per element) |
| uint32 | 4 bytes | Game option enabled (repeated per element) |
| uint32 | 4 bytes | Map option array count (per the active map script's own custom options; 0 for a fixed .Civ5Map file) |
| varstring | var bytes | Map option name (repeated per element) |
| uint32 | 4 bytes | Map option value (repeated per element) |
| varstring | var bytes | Game version 2 |
| uint32 | 4 bytes | Should-notify Steam invite array count |
| uint8[] | count bytes | Should-notify Steam invite array elements (bool) |
| uint32 | 4 bytes | Should-notify email array count |
| uint8[] | count bytes | Should-notify email array elements (bool) |
| uint32 | 4 bytes | Turn-notify email address array count |
| varstring[] | var bytes | Turn-notify email address array elements |

### Compressed Block

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[8] | 8 bytes | Padding, always `[2,0,0,0,0,0,1,0]` |

ZLIB-compressed data follows, starting with the 0x789C signature. Everything below is inside the decompressed contents.

### Decompressed Header

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Save file version — 0x0B (11) enables the named arrays |
| uint32 | 4 bytes | End turn messages sent |
| uint32 | 4 bytes | Elapsed game turns (turns played since Start turn - the displayed/current turn is Start turn + this) |
| uint32 | 4 bytes | Start turn (the turn this game began counting from - 0 for a normal game, or an offset for a scenario/late-era start) |
| uint32 | 4 bytes | Winning turn (0 if no winner yet; relative to Start turn, same as Elapsed game turns) |
| int32 | 4 bytes | Start year — defaults to -4000 (4000 BC); only differs when the map file itself has a custom start year baked in (scenario/mod maps), independent of the era chosen at game setup |
| int32 | 4 bytes | Estimate end turn |
| int32 | 4 bytes | Default estimate end turn |
| int32 | 4 bytes | Turn slice |
| int32 | 4 bytes | Cutoff slice |
| int32 | 4 bytes | Num cities (global count across all civs, not just the player) |
| int32 | 4 bytes | Total population |
| int32 | 4 bytes | No-nukes count |
| int32 | 4 bytes | Nukes exploded |
| int32 | 4 bytes | Max population |
| int32 | 4 bytes | Unused1 (typically 0) |
| int32 | 4 bytes | Unused2 (typically 0) |
| int32 | 4 bytes | Unused3 (typically 0) |
| int32 | 4 bytes | Init population |
| int32 | 4 bytes | Init land |
| int32 | 4 bytes | Init tech |
| int32 | 4 bytes | Init wonders |
| int32 | 4 bytes | AI auto play |
| int32 | 4 bytes | Total religion tech cost |
| int32 | 4 bytes | Cached world religion tech progress |
| int32 | 4 bytes | United Nations countdown |
| int32 | 4 bytes | Num victory votes tallied |
| int32 | 4 bytes | Num victory votes expected |
| int32 | 4 bytes | Votes needed for diplo victory |
| int32 | 4 bytes | Map score mod |
| byte | 1 byte | Score dirty (bool) |
| byte | 1 byte | Circumnavigated (bool) |
| byte | 1 byte | Final initialized (bool) |
| byte | 1 byte | Hot PBEM between turns (bool) |
| byte | 1 byte | Nukes valid (bool) |
| byte | 1 byte | End game tech researched (bool) |
| byte | 1 byte | Tuner ever connected (bool) |
| byte | 1 byte | Tutorial ever attacked (bool) |
| byte | 1 byte | Static tutorial active (bool) |
| byte | 1 byte | Ever right-click moved (bool) |
| uint32 | 4 bytes | Advisor messages viewed array count |
| varstring[] | var bytes | Advisor message IDs (e.g. `RIGHT_CLICK_MOVE`, `CHOOSE_IDEOLOGY`) |
| uint32 | 4 bytes | Handicap: 0=HANDICAP_SETTLER, 1=HANDICAP_CHIEFTAIN, 2=HANDICAP_WARLORD, 3=HANDICAP_PRINCE, 4=HANDICAP_KING, 5=HANDICAP_EMPEROR, 6=HANDICAP_IMMORTAL, 7=HANDICAP_DEITY |
| int32 | 4 bytes | Pause player (-1 if none) |
| int32 | 4 bytes | AI auto play return player (-1 if none) |
| int32 | 4 bytes | Best land unit player |
| int32 | 4 bytes | Winner (-1 if no winner yet; whole-game winner, not tied to whichever player this file happens to track) |
| uint32 | 4 bytes | Victory: 0=VICTORY_TIME, 1=VICTORY_SPACE_RACE, 2=VICTORY_DOMINATION, 3=VICTORY_CULTURAL, 4=VICTORY_DIPLOMATIC (0xFFFFFFFF if none yet) |
| uint32 | 4 bytes | GameState: 0=GAMESTATE_ON, 1=GAMESTATE_OVER, 2=GAMESTATE_EXTENDED |
| int32 | 4 bytes | Best wonders player |
| int32 | 4 bytes | Best policies player |
| int32 | 4 bytes | Best great people player |
| int32 | 4 bytes | Religion tech (-1 if none) |
| int32 | 4 bytes | Industrial route (1=railroad, matching `.civ5map`'s RouteType) |
| varstring | var bytes | Script data |
| int32[64] | 256 bytes | End turn messages received, per player (no length prefix) |
| int32[64] | 256 bytes | Rank -> player: which player holds each rank |
| int32[64] | 256 bytes | Player -> rank: each player's rank |
| int32[64] | 256 bytes | Player score |
| int32[64] | 256 bytes | Rank -> team: which team holds each rank |
| int32[64] | 256 bytes | Team -> rank: each team's rank |
| int32[64] | 256 bytes | Team score |

### Version-Dependent Unit Data

If save file version == 0x0B (11):

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Unit name array count |
| (varstring,uint32)[] | var bytes | Unit name array: (unit name, count created) per unit type |
| uint32 | 4 bytes | Unit class array count |
| (varstring,uint32)[] | var bytes | Unit class array: (unit class, count created) per unit class |
| uint32 | 4 bytes | Building class array count |
| (varstring,uint32)[] | var bytes | Building class array: (building class, count created) per building class |
| byte[2366] | 2366 bytes | Unknown padding |

Otherwise:

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Unit created count array length |
| (uint32,int32)[] | var bytes | Unit created count: (type name hash, count created) per unit type |
| uint32 | 4 bytes | Unit class created count array length |
| (uint32,int32)[] | var bytes | Unit class created count: (type name hash, count created) per unit class |
| uint32 | 4 bytes | Building class created count array length |
| (uint32,int32)[] | var bytes | Building class created count: (type name hash, count created) per building class |
| uint32 | 4 bytes | Project created count array length |
| (uint32,int32)[] | var bytes | Project created count: (type name hash, count created) per project type |
| uint32 | 4 bytes | Vote outcome array length |
| (uint32,int32)[] | var bytes | Vote outcome: (type name hash, outcome) per vote type |
| uint32 | 4 bytes | Secretary general timer array length |
| (uint32,int32)[] | var bytes | Secretary general timer: (type name hash, timer) per vote source type |
| uint32 | 4 bytes | Vote timer array length |
| (uint32,int32)[] | var bytes | Vote timer: (type name hash, timer) per vote source type |
| uint32 | 4 bytes | Diplo vote array length |
| (uint32,int32)[] | var bytes | Diplo vote: (type name hash, value) per vote source type |
| int32[63] | 252 bytes | Votes cast |
| int32[63] | 252 bytes | Previous votes cast |
| int32[63] | 252 bytes | Num votes for team |
| uint32 | 4 bytes | Special unit valid array length |
| (uint32,bool)[] | var bytes | Special unit valid: (type name hash, is valid) per special unit type — 1-byte bool value, not 4 |
| uint32 | 4 bytes | Team victory rank array length |
| (uint32,int32[5])[] | var bytes | Team victory rank: (type name hash, rank per victory-point-award) per victory type — the sub-array size (5) isn't stored in the file |
| uint32 | 4 bytes | Destroyed cities array count |
| varstring[] | var bytes | Destroyed city names |

### Great Person

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Great person array count |
| varstring[] | var bytes | Great person names |

### Votes

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32 | 4 bytes | Vote selections: num slots |
| int32 | 4 bytes | Vote selections: last index |
| int32 | 4 bytes | Vote selections: free list head |
| int32 | 4 bytes | Vote selections: free list count |
| int32 | 4 bytes | Vote selections: current ID |
| int32[numSlots] | numSlots * 4 bytes | Vote selections: next-free-index per slot |
| uint32 | 4 bytes | Vote selections: entry count (0 when no World Congress votes have been proposed yet; entry format not decoded) |
| int32 | 4 bytes | Votes triggered: num slots |
| int32 | 4 bytes | Votes triggered: last index |
| int32 | 4 bytes | Votes triggered: free list head |
| int32 | 4 bytes | Votes triggered: free list count |
| int32 | 4 bytes | Votes triggered: current ID |
| int32[numSlots] | numSlots * 4 bytes | Votes triggered: next-free-index per slot |
| uint32 | 4 bytes | Votes triggered: entry count (0 when no votes have been triggered yet; entry format not decoded) |

### Random State

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Map rand: version |
| uint32 | 4 bytes | Map rand: seed |
| uint32 | 4 bytes | Map rand: call count |
| uint32 | 4 bytes | Map rand: reset count |
| byte | 1 byte | Map rand: extended callstack debugging flag (always 0 in a release build) |
| uint32 | 4 bytes | Other rand: version |
| uint32 | 4 bytes | Other rand: seed |
| uint32 | 4 bytes | Other rand: call count |
| uint32 | 4 bytes | Other rand: reset count |
| byte | 1 byte | Other rand: extended callstack debugging flag (always 0 in a release build) |
| uint32 | 4 bytes | Replay message version |

### Save File Replay Events

Same format as replay events in the shared Replay File Format section - the save file's own event log, read by the same code.

### Session & Barbarian State

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32 | 4 bytes | Num sessions (how many times this game has been reloaded) |
| uint32 | 4 bytes | Plot extra yield array count |
| (int32,int32,uint32,int32[])[] | var bytes | Plot extra yield array elements: (x, y, extra-yield count, extra-yield values per yield type) |
| uint32 | 4 bytes | Plot extra cost array count |
| (int32,int32,int32)[] | var bytes | Plot extra cost array elements: (x, y, cost) |
| uint8 | 1 byte | Archaeology triggered (bool) |
| int32 | 4 bytes | Earliest barbarian release turn |

### Game Deals

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Deals format version |
| uint32 | 4 bytes | Proposed deal array count |
| Deal[] | var bytes | Proposed deals (in-progress negotiations; typically empty) |
| uint32 | 4 bytes | Current deal array count |
| Deal[] | var bytes | Currently active deals |
| uint32 | 4 bytes | Historical deal array count |
| Deal[] | var bytes | Past (expired/cancelled) deals |

#### Deal Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Deal format version |
| int32 | 4 bytes | From player |
| int32 | 4 bytes | To player |
| int32 | 4 bytes | Final turn |
| int32 | 4 bytes | Duration |
| int32 | 4 bytes | Start turn |
| uint8 | 1 byte | Considering for renewal (bool) |
| uint8 | 1 byte | Checked for renewal (bool, only if deal format version >= 3) |
| uint8 | 1 byte | Deal cancelled (bool) |
| int32 | 4 bytes | Peace treaty type |
| int32 | 4 bytes | Surrendering player |
| int32 | 4 bytes | Demanding player |
| int32 | 4 bytes | Requesting player |
| uint32 | 4 bytes | Traded item array count |
| TradedItem[] | var bytes | Traded items |

#### Traded Item Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Traded item format version |
| int32 | 4 bytes | Item type (TradeableItems: -1=NONE, 0=GOLD, 1=GOLD_PER_TURN, 2=MAPS, 3=RESOURCES, 4=CITIES, 5=UNITS, 6=OPEN_BORDERS, 7=DEFENSIVE_PACT, 8=RESEARCH_AGREEMENT, 9=TRADE_AGREEMENT, 10=PERMANENT_ALLIANCE, 11=SURRENDER, 12=TRUCE, 13=PEACE_TREATY, 14=THIRD_PARTY_PEACE, 15=THIRD_PARTY_WAR, 16=THIRD_PARTY_EMBARGO, 17=ALLOW_EMBASSY, 18=DECLARATION_OF_FRIENDSHIP, 19=VOTE_COMMITMENT) |
| int32 | 4 bytes | Duration |
| int32 | 4 bytes | Final turn |
| int32 | 4 bytes | Data 1 (meaning depends on item type: GOLD/GOLD_PER_TURN=amount, RESOURCES=ResourceTypes index, CITIES=city's X coordinate, THIRD_PARTY_PEACE/THIRD_PARTY_WAR=TeamTypes (not a player index), VOTE_COMMITMENT=ResolutionID, other types unused/0) |
| int32 | 4 bytes | Data 2 (RESOURCES=amount, CITIES=city's Y coordinate, VOTE_COMMITMENT=VoteChoice, otherwise unused/0) |
| int32 | 4 bytes | Data 3 (only if traded item format version >= 2; VOTE_COMMITMENT=NumVotes, otherwise unused/0) |
| uint8 | 1 byte | Flag 1 (bool, only if traded item format version >= 2; VOTE_COMMITMENT=is this a repeal vote, otherwise always false) |
| int32 | 4 bytes | From player |
| uint8 | 1 byte | From renewed (bool) |
| uint8 | 1 byte | To renewed (bool) |

### Game Religions

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Religions format version |
| int32 | 4 bytes | Minimum faith for next pantheon (only if religions format version >= 3) |
| uint32 | 4 bytes | Legacy minimum faith for next great prophet, discarded (only if religions format version < 4) |
| uint32 | 4 bytes | Current religion array count (only if religions format version >= 2) |
| Religion[] | var bytes | Current religions - both pantheons and founded religions are listed here (only if religions format version >= 2) |

#### Religion Element

A pantheon has its own entry too: religion type 0 (RELIGION_PANTHEON), pantheon bool set, exactly 1 belief, and a meaningless/uninitialized holy city X/Y (a pantheon has no holy city).

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Religion format version |
| int32 | 4 bytes | Religion type |
| int32 | 4 bytes | Founder player |
| int32 | 4 bytes | Holy city X |
| int32 | 4 bytes | Holy city Y |
| int32 | 4 bytes | Turn founded |
| uint8 | 1 byte | Pantheon (bool, only if religion format version >= 2) |
| uint8 | 1 byte | Enhanced (bool, only if religion format version >= 4) |
| byte[128] | 128 bytes | Custom name: fixed-size null-terminated buffer (only if religion format version >= 3) |
| ReligionBeliefs | var bytes | Beliefs |

#### Religion Beliefs Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Beliefs format version |
| int32[22] | 88 bytes | Modifiers: faith from dying units, river happiness, plot culture cost, city-range strike, combat vs. enemy cities, combat vs. friendly cities, friendly heal change, city-state friendship, land barbarian conversion percent, spread strength, spread distance, prophet strength, prophet cost, missionary strength, missionary cost, friendly city-state spread, great person expended faith, city-state minimum influence, city-state influence, other-religion pressure erosion, spy pressure, inquisitor pressure retention |
| int32 | 4 bytes | Faith building tourism (only if beliefs format version >= 2) |
| int32 | 4 bytes | Obsolete era |
| int32 | 4 bytes | Resource revealed |
| int32 | 4 bytes | Spread modifier doubling tech |
| uint32 | 4 bytes | Belief array count |
| uint32[] | (count * 4) bytes | Belief hashes (hash of each chosen belief's type string) |
| uint32 | 4 bytes | Building class override array count (a ruleset-wide constant - GC.getNumBuildingClassInfos() - not a per-religion value) |
| (uint32,int32)[] | var bytes | Building class overrides: (type hash, value); value present only when hash is non-zero |

### Game Culture

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Culture format version |
| uint32 | 4 bytes | Current great work array count |
| GreatWork[] | var bytes | Current great works |
| uint8 | 1 byte | Reported someone influential (bool, only if culture format version >= 2) |

#### Great Work Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Great work format version |
| varstring | var bytes | Legacy great work name, discarded (only if great work format version == 1) |
| varstring | var bytes | Great person name |
| int32 | 4 bytes | Great work type: 1-indexed, names the specific work |
| int32 | 4 bytes | Great work class (only if great work format version >= 3): 1=Art, 2=Artifact, 3=Literature, 4=Music |
| int32 | 4 bytes | Turn founded |
| int32 | 4 bytes | Era |
| int32 | 4 bytes | Player |

### Game Leagues

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Leagues format version |
| int32 | 4 bytes | Generated ID count (only if leagues format version >= 4) |
| uint32 | 4 bytes | Active league array count |
| League[] | var bytes | Active leagues (0 or 1 - one World Congress/United Nations per game) |
| int32 | 4 bytes | Num leagues ever founded (only if leagues format version >= 2) |
| int32 | 4 bytes | Diplomatic victor player (only if leagues format version >= 3, else NO_PLAYER/-1) |
| int32 | 4 bytes | Last era trigger (only if leagues format version >= 5, else NO_ERA/-1) |

#### League Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | League format version |
| int32 | 4 bytes | League ID |
| uint8 | 1 byte | United Nations (bool, only if league format version >= 4) |
| uint8 | 1 byte | In session (bool, only if league format version >= 2) |
| int32 | 4 bytes | Turns until next session |
| int32 | 4 bytes | Num resolutions ever enacted |
| uint32 | 4 bytes | Enact proposal array count |
| Resolution[] | var bytes | Enact proposals (Resolution + format version, no extra fields) |
| uint32 | 4 bytes | Repeal proposal array count |
| (Resolution,int32,VoterDecision)[] | var bytes | Repeal proposals: (Resolution, format version, target resolution ID, repeal decision) |
| uint32 | 4 bytes | Active resolution array count |
| (Resolution,uint32,int32)[] | var bytes | Active resolutions: (Resolution, format version, [legacy discarded int32 only if version < 2], turn enacted) |
| uint32 | 4 bytes | Member array count |
| LeagueMember[] | var bytes | Members |
| int32 | 4 bytes | Host player (only if league format version >= 3) |
| uint32 | 4 bytes | Project array count (only if league format version >= 5) |
| LeagueProject[] | var bytes | Projects (only if league format version >= 5) |
| int32 | 4 bytes | Consecutive hosted sessions (only if league format version >= 7) |
| int32 | 4 bytes | Assigned name (only if league format version >= 7) |
| byte[128] | 128 bytes | Custom name: fixed-size null-terminated buffer (only if league format version >= 7) |
| int32 | 4 bytes | Last special session (only if league format version >= 8) |
| int32 | 4 bytes | Current special session (only if league format version >= 8) |
| uint32 | 4 bytes | Enact-proposals-on-hold array count (only if league format version >= 8) |
| Resolution[] | var bytes | Enact proposals on hold (only if league format version >= 8) |
| uint32 | 4 bytes | Repeal-proposals-on-hold array count (only if league format version >= 8) |
| (Resolution,int32,VoterDecision)[] | var bytes | Repeal proposals on hold (only if league format version >= 8) |

#### League Member Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32 | 4 bytes | Player |
| int32 | 4 bytes | Extra votes (only if league format version >= 9) |
| varstring | var bytes | Vote sources: human-readable breakdown text, e.g. "[NEWLINE][ICON_BULLET]4 from membership..." (only if league format version >= 10) |
| uint8 | 1 byte | May propose (bool) |
| int32 | 4 bytes | Proposals made (only if league format version >= 12) |
| int32 | 4 bytes | Votes |
| int32 | 4 bytes | Abstained votes (only if league format version >= 13) |
| uint8 | 1 byte | Ever been host (bool, only if league format version >= 14) |
| uint8 | 1 byte | Always been host (bool, only if league format version >= 14, real default true when absent) |

#### League Project Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32 | 4 bytes | Project type |
| uint32 | 4 bytes | Production list length |
| int32[] | var bytes | Per-member production contribution |
| uint8 | 1 byte | Complete (bool, only if league format version >= 6) |
| uint8 | 1 byte | Progress warning sent (bool, only if league format version >= 11) |

#### Resolution Element

Shared base for an enact proposal, a repeal proposal's target, and an already-active resolution.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Resolution format version |
| int32 | 4 bytes | Resolution ID (only if resolution format version >= 2, else generated fresh at load, not read) |
| int32 | 4 bytes | Resolution type |
| int32 | 4 bytes | League ID |
| ResolutionEffects | var bytes | Effects |
| VoterDecision | var bytes | Voter decision (every member's cast vote) |
| ProposerDecision | var bytes | Proposer decision (the proposing player's own vote) |

An enact/repeal proposal adds its own format version (uint32) plus, for a repeal proposal, a target resolution ID (int32) and a repeal VoterDecision. An active resolution adds its own format version (uint32), a legacy discarded int32 (only if that version < 2), and a turn-enacted int32.

#### Resolution Effects Element

The full set of possible resolution effects; only the fields relevant to the resolution's type are meaningful, the rest sit at zero.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Effects format version |
| uint8 | 1 byte | Diplomatic victory (bool) |
| uint8 | 1 byte | Change league host (bool, only if effects format version >= 2) |
| int32 | 4 bytes | One-time gold |
| int32 | 4 bytes | One-time gold percent |
| uint8 | 1 byte | Raise city-state influence to neutral (bool) |
| int32 | 4 bytes | League project enabled (only if effects format version >= 3) |
| int32 | 4 bytes | Gold per turn |
| int32 | 4 bytes | Resource quantity |
| uint8 | 1 byte | Embargo city-states (bool) |
| uint8 | 1 byte | Embargo player (bool) |
| uint8 | 1 byte | No resource happiness (bool) |
| int32 | 4 bytes | Unit maintenance gold percent |
| int32 | 4 bytes | Member discovered tech mod |
| int32 | 4 bytes | Culture per wonder (only if effects format version >= 4) |
| int32 | 4 bytes | Culture per natural wonder (only if effects format version >= 4) |
| uint8 | 1 byte | No training nuclear weapons (bool, only if effects format version >= 5) |
| int32 | 4 bytes | Votes for following religion (only if effects format version >= 6) |
| int32 | 4 bytes | Holy city tourism (only if effects format version >= 6) |
| int32 | 4 bytes | Religion spread strength mod (only if effects format version >= 9) |
| int32 | 4 bytes | Votes for following ideology (only if effects format version >= 6) |
| int32 | 4 bytes | Other ideology rebellion mod (only if effects format version >= 6) |
| int32 | 4 bytes | Artsy great person rate mod (only if effects format version >= 7) |
| int32 | 4 bytes | Sciencey great person rate mod (only if effects format version >= 7) |
| int32 | 4 bytes | Great person tile improvement culture (only if effects format version >= 8) |
| int32 | 4 bytes | Landmark culture (only if effects format version >= 8) |

#### Voter/Proposer Decision Elements

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Decision format version |
| int32 | 4 bytes | Decision type |

A ProposerDecision adds its own format version (uint32) and a single PlayerVote (the proposer's own vote). A VoterDecision adds its own format version (uint32), a count (uint32), then that many PlayerVote entries - one per player who cast a vote.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | PlayerVote format version |
| int32 | 4 bytes | Player |
| int32 | 4 bytes | Num votes |
| int32 | 4 bytes | Choice |

### Game Trade

The last of the game's whole-object sections - every active trade route (land or sea, international or internal) in the game, plus a running tech-difference matrix used to gate trade route yields.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Trade format version |
| uint32 | 4 bytes | Trade connection array count |
| TradeConnection[] | var bytes | Trade connections |
| int32[22][22] | 1936 bytes | Tech difference matrix, MAX_MAJOR_CIVS x MAX_MAJOR_CIVS (only if trade format version >= 3, else all zero) |
| int32 | 4 bytes | Next trade connection ID |

#### Trade Connection Element

| Type | Size | Description |
| ---- | ---- | ----------- |
| int32 | 4 bytes | Connection ID (only if trade format version >= 1, else MAX_INT - not read from the stream; -1 when empty) |
| int32 | 4 bytes | Origin X (-1 when empty) |
| int32 | 4 bytes | Origin Y (-1 when empty) |
| int32 | 4 bytes | Destination X (-1 when empty) |
| int32 | 4 bytes | Destination Y (-1 when empty) |
| int32 | 4 bytes | Origin owner (player; -1/NO_PLAYER when empty) |
| int32 | 4 bytes | Destination owner (player; -1/NO_PLAYER when empty) |
| int32 | 4 bytes | Domain (DomainTypes: 0=SEA, 1=AIR, 2=LAND, 3=IMMOBILE, 4=HOVER; -1=NO_DOMAIN when empty) |
| int32 | 4 bytes | Connection type (TradeConnectionType: 0=INTERNATIONAL, 1=FOOD, 2=PRODUCTION; 3 when empty - a count sentinel reused as "none") |
| int32 | 4 bytes | Trade unit location index (position along the route's path) |
| uint8 | 1 byte | Trade unit moving forward (bool) |
| int32 | 4 bytes | Trade unit's unit ID |
| int32 | 4 bytes | Circuits completed |
| int32 | 4 bytes | Circuits to complete |
| int32 | 4 bytes | Turn route was completed (only if trade format version >= 2) |
| uint32 | 4 bytes | Plot list length (the route's tile path) |
| (int32,int32)[] | var bytes | Plot list: (x, y) per tile |
| int32[6] | 24 bytes | Origin yields per yield type (food/production/gold/science/culture/faith) |
| int32[6] | 24 bytes | Destination yields per yield type, same order |

### Embedded Database

This is a size-prefixed embedded SQLite database blob (`Civ5SavedGameDatabase.db`). Its schema is fixed - exactly one generic key-value table:

```sql
CREATE TABLE SimpleValues(Name TEXT Primary Key, Value VARIANT)
```

plus its auto-generated unique index (`sqlite_autoindex_SimpleValues_1`).

For a vanilla game this table is empty (SQLite version `3.7.17`, page size 1024, 3 pages, 3072 bytes total). Mods can read and write arbitrary rows into it through the modding API, so the blob size and contents vary by mod.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Blob size |
| byte[] | var bytes | Raw blob content, starting with the literal SQLite file header `"SQLite format 3\0"` |

### Map Header

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Map format version (typically 1) |
| int32 | 4 bytes | Grid width |
| int32 | 4 bytes | Grid height |
| int32 | 4 bytes | Land plot count |
| int32 | 4 bytes | Owned plot count |
| int32 | 4 bytes | Num natural wonders |
| int32 | 4 bytes | Top latitude |
| int32 | 4 bytes | Bottom latitude |
| uint8 | 1 byte | Wrap X (bool) |
| uint8 | 1 byte | Wrap Y (bool) |
| byte[16] | 16 bytes | Map GUID: Data1(4)+Data2(2)+Data3(2)+Data4(8), standard GUID layout - unique per map |
| uint32 | 4 bytes | Resource count array length |
| (uint32,int32)[] | var bytes | Total resource counts across the whole map: (type hash, count); count present only when hash is non-zero |
| uint32 | 4 bytes | Resource-on-land count array length |
| (uint32,int32)[] | var bytes | Resource counts restricted to land tiles only, same shape |
