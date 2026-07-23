package fileio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Constants for Civ5 map file processing
const (
	// City state offset (city states start at index 32)
	CityStateOffset = 32

	// Scenario version encodes both a format version and a scenario flag in one byte
	VersionMask  = 0xF
	ScenarioBits = 4

	// Map format versions that change binary layout
	MapVersion11 = 11
	MapVersion12 = 12

	// Version-specific data sizes
	BuildingDataSizeV11 = 32
	BuildingDataSizeV12 = 64
	UnitDataSizeV11     = 48 // Size of Civ5UnitHeaderV11
	UnitDataSizeV12     = 84 // Size of Civ5UnitHeaderV12

	// String buffer sizes
	PlayerNameSize   = 64
	LeaderNameSize   = 64
	CivNameSize      = 64
	CivTypeSize      = 64
	TeamColorSize    = 64
	EraSize          = 64
	HandicapSize     = 64
	PromotionSizeV11 = 32
	PromotionSizeV12 = 64

	// Flag bit positions
	IsPuppetStateFlag = 1
	IsOccupiedFlag    = 2

	// Special values
	InvalidCityId = -1    // Sentinel used in our parsed data model
	RawNoCityId   = 65535 // Sentinel used in the raw file format (uint16 max)

	// Data structure sizes
	CivDataSize = 436
)

type Civ5MapHeader struct {
	ScenarioVersion        uint8
	Width                  uint32
	Height                 uint32
	Players                uint8
	Settings               [4]uint8
	TerrainDataSize        uint32
	FeatureTerrainDataSize uint32
	FeatureWonderDataSize  uint32
	ResourceDataSize       uint32
	ModDataSize            uint32
	MapNameLength          uint32
	MapDescriptionLength   uint32
}

type Civ5MapTile struct {
	TerrainType        uint8
	ResourceType       uint8
	FeatureTerrainType uint8
	RiverData          uint8
	Elevation          uint8
	Continent          uint8
	FeatureWonderType  uint8
	ResourceAmount     uint8
}

type Civ5MapTilePhysical struct {
	X                  int
	Y                  int
	TerrainType        int
	ResourceType       int
	FeatureTerrainType int
	RiverData          int
	Elevation          int
	Continent          int
	FeatureWonderType  int
	ResourceAmount     int
}

type Civ5GameDescriptionHeader struct {
	Unknown1              [68]byte
	MaxTurns              uint32
	Unknown2              [4]byte
	StartYear             int32
	PlayerCount           uint8
	CityStateCount        uint8
	TeamCount             uint8
	Unknown3              byte
	ImprovementDataSize   uint32
	UnitTypeDataSize      uint32
	TechTypeDataSize      uint32
	PolicyTypeDataSize    uint32
	BuildingTypeDataSize  uint32
	PromotionTypeDataSize uint32
	UnitDataSize          uint32
	UnitNameDataSize      uint32
	CityDataSize          uint32
}

type Civ5UnitHeaderV11 struct {
	Unknown1        [2]byte
	NameIndex       uint16
	Experience      uint32
	Health          uint32
	UnitType        uint8
	Owner           uint8
	FacingDirection uint8
	Status          uint8
	Promotion       [PromotionSizeV11]byte
}

type Civ5UnitHeaderV12 struct {
	Unknown1        [2]byte
	NameIndex       uint16
	Experience      uint32
	Health          uint32
	UnitType        uint32
	Owner           uint8
	FacingDirection uint8
	Status          uint8
	Unknown2        byte
	Promotion       [PromotionSizeV12]byte
}

type Civ5UnitData struct {
	Name            string
	Experience      int
	Health          int
	UnitType        int
	Owner           int
	FacingDirection int
	Status          int
	PromotionInfo   []byte
}

type Civ5CityHeader struct {
	Name       [PlayerNameSize]byte
	Owner      uint8
	Flags      uint8
	Population uint16
	Health     uint32
}

type Civ5CityData struct {
	Name            string
	Owner           int
	OwnerAdjusted   int
	IsNameLocalized bool
	IsPuppetState   bool
	IsOccupied      bool
	Population      int
	Health          int
	BuildingInfo    []uint8
}

type Civ5MapTileHeader struct {
	CityId      uint16
	Unknown     [2]byte // seems to be unused
	Owner       uint8
	Improvement uint8
	RouteType   uint8
	RouteOwner  uint8
}

