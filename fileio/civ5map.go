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

	// Unit Status bit positions
	FortifiedBit  = 0
	EmbarkedBit   = 1
	GarrisonedBit = 2

	// Special values
	InvalidCityId    = -1    // Used in our parsed data model
	NoCityIdSentinel = 65535 // Used in the raw file format (uint16 max)

	InvalidUnitId    = -1    // Used in our parsed data model
	NoUnitIdSentinel = 65535 // Used in the raw file format (uint16 max)

	// BarbarianOwner is the Owner value for a unit with no player/city-state (units only).
	BarbarianOwner = 96

	MaxCityStatesPerPlayer = 64

	// NoNameIndexSentinel is the NameIndex value meaning "no custom name".
	NoNameIndexSentinel = 65535

	// UnitNameRecordSize is the byte size of each entry in the unit name array.
	UnitNameRecordSize = 64

	// TeamNameRecordSize is the fixed size, in bytes, of each null-terminated team name entry.
	TeamNameRecordSize = 64

	// Data structure sizes
	CivDataSize = 436
)

// Diplomacy relationship types
const (
	DiplomacyInContact = iota
	DiplomacyAtWar
	DiplomacyPermanentWarOrPeace
	DiplomacyOpenBorders
	DiplomacyDefensivePact
	diplomacyRelationshipTypeCount
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
	GameSpeed             [64]byte // null-terminated string, e.g. GAMESPEED_STANDARD
	StartingTurn          uint32
	MaxTurns              uint32
	TargetScore           uint32
	StartYear             int32
	PlayerCount           uint8
	CityStateCount        uint8
	TeamCount             uint8
	Unknown               byte
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
	StackedUnitHandle uint16 // index of another unit sharing this unit's tile or 0xFFFF for no unit
	NameIndex         uint16
	Experience        uint32
	Health            uint32
	UnitType          uint8
	Owner             uint8
	FacingDirection   uint8
	Status            uint8
	Promotion         [PromotionSizeV11]byte
}

type Civ5UnitHeaderV12 struct {
	StackedUnitHandle uint16 // index of another unit sharing this unit's tile or 0xFFFF for no unit
	NameIndex         uint16
	Experience        uint32
	Health            uint32
	UnitType          uint32
	Owner             uint8
	FacingDirection   uint8
	Status            uint8
	Unknown           byte
	Promotion         [PromotionSizeV12]byte
}

type Civ5UnitData struct {
	Name                string // unit type name (e.g. UNIT_SETTLER), resolved against the unit type list
	CustomName          string // scenario-assigned custom name, resolved via NameIndex; empty if none
	Experience          int
	Health              int
	UnitType            int
	Owner               int
	FacingDirection     int
	FacingDirectionName string // resolved from FacingDirection; "" if out of range
	Status              int
	PromotionInfo       []byte
	Promotions          []string // names decoded from PromotionInfo against the promotion type list
	StackedUnitId       int      // index into Civ5MapData.UnitData of another unit on the same tile, or InvalidUnitId
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
	Buildings       []string // names decoded from BuildingInfo against the building type list
}

type Civ5MapTileHeader struct {
	CityId      uint16
	UnitId      uint16 // 0xFFFF (NoUnitIdSentinel) if no unit is on this tile; otherwise an index into UnitData
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
	Index              int
	LeaderName         string // override leader name; usually empty unless the scenario sets one
	CivName            string // override civ name; usually empty unless the scenario sets one
	CivType            string // default civ identifier
	TeamColor          string
	Era                string
	Handicap           string
	Culture            int
	Gold               int
	StartPositionX     int
	StartPositionY     int
	Team               int
	Playable           bool
	Policies           []string    // names decoded from the player's Policies bitset against the policy type list
	CityStateInfluence map[int]int // city-state index -> influence value (major civs only)
}

// Civ5TeamData holds one team's name.
type Civ5TeamData struct {
	Name string
}

// Civ5TeamRelationships holds team diplomacy data, indexed by team; each slice lists the other
// teams it has that relationship with (symmetric).
type Civ5TeamRelationships struct {
	InContactWith           [][]int
	AtWarWith               [][]int
	PermanentWarOrPeaceWith [][]int
	OpenBordersWith         [][]int
	DefensivePactWith       [][]int
}

