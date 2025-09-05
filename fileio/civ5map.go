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
	InvalidCityId = -1

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

	maximumPossibleUnits := 0
	if version == 12 {
		maximumPossibleUnits = len(unitData) / UnitDataSizeV12
	} else {
		maximumPossibleUnits = len(unitData) / UnitDataSizeV11
	}
	fmt.Println("Maximum possible units: ", maximumPossibleUnits)

	if numberUnits > uint32(maximumPossibleUnits) {
		numberUnits = uint32(maximumPossibleUnits)
		fmt.Println("Something wrong with number of units, reduced to", numberUnits)
	}

	allUnits := make([]*Civ5UnitData, int(numberUnits))
	for i := 0; i < int(numberUnits); i++ {
		switch version {
		case 12:
			unitData := Civ5UnitHeaderV12{}
			if err := binary.Read(streamReader, binary.LittleEndian, &unitData); err != nil {
				return nil, err
			}
			allUnits[i] = &Civ5UnitData{
				Name:            "",
				Experience:      int(unitData.Experience),
				Health:          int(unitData.Health),
				UnitType:        int(unitData.UnitType),
				Owner:           int(unitData.Owner),
				FacingDirection: int(unitData.FacingDirection),
				Status:          int(unitData.Status),
			}
		case 11:
			unitData := Civ5UnitHeaderV11{}
			if err := binary.Read(streamReader, binary.LittleEndian, &unitData); err != nil {
				return nil, err
			}

			allUnits[i] = &Civ5UnitData{
				Name:            "",
				Experience:      int(unitData.Experience),
				Health:          int(unitData.Health),
				UnitType:        int(unitData.UnitType),
				Owner:           int(unitData.Owner),
				FacingDirection: int(unitData.FacingDirection),
				Status:          int(unitData.Status),
			}
		}
	}

	return allUnits, nil
}

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

	allCities := make([]*Civ5CityData, int(numberCities))

	for i := 0; i < int(numberCities); i++ {
		cityData := Civ5CityHeader{}
		if err := binary.Read(streamReader, binary.LittleEndian, &cityData); err != nil {
			return nil, err
		}

		cityName := ""
		for j := 0; j < len(cityData.Name); j++ {
			if cityData.Name[j] == 0 {
				break
			}
			cityName += string(cityData.Name[j])
		}

		owner := cityData.Owner
		isCityState := owner >= CityStateOffset
		ownerAdjusted := owner
		if isCityState {
			ownerAdjusted = owner - CityStateOffset
		}

		flags := cityData.Flags
		isNameLocalized := flags&1 != 0
		isPuppetState := (flags>>IsPuppetStateFlag)&1 != 0
		isOccupied := (flags>>IsOccupiedFlag)&1 != 0

		// Version-specific building data size
		buildingDataSize := 0
		if version == 12 {
			buildingDataSize = BuildingDataSizeV12
		} else {
			buildingDataSize = BuildingDataSizeV11
		}

		buildingInfo := make([]byte, buildingDataSize)
		if err := binary.Read(streamReader, binary.LittleEndian, &buildingInfo); err != nil {
			return nil, err
		}

		allCities[i] = &Civ5CityData{
			Name:            cityName,
			Owner:           int(owner),
			OwnerAdjusted:   int(ownerAdjusted),
			IsNameLocalized: isNameLocalized,
			IsPuppetState:   isPuppetState,
			IsOccupied:      isOccupied,
			Population:      int(cityData.Population), // 100% health is 100000
			Health:          int(cityData.Health),
			BuildingInfo:    buildingInfo[:],
		}
	}
	return allCities, nil
}

