package fileio

import (
	"fmt"
	"strings"
)

// civName resolves a player index against the save's own civ roster. Returns "" for -1
// (NO_PLAYER) or an out-of-range index.
func civName(allCivs []Civ5ReplayCiv, player int32) string {
	if player < 0 || int(player) >= len(allCivs) {
		return ""
	}
	return allCivs[player].Name
}

// cityFoundedEventTypeId is the replay event TypeId for "X is founded." events.
const cityFoundedEventTypeId = 1
const cityFoundedSuffix = " is founded."

// knownCityLocations maps every coordinate where a city-founding event was recorded in the
// save's own embedded replay events to that city's name. A city keeps its founding name/location
// even after being captured. Only covers cities founded within the events this save recorded.
func knownCityLocations(events []Civ5ReplayEvent) map[[2]int]string {
	locations := make(map[[2]int]string)
	for _, e := range events {
		if e.TypeId != cityFoundedEventTypeId || !strings.HasSuffix(e.Text, cityFoundedSuffix) {
			continue
		}
		cityName := strings.TrimSuffix(e.Text, cityFoundedSuffix)
		for _, t := range e.Tiles {
			locations[[2]int{t.X, t.Y}] = cityName
		}
	}
	return locations
}

// cityLabel formats a coordinate's known city name for display, "" if unknown.
func cityLabel(cityNames map[[2]int]string, x, y int32) string {
	if name, ok := cityNames[[2]int{int(x), int(y)}]; ok {
		return "(" + name + ")"
	}
	return ""
}

// maxPlotPathTilesShown caps how many tiles plotPath prints in full before switching to a
// first/last summary.
const maxPlotPathTilesShown = 50

// formatPlotTiles renders a slice of tiles as "(x,y) (x,y) ...", space-separated, no brackets.
func formatPlotTiles(plots []TradeConnectionPlot) string {
	s := ""
	for i, p := range plots {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("(%d,%d)", p.X, p.Y)
	}
	return s
}

// plotPath formats a trade connection's plot list as its tile-by-tile path, e.g. "[(1,2) (1,3)]".
// Beyond maxPlotPathTilesShown tiles, only the first/last 3 are shown.
func plotPath(plots []TradeConnectionPlot) string {
	if len(plots) == 0 {
		return "[]"
	}
	if len(plots) > maxPlotPathTilesShown {
		return fmt.Sprintf("[%s ... %s]", formatPlotTiles(plots[:3]), formatPlotTiles(plots[len(plots)-3:]))
	}
	return "[" + formatPlotTiles(plots) + "]"
}

// tradedItemPayload describes a TradedItem's Data1/Data2/Data3 meaning for known item types:
//
//	TRADE_ITEM_GOLD/GOLD_PER_TURN: amount
//	TRADE_ITEM_RESOURCES: ResourceTypes index, amount
//	TRADE_ITEM_CITIES: the city's own X/Y tile coordinates, not an ID
//	TRADE_ITEM_THIRD_PARTY_PEACE/THIRD_PARTY_WAR: a TeamTypes value, not a player index
//	TRADE_ITEM_VOTE_COMMITMENT: ResolutionID, VoteChoice, NumVotes, isRepeal
//
// Every other item type falls back to the generic data1/data2/data3 dump.
func tradedItemPayload(ti TradedItem) string {
	switch ti.ItemType {
	case 0, 1: // TRADE_ITEM_GOLD, TRADE_ITEM_GOLD_PER_TURN
		return fmt.Sprintf("amount=%d", ti.Data1)
	case 3: // TRADE_ITEM_RESOURCES
		return fmt.Sprintf("resourceType=%d(%s) amount=%d", ti.Data1, typeName(resourceTypeNames, int(ti.Data1)), ti.Data2)
	case 4: // TRADE_ITEM_CITIES
		return fmt.Sprintf("cityX=%d cityY=%d", ti.Data1, ti.Data2)
	case 14, 15: // TRADE_ITEM_THIRD_PARTY_PEACE, TRADE_ITEM_THIRD_PARTY_WAR
		return fmt.Sprintf("thirdPartyTeam=%d", ti.Data1)
	case 19: // TRADE_ITEM_VOTE_COMMITMENT
		return fmt.Sprintf("resolutionID=%d voteChoice=%d numVotes=%d isRepeal=%v", ti.Data1, ti.Data2, ti.Data3, ti.Flag1)
	default:
		if ti.Data1 == 0 && ti.Data2 == 0 && ti.Data3 == 0 {
			return ""
		}
		return fmt.Sprintf("data1=%d data2=%d data3=%d", ti.Data1, ti.Data2, ti.Data3)
	}
}

