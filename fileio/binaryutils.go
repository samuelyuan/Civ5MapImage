package fileio

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"strconv"
	"strings"
)

// Constants for binary reading operations
const (
	MaxArrayLength = 100000 // Maximum reasonable array length to prevent memory issues
)

// readVarString reads a variable-length string from the binary stream
// Format: [length:uint32][string:bytes]
func readVarString(reader *io.SectionReader, varName string) (string, error) {
	variableLength := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &variableLength); err != nil {
		return "", fmt.Errorf("failed to load variable length for %s: %w", varName, err)
	}

	stringValue := make([]byte, variableLength)
	if err := binary.Read(reader, binary.LittleEndian, &stringValue); err != nil {
		return "", fmt.Errorf("failed to load string value for %s (length: %v): %w", varName, variableLength, err)
	}

	return string(stringValue[:]), nil
}

// readArray reads an array of file config entries from the binary stream
func readArray(reader *io.SectionReader, arrayName string, fileConfigEntries []Civ5ReplayFileConfigEntry) error {
	arrayLength := unsafeReadUint32(reader)
	if arrayLength > MaxArrayLength {
		return fmt.Errorf("array length may be too long for %s: %d", arrayName, arrayLength)
	}
	for i := 0; i < int(arrayLength); i++ {
		if _, err := readFileConfig(reader, fileConfigEntries); err != nil {
			return fmt.Errorf("failed to read array element %d for %s: %w", i, arrayName, err)
		}
	}
	return nil
}

// readFileConfig reads a file configuration entry from the binary stream
func readFileConfig(reader *io.SectionReader, fileConfigEntries []Civ5ReplayFileConfigEntry) ([]string, error) {
	pos, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}

	fieldValues := make([]string, 0)

	for i := 0; i < len(fileConfigEntries); i++ {
		fileConfigEntry := fileConfigEntries[i]
		if fileConfigEntry.VariableType == "varstring" {
			value, err := readVarString(reader, "varstring_"+fileConfigEntry.VariableName)
			if err != nil {
				return nil, err
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(str):%v", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "float32" {
			value := float32(0)
			if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
				return nil, fmt.Errorf("failed to load float32 for %s: %w", fileConfigEntry.VariableName, err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(f32):%f", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "uint32" {
			value := unsafeReadUint32(reader)
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u32):%d", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "int32" {
			signedIntValue := int32(0)
			if err := binary.Read(reader, binary.LittleEndian, &signedIntValue); err != nil {
				return nil, fmt.Errorf("failed to load int32 for %s: %w", fileConfigEntry.VariableName, err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(i32):%d", fileConfigEntry.VariableName, signedIntValue))
		} else if fileConfigEntry.VariableType == "uint16" {
			value := unsafeReadUint16(reader)
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u16):%d", fileConfigEntry.VariableName, value))
		} else if fileConfigEntry.VariableType == "uint8" {
			unsignedIntValue := uint8(0)
			if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
				return nil, fmt.Errorf("failed to load uint8 for %s: %w", fileConfigEntry.VariableName, err)
			}
			fieldValues = append(fieldValues, fmt.Sprintf("%v(u8):%d", fileConfigEntry.VariableName, unsignedIntValue))
		} else if strings.Contains(fileConfigEntry.VariableType, "bytearray") {
			byteArrayLength, err := strconv.Atoi(fileConfigEntry.VariableType[len("bytearray:"):])
			if err != nil {
				return nil, fmt.Errorf("invalid byte array type in file config for %s: %w", fileConfigEntry.VariableName, err)
			}

			byteBlock := make([]byte, byteArrayLength)
			if err := binary.Read(reader, binary.LittleEndian, &byteBlock); err != nil {
				return nil, fmt.Errorf("invalid byte array data for %s: %w", fileConfigEntry.VariableName, err)
			}

			fieldValues = append(fieldValues, fmt.Sprintf("%v(bytearray):%v", fileConfigEntry.VariableName, byteBlock))
		} else {
			fmt.Println("Unknown variable type:", fileConfigEntry.VariableType)
		}
	}

	fmt.Printf("File Pos: 0x%X, ", pos)
	fmt.Println("Field values:", fieldValues)
	return fieldValues, nil
}

// unsafeReadUint32 reads a uint32 from the binary stream, panicking on error
// This is used for internal operations where errors should not occur
func unsafeReadUint32(reader *io.SectionReader) uint32 {
	unsignedIntValue := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		panic(fmt.Sprintf("failed to load uint32: %v", err))
	}
	return unsignedIntValue
}

// unsafeReadUint16 reads a uint16 from the binary stream, panicking on error
// This is used for internal operations where errors should not occur
func unsafeReadUint16(reader *io.SectionReader) uint16 {
	unsignedIntValue := uint16(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		panic(fmt.Sprintf("failed to load uint16: %v", err))
	}
	return unsignedIntValue
}

// unsafeReadFixedInt32Array reads a fixed-length array of int32 values with no length prefix
func unsafeReadFixedInt32Array(reader *io.SectionReader, count int) []int32 {
	values := make([]int32, count)
	for i := 0; i < count; i++ {
		values[i] = int32(unsafeReadUint32(reader))
	}
	return values
}

// unsafeReadByte reads a single byte from the binary stream, panicking on error.
func unsafeReadByte(reader *io.SectionReader) byte {
	value := make([]byte, 1)
	if _, err := io.ReadFull(reader, value); err != nil {
		panic(fmt.Sprintf("failed to load byte: %v", err))
	}
	return value[0]
}

// unsafeReadFixedBytes reads a fixed-length raw byte block from the binary stream, panicking on error.
func unsafeReadFixedBytes(reader *io.SectionReader, count int) []byte {
	value := make([]byte, count)
	if _, err := io.ReadFull(reader, value); err != nil {
		panic(fmt.Sprintf("failed to load %d raw bytes: %v", count, err))
	}
	return value
}

