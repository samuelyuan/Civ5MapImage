package fileio

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidationResult is one named cross-check: how many data points agreed out of how many were
// checkable. Omitted entirely when Total is 0.
type ValidationResult struct {
	Name   string
	Passed int
	Total  int
	Notes  string // extra context alongside the count, or empty
}

func (v ValidationResult) String() string {
	s := fmt.Sprintf("Validated %d/%d %s", v.Passed, v.Total, v.Name)
	if v.Notes != "" {
		s += " (" + v.Notes + ")"
	}
	return s
}

func PrintValidationResults(results []ValidationResult) {
	for _, r := range results {
		fmt.Println(r.String())
	}
}

func plotTypeFromMapTile(mapData *Civ5MapData, row, column int) (int8, bool) {
	if row < 0 || row >= len(mapData.MapTiles) || column < 0 || column >= len(mapData.MapTiles[row]) {
		return 0, false
	}
	if IsWaterTile(mapData, row, column) {
		return 3, true // PLOT_OCEAN
	}
	switch mapData.MapTiles[row][column].Elevation {
	case 2:
		return 0, true // PLOT_MOUNTAIN
	case 1:
		return 1, true // PLOT_HILLS
	default:
		return 2, true // PLOT_LAND
	}
}

// mapRiverEdges reads a .civ5map tile's RiverData into (E, SE, SW) booleans: 1=E, 2=SE, 4=SW.
func mapRiverEdges(riverData int) (e, se, sw bool) {
	return riverData&1 != 0, riverData&2 != 0, riverData&4 != 0
}

// replayRiverEdges reads a .civ5replay tile's RiverBits into (E, SE, SW) booleans: bit>>1(2)=SE,
// >>2(4)=E, >>3(8)=SW - a different bit-to-edge assignment than the .civ5map format.
func replayRiverEdges(riverBits int8) (e, se, sw bool) {
	b := int(riverBits)
	return b&4 != 0, b&2 != 0, b&8 != 0
}

// ValidateMapAndReplay excludes Feature/FeatureWonder: unlike the hardcoded PlotType enum,
// they're mod-dependent XML database indices.
func ValidateMapAndReplay(mapData *Civ5MapData, replayData *Civ5ReplayData) []ValidationResult {
	var results []ValidationResult

	mapWidth := int(mapData.MapHeader.Width)
	mapHeight := int(mapData.MapHeader.Height)
	dimensionsMatch := mapWidth == replayData.MapWidth && mapHeight == replayData.MapHeight
	dimPassed := 0
	if dimensionsMatch {
		dimPassed = 1
	}
	results = append(results, ValidationResult{
		Name:   "grid dimensions",
		Passed: dimPassed,
		Total:  1,
		Notes:  fmt.Sprintf("map=%dx%d replay=%dx%d", mapWidth, mapHeight, replayData.MapWidth, replayData.MapHeight),
	})

	if !dimensionsMatch || len(replayData.Tiles) == 0 {
		return results // nothing further to compare without matching, non-empty grids
	}

	plotTypeMatches, terrainMatches, riverEdgeMatches := 0, 0, 0
	totalTiles := mapWidth * mapHeight
	totalRiverEdges := 0
	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			idx := y*mapWidth + x
			if idx >= len(replayData.Tiles) {
				continue
			}
			replayTile := replayData.Tiles[idx]

			if expectedPlotType, ok := plotTypeFromMapTile(mapData, y, x); ok && expectedPlotType == replayTile.PlotType {
				plotTypeMatches++
			}

			mapTerrain := int8(mapData.MapTiles[y][x].TerrainType)
			if mapTerrain == replayTile.TerrainType {
				terrainMatches++
			}

			mapE, mapSE, mapSW := mapRiverEdges(mapData.MapTiles[y][x].RiverData)
			replayE, replaySE, replaySW := replayRiverEdges(replayTile.RiverBits)
			totalRiverEdges += 3
			if mapE == replayE {
				riverEdgeMatches++
			}
			if mapSE == replaySE {
				riverEdgeMatches++
			}
			if mapSW == replaySW {
				riverEdgeMatches++
			}
		}
	}

	results = append(results, ValidationResult{Name: "tile PlotType (ocean/hills/mountain/land)", Passed: plotTypeMatches, Total: totalTiles})
	results = append(results, ValidationResult{Name: "tile TerrainType (raw index)", Passed: terrainMatches, Total: totalTiles})
	results = append(results, ValidationResult{Name: "river edges (E/SE/SW per tile)", Passed: riverEdgeMatches, Total: totalRiverEdges})

	return results
}

// ValidateMapReplayCompatible turns ValidateMapAndReplay's results into a pass/fail gate. Requires
// replayData.MapWidth/MapHeight/Tiles; skipped (with a printed reason) when a replay source doesn't
// populate them.
func ValidateMapReplayCompatible(mapData *Civ5MapData, replayData *Civ5ReplayData) ([]ValidationResult, error) {
	var missingFields []string
	if replayData.MapWidth <= 0 || replayData.MapHeight <= 0 {
		missingFields = append(missingFields, "MapWidth/MapHeight")
	}
	if len(replayData.Tiles) == 0 {
		missingFields = append(missingFields, "Tiles")
	}
	if len(missingFields) > 0 {
		fmt.Printf("Skipping map/replay geography validation: replay data is missing %s\n", strings.Join(missingFields, ", "))
		return nil, nil
	}

	results := ValidateMapAndReplay(mapData, replayData)
	for _, r := range results {
		switch r.Name {
		case "grid dimensions":
			if r.Passed != r.Total {
				return results, fmt.Errorf("map and replay grid dimensions do not match (%s)", r.Notes)
			}
		case "tile PlotType (ocean/hills/mountain/land)":
			// PlotType/TerrainType are hardcoded: any mismatch means the wrong map.
			if r.Total > 0 && r.Passed != r.Total {
				return results, fmt.Errorf("map and replay tile geography disagree on %d/%d tiles (PlotType) - this is likely the wrong map for this replay", r.Total-r.Passed, r.Total)
			}
		case "tile TerrainType (raw index)":
			if r.Total > 0 && r.Passed != r.Total {
				return results, fmt.Errorf("map and replay tile geography disagree on %d/%d tiles (TerrainType) - this is likely the wrong map for this replay", r.Total-r.Passed, r.Total)
			}
		}
	}
	return results, nil
}

// ValidateMapFilenameMatch: a mismatch is only a warning, never a hard failure.
func ValidateMapFilenameMatch(mapFilename string, replayData *Civ5ReplayData) *ValidationResult {
	if replayData.MapFileStem == "" {
		return nil // e.g. a replay converted from a .civ5save - nothing to compare
	}
	mapStem := strings.TrimSuffix(filepath.Base(mapFilename), filepath.Ext(mapFilename))
	passed := 0
	if strings.EqualFold(mapStem, replayData.MapFileStem) {
		passed = 1
	}
	return &ValidationResult{
		Name:   "map filename matches the one recorded in the replay",
		Passed: passed,
		Total:  1,
		Notes:  fmt.Sprintf("provided -map=%q, but replay's own recorded map=%q", mapStem, replayData.MapFileStem),
	}
}

func ValidateFileExtension(filename string, allowedExt ...string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range allowedExt {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}
	return fmt.Errorf("filename %q has extension %q, expected %s", filename, ext, strings.Join(allowedExt, " or "))
}
