package graphics

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/samuelyuan/Civ5MapImage/fileio"
)

// newLabelGeometryTestMap builds a 1x1 map with a single named city tile for label tests.
func newLabelGeometryTestMap(cityName string, owner int, teamColor, civType string) *fileio.Civ5MapData {
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{X: 0, Y: 0, CityId: 0, CityName: cityName, Owner: owner}},
		},
	}
	if civType != "" {
		mapData.Civ5PlayerData = []*fileio.Civ5PlayerData{{Index: 0, CivType: civType, TeamColor: teamColor}}
		mapData.CityOwnerIndexMap = map[int]int{owner: 0}
	}
	return mapData
}

func TestPhysicalCityNameLabel(t *testing.T) {
	const radius = 16.0
	mapData := newLabelGeometryTestMap("Rome", -1, "", "")

	label := PhysicalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(radius, 100))

	// Tile (0, 0) is centered at (24, 84) on a canvas 100 tall; the name is centered above it: 6 px per character, 4 characters.
	wantX, wantY := 24.0-12.0, 84.0-8.0
	if label.Text != "Rome" || label.X != wantX || label.Y != wantY {
		t.Errorf("PhysicalCityNameLabel() = %+v, want {Text:Rome X:%v Y:%v}", label, wantX, wantY)
	}
	if label.R != 255 || label.G != 255 || label.B != 255 {
		t.Errorf("PhysicalCityNameLabel() color = (%d,%d,%d), want white", label.R, label.G, label.B)
	}
}

func TestPhysicalCityNameLabelTrimsNullByte(t *testing.T) {
	mapData := newLabelGeometryTestMap("Rome\x00garbage", -1, "", "")
	label := PhysicalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if label.Text != "Rome" {
		t.Errorf("PhysicalCityNameLabel().Text = %q, want %q", label.Text, "Rome")
	}
}

func TestPoliticalCityNameLabelKnownColor(t *testing.T) {
	const radius = 16.0
	mapData := newLabelGeometryTestMap("Rome", 0, "PLAYERCOLOR_BLACK", "CIVILIZATION_ROME")

	label := PoliticalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(radius, 100))

	renderColor := civColorMap["PLAYERCOLOR_BLACK"]
	wantColor := blendColor(renderColor.InnerColor, color.RGBA{255, 255, 255, 255}, 0.2)
	if label.R != wantColor.R || label.G != wantColor.G || label.B != wantColor.B {
		t.Errorf("PoliticalCityNameLabel() color = %d,%d,%d, want fully blended %d,%d,%d",
			label.R, label.G, label.B, wantColor.R, wantColor.G, wantColor.B)
	}
}

func TestPoliticalCityNameLabelUnknownColorFallsBackToWhite(t *testing.T) {
	mapData := newLabelGeometryTestMap("Rome", -1, "", "")
	label := PoliticalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, newTileLayout(16.0, 100))
	if label.R != 255 || label.G != 255 || label.B != 255 {
		t.Errorf("PoliticalCityNameLabel() color = (%d,%d,%d), want white fallback", label.R, label.G, label.B)
	}
}

// A label is centered on its own tile at any row and map height.
func TestLabelIsCenteredOnItsTile(t *testing.T) {
	const radius, rows = 16.0, 7
	l := newTileLayout(radius, 400)
	for row := 0; row < rows; row++ {
		pos := fileio.TilePos{Row: row, Col: 3}
		x, y := l.center(pos)
		labelX, labelY := cityLabelPosition(l, pos, "Rome")
		if labelX != x-12 || labelY != y-radius/2 {
			t.Errorf("row %d: label at (%v, %v), want (%v, %v)", row, labelX, labelY, x-12, y-radius/2)
		}
	}
}

const testRadius = 16.0

// siteAt is a city with its marker at (cx, cy) and its label at the default place: centered above the marker.
func siteAt(name string, cx, cy float64) citySite {
	return citySite{
		label: ColoredText{Text: name, X: cx - 7*float64(len(name))/2, Y: cy - testRadius/2, R: 200, G: 100, B: 50},
		cx:    cx, cy: cy,
	}
}

func TestPlaceLabelsLeavesLabelsWhereTheyAreWhenNothingOverlaps(t *testing.T) {
	sites := []citySite{siteAt("Rome", 100, 100), siteAt("Kyiv", 300, 100), siteAt("Oslo", 100, 300)}

	placed := placeLabels(sites, testRadius)

	for i, site := range sites {
		if placed[i] != site.label {
			t.Errorf("label %d moved: %+v, want %+v", i, placed[i], site.label)
		}
	}
}

