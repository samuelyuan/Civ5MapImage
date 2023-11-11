package fileio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	Promotion       [32]byte
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
	Promotion       [64]byte
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
	Name       [64]byte
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
	BuildingInfo    []byte
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
	LeaderName     [64]byte
	CivName        [64]byte
	CivType        [64]byte
	TeamColor      [64]byte
	Era            [64]byte
	Handicap       [64]byte
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
	MapTiles            [][]*Civ5MapTile
	MapTileImprovements [][]*Civ5MapTileImprovement
	Civ5PlayerData      []*Civ5PlayerData
	CityOwnerIndexMap   map[int]int
	CivColorOverrides   []CivColorOverride
}

func byteArrayToStringArray(byteArray []byte) []string {
	str := ""
	arr := make([]string, 0)
	for i := 0; i < len(byteArray); i++ {
		if byteArray[i] == 0 {
			arr = append(arr, str)
			str = ""
		} else {
			str += string(byteArray[i])
		}
	}
	return arr
}

func ParseUnitData(unitData []byte, version int) ([]*Civ5UnitData, error) {
	if len(unitData) == 0 {
		return nil, nil
	}
	streamReader := io.NewSectionReader(bytes.NewReader(unitData), int64(0), int64(len(unitData)))

	numberUnits := uint32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &numberUnits); err != nil {
		return nil, err
	}
	fmt.Println("Number units: ", numberUnits)

	maximumPossibleUnits := 0
	if version == 12 {
		maximumPossibleUnits = len(unitData) / 84
	} else {
		maximumPossibleUnits = len(unitData) / 48
	}
	fmt.Println("Maximum possible units: ", maximumPossibleUnits)

	if numberUnits > uint32(maximumPossibleUnits) {
		numberUnits = uint32(maximumPossibleUnits)
		fmt.Println("Something wrong with number of units, reduced to", numberUnits)
	}

	allUnits := make([]*Civ5UnitData, int(numberUnits))
	for i := 0; i < int(numberUnits); i++ {
		if version == 12 {
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
		} else if version == 11 {
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
	numberCities := uint32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &numberCities); err != nil {
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
		isCityState := owner >= 32
		ownerAdjusted := owner
		if isCityState {
			ownerAdjusted = owner - 32
		}

		flags := cityData.Flags
		isNameLocalized := flags&1 != 0
		isPuppetState := (flags>>1)&1 != 0
		isOccupied := (flags>>2)&1 != 0

		// 32 for v11, 64 for v12
		buildingDataSize := 0
		if version == 12 {
			buildingDataSize = 64
		} else {
			buildingDataSize = 32
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
	civDataSize := 436
	allCivs := make([]Civ5PlayerHeader, len(inputData)/civDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &allCivs); err != nil {
		return nil, err
	}

	allPlayerData := make([]*Civ5PlayerData, len(allCivs))
	for i := 0; i < len(allCivs); i++ {
		originalCivName := string(strings.Split(string(allCivs[i].CivType[:]), "\x00")[0])
		teamColor := string(strings.Split(string(allCivs[i].TeamColor[:]), "\x00")[0])
		fmt.Println("Civ", i, ": Name:", originalCivName, ", Team color:", teamColor,
			", Team:", allCivs[i].Team, ", Playable:", allCivs[i].Playable)

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
	for i := 0; i < height; i++ {
		mapTiles[i] = make([]*Civ5MapTileImprovement, width)
		for j := 0; j < width; j++ {
			tileInfo := Civ5MapTileHeader{}
			if err := binary.Read(streamReader, binary.LittleEndian, &tileInfo); err != nil {
				return nil, err
			}

			newCityId := int(tileInfo.CityId)
			if tileInfo.CityId == 65535 {
				newCityId = -1
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
	outputStr := "["
	for i := 0; i < len(list); i++ {
		outputStr += "\"" + list[i] + "\""
		if i != len(list)-1 {
			outputStr += ", "
		}
	}
	outputStr += "]"
	fmt.Println(outputStr)
}

func ReadCiv5MapFile(filename string) (*Civ5MapData, error) {
	inputFile, err := os.Open(filename)
	defer inputFile.Close()
	if err != nil {
		log.Fatal("Failed to load map: ", err)
		return nil, err
	}
	fi, err := inputFile.Stat()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	fileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), fileLength)

	mapHeader := Civ5MapHeader{}
	if err := binary.Read(streamReader, binary.LittleEndian, &mapHeader); err != nil {
		return nil, err
	}

	version := mapHeader.ScenarioVersion & 0xF
	scenario := mapHeader.ScenarioVersion >> 4
	fmt.Println("Scenario: ", scenario)
	fmt.Println("Version: ", version)

	hasWorldWrap := (mapHeader.Settings[0] & 1) != 0
	hasRandomResources := (mapHeader.Settings[0] >> 1 & 1) != 0
	hasRandomGoodies := (mapHeader.Settings[0] >> 2 & 1) != 0
	fmt.Println("Has world wrap: ", hasWorldWrap)
	fmt.Println("Has random resources: ", hasRandomResources)
	fmt.Println("Has random goodies: ", hasRandomGoodies)

	terrainDataBytes := make([]byte, mapHeader.TerrainDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &terrainDataBytes); err != nil {
		return nil, err
	}
	terrainList := byteArrayToStringArray(terrainDataBytes)
	fmt.Println("Terrain data:")
	printList(terrainList)

	featureTerrainDataBytes := make([]byte, mapHeader.FeatureTerrainDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &featureTerrainDataBytes); err != nil {
		return nil, err
	}
	featureTerrainList := byteArrayToStringArray(featureTerrainDataBytes)
	fmt.Println("Feature terrain data:")
	printList(featureTerrainList)

	featureWonderDataBytes := make([]byte, mapHeader.FeatureWonderDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &featureWonderDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Feature wonder data:")
	printList(byteArrayToStringArray(featureWonderDataBytes))

	resourceDataBytes := make([]byte, mapHeader.ResourceDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &resourceDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Resource data:")
	resourceList := byteArrayToStringArray(resourceDataBytes)
	printList(resourceList)

	modDataBytes := make([]byte, mapHeader.ModDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &modDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Mod data:", string(modDataBytes))

	mapNameBytes := make([]byte, mapHeader.MapNameLength)
	if err := binary.Read(streamReader, binary.LittleEndian, &mapNameBytes); err != nil {
		return nil, err
	}
	fmt.Println("Map name: ", string(mapNameBytes))

	mapDescriptionBytes := make([]byte, mapHeader.MapDescriptionLength)
	if err := binary.Read(streamReader, binary.LittleEndian, &mapDescriptionBytes); err != nil {
		return nil, err
	}
	fmt.Println("Map description: ", string(mapDescriptionBytes))

	// Earlier versions don't have this field
	if version >= 11 {
		worldSizeStringLength := uint32(0)
		if err := binary.Read(streamReader, binary.LittleEndian, &worldSizeStringLength); err != nil {
			return nil, err
		}

		worldSize := make([]byte, worldSizeStringLength)
		if err := binary.Read(streamReader, binary.LittleEndian, &worldSize); err != nil {
			return nil, err
		}
		fmt.Println("World size: ", string(worldSize))
	}

	fmt.Println("Reading map tiles...")
	fmt.Println("Map height: ", mapHeader.Height)
	fmt.Println("Map width: ", mapHeader.Width)

	mapTiles := make([][]*Civ5MapTile, mapHeader.Height)
	for i := 0; i < int(mapHeader.Height); i++ {
		mapTiles[i] = make([]*Civ5MapTile, mapHeader.Width)
		for j := 0; j < int(mapHeader.Width); j++ {
			tile := Civ5MapTile{}
			if err := binary.Read(streamReader, binary.LittleEndian, &tile); err != nil {
				return nil, err
			}
			mapTiles[i][j] = &tile
		}
	}

	fmt.Println("Reading game description header...")
	gameDescriptionHeader := Civ5GameDescriptionHeader{}
	if err := binary.Read(streamReader, binary.LittleEndian, &gameDescriptionHeader); err != nil {
		return nil, err
	}
	fmt.Println("gameDescriptionHeader: ", gameDescriptionHeader)

	// New fields for game description
	victoryDataSize := uint32(0)
	gameOptionDataSize := uint32(0)
	if version >= 11 {
		if err := binary.Read(streamReader, binary.LittleEndian, &victoryDataSize); err != nil {
			return nil, err
		}
		if err := binary.Read(streamReader, binary.LittleEndian, &gameOptionDataSize); err != nil {
			return nil, err
		}
	}

	fmt.Println("Max turns: ", gameDescriptionHeader.MaxTurns)
	fmt.Println("Start year: ", gameDescriptionHeader.StartYear)
	fmt.Println("Player count: ", gameDescriptionHeader.PlayerCount)
	fmt.Println("City state count: ", gameDescriptionHeader.CityStateCount)
	fmt.Println("Team count: ", gameDescriptionHeader.TeamCount)

	improvementDataBytes := make([]byte, gameDescriptionHeader.ImprovementDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &improvementDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Improvement data:")
	tileImprovementList := byteArrayToStringArray(improvementDataBytes)
	printList(tileImprovementList)

	unitTypeDataBytes := make([]byte, gameDescriptionHeader.UnitTypeDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &unitTypeDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Unit type data:")
	printList(byteArrayToStringArray(unitTypeDataBytes))

	techTypeDataBytes := make([]byte, gameDescriptionHeader.TechTypeDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &techTypeDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Tech type data:")
	printList(byteArrayToStringArray(techTypeDataBytes))

	policyTypeDataBytes := make([]byte, gameDescriptionHeader.PolicyTypeDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &policyTypeDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Policy type data:")
	printList(byteArrayToStringArray(policyTypeDataBytes))

	buildingTypeDataBytes := make([]byte, gameDescriptionHeader.BuildingTypeDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &buildingTypeDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Building type data:")
	printList(byteArrayToStringArray(buildingTypeDataBytes))

	promotionTypeDataBytes := make([]byte, gameDescriptionHeader.PromotionTypeDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &promotionTypeDataBytes); err != nil {
		return nil, err
	}
	fmt.Println("Promotion type data:")
	printList(byteArrayToStringArray(promotionTypeDataBytes))

	fmt.Println("Unit data size: ", gameDescriptionHeader.UnitDataSize)
	unitDataBytes := make([]byte, gameDescriptionHeader.UnitDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &unitDataBytes); err != nil {
		return nil, err
	}

	fmt.Println("Unit name data size: ", gameDescriptionHeader.UnitNameDataSize)
	unitNameDataBytes := make([]byte, gameDescriptionHeader.UnitNameDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &unitNameDataBytes); err != nil {
		return nil, err
	}

	fmt.Println("City data size: ", gameDescriptionHeader.CityDataSize)
	cityDataBytes := make([]byte, gameDescriptionHeader.CityDataSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &cityDataBytes); err != nil {
		return nil, err
	}

	if version >= 11 {
		fmt.Println("Victory data size: ", victoryDataSize)
		victoryDataBytes := make([]byte, victoryDataSize)
		if err := binary.Read(streamReader, binary.LittleEndian, &victoryDataBytes); err != nil {
			return nil, err
		}
		fmt.Println("Victory data: ", byteArrayToStringArray(victoryDataBytes))

		fmt.Println("Game option data size: ", gameOptionDataSize)
		gameOptionDataBytes := make([]byte, gameOptionDataSize)
		if err := binary.Read(streamReader, binary.LittleEndian, &gameOptionDataBytes); err != nil {
			return nil, err
		}
		fmt.Println("Game option data: ", byteArrayToStringArray(gameOptionDataBytes))
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

	playerCivData := make([]byte, 436*(int(gameDescriptionHeader.PlayerCount)+int(gameDescriptionHeader.CityStateCount)))
	_, err = inputFile.ReadAt(playerCivData, fileLength-int64(len(mapTileProperties))-int64(len(playerCivData)))
	if err != nil {
		return nil, err
	}

	allPlayerData, err := ParseCivData(playerCivData)
	if err != nil {
		return nil, err
	}

	// Find max city id
	maxCityId := 0
	for i := 0; i < int(mapHeader.Height); i++ {
		for j := 0; j < int(mapHeader.Width); j++ {
			cityId := mapTileImprovementData[i][j].CityId
			if cityId != -1 && cityId > maxCityId {
				maxCityId = cityId
			}
		}
	}
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
	for i := 0; i < int(mapHeader.Height); i++ {
		for j := 0; j < int(mapHeader.Width); j++ {
			cityId := mapTileImprovementData[i][j].CityId
			if cityId != -1 && cityId < len(cityData) {
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
					mapTileImprovementData[i][j].CityName = localizedName
				} else {
					mapTileImprovementData[i][j].CityName = cityData[cityId].Name
				}

				fmt.Println(fmt.Sprintf("Set city %v at (%v, %v)", mapTileImprovementData[i][j].CityName, j, i))
			}
		}
	}

	cityOwnerMap := make(map[int][]string)
	cityOwnerIndexMap := make(map[int]int)
	for i := 0; i < int(gameDescriptionHeader.PlayerCount); i++ {
		cityOwnerMap[i] = make([]string, 0)
		cityOwnerIndexMap[i] = i
	}
	for i := 0; i < int(gameDescriptionHeader.CityStateCount); i++ {
		cityOwnerMap[i+32] = make([]string, 0)
		cityOwnerIndexMap[i+32] = int(gameDescriptionHeader.PlayerCount) + i
	}

	for i := 0; i < len(cityData); i++ {
		owner := cityData[i].Owner
		if _, ok := cityOwnerMap[owner]; !ok {
			cityOwnerMap[owner] = make([]string, 0)
		}
		cityOwnerMap[owner] = append(cityOwnerMap[owner], cityData[i].Name)
	}
	fmt.Println("City owner map:", cityOwnerMap)
	fmt.Println("City owner index map:", cityOwnerIndexMap)

	// No overrides by default
	civColorOverrides := []CivColorOverride{}

	mapData := &Civ5MapData{
		MapHeader:           mapHeader,
		TerrainList:         terrainList,
		FeatureTerrainList:  featureTerrainList,
		ResourceList:        resourceList,
		TileImprovementList: tileImprovementList,
		MapTiles:            mapTiles,
		MapTileImprovements: mapTileImprovementData,
		Civ5PlayerData:      allPlayerData,
		CityOwnerIndexMap:   cityOwnerIndexMap,
		CivColorOverrides:   civColorOverrides,
	}
	return mapData, err
}