type HashValuePair struct {
	Hash  uint32
	Value int32
}

func readHashValuePairs(reader *io.SectionReader, count int) []HashValuePair {
	pairs := make([]HashValuePair, count)
	for i := 0; i < count; i++ {
		hash := unsafeReadUint32(reader)
		var value int32
		if hash != 0 {
			value = int32(unsafeReadUint32(reader))
		}
		pairs[i] = HashValuePair{Hash: hash, Value: value}
	}
	return pairs
}

type HashBoolPair struct {
	Hash  uint32
	Value bool
}

func readHashBoolPairs(reader *io.SectionReader, count int) []HashBoolPair {
	pairs := make([]HashBoolPair, count)
	for i := 0; i < count; i++ {
		hash := unsafeReadUint32(reader)
		var value bool
		if hash != 0 {
			value = unsafeReadByte(reader) != 0
		}
		pairs[i] = HashBoolPair{Hash: hash, Value: value}
	}
	return pairs
}

type HashIntArrayPair struct {
	Hash   uint32
	Values []int32
}

func readHashIntArrayPairs(reader *io.SectionReader, count int, subArraySize int) []HashIntArrayPair {
	pairs := make([]HashIntArrayPair, count)
	for i := 0; i < count; i++ {
		hash := unsafeReadUint32(reader)
		var values []int32
		if hash != 0 {
			values = unsafeReadFixedInt32Array(reader, subArraySize)
		}
		pairs[i] = HashIntArrayPair{Hash: hash, Values: values}
	}
	return pairs
}

type FreeListArrayHeader struct {
	NumSlots      int32
	LastIndex     int32
	FreeListHead  int32
	FreeListCount int32
	CurrentID     int32
	NextFreeIndex []int32
}

func readFreeListArrayHeader(reader *io.SectionReader) FreeListArrayHeader {
	numSlots := int32(unsafeReadUint32(reader))
	lastIndex := int32(unsafeReadUint32(reader))
	freeListHead := int32(unsafeReadUint32(reader))
	freeListCount := int32(unsafeReadUint32(reader))
	currentID := int32(unsafeReadUint32(reader))
	nextFreeIndex := unsafeReadFixedInt32Array(reader, int(numSlots))
	return FreeListArrayHeader{
		NumSlots:      numSlots,
		LastIndex:     lastIndex,
		FreeListHead:  freeListHead,
		FreeListCount: freeListCount,
		CurrentID:     currentID,
		NextFreeIndex: nextFreeIndex,
	}
}

func readEmptyFreeListTrashArray(reader *io.SectionReader, name string) FreeListArrayHeader {
	header := readFreeListArrayHeader(reader)
	count := unsafeReadUint32(reader)
	if count != 0 {
		panic(fmt.Sprintf("%s: non-empty FreeListTrashArray (count=%d) not supported", name, count))
	}
	return header
}

type RandomState struct {
	Version    uint32
	Seed       uint32
	CallCount  uint32
	ResetCount uint32
}

func readRandomState(reader *io.SectionReader) RandomState {
	version := unsafeReadUint32(reader)
	seed := unsafeReadUint32(reader)
	callCount := unsafeReadUint32(reader)
	resetCount := unsafeReadUint32(reader)
	unsafeReadByte(reader) // extended callstack debugging flag - always false in a release build
	return RandomState{Version: version, Seed: seed, CallCount: callCount, ResetCount: resetCount}
}

type PlotExtraYield struct {
	X          int32
	Y          int32
	ExtraYield []int32
}

func readPlotExtraYields(reader *io.SectionReader) []PlotExtraYield {
	count := unsafeReadUint32(reader)
	yields := make([]PlotExtraYield, count)
	for i := 0; i < int(count); i++ {
		x := int32(unsafeReadUint32(reader))
		y := int32(unsafeReadUint32(reader))
		extraYieldCount := unsafeReadUint32(reader)
		extraYield := unsafeReadFixedInt32Array(reader, int(extraYieldCount))
		yields[i] = PlotExtraYield{X: x, Y: y, ExtraYield: extraYield}
	}
	return yields
}

type TradedItem struct {
	ItemType    int32
	Duration    int32
	FinalTurn   int32
	Data1       int32
	Data2       int32
	Data3       int32 // only if version >= 2
	Flag1       bool  // only if version >= 2
	FromPlayer  int32
	FromRenewed bool
	ToRenewed   bool
}

func readTradedItem(reader *io.SectionReader) TradedItem {
	version := unsafeReadUint32(reader)
	itemType := int32(unsafeReadUint32(reader))
	duration := int32(unsafeReadUint32(reader))
	finalTurn := int32(unsafeReadUint32(reader))
	data1 := int32(unsafeReadUint32(reader))
	data2 := int32(unsafeReadUint32(reader))
	var data3 int32
	var flag1 bool
	if version >= 2 {
		data3 = int32(unsafeReadUint32(reader))
		flag1 = unsafeReadByte(reader) != 0
	}
	fromPlayer := int32(unsafeReadUint32(reader))
	fromRenewed := unsafeReadByte(reader) != 0
	toRenewed := unsafeReadByte(reader) != 0
	return TradedItem{
		ItemType: itemType, Duration: duration, FinalTurn: finalTurn,
		Data1: data1, Data2: data2, Data3: data3, Flag1: flag1,
		FromPlayer: fromPlayer, FromRenewed: fromRenewed, ToRenewed: toRenewed,
	}
}

