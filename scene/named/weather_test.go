package named_test

import (
	"bytes"
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/weather"
)

// a tiny weather for the tests, and a moment in it: picked by the author
// of this file, 2026-11-12, as the least grid the solver holds and two
// of its longest steps, an hour each
const (
	tinyAcross, tinyRows = 16, 8
	twoHours             = 2 * time.Hour
	oneSecond            = "1s"
)

// the weather's defaults, defined by cmd/weather's flags, which the
// scene's defaults are held to here
const (
	commandLimit, commandRadius               = 80.0, 6e7
	commandDay, commandDrag, commandRelax     = 18000, 1e6, 2e6
	commandRate, commandWidth, commandSpin    = 3.8e-17, 1.3e6, 1.5e-5
	commandBands, commandCooking, commandSeed = 15.0, 1e6, 7
	halfTurn                                  = 180
)

// tiny is the with block of a tiny weather whose scene seconds are its
// own seconds.
func tiny() map[string]any {
	return map[string]any{"across": tinyAcross, "rows": tinyRows,
		"pace": oneSecond}
}

// weatherMaps is the weather scene from the registry.
func weatherMaps(t *testing.T) *named.Maps {
	t.Helper()
	scene, err := named.GetMaps("weather")
	if err != nil {
		t.Fatal(err)
	}
	return scene
}