// Of two labels that would overlap, the earlier keeps its place and the later moves clear of it.
func TestPlaceLabelsMovesALaterLabelOffAnEarlierOne(t *testing.T) {
	// Same row, 70 px apart: the labels meet, but neither city's marker is under the other's label.
	sites := []citySite{siteAt("Port-au-Prince", 100, 100), siteAt("Santo Domingo", 170, 100)}
	if !labelBox(sites[0].label).Overlaps(labelBox(sites[1].label)) {
		t.Fatal("test setup: the default labels should overlap")
	}
	if labelBox(sites[0].label).Overlaps(markerBox(sites[1], testRadius)) || labelBox(sites[1].label).Overlaps(markerBox(sites[0], testRadius)) {
		t.Fatal("test setup: no label should cover the other city's marker")
	}

	placed := placeLabels(sites, testRadius)

	if placed[0] != sites[0].label {
		t.Errorf("the earlier label moved: %+v, want %+v", placed[0], sites[0].label)
	}
	if labelBox(placed[0]).Overlaps(labelBox(placed[1])) {
		t.Errorf("labels still overlap: %v and %v", labelBox(placed[0]), labelBox(placed[1]))
	}
	// Below its own marker, at the same horizontal place, is the first free place it tries.
	if want := sites[1].cy + testRadius + 1; placed[1].X != sites[1].label.X || placed[1].Y != want {
		t.Errorf("the later label is at (%v, %v), want below its marker at (%v, %v)", placed[1].X, placed[1].Y, sites[1].label.X, want)
	}
}

// Text that touches end to end, with no gap for a halo, counts as overlapping.
func TestPlaceLabelsTreatsTextThatTouchesAsOverlapping(t *testing.T) {
	first := siteAt("Kingston", 100, 100) // its text spans 56 px from x=72
	second := siteAt("Kingston", 156, 100)
	second.label.X = first.label.X + 56 // starting exactly where the first ends
	second.cx = second.label.X + 28

	placed := placeLabels([]citySite{first, second}, testRadius)

	if placed[1] == second.label {
		t.Errorf("the second label stayed where it touches the first: %+v", placed[1])
	}
}