// Diplomatic deal (proposed, active, or historical).
type Deal struct {
	FromPlayer            int32
	ToPlayer              int32
	FinalTurn             int32
	Duration              int32
	StartTurn             int32
	ConsideringForRenewal bool
	CheckedForRenewal     bool // only if version >= 3
	DealCancelled         bool
	PeaceTreatyType       int32
	SurrenderingPlayer    int32
	DemandingPlayer       int32
	RequestingPlayer      int32
	TradedItems           []TradedItem
}

func readDeal(reader *io.SectionReader) Deal {
	version := unsafeReadUint32(reader)
	fromPlayer := int32(unsafeReadUint32(reader))
	toPlayer := int32(unsafeReadUint32(reader))
	finalTurn := int32(unsafeReadUint32(reader))
	duration := int32(unsafeReadUint32(reader))
	startTurn := int32(unsafeReadUint32(reader))
	consideringForRenewal := unsafeReadByte(reader) != 0
	var checkedForRenewal bool
	if version >= 3 {
		checkedForRenewal = unsafeReadByte(reader) != 0
	}
	dealCancelled := unsafeReadByte(reader) != 0
	peaceTreatyType := int32(unsafeReadUint32(reader))
	surrenderingPlayer := int32(unsafeReadUint32(reader))
	demandingPlayer := int32(unsafeReadUint32(reader))
	requestingPlayer := int32(unsafeReadUint32(reader))
	entriesToRead := unsafeReadUint32(reader)
	if version < 2 {
		panic(fmt.Sprintf("readDeal: version %d (pre-version-2 traded item format) not supported", version))
	}
	tradedItems := make([]TradedItem, entriesToRead)
	for i := range tradedItems {
		tradedItems[i] = readTradedItem(reader)
	}
	return Deal{
		FromPlayer: fromPlayer, ToPlayer: toPlayer, FinalTurn: finalTurn,
		Duration: duration, StartTurn: startTurn,
		ConsideringForRenewal: consideringForRenewal, CheckedForRenewal: checkedForRenewal, DealCancelled: dealCancelled,
		PeaceTreatyType: peaceTreatyType, SurrenderingPlayer: surrenderingPlayer,
		DemandingPlayer: demandingPlayer, RequestingPlayer: requestingPlayer,
		TradedItems: tradedItems,
	}
}

// Proposed, active, and historical deals.
type GameDeals struct {
	ProposedDeals   []Deal
	CurrentDeals    []Deal
	HistoricalDeals []Deal
}

func readGameDeals(reader *io.SectionReader) GameDeals {
	unsafeReadUint32(reader) // version - not needed

	readDealArray := func() []Deal {
		count := unsafeReadUint32(reader)
		deals := make([]Deal, count)
		for i := range deals {
			deals[i] = readDeal(reader)
		}
		return deals
	}

	return GameDeals{
		ProposedDeals:   readDealArray(),
		CurrentDeals:    readDealArray(),
		HistoricalDeals: readDealArray(),
	}
}

// The effects granted by a religion (or pantheon), plus which beliefs were chosen and
// which building classes they enable.
type ReligionBeliefs struct {
	// 22 individually-named int modifiers, in file order:
	// FaithFromDyingUnits, RiverHappiness, PlotCultureCostModifier, CityRangeStrikeModifier,
	// CombatModifierEnemyCities, CombatModifierFriendlyCities, FriendlyHealChange,
	// CityStateFriendshipModifier, LandBarbarianConversionPercent, SpreadStrengthModifier,
	// SpreadDistanceModifier, ProphetStrengthModifier, ProphetCostModifier,
	// MissionaryStrengthModifier, MissionaryCostModifier, FriendlyCityStateSpreadModifier,
	// GreatPersonExpendedFaith, CityStateMinimumInfluence, CityStateInfluenceModifier,
	// OtherReligionPressureErosion, SpyPressure, InquisitorPressureRetention
	Modifiers                  []int32
	FaithBuildingTourism       int32 // only if version >= 2
	ObsoleteEra                int32
	ResourceRevealed           int32
	SpreadModifierDoublingTech int32
	BeliefHashes               []uint32        // Hash of each chosen belief's type string
	BuildingClassOverrides     []HashValuePair // per-building-class faith/yield override
}

func readReligionBeliefs(reader *io.SectionReader) ReligionBeliefs {
	version := unsafeReadUint32(reader)
	modifiers := unsafeReadFixedInt32Array(reader, 22)

	var faithBuildingTourism int32
	if version >= 2 {
		faithBuildingTourism = int32(unsafeReadUint32(reader))
	}

	obsoleteEra := int32(unsafeReadUint32(reader))
	resourceRevealed := int32(unsafeReadUint32(reader))
	spreadModifierDoublingTech := int32(unsafeReadUint32(reader))

	beliefCount := unsafeReadUint32(reader)
	beliefHashes := make([]uint32, beliefCount)
	for i := range beliefHashes {
		beliefHashes[i] = unsafeReadUint32(reader)
	}

	buildingClassCount := unsafeReadUint32(reader)
	buildingClassOverrides := readHashValuePairs(reader, int(buildingClassCount))

	return ReligionBeliefs{
		Modifiers: modifiers, FaithBuildingTourism: faithBuildingTourism,
		ObsoleteEra: obsoleteEra, ResourceRevealed: resourceRevealed, SpreadModifierDoublingTech: spreadModifierDoublingTech,
		BeliefHashes: beliefHashes, BuildingClassOverrides: buildingClassOverrides,
	}
}

// One founded religion or pantheon
// A pantheon is its own entry here too (ReligionType==0/RELIGION_PANTHEON, Pantheon==true,
// exactly 1 belief and a meaningless/uninitialized HolyCityX/Y - a pantheon has no holy city).
type Religion struct {
	ReligionType int32
	Founder      int32
	HolyCityX    int32
	HolyCityY    int32
	TurnFounded  int32
	Pantheon     bool // only if version >= 2
	Enhanced     bool // only if version >= 4
	CustomName   string
	Beliefs      ReligionBeliefs
}