const (
	RouteRoad     = 0
	RouteRailroad = 1
	RouteNone     = 255
)

type Civ5MapTileImprovement struct {
	X           int
	Y           int
	CityId      int
	CityName    string
	UnitId      int // index into Civ5MapData.UnitData, or InvalidUnitId if no unit is on this tile
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
	BuildingTypeList    []string
	PromotionTypeList   []string
	PolicyTypeList      []string
	UnitTypeList        []string
	UnitNameList        []string
	MapTiles            [][]*Civ5MapTilePhysical
	MapTileImprovements [][]*Civ5MapTileImprovement
	CityData            []*Civ5CityData
	UnitData            []*Civ5UnitData
	Civ5PlayerData      []*Civ5PlayerData
	TeamData            []*Civ5TeamData // names only
	TeamRelationships   Civ5TeamRelationships
	TeamVisibility      [][][2]int // per team's visible [x,y] tiles
	CityOwnerIndexMap   map[int]int
	CivColorOverrides   []CivColorOverride
}

// Civ5GameTypeLists holds type lists used to decode bitset fields and resolve unit type/custom names.
type Civ5GameTypeLists struct {
	BuildingTypeList  []string
	PromotionTypeList []string
	PolicyTypeList    []string
	UnitTypeList      []string // index by Civ5UnitData.UnitType to get the unit type name
	UnitNameList      []string // index by a unit's NameIndex to get its custom name
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

// parseUnitNameArray parses the unit name array (leading count skipped, unreliable) into
// fixed-size null-terminated records, indexed by NameIndex.
func parseUnitNameArray(data []byte) []string {
	if len(data) < 4 {
		return nil
	}
	nameFreeListHead := binary.LittleEndian.Uint32(data[:4])
	fmt.Println("Unit name allocator free-list head: ", int32(nameFreeListHead))

	records := data[4:]
	count := len(records) / UnitNameRecordSize
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = nullTerminatedString(records[i*UnitNameRecordSize : (i+1)*UnitNameRecordSize])
	}
	return names
}

// ParseUnitData parses the raw unit section, using typeLists to decode promotions and resolve names.
func ParseUnitData(unitData []byte, version int, typeLists Civ5GameTypeLists) ([]*Civ5UnitData, error) {
	if len(unitData) < 4 {
		return nil, nil
	}
	streamReader := io.NewSectionReader(bytes.NewReader(unitData), int64(0), int64(len(unitData)))

	unitFreeListHead, err := readUint32(streamReader)
	if err != nil {
		return nil, err
	}
	fmt.Println("Unit allocator free-list head: ", int32(unitFreeListHead))

	maximumPossibleUnits := maxUnitCountForVersion(len(unitData)-4, version)
	fmt.Println("Maximum possible units (buffer capacity): ", maximumPossibleUnits)

	allUnits := make([]*Civ5UnitData, maximumPossibleUnits)
	for i := 0; i < maximumPossibleUnits; i++ {
		unit, err := readUnit(streamReader, version, typeLists)
		if err != nil {
			return nil, err
		}
		allUnits[i] = unit
	}

	return allUnits, nil
}

// maxUnitCountForVersion returns how many unit records fit in dataLen bytes.
func maxUnitCountForVersion(dataLen, version int) int {
	if version == MapVersion12 {
		return dataLen / UnitDataSizeV12
	}
	return dataLen / UnitDataSizeV11
}

