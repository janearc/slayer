//go:build netcdf

package main

import (
	"bytes"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Example_film shows what a film says it would write, and writes none.
func Example_film() {
	tool((&recorder{}).launch).RunWith([]string{"film", "--into", "out/f",
		"--frames", "3", "--dry-run"}, strings.NewReader(""), os.Stdout,
		os.Stderr)
	// Output: would write out/f/frame-0001.png through out/f/frame-0003.png
}

// wrapped is a longitude brought into 0 up to 360, to compare places that
// a whole number of turns apart.
func wrapped(lon float64) float64 {
	return lon - fullTurn*math.Floor(lon/fullTurn)
}

// filmedEarth builds an earth of 80 triangles and gives it weather of the
// steps' covers: step k is a quarter of k, and a thousandth more for each
// triangle, so a value says which step and which triangle it is.
func filmedEarth(t *testing.T, steps int) {
	t.Helper()
	viewerFixture(t)
	_, earth, _, _ := viewerFiles(t)
	start(options{earth: earth, earthDepth: 1, earthOnly: true, noHud: true,
		distance: filmDistance, fov: defaultFov, look: "-100,10"})
	triangles := len(view.earth.corners) / 9
	for step := range steps {
		cover := make([]float64, triangles)
		for triangle := range cover {
			cover[triangle] = 0.25*float64(step) + 0.001*float64(triangle)
		}
		view.earth.weather = append(view.earth.weather, weatherStep{index: step,
			cover: cover})
	}
	if steps > 0 {
		view.earth.cover = view.earth.weather[0].cover
	}
}

// filmed is what a frame of a film showed when it was written.
type filmed struct {
	path     string
	lon, lat float64
	cover    []float64
}

// shoot draws a film with a writer that keeps what each frame showed
// instead of drawing it.
func shoot(t *testing.T, settings options) []filmed {
	t.Helper()
	var frames []filmed
	err := drawFilm(settings, func(path string) {
		frames = append(frames, filmed{path: path, lon: view.lon, lat: view.lat,
			cover: append([]float64(nil), view.earth.cover...)})
	})
	if err != nil {
		t.Fatal(err)
	}
	return frames
}

func TestFramesAreNumberedFromOneInFourDigits(t *testing.T) {
	for number, want := range map[int]string{1: "frame-0001.png",
		372: "frame-0372.png", 12345: "frame-12345.png"} {
		if got := frameName(number); got != want {
			t.Fatalf("frame %d is %s, want %s", number, got, want)
		}
	}
}

func TestTheCameraGoesRoundWestwardWhileItsLatitudeSwingsFromNorthToSouth(t *testing.T) {
	for _, turns := range []float64{2, 1} {
		settings := options{turns: turns, tilt: 35}
		for _, test := range []struct{ share, lon, lat float64 }{
			{0, -100, 35}, {0.5, -100 - 180*turns, 0},
			{1, -100 - 360*turns, -35}} {
			lon, lat := filmPlace(test.share, -100, settings)
			nearly(t, lon, test.lon)
			if math.Abs(lat-test.lat) > 1e-9 {
				t.Fatalf("turns %g share %g: latitude %v, want %v", turns,
					test.share, lat, test.lat)
			}
		}
		before, _ := filmPlace(0.25, -100, settings)
		after, _ := filmPlace(0.26, -100, settings)
		if after >= before {
			t.Fatalf("longitude %v then %v: not westward", before, after)
		}
	}
}

func TestAFilmEndsWhereWholeTurnsBringItAndEveryLongitudeIsSeen(t *testing.T) {
	settings := options{turns: 2, tilt: 35}
	first, _ := filmPlace(0, 10, settings)
	last, _ := filmPlace(1, 10, settings)
	nearly(t, wrapped(first), wrapped(last))
	seen := map[int]bool{}
	for frame := range 372 {
		lon, _ := filmPlace(float64(frame)/371, 10, settings)
		seen[int(wrapped(lon)/10)] = true
	}
	if len(seen) != 36 {
		t.Fatalf("%d of 36 sectors of longitude seen", len(seen))
	}
}

func TestADrawnFilmGoesThroughItsPathFromTheLooksLongitude(t *testing.T) {
	filmedEarth(t, 0)
	into := filepath.Join(t.TempDir(), "made", "frames")
	frames := shoot(t, options{into: into, frames: 5, turns: 2, tilt: 35})
	if len(frames) != 5 || view.mode != earthView {
		t.Fatalf("%d frames, mode %d", len(frames), view.mode)
	}
	for number, frame := range frames {
		if want := filepath.Join(into, frameName(number+1)); frame.path != want {
			t.Fatalf("frame %d at %s, want %s", number+1, frame.path, want)
		}
	}
	nearly(t, frames[0].lon, -100)
	nearly(t, frames[2].lon, -100-360)
	nearly(t, frames[4].lon, -100-720)
	nearly(t, frames[0].lat+35, 70)
	nearly(t, frames[2].lat+1, 1)
	nearly(t, frames[4].lat+35, 0.0+0.0)
	if len(frames[0].cover) != 0 || view.earth.cover != nil {
		t.Fatal("a film with no weather laid weather")
	}
}

func TestTheCrossfadeIsExactOnAStepHalfWayBetweenAndAtBothEnds(t *testing.T) {
	steps := []weatherStep{{cover: []float64{0.2, math.NaN()}},
		{cover: []float64{0.6, 1}}, {cover: []float64{1, 0}}}
	for _, test := range []struct {
		position float64
		want     [2]float64
	}{
		{0, [2]float64{0.2, math.NaN()}}, {0.5, [2]float64{0.4, math.NaN()}},
		{1, [2]float64{0.6, 1}}, {1.5, [2]float64{0.8, 0.5}},
		{2, [2]float64{1, 0}}, {-3, [2]float64{0.2, math.NaN()}},
		{9, [2]float64{1, 0}},
	} {
		got := make([]float64, 2)
		blendCover(steps, test.position, got)
		for triangle, want := range test.want {
			if math.IsNaN(want) != math.IsNaN(got[triangle]) ||
				!math.IsNaN(want) && math.Abs(got[triangle]-want) > 1e-12 {
				t.Fatalf("position %v triangle %d: %v, want %v", test.position,
					triangle, got[triangle], want)
			}
		}
	}
}

func TestOneStepIsHeldForTheWholeFilm(t *testing.T) {
	got := make([]float64, 1)
	blendCover([]weatherStep{{cover: []float64{0.3}}}, 0, got)
	nearly(t, got[0], 0.3)
}

func TestAFilmRunsOnceThroughTheKeptStepsEvenly(t *testing.T) {
	filmedEarth(t, 3)
	frames := shoot(t, options{into: t.TempDir(), frames: 5, turns: 1, tilt: 35})
	for frame, quarters := range []float64{0, 0.5, 1, 1.5, 2} {
		nearly(t, frames[frame].cover[0], 0.25*quarters)
		nearly(t, frames[frame].cover[3], 0.25*quarters+0.003)
	}
}

func TestEachFrameKeepsItsOwnCoverAndTheFirstAndLastAreTheEndSteps(t *testing.T) {
	filmedEarth(t, 3)
	frames := shoot(t, options{into: t.TempDir(), frames: 2, turns: 1, tilt: 35})
	nearly(t, frames[0].cover[0], 0)
	nearly(t, frames[1].cover[0], 0.5)
}

func TestFilmCommandDefaultsAreTheDocumentedOnes(t *testing.T) {
	status, said, kept := runTool("film", "--into", "out")
	want := options{colourways: "colourways", colourway: "twilight",
		depth: 13, earthDepth: 7, wideDepth: 8, fov: 1, distance: filmDistance,
		weatherVar: "tcc", weatherTime: firstWithData, noHud: true,
		into: "out", frames: 372, turns: 2, tilt: 35}
	if status != 0 || kept.got != want {
		t.Fatalf("status %d, got %+v: %s", status, kept.got, said)
	}
}

func TestFilmFlagsAreTakenAndTheDistanceMayBeSet(t *testing.T) {
	status, said, kept := runTool("film", "--into", "o", "--frames", "10",
		"--turns", "1.5", "--tilt", "20", "--distance", "5", "--look", "1,2")
	got := kept.got
	if status != 0 || got.frames != 10 || got.turns != 1.5 || got.tilt != 20 ||
		got.distance != 5 || got.look != "1,2" || !got.noHud {
		t.Fatalf("status %d, got %+v: %s", status, got, said)
	}
}

func TestOnlyAFilmHasIntoFramesTurnsAndTilt(t *testing.T) {
	for _, flag := range []string{"--into", "--frames", "--turns", "--tilt"} {
		for _, verb := range []string{"window", "draw"} {
			status, said, kept := runTool(verb, flag, "1")
			if status == 0 || kept.calls != 0 ||
				!strings.Contains(said, "unknown flag: "+flag) {
				t.Fatalf("%s took %s: %d %s", verb, flag, status, said)
			}
		}
	}
}

func TestAFilmRefusesWhatItCannotDrawInASentenceThatNamesIt(t *testing.T) {
	for _, test := range []struct {
		name  string
		words []string
		want  string
	}{
		{"no into", []string{"film"}, "film needs --into"},
		{"empty into", []string{"film", "--into", ""}, "film needs --into"},
		{"one frame", []string{"film", "--into", "o", "--frames", "1"}, "--frames is 1; a film needs at least 2"},
		{"no frames", []string{"film", "--into", "o", "--frames", "0"}, "--frames is 0"},
		{"negative frames", []string{"film", "--into", "o", "--frames", "-3"}, "--frames is -3"},
		{"frames not a number", []string{"film", "--into", "o", "--frames", "many"}, `--frames wants a whole number, got "many"`},
		{"no turns", []string{"film", "--into", "o", "--turns", "0"}, "--turns is 0; it must be more than 0"},
		{"negative turns", []string{"film", "--into", "o", "--turns", "-1"}, "--turns is -1"},
		{"tilt at the pole", []string{"film", "--into", "o", "--tilt", "90"}, "--tilt is 90"},
		{"tilt past it", []string{"film", "--into", "o", "--tilt", "95"}, "--tilt is 95"},
		{"negative tilt", []string{"film", "--into", "o", "--tilt", "-5"}, "--tilt is -5"},
		{"a step of its own", []string{"film", "--into", "o", "--weather", "w.nc", "--weather-time", "1"}, "a film runs through all the steps"},
		{"too near", []string{"film", "--into", "o", "--distance", "1"}, "--distance is 1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, said, kept := runTool(test.words...)
			if status == 0 || kept.calls != 0 || !strings.Contains(said, test.want) {
				t.Fatalf("status %d, %d calls, want %q: %s", status,
					kept.calls, test.want, said)
			}
		})
	}
}