type Civ5PlayerHeader struct {
	Policies       [32]byte
	LeaderName     [LeaderNameSize]byte
	CivName        [CivNameSize]byte
	CivType        [CivTypeSize]byte
	TeamColor      [TeamColorSize]byte
	Era            [EraSize]byte
	Handicap       [HandicapSize]byte
	Culture        uint32
	Gold           uint32
	StartPositionX uint32
	StartPositionY uint32
	Team           uint8
	Playable       uint8
	Unknown1       [2]byte
}

type Civ5PlayerData struct {
	Index     int
	CivType   string
	TeamColor string
}

type Civ5MapTileImprovement struct {
	X           int
	Y           int
	CityId      int
	CityName    string
	Owner       int
	Improvement int
	RouteType   int
	RouteOwner  int
}

type CivColorOverride struct {
	CivKey     string
	OuterColor CivColorInfo
	InnerColor CivColorInfo
}

type CivColorInfo struct {
	Model         string
	ColorConstant string
	Red           float64
	Green         float64
	Blue          float64
}

type Civ5MapData struct {
	MapHeader           Civ5MapHeader
	TerrainList         []string
	FeatureTerrainList  []string
	ResourceList        []string
	TileImprovementList []string
	MapTiles            [][]*Civ5MapTilePhysical
	MapTileImprovements [][]*Civ5MapTileImprovement
	CityData            []*Civ5CityData
	Civ5PlayerData      []*Civ5PlayerData
	CityOwnerIndexMap   map[int]int
	CivColorOverrides   []CivColorOverride
}

// byteArrayToStringArray splits a null-separated byte buffer into a list of strings
func byteArrayToStringArray(byteArray []byte) []string {
	var builder strings.Builder
	arr := make([]string, 0)
	for i := 0; i < len(byteArray); i++ {
		if byteArray[i] == 0 {
			arr = append(arr, builder.String())
			builder.Reset()
		} else {
			builder.WriteByte(byteArray[i])
		}
	}
	return arr
}

// nullTerminatedString reads a fixed-size byte buffer as a string, stopping at the first null byte
func nullTerminatedString(b []byte) string {
	if idx := bytes.IndexByte(b, 0); idx >= 0 {
		return string(b[:idx])
	}
	return string(b)
}

// ParseUnitData parses the raw unit section of a map file into unit data
func ParseUnitData(unitData []byte, version int) ([]*Civ5UnitData, error) {
	if len(unitData) == 0 {
		return nil, nil
	}
	streamReader := io.NewSectionReader(bytes.NewReader(unitData), int64(0), int64(len(unitData)))

	numberUnits, err := readUint32(streamReader)
	if err != nil {
		return nil, err
	}
	fmt.Println("Number units: ", numberUnits)

	maximumPossibleUnits := maxUnitCountForVersion(len(unitData), version)
	fmt.Println("Maximum possible units: ", maximumPossibleUnits)

	if numberUnits > uint32(maximumPossibleUnits) {
		numberUnits = uint32(maximumPossibleUnits)
		fmt.Println("Something wrong with number of units, reduced to", numberUnits)
	}

	allUnits := make([]*Civ5UnitData, int(numberUnits))
	for i := 0; i < int(numberUnits); i++ {
		unit, err := readUnit(streamReader, version)
		if err != nil {
			return nil, err
		}
		allUnits[i] = unit
	}

	return allUnits, nil
}

// maxUnitCountForVersion returns how many units could possibly fit in a buffer of the given size
func maxUnitCountForVersion(dataLen, version int) int {
	if version == MapVersion12 {
		return dataLen / UnitDataSizeV12
	}
	return dataLen / UnitDataSizeV11
}

// readUnit reads a single unit record using the binary layout for the given version.
// Unrecognized versions yield a nil entry, matching the file's historical behavior.
func readUnit(reader *io.SectionReader, version int) (*Civ5UnitData, error) {
	switch version {
	case MapVersion12:
		header := Civ5UnitHeaderV12{}
		if err := readStruct(reader, &header); err != nil {
			return nil, err
		}
		return &Civ5UnitData{
			Experience:      int(header.Experience),
			Health:          int(header.Health),
			UnitType:        int(header.UnitType),
			Owner:           int(header.Owner),
			FacingDirection: int(header.FacingDirection),
			Status:          int(header.Status),
		}, nil
	case MapVersion11:
		header := Civ5UnitHeaderV11{}
		if err := readStruct(reader, &header); err != nil {
			return nil, err
		}
		return &Civ5UnitData{
			Experience:      int(header.Experience),
			Health:          int(header.Health),
			UnitType:        int(header.UnitType),
			Owner:           int(header.Owner),
			FacingDirection: int(header.FacingDirection),
			Status:          int(header.Status),
		}, nil
	default:
		return nil, nil
	}
}