func readReligion(reader *io.SectionReader) Religion {
	version := unsafeReadUint32(reader)
	religionType := int32(unsafeReadUint32(reader))
	founder := int32(unsafeReadUint32(reader))
	holyCityX := int32(unsafeReadUint32(reader))
	holyCityY := int32(unsafeReadUint32(reader))
	turnFounded := int32(unsafeReadUint32(reader))

	var pantheon, enhanced bool
	if version >= 2 {
		pantheon = unsafeReadByte(reader) != 0
	}
	if version >= 4 {
		enhanced = unsafeReadByte(reader) != 0
	}

	var customName string
	if version >= 3 {
		nameBytes := make([]byte, 128) // fixed-size null-terminated buffer
		for i := range nameBytes {
			nameBytes[i] = unsafeReadByte(reader)
		}
		n := 0
		for n < len(nameBytes) && nameBytes[n] != 0 {
			n++
		}
		customName = string(nameBytes[:n])
	}

	return Religion{
		ReligionType: religionType, Founder: founder, HolyCityX: holyCityX, HolyCityY: holyCityY,
		TurnFounded: turnFounded, Pantheon: pantheon, Enhanced: enhanced, CustomName: customName,
		Beliefs: readReligionBeliefs(reader),
	}
}

type GameReligions struct {
	MinimumFaithNextPantheon int32
	CurrentReligions         []Religion
}

func readGameReligions(reader *io.SectionReader) GameReligions {
	version := unsafeReadUint32(reader)

	var minFaithNextPantheon int32
	if version >= 3 {
		minFaithNextPantheon = int32(unsafeReadUint32(reader))
	}
	if version < 4 {
		unsafeReadUint32(reader) // legacy variable for minimumFaithNextGreatProphet, eliminated in version 4, discarded
	}

	var religions []Religion
	if version >= 2 {
		count := unsafeReadUint32(reader)
		religions = make([]Religion, count)
		for i := range religions {
			religions[i] = readReligion(reader)
		}
	}

	return GameReligions{MinimumFaithNextPantheon: minFaithNextPantheon, CurrentReligions: religions}
}

// GreatWork is one Great Work slotted into a building.
type GreatWork struct {
	GreatPersonName string
	GWType          int32
	ClassType       int32 // only if version >= 3; GreatWorkClass: 1=Art, 2=Artifact, 3=Literature, 4=Music
	TurnFounded     int32
	Era             int32
	Player          int32
}

func readGreatWork(reader *io.SectionReader) GreatWork {
	version := unsafeReadUint32(reader)
	if version == 1 {
		if _, err := readVarString(reader, "oldGreatWorkName"); err != nil { // legacy field, discarded
			panic(err)
		}
	}
	greatPersonName, err := readVarString(reader, "greatPersonName")
	if err != nil {
		panic(err)
	}
	gwType := int32(unsafeReadUint32(reader))
	var classType int32
	if version >= 3 {
		classType = int32(unsafeReadUint32(reader))
	}
	turnFounded := int32(unsafeReadUint32(reader))
	era := int32(unsafeReadUint32(reader))
	player := int32(unsafeReadUint32(reader))
	return GreatWork{
		GreatPersonName: greatPersonName, GWType: gwType, ClassType: classType,
		TurnFounded: turnFounded, Era: era, Player: player,
	}
}

// GameCulture holds the game's great works and whether cultural influence has been reported.
type GameCulture struct {
	CurrentGreatWorks          []GreatWork
	ReportedSomeoneInfluential bool // only if version >= 2
}

func readGameCulture(reader *io.SectionReader) GameCulture {
	version := unsafeReadUint32(reader)

	count := unsafeReadUint32(reader)
	greatWorks := make([]GreatWork, count)
	for i := range greatWorks {
		greatWorks[i] = readGreatWork(reader)
	}

	var reportedSomeoneInfluential bool
	if version >= 2 {
		reportedSomeoneInfluential = unsafeReadByte(reader) != 0
	}

	return GameCulture{CurrentGreatWorks: greatWorks, ReportedSomeoneInfluential: reportedSomeoneInfluential}
}

// ResolutionDecision is which of the possible choices for a resolution/proposal type was decided
// (e.g. which player to embargo).
type ResolutionDecision struct {
	Type int32
}

func readResolutionDecision(reader *io.SectionReader) ResolutionDecision {
	unsafeReadUint32(reader) // version, always 1, no version-gated fields
	return ResolutionDecision{Type: int32(unsafeReadUint32(reader))}
}

// PlayerVote is one player's vote weight and choice.
type PlayerVote struct {
	Player   int32
	NumVotes int32
	Choice   int32
}

func readPlayerVote(reader *io.SectionReader) PlayerVote {
	unsafeReadUint32(reader) // version, always 1, no version-gated fields
	return PlayerVote{
		Player:   int32(unsafeReadUint32(reader)),
		NumVotes: int32(unsafeReadUint32(reader)),
		Choice:   int32(unsafeReadUint32(reader)),
	}
}

// ProposerDecision is the proposing player's own single vote/choice.
type ProposerDecision struct {
	Decision ResolutionDecision
	Vote     PlayerVote
}

func readProposerDecision(reader *io.SectionReader) ProposerDecision {
	decision := readResolutionDecision(reader)
	unsafeReadUint32(reader) // version, always 1, no version-gated fields
	return ProposerDecision{Decision: decision, Vote: readPlayerVote(reader)}
}

// VoterDecision is every player's vote cast on a resolution/proposal.
type VoterDecision struct {
	Decision ResolutionDecision
	Votes    []PlayerVote
}