func ParseCivData(inputData []byte) ([]*Civ5PlayerData, error) {
	streamReader := io.NewSectionReader(bytes.NewReader(inputData), int64(0), int64(len(inputData)))
	allCivs := make([]Civ5PlayerHeader, len(inputData)/CivDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &allCivs); err != nil {
		return nil, err
	}

	allPlayerData := make([]*Civ5PlayerData, len(allCivs))

	fmt.Printf("\n=== Civilizations (%d civs) ===\n", len(allCivs))
	for i := 0; i < len(allCivs); i++ {
		originalCivName := string(strings.Split(string(allCivs[i].CivType[:]), "\x00")[0])
		teamColor := string(strings.Split(string(allCivs[i].TeamColor[:]), "\x00")[0])

		fmt.Printf("  %d. %s\n", i+1, originalCivName)
		fmt.Printf("      Team Color: %s\n", teamColor)
		fmt.Printf("      Team: %d\n", allCivs[i].Team)
		fmt.Printf("      Playable: %t\n", allCivs[i].Playable != 0)
		if i < len(allCivs)-1 {
			fmt.Println() // Add spacing between civs
		}

		allPlayerData[i] = &Civ5PlayerData{
			Index:     i,
			CivType:   originalCivName,
			TeamColor: teamColor,
		}
	}
	return allPlayerData, nil
}

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
			if err := binary.Read(streamReader, binary.LittleEndian, &tileInfo); err != nil {
				return nil, err
			}

			newCityId := int(tileInfo.CityId)
			if tileInfo.CityId == 65535 {
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

func printList(list []string) {
	fmt.Printf("[%s]\n", strings.Join(list, ", "))
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

// Helper function to read string lists from binary data
func readStringList(reader *io.SectionReader, size uint32, listName string) ([]string, error) {
	dataBytes, err := readByteArray(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s data: %w", listName, err)
	}
	stringList := byteArrayToStringArray(dataBytes)

	// Pretty print the string list
	fmt.Printf("\n=== %s (%d items) ===\n", listName, len(stringList))
	if len(stringList) == 0 {
		fmt.Println("(empty)")
	} else {
		for i, item := range stringList {
			if item == "" {
				fmt.Printf("  %d. (empty)\n", i+1)
			} else {
				fmt.Printf("  %d. %s\n", i+1, item)
			}
		}
	}

	return stringList, nil
}

// parsePhysicalMapTiles reads and parses the physical terrain data from map tiles
func parsePhysicalMapTiles(reader *io.SectionReader, header *Civ5MapHeader) ([][]*Civ5MapTilePhysical, error) {
	mapTiles := make([][]*Civ5MapTilePhysical, header.Height)
	for i := 0; i < int(header.Height); i++ {
		mapTiles[i] = make([]*Civ5MapTilePhysical, header.Width)
		for j := 0; j < int(header.Width); j++ {
			tile := Civ5MapTile{}
			if err := binary.Read(reader, binary.LittleEndian, &tile); err != nil {
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
			if cityId != -1 && cityId > maxCityId {
				maxCityId = cityId
			}
		}
	}
	return maxCityId
}

// isEndOfFile checks if the reader has reached the end of the file
func isEndOfFile(reader *io.SectionReader) bool {
	currentPosition, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}
	return reader.Size() == currentPosition
}

// createPhysicalMapData creates a map data structure with only physical terrain data for early EOF cases
func createPhysicalMapData(header *Civ5MapHeader, terrainList, featureTerrainList, resourceList []string, mapTiles [][]*Civ5MapTilePhysical) *Civ5MapData {
	fmt.Println("Reached end of file. Skip reading game description header.")
	return &Civ5MapData{
		MapHeader:           *header,
		TerrainList:         terrainList,
		FeatureTerrainList:  featureTerrainList,
		ResourceList:        resourceList,
		TileImprovementList: []string{},
		MapTiles:            mapTiles,
		MapTileImprovements: [][]*Civ5MapTileImprovement{},
		CityData:            []*Civ5CityData{},
		Civ5PlayerData:      []*Civ5PlayerData{},
		CityOwnerIndexMap:   map[int]int{},
		CivColorOverrides:   []CivColorOverride{},
	}
}

// Helper function to process city names and apply localization
func processCityNames(mapTileImprovements [][]*Civ5MapTileImprovement, cityData []*Civ5CityData, height, width uint32) {
	// Early exit if no city data
	if len(cityData) == 0 {
		return
	}

	cityCount := 0
	fmt.Printf("\n=== Processing City Names ===\n")

	for i := 0; i < int(height); i++ {
		for j := 0; j < int(width); j++ {
			cityId := mapTileImprovements[i][j].CityId
			if cityId == InvalidCityId {
				continue
			}
			if cityId >= len(cityData) {
				continue
			}

			if cityData[cityId].IsNameLocalized {
				localizedName := cityData[cityId].Name
				if strings.Contains(localizedName, "CITY_NAME_") {
					localizedName = localizedName[strings.Index(localizedName, "CITY_NAME_")+len("CITY_NAME_"):]
				}
				if strings.Contains(localizedName, "CITYSTATE_") {
					localizedName = localizedName[strings.Index(localizedName, "CITYSTATE_")+len("CITYSTATE_"):]
				}
				localizedName = strings.Replace(localizedName, "_", " ", -1)
				// If city name has multiple words, set each word's first letter to uppercase
				localizedName = cases.Title(language.Und).String(localizedName)
				mapTileImprovements[i][j].CityName = localizedName
			} else {
				mapTileImprovements[i][j].CityName = cityData[cityId].Name
			}
			cityCount++
			fmt.Printf("  %d. %s at (%d, %d)\n", cityCount, mapTileImprovements[i][j].CityName, j, i)
		}
	}

	fmt.Printf("Processed %d cities\n", cityCount)
}

// Helper function to build city owner maps
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
	for i := 0; i < len(cityData); i++ {
		owner := cityData[i].Owner
		if _, ok := cityOwnerMap[owner]; !ok {
			cityOwnerMap[owner] = make([]string, 0)
		}
		cityOwnerMap[owner] = append(cityOwnerMap[owner], cityData[i].Name)
	}

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
			index := cityOwnerIndexMap[owner]
			fmt.Printf("  Owner %d -> Index %d\n", owner, index)
		}
	}
	return cityOwnerMap, cityOwnerIndexMap
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

