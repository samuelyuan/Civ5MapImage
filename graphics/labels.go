package graphics

import (
	"image"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// ColoredText is a text label plus the position and color to draw it at.
type ColoredText struct {
	Text    string
	X, Y    float64
	R, G, B uint8
}

// trimCityName trims a stored city name at its first null byte.
func trimCityName(name string) string {
	if i := strings.IndexByte(name, 0); i >= 0 {
		return name[:i]
	}
	return name
}

// cityNameText returns the display city name of the tile at pos.
func cityNameText(mapData *fileio.Civ5MapData, pos fileio.TilePos) string {
	return trimCityName(mapData.TileImprovement(pos).CityName)
}

// cityLabelPosition returns the anchor for a tile's city label, centered above the tile.
func cityLabelPosition(l tileLayout, pos fileio.TilePos, cityName string) (float64, float64) {
	halfWidth := 6.0 * float64(len(cityName)) / 2.0
	x, y := l.center(pos)
	return x - halfWidth, y - l.radius/2
}

// PhysicalCityNameLabel returns the city label of the tile at pos, in white.
func PhysicalCityNameLabel(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout) ColoredText {
	cityName := cityNameText(mapData, pos)
	x, y := cityLabelPosition(l, pos, cityName)
	return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
}

// PoliticalCityNameLabel returns the tile's city label in its civ's color, white if unrecognized.
func PoliticalCityNameLabel(mapData *fileio.Civ5MapData, pos fileio.TilePos, l tileLayout) ColoredText {
	cityName := cityNameText(mapData, pos)
	x, y := cityLabelPosition(l, pos, cityName)

	tileColor := fileio.GetPoliticalMapTileColor(mapData, pos)
	renderColor, ok := civColorMap[tileColor]
	if !ok {
		return ColoredText{Text: cityName, X: x, Y: y, R: 255, G: 255, B: 255}
	}

	textColor := markerColor(shadesFor(renderColor, tileIsMinor(mapData, pos)).border)
	return ColoredText{Text: cityName, X: x, Y: y, R: textColor.R, G: textColor.G, B: textColor.B}
}

// citySite is a city's label at its default place (above its marker), and where the marker is.
type citySite struct {
	label  ColoredText
	cx, cy float64
}

// labelBox returns the pixels a label's text and halo occupy, with a pixel of room around them.
func labelBox(label ColoredText) image.Rectangle {
	x, y := int(math.Floor(label.X)), int(math.Floor(label.Y))
	return image.Rect(x-2, y-13, x+7*utf8.RuneCountInString(label.Text)+2, y+4)
}

// markerBox returns the pixels a city's marker occupies, with room around them.
func markerBox(site citySite, radius float64) image.Rectangle {
	reach := int(radius * 3 / 8)
	x, y := int(math.Floor(site.cx)), int(math.Floor(site.cy))
	return image.Rect(x-reach, y-reach, x+reach+1, y+reach+1)
}

// coveredArea is how many pixels of box the obstacles cover, counted once for each obstacle.
func coveredArea(box image.Rectangle, obstacles []image.Rectangle) int {
	total := 0
	for _, o := range obstacles {
		in := box.Intersect(o)
		total += in.Dx() * in.Dy()
	}
	return total
}

// labelOptions returns a site's candidate label places, best first: above, below, right, left, then above and below shifted sideways.
func labelOptions(site citySite, radius float64) [][2]float64 {
	width := 7 * float64(utf8.RuneCountInString(site.label.Text))
	below, beside := site.cy+radius+1, site.cy+radius/3
	nudge := radius * 3 / 8
	return [][2]float64{
		{site.label.X, site.label.Y},
		{site.label.X, below},
		{site.cx + radius/2, beside},
		{site.cx - radius/2 - width, beside},
		{site.cx - nudge, site.label.Y},
		{site.cx + nudge - width, site.label.Y},
		{site.cx - nudge, below},
		{site.cx + nudge - width, below},
	}
}

// placeLabels moves labels off other labels and other cities' markers. Each keeps its default place if free, else the first free labelOptions place, else the least covered; earlier labels win ties.
func placeLabels(sites []citySite, radius float64) []ColoredText {
	markers := make([]image.Rectangle, len(sites))
	for i, site := range sites {
		markers[i] = markerBox(site, radius)
	}

	placed := make([]ColoredText, len(sites))
	taken := make([]image.Rectangle, 0, len(sites)) // the boxes of the labels placed so far
	for i, site := range sites {
		obstacles := append([]image.Rectangle(nil), taken...)
		for j, marker := range markers {
			if j != i {
				obstacles = append(obstacles, marker)
			}
		}

		best, bestCovered := site.label, -1
		for _, option := range labelOptions(site, radius) {
			candidate := site.label
			candidate.X, candidate.Y = option[0], option[1]
			covered := coveredArea(labelBox(candidate), obstacles)
			if bestCovered < 0 || covered < bestCovered {
				best, bestCovered = candidate, covered
			}
			if covered == 0 {
				break
			}
		}
		placed[i] = best
		taken = append(taken, labelBox(best))
	}
	return placed
}

// placedCityLabels returns the cities' labels in row-major order, placed by placeLabels; labelFor gives the default place.
func placedCityLabels(mapData *fileio.Civ5MapData, mapSize fileio.MapSize, l tileLayout, labelFor func(fileio.TilePos) ColoredText) []ColoredText {
	var sites []citySite
	for _, pos := range allTiles(mapSize) {
		if mapData.TileImprovement(pos).CityName == "" {
			continue
		}
		if label := labelFor(pos); label.Text != "" {
			cx, cy := l.center(pos)
			sites = append(sites, citySite{label, cx, cy})
		}
	}
	return placeLabels(sites, l.radius)
}