// ParseCityData parses the raw city section of a map file into city data
func ParseCityData(cityData []byte, version int, maxCityId int) ([]*Civ5CityData, error) {
	if len(cityData) == 0 {
		return nil, nil
	}
	streamReader := io.NewSectionReader(bytes.NewReader(cityData), int64(0), int64(len(cityData)))

	// This number is not always accurate because it sometimes underestimates the number of cities
	numberCities, err := readUint32(streamReader)
	if err != nil {
		return nil, err
	}
	fmt.Println("Number cities: ", numberCities)

	if maxCityId+1 > int(numberCities) {
		numberCities = uint32(maxCityId) + 1
		fmt.Println("Number of cities should be", maxCityId+1)
	}

	buildingDataSize := buildingDataSizeForVersion(version)
	allCities := make([]*Civ5CityData, int(numberCities))
	for i := 0; i < int(numberCities); i++ {
		city, err := readCity(streamReader, buildingDataSize)
		if err != nil {
			return nil, err
		}
		allCities[i] = city
	}
	return allCities, nil
}

// buildingDataSizeForVersion returns the per-city building data size for the given format version
func buildingDataSizeForVersion(version int) int {
	if version == MapVersion12 {
		return BuildingDataSizeV12
	}
	return BuildingDataSizeV11
}

// adjustedCityOwner converts a city-state owner index back into its 0-based city-state index
func adjustedCityOwner(owner uint8) uint8 {
	if owner >= CityStateOffset {
		return owner - CityStateOffset
	}
	return owner
}

// readCity reads a single city record and its trailing building data
func readCity(reader *io.SectionReader, buildingDataSize int) (*Civ5CityData, error) {
	header := Civ5CityHeader{}
	if err := readStruct(reader, &header); err != nil {
		return nil, err
	}

	buildingInfo, err := readByteArray(reader, uint32(buildingDataSize))
	if err != nil {
		return nil, err
	}

	return &Civ5CityData{
		Name:            nullTerminatedString(header.Name[:]),
		Owner:           int(header.Owner),
		OwnerAdjusted:   int(adjustedCityOwner(header.Owner)),
		IsNameLocalized: header.Flags&1 != 0,
		IsPuppetState:   (header.Flags>>IsPuppetStateFlag)&1 != 0,
		IsOccupied:      (header.Flags>>IsOccupiedFlag)&1 != 0,
		Population:      int(header.Population), // 100% health is 100000
		Health:          int(header.Health),
		BuildingInfo:    buildingInfo,
	}, nil
}

// ParseCivData parses the raw civilization section of a map file into player data
func ParseCivData(inputData []byte) ([]*Civ5PlayerData, error) {
	allCivs, err := parseCivHeaders(inputData)
	if err != nil {
		return nil, err
	}
	reportCivData(allCivs)
	return civHeadersToPlayerData(allCivs), nil
}

// parseCivHeaders reads the fixed-size civilization headers from the raw byte buffer
func parseCivHeaders(inputData []byte) ([]Civ5PlayerHeader, error) {
	streamReader := io.NewSectionReader(bytes.NewReader(inputData), int64(0), int64(len(inputData)))
	allCivs := make([]Civ5PlayerHeader, len(inputData)/CivDataSize)
	if err := readStruct(streamReader, &allCivs); err != nil {
		return nil, err
	}
	return allCivs, nil
}

// civHeadersToPlayerData maps raw civilization headers to the public player data model
func civHeadersToPlayerData(allCivs []Civ5PlayerHeader) []*Civ5PlayerData {
	allPlayerData := make([]*Civ5PlayerData, len(allCivs))
	for i, civ := range allCivs {
		allPlayerData[i] = &Civ5PlayerData{
			Index:     i,
			CivType:   nullTerminatedString(civ.CivType[:]),
			TeamColor: nullTerminatedString(civ.TeamColor[:]),
		}
	}
	return allPlayerData
}

// reportCivData prints a human-readable summary of the parsed civilizations
func reportCivData(allCivs []Civ5PlayerHeader) {
	fmt.Printf("\n=== Civilizations (%d civs) ===\n", len(allCivs))
	for i, civ := range allCivs {
		fmt.Printf("  %d. %s\n", i+1, nullTerminatedString(civ.CivType[:]))
		fmt.Printf("      Team Color: %s\n", nullTerminatedString(civ.TeamColor[:]))
		fmt.Printf("      Team: %d\n", civ.Team)
		fmt.Printf("      Playable: %t\n", civ.Playable != 0)
		if i < len(allCivs)-1 {
			fmt.Println() // Add spacing between civs
		}
	}
}

