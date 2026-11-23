package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// recorder is a launcher that keeps what it was given and opens nothing.
type recorder struct {
	calls int
	got   options
}

// launch keeps one run's options.
func (kept *recorder) launch(settings options) error {
	kept.calls++
	kept.got = settings
	return nil
}

// runTool runs the command line in process, and says the status, what it
// wrote and what the recorder got.
func runTool(words ...string) (int, string, *recorder) {
	kept := &recorder{}
	var out bytes.Buffer
	status := tool(kept.launch).RunWith(words, strings.NewReader(""), &out, &out)
	return status, out.String(), kept
}

// Example_window shows the first line of the window verb's help.
func Example_window() {
	var out bytes.Buffer
	tool((&recorder{}).launch).RunWith([]string{"window", "--help"},
		strings.NewReader(""), &out, &out)
	fmt.Println(strings.SplitN(out.String(), "\n", 2)[0])
	// Output: grissom window [ARGS]: open the viewer's window
}

// Example_draw shows what a draw says it would write, and writes none.
func Example_draw() {
	tool((&recorder{}).launch).RunWith([]string{"draw", "--shot", "out/check",
		"--view", "-1", "--dry-run"}, strings.NewReader(""), os.Stdout, os.Stderr)
	// Output:
	// would write out/check-sky.png
	// would write out/check-earth.png
	// would write out/check-standing.png
}

func TestTheVerbsAndFlagsAreDeclaredSoundly(t *testing.T) {
	if err := tool((&recorder{}).launch).Examples(os.DirFS(".")); err != nil {
		t.Fatal(err)
	}
}

func TestEveryFlagIsParsedToItsValue(t *testing.T) {
	status, said, kept := runTool("draw", "--data", "d", "--earth", "e",
		"--sky", "s", "--depth", "5", "--earth-depth", "4", "--wide-depth", "3",
		"--fov", "2.5", "--look", "-10.5,20", "--colourways", "c",
		"--colourway", "dawn", "--ground", "2", "--chroma", "1",
		"--earth-image", "1", "--shot", "out/x", "--view", "2",
		"--distance", "4.2", "--earth-only", "--no-hud", "--weather", "w.nc",
		"--weather-var", "msl", "--weather-time", "4")
	want := options{data: "d", earth: "e", sky: "s", look: "-10.5,20",
		colourways: "c", colourway: "dawn", shot: "out/x", depth: 5,
		earthDepth: 4, wideDepth: 3, view: 2, ground: 2, chroma: 1,
		earthImage: 1, fov: 2.5, distance: 4.2, earthOnly: true, noHud: true,
		weather: "w.nc", weatherVar: "msl", weatherTime: 4}
	if status != 0 || kept.calls != 1 || kept.got != want {
		t.Fatalf("status %d, %d calls, got %+v: %s", status, kept.calls, kept.got, said)
	}
}

func TestAWindowWithNoFlagsGetsTodaysDefaults(t *testing.T) {
	status, said, kept := runTool("window")
	want := options{colourways: "colourways", colourway: "twilight",
		depth: 13, earthDepth: 7, wideDepth: 8, fov: 1, distance: 3.2,
		weatherVar: "tcc", weatherTime: firstWithData}
	if status != 0 || kept.got != want {
		t.Fatalf("status %d, got %+v: %s", status, kept.got, said)
	}
}

func TestHelpNamesEveryFlagWithTwoDashesAndNoOneDashLongFlag(t *testing.T) {
	oneDash := regexp.MustCompile(`(^|\s)-[a-z]{2,}`)
	shared := []string{"data", "earth", "sky", "depth", "earth-depth",
		"wide-depth", "fov", "look", "colourways", "colourway", "ground",
		"chroma", "earth-image", "distance", "earth-only", "no-hud",
		"weather", "weather-var", "weather-time"}
	for verb, own := range map[string][]string{"window": nil, "draw": {"shot", "view"},
		"film": {"into", "frames", "turns", "tilt"}} {
		status, page, _ := runTool(verb, "--help")
		if status != 0 {
			t.Fatalf("%s --help gave %d: %s", verb, status, page)
		}
		for _, name := range append(append([]string{}, shared...), own...) {
			if !strings.Contains(page, "--"+name+" ") {
				t.Errorf("%s help does not name --%s:\n%s", verb, name, page)
			}
		}
		if oneDash.MatchString(page) {
			t.Errorf("%s help names a one-dash long flag:\n%s", verb, page)
		}
	}
}

func TestOnlyADrawHasShotAndView(t *testing.T) {
	status, said, kept := runTool("window", "--shot", "out/x")
	if status == 0 || kept.calls != 0 || !strings.Contains(said, "unknown flag: --shot") {
		t.Fatalf("window took --shot: %d %s", status, said)
	}
}

