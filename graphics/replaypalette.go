package graphics

import (
	"image/color"
	"maps"
	"slices"
	"strings"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// terrainColors are the physical map's terrain colors.
var terrainColors = map[string]color.RGBA{
	"TERRAIN_GRASS":  {105, 125, 54, 255},
	"TERRAIN_PLAINS": {127, 121, 71, 255},
	"TERRAIN_DESERT": {200, 200, 164, 255},
	"TERRAIN_TUNDRA": {118, 123, 117, 255},
	"TERRAIN_SNOW":   {238, 249, 255, 255},
	"TERRAIN_COAST":  {95, 149, 149, 255},
	"TERRAIN_OCEAN":  {47, 74, 93, 255},
}

// unknownTerrainColor is what a terrain missing from terrainColors draws as.
var unknownTerrainColor = color.RGBA{0, 0, 0, 255}

// GetPhysicalMapTileColor returns the color of a terrain, or unknownTerrainColor if it has none.
func GetPhysicalMapTileColor(terrain string) color.RGBA {
	if c, ok := terrainColors[terrain]; ok {
		return c
	}
	return unknownTerrainColor
}

// replayPinnedColors lists every flat color the renderer can draw (black background, outline shades, civ colors via shadesFor/markerColor).
func replayPinnedColors(mapData *fileio.Civ5MapData) []color.RGBA {
	white := color.RGBA{255, 255, 255, 255}
	colors := []color.RGBA{
		{0, 0, 0, 255}, white,
		mountainBaseColor, mountainPeakColor, riverColor, railroadColor, roadColor, unknownRouteColor,
	}
	terrainShades := []color.RGBA{unknownTerrainColor}
	for _, terrain := range slices.Sorted(maps.Keys(terrainColors)) {
		terrainShades = append(terrainShades, terrainColors[terrain])
	}
	for _, c := range terrainShades {
		colors = append(colors, c, tileOutlineColor(c))
	}
	for _, player := range mapData.Civ5PlayerData {
		civColor, ok := civColorMap[player.TeamColor]
		if !ok {
			continue
		}
		shades := shadesFor(civColor, strings.Contains(player.CivType, "MINOR"))
		colors = append(colors, shades.fill, shades.accent, markerColor(shades.accent), tileOutlineColor(shades.fill))
	}
	return colors
}

// replayPalette is every color the replay renderer can draw, deduplicated, as an exact GIF palette.
func replayPalette(mapData *fileio.Civ5MapData) color.Palette {
	seen := map[color.RGBA]bool{}
	var palette color.Palette
	for _, c := range replayPinnedColors(mapData) {
		if !seen[c] {
			seen[c] = true
			palette = append(palette, c)
		}
	}
	return palette
}