// ParseMapTileProperties parses city/owner/route data for every tile on the map
func ParseMapTileProperties(inputData []byte, height int, width int) ([][]*Civ5MapTileImprovement, error) {
	streamReader := io.NewSectionReader(bytes.NewReader(inputData), int64(0), int64(len(inputData)))

	mapTiles := make([][]*Civ5MapTileImprovement, height)
	expectedTileSize := height * width * binary.Size(Civ5MapTileHeader{})
	if len(inputData) < expectedTileSize {
		return nil, fmt.Errorf("input data length is not sufficient for the expected tile size")
	}

	for i := 0; i < height; i++ {
		mapTiles[i] = make([]*Civ5MapTileImprovement, width)
		for j := 0; j < width; j++ {
			tileInfo := Civ5MapTileHeader{}
			if err := readStruct(streamReader, &tileInfo); err != nil {
				return nil, err
			}

			newCityId := int(tileInfo.CityId)
			if tileInfo.CityId == RawNoCityId {
				newCityId = InvalidCityId
			}

			mapTiles[i][j] = &Civ5MapTileImprovement{
				X:           j,
				Y:           i,
				CityId:      newCityId,
				Owner:       int(tileInfo.Owner),
				Improvement: int(tileInfo.Improvement),
				RouteType:   int(tileInfo.RouteType),
				RouteOwner:  int(tileInfo.RouteOwner),
			}
		}
	}

	return mapTiles, nil
}

// readUint32 reads a uint32 from the binary stream
func readUint32(reader *io.SectionReader) (uint32, error) {
	var value uint32
	if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
		return 0, err
	}
	return value, nil
}

// readByteArray reads a byte array of specified size from the binary stream
func readByteArray(reader *io.SectionReader, size uint32) ([]byte, error) {
	dataBytes := make([]byte, size)
	if err := binary.Read(reader, binary.LittleEndian, &dataBytes); err != nil {
		return nil, err
	}
	return dataBytes, nil
}

// readStruct reads a struct from the binary stream
func readStruct(reader *io.SectionReader, data interface{}) error {
	return binary.Read(reader, binary.LittleEndian, data)
}

// readStringList reads a null-separated string list of the given byte size from the binary stream
func readStringList(reader *io.SectionReader, size uint32) ([]string, error) {
	dataBytes, err := readByteArray(reader, size)
	if err != nil {
		return nil, err
	}
	return byteArrayToStringArray(dataBytes), nil
}

// reportStringList prints a human-readable summary of a named string list
func reportStringList(name string, list []string) {
	fmt.Printf("\n=== %s (%d items) ===\n", name, len(list))
	if len(list) == 0 {
		fmt.Println("(empty)")
		return
	}
	for i, item := range list {
		if item == "" {
			fmt.Printf("  %d. (empty)\n", i+1)
		} else {
			fmt.Printf("  %d. %s\n", i+1, item)
		}
	}
}

// readReportedStringList reads a named string list section and logs a summary of its contents
func readReportedStringList(reader *io.SectionReader, size uint32, name string) ([]string, error) {
	list, err := readStringList(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s data: %w", name, err)
	}
	reportStringList(name, list)
	return list, nil
}

// parsePhysicalMapTiles reads and parses the physical terrain data from map tiles
func parsePhysicalMapTiles(reader *io.SectionReader, header *Civ5MapHeader) ([][]*Civ5MapTilePhysical, error) {
	mapTiles := make([][]*Civ5MapTilePhysical, header.Height)
	for i := 0; i < int(header.Height); i++ {
		mapTiles[i] = make([]*Civ5MapTilePhysical, header.Width)
		for j := 0; j < int(header.Width); j++ {
			tile := Civ5MapTile{}
			if err := readStruct(reader, &tile); err != nil {
				return nil, fmt.Errorf("failed to read map tile at position (%d, %d): %w", i, j, err)
			}
			mapTiles[i][j] = &Civ5MapTilePhysical{
				X:                  j,
				Y:                  i,
				TerrainType:        int(tile.TerrainType),
				ResourceType:       int(tile.ResourceType),
				FeatureTerrainType: int(tile.FeatureTerrainType),
				RiverData:          int(tile.RiverData),
				Elevation:          int(tile.Elevation),
				Continent:          int(tile.Continent),
				FeatureWonderType:  int(tile.FeatureWonderType),
				ResourceAmount:     int(tile.ResourceAmount),
			}
		}
	}
	return mapTiles, nil
}

