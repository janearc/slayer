//go:build draw

package hillside

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
)

// giantHeader is a set of melancholia's giant alone, in space, lit from
// the side, seen from a little above its equator through a pinhole, so a
// frame is one instant, at an exposure said, since only a rig works one
// out, before its planet block.
const giantHeader = `scene:
  name: a giant's weather
  light: sun
  sun:
    azimuth: 90
    altitude: 0
  film:
    width: 24
    rays: 1
    wide: 50
    exposure: 1
  cameras:
    afar:
      title: "the giant, whole"
      orientation:
        point_to: the giant
        alpha: 0
      location:
        loc_relative:
          object: the giant
          standoff_x: 0
          standoff_y: -1.6e8
          standoff_z: 0
`

// tinyWeather is a weather block small enough for a test: the least grid
// the planet reads, reached an hour on, kept every hour, an hour of
// weather a second of scene time, a quarter turn a second. Picked by the
// author of this file, 2026-11-12.
const tinyWeather = `    weather:
      cells: 32
      day: 18000
      drag: 1.0e7
      relax: 1.0e6
      storms: {rate: 3.8e-17, wide: 1.6e6, strength: 1.0e-5}
      seed: 7
      moment: 3600
      every: 3600
%s`

// giantSet is a set file of melancholia's giant with a weather block of
// its own and no rings, written beside a copy of the sources file, and
// its path.
func giantSet(t *testing.T, filmLines string) string {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("..", "sets", "melancholia.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var block []string
	skipping := false
	for _, line := range giantBlock(t, string(text)) {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 4 && strings.TrimSpace(line) != "" {
			key := strings.SplitN(strings.TrimSpace(line), ":", 2)[0]
			skipping = slices.Contains([]string{"sky", "weather", "rings",
				"ringlets", "xrays"}, key)
			if key == "pole" {
				line, skipping = "    pole: {azimuth: 180, altitude: 80}",
					false
			}
		}
		if !skipping {
			block = append(block, line)
		}
	}
	directory := t.TempDir()
	sources, err := os.ReadFile(filepath.Join("..", "sources.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "sources.yaml"),
		sources, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "giant.yaml")
	body := giantHeader + strings.Join(block, "\n") + "\n" +
		fmt.Sprintf(tinyWeather, filmLines)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// giantBlock is the planet block of a set's text, to the scene's next
// key.
func giantBlock(t *testing.T, text string) []string {
	t.Helper()
	lines := strings.Split(text, "\n")
	first, last := -1, len(lines)
	for at, line := range lines {
		next := strings.HasPrefix(line, "  ") &&
			!strings.HasPrefix(line, "   ") && !strings.HasPrefix(line, "  #")
		switch {
		case line == "  planet:":
			first = at
		case next && first >= 0 && at > first && last == len(lines):
			last = at
		}
	}
	if first < 0 {
		t.Fatal("sets/melancholia.yaml has no planet block")
	}
	return lines[first:last]
}

// giantJob is a job of a set at a small size on the cpu, as a still at
// nothing or a film of two frames a second apart.
func giantJob(t *testing.T, path string, film bool) job.Job {
	t.Helper()
	timing := "at: {value: 0, unit: s}\n"
	if film {
		timing = "fps: 1\nduration: {value: 2, unit: s}\n"
	}
	return jobFrom(t, fmt.Sprintf("scene: {set: %s}\nfilm: {width: 24, "+
		"height: 24, rays: 1}\nrenderer: cpu\n%s", path, timing))
}

// lightOf is a session's frame's light, the session opened from a job.
func lightOf(t *testing.T, wanted job.Job, index int) []float32 {
	t.Helper()
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	drawn, err := session.Frame(context.Background(), index, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	var numbers []float32
	for _, pixel := range drawn.Light().Pix {
		numbers = append(numbers, pixel.R, pixel.G, pixel.B)
	}
	return numbers
}

// A planet alone, with no ground, opens as a film, and its first frame
// is its still, its weather at the set's moment and unturned.
func TestAGroundlessPlanetFilmsAndItsFirstFrameIsItsStill(t *testing.T) {
	t.Setenv("SLAYER_WEATHER", t.TempDir())
	path := giantSet(t, "      pace: 3600\n      turning: 90\n")
	still := lightOf(t, giantJob(t, path, false), 0)
	first := lightOf(t, giantJob(t, path, true), 0)
	if !slices.Equal(still, first) {
		t.Error("the film's first frame is not its still")
	}
	later := lightOf(t, giantJob(t, path, true), 1)
	if slices.Equal(first, later) {
		t.Error("a second on, an hour of weather and a quarter turn " +
			"changed nothing")
	}
}

// anti-test: a set whose weather says no pace and no turning draws every
// frame of a film as its still.
func TestAWeatherWithNoPaceHoldsItsMoment(t *testing.T) {
	t.Setenv("SLAYER_WEATHER", t.TempDir())
	path := giantSet(t, "")
	first := lightOf(t, giantJob(t, path, true), 0)
	later := lightOf(t, giantJob(t, path, true), 1)
	if !slices.Equal(first, later) {
		t.Error("a weather that says no pace moved")
	}
}

// anti-test: a world with nothing to stand on is refused as a film, by
// name; the world half draws it as a still.
func TestAWorldAloneIsNotFilmed(t *testing.T) {
	absolute, err := filepath.Abs(sessionWidow)
	if err != nil {
		t.Fatal(err)
	}
	wanted := jobFrom(t, fmt.Sprintf("scene: {set: %s}\nfilm: {width: 8, "+
		"height: 8, rays: 1}\nrenderer: cpu\nfps: 1\n"+
		"duration: {value: 1, unit: s}\n", absolute))
	if _, err := Open(context.Background(), wanted, Stamp{}); err == nil ||
		!strings.Contains(err.Error(), "the world half") {
		t.Errorf("a world alone as a film: %v", err)
	}
}

// A film asked for a frame before one it has drawn starts its weather
// again from still air, so the earlier frame is as a film from the start
// draws it: the weather never runs backward.
func TestAnEarlierFrameStartsTheWeatherAgain(t *testing.T) {
	t.Setenv("SLAYER_WEATHER", t.TempDir())
	path := giantSet(t, "      pace: 3600\n      turning: 90\n")
	still := lightOf(t, giantJob(t, path, false), 0)
	session, err := Open(context.Background(), giantJob(t, path, true),
		Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, index := range []int{1, 0} {
		drawn, err := session.Frame(context.Background(), index, Ask{})
		if err != nil {
			t.Fatal(err)
		}
		if index != 0 {
			continue
		}
		var numbers []float32
		for _, pixel := range drawn.Light().Pix {
			numbers = append(numbers, pixel.R, pixel.G, pixel.B)
		}
		if !slices.Equal(numbers, still) {
			t.Error("frame 0 drawn after frame 1 is not the still")
		}
	}
}
