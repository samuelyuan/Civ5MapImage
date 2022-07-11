# Civ5MapImage

This program will convert a Civ 5 map with the file extension .Civ5Map to a PNG image.

### Command-Line Usage

```
./Civ5MapImage.exe -input=[input filename] -output=[output filename]
```

Example
```
./Civ5MapImage.exe -input=earth.Civ5Map -output=earth.png
```

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth.png" alt="earth" width="550" height="300" />
</div>

### Examples

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/europe.png" alt="europe" width="415" height="300" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/world.png" alt="world" width="550" height="300" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/india.png" alt="india" width="300" height="300" />
</div>

### File format

This file format covers .civ5map files, which stores the map data. All data is stored in little endian.

#### Header

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint8 | 1 byte | ScenarioVersion (The leftmost 4 bits are for scenario. The rightmost 4 bits are for version, which is set to 12 for newer files.) |
| uint32 | 4 bytes | Map width |
| uint32 | 4 bytes | Map height |
| uint8 | 1 byte| Number of players |
| uint8[4] | 4 bytes | Settings (hasWorldWrap, hasRandomResources, hasRandomGoodies) |
| uint32 | 4 bytes | TerrainDataSize (Length of terrain list) |
| uint32 | 4 bytes | FeatureTerrainDataSize (Length of feature terrain list) |
| uint32 | 4 bytes | FeatureWonderDataSize (Length of feature wonder list) |
| uint32 | 4 bytes | ResourceDataSize  (Length of resource list) |
| uint32 | 4 bytes | ModDataSize |
| uint32 | 4 bytes | MapNameLength |
| uint32 | 4 bytes | MapDescriptionLength |

Following the header, is a list of strings whose size is determined in the header. Each string list will have a zero byte to split items.

#### Geography list data

| Type | Size | Description |
| ---- | ---- | ----------- |
| String list | TerrainDataSize bytes | Terrain (e.g. TERRAIN_GRASS) |
| String list | FeatureTerrainDataSize bytes | Feature terrain (e.g. FEATURE_ICE) |
| String list | FeatureWonderDataSize bytes | Feature wonders(e.g. FEATURE_CRATER, FEATURE_FUJI) |
| String list | ResourceDataSize bytes | Resources (e.g. RESOURCE_IRON) |
| String | ModDataSize bytes | modData |
| String | MapNameLength bytes | Map name |
| String | MapDescriptionLength bytes | Map description |

If the version is >= 11, there will be an unknown string in the file

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | unknownStringLength |
| String list | unknownStringLength bytes | unknownStringBytes |

#### Map geography

The map data is inverted, which means that the bottommost row rendered on the screen is stored on the top row of the array and the topmost row rendered on the screen is stored on the last row of the array.

| Type | Size | Description |
| ---- | ---- | ----------- |
| MapTile[Height][Width] | (Height * Width * 8) bytes | Map geography |

#### Map tile data

The size of this struct is 8 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint8 | 1 byte | TerrainType (index in terrain list) |
| uint8 | 1 byte | ResourceType (index in resource list, 0xff if none) |
| uint8 | 1 byte | FeatureTerrainType (index in feature terrain list, 0xff if none) |
| uint8 | 1 byte | RiverData (The low 3 bits means the tile border has a river. Only 3 edges needs to be marked per tile. 4 is southwest edge, 2 is southeast edge, 1 is eastern edge) |
| uint8 | 1 byte | Elevation (0 = flat, 1 = hills, 2 = mountain) |
| uint8 | 1 byte | Continent |
| uint8 | 1 byte | FeatureWonderType (index in feature wonder list, 0xff if none) |
| uint8 | 1 byte | ResourceAmount |

#### Game description header

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[68] | 68 bytes | Unknown |
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

#### Game description data

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

#### City format

| Type | Size | Description |
| ---- | ---- | ----------- |
| byte[64] | 64 bytes | City name |
| byte | 1 byte | Owner |
| byte | 1 byte | Settings |
| uint16 | 1 byte | Population |
| uint32 | 1 byte | Health (100% health is 100000) |
| byte[] | 32 bytes for version 11, 64 bytes for version 12 | Building data |

#### Map tile improvement properties

This data is stored at the end of the file and contains the map improvements. It is separate from the map terrain array mentioned earlier. In between the game description data and map tile properties, there is a block of unknown data followed by the team and civilization information, but this part still hasn't been decoded. In order to access this block, the program reads the map tile data starting from the end of the file instead of sequentially reading data like before.

| Type | Size | Description |
| ---- | ---- | ----------- |
| MapTileImprovement[Height][Width] | (Height * Width * 8) bytes | 2D array of map tile improvements |

#### Map tile improvement data

The size of this struct is 8 bytes.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint16 | 2 bytes | City id |
| byte[2] | 2 bytes | Unknown |
| uint8 | 1 byte | Owner |
| uint8 | 1 byte | Improvement |
| uint8 | 1 byte | RouteType (0 = road, 1 = railroad, 0xff = none) |
| uint8 | 1 byte | RouteOwner |