// findMaxCityId finds the highest city ID in the map tile improvement data
func findMaxCityId(mapTileImprovements [][]*Civ5MapTileImprovement, height, width int) int {
	maxCityId := 0
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			cityId := mapTileImprovements[i][j].CityId
			if cityId != InvalidCityId && cityId > maxCityId {
				maxCityId = cityId
			}
		}
	}
	return maxCityId
}

// isEndOfFile reports whether the reader has reached the end of the file
func isEndOfFile(reader *io.SectionReader) (bool, error) {
	currentPosition, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, err
	}
	return reader.Size() == currentPosition, nil
}

// createPhysicalMapData builds a map data structure containing only physical terrain data,
// used when the file ends before any game/city data is present
func createPhysicalMapData(header *Civ5MapHeader, terrainList, featureTerrainList, resourceList []string, mapTiles [][]*Civ5MapTilePhysical) *Civ5MapData {
	fmt.Println("Reached end of file. Skip reading game description header.")
	return buildMapData(header, terrainList, featureTerrainList, resourceList, mapTiles,
		[][]*Civ5MapTileImprovement{}, []*Civ5CityData{}, []*Civ5PlayerData{}, map[int]int{})
}

// resolvedCityName captures a city name resolved onto a specific tile, for reporting purposes
type resolvedCityName struct {
	Name string
	X, Y int
}

// resolveCityNames fills in the display name for every city tile, applying localization where needed
func resolveCityNames(mapTileImprovements [][]*Civ5MapTileImprovement, cityData []*Civ5CityData, height, width uint32) []resolvedCityName {
	resolved := make([]resolvedCityName, 0)
	for i := 0; i < int(height); i++ {
		for j := 0; j < int(width); j++ {
			tile := mapTileImprovements[i][j]
			if tile.CityId == InvalidCityId || tile.CityId >= len(cityData) {
				continue
			}
			tile.CityName = resolveCityName(cityData[tile.CityId])
			resolved = append(resolved, resolvedCityName{Name: tile.CityName, X: j, Y: i})
		}
	}
	return resolved
}

// resolveCityName returns the display name for a city, localizing it if necessary
func resolveCityName(city *Civ5CityData) string {
	if !city.IsNameLocalized {
		return city.Name
	}

	name := city.Name
	if idx := strings.Index(name, "CITY_NAME_"); idx != -1 {
		name = name[idx+len("CITY_NAME_"):]
	}
	if idx := strings.Index(name, "CITYSTATE_"); idx != -1 {
		name = name[idx+len("CITYSTATE_"):]
	}
	name = strings.Replace(name, "_", " ", -1)
	// If city name has multiple words, set each word's first letter to uppercase
	return cases.Title(language.Und).String(name)
}

// reportCityNames prints a human-readable summary of resolved city names
func reportCityNames(resolved []resolvedCityName) {
	fmt.Printf("\n=== Processing City Names ===\n")
	for i, city := range resolved {
		fmt.Printf("  %d. %s at (%d, %d)\n", i+1, city.Name, city.X, city.Y)
	}
	fmt.Printf("Processed %d cities\n", len(resolved))
}

// buildCityOwnerMaps builds the mapping from owner index to city names, and to a compact player index
func buildCityOwnerMaps(cityData []*Civ5CityData, playerCount, cityStateCount uint8) (map[int][]string, map[int]int) {
	cityOwnerMap := make(map[int][]string)
	cityOwnerIndexMap := make(map[int]int)

	// Initialize player maps
	for i := 0; i < int(playerCount); i++ {
		cityOwnerMap[i] = make([]string, 0)
		cityOwnerIndexMap[i] = i
	}

	// Initialize city state maps
	for i := 0; i < int(cityStateCount); i++ {
		cityOwnerMap[i+CityStateOffset] = make([]string, 0)
		cityOwnerIndexMap[i+CityStateOffset] = int(playerCount) + i
	}

	// Populate with actual city data
	for _, city := range cityData {
		if _, ok := cityOwnerMap[city.Owner]; !ok {
			cityOwnerMap[city.Owner] = make([]string, 0)
		}
		cityOwnerMap[city.Owner] = append(cityOwnerMap[city.Owner], city.Name)
	}

	return cityOwnerMap, cityOwnerIndexMap
}

