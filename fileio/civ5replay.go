package fileio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

type Civ5ReplayFileConfigEntry struct {
	VariableType string
	VariableName string
}

var worldSizeNames = []string{
	"WORLDSIZE_DUEL",
	"WORLDSIZE_TINY",
	"WORLDSIZE_SMALL",
	"WORLDSIZE_STANDARD",
	"WORLDSIZE_LARGE",
	"WORLDSIZE_HUGE",
}

var gameSpeedNames = []string{
	"GAMESPEED_MARATHON",
	"GAMESPEED_EPIC",
	"GAMESPEED_STANDARD",
	"GAMESPEED_QUICK",
}

var climateNames = []string{
	"CLIMATE_TEMPERATE",
	"CLIMATE_TROPICAL",
	"CLIMATE_ARID",
	"CLIMATE_ROCKY",
	"CLIMATE_COLD",
}

var seaLevelNames = []string{
	"SEALEVEL_LOW",
	"SEALEVEL_MEDIUM",
	"SEALEVEL_HIGH",
}

var eraNames = []string{
	"ERA_ANCIENT",
	"ERA_CLASSICAL",
	"ERA_MEDIEVAL",
	"ERA_RENAISSANCE",
	"ERA_INDUSTRIAL",
	"ERA_MODERN",
	"ERA_POSTMODERN",
	"ERA_FUTURE",
}

var victoryTypeNames = []string{
	"VICTORY_TIME",
	"VICTORY_SPACE_RACE",
	"VICTORY_DOMINATION",
	"VICTORY_CULTURAL",
	"VICTORY_DIPLOMATIC",
}

var handicapNames = []string{
	"HANDICAP_SETTLER",
	"HANDICAP_CHIEFTAIN",
	"HANDICAP_WARLORD",
	"HANDICAP_PRINCE",
	"HANDICAP_KING",
	"HANDICAP_EMPEROR",
	"HANDICAP_IMMORTAL",
	"HANDICAP_DEITY",
}

var plotTypeNames = []string{
	"PLOT_MOUNTAIN",
	"PLOT_HILLS",
	"PLOT_LAND",
	"PLOT_OCEAN",
}

var gameStateNames = []string{
	"GAMESTATE_ON",
	"GAMESTATE_OVER",
	"GAMESTATE_EXTENDED",
}

// tradeItemTypeNames is TradedItem.ItemType. TRADE_ITEM_NONE=-1 isn't included since typeName
// already returns "" for any negative index.
var tradeItemTypeNames = []string{
	"TRADE_ITEM_GOLD",
	"TRADE_ITEM_GOLD_PER_TURN",
	"TRADE_ITEM_MAPS",
	"TRADE_ITEM_RESOURCES",
	"TRADE_ITEM_CITIES",
	"TRADE_ITEM_UNITS",
	"TRADE_ITEM_OPEN_BORDERS",
	"TRADE_ITEM_DEFENSIVE_PACT",
	"TRADE_ITEM_RESEARCH_AGREEMENT",
	"TRADE_ITEM_TRADE_AGREEMENT", // not in use
	"TRADE_ITEM_PERMANENT_ALLIANCE",
	"TRADE_ITEM_SURRENDER",
	"TRADE_ITEM_TRUCE",
	"TRADE_ITEM_PEACE_TREATY",
	"TRADE_ITEM_THIRD_PARTY_PEACE",
	"TRADE_ITEM_THIRD_PARTY_WAR",
	"TRADE_ITEM_THIRD_PARTY_EMBARGO", // not in use
	"TRADE_ITEM_ALLOW_EMBASSY",
	"TRADE_ITEM_DECLARATION_OF_FRIENDSHIP", // only "traded" between human players
	"TRADE_ITEM_VOTE_COMMITMENT",
}

// resourceTypeNames is ResourceTypes, in the declaration order of the game's CIV5Resources.xml.
var resourceTypeNames = []string{
	"RESOURCE_IRON",
	"RESOURCE_HORSE",
	"RESOURCE_COAL",
	"RESOURCE_OIL",
	"RESOURCE_ALUMINUM",
	"RESOURCE_URANIUM",
	"RESOURCE_WHEAT",
	"RESOURCE_COW",
	"RESOURCE_SHEEP",
	"RESOURCE_DEER",
	"RESOURCE_BANANA",
	"RESOURCE_FISH",
	"RESOURCE_STONE",
	"RESOURCE_WHALE",
	"RESOURCE_PEARLS",
	"RESOURCE_GOLD",
	"RESOURCE_SILVER",
	"RESOURCE_GEMS",
	"RESOURCE_MARBLE",
	"RESOURCE_IVORY",
	"RESOURCE_FUR",
	"RESOURCE_DYE",
	"RESOURCE_SPICES",
	"RESOURCE_SILK",
	"RESOURCE_SUGAR",
	"RESOURCE_COTTON",
	"RESOURCE_WINE",
	"RESOURCE_INCENSE",
}

// greatWorkClassNames is GreatWork.ClassType (GreatWorkClass), 1-indexed - index 0 is unused.
var greatWorkClassNames = []string{
	"",
	"GREAT_WORK_ART",
	"GREAT_WORK_ARTIFACT",
	"GREAT_WORK_LITERATURE",
	"GREAT_WORK_MUSIC",
}