func readVoterDecision(reader *io.SectionReader) VoterDecision {
	decision := readResolutionDecision(reader)
	unsafeReadUint32(reader) // version, always 1, no version-gated fields
	count := unsafeReadUint32(reader)
	votes := make([]PlayerVote, count)
	for i := range votes {
		votes[i] = readPlayerVote(reader)
	}
	return VoterDecision{Decision: decision, Votes: votes}
}

// ResolutionEffects is the full set of possible effects a World Congress resolution can have; only
// the fields relevant to its ResolutionType are meaningful, the rest sit at their zero value.
type ResolutionEffects struct {
	DiplomaticVictory                 bool
	ChangeLeagueHost                  bool // only if version >= 2
	OneTimeGold                       int32
	OneTimeGoldPercent                int32
	RaiseCityStateInfluenceToNeutral  bool
	LeagueProjectEnabled              int32 // only if version >= 3
	GoldPerTurn                       int32
	ResourceQuantity                  int32
	EmbargoCityStates                 bool
	EmbargoPlayer                     bool
	NoResourceHappiness               bool
	UnitMaintenanceGoldPercent        int32
	MemberDiscoveredTechMod           int32
	CulturePerWonder                  int32 // only if version >= 4
	CulturePerNaturalWonder           int32 // only if version >= 4
	NoTrainingNuclearWeapons          bool  // only if version >= 5
	VotesForFollowingReligion         int32 // only if version >= 6
	HolyCityTourism                   int32 // only if version >= 6
	ReligionSpreadStrengthMod         int32 // only if version >= 9
	VotesForFollowingIdeology         int32 // only if version >= 6
	OtherIdeologyRebellionMod         int32 // only if version >= 6
	ArtsyGreatPersonRateMod           int32 // only if version >= 7
	ScienceyGreatPersonRateMod        int32 // only if version >= 7
	GreatPersonTileImprovementCulture int32 // only if version >= 8
	LandmarkCulture                   int32 // only if version >= 8
}

func readResolutionEffects(reader *io.SectionReader) ResolutionEffects {
	version := unsafeReadUint32(reader)
	e := ResolutionEffects{}
	e.DiplomaticVictory = unsafeReadByte(reader) != 0
	if version >= 2 {
		e.ChangeLeagueHost = unsafeReadByte(reader) != 0
	}
	e.OneTimeGold = int32(unsafeReadUint32(reader))
	e.OneTimeGoldPercent = int32(unsafeReadUint32(reader))
	e.RaiseCityStateInfluenceToNeutral = unsafeReadByte(reader) != 0
	if version >= 3 {
		e.LeagueProjectEnabled = int32(unsafeReadUint32(reader))
	}
	e.GoldPerTurn = int32(unsafeReadUint32(reader))
	e.ResourceQuantity = int32(unsafeReadUint32(reader))
	e.EmbargoCityStates = unsafeReadByte(reader) != 0
	e.EmbargoPlayer = unsafeReadByte(reader) != 0
	e.NoResourceHappiness = unsafeReadByte(reader) != 0
	e.UnitMaintenanceGoldPercent = int32(unsafeReadUint32(reader))
	e.MemberDiscoveredTechMod = int32(unsafeReadUint32(reader))
	if version >= 4 {
		e.CulturePerWonder = int32(unsafeReadUint32(reader))
		e.CulturePerNaturalWonder = int32(unsafeReadUint32(reader))
	}
	if version >= 5 {
		e.NoTrainingNuclearWeapons = unsafeReadByte(reader) != 0
	}
	if version >= 6 {
		e.VotesForFollowingReligion = int32(unsafeReadUint32(reader))
		e.HolyCityTourism = int32(unsafeReadUint32(reader))
	}
	if version >= 9 {
		e.ReligionSpreadStrengthMod = int32(unsafeReadUint32(reader))
	}
	if version >= 6 {
		e.VotesForFollowingIdeology = int32(unsafeReadUint32(reader))
		e.OtherIdeologyRebellionMod = int32(unsafeReadUint32(reader))
	}
	if version >= 7 {
		e.ArtsyGreatPersonRateMod = int32(unsafeReadUint32(reader))
		e.ScienceyGreatPersonRateMod = int32(unsafeReadUint32(reader))
	}
	if version >= 8 {
		e.GreatPersonTileImprovementCulture = int32(unsafeReadUint32(reader))
		e.LandmarkCulture = int32(unsafeReadUint32(reader))
	}
	return e
}

// Resolution is the base type shared by an enacted/repeal proposal and an active (already enacted)
// resolution.
type Resolution struct {
	ID               int32 // only if version >= 2, else generated fresh at load and not read from the stream
	Type             int32
	League           int32
	Effects          ResolutionEffects
	VoterDecision    VoterDecision
	ProposerDecision ProposerDecision
}

func readResolution(reader *io.SectionReader) Resolution {
	version := unsafeReadUint32(reader)
	var id int32
	if version >= 2 {
		id = int32(unsafeReadUint32(reader))
	}
	resType := int32(unsafeReadUint32(reader))
	league := int32(unsafeReadUint32(reader))
	effects := readResolutionEffects(reader)
	voterDecision := readVoterDecision(reader)
	proposerDecision := readProposerDecision(reader)
	return Resolution{
		ID: id, Type: resType, League: league,
		Effects: effects, VoterDecision: voterDecision, ProposerDecision: proposerDecision,
	}
}

// Proposal is the base type shared by an enact proposal and a repeal proposal.
type Proposal struct {
	Resolution     Resolution
	ProposalPlayer int32
}

func readProposal(reader *io.SectionReader) Proposal {
	resolution := readResolution(reader)
	unsafeReadUint32(reader) // version, always 1, no version-gated fields
	return Proposal{Resolution: resolution, ProposalPlayer: int32(unsafeReadUint32(reader))}
}

