package door

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	scenes "github.com/janearc/slayer/scene/named"
)

// weatherJob is a job of the tests' small weather on the cpu, a scene
// second an hour of its own, with more lines after it.
func weatherJob(t *testing.T, with, after string) string {
	t.Helper()
	return jobLine + fmt.Sprintf("scene: {named: weather, with: "+
		"{across: %d, rows: %d, pace: 1h%s}}\n", number(t, "weather-across"),
		number(t, "weather-rows"), with) + after
}

// weatherFilm is a two-second film of the small weather at one frame a
// second, on the cpu, so its frames are an hour of weather apart.
func weatherFilm(t *testing.T, with, after string) string {
	t.Helper()
	return weatherJob(t, with, "renderer: cpu\nfps: 1\n"+
		"duration: {value: 2, unit: s}\n"+after)
}

// cached keeps the weather's checkpoints under a cache home of the
// test's own.
func cached(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SLAYER_CACHE", dir)
	return dir
}

// mapsAt is the small weather's maps at a moment, on the cpu, built
// through the registry as a caller other than the door would.
func mapsAt(t *testing.T, moment time.Duration) []scenes.Map {
	t.Helper()
	scene, err := scenes.GetMaps("weather")
	if err != nil {
		t.Fatal(err)
	}
	ask, err := scenes.AskMaps(scenes.CPU, true, "", nothing, nil)
	if err != nil {
		t.Fatal(err)
	}
	built, err := scene.Build(map[string]any{
		"across": number(t, "weather-across"),
		"rows":   number(t, "weather-rows"), "pace": "1h"}, ask)
	if err != nil {
		t.Fatal(err)
	}
	maps, err := built.At(moment)
	if err != nil {
		t.Fatal(err)
	}
	return maps
}

// the names of the small weather's maps at its first two frames, as
// cmd/weather names them
var weatherNames = []string{
	"aged-00000000.png", "aged-00003600.png",
	"bands-00000000.png", "bands-00003600.png",
	"manifest.json",
	"spun-00000000.png", "spun-00003600.png",
}

// A film of the weather writes three maps a frame under the command's
// names, each the weather's own at that moment, byte for byte.
func TestDrawWeatherWritesThreeMapsAFrame(t *testing.T) {
	cached(t)
	directory := t.TempDir()
	path := written(t, directory, "sky.yaml", weatherFilm(t, "",
		"outputs: {frames: maps}\n"))
	env := envIn(t, context.Background(), directory)
	said := drew(t, env, path)
	folder := filepath.Join(directory, "maps")
	if got := files(t, folder); !slices.Equal(got, weatherNames) {
		t.Errorf("the frames folder holds %v", got)
	}
	for _, one := range mapsAt(t, time.Second) {
		file := filepath.Join(folder, one.Name()+".png")
		got := pictureOf(t, file)
		if !bytes.Equal(got.Pix, one.Picture().Pix) {
			t.Errorf("%s is not the weather's own", one.Name())
		}
	}
	if !strings.Contains(said, "frame 2 of 2: spun-00003600, on the cpu") {
		t.Errorf("the frames are not said: %q", said)
	}
}

// A second run of the same film reaches its frames from the cache and
// writes the same maps; a changed job draws its maps again over the
// earlier version's, and says so.
func TestDrawWeatherResumesAndRedrawsAChangedJob(t *testing.T) {
	cached(t)
	directory := t.TempDir()
	path := written(t, directory, "sky.yaml", weatherFilm(t, "",
		"outputs: {frames: maps}\n"))
	env := envIn(t, context.Background(), directory)
	drew(t, env, path)
	spun := filepath.Join(directory, "maps", "spun-00003600.png")
	first := pictureOf(t, spun)
	said := drew(t, env, path)
	if !strings.Contains(said, "3600 s: 0 steps") ||
		!bytes.Equal(first.Pix, pictureOf(t, spun).Pix) {
		t.Errorf("the second run did not reach its frames from the "+
			"cache: %q", said)
	}
	// anti-test of TestDrawWeatherResumesAndRedrawsAChangedJob
	written(t, directory, "sky.yaml", weatherFilm(t, ", seed: 8",
		"outputs: {frames: maps}\n"))
	said = drew(t, env, path)
	if !strings.Contains(said, "earlier version of this job") ||
		bytes.Equal(first.Pix, pictureOf(t, spun).Pix) {
		t.Errorf("the changed job's maps were not drawn again: %q", said)
	}
}