// onCPU is what a caller asks of the weather on the cpu, with a cache
// or none.
func onCPU(t *testing.T, cache string) named.MapsAsk {
	t.Helper()
	ask, err := named.AskMaps(named.CPU, true, cache, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	return ask
}

// built is the tiny weather on the cpu.
func built(t *testing.T, cache string) named.Mapper {
	t.Helper()
	mapper, err := weatherMaps(t).Build(tiny(), onCPU(t, cache))
	if err != nil {
		t.Fatal(err)
	}
	return mapper
}

// at is the maps at a moment.
func at(t *testing.T, mapper named.Mapper, moment time.Duration) []named.Map {
	t.Helper()
	maps, err := mapper.At(moment)
	if err != nil {
		t.Fatal(err)
	}
	return maps
}

// The weather makes its three maps in order, named as cmd/weather names
// them by its clock, and a film shows its bands by default.
func TestTheWeatherMakesThreeMapsNamedByItsClock(t *testing.T) {
	scene := weatherMaps(t)
	if !slices.Equal(scene.Pictures(), []string{"spun", "bands", "aged"}) {
		t.Errorf("pictures %v", scene.Pictures())
	}
	mapper := built(t, "")
	maps := at(t, mapper, twoHours)
	want := []string{"spun-00007200", "bands-00007200", "aged-00007200"}
	for i, one := range maps {
		if one.Name() != want[i] || one.Kind() != scene.Pictures()[i] ||
			one.Picture().Bounds() != image.Rect(0, 0, tinyAcross,
				tinyRows) {
			t.Errorf("map %d: %s %s %v", i, one.Kind(), one.Name(),
				one.Picture().Bounds())
		}
	}
	if mapper.Shows() != "bands" || mapper.Renderer() != named.CPU {
		t.Errorf("shows %s on %s", mapper.Shows(), mapper.Renderer())
	}
}

// A moment reached through the scene is the weather package's own, run
// straight from still air with the same values, byte for byte, on the
// cpu: through the cache and without it.
func TestAMomentIsTheWeathersOwn(t *testing.T) {
	grid, err := weather.NewGrid(tinyAcross, tinyRows,
		commandLimit*math.Pi/halfTurn, commandRadius)
	if err != nil {
		t.Fatal(err)
	}
	stirring, err := weather.NewStirring(commandRate, commandWidth,
		commandSpin)
	if err != nil {
		t.Fatal(err)
	}
	sky, err := weather.New(grid, commandDay, commandDrag, commandRelax,
		stirring, commandSeed)
	if err != nil {
		t.Fatal(err)
	}
	sky.Run(twoHours.Seconds())
	want := [][]byte{sky.Spun().Pix,
		sky.Bands(commandBands * math.Pi / halfTurn).Pix,
		sky.Aged(commandCooking).Pix}
	for _, cache := range []string{"", t.TempDir()} {
		for i, one := range at(t, built(t, cache), twoHours) {
			if !bytes.Equal(one.Picture().Pix, want[i]) {
				t.Errorf("cache %q: %s differs from the weather's own",
					cache, one.Name())
			}
		}
	}
}

// A moment reached twice is the same; so is one reached after a later
// one, which starts again from still air, and one reached again through
// the cache by a weather built anew.
func TestAMomentReachedTwiceIsTheSame(t *testing.T) {
	cache := t.TempDir()
	mapper := built(t, cache)
	first := at(t, mapper, twoHours)
	again := at(t, mapper, twoHours)
	at(t, mapper, 2*twoHours)
	back := at(t, mapper, twoHours)
	anew := at(t, built(t, cache), twoHours)
	for i := range first {
		for _, other := range [][]named.Map{again, back, anew} {
			if !bytes.Equal(first[i].Picture().Pix,
				other[i].Picture().Pix) || first[i].Name() != other[i].Name() {
				t.Errorf("%s reached again is %s, and differs",
					first[i].Name(), other[i].Name())
			}
		}
	}
}

// anti-test: two moments a step apart are not the same picture.
func TestTwoMomentsDiffer(t *testing.T) {
	mapper := built(t, "")
	early := at(t, mapper, twoHours)
	late := at(t, mapper, 50*twoHours)
	if bytes.Equal(early[0].Picture().Pix, late[0].Picture().Pix) {
		t.Error("the spin a hundred hours on is the spin at two")
	}
}

// nothingSpaced is defined: no spacing between a cache's moments.
const nothingSpaced = 0

// anti-tests: each parameter's near miss is refused by name, before the
// weather is built.
func TestTheWeathersNearMissesAreRefused(t *testing.T) {
	for _, miss := range []struct {
		key   string
		value any
		want  error
	}{
		{"across", 15, named.ErrBounds},
		{"across", 2, named.ErrBounds},
		{"rows", 3, named.ErrBounds},
		{"limit", 90.0, named.ErrBounds},
		{"limit", 0.0, named.ErrBounds},
		{"radius", 0.0, named.ErrBounds},
		{"day", -1.0, named.ErrBounds},
		{"drag", 0.0, named.ErrBounds},
		{"relax", 0.0, named.ErrBounds},
		{"rate", 0.0, named.ErrBounds},
		{"storm-width", 0.0, named.ErrBounds},
		{"strength", 0.0, named.ErrBounds},
		{"bands", 0.0, named.ErrBounds},
		{"seed", -1, named.ErrBounds},
		{"show", "band", named.ErrChoice},
		{"pace", "0s", named.ErrBounds},
		{"pace", 3600, named.ErrKind},
		{"wide", commandWidth, named.ErrNoParameter},
	} {
		with := tiny()
		with[miss.key] = miss.value
		mapper, err := weatherMaps(t).Build(with, onCPU(t, ""))
		if !errors.Is(err, miss.want) || mapper != nil ||
			!strings.Contains(err.Error(), miss.key) {
			t.Errorf("%s %v: %v, %v", miss.key, miss.value, mapper, err)
		}
	}
}

// anti-tests: a renderer the weather is not carried by is refused, and
// so is a cache with no spacing between its moments.
func TestTheWeatherRefusesWhatItCannotDo(t *testing.T) {
	ask, err := named.AskMaps("gpu-native", true, "", nothingSpaced, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := weatherMaps(t).Build(tiny(), ask); !errors.Is(err,
		named.ErrRenderer) {
		t.Errorf("gpu-native: %v", err)
	}
	if _, err := named.AskMaps(named.CPU, true, t.TempDir(), nothingSpaced,
		nil); !errors.Is(err, named.ErrEvery) {
		t.Errorf("a cache spaced by nothing: %v", err)
	}
	if weatherMaps(t).Check(map[string]any{"across": 16}) != nil {
		t.Error("a good value does not check")
	}
}

// Each kind finds its own scenes by name and refuses the other's,
// saying which kind the name is.
func TestEachKindRefusesTheOthersNames(t *testing.T) {
	if _, err := named.Get("weather"); !errors.Is(err, named.ErrNoScene) ||
		!strings.Contains(err.Error(), "drawn as maps") {
		t.Errorf("Get weather: %v", err)
	}
	if _, err := named.GetMaps("orbital"); !errors.Is(err,
		named.ErrNoScene) || !strings.Contains(err.Error(), "traced") {
		t.Errorf("GetMaps orbital: %v", err)
	}
	if _, err := named.GetMaps("wether"); !errors.Is(err,
		named.ErrNoScene) || !strings.Contains(err.Error(), "weather") {
		t.Errorf("GetMaps wether: %v", err)
	}
	scene := weatherMaps(t)
	if scene.About() == "" || !scene.Draws(named.GPU) ||
		!slices.Equal(scene.Renderers(), []named.Renderer{named.GPU,
			named.CPU}) {
		t.Errorf("weather says %q, %v", scene.About(), scene.Renderers())
	}
}

// The weather's cache is the one hillside keeps: the weather folder
// under SLAYER_CACHE.
func TestTheWeathersCacheIsHillsides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SLAYER_CACHE", dir)
	want := filepath.Join(dir, "weather")
	if got, err := weatherMaps(t).Cache(); err != nil || got != want {
		t.Errorf("cache %q, %v; want %q", got, err, want)
	}
}

// anti-test: with SLAYER_CACHE unset the weather's cache is refused by
// name, and nothing is made under the renders home.
func TestTheWeathersCacheIsRefusedWithNoHome(t *testing.T) {
	renders := t.TempDir()
	t.Setenv("SLAYER_CACHE", "")
	t.Setenv("SLAYER_RENDERS", renders)
	got, err := weatherMaps(t).Cache()
	if err == nil || !strings.Contains(err.Error(), "SLAYER_CACHE") {
		t.Errorf("cache %q, %v; want a refusal naming SLAYER_CACHE", got,
			err)
	}
	if found, _ := os.ReadDir(renders); len(found) != 0 {
		t.Errorf("the renders home was written to: %v", found)
	}
}