// readUnit reads one unit record for the given version, using typeLists to resolve names.
func readUnit(reader *io.SectionReader, version int, typeLists Civ5GameTypeLists) (*Civ5UnitData, error) {
	var stackedUnitHandle, nameIndex uint16
	var experience, health uint32
	var unitType int
	var owner, facingDirection, status uint8
	var promotionInfo []byte

	switch version {
	case MapVersion12:
		header := Civ5UnitHeaderV12{}
		if err := readStruct(reader, &header); err != nil {
			return nil, err
		}
		stackedUnitHandle = header.StackedUnitHandle
		nameIndex = header.NameIndex
		experience, health = header.Experience, header.Health
		unitType = int(header.UnitType)
		owner, facingDirection, status = header.Owner, header.FacingDirection, header.Status
		promotionInfo = header.Promotion[:]
	case MapVersion11:
		header := Civ5UnitHeaderV11{}
		if err := readStruct(reader, &header); err != nil {
			return nil, err
		}
		stackedUnitHandle = header.StackedUnitHandle
		nameIndex = header.NameIndex
		experience, health = header.Experience, header.Health
		unitType = int(header.UnitType)
		owner, facingDirection, status = header.Owner, header.FacingDirection, header.Status
		promotionInfo = header.Promotion[:]
	default:
		return nil, nil
	}

	stackedUnitId := InvalidUnitId
	if stackedUnitHandle != NoUnitIdSentinel {
		stackedUnitId = int(stackedUnitHandle)
	}

	return &Civ5UnitData{
		Name:                typeName(typeLists.UnitTypeList, unitType),
		CustomName:          customUnitName(typeLists.UnitNameList, nameIndex),
		Experience:          int(experience),
		Health:              int(health),
		UnitType:            unitType,
		Owner:               int(owner),
		FacingDirection:     int(facingDirection),
		FacingDirectionName: facingDirectionName(int(facingDirection)),
		Status:              int(status),
		PromotionInfo:       promotionInfo,
		Promotions:          decodeBitset(promotionInfo, typeLists.PromotionTypeList),
		StackedUnitId:       stackedUnitId,
	}, nil
}