// greatWorkTypeNames is GreatWork.GWType (GreatWorkType), 1-indexed - index 0 is unused.
var greatWorkTypeNames = []string{
	"",
	"GREAT_WORK_OTHER_ERA_ANCIENT_RUIN",
	"GREAT_WORK_OTHER_ERA_BARBARIAN_CAMP",
	"GREAT_WORK_OTHER_ERA_BATTLE_MELEE_1",
	"GREAT_WORK_OTHER_ERA_BATTLE_MELEE_2",
	"GREAT_WORK_OTHER_ERA_BATTLE_RANGED",
	"GREAT_WORK_OTHER_ERA_RAZED_CITY",
	"GREAT_WORK_ANCIENT_ERA_ANCIENT_RUIN",
	"GREAT_WORK_ANCIENT_ERA_BARBARIAN_CAMP",
	"GREAT_WORK_ANCIENT_ERA_BATTLE_MELEE_1",
	"GREAT_WORK_ANCIENT_ERA_BATTLE_MELEE_2",
	"GREAT_WORK_ANCIENT_ERA_BATTLE_RANGED",
	"GREAT_WORK_ANCIENT_ERA_RAZED_CITY",
	"GREAT_WORK_CLASSICAL_ERA_ANCIENT_RUIN",
	"GREAT_WORK_CLASSICAL_ERA_BARBARIAN_CAMP",
	"GREAT_WORK_CLASSICAL_ERA_BATTLE_MELEE_1",
	"GREAT_WORK_CLASSICAL_ERA_BATTLE_MELEE_2",
	"GREAT_WORK_CLASSICAL_ERA_BATTLE_RANGED",
	"GREAT_WORK_CLASSICAL_ERA_RAZED_CITY",
	"GREAT_WORK_MEDIEVAL_ERA_ANCIENT_RUIN",
	"GREAT_WORK_MEDIEVAL_ERA_BARBARIAN_CAMP",
	"GREAT_WORK_MEDIEVAL_ERA_BATTLE_MELEE_1",
	"GREAT_WORK_MEDIEVAL_ERA_BATTLE_MELEE_2",
	"GREAT_WORK_MEDIEVAL_ERA_BATTLE_RANGED",
	"GREAT_WORK_MEDIEVAL_ERA_RAZED_CITY",
	"GREAT_WORK_RENAISSANCE_ERA_ANCIENT_RUIN",
	"GREAT_WORK_RENAISSANCE_ERA_BARBARIAN_CAMP",
	"GREAT_WORK_RENAISSANCE_ERA_BATTLE_MELEE_1",
	"GREAT_WORK_RENAISSANCE_ERA_BATTLE_MELEE_2",
	"GREAT_WORK_RENAISSANCE_ERA_BATTLE_RANGED",
	"GREAT_WORK_RENAISSANCE_ERA_RAZED_CITY",
	"GREAT_WORK_ODYSSEY",
	"GREAT_WORK_FABLES",
	"GREAT_WORK_OEDIPUS",
	"GREAT_WORK_LYSISTRATA",
	"GREAT_WORK_AENEID",
	"GREAT_WORK_METAMORPHOSES",
	"GREAT_WORK_TRUE_STORY",
	"GREAT_WORK_RUBAIYAT",
	"GREAT_WORK_MASNAVI",
	"GREAT_WORK_RAMAYAMA",
	"GREAT_WORK_MAHABHARATA",
	"GREAT_WORK_URUBHANGA",
	"GREAT_WORK_ABHIJNANASAKUTALAM",
	"GREAT_WORK_CHU_CI",
	"GREAT_WORK_SHUIHU_ZHUAN",
	"GREAT_WORK_JIN_PING_MEI",
	"GREAT_WORK_HONG_LOU_MENG",
	"GREAT_WORK_KOKIN_WAKASHU",
	"GREAT_WORK_GENJI_MONOGATARI",
	"GREAT_WORK_TSUREZUREGUSA",
	"GREAT_WORK_SHINJUTEN_NO_AMIJIMA",
	"GREAT_WORK_UGETSU_MONOGATARI",
	"GREAT_WORK_WAGAHAI_WA_NEKO",
	"GREAT_WORK_EL_INGENIOSO",
	"GREAT_WORK_MARIA",
	"GREAT_WORK_MARTIN_FIERRO",
	"GREAT_WORK_AZUL",
	"GREAT_WORK_GIRART_DE_VIENNE",
	"GREAT_WORK_LA_VIE_DE_GARGANTUA",
	"GREAT_WORK_CANDIDE_OU_LOPTIMISME",
	"GREAT_WORK_LES_TROIS_MOUSQUETAIRES",
	"GREAT_WORK_LES_MISERABLES",
	"GREAT_WORK_VIGNT_MILLE_LIEUES",
	"GREAT_WORK_A_LA_RECHERCHE",
	"GREAT_WORK_DIVINA_COMMEDIA",
	"GREAT_WORK_IL_CANZONIERE",
	"GREAT_WORK_ORLANDO_FURIOSO",
	"GREAT_WORK_SAUL",
	"GREAT_WORK_DIE_RAUBER",
	"GREAT_WORK_FAUST",
	"GREAT_WORK_REVIZOR",
	"GREAT_WORK_PRESTIPLENIYE",
	"GREAT_WORK_VOYNA_I_MIR",
	"GREAT_WORK_CHAYKA",
	"GREAT_WORK_NA_DNE",
	"GREAT_WORK_CANTERBURY_TALES",
	"GREAT_WORK_LE_MORTE_DARTHUR",
	"GREAT_WORK_THE_FAERIE_QUEENE",
	"GREAT_WORK_MACBETH",
	"GREAT_WORK_PRIDE_AND_PREJUDICE",
	"GREAT_WORK_FRANKENSTEIN",
	"GREAT_WORK_A_CHRISTMAS_CAROL",
	"GREAT_WORK_ALICES_ADVANTURES",
	"GREAT_WORK_BARRACK_ROOM_BALLADS",
	"GREAT_WORK_THE_TIME_MACHINE",
	"GREAT_WORK_THE_SIGN_OF_THE_FOUR",
	"GREAT_WORK_LADY_CHATTERLEYS_LOVER",
	"GREAT_WORK_THE_SKETCH_BOOK",
	"GREAT_WORK_THE_LAST_OF_THE_MOHICANS",
	"GREAT_WORK_THE_RAVEN",
	"GREAT_WORK_TWICE_TOLD_TALE",
	"GREAT_WORK_MOBY_DICK",
	"GREAT_WORK_UNCLE_TOMS_CABIN",
	"GREAT_WORK_WALDEN",
	"GREAT_WORK_LEAVES_OF_GRASS",
	"GREAT_WORK_ADVENTURES_OF_HUCKLEBERRY",
	"GREAT_WORK_POEMS",
	"GREAT_WORK_RED_BADGE_OF_COURAGE",
	"GREAT_WORK_WONDERFUL_WIZARD_OF_OZ",
	"GREAT_WORK_THE_GREAT_GATSBY",
	"GREAT_WORK_NIN_ME_SCHARA",
	"GREAT_WORK_HYMN_TO_THE_MUSE",
	"GREAT_WORK_GUANGLING_SAN",
	"GREAT_WORK_LAMMA_BADA_YATATHANNA",
	"GREAT_WORK_WHIRLING_DERVISH",
	"GREAT_WORK_UT_QUEANT_LAXIS",
	"GREAT_WORK_MESSE_DE_NOSTRE_DAME",
	"GREAT_WORK_NYMPHES_DES_BOIS",
	"GREAT_WORK_PAYOJI_MAINE",
	"GREAT_WORK_SPEM_IN_ALIUM",
	"GREAT_WORK_MISSA_PAPAE_MARCELLI",
	"GREAT_WORK_XICOCHI",
	"GREAT_WORK_ROKUDAN_NO_SHIRABE",
	"GREAT_WORK_BRANDENBERG_CONCERTOS",
	"GREAT_WORK_GLORIA_LAUS_ET_HONOR",
	"GREAT_WORK_SALVE_REGINA",
	"GREAT_WORK_EINE_KLEINE_NACHTMUSIC",
	"GREAT_WORK_SAKURAGARI",
	"GREAT_WORK_FIFTH_SYMPHONY",
	"GREAT_WORK_CAPRICE_NO_24",
	"GREAT_WORK_TE_DEUM",
	"GREAT_WORK_ETUDE_OP_10",
	"GREAT_WORK_DIE_WALKURE",
	"GREAT_WORK_LA_FORZA_DEL_DESTINO",
	"GREAT_WORK_CONTRADANZAS_CUBANAS",
	"GREAT_WORK_DIE_FLEDERMAUS",
	"GREAT_WORK_HUNGARIAN_DANCES",
	"GREAT_WORK_NIGHT_ON_BALD_MOUNTAIN",
	"GREAT_WORK_1812_OVERTURE",
	"GREAT_WORK_FROM_THE_NEW_WORLD",
	"GREAT_WORK_DAHA",
	"GREAT_WORK_CATHEDRALE_ENGLOUTIE",
	"GREAT_WORK_VARIATIONS_ON_AMERICA",
	"GREAT_WORK_LAU_DUANG_DUEN",
	"GREAT_WORK_ANA_HAWEET",
	"GREAT_WORK_SERMON_OF_SADNESS",
	"GREAT_WORK_I_GOT_RHYTHM",
	"GREAT_WORK_SENSEMAYA",
	"GREAT_WORK_SINFONIA_INDIA",
	"GREAT_WORK_LADRANG_SRI_DUHITO",
	"GREAT_WORK_IN_A_LANDSCAPE",
	"GREAT_WORK_JHAPTAL",
	"GREAT_WORK_DIDGERIDOO",
	"GREAT_WORK_KATCINA_DANCES",
	"GREAT_WORK_ELECTRIC_COUNTERPOINT",
	"GREAT_WORK_AL_CAPONE",
	"GREAT_WORK_MORNING_STAR",
	"GREAT_WORK_ALSO_SPRACH_ZARATHUSTRA",
	"GREAT_WORK_SATURN",
	"GREAT_WORK_CARAVAN",
	"GREAT_WORK_SYMPHONY_NO_1",
	"GREAT_WORK_MBUBE",
	"GREAT_WORK_SHAKER_LOOPS",
	"GREAT_WORK_AVE_MARIS",
	"GREAT_WORK_REQUIEM_KYRIE",
	"GREAT_WORK_SALVE_REGINA_A_4",
	"GREAT_WORK_BONJOUR_MON_COEUR",
	"GREAT_WORK_O_MAGNUM_MYSTERIUM",
	"GREAT_WORK_POICHE_LAVIDA",
	"GREAT_WORK_AGNUS_DEI",
	"GREAT_WORK_WATER_MUSIC",
	"GREAT_WORK_SYMPHONE_NO_93",
	"GREAT_WORK_HUNGARIAN_RHAPSODY",
	"GREAT_WORK_DAS_VERLASSENE",
	"GREAT_WORK_SYMPHONY_NO_5",
	"GREAT_WORK_SYMPHONY_NO_3",
	"GREAT_WORK_PARADE",
	"GREAT_WORK_LULU",
	"GREAT_WORK_STRING_QUARTET",
	"GREAT_WORK_YOUNG_PERSONS_GUIDE",
	"GREAT_WORK_GIRL_WITH_PEARL_EARRING",
	"GREAT_WORK_HENRY_VIII",
	"GREAT_WORK_HUNTERS_IN_THE_SNOW",
	"GREAT_WORK_LES_DEMOISELLES",
	"GREAT_WORK_LUNCHEON",
	"GREAT_WORK_MONA_LISA",
	"GREAT_WORK_SUNDAY_AFTERNOON",
	"GREAT_WORK_VIEW_OF_TOLEDO",
	"GREAT_WORK_WATER_LILIES",
	"GREAT_WORK_STARRY_NIGHT",
	"GREAT_WORK_BIRTH_OF_VENUS",
	"GREAT_WORK_BINDO_ALTOVITI",
	"GREAT_WORK_SAINT_GEORGE_DRAGON",
	"GREAT_WORK_CREATION_OF_ADAM",
	"GREAT_WORK_GALLERY_OF_THE_LOUVRE",
	"GREAT_WORK_WASHINGTON_CROSSING",
	"GREAT_WORK_ARRANGEMENT_IN_GREY",
	"GREAT_WORK_BREEZING_UP",
	"GREAT_WORK_A_THOUSAND_LI",
	"GREAT_WORK_DWELLING_IN_THE_FUCHUN",
	"GREAT_WORK_ALONG_THE_RIVER",
	"GREAT_WORK_LEAF_2",
	"GREAT_WORK_EMPEROR_TAIZONG",
	"GREAT_WORK_SPRING_MORNING",
	"GREAT_WORK_THE_GREAT_WAVE",
	"GREAT_WORK_EVENING_SHOWER",
	"GREAT_WORK_L_ABSINTHE",
	"GREAT_WORK_UPAUPA_SCHNEKLUD",
	"GREAT_WORK_A_BAR_AT_THE_FOLIES",
	"GREAT_WORK_BALL_AT_THE_MOULIN",
	"GREAT_WORK_THE_KISS",
	"GREAT_WORK_AMERICAN_GOTHIC",
	"GREAT_WORK_SRI_SHANMUKAHA",
	"GREAT_WORK_DAUD_RECEIVES_A_ROBE",
	"GREAT_WORK_YUSEF_AND_ZULEYKHA",
	"GREAT_WORK_THE_ASCENT_OF_MUHAMMAD",
	"GREAT_WORK_LIBERTY_LEADING_THE_PEOPLE",
	"GREAT_WORK_SAINT_JEROME_IN_HIS_STUDY",
	"GREAT_WORK_PARTHENON_FRIEZE",
	"GREAT_WORK_RUE_DE_PARIS",
	"GREAT_WORK_NIGHT_WATCH",
	"GREAT_WORK_DEATH_OF_MARAT",
	"GREAT_WORK_ETIENNE_CHEVALIER",
	"GREAT_WORK_SAINT_FRANCIS",
	"GREAT_WORK_HATAORI",
	"GREAT_WORK_JANE_GREY",
	"GREAT_WORK_WANDERER",
	"GREAT_WORK_BABEL",
	"GREAT_WORK_YOUNG_LADY",
	"GREAT_WORK_KINDRED",
	"GREAT_WORK_BLUE_BOY",
	"GREAT_WORK_SECOND_OF_MAY",
	"GREAT_WORK_LAST_CART",
	"GREAT_WORK_EARLY_AUTUMN",
	"GREAT_WORK_APPRECIATING_LOTUSES",
	"GREAT_WORK_EQUESTRIAN_PORTRAIT",
	"GREAT_WORK_THE_ARMADA",
	"GREAT_WORK_FRANCIS_I",
	"GREAT_WORK_SELF_PORTRAIT",
	"GREAT_WORK_VERTUMNUS",
	"GREAT_WORK_LAST_SUPPER",
	"GREAT_WORK_PORTRAIT_OF_A_MAN",
	"GREAT_WORK_FRONT_OF_THE_MIRROR",
	"GREAT_WORK_ADORATION",
	"GREAT_WORK_RUBENS_AND_ISABELLA",
	"GREAT_WORK_CHILDS_BATH",
	"GREAT_WORK_THE_CARD_PLAYERS",
	"GREAT_WORK_THE_SWING",
	"GREAT_WORK_GIANT_MAGNOLIAS",
	"GREAT_WORK_SAINT_GEORGE_KILLING",
	"GREAT_WORK_JACQUES_AND_BERTHE",
	"GREAT_WORK_DASH_FOR_TIMBER",
	"GREAT_WORK_GEORGE_WASHINGTON_PORTRAIT",
	"GREAT_WORK_ATTACK_ON_BEIJING",
	"GREAT_WORK_SUMO_WRESTLER",
	"GREAT_WORK_ROSETTA",
	"GREAT_WORK_MAWANGDUI",
	"GREAT_WORK_DEAD_SEA",
	"GREAT_WORK_GILGAMESH",
	"GREAT_WORK_NAG_HAMMADI",
	"GREAT_WORK_LINEAR_B",
	"GREAT_WORK_PAPYRUS_ANI",
	"GREAT_WORK_RUNESTONE",
	"GREAT_WORK_HAMMURABI",
	"GREAT_WORK_BAKHSHALI",
	"GREAT_WORK_CODEX_PEREZ",
	"GREAT_WORK_JUSHICHIJO_KEMPO",
	"GREAT_WORK_KELLS",
	"GREAT_WORK_CODEX_BORGIA",
	"GREAT_WORK_KEDUKAN_BUKRIT",
	"GREAT_WORK_TIGER_IN_TROPICAL_STORM",
	"GREAT_WORK_THE_HOME_GUARD",
	"GREAT_WORK_CEREMONIAL_SITTING",
	"GREAT_WORK_RUSSIAN_TSAR",
	"GREAT_WORK_CRUCIFIXION_OF_SAINT_PETER",
	"GREAT_WORK_LANDSCAPE_WITH_ANT_BEAR",
	"GREAT_WORK_DUTCH_MEN_O_WAR",
	"GREAT_WORK_HARBOR_IN_BALAKLAVA",
	"GREAT_WORK_PEACEMAKERS",
	"GREAT_WORK_PORTRAIT_OF_CHITASEI",
	"GREAT_WORK_PORTRAIT_OF_LOUIS",
	"GREAT_WORK_PORTRAIT_OF_NASSER",
	"GREAT_WORK_FIELD_WITH_SHEAFS",
	"GREAT_WORK_AT_THE_EDGE_OF_THE_BROOK",
	"GREAT_WORK_THE_KISS_ALT",
	"GREAT_WORK_MOVING_HOUSE",
	"GREAT_WORK_MEXICAN_EXPEDITION",
	"GREAT_WORK_LAST_DAY_OF_POMPEII",
	"GREAT_WORK_THE_ACCOLADE",
}