func printDealDetail(allCivs []Civ5ReplayCiv, label string, i int, d Deal) {
	fmt.Printf("  %s[%d]: from=%d(%s) to=%d(%s) items=%d\n",
		label, i, d.FromPlayer, civName(allCivs, d.FromPlayer), d.ToPlayer, civName(allCivs, d.ToPlayer), len(d.TradedItems))
	fmt.Printf("    turns: start=%d final=%d duration=%d\n", d.StartTurn, d.FinalTurn, d.Duration)
	fmt.Printf("    renewal: considering=%v checked=%v cancelled=%v\n", d.ConsideringForRenewal, d.CheckedForRenewal, d.DealCancelled)

	// Default to -1/NO_PLAYER when not applicable.
	var situational []string
	if d.PeaceTreatyType != -1 {
		situational = append(situational, fmt.Sprintf("peaceTreatyType=%d", d.PeaceTreatyType))
	}
	if d.SurrenderingPlayer != -1 {
		situational = append(situational, fmt.Sprintf("surrenderingPlayer=%d(%s)", d.SurrenderingPlayer, civName(allCivs, d.SurrenderingPlayer)))
	}
	if d.DemandingPlayer != -1 {
		situational = append(situational, fmt.Sprintf("demandingPlayer=%d(%s)", d.DemandingPlayer, civName(allCivs, d.DemandingPlayer)))
	}
	if d.RequestingPlayer != -1 {
		situational = append(situational, fmt.Sprintf("requestingPlayer=%d(%s)", d.RequestingPlayer, civName(allCivs, d.RequestingPlayer)))
	}
	if len(situational) > 0 {
		fmt.Println("    " + strings.Join(situational, " "))
	}

	for j, ti := range d.TradedItems {
		// Duration/finalTurn per item don't always match the deal's own duration/finalTurn.
		line := fmt.Sprintf("    item[%d]: itemType=%d(%s) fromPlayer=%d(%s) duration=%d finalTurn=%d",
			j, ti.ItemType, typeName(tradeItemTypeNames, int(ti.ItemType)), ti.FromPlayer, civName(allCivs, ti.FromPlayer), ti.Duration, ti.FinalTurn)
		if payload := tradedItemPayload(ti); payload != "" {
			line += " " + payload
		}
		// fromRenewed/toRenewed default to false; set only during deal-renewal matching.
		if ti.FromRenewed || ti.ToRenewed {
			line += fmt.Sprintf(" fromRenewed=%v toRenewed=%v", ti.FromRenewed, ti.ToRenewed)
		}
		fmt.Println(line)
	}
}

// printReplayEvents prints the full decoded replay event list, one line per event.
func printReplayEvents(events []Civ5ReplayEvent) {
	fmt.Printf("Read %d replay events\n", len(events))
	for i, e := range events {
		fmt.Printf("  replayEvent[%d]: turn=%d type=%d civ=%d tiles=%v text=%q\n", i, e.Turn, e.TypeId, e.CivId, e.Tiles, e.Text)
	}
}

// printGameDeals prints GameDeals' three deal categories via printDealDetail.
func printGameDeals(allCivs []Civ5ReplayCiv, gameDeals GameDeals) {
	fmt.Printf("gameDeals: %d proposed, %d current, %d historical\n", len(gameDeals.ProposedDeals), len(gameDeals.CurrentDeals), len(gameDeals.HistoricalDeals))
	printList := func(label string, deals []Deal) {
		for i, d := range deals {
			printDealDetail(allCivs, label, i, d)
		}
	}
	printList("proposedDeals", gameDeals.ProposedDeals)
	printList("currentDeals", gameDeals.CurrentDeals)
	printList("historicalDeals", gameDeals.HistoricalDeals)
}

// printGameReligions prints every current religion, pantheons included.
func printGameReligions(gameReligions GameReligions) {
	fmt.Printf("gameReligions: minFaithNextPantheon=%d, %d religions\n", gameReligions.MinimumFaithNextPantheon, len(gameReligions.CurrentReligions))
	for i, r := range gameReligions.CurrentReligions {
		fmt.Printf("  religion[%d]: type=%d founder=%d holyCity=(%d,%d) turnFounded=%d pantheon=%v enhanced=%v customName=%q\n",
			i, r.ReligionType, r.Founder, r.HolyCityX, r.HolyCityY, r.TurnFounded, r.Pantheon, r.Enhanced, r.CustomName)
	}
}