// ParseCityData parses the raw city section, using buildingTypeList to decode each city's buildings.
func ParseCityData(cityData []byte, version int, maxCityId int, buildingTypeList []string) ([]*Civ5CityData, error) {
	if len(cityData) == 0 {
		return nil, nil
	}
	streamReader := io.NewSectionReader(bytes.NewReader(cityData), int64(0), int64(len(cityData)))

	// The leading 4 bytes are the allocator's free-list head index, not a record count
	cityFreeListHead, err := readUint32(streamReader)
	if err != nil {
		return nil, err
	}
	fmt.Println("City allocator free-list head: ", int32(cityFreeListHead))

	numberCities := uint32(maxCityId) + 1

	buildingDataSize := buildingDataSizeForVersion(version)
	allCities := make([]*Civ5CityData, int(numberCities))
	for i := 0; i < int(numberCities); i++ {
		city, err := readCity(streamReader, buildingDataSize, buildingTypeList)
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

// typeName resolves an index into a type list, returning "" if out of range.
func typeName(typeList []string, index int) string {
	if index < 0 || index >= len(typeList) {
		return ""
	}
	return typeList[index]
}

// facingDirectionName resolves a unit's raw FacingDirection byte, returning "" if out of range.
func facingDirectionName(raw int) string {
	return typeName(facingDirectionNames, raw)
}

// customUnitName resolves NameIndex into a name, or "" if unset or out of range.
func customUnitName(unitNameList []string, nameIndex uint16) string {
	if nameIndex == NoNameIndexSentinel || int(nameIndex) >= len(unitNameList) {
		return ""
	}
	return unitNameList[nameIndex]
}

// decodeBitset returns the names of every set bit in data, where bit N is typeList[N].
func decodeBitset(data []byte, typeList []string) []string {
	var names []string
	for i, name := range typeList {
		byteIndex := i / 8
		if byteIndex >= len(data) {
			break
		}
		if (data[byteIndex]>>uint(i%8))&1 != 0 {
			names = append(names, name)
		}
	}
	return names
}

// triangularPairCount returns the number of unordered pairs among n elements (n*(n-1)/2).
func triangularPairCount(n int) int {
	return n * (n - 1) / 2
}

// relationshipBitsetSize returns the per-type bitset size in bytes for teamCount teams.
func relationshipBitsetSize(teamCount int) int {
	return (triangularPairCount(teamCount) + 7) / 8
}

// relationshipBitIndex returns the bit index for team pair (large, small), large > small.
func relationshipBitIndex(large, small int) int {
	return triangularPairCount(large) + small
}

// decodeRelationshipPairs decodes one relationship type's bitset into every team pair (i, j), i > j.
func decodeRelationshipPairs(bitsetData []byte, teamCount int) [][2]int {
	var pairs [][2]int
	for i := 1; i < teamCount; i++ {
		for j := 0; j < i; j++ {
			bitIndex := relationshipBitIndex(i, j)
			byteIndex := bitIndex / 8
			if byteIndex >= len(bitsetData) {
				continue
			}
			if (bitsetData[byteIndex]>>uint(bitIndex%8))&1 != 0 {
				pairs = append(pairs, [2]int{i, j})
			}
		}
	}
	return pairs
}

// ParseTeamNames decodes the team name array into one Civ5TeamData per team.
func ParseTeamNames(teamNameData []byte, teamCount int) []*Civ5TeamData {
	teams := make([]*Civ5TeamData, teamCount)
	for i := 0; i < teamCount; i++ {
		start := i * TeamNameRecordSize
		end := start + TeamNameRecordSize
		name := ""
		if end <= len(teamNameData) {
			name = nullTerminatedString(teamNameData[start:end])
		}
		teams[i] = &Civ5TeamData{Name: name}
	}
	return teams
}

// relationshipTypeSlice returns relType's byte slice of relationshipData, clamped to what's available.
func relationshipTypeSlice(relationshipData []byte, relType, typeSize int) (slice []byte, ok bool) {
	start := relType * typeSize
	if start > len(relationshipData) {
		return nil, false
	}
	end := start + typeSize
	if end > len(relationshipData) {
		end = len(relationshipData)
	}
	return relationshipData[start:end], true
}

// ParseTeamRelationships decodes relationshipData into Civ5TeamRelationships.
func ParseTeamRelationships(relationshipData []byte, teamCount int) Civ5TeamRelationships {
	rel := Civ5TeamRelationships{
		InContactWith:           make([][]int, teamCount),
		AtWarWith:               make([][]int, teamCount),
		PermanentWarOrPeaceWith: make([][]int, teamCount),
		OpenBordersWith:         make([][]int, teamCount),
		DefensivePactWith:       make([][]int, teamCount),
	}

	assign := func(pairs [][2]int, target [][]int) {
		for _, pair := range pairs {
			i, j := pair[0], pair[1]
			target[i] = append(target[i], j)
			target[j] = append(target[j], i)
		}
	}

	relationshipTargets := [][][]int{
		DiplomacyInContact:           rel.InContactWith,
		DiplomacyAtWar:               rel.AtWarWith,
		DiplomacyPermanentWarOrPeace: rel.PermanentWarOrPeaceWith,
		DiplomacyOpenBorders:         rel.OpenBordersWith,
		DiplomacyDefensivePact:       rel.DefensivePactWith,
	}

	typeSize := relationshipBitsetSize(teamCount)
	for relType := 0; relType < diplomacyRelationshipTypeCount; relType++ {
		slice, ok := relationshipTypeSlice(relationshipData, relType, typeSize)
		if !ok {
			break
		}
		pairs := decodeRelationshipPairs(slice, teamCount)
		assign(pairs, relationshipTargets[relType])
	}

	return rel
}

// ParseTeamVisibility decodes visibilityData into each team's visible tiles, indexed by team.
func ParseTeamVisibility(visibilityData []byte, mapSize MapSize, teamCount int) [][][2]int {
	tileCount := mapSize.Width * mapSize.Height
	matrix := make([][][2]int, teamCount)
	for t := 0; t < teamCount; t++ {
		var visible [][2]int
		for y := 0; y < mapSize.Height; y++ {
			for x := 0; x < mapSize.Width; x++ {
				bitIndex := t*tileCount + y*mapSize.Width + x
				byteIndex := bitIndex / 8
				if byteIndex >= len(visibilityData) {
					continue
				}
				if (visibilityData[byteIndex]>>uint(bitIndex%8))&1 != 0 {
					visible = append(visible, [2]int{x, y})
				}
			}
		}
		matrix[t] = visible
	}
	return matrix
}

// ParseCityStateInfluence decodes each player's influence with every city-state, indexed by player.
func ParseCityStateInfluence(influenceData []byte, playerCount, cityStateCount int) []map[int]int {
	result := make([]map[int]int, playerCount)
	for p := 0; p < playerCount; p++ {
		influence := make(map[int]int, cityStateCount)
		for cs := 0; cs < cityStateCount; cs++ {
			offset := (p*MaxCityStatesPerPlayer + cs) * 4
			if offset+4 > len(influenceData) {
				break
			}
			influence[cs] = int(int32(binary.LittleEndian.Uint32(influenceData[offset : offset+4])))
		}
		result[p] = influence
	}
	return result
}

// readCity reads a single city record and its trailing building data, decoded via buildingTypeList.
func readCity(reader *io.SectionReader, buildingDataSize int, buildingTypeList []string) (*Civ5CityData, error) {
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
		Buildings:       decodeBitset(buildingInfo, buildingTypeList),
	}, nil
}

// ParseCivData parses the raw civilization section, using policyTypeList to decode each player's policies.
func ParseCivData(inputData []byte, policyTypeList []string) ([]*Civ5PlayerData, error) {
	allCivs, err := parseCivHeaders(inputData)
	if err != nil {
		return nil, err
	}
	reportCivData(allCivs)
	return civHeadersToPlayerData(allCivs, policyTypeList), nil
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

// civHeadersToPlayerData maps raw civ headers to player data, decoding Policies via policyTypeList.
func civHeadersToPlayerData(allCivs []Civ5PlayerHeader, policyTypeList []string) []*Civ5PlayerData {
	allPlayerData := make([]*Civ5PlayerData, len(allCivs))
	for i, civ := range allCivs {
		allPlayerData[i] = &Civ5PlayerData{
			Index:          i,
			LeaderName:     nullTerminatedString(civ.LeaderName[:]),
			CivName:        nullTerminatedString(civ.CivName[:]),
			CivType:        nullTerminatedString(civ.CivType[:]),
			TeamColor:      nullTerminatedString(civ.TeamColor[:]),
			Era:            nullTerminatedString(civ.Era[:]),
			Handicap:       nullTerminatedString(civ.Handicap[:]),
			Culture:        int(civ.Culture),
			Gold:           int(civ.Gold),
			StartPositionX: int(civ.StartPositionX),
			StartPositionY: int(civ.StartPositionY),
			Team:           int(civ.Team),
			Playable:       civ.Playable != 0,
			Policies:       decodeBitset(civ.Policies[:], policyTypeList),
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
			if tileInfo.CityId == NoCityIdSentinel {
				newCityId = InvalidCityId
			}

			newUnitId := int(tileInfo.UnitId)
			if tileInfo.UnitId == NoUnitIdSentinel {
				newUnitId = InvalidUnitId
			}

			mapTiles[i][j] = &Civ5MapTileImprovement{
				X:           j,
				Y:           i,
				CityId:      newCityId,
				UnitId:      newUnitId,
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

// createPhysicalMapData builds map data with only physical terrain, for files that end early.
func createPhysicalMapData(header *Civ5MapHeader, terrainList, featureTerrainList, resourceList []string, mapTiles [][]*Civ5MapTilePhysical) *Civ5MapData {
	fmt.Println("Reached end of file. Skip reading game description header.")
	return buildMapData(header, terrainList, featureTerrainList, resourceList, mapTiles,
		[][]*Civ5MapTileImprovement{}, []*Civ5CityData{}, nil, []*Civ5PlayerData{}, []*Civ5TeamData{},
		Civ5TeamRelationships{}, nil, map[int]int{}, Civ5GameTypeLists{})
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
		for _, owner := range GetSortedKeys(cityOwnerMap) {
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
		for _, owner := range GetSortedKeys(cityOwnerIndexMap) {
			fmt.Printf("  Owner %d -> Index %d\n", owner, cityOwnerIndexMap[owner])
		}
	}
}

// reportTeamRelationships prints each team's diplomatic relationships, one line per pair.
func reportTeamRelationships(teamNames []*Civ5TeamData, rel Civ5TeamRelationships) {
	fmt.Printf("\n=== Team Relationships ===\n")
	if len(teamNames) == 0 {
		fmt.Println("(empty)")
		return
	}
	printOnce := func(label string, otherTeams []int, selfIndex int) {
		var names []string
		for _, other := range otherTeams {
			if other <= selfIndex {
				continue // already printed from the other team's perspective
			}
			names = append(names, teamNames[other].Name)
		}
		if len(names) > 0 {
			fmt.Printf("  %s %s: %s\n", teamNames[selfIndex].Name, label, strings.Join(names, ", "))
		}
	}
	for i := range teamNames {
		printOnce("in contact with", rel.InContactWith[i], i)
		printOnce("at war with", rel.AtWarWith[i], i)
		printOnce("permanent war/peace with", rel.PermanentWarOrPeaceWith[i], i)
		printOnce("open borders with", rel.OpenBordersWith[i], i)
		printOnce("defensive pact with", rel.DefensivePactWith[i], i)
	}
}

// GetSortedKeys returns the keys of a map sorted in ascending order
func GetSortedKeys[K comparable, V any](m map[K]V) []K {
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
	cityData []*Civ5CityData, unitData []*Civ5UnitData, playerData []*Civ5PlayerData, teamData []*Civ5TeamData,
	teamRelationships Civ5TeamRelationships, teamVisibility [][][2]int,
	cityOwnerIndexMap map[int]int, typeLists Civ5GameTypeLists) *Civ5MapData {

	return &Civ5MapData{
		MapHeader:           *header,
		TerrainList:         terrainList,
		FeatureTerrainList:  featureTerrainList,
		ResourceList:        resourceList,
		TileImprovementList: []string{}, // This could be populated if needed
		BuildingTypeList:    typeLists.BuildingTypeList,
		PromotionTypeList:   typeLists.PromotionTypeList,
		PolicyTypeList:      typeLists.PolicyTypeList,
		UnitTypeList:        typeLists.UnitTypeList,
		UnitNameList:        typeLists.UnitNameList,
		MapTiles:            mapTiles,
		MapTileImprovements: improvements,
		CityData:            cityData,
		UnitData:            unitData,
		Civ5PlayerData:      playerData,
		TeamData:            teamData,
		TeamRelationships:   teamRelationships,
		TeamVisibility:      teamVisibility,
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

// openMapFileReader reads a map file into memory, so parsing its many small structs costs no system calls,
// and returns the contents for random access along with a section reader spanning all of them.
func openMapFileReader(filename string) (io.ReaderAt, int64, *io.SectionReader, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to load map: %w", err)
	}
	contents := bytes.NewReader(data)
	return contents, int64(len(data)), io.NewSectionReader(contents, 0, int64(len(data))), nil
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

// skipMapMetadata reads and logs mod/name/description/world-size fields; none are retained.
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
	fmt.Printf("Game Speed: %s\n", nullTerminatedString(header.GameSpeed[:]))
	fmt.Printf("Starting Turn: %d\n", header.StartingTurn)
	fmt.Printf("Max Turns: %d\n", header.MaxTurns)
	fmt.Printf("Target Score: %d\n", header.TargetScore)
	fmt.Printf("Start Year: %d\n", header.StartYear)
	fmt.Printf("Players: %d\n", header.PlayerCount)
	fmt.Printf("City States: %d\n", header.CityStateCount)
	fmt.Printf("Teams: %d\n", header.TeamCount)
}

// gameDescriptionSectionData bundles byte blobs from the game description section for later parsing.
type gameDescriptionSectionData struct {
	UnitDataBytes         []byte
	CityDataBytes         []byte
	RelationshipDataBytes []byte
	InfluenceDataBytes    []byte
	VisibilityDataBytes   []byte
}

// readGameDescriptionSection reads the game description header through the visibility bitset.
func readGameDescriptionSection(reader *io.SectionReader, version int, mapSize MapSize) (Civ5GameDescriptionHeader, gameDescriptionSectionData, Civ5GameTypeLists, error) {
	fmt.Println("Reading game description header...")
	gameDescriptionHeader := Civ5GameDescriptionHeader{}
	typeLists := Civ5GameTypeLists{}
	if err := readStruct(reader, &gameDescriptionHeader); err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}
	reportGameDescriptionHeader(&gameDescriptionHeader)

	victoryDataSize := uint32(0)
	gameOptionDataSize := uint32(0)
	if version >= MapVersion11 {
		var err error
		victoryDataSize, err = readUint32(reader)
		if err != nil {
			return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
		}
		gameOptionDataSize, err = readUint32(reader)
		if err != nil {
			return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
		}
		fmt.Printf("Victory Data Size: %d bytes\n", victoryDataSize)
		fmt.Printf("Game Option Data Size: %d bytes\n", gameOptionDataSize)
	}

	// Only Policy/Building/Promotion/UnitType lists are retained; the rest are read and logged only.
	if _, err := readReportedStringList(reader, gameDescriptionHeader.ImprovementDataSize, "Improvement data"); err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}
	unitTypeList, err := readReportedStringList(reader, gameDescriptionHeader.UnitTypeDataSize, "Unit type data")
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}
	if _, err := readReportedStringList(reader, gameDescriptionHeader.TechTypeDataSize, "Tech type data"); err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}
	policyTypeList, err := readReportedStringList(reader, gameDescriptionHeader.PolicyTypeDataSize, "Policy type data")
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}
	buildingTypeList, err := readReportedStringList(reader, gameDescriptionHeader.BuildingTypeDataSize, "Building type data")
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}
	promotionTypeList, err := readReportedStringList(reader, gameDescriptionHeader.PromotionTypeDataSize, "Promotion type data")
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}

	fmt.Println("Unit data size: ", gameDescriptionHeader.UnitDataSize)
	unitDataBytes, err := readByteArray(reader, gameDescriptionHeader.UnitDataSize)
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}

	fmt.Println("Unit name data size: ", gameDescriptionHeader.UnitNameDataSize)
	unitNameBytes, err := readByteArray(reader, gameDescriptionHeader.UnitNameDataSize)
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}
	unitNameList := parseUnitNameArray(unitNameBytes)

	typeLists = Civ5GameTypeLists{
		BuildingTypeList:  buildingTypeList,
		PromotionTypeList: promotionTypeList,
		PolicyTypeList:    policyTypeList,
		UnitTypeList:      unitTypeList,
		UnitNameList:      unitNameList,
	}

	fmt.Println("City data size: ", gameDescriptionHeader.CityDataSize)
	cityDataBytes, err := readByteArray(reader, gameDescriptionHeader.CityDataSize)
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}

	if version >= MapVersion11 {
		if _, err := readReportedStringList(reader, victoryDataSize, "Victory data"); err != nil {
			return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
		}
		if _, err := readReportedStringList(reader, gameOptionDataSize, "Game option data"); err != nil {
			return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
		}
	}

	// Team relationship bitsets, right after city data.
	relationshipDataSize := relationshipBitsetSize(int(gameDescriptionHeader.TeamCount)) * diplomacyRelationshipTypeCount
	relationshipDataBytes, err := readByteArray(reader, uint32(relationshipDataSize))
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}

	// City-state influence array, right after the relationship bitsets.
	influenceDataSize := int(gameDescriptionHeader.PlayerCount) * MaxCityStatesPerPlayer * 4
	influenceDataBytes, err := readByteArray(reader, uint32(influenceDataSize))
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}

	// Per-team tile visibility bitset, right after the influence array.
	visibilityDataSize := (mapSize.Width*mapSize.Height*int(gameDescriptionHeader.TeamCount) + 7) / 8
	visibilityDataBytes, err := readByteArray(reader, uint32(visibilityDataSize))
	if err != nil {
		return gameDescriptionHeader, gameDescriptionSectionData{}, typeLists, err
	}

	return gameDescriptionHeader, gameDescriptionSectionData{
		UnitDataBytes:         unitDataBytes,
		CityDataBytes:         cityDataBytes,
		RelationshipDataBytes: relationshipDataBytes,
		InfluenceDataBytes:    influenceDataBytes,
		VisibilityDataBytes:   visibilityDataBytes,
	}, typeLists, nil
}

