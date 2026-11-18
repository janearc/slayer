//go:build netcdf

package main

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/celestial-data/netcdf"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// The shape of the weather fixture: six time steps six hours apart, of
// which the odd ones hold data and the even ones are all fill, as ERA-40's
// total cloud cover is.
const (
	fixtureSteps = 6
	fixtureHours = 6
	// holeStep, holeRow and holeColumn are one cell of the step that has
	// no data, under the octant triangle's centre.
	holeStep, holeRow, holeColumn = 3, 3, 1
)

// fixtureValue is the fixture's value in a cell: a tenth for each column
// of longitude and a hundredth for each step, so a place's value says
// where and when it was read.
func fixtureValue(step, row, column int) float64 {
	return 0.1*float64(column) + 0.01*float64(step)
}

// weatherFixture writes a small netCDF grid with the adapter's WriteGrid
// into a temporary folder and gives its path. holding says which steps
// hold data; the others are all fill.
func weatherFixture(t *testing.T, holding func(step int) bool) string {
	t.Helper()
	steps := make([][]float64, fixtureSteps)
	for step := range steps {
		for row := range 4 {
			for column := range 4 {
				value := fixtureValue(step, row, column)
				if !holding(step) || (step == holeStep &&
					row == holeRow && column == holeColumn) {
					value = math.NaN()
				}
				steps[step] = append(steps[step], value)
			}
		}
	}
	times := make([]float64, fixtureSteps)
	for step := range times {
		times[step] = float64(step * fixtureHours)
	}
	path := filepath.Join(t.TempDir(), "weather.nc")
	err := netcdf.WriteGrid(path, netcdf.GridFile{Format: netcdf.FormatClassic,
		Latitudes: []float64{-60, -20, 20, 60}, Longitudes: []float64{0, 90, 180, 270},
		TimeUnits: "hours since 2000-01-01 00:00:00", Times: times,
		Variables: []netcdf.GridVariable{{Name: "tcc",
			LongName: "total cloud cover", Units: "(0 - 1)", Steps: steps}}})
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// oddSteps holds data at the odd time steps only.
func oddSteps(step int) bool { return step%2 == 1 }

// openFixture opens the weather fixture with the options it is given.
func openFixture(t *testing.T, settings options) (*weatherFile, error) {
	t.Helper()
	settings.weather = weatherFixture(t, oddSteps)
	settings.weatherVar = defaultWeatherVariable
	return openWeather(settings)
}

// nearly compares a computed cover with the value worked out by hand.
func nearly(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.12g, want %.12g", got, want)
	}
}

// twoTriangles are the octant over the x, y and z axes, whose centre is
// at longitude 45, and the one over -x, -y and z, at longitude 225. Both
// are at latitude 35.26, between the fixture's rows 20 and 60.
var twoTriangles = []float32{1, 0, 0, 0, 1, 0, 0, 0, 1, -1, 0, 0, 0, -1, 0, 0, 0, 1}

func TestNoWeatherFileIsNoWeather(t *testing.T) {
	found, err := openWeather(options{})
	if found != nil || err != nil {
		t.Fatalf("got %v, %v", found, err)
	}
}

func TestOnlyTheStepsThatHoldDataAreKeptAndTheirDatesAreRight(t *testing.T) {
	found, err := openFixture(t, options{weatherTime: firstWithData})
	if err != nil {
		t.Fatal(err)
	}
	if len(found.kept) != 3 || found.total != fixtureSteps {
		t.Fatalf("%d kept of %d", len(found.kept), found.total)
	}
	for position, want := range []int{1, 3, 5} {
		step := found.kept[position]
		hours := step.Time().Sub(found.kept[0].Time()).Hours()
		if step.Index() != want || hours != float64((want-1)*fixtureHours) {
			t.Fatalf("kept %d is step %d, %v hours on", position, step.Index(), hours)
		}
	}
	if got := found.kept[0].Time().Format(dateLayout); got != "2000-01-01 06:00" {
		t.Fatalf("first date %s", got)
	}
}