// printGameCulture prints every current great work, resolving its player index to a civ name.
func printGameCulture(allCivs []Civ5ReplayCiv, gameCulture GameCulture) {
	fmt.Printf("gameCulture: %d great works, reportedSomeoneInfluential=%v\n", len(gameCulture.CurrentGreatWorks), gameCulture.ReportedSomeoneInfluential)
	for i, gw := range gameCulture.CurrentGreatWorks {
		fmt.Printf("  greatWork[%d]: name=%q type=%d(%s) class=%d(%s) turnFounded=%d era=%d player=%d(%s)\n",
			i, gw.GreatPersonName, gw.GWType, typeName(greatWorkTypeNames, int(gw.GWType)), gw.ClassType, typeName(greatWorkClassNames, int(gw.ClassType)), gw.TurnFounded, gw.Era, gw.Player, civName(allCivs, gw.Player))
	}
}

// printGameLeagues prints every active league as a multi-line block plus its members.
func printGameLeagues(gameLeagues GameLeagues) {
	fmt.Printf("gameLeagues: %d active leagues, numLeaguesEverFounded=%d, diplomaticVictor=%d\n", len(gameLeagues.ActiveLeagues), gameLeagues.NumLeaguesEverFounded, gameLeagues.DiplomaticVictor)
	for i, l := range gameLeagues.ActiveLeagues {
		fmt.Printf("  league[%d]: id=%d customName=%q host=%d\n", i, l.ID, l.CustomName, l.Host)
		fmt.Printf("    status: inSession=%v turnsUntilSession=%d numResolutionsEverEnacted=%d\n", l.InSession, l.TurnsUntilSession, l.NumResolutionsEverEnacted)
		fmt.Printf("    counts: members=%d enactProposals=%d repealProposals=%d activeResolutions=%d\n", len(l.Members), len(l.EnactProposals), len(l.RepealProposals), len(l.ActiveResolutions))
		for j, m := range l.Members {
			fmt.Printf("    member[%d]: player=%d votes=%d abstainedVotes=%d mayPropose=%v everBeenHost=%v voteSources=%q\n",
				j, m.Player, m.Votes, m.AbstainedVotes, m.MayPropose, m.EverBeenHost, m.VoteSources)
		}
	}
}

// printGameTrade prints every real trade connection as a multi-line block, resolving owners to
// civ names and origin/dest tiles to known city names from allReplayEvents.
func printGameTrade(allCivs []Civ5ReplayCiv, allReplayEvents []Civ5ReplayEvent, gameTrade GameTrade) {
	cityNames := knownCityLocations(allReplayEvents)
	realConnectionCount := 0
	for _, tc := range gameTrade.TradeConnections {
		if tc.ID >= 0 {
			realConnectionCount++
		}
	}
	fmt.Printf("gameTrade: %d trade connections (%d real, %d empty slots), nextID=%d\n",
		len(gameTrade.TradeConnections), realConnectionCount, len(gameTrade.TradeConnections)-realConnectionCount, gameTrade.NextID)
	for i, tc := range gameTrade.TradeConnections {
		// TradeConnections is fixed-size; an empty slot has ID=-1.
		if tc.ID < 0 {
			continue
		}
		fmt.Printf("  tradeConnection[%d]: id=%d domain=%d(%s) connType=%d(%s)\n",
			i, tc.ID, tc.Domain, typeName(domainTypeNames, int(tc.Domain)), tc.ConnectionType, typeName(tradeConnectionTypeNames, int(tc.ConnectionType)))
		fmt.Printf("    origin=(%d,%d)%s owner=%d(%s)\n", tc.OriginX, tc.OriginY, cityLabel(cityNames, tc.OriginX, tc.OriginY), tc.OriginOwner, civName(allCivs, tc.OriginOwner))
		fmt.Printf("    dest=(%d,%d)%s owner=%d(%s)\n", tc.DestX, tc.DestY, cityLabel(cityNames, tc.DestX, tc.DestY), tc.DestOwner, civName(allCivs, tc.DestOwner))
		fmt.Printf("    circuits: completed=%d toComplete=%d\n", tc.CircuitsCompleted, tc.CircuitsToComplete)
		fmt.Printf("    path(%d): %s\n", len(tc.PlotList), plotPath(tc.PlotList))
	}
}