// readFileTail reads a fixed-size section of a file, ending precedingBytes before the end of the file
func readFileTail(inputFile io.ReaderAt, fileLength int64, size, precedingBytes int) ([]byte, error) {
	data := make([]byte, size)
	offset := fileLength - int64(precedingBytes) - int64(size)
	if _, err := inputFile.ReadAt(data, offset); err != nil {
		return nil, err
	}
	return data, nil
}

// readTailSections reads map tile properties, player civ data, and team names from the end of the file.
func readTailSections(inputFile io.ReaderAt, fileLength int64, mapHeader *Civ5MapHeader, gameDescriptionHeader *Civ5GameDescriptionHeader, policyTypeList []string) ([][]*Civ5MapTileImprovement, []*Civ5PlayerData, []*Civ5TeamData, error) {
	mapTilePropertiesSize := int(mapHeader.Height) * int(mapHeader.Width) * binary.Size(Civ5MapTileHeader{})
	mapTileProperties, err := readFileTail(inputFile, fileLength, mapTilePropertiesSize, 0)
	if err != nil {
		return nil, nil, nil, err
	}

	mapTileImprovementData, err := ParseMapTileProperties(mapTileProperties, int(mapHeader.Height), int(mapHeader.Width))
	if err != nil {
		return nil, nil, nil, err
	}

	playerCivDataSize := CivDataSize * (int(gameDescriptionHeader.PlayerCount) + int(gameDescriptionHeader.CityStateCount))
	playerCivData, err := readFileTail(inputFile, fileLength, playerCivDataSize, mapTilePropertiesSize)
	if err != nil {
		return nil, nil, nil, err
	}

	allPlayerData, err := ParseCivData(playerCivData, policyTypeList)
	if err != nil {
		return nil, nil, nil, err
	}

	teamCount := int(gameDescriptionHeader.TeamCount)
	teamNames, err := readTeamNamesSection(inputFile, fileLength, mapTilePropertiesSize+playerCivDataSize, teamCount)
	if err != nil {
		return nil, nil, nil, err
	}

	return mapTileImprovementData, allPlayerData, teamNames, nil
}