// domainTypeNames is DomainTypes (TradeConnection.Domain), 0-indexed.
var domainTypeNames = []string{
	"DOMAIN_SEA",
	"DOMAIN_AIR",
	"DOMAIN_LAND",
	"DOMAIN_IMMOBILE",
	"DOMAIN_HOVER",
}

// tradeConnectionTypeNames is TradeConnectionType (TradeConnection.ConnectionType), 0-indexed.
var tradeConnectionTypeNames = []string{
	"TRADE_CONNECTION_INTERNATIONAL",
	"TRADE_CONNECTION_FOOD",
	"TRADE_CONNECTION_PRODUCTION",
}

var gameOptionNames = []string{
	"GAMEOPTION_NO_CITY_RAZING",
	"GAMEOPTION_NO_BARBARIANS",
	"GAMEOPTION_RAGING_BARBARIANS",
	"GAMEOPTION_ALWAYS_WAR",
	"GAMEOPTION_ALWAYS_PEACE",
	"GAMEOPTION_ONE_CITY_CHALLENGE",
	"GAMEOPTION_NO_CHANGING_WAR_PEACE",
	"GAMEOPTION_NEW_RANDOM_SEED",
	"GAMEOPTION_LOCK_MODS",
	"GAMEOPTION_COMPLETE_KILLS",
	"GAMEOPTION_NO_GOODY_HUTS",
	"GAMEOPTION_RANDOM_PERSONALITIES",
	"GAMEOPTION_POLICY_SAVING",
	"GAMEOPTION_PROMOTION_SAVING",
	"GAMEOPTION_END_TURN_TIMER_ENABLED",
	"GAMEOPTION_QUICK_COMBAT",
	"GAMEOPTION_DISABLE_START_BIAS",
	"GAMEOPTION_NO_SCIENCE",
	"GAMEOPTION_NO_POLICIES",
	"GAMEOPTION_NO_HAPPINESS",
	"GAMEOPTION_NO_TUTORIAL",
	"GAMEOPTION_NO_RELIGION",
}