// A still of the weather writes the map it shows to its picture.
func TestDrawWeatherStillIsTheMapItShows(t *testing.T) {
	cached(t)
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", weatherJob(t,
		", show: aged", "renderer: cpu\nat: {value: 1, unit: s}\n"+
			"outputs: {picture: aged.png}\n"))
	env := envIn(t, context.Background(), directory)
	drew(t, env, path)
	want := mapsAt(t, time.Second)[2]
	aged := pictureOf(t, filepath.Join(directory, "aged.png"))
	if want.Kind() != "aged" || !bytes.Equal(aged.Pix, want.Picture().Pix) {
		t.Error("the still is not the aged map")
	}
}

// what a scene of maps cannot be drawn with is refused by
// name, before anything is written; the plain film draws.
// anti-test of TestDrawWeatherWritesThreeMapsAFrame
func TestDrawWeatherRefusesWhatMapsCannotTake(t *testing.T) {
	cached(t)
	for _, miss := range []struct{ line, words string }{
		{"film: {width: 64, height: 32}\n", "a film size"},
		{"film: {rays: 2}\n", "rays"},
		{"film: {wide: 40}\n", "a lens"},
		{"film: {exposure: 1}\n", "an exposure"},
		{"quality: preview\n", "a quality"},
		{"print: {knee: 0.5, top: 4}\n", "a print"},
		{"stock: {name: ash}\n", "a stock"},
		{"shutter: {angle: {value: 180, unit: deg}, glimpses: 2}\n",
			"a shutter"},
	} {
		directory := t.TempDir()
		path := written(t, directory, "sky.yaml", weatherFilm(t, "",
			miss.line+"outputs: {frames: maps}\n"))
		env := envIn(t, context.Background(), directory)
		got := running(Tool(env),
			"draw", path)
		if got.code == nothing || !strings.Contains(got.wrongSaid,
			"cannot be drawn with "+miss.words) {
			t.Errorf("%q: %d %q", miss.line, got.code, got.wrongSaid)
		}
		if len(files(t, directory)) != one {
			t.Errorf("%q wrote %v", miss.line, files(t, directory))
		}
	}
}

// a renderer the weather is not carried by, and a parameter
// it has not, are refused by name.
// anti-test of TestDrawWeatherStillIsTheMapItShows
func TestDrawWeatherRefusesItsNearMisses(t *testing.T) {
	cached(t)
	for _, miss := range []struct{ job, words string }{
		{strings.Replace(weatherFilm(t, "", ""), "renderer: cpu",
			"renderer: gpu-native", one), "not carried by \"gpu-native\""},
		{weatherFilm(t, ", storms: 3", ""), "has no \"storms\""},
	} {
		directory := t.TempDir()
		path := written(t, directory, "sky.yaml", miss.job)
		env := envIn(t, context.Background(), directory)
		got := running(Tool(env),
			"draw", path)
		if got.code == nothing || !strings.Contains(got.wrongSaid,
			miss.words) {
			t.Errorf("%d %q, want %q", got.code, got.wrongSaid, miss.words)
		}
	}
}

// A frames spacing is a film's span over its frames, a still's moment,
// or the lonely spacing for a still at nothing.
func TestSpacing(t *testing.T) {
	for _, each := range []struct {
		times []time.Duration
		want  time.Duration
	}{
		{[]time.Duration{time.Second, 3 * time.Second, 5 * time.Second},
			2 * time.Second},
		{[]time.Duration{time.Minute}, time.Minute},
		// anti-test of TestSpacing
		{[]time.Duration{nothing}, lonelySpacing},
	} {
		if got := spacing(each.times); got != each.want {
			t.Errorf("%v: %s, want %s", each.times, got, each.want)
		}
	}
}