// reportCityOwnerMaps prints a human-readable summary of the city owner maps
func reportCityOwnerMaps(cityOwnerMap map[int][]string, cityOwnerIndexMap map[int]int) {
	fmt.Printf("\n=== City Owner Map ===\n")
	if len(cityOwnerMap) == 0 {
		fmt.Println("(empty)")
	} else {
		for _, owner := range getSortedKeys(cityOwnerMap) {
			cities := cityOwnerMap[owner]
			if len(cities) > 0 {
				fmt.Printf("  Owner %d: %s\n", owner, strings.Join(cities, ", "))
			}
		}
	}

	fmt.Printf("\n=== City Owner Index Map ===\n")
	if len(cityOwnerIndexMap) == 0 {
		fmt.Println("(empty)")
	} else {
		for _, owner := range getSortedKeys(cityOwnerIndexMap) {
			fmt.Printf("  Owner %d -> Index %d\n", owner, cityOwnerIndexMap[owner])
		}
	}
}

// getSortedKeys returns the keys of a map sorted in ascending order
func getSortedKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		switch v := any(keys[i]).(type) {
		case int:
			return v < any(keys[j]).(int)
		case string:
			return v < any(keys[j]).(string)
		default:
			return false
		}
	})
	return keys
}

// buildMapData assembles the final map data structure from its parsed components
func buildMapData(header *Civ5MapHeader, terrainList, featureTerrainList, resourceList []string,
	mapTiles [][]*Civ5MapTilePhysical, improvements [][]*Civ5MapTileImprovement,
	cityData []*Civ5CityData, playerData []*Civ5PlayerData,
	cityOwnerIndexMap map[int]int) *Civ5MapData {

	return &Civ5MapData{
		MapHeader:           *header,
		TerrainList:         terrainList,
		FeatureTerrainList:  featureTerrainList,
		ResourceList:        resourceList,
		TileImprovementList: []string{}, // This could be populated if needed
		MapTiles:            mapTiles,
		MapTileImprovements: improvements,
		CityData:            cityData,
		Civ5PlayerData:      playerData,
		CityOwnerIndexMap:   cityOwnerIndexMap,
		CivColorOverrides:   []CivColorOverride{}, // No overrides by default
	}
}

// mapVersion extracts the binary format version from the scenario version byte
func mapVersion(scenarioVersion uint8) int {
	return int(scenarioVersion & VersionMask)
}

// mapScenario extracts the scenario flag from the scenario version byte
func mapScenario(scenarioVersion uint8) int {
	return int(scenarioVersion >> ScenarioBits)
}

// reportMapHeaderInfo prints a human-readable summary of the map header
func reportMapHeaderInfo(header *Civ5MapHeader, version, scenario int) {
	fmt.Println("Scenario: ", scenario)
	fmt.Println("Version: ", version)
	fmt.Println("Has world wrap: ", header.Settings[0]&1 != 0)
	fmt.Println("Has random resources: ", header.Settings[0]>>1&1 != 0)
	fmt.Println("Has random goodies: ", header.Settings[0]>>2&1 != 0)
}

// openMapFileReader opens a map file and returns a section reader spanning its entire contents
func openMapFileReader(filename string) (*os.File, int64, *io.SectionReader, error) {
	inputFile, err := os.Open(filename)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to load map: %w", err)
	}
	fi, err := inputFile.Stat()
	if err != nil {
		inputFile.Close()
		return nil, 0, nil, fmt.Errorf("failed to get file info for %q: %w", filename, err)
	}
	fileLength := fi.Size()
	return inputFile, fileLength, io.NewSectionReader(inputFile, int64(0), fileLength), nil
}

// readTerrainTypeLists reads the terrain, feature terrain, feature wonder, and resource type lists
func readTerrainTypeLists(reader *io.SectionReader, header *Civ5MapHeader) (terrainList, featureTerrainList, resourceList []string, err error) {
	terrainList, err = readReportedStringList(reader, header.TerrainDataSize, "Terrain data")
	if err != nil {
		return nil, nil, nil, err
	}

	featureTerrainList, err = readReportedStringList(reader, header.FeatureTerrainDataSize, "Feature terrain data")
	if err != nil {
		return nil, nil, nil, err
	}

	if _, err = readReportedStringList(reader, header.FeatureWonderDataSize, "Feature wonder data"); err != nil {
		return nil, nil, nil, err
	}

	resourceList, err = readReportedStringList(reader, header.ResourceDataSize, "Resource data")
	if err != nil {
		return nil, nil, nil, err
	}

	return terrainList, featureTerrainList, resourceList, nil
}