// EnactProposal is a proposal not yet voted on/enacted.
type EnactProposal struct {
	Proposal Proposal
}

func readEnactProposal(reader *io.SectionReader) EnactProposal {
	proposal := readProposal(reader)
	unsafeReadUint32(reader) // version, always 1, no extra fields
	return EnactProposal{Proposal: proposal}
}

// ActiveResolution is a resolution that has already been enacted and is in effect.
type ActiveResolution struct {
	Resolution  Resolution
	TurnEnacted int32
}

func readActiveResolution(reader *io.SectionReader) ActiveResolution {
	resolution := readResolution(reader)
	version := unsafeReadUint32(reader)
	if version < 2 {
		unsafeReadUint32(reader) // legacy ID field, moved into Resolution itself as of version 2, discarded
	}
	return ActiveResolution{Resolution: resolution, TurnEnacted: int32(unsafeReadUint32(reader))}
}

// RepealProposal is a proposal to repeal an already-active resolution.
type RepealProposal struct {
	Proposal           Proposal
	TargetResolutionID int32
	RepealDecision     VoterDecision
}

func readRepealProposal(reader *io.SectionReader) RepealProposal {
	proposal := readProposal(reader)
	unsafeReadUint32(reader) // version, always 1, no version-gated fields
	targetResolutionID := int32(unsafeReadUint32(reader))
	return RepealProposal{
		Proposal: proposal, TargetResolutionID: targetResolutionID, RepealDecision: readVoterDecision(reader),
	}
}

// LeagueMember is one civ's membership record within a League.
type LeagueMember struct {
	Player         int32
	ExtraVotes     int32  // only if version >= 9
	VoteSources    string // only if version >= 10
	MayPropose     bool
	Proposals      int32 // only if version >= 12
	Votes          int32
	AbstainedVotes int32 // only if version >= 13
	EverBeenHost   bool  // only if version >= 14
	AlwaysBeenHost bool  // only if version >= 14, real default true when absent
}

func readLeagueMember(reader *io.SectionReader, version uint32) LeagueMember {
	m := LeagueMember{AlwaysBeenHost: true}
	m.Player = int32(unsafeReadUint32(reader))
	if version >= 9 {
		m.ExtraVotes = int32(unsafeReadUint32(reader))
	}
	if version >= 10 {
		s, err := readVarString(reader, "voteSources")
		if err != nil {
			panic(err)
		}
		m.VoteSources = s
	}
	m.MayPropose = unsafeReadByte(reader) != 0
	if version >= 12 {
		m.Proposals = int32(unsafeReadUint32(reader))
	}
	m.Votes = int32(unsafeReadUint32(reader))
	if version >= 13 {
		m.AbstainedVotes = int32(unsafeReadUint32(reader))
	}
	if version >= 14 {
		m.EverBeenHost = unsafeReadByte(reader) != 0
		m.AlwaysBeenHost = unsafeReadByte(reader) != 0
	}
	return m
}

// LeagueProject is a World Congress project (e.g. International Space Station) and each member's
// production contribution toward it.
type LeagueProject struct {
	Type                int32
	ProductionList      []int32
	Complete            bool // only if version >= 6
	ProgressWarningSent bool // only if version >= 11
}

func readLeagueProject(reader *io.SectionReader, version uint32) LeagueProject {
	p := LeagueProject{}
	p.Type = int32(unsafeReadUint32(reader))
	listSize := unsafeReadUint32(reader)
	p.ProductionList = unsafeReadFixedInt32Array(reader, int(listSize))
	if version >= 6 {
		p.Complete = unsafeReadByte(reader) != 0
	}
	if version >= 11 {
		p.ProgressWarningSent = unsafeReadByte(reader) != 0
	}
	return p
}

// League is one World Congress/United Nations instance.
type League struct {
	ID                        int32
	UnitedNations             bool // only if version >= 4
	InSession                 bool // only if version >= 2
	TurnsUntilSession         int32
	NumResolutionsEverEnacted int32
	EnactProposals            []EnactProposal
	RepealProposals           []RepealProposal
	ActiveResolutions         []ActiveResolution
	Members                   []LeagueMember
	Host                      int32            // only if version >= 3
	Projects                  []LeagueProject  // only if version >= 5
	ConsecutiveHostedSessions int32            // only if version >= 7
	AssignedName              int32            // only if version >= 7
	CustomName                string           // only if version >= 7, fixed char[128] buffer
	LastSpecialSession        int32            // only if version >= 8
	CurrentSpecialSession     int32            // only if version >= 8
	EnactProposalsOnHold      []EnactProposal  // only if version >= 8
	RepealProposalsOnHold     []RepealProposal // only if version >= 8
}