type Civ5ReplayCiv struct {
	CivilizationIndex int
	LeaderTypeIndex   int
	PlayerColorIndex  int
	Difficulty        string
	Leader            string
	LongName          string
	Name              string
	Demonym           string
}

type Civ5ReplayEventTile struct {
	X int
	Y int
}

type Civ5ReplayEvent struct {
	Turn   int
	TypeId int
	Tiles  []Civ5ReplayEventTile
	CivId  int
	Text   string
}

type Civ5ReplayCivDataset struct {
	CivIndex      int
	DatasetValues map[string][]Civ5ReplayDataEntry
}

type Civ5ReplayDataEntry struct {
	Turn  int
	Value int
}

// Civ5ReplayTile is one tile's physical-geography snapshot. PlotType is hardcoded:
// 0=PLOT_MOUNTAIN, 1=PLOT_HILLS, 2=PLOT_LAND, 3=PLOT_OCEAN.
type Civ5ReplayTile struct {
	PlotType    int8
	TerrainType int8
	Feature     int8
	RiverBits   int8
}

type Civ5ReplayData struct {
	PlayerCiv       string
	IsReplayFile    bool
	AllCivs         []Civ5ReplayCiv
	AllReplayEvents []Civ5ReplayEvent
	DatasetNames    []string
	DatasetValues   []Civ5ReplayCivDataset
	MapFileStem string // empty when unknown (e.g. converted from a .civ5save)
	// MapWidth/MapHeight are the map's dimensions, as embedded in the .civ5replay. 0 when unknown.
	MapWidth  int
	MapHeight int
	// Tiles is the per-plot geography snapshot, row-major: index = y*MapWidth+x. Empty when
	// MapWidth/MapHeight are 0.
	Tiles []Civ5ReplayTile

	WorldSize       string
	Climate         string
	SeaLevel        string
	GameSpeed       string
	Era             string
	VictoryTypes    []string
	VictoryAchieved string
	GameOptions     []string
}