// skipMapMetadata reads (and logs) the mod data, map name, map description, and world size fields.
// None of this data is retained in the parsed map, matching the file's historical behavior.
func skipMapMetadata(reader *io.SectionReader, header *Civ5MapHeader, version int) error {
	modDataBytes, err := readByteArray(reader, header.ModDataSize)
	if err != nil {
		return err
	}
	fmt.Println("Mod data:", string(modDataBytes))

	mapNameBytes, err := readByteArray(reader, header.MapNameLength)
	if err != nil {
		return err
	}
	fmt.Println("Map name: ", string(mapNameBytes))

	mapDescriptionBytes, err := readByteArray(reader, header.MapDescriptionLength)
	if err != nil {
		return err
	}
	fmt.Println("Map description: ", string(mapDescriptionBytes))

	// Earlier versions don't have this field
	if version >= MapVersion11 {
		worldSizeStringLength, err := readUint32(reader)
		if err != nil {
			return err
		}
		worldSize, err := readByteArray(reader, worldSizeStringLength)
		if err != nil {
			return err
		}
		fmt.Println("World size: ", string(worldSize))
	}

	return nil
}

// reportGameDescriptionHeader prints a human-readable summary of the game description header
func reportGameDescriptionHeader(header *Civ5GameDescriptionHeader) {
	fmt.Println("\n=== Game Description ===")
	fmt.Printf("Max Turns: %d\n", header.MaxTurns)
	fmt.Printf("Start Year: %d\n", header.StartYear)
	fmt.Printf("Players: %d\n", header.PlayerCount)
	fmt.Printf("City States: %d\n", header.CityStateCount)
	fmt.Printf("Teams: %d\n", header.TeamCount)
	// Debug: Print raw struct for unknown field analysis
	fmt.Printf("Raw struct: %+v\n", *header)
}

// readGameDescriptionSection reads the game description header and the type/unit/city data
// sections that follow it, returning the raw unit and city data for later parsing
func readGameDescriptionSection(reader *io.SectionReader, version int) (Civ5GameDescriptionHeader, []byte, []byte, error) {
	fmt.Println("Reading game description header...")
	gameDescriptionHeader := Civ5GameDescriptionHeader{}
	if err := readStruct(reader, &gameDescriptionHeader); err != nil {
		return gameDescriptionHeader, nil, nil, err
	}
	reportGameDescriptionHeader(&gameDescriptionHeader)

	victoryDataSize := uint32(0)
	gameOptionDataSize := uint32(0)
	if version >= MapVersion11 {
		var err error
		victoryDataSize, err = readUint32(reader)
		if err != nil {
			return gameDescriptionHeader, nil, nil, err
		}
		gameOptionDataSize, err = readUint32(reader)
		if err != nil {
			return gameDescriptionHeader, nil, nil, err
		}
		fmt.Printf("Victory Data Size: %d bytes\n", victoryDataSize)
		fmt.Printf("Game Option Data Size: %d bytes\n", gameOptionDataSize)
	}

	namedListSizes := []struct {
		size uint32
		name string
	}{
		{gameDescriptionHeader.ImprovementDataSize, "Improvement data"},
		{gameDescriptionHeader.UnitTypeDataSize, "Unit type data"},
		{gameDescriptionHeader.TechTypeDataSize, "Tech type data"},
		{gameDescriptionHeader.PolicyTypeDataSize, "Policy type data"},
		{gameDescriptionHeader.BuildingTypeDataSize, "Building type data"},
		{gameDescriptionHeader.PromotionTypeDataSize, "Promotion type data"},
	}
	for _, list := range namedListSizes {
		if _, err := readReportedStringList(reader, list.size, list.name); err != nil {
			return gameDescriptionHeader, nil, nil, err
		}
	}

	fmt.Println("Unit data size: ", gameDescriptionHeader.UnitDataSize)
	unitDataBytes, err := readByteArray(reader, gameDescriptionHeader.UnitDataSize)
	if err != nil {
		return gameDescriptionHeader, nil, nil, err
	}

	fmt.Println("Unit name data size: ", gameDescriptionHeader.UnitNameDataSize)
	if _, err := readByteArray(reader, gameDescriptionHeader.UnitNameDataSize); err != nil {
		return gameDescriptionHeader, nil, nil, err
	}

	fmt.Println("City data size: ", gameDescriptionHeader.CityDataSize)
	cityDataBytes, err := readByteArray(reader, gameDescriptionHeader.CityDataSize)
	if err != nil {
		return gameDescriptionHeader, nil, nil, err
	}

	if version >= MapVersion11 {
		if _, err := readReportedStringList(reader, victoryDataSize, "Victory data"); err != nil {
			return gameDescriptionHeader, nil, nil, err
		}
		if _, err := readReportedStringList(reader, gameOptionDataSize, "Game option data"); err != nil {
			return gameDescriptionHeader, nil, nil, err
		}
	}

	return gameDescriptionHeader, unitDataBytes, cityDataBytes, nil
}