func readLeague(reader *io.SectionReader) League {
	version := unsafeReadUint32(reader)
	l := League{}
	l.ID = int32(unsafeReadUint32(reader))
	if version >= 4 {
		l.UnitedNations = unsafeReadByte(reader) != 0
	}
	if version >= 2 {
		l.InSession = unsafeReadByte(reader) != 0
	}
	l.TurnsUntilSession = int32(unsafeReadUint32(reader))
	l.NumResolutionsEverEnacted = int32(unsafeReadUint32(reader))

	enactCount := unsafeReadUint32(reader)
	l.EnactProposals = make([]EnactProposal, enactCount)
	for i := range l.EnactProposals {
		l.EnactProposals[i] = readEnactProposal(reader)
	}

	repealCount := unsafeReadUint32(reader)
	l.RepealProposals = make([]RepealProposal, repealCount)
	for i := range l.RepealProposals {
		l.RepealProposals[i] = readRepealProposal(reader)
	}

	activeCount := unsafeReadUint32(reader)
	l.ActiveResolutions = make([]ActiveResolution, activeCount)
	for i := range l.ActiveResolutions {
		l.ActiveResolutions[i] = readActiveResolution(reader)
	}

	memberCount := unsafeReadUint32(reader)
	l.Members = make([]LeagueMember, memberCount)
	for i := range l.Members {
		l.Members[i] = readLeagueMember(reader, version)
	}

	if version >= 3 {
		l.Host = int32(unsafeReadUint32(reader))
	}

	if version >= 5 {
		projectCount := unsafeReadUint32(reader)
		l.Projects = make([]LeagueProject, projectCount)
		for i := range l.Projects {
			l.Projects[i] = readLeagueProject(reader, version)
		}
	}

	if version >= 7 {
		l.ConsecutiveHostedSessions = int32(unsafeReadUint32(reader))
		l.AssignedName = int32(unsafeReadUint32(reader))
		nameBytes := make([]byte, 128) // fixed-size null-terminated buffer
		for i := range nameBytes {
			nameBytes[i] = unsafeReadByte(reader)
		}
		n := 0
		for n < len(nameBytes) && nameBytes[n] != 0 {
			n++
		}
		l.CustomName = string(nameBytes[:n])
	}

	if version >= 8 {
		l.LastSpecialSession = int32(unsafeReadUint32(reader))
		l.CurrentSpecialSession = int32(unsafeReadUint32(reader))

		enactOnHoldCount := unsafeReadUint32(reader)
		l.EnactProposalsOnHold = make([]EnactProposal, enactOnHoldCount)
		for i := range l.EnactProposalsOnHold {
			l.EnactProposalsOnHold[i] = readEnactProposal(reader)
		}

		repealOnHoldCount := unsafeReadUint32(reader)
		l.RepealProposalsOnHold = make([]RepealProposal, repealOnHoldCount)
		for i := range l.RepealProposalsOnHold {
			l.RepealProposalsOnHold[i] = readRepealProposal(reader)
		}
	}

	return l
}

// GameLeagues holds every active World Congress/United Nations league and related tallies.
type GameLeagues struct {
	GeneratedIDCount      int32 // only if version >= 4
	ActiveLeagues         []League
	NumLeaguesEverFounded int32 // only if version >= 2
	DiplomaticVictor      int32 // only if version >= 3, else NO_PLAYER (-1)
	LastEraTrigger        int32 // only if version >= 5, else NO_ERA (-1)
}

func readGameLeagues(reader *io.SectionReader) GameLeagues {
	version := unsafeReadUint32(reader)
	g := GameLeagues{DiplomaticVictor: -1, LastEraTrigger: -1}
	if version >= 4 {
		g.GeneratedIDCount = int32(unsafeReadUint32(reader))
	}
	leagueCount := unsafeReadUint32(reader)
	g.ActiveLeagues = make([]League, leagueCount)
	for i := range g.ActiveLeagues {
		g.ActiveLeagues[i] = readLeague(reader)
	}
	if version >= 2 {
		g.NumLeaguesEverFounded = int32(unsafeReadUint32(reader))
	}
	if version >= 3 {
		g.DiplomaticVictor = int32(unsafeReadUint32(reader))
	}
	if version >= 5 {
		g.LastEraTrigger = int32(unsafeReadUint32(reader))
	}
	return g
}

// TradeConnectionPlot is one tile of a trade route's path.
type TradeConnectionPlot struct {
	X int32
	Y int32
}

// TradeConnection is one active trade route (land or sea, international or internal) between two cities.
type TradeConnection struct {
	ID                     int32 // only if version >= 1, else MAX_INT (not read from the stream)
	OriginX                int32
	OriginY                int32
	DestX                  int32
	DestY                  int32
	OriginOwner            int32
	DestOwner              int32
	Domain                 int32
	ConnectionType         int32
	TradeUnitLocationIndex int32
	TradeUnitMovingForward bool
	UnitID                 int32
	CircuitsCompleted      int32
	CircuitsToComplete     int32
	TurnRouteComplete      int32 // only if version >= 2
	PlotList               []TradeConnectionPlot
	OriginYields           []int32 // 6 entries: food/production/gold/science/culture/faith
	DestYields             []int32 // 6 entries, same order
}

const numYieldTypes = 6 // food, production, gold, science, culture, faith
const maxMajorCivs = 22

func readTradeConnection(reader *io.SectionReader, version uint32) TradeConnection {
	c := TradeConnection{}
	if version >= 1 {
		c.ID = int32(unsafeReadUint32(reader))
	}
	c.OriginX = int32(unsafeReadUint32(reader))
	c.OriginY = int32(unsafeReadUint32(reader))
	c.DestX = int32(unsafeReadUint32(reader))
	c.DestY = int32(unsafeReadUint32(reader))
	c.OriginOwner = int32(unsafeReadUint32(reader))
	c.DestOwner = int32(unsafeReadUint32(reader))
	c.Domain = int32(unsafeReadUint32(reader))
	c.ConnectionType = int32(unsafeReadUint32(reader))
	c.TradeUnitLocationIndex = int32(unsafeReadUint32(reader))
	c.TradeUnitMovingForward = unsafeReadByte(reader) != 0
	c.UnitID = int32(unsafeReadUint32(reader))
	c.CircuitsCompleted = int32(unsafeReadUint32(reader))
	c.CircuitsToComplete = int32(unsafeReadUint32(reader))
	if version >= 2 {
		c.TurnRouteComplete = int32(unsafeReadUint32(reader))
	}

	plotCount := unsafeReadUint32(reader)
	c.PlotList = make([]TradeConnectionPlot, plotCount)
	for i := range c.PlotList {
		c.PlotList[i] = TradeConnectionPlot{
			X: int32(unsafeReadUint32(reader)),
			Y: int32(unsafeReadUint32(reader)),
		}
	}

	c.OriginYields = make([]int32, numYieldTypes)
	c.DestYields = make([]int32, numYieldTypes)
	for i := 0; i < numYieldTypes; i++ {
		c.OriginYields[i] = int32(unsafeReadUint32(reader))
		c.DestYields[i] = int32(unsafeReadUint32(reader))
	}

	return c
}