// Helper function to build the final map data structure
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

func ReadCiv5MapFile(filename string) (*Civ5MapData, error) {
	inputFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load map: %w", err)
	}
	defer inputFile.Close()
	fi, err := inputFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %q: %w", filename, err)
	}
	fileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), fileLength)

	mapHeader := Civ5MapHeader{}
	if err := readStruct(streamReader, &mapHeader); err != nil {
		return nil, err
	}

	version := mapHeader.ScenarioVersion & 0xF
	scenario := mapHeader.ScenarioVersion >> 4
	fmt.Println("Scenario: ", scenario)
	fmt.Println("Version: ", version)

	hasWorldWrap, hasRandomResources, hasRandomGoodies := (mapHeader.Settings[0]&1) != 0, (mapHeader.Settings[0]>>1&1) != 0, (mapHeader.Settings[0]>>2&1) != 0
	fmt.Println("Has world wrap: ", hasWorldWrap)
	fmt.Println("Has random resources: ", hasRandomResources)
	fmt.Println("Has random goodies: ", hasRandomGoodies)

	terrainList, err := readStringList(streamReader, mapHeader.TerrainDataSize, "Terrain data")
	if err != nil {
		return nil, err
	}

	featureTerrainList, err := readStringList(streamReader, mapHeader.FeatureTerrainDataSize, "Feature terrain data")
	if err != nil {
		return nil, err
	}

	_, err = readStringList(streamReader, mapHeader.FeatureWonderDataSize, "Feature wonder data")
	if err != nil {
		return nil, err
	}

	resourceList, err := readStringList(streamReader, mapHeader.ResourceDataSize, "Resource data")
	if err != nil {
		return nil, err
	}

	modDataBytes, err := readByteArray(streamReader, mapHeader.ModDataSize)
	if err != nil {
		return nil, err
	}
	fmt.Println("Mod data:", string(modDataBytes))

	mapNameBytes, err := readByteArray(streamReader, mapHeader.MapNameLength)
	if err != nil {
		return nil, err
	}
	fmt.Println("Map name: ", string(mapNameBytes))

	mapDescriptionBytes, err := readByteArray(streamReader, mapHeader.MapDescriptionLength)
	if err != nil {
		return nil, err
	}
	fmt.Println("Map description: ", string(mapDescriptionBytes))

	// Earlier versions don't have this field
	if version >= 11 {
		worldSizeStringLength, err := readUint32(streamReader)
		if err != nil {
			return nil, err
		}

		worldSize, err := readByteArray(streamReader, worldSizeStringLength)
		if err != nil {
			return nil, err
		}
		fmt.Println("World size: ", string(worldSize))
	}

	fmt.Println("Reading map tiles...")
	fmt.Println("Map height: ", mapHeader.Height)
	fmt.Println("Map width: ", mapHeader.Width)

	mapTiles, err := parsePhysicalMapTiles(streamReader, &mapHeader)
	if err != nil {
		return nil, err
	}

	if isEndOfFile(streamReader) {
		return createPhysicalMapData(&mapHeader, terrainList, featureTerrainList, resourceList, mapTiles), nil
	}

	fmt.Println("Reading game description header...")
	gameDescriptionHeader := Civ5GameDescriptionHeader{}
	if err := readStruct(streamReader, &gameDescriptionHeader); err != nil {
		return nil, err
	}

	fmt.Println("\n=== Game Description ===")
	fmt.Printf("Max Turns: %d\n", gameDescriptionHeader.MaxTurns)
	fmt.Printf("Start Year: %d\n", gameDescriptionHeader.StartYear)
	fmt.Printf("Players: %d\n", gameDescriptionHeader.PlayerCount)
	fmt.Printf("City States: %d\n", gameDescriptionHeader.CityStateCount)
	fmt.Printf("Teams: %d\n", gameDescriptionHeader.TeamCount)

	// Debug: Print raw struct for unknown field analysis
	fmt.Printf("Raw struct: %+v\n", gameDescriptionHeader)

	// New fields for game description
	victoryDataSize := uint32(0)
	gameOptionDataSize := uint32(0)
	if version >= 11 {
		victoryDataSize, err = readUint32(streamReader)
		if err != nil {
			return nil, err
		}
		gameOptionDataSize, err = readUint32(streamReader)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Victory Data Size: %d bytes\n", victoryDataSize)
		fmt.Printf("Game Option Data Size: %d bytes\n", gameOptionDataSize)
	}

	_, err = readStringList(streamReader, gameDescriptionHeader.ImprovementDataSize, "Improvement data")
	if err != nil {
		return nil, err
	}

	_, err = readStringList(streamReader, gameDescriptionHeader.UnitTypeDataSize, "Unit type data")
	if err != nil {
		return nil, err
	}

	_, err = readStringList(streamReader, gameDescriptionHeader.TechTypeDataSize, "Tech type data")
	if err != nil {
		return nil, err
	}

	_, err = readStringList(streamReader, gameDescriptionHeader.PolicyTypeDataSize, "Policy type data")
	if err != nil {
		return nil, err
	}

	_, err = readStringList(streamReader, gameDescriptionHeader.BuildingTypeDataSize, "Building type data")
	if err != nil {
		return nil, err
	}

	_, err = readStringList(streamReader, gameDescriptionHeader.PromotionTypeDataSize, "Promotion type data")
	if err != nil {
		return nil, err
	}

	fmt.Println("Unit data size: ", gameDescriptionHeader.UnitDataSize)
	unitDataBytes, err := readByteArray(streamReader, gameDescriptionHeader.UnitDataSize)
	if err != nil {
		return nil, err
	}

	fmt.Println("Unit name data size: ", gameDescriptionHeader.UnitNameDataSize)
	_, err = readByteArray(streamReader, gameDescriptionHeader.UnitNameDataSize)
	if err != nil {
		return nil, err
	}

	fmt.Println("City data size: ", gameDescriptionHeader.CityDataSize)
	cityDataBytes, err := readByteArray(streamReader, gameDescriptionHeader.CityDataSize)
	if err != nil {
		return nil, err
	}

	if version >= 11 {
		_, err = readStringList(streamReader, victoryDataSize, "Victory data")
		if err != nil {
			return nil, err
		}

		_, err = readStringList(streamReader, gameOptionDataSize, "Game option data")
		if err != nil {
			return nil, err
		}
	}

	mapTileProperties := make([]byte, int(mapHeader.Height)*int(mapHeader.Width)*8)
	_, err = inputFile.ReadAt(mapTileProperties, fileLength-int64(len(mapTileProperties)))
	if err != nil {
		return nil, err
	}

	mapTileImprovementData, err := ParseMapTileProperties(mapTileProperties, int(mapHeader.Height), int(mapHeader.Width))
	if err != nil {
		return nil, err
	}

	playerCivData := make([]byte, CivDataSize*(int(gameDescriptionHeader.PlayerCount)+int(gameDescriptionHeader.CityStateCount)))
	_, err = inputFile.ReadAt(playerCivData, fileLength-int64(len(mapTileProperties))-int64(len(playerCivData)))
	if err != nil {
		return nil, err
	}

	allPlayerData, err := ParseCivData(playerCivData)
	if err != nil {
		return nil, err
	}

	maxCityId := findMaxCityId(mapTileImprovementData, int(mapHeader.Height), int(mapHeader.Width))
	fmt.Println("Max city id is", maxCityId)

	cityData, err := ParseCityData(cityDataBytes, int(version), maxCityId)
	if err != nil {
		return nil, err
	}

	_, err = ParseUnitData(unitDataBytes, int(version))
	if err != nil {
		return nil, err
	}

	// Fill in city names
	processCityNames(mapTileImprovementData, cityData, mapHeader.Height, mapHeader.Width)

	_, cityOwnerIndexMap := buildCityOwnerMaps(cityData, gameDescriptionHeader.PlayerCount, gameDescriptionHeader.CityStateCount)

	mapData := buildMapData(&mapHeader, terrainList, featureTerrainList, resourceList,
		mapTiles, mapTileImprovementData, cityData, allPlayerData, cityOwnerIndexMap)
	return mapData, nil
}