// readFileTail reads a fixed-size section of a file, ending precedingBytes before the end of the file
func readFileTail(inputFile *os.File, fileLength int64, size, precedingBytes int) ([]byte, error) {
	data := make([]byte, size)
	offset := fileLength - int64(precedingBytes) - int64(size)
	if _, err := inputFile.ReadAt(data, offset); err != nil {
		return nil, err
	}
	return data, nil
}

// readTailSections reads the map tile properties and player civilization data that are
// stored at fixed-size offsets from the end of the file
func readTailSections(inputFile *os.File, fileLength int64, mapHeader *Civ5MapHeader, gameDescriptionHeader *Civ5GameDescriptionHeader) ([][]*Civ5MapTileImprovement, []*Civ5PlayerData, error) {
	mapTilePropertiesSize := int(mapHeader.Height) * int(mapHeader.Width) * binary.Size(Civ5MapTileHeader{})
	mapTileProperties, err := readFileTail(inputFile, fileLength, mapTilePropertiesSize, 0)
	if err != nil {
		return nil, nil, err
	}

	mapTileImprovementData, err := ParseMapTileProperties(mapTileProperties, int(mapHeader.Height), int(mapHeader.Width))
	if err != nil {
		return nil, nil, err
	}

	playerCivDataSize := CivDataSize * (int(gameDescriptionHeader.PlayerCount) + int(gameDescriptionHeader.CityStateCount))
	playerCivData, err := readFileTail(inputFile, fileLength, playerCivDataSize, mapTilePropertiesSize)
	if err != nil {
		return nil, nil, err
	}

	allPlayerData, err := ParseCivData(playerCivData)
	if err != nil {
		return nil, nil, err
	}

	return mapTileImprovementData, allPlayerData, nil
}

func ReadCiv5MapFile(filename string) (*Civ5MapData, error) {
	inputFile, fileLength, streamReader, err := openMapFileReader(filename)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	mapHeader := Civ5MapHeader{}
	if err := readStruct(streamReader, &mapHeader); err != nil {
		return nil, err
	}

	version := mapVersion(mapHeader.ScenarioVersion)
	scenario := mapScenario(mapHeader.ScenarioVersion)
	reportMapHeaderInfo(&mapHeader, version, scenario)

	terrainList, featureTerrainList, resourceList, err := readTerrainTypeLists(streamReader, &mapHeader)
	if err != nil {
		return nil, err
	}

	if err := skipMapMetadata(streamReader, &mapHeader, version); err != nil {
		return nil, err
	}

	fmt.Println("Reading map tiles...")
	fmt.Println("Map height: ", mapHeader.Height)
	fmt.Println("Map width: ", mapHeader.Width)
	mapTiles, err := parsePhysicalMapTiles(streamReader, &mapHeader)
	if err != nil {
		return nil, err
	}

	atEndOfFile, err := isEndOfFile(streamReader)
	if err != nil {
		return nil, err
	}
	if atEndOfFile {
		return createPhysicalMapData(&mapHeader, terrainList, featureTerrainList, resourceList, mapTiles), nil
	}

	gameDescriptionHeader, unitDataBytes, cityDataBytes, err := readGameDescriptionSection(streamReader, version)
	if err != nil {
		return nil, err
	}

	mapTileImprovementData, allPlayerData, err := readTailSections(inputFile, fileLength, &mapHeader, &gameDescriptionHeader)
	if err != nil {
		return nil, err
	}

	maxCityId := findMaxCityId(mapTileImprovementData, int(mapHeader.Height), int(mapHeader.Width))
	fmt.Println("Max city id is", maxCityId)

	cityData, err := ParseCityData(cityDataBytes, version, maxCityId)
	if err != nil {
		return nil, err
	}

	if _, err := ParseUnitData(unitDataBytes, version); err != nil {
		return nil, err
	}

	if len(cityData) > 0 {
		resolvedCities := resolveCityNames(mapTileImprovementData, cityData, mapHeader.Height, mapHeader.Width)
		reportCityNames(resolvedCities)
	}

	cityOwnerMap, cityOwnerIndexMap := buildCityOwnerMaps(cityData, gameDescriptionHeader.PlayerCount, gameDescriptionHeader.CityStateCount)
	reportCityOwnerMaps(cityOwnerMap, cityOwnerIndexMap)

	return buildMapData(&mapHeader, terrainList, featureTerrainList, resourceList,
		mapTiles, mapTileImprovementData, cityData, allPlayerData, cityOwnerIndexMap), nil
}