func TestCoverIsOneValueAtEachTrianglesCentreForEachKeptStep(t *testing.T) {
	found, err := openFixture(t, options{weatherTime: firstWithData})
	if err != nil {
		t.Fatal(err)
	}
	steps := found.covers(twoTriangles)
	if len(steps) != 3 {
		t.Fatalf("%d steps", len(steps))
	}
	for position, step := range steps {
		if len(step.cover) != 2 || step.index != 1+2*position {
			t.Fatalf("step %d has %d covers", step.index, len(step.cover))
		}
		nearly(t, step.cover[1], 0.25+0.01*float64(step.index))
	}
	nearly(t, steps[0].cover[0], 0.05+0.01)
	nearly(t, steps[2].cover[0], 0.05+0.05)
	if !math.IsNaN(steps[1].cover[0]) {
		t.Fatalf("a triangle over a cell with no data has cover %v", steps[1].cover[0])
	}
}

func TestTheCentreOfATriangleIsTheMeanOfItsCornersOnTheSphere(t *testing.T) {
	at := triangleCentre(twoTriangles, 0)
	lon, lat := lonLat(at)
	nearly(t, lon, 45)
	nearly(t, lat, math.Atan(1/math.Sqrt2)*180/math.Pi)
	closeFloat(t, math.Sqrt(dot(at, at)), 1)
}

func TestTheWeatherTimeDefaultsToTheFirstStepWithData(t *testing.T) {
	found, err := openFixture(t, options{weatherTime: firstWithData})
	if err != nil || found.chosen != 0 || found.kept[found.chosen].Index() != 1 {
		t.Fatalf("chose %d: %v", found.chosen, err)
	}
}

func TestAnExplicitWeatherTimeIsTheFilesStepNotTheKeptPosition(t *testing.T) {
	found, err := openFixture(t, options{weatherTime: 5})
	if err != nil || found.chosen != 2 || found.kept[found.chosen].Index() != 5 {
		t.Fatalf("chose %d: %v", found.chosen, err)
	}
}

func TestAStepWithNoDataIsRefusedNamingTheNearestStepsThatHaveData(t *testing.T) {
	for _, test := range []struct {
		step int
		near string
	}{{0, "are 1"}, {2, "are 1 and 3"}, {4, "are 3 and 5"}, {99, "are 5"}} {
		_, err := openFixture(t, options{weatherTime: test.step})
		if err == nil || !strings.Contains(err.Error(), "no data at time step") ||
			!strings.Contains(err.Error(), test.near) {
			t.Fatalf("step %d: %v", test.step, err)
		}
	}
}

func TestAFileWhoseStepsAreAllFillIsRefusedAsHoldingNoData(t *testing.T) {
	path := weatherFixture(t, func(int) bool { return false })
	_, err := openWeather(options{weather: path, weatherVar: "tcc",
		weatherTime: firstWithData})
	if err == nil || !strings.Contains(err.Error(), "holds no data at any of its 6") ||
		!strings.Contains(err.Error(), path) {
		t.Fatalf("got %v", err)
	}
}

func TestAMissingVariableIsTheAdaptersErrorNamingTheFileAndTheVariable(t *testing.T) {
	path := weatherFixture(t, oddSteps)
	_, err := openWeather(options{weather: path, weatherVar: "nothing",
		weatherTime: firstWithData})
	if err == nil || !strings.Contains(err.Error(), "nothing") ||
		!strings.Contains(err.Error(), "weather.nc") {
		t.Fatalf("got %v", err)
	}
}

func TestAMissingFileIsTheAdaptersError(t *testing.T) {
	_, err := openWeather(options{weather: filepath.Join(t.TempDir(), "no.nc"),
		weatherVar: "tcc", weatherTime: firstWithData})
	if err == nil || !strings.Contains(err.Error(), "no.nc") {
		t.Fatalf("got %v", err)
	}
}

func TestAWeatherTimeNeedsAFileAndCountsFromZero(t *testing.T) {
	for _, test := range []struct{ words []string }{
		{[]string{"window", "--weather-time", "1"}},
		{[]string{"window", "--weather-var", "msl"}},
		{[]string{"window", "--weather", "w.nc", "--weather-time", "-2"}},
	} {
		status, said, kept := runTool(test.words...)
		if status == 0 || kept.calls != 0 || !strings.Contains(said, "--weather") {
			t.Fatalf("%v: %d %s", test.words, status, said)
		}
	}
}