func TestTheParserRefusesWhatItDoesNotKnow(t *testing.T) {
	for _, test := range []struct {
		name  string
		words []string
		want  string
	}{
		{"unknown flag", []string{"window", "--nonsense"}, "unknown flag: --nonsense"},
		{"one dash long", []string{"window", "-depth", "3"}, "unknown shorthand flag"},
		{"flag without value", []string{"window", "--depth"}, "flag needs an argument"},
		{"unknown verb", []string{"fly"}, "there is no verb fly"},
		{"flag twice", []string{"window", "--depth", "3", "--depth", "4"}, "twice"},
		{"value for a switch", []string{"window", "--earth-only=yes"}, "a switch takes no value"},
		{"value for another switch", []string{"window", "--no-hud=1"}, "a switch takes no value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, said, kept := runTool(test.words...)
			if status != 2 || kept.calls != 0 || !strings.Contains(said, test.want) {
				t.Fatalf("status %d, %d calls, want %q: %s", status, kept.calls, test.want, said)
			}
		})
	}
}

func TestAValueOfTheWrongKindIsRefusedBeforeTheViewerOpens(t *testing.T) {
	for _, test := range []struct{ name, flag, want string }{
		{"depth", "--depth=deep", "--depth wants a whole number"},
		{"fov", "--fov=wide", "--fov wants a number"},
		{"ground", "--ground=1.5", "--ground wants a whole number"},
		{"distance", "--distance=far", "--distance wants a number"},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, said, kept := runTool("window", test.flag)
			if status != 1 || kept.calls != 0 || !strings.Contains(said, test.want) {
				t.Fatalf("status %d, %d calls: %s", status, kept.calls, said)
			}
		})
	}
}

func TestEarthOnlyNeedsTheEarthAndSaysSo(t *testing.T) {
	for _, verb := range []string{"window", "draw"} {
		words := []string{verb, "--earth-only", "--shot", "x"}
		if verb == "window" {
			words = words[:2]
		}
		status, said, kept := runTool(words...)
		if status != 1 || kept.calls != 0 || !strings.Contains(said, "--earth-only needs --earth") {
			t.Fatalf("%s: status %d, %d calls: %s", verb, status, kept.calls, said)
		}
	}
}

func TestADistanceInsideTheEarthIsRefused(t *testing.T) {
	for _, distance := range []string{"0", "-2", "1", "0.5"} {
		status, said, kept := runTool("window", "--distance="+distance)
		if status != 1 || kept.calls != 0 || !strings.Contains(said, "must be more than 1") {
			t.Fatalf("distance %s: status %d, %d calls: %s", distance, status, kept.calls, said)
		}
	}
	status, said, kept := runTool("window", "--distance", "1.5")
	if status != 0 || kept.got.distance != 1.5 {
		t.Fatalf("distance 1.5: status %d, got %v: %s", status, kept.got.distance, said)
	}
}

func TestEarthOnlyDrawsTheEarthUnlessAskedForAnotherViewWithNoSky(t *testing.T) {
	status, said, kept := runTool("draw", "--earth-only", "--earth", "e", "--shot", "x")
	if status != 0 || kept.got.view != earthView {
		t.Fatalf("default view: status %d, view %d: %s", status, kept.got.view, said)
	}
	status, _, kept = runTool("draw", "--earth-only", "--earth", "e", "--shot", "x", "--view", "2")
	if status != 0 || kept.got.view != placeView {
		t.Fatalf("standing view: status %d, view %d", status, kept.got.view)
	}
	for _, view := range []string{"0", "-1"} {
		status, said, kept = runTool("draw", "--earth-only", "--earth", "e", "--shot", "x", "--view="+view)
		if status != 1 || kept.calls != 0 || !strings.Contains(said, "--earth-only has no sky") {
			t.Fatalf("view %s: status %d, %d calls: %s", view, status, kept.calls, said)
		}
	}
	status, said, _ = runTool("draw", "--earth-only", "--earth", "e", "--shot", "x", "--dry-run")
	if status != 0 || said != "would write x-earth.png\n" {
		t.Fatalf("dry run: status %d: %q", status, said)
	}
}

func TestADrawNeedsAShotAndAViewThatExists(t *testing.T) {
	for _, test := range []struct {
		name  string
		words []string
		want  string
	}{
		{"no shot", []string{"draw"}, "draw needs --shot"},
		{"empty shot", []string{"draw", "--shot="}, "draw needs --shot"},
		{"view too high", []string{"draw", "--shot", "x", "--view", "3"}, "must be -1, 0, 1 or 2"},
		{"view too low", []string{"draw", "--shot", "x", "--view=-2"}, "must be -1, 0, 1 or 2"},
		{"negative depth", []string{"window", "--earth-depth=-1"}, "cannot be less than 0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, said, kept := runTool(test.words...)
			if status != 1 || kept.calls != 0 || !strings.Contains(said, test.want) {
				t.Fatalf("status %d, %d calls, want %q: %s", status, kept.calls, test.want, said)
			}
		})
	}
}