func TestADryFilmWritesNothing(t *testing.T) {
	into := filepath.Join(t.TempDir(), "never")
	status, said, kept := runTool("film", "--into", into, "--dry-run")
	if status != 0 || kept.calls != 0 || !strings.Contains(said, "frame-0372.png") {
		t.Fatalf("status %d: %s", status, said)
	}
	if _, err := os.Stat(into); err == nil {
		t.Fatal("a dry film made its directory")
	}
}

func TestAFilmIsDrawnThroughMetalWithNoDotAndTheWeatherMoving(t *testing.T) {
	viewerFixture(t)
	_, earth, _, themes := viewerFiles(t)
	into := filepath.Join(t.TempDir(), "frames")
	err := run(options{earth: earth, colourways: themes, colourway: "twilight",
		earthDepth: 1, earthOnly: true, noHud: true, distance: filmDistance,
		fov: defaultFov, into: into, frames: 3, turns: 2, tilt: 35,
		weather: weatherFixture(t, oddSteps), weatherVar: "tcc",
		weatherTime: firstWithData})
	if err != nil {
		t.Fatal(err)
	}
	var pictures [][]byte
	for number := 1; number <= 3; number++ {
		raw, err := os.ReadFile(filepath.Join(into, frameName(number)))
		if err != nil {
			t.Fatal(err)
		}
		picture, err := png.Decode(bytes.NewReader(raw))
		if err != nil || picture.Bounds().Dx() != 1400 {
			t.Fatalf("frame %d: %v %v", number, picture.Bounds(), err)
		}
		pictures = append(pictures, raw)
	}
	if bytes.Equal(pictures[0], pictures[1]) || bytes.Equal(pictures[0], pictures[2]) {
		t.Fatal("frames show the same picture")
	}
	if _, err := os.Stat(filepath.Join(into, frameName(4))); err == nil {
		t.Fatal("a fourth frame was drawn")
	}
}