// mapFilenameStem extracts the map name from a full file path, removing the directory and extension
func mapFilenameStem(rawPath string) string {
	normalized := strings.ReplaceAll(rawPath, `\`, "/")
	base := path.Base(normalized)
	return strings.TrimSuffix(base, path.Ext(base))
}

func readCivs(reader *io.SectionReader) []Civ5ReplayCiv {
	civsLength := unsafeReadUint32(reader)
	allCivs := make([]Civ5ReplayCiv, 0)

	for i := 0; i < int(civsLength); i++ {
		civilizationIndex := unsafeReadUint32(reader)
		leaderTypeIndex := unsafeReadUint32(reader)
		playerColorIndex := unsafeReadUint32(reader)
		difficultyIndex := unsafeReadUint32(reader)
		leader, err := readVarString(reader, "leader")
		if err != nil {
			panic(fmt.Sprintf("failed to read leader: %v", err))
		}
		longName, err := readVarString(reader, "longName")
		if err != nil {
			panic(fmt.Sprintf("failed to read longName: %v", err))
		}
		name, err := readVarString(reader, "name")
		if err != nil {
			panic(fmt.Sprintf("failed to read name: %v", err))
		}
		demonym, err := readVarString(reader, "demonym")
		if err != nil {
			panic(fmt.Sprintf("failed to read demonym: %v", err))
		}

		civData := Civ5ReplayCiv{
			CivilizationIndex: int(civilizationIndex),
			LeaderTypeIndex:   int(leaderTypeIndex),
			PlayerColorIndex:  int(playerColorIndex),
			Difficulty:        typeName(handicapNames, int(difficultyIndex)),
			Leader:            leader,
			LongName:          longName,
			Name:              name,
			Demonym:           demonym,
		}
		allCivs = append(allCivs, civData)
	}

	return allCivs
}

func readEvents(reader *io.SectionReader) []Civ5ReplayEvent {
	eventsLength := unsafeReadUint32(reader)
	allReplayEvents := make([]Civ5ReplayEvent, eventsLength)

	for i := 0; i < int(eventsLength); i++ {
		turn := unsafeReadUint32(reader)
		typeId := unsafeReadUint32(reader)

		numTiles := unsafeReadUint32(reader)
		tileData := make([]Civ5ReplayEventTile, numTiles)
		for i := 0; i < int(numTiles); i++ {
			tileX := unsafeReadUint16(reader)
			tileY := unsafeReadUint16(reader)

			tileData[i] = Civ5ReplayEventTile{
				X: int(tileX),
				Y: int(tileY),
			}
		}

		civId := int32(unsafeReadUint32(reader))
		eventText, err := readVarString(reader, "eventText")
		if err != nil {
			panic(fmt.Sprintf("failed to read eventText: %v", err))
		}

		allReplayEvents[i] = Civ5ReplayEvent{
			Turn:   int(turn),
			TypeId: int(typeId),
			Tiles:  tileData,
			CivId:  int(civId),
			Text:   eventText,
		}
	}

	return allReplayEvents
}

func GroupEventsByTurn(replayEvents []Civ5ReplayEvent) map[int][]Civ5ReplayEvent {
	replayTurns := make(map[int][]Civ5ReplayEvent)

	for i := 0; i < len(replayEvents); i++ {
		turn := replayEvents[i].Turn
		_, ok := replayTurns[turn]
		if !ok {
			replayTurns[turn] = make([]Civ5ReplayEvent, 0)
		}
		replayTurns[turn] = append(replayTurns[turn], replayEvents[i])
	}
	return replayTurns
}

func readDatasetNames(streamReader *io.SectionReader) []string {
	datasetLength := unsafeReadUint32(streamReader)
	datasetNames := make([]string, int(datasetLength))
	for i := 0; i < int(datasetLength); i++ {
		name, err := readVarString(streamReader, "datasetNames")
		if err != nil {
			panic(fmt.Sprintf("failed to read dataset name %d: %v", i, err))
		}
		datasetNames[i] = name
	}
	return datasetNames
}

func readDatasetValues(streamReader *io.SectionReader) [][][]Civ5ReplayDataEntry {
	datasetValuesArray1Length := unsafeReadUint32(streamReader)
	datasetByCiv := make([][][]Civ5ReplayDataEntry, int(datasetValuesArray1Length))

	for i := 0; i < int(datasetValuesArray1Length); i++ {
		datasetValuesArray2Length := unsafeReadUint32(streamReader)
		datasetByCategory := make([][]Civ5ReplayDataEntry, int(datasetValuesArray2Length))

		for j := 0; j < int(datasetValuesArray2Length); j++ {
			numDatasetValues := unsafeReadUint32(streamReader)

			datasetArray := make([]Civ5ReplayDataEntry, numDatasetValues)
			for k := 0; k < int(numDatasetValues); k++ {
				turn := unsafeReadUint32(streamReader)
				value := unsafeReadUint32(streamReader)
				datasetArray[k] = Civ5ReplayDataEntry{
					Turn:  int(turn),
					Value: int(value),
				}
			}

			datasetByCategory[j] = datasetArray
		}
		datasetByCiv[i] = datasetByCategory
	}
	return datasetByCiv
}

func buildCivDatasetValues(streamReader *io.SectionReader, datasetNames []string) []Civ5ReplayCivDataset {
	datasetValues := readDatasetValues(streamReader)

	allCivDatasetValues := make([]Civ5ReplayCivDataset, len(datasetValues))
	for civIndex := 0; civIndex < len(datasetValues); civIndex++ {
		dataMap := make(map[string][]Civ5ReplayDataEntry, 0)
		for datasetNameIndex := 0; datasetNameIndex < len(datasetNames); datasetNameIndex++ {
			datasetName := datasetNames[datasetNameIndex]
			dataMap[datasetName] = datasetValues[civIndex][datasetNameIndex]
		}

		allCivDatasetValues[civIndex] = Civ5ReplayCivDataset{
			CivIndex:      civIndex,
			DatasetValues: dataMap,
		}
	}
	return allCivDatasetValues
}

func ReadCiv5ReplayFile(filename string) (*Civ5ReplayData, error) {
	inputFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load replay file %q: %w", filename, err)
	}
	defer inputFile.Close()

	fi, err := inputFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %q: %w", filename, err)
	}
	fileLength := fi.Size()
	streamReader := io.NewSectionReader(inputFile, int64(0), fileLength)
	fmt.Println("Loading Civ5Replay...")

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:4",
			VariableName: "gameName",
		},
		{
			VariableType: "uint32",
			VariableName: "unknownUint1",
		},
		{
			VariableType: "varstring",
			VariableName: "gameVersion",
		},
		{
			VariableType: "varstring",
			VariableName: "gameBuild",
		},
		{
			VariableType: "uint32",
			VariableName: "currentTurnNumber",
		},
		{
			VariableType: "bytearray:1",
			VariableName: "unknownByte1",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read initial file config: %w", err)
	}

	playerCiv, err := readVarString(streamReader, "playerCiv")
	if err != nil {
		return nil, fmt.Errorf("failed to read player civ: %w", err)
	}

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "difficulty",
		},
		{
			VariableType: "varstring",
			VariableName: "eraStart",
		},
		{
			VariableType: "varstring",
			VariableName: "eraEnd",
		},
		{
			VariableType: "varstring",
			VariableName: "gameSpeed",
		},
		{
			VariableType: "varstring",
			VariableName: "worldSize",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read difficulty/era/gameSpeed/worldSize config: %w", err)
	}

	mapFilename, err := readVarString(streamReader, "mapFilename")
	if err != nil {
		return nil, fmt.Errorf("failed to read mapFilename: %w", err)
	}

	readArray(streamReader, "dlc", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "bytearray:16",
			VariableName: "dlcId",
		},
		{
			VariableType: "bytearray:4",
			VariableName: "dlcEnabled",
		},
		{
			VariableType: "varstring",
			VariableName: "dlcName",
		},
	})

	readArray(streamReader, "mods", []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "modId",
		},
		{
			VariableType: "bytearray:4",
			VariableName: "modVersion",
		},
		{
			VariableType: "varstring",
			VariableName: "modName",
		},
	})

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "varstring",
			VariableName: "civName",
		},
		{
			VariableType: "varstring",
			VariableName: "leaderName",
		},
		{
			VariableType: "varstring",
			VariableName: "playerColor",
		},
		{
			VariableType: "uint32",
			VariableName: "replayVersion",
		},
		{
			VariableType: "uint32",
			VariableName: "activePlayerIndex",
		},
		{
			VariableType: "varstring",
			VariableName: "mapFilename2",
		},
	})

	worldSizeIndex := unsafeReadUint32(streamReader)
	worldSizeName := typeName(worldSizeNames, int(worldSizeIndex))
	fmt.Println("World size:", worldSizeName)

	climateIndex := unsafeReadUint32(streamReader)
	seaLevelIndex := unsafeReadUint32(streamReader)
	eraIndex := unsafeReadUint32(streamReader)
	gameSpeedIndex := unsafeReadUint32(streamReader)
	climateName := typeName(climateNames, int(climateIndex))
	seaLevelName := typeName(seaLevelNames, int(seaLevelIndex))
	eraName := typeName(eraNames, int(eraIndex))
	gameSpeedName := typeName(gameSpeedNames, int(gameSpeedIndex))
	fmt.Println("Climate:", climateName)
	fmt.Println("Sea level:", seaLevelName)
	fmt.Println("Era:", eraName)
	fmt.Println("Game speed:", gameSpeedName)

	gameOptionCount := unsafeReadUint32(streamReader)
	gameOptions := make([]string, 0, gameOptionCount)
	for i := 0; i < int(gameOptionCount); i++ {
		value := unsafeReadUint32(streamReader)
		gameOptions = append(gameOptions, typeName(gameOptionNames, int(value)))
	}
	fmt.Println("Game options enabled:", gameOptions)

	victoryTypeCount := unsafeReadUint32(streamReader)
	victoryTypes := make([]string, 0, victoryTypeCount)
	for i := 0; i < int(victoryTypeCount); i++ {
		value := unsafeReadUint32(streamReader)
		victoryTypes = append(victoryTypes, typeName(victoryTypeNames, int(value)))
	}
	fmt.Println("Victory types enabled:", victoryTypes)

	victoryTypeIndex := int32(unsafeReadUint32(streamReader))
	victoryTypeName := typeName(victoryTypeNames, int(victoryTypeIndex))
	fmt.Println("Victory type (index):", victoryTypeIndex, "name:", victoryTypeName)

	unknownByte2 := [1]byte{}
	if err := binary.Read(streamReader, binary.LittleEndian, &unknownByte2); err != nil {
		return nil, fmt.Errorf("failed to read block: %w", err)
	}

	_, err = readFileConfig(streamReader, []Civ5ReplayFileConfigEntry{
		{
			VariableType: "uint32",
			VariableName: "startTurn",
		},
		{
			VariableType: "int32", // startYear can be negative, e.g. 4000 BC
			VariableName: "startYear",
		},
		{
			VariableType: "uint32",
			VariableName: "endTurn",
		},
		{
			VariableType: "varstring",
			VariableName: "endYear",
		},
		{
			VariableType: "uint32",
			VariableName: "zeroStartYear",
		},
		{
			VariableType: "uint32",
			VariableName: "zeroEndYear",
		},
	})

	allCivs := readCivs(streamReader)

	datasetNames := readDatasetNames(streamReader)
	datasetValues := buildCivDatasetValues(streamReader, datasetNames)

	// Read unknown value
	_ = unsafeReadUint32(streamReader)

	allReplayEvents := readEvents(streamReader)

	mapWidth := unsafeReadUint32(streamReader)
	mapHeight := unsafeReadUint32(streamReader)
	fmt.Println("Map width:", mapWidth, ", height:", mapHeight)

	tileCount := unsafeReadUint32(streamReader)
	tiles := make([]Civ5ReplayTile, tileCount)
	for i := range tiles {
		unsafeReadUint32(streamReader) // mapEntryCount, always 1
		unsafeReadUint32(streamReader) // turnKey, always equals this file's own "End turn"
		tiles[i] = Civ5ReplayTile{
			PlotType:    int8(unsafeReadByte(streamReader)),
			TerrainType: int8(unsafeReadByte(streamReader)),
			Feature:     int8(unsafeReadByte(streamReader)),
			RiverBits:   int8(unsafeReadByte(streamReader)),
		}
	}

	replayData := Civ5ReplayData{
		PlayerCiv:       playerCiv,
		IsReplayFile:    true,
		AllCivs:         allCivs,
		AllReplayEvents: allReplayEvents,
		DatasetNames:    datasetNames,
		DatasetValues:   datasetValues,
		MapFileStem:     mapFilenameStem(mapFilename),
		MapWidth:        int(mapWidth),
		MapHeight:       int(mapHeight),
		Tiles:           tiles,
		WorldSize:       worldSizeName,
		Climate:         climateName,
		SeaLevel:        seaLevelName,
		GameSpeed:       gameSpeedName,
		Era:             eraName,
		VictoryTypes:    victoryTypes,
		GameOptions:     gameOptions,
		VictoryAchieved: victoryTypeName,
	}

	return &replayData, nil
}