func TestTheWeatherFlagsReachTheOptions(t *testing.T) {
	_, _, kept := runTool("draw", "--shot", "x", "--weather", "w.nc",
		"--weather-var", "msl", "--weather-time", "3")
	if kept.got.weather != "w.nc" || kept.got.weatherVar != "msl" ||
		kept.got.weatherTime != 3 {
		t.Fatalf("got %+v", kept.got)
	}
}

func TestCloudIsMixedInLightSoHalfCoverOverBlackIsBrighterThanHalfTheBytes(t *testing.T) {
	white := srgb.RGB8(255, 255, 255).Swatch()
	colours := make([]byte, 3*bytesPerCorner*2)
	for i := 3; i < len(colours); i += bytesPerCorner {
		colours[i] = 255
	}
	mixClouds(colours, []float64{0.5, math.NaN()}, white)
	if colours[0] < 170 || colours[0] > 200 || colours[0] != colours[1] {
		t.Fatalf("half cover over black is %v", colours[:4])
	}
	for _, kept := range colours[3*bytesPerCorner : 3*bytesPerCorner+3] {
		if kept != 0 {
			t.Fatalf("a triangle with no data changed: %v", colours)
		}
	}
}

func TestFullAndMoreThanFullCoverReachTheCloudAndNoCoverLeavesTheGround(t *testing.T) {
	white := srgb.RGB8(200, 200, 200).Swatch()
	for cover, want := range map[float64]byte{0: 0, -1: 0, 1: 200, 7: 200} {
		colours := make([]byte, 3*bytesPerCorner)
		mixClouds(colours, []float64{cover}, white)
		if got := int(colours[0]); got < int(want)-1 || got > int(want)+1 {
			t.Fatalf("cover %v gives %d, want %d", cover, got, want)
		}
	}
}

func TestNoHudLeavesOutTheDotAtTheObserversPlace(t *testing.T) {
	viewerFixture(t)
	_, earth, _, _ := viewerFiles(t)
	start(options{earth: earth, earthDepth: 1, earthOnly: true,
		distance: defaultDistance, fov: defaultFov, noHud: true})
	view.mode, view.earth.showing = earthView, len(view.earth.images)
	recolourEarth()
	clean := append([]byte{}, view.earth.colours...)
	view.earth.marked = 0
	recolourEarth()
	if string(clean) != string(view.earth.colours) {
		t.Fatal("--no-hud still draws the dot")
	}
	view.noHud = false
	recolourEarth()
	if string(clean) == string(view.earth.colours) {
		t.Fatal("the dot is not drawn with the hud")
	}
}

func TestAWeatherShotLaysTheChosenStepsCoverOverTheEarth(t *testing.T) {
	viewerFixture(t)
	_, earth, _, themes := viewerFiles(t)
	path := weatherFixture(t, oddSteps)
	err := run(options{earth: earth, colourways: themes, colourway: "twilight",
		earthDepth: 1, earthOnly: true, noHud: true, distance: 4.2,
		fov: defaultFov, shot: filepath.Join(t.TempDir(), "w"), view: earthView,
		weather: path, weatherVar: "tcc", weatherTime: 3})
	if err != nil {
		t.Fatal(err)
	}
	triangles := len(view.earth.corners) / 9
	if len(view.earth.weather) != 3 || len(view.earth.cover) != triangles ||
		&view.earth.cover[0] != &view.earth.weather[1].cover[0] {
		t.Fatalf("%d steps, %d covers", len(view.earth.weather), len(view.earth.cover))
	}
}

func TestARunWithABadWeatherFileStopsBeforeAnythingIsBuilt(t *testing.T) {
	viewerFixture(t)
	err := run(options{weather: "missing.nc", weatherVar: "tcc",
		weatherTime: firstWithData})
	if err == nil || !strings.Contains(err.Error(), "missing.nc") ||
		len(view.earth.corners) != 0 {
		t.Fatalf("got %v, %d corners", err, len(view.earth.corners))
	}
}