// GameTrade is every active trade route in the game, plus the running tech-difference matrix used
// to gate trade route yields.
type GameTrade struct {
	TradeConnections []TradeConnection
	TechDifference   [][]int32 // MAX_MAJOR_CIVS x MAX_MAJOR_CIVS (only if version >= 3, else all zero)
	NextID           int32
}

func readGameTrade(reader *io.SectionReader) GameTrade {
	version := unsafeReadUint32(reader)

	connectionCount := unsafeReadUint32(reader)
	connections := make([]TradeConnection, connectionCount)
	for i := range connections {
		connections[i] = readTradeConnection(reader, version)
	}

	techDifference := make([][]int32, maxMajorCivs)
	if version >= 3 {
		for i := range techDifference {
			techDifference[i] = unsafeReadFixedInt32Array(reader, maxMajorCivs)
		}
	} else {
		for i := range techDifference {
			techDifference[i] = make([]int32, maxMajorCivs)
		}
	}

	nextID := int32(unsafeReadUint32(reader))

	return GameTrade{TradeConnections: connections, TechDifference: techDifference, NextID: nextID}
}

// readEmbeddedDatabase reads the size-prefixed embedded SQLite database blob
// (Civ5SavedGameDatabase.db) - a uint32 byte count, then that many raw bytes. It's a fixed,
// byte-identical compiled-in schema/template, not live per-game data, so it's returned as raw
// bytes rather than parsed.
func readEmbeddedDatabase(reader *io.SectionReader) []byte {
	size := unsafeReadUint32(reader)
	return unsafeReadFixedBytes(reader, int(size))
}

// MapHeader is the fixed-size prefix before the per-tile data that makes up the true bulk of the map.
type MapHeader struct {
	Version              uint32
	GridWidth            int32
	GridHeight           int32
	LandPlotCount        int32
	OwnedPlotCount       int32
	NumNaturalWonders    int32
	TopLatitude          int32
	BottomLatitude       int32
	WrapX                bool
	WrapY                bool
	GUID                 []byte          // 16 bytes: Data1(4)+Data2(2)+Data3(2)+Data4(8), standard GUID layout
	ResourceCounts       []HashValuePair // total count per resource type across the whole map
	ResourceCountsOnLand []HashValuePair // count per resource type on land tiles only
}

func readMapHeader(reader *io.SectionReader) MapHeader {
	h := MapHeader{}
	h.Version = unsafeReadUint32(reader)
	h.GridWidth = int32(unsafeReadUint32(reader))
	h.GridHeight = int32(unsafeReadUint32(reader))
	h.LandPlotCount = int32(unsafeReadUint32(reader))
	h.OwnedPlotCount = int32(unsafeReadUint32(reader))
	h.NumNaturalWonders = int32(unsafeReadUint32(reader))
	h.TopLatitude = int32(unsafeReadUint32(reader))
	h.BottomLatitude = int32(unsafeReadUint32(reader))
	h.WrapX = unsafeReadByte(reader) != 0
	h.WrapY = unsafeReadByte(reader) != 0
	h.GUID = unsafeReadFixedBytes(reader, 16)

	resourceCount := unsafeReadUint32(reader)
	h.ResourceCounts = readHashValuePairs(reader, int(resourceCount))
	resourceOnLandCount := unsafeReadUint32(reader)
	h.ResourceCountsOnLand = readHashValuePairs(reader, int(resourceOnLandCount))

	return h
}

// Fixed-size per-plot array constants. reallyMaxPlayers/reallyMaxTeams are both 80 - a headroom
// constant, well above the real 64 players/22 major civs.
const (
	reallyMaxPlayers   = 80
	reallyMaxTeams     = 80
	numInvisibleTypes  = 1
	plotBoolFieldWords = 4 // a 128-bit (4x uint32) per-team revealed bitset
)

// idInfo is a (player, free-list index) pair identifying one city.
type idInfo struct {
	Owner int32
	ID    int32
}

func readIDInfo(reader *io.SectionReader) idInfo {
	return idInfo{Owner: int32(unsafeReadUint32(reader)), ID: int32(unsafeReadUint32(reader))}
}

// PlotArchaeologyData is the buried Great Work/artifact
// data (if any) recoverable from this plot via archaeology.
type PlotArchaeologyData struct {
	ArtifactType int32
	Era          int32
	Player1      int32
	Player2      int32
	Work         int32 // only if version >= 2 (always true in practice - writer always stamps version 2)
}

func readPlotArchaeologyData(reader *io.SectionReader) PlotArchaeologyData {
	version := unsafeReadUint32(reader)
	d := PlotArchaeologyData{
		ArtifactType: int32(unsafeReadUint32(reader)),
		Era:          int32(unsafeReadUint32(reader)),
		Player1:      int32(unsafeReadUint32(reader)),
		Player2:      int32(unsafeReadUint32(reader)),
	}
	if version >= 2 {
		d.Work = int32(unsafeReadUint32(reader))
	}
	return d
}

func civ5StringHash(s string) uint32 {
	table := crc32.MakeTable(crc32.IEEE)
	crc := uint32(0xFFFFFFFF)
	for i := 0; i < len(s); i++ {
		crc = table[byte(crc)^s[i]] ^ (crc >> 8)
	}
	return crc
}