// Every place a label may go, in the order they are tried, for a marker at (100, 100), a radius of 16 and a 56 px label.
func TestLabelOptionsAreTheEightPlacesAroundTheMarker(t *testing.T) {
	site := siteAt("Kingston", 100, 100)

	got := labelOptions(site, testRadius)

	beside := 100 + testRadius/3
	want := [][2]float64{
		{72, 92},      // the default place: centered above the marker
		{72, 117},     // below it
		{108, beside}, // beside it on the right
		{36, beside},  // beside it on the left
		{94, 92},      // above, starting at the marker
		{50, 92},      // above, ending at the marker
		{94, 117},     // below, starting at the marker
		{50, 117},     // below, ending at the marker
	}
	if len(got) != len(want) {
		t.Fatalf("labelOptions() returned %d places, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i][0]-want[i][0]) > 1e-9 || math.Abs(got[i][1]-want[i][1]) > 1e-9 {
			t.Errorf("option %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// A label also keeps clear of the markers of other cities, not only of their labels.
func TestPlaceLabelsMovesALabelOffAnotherCitysMarker(t *testing.T) {
	rome := siteAt("Constantinople", 100, 100)
	// A city whose marker sits in the middle of Rome's default label, with no label of its own to collide with.
	other := citySite{label: ColoredText{Text: "X", X: 400, Y: 400}, cx: 100, cy: 100 - testRadius/2 - 4}
	if !labelBox(rome.label).Overlaps(markerBox(other, testRadius)) {
		t.Fatal("test setup: the other city's marker should sit on Rome's default label")
	}

	placed := placeLabels([]citySite{rome, other}, testRadius)

	if labelBox(placed[0]).Overlaps(markerBox(other, testRadius)) {
		t.Errorf("label %v still covers the other marker %v", labelBox(placed[0]), markerBox(other, testRadius))
	}
}

// Only the position changes: the text and color of a moved label are the same.
func TestPlaceLabelsChangesOnlyPositions(t *testing.T) {
	sites := []citySite{siteAt("Kingston", 100, 100), siteAt("Port-au-Prince", 125, 100)}

	placed := placeLabels(sites, testRadius)

	for i, site := range sites {
		if placed[i].Text != site.label.Text || placed[i].R != site.label.R || placed[i].G != site.label.G || placed[i].B != site.label.B {
			t.Errorf("label %d text or color changed: %+v, was %+v", i, placed[i], site.label)
		}
	}
}

// However crowded, a label is never worse off than at its default place: it takes the free place if there is one, or else the least covered.
func TestPlaceLabelsNeverCoversMoreThanTheDefaultPlaceWould(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	names := []string{"Rome", "Kyiv", "Constantinople", "Port-au-Prince", "Al", "Minneapolis"}
	for round := 0; round < 20; round++ {
		var sites []citySite
		for i := 0; i < 40; i++ {
			sites = append(sites, siteAt(names[rng.Intn(len(names))], 60+rng.Float64()*260, 40+rng.Float64()*120))
		}

		placed := placeLabels(sites, testRadius)

		for i, site := range sites {
			var obstacles []citySite
			for j := range sites {
				if j != i {
					obstacles = append(obstacles, sites[j])
				}
			}
			covered := func(label ColoredText) int {
				total := 0
				for j := 0; j < i; j++ {
					in := labelBox(label).Intersect(labelBox(placed[j]))
					total += in.Dx() * in.Dy()
				}
				for _, o := range obstacles {
					in := labelBox(label).Intersect(markerBox(o, testRadius))
					total += in.Dx() * in.Dy()
				}
				return total
			}
			if got, byDefault := covered(placed[i]), covered(site.label); got > byDefault {
				t.Fatalf("round %d label %d %q: covers %d pixels where it was put, %d at its default place", round, i, site.label.Text, got, byDefault)
			}
		}
	}
}

func TestPlacedCityLabelsSkipsTilesWithoutACityOrAName(t *testing.T) {
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{CityName: "Rome"}, {}, {CityName: "Kyiv"}, {CityName: "Nameless"}},
		},
	}
	size := fileio.MapSize{Height: 1, Width: 4}
	l := layoutForMap(size, testRadius)
	labelFor := func(pos fileio.TilePos) ColoredText {
		if pos.Col == 3 {
			return ColoredText{} // a city with no displayable name
		}
		return ColoredText{Text: mapData.TileImprovement(pos).CityName}
	}

	placed := placedCityLabels(mapData, size, l, labelFor)

	var names []string
	for _, label := range placed {
		names = append(names, label.Text)
	}
	if got := strings.Join(names, ","); got != "Rome,Kyiv" {
		t.Errorf("placed labels = %q, want %q", got, "Rome,Kyiv")
	}
}

// Drawn through DrawPoliticalCityNames, two neighboring cities with long names don't end up on top of each other.
func TestDrawPoliticalCityNamesSeparatesOverlappingLabels(t *testing.T) {
	mr := NewMapRenderer(DefaultDrawingConfig())
	canvas := NewMockCanvas(400, 200)
	mapData := &fileio.Civ5MapData{
		MapTileImprovements: [][]*fileio.Civ5MapTileImprovement{
			{{CityName: "Constantinople", Owner: -1, CityId: -1}, {CityName: "Thessalonica", Owner: -1, CityId: -1}},
		},
	}
	size := fileio.MapSize{Height: 1, Width: 2}
	l := layoutForMap(size, mr.config.Radius)
	first := PoliticalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 0}, l)
	second := PoliticalCityNameLabel(mapData, fileio.TilePos{Row: 0, Col: 1}, l)
	if !labelBox(first).Overlaps(labelBox(second)) {
		t.Fatal("test setup: the default labels should overlap")
	}

	mr.DrawPoliticalCityNames(canvas, mapData, size, l)

	var drawn []string // each label's own DrawString is the last of its five (four halo copies, then the label)
	for _, op := range canvas.GetOperations() {
		if strings.HasPrefix(op, "DrawString") {
			drawn = append(drawn, op)
		}
	}
	if len(drawn) != 10 {
		t.Fatalf("recorded %d DrawString ops, want 10 (two labels of five): %v", len(drawn), drawn)
	}
	if want := fmt.Sprintf(`DrawString("Thessalonica", %.2f, %.2f)`, second.X, second.Y); drawn[9] == want {
		t.Errorf("the second label was drawn at its default place, on top of the first: %s", drawn[9])
	}
}