// readTeamNamesSection reads the team name array and decodes it via ParseTeamNames.
func readTeamNamesSection(inputFile io.ReaderAt, fileLength int64, precedingBytes int, teamCount int) ([]*Civ5TeamData, error) {
	if teamCount <= 0 {
		return nil, nil
	}

	teamNameDataSize := TeamNameRecordSize * teamCount
	teamNameData, err := readFileTail(inputFile, fileLength, teamNameDataSize, precedingBytes)
	if err != nil {
		return nil, err
	}

	return ParseTeamNames(teamNameData, teamCount), nil
}

func ReadCiv5MapFile(filename string) (*Civ5MapData, error) {
	inputFile, fileLength, streamReader, err := openMapFileReader(filename)
	if err != nil {
		return nil, err
	}

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

	gameDescriptionHeader, sectionData, typeLists, err := readGameDescriptionSection(streamReader, version, MapSize{Height: int(mapHeader.Height), Width: int(mapHeader.Width)})
	if err != nil {
		return nil, err
	}

	teamCount := int(gameDescriptionHeader.TeamCount)
	teamRelationships := ParseTeamRelationships(sectionData.RelationshipDataBytes, teamCount)
	teamVisibility := ParseTeamVisibility(sectionData.VisibilityDataBytes, MapSize{Height: int(mapHeader.Height), Width: int(mapHeader.Width)}, teamCount)

	mapTileImprovementData, allPlayerData, teamData, err := readTailSections(inputFile, fileLength, &mapHeader, &gameDescriptionHeader, typeLists.PolicyTypeList)
	if err != nil {
		return nil, err
	}

	maxCityId := findMaxCityId(mapTileImprovementData, int(mapHeader.Height), int(mapHeader.Width))
	fmt.Println("Max city id is", maxCityId)

	cityData, err := ParseCityData(sectionData.CityDataBytes, version, maxCityId, typeLists.BuildingTypeList)
	if err != nil {
		return nil, err
	}

	unitData, err := ParseUnitData(sectionData.UnitDataBytes, version, typeLists)
	if err != nil {
		return nil, err
	}

	cityStateInfluence := ParseCityStateInfluence(sectionData.InfluenceDataBytes, int(gameDescriptionHeader.PlayerCount), int(gameDescriptionHeader.CityStateCount))
	for i, influence := range cityStateInfluence {
		if i < len(allPlayerData) {
			allPlayerData[i].CityStateInfluence = influence
		}
	}

	if len(cityData) > 0 {
		resolvedCities := resolveCityNames(mapTileImprovementData, cityData, mapHeader.Height, mapHeader.Width)
		reportCityNames(resolvedCities)
	}

	cityOwnerMap, cityOwnerIndexMap := buildCityOwnerMaps(cityData, gameDescriptionHeader.PlayerCount, gameDescriptionHeader.CityStateCount)
	reportCityOwnerMaps(cityOwnerMap, cityOwnerIndexMap)

	reportTeamRelationships(teamData, teamRelationships)

	return buildMapData(&mapHeader, terrainList, featureTerrainList, resourceList,
		mapTiles, mapTileImprovementData, cityData, unitData, allPlayerData, teamData,
		teamRelationships, teamVisibility, cityOwnerIndexMap, typeLists), nil
}
