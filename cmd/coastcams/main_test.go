package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/set"
)

// likeEarthSet is a set with one generated planet like the Earth, by the
// amount of ice (its water) the test gives it.
const likeEarthSet = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: coast cameras test
  seed: 7
  setting: space
  light: sun
  vibes: on
  sun: {azimuth: 225, altitude: 20}
  film: {width: 80, height: 80, rays: 1, exposure: 0.003, wide: 50}
  cameras:
    portrait:
      location: {at: [-23317959, 0, 8487043]}
      orientation: {look_at: [0, 0, 0]}
  world:
    unit: 1
    bodies:
      body:
        generate:
          mass: 1
          makeup: {metal: 0.32, rock: 0.6798, ice: ICE, gas: 0}
          day: 23.93
          tilt: 23.44
          distance: 1
          sun: {mass: 1, luminosity: 1}
          age: 4.5e9
          seed: 1
        middle: [0, 0, 0]
        detail: 5
`

// run runs the tool with args and gives its exit code and output.
func run(args ...string) (int, string) {
	var out bytes.Buffer
	code := tool().Run(args, &out, &out)
	return code, out.String()
}

// setFile writes a set with the given ice into a folder of its own.
func setFile(t *testing.T, ice, rock string) string {
	t.Helper()
	text := strings.ReplaceAll(likeEarthSet, "ICE", ice)
	text = strings.ReplaceAll(text, "0.6798", rock)
	file := filepath.Join(t.TempDir(), "set.yaml")
	if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

// Example_cameras shows a direction that is not three numbers refused.
// The cameras themselves are not shown: they move with the terrain.
func Example_cameras() {
	tool().Run([]string{"cameras", "set.yaml", "--from", "1,0"}, os.Stdout,
		os.Stdout)
	// Output:
	// coastcams cameras: cameras: --from "1,0" is not x,y,z
}

// The tool's verbs, flags and examples are as pzm requires.
func TestToolIsDeclaredWhole(t *testing.T) {
	if err := tool().Examples(os.DirFS(".")); err != nil {
		t.Error(err)
	}
}

// An Earth like set prints both cameras, each with a place and a look_at.
func TestCamerasPrintsBothCameras(t *testing.T) {
	code, out := run("cameras", setFile(t, "0.0002", "0.6798"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, want := range []string{"coast-low:", "coast-high:", "location:",
		"look_at:"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in\n%s", want, out)
		}
	}
}

// placeTolerance is how far, in metres, a printed camera may stand from
// where it was found: the YAML is printed to the metre.
const placeTolerance = 1

// Anti-test: the cameras printed, pasted into the set and read back the
// way a draw reads them, stand and look where the finder put them, not a
// quarter turn about x away, as they did while the world's coordinates
// were printed where a set file wants a map's.
func TestPrintedCamerasReadBackWhereFound(t *testing.T) {
	file := setFile(t, "0.0002", "0.6798")
	code, out := run("cameras", file)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	planet, middle, detail, err := planetOf(file, "")
	if err != nil {
		t.Fatal(err)
	}
	found, err := generate.FindCoast(planet, detail, maths.XAxis,
		defaultHeight)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := os.ReadFile(file)
	pasted := strings.Replace(string(source), "  cameras:\n",
		"  cameras:\n"+out, 1)
	read, err := set.Read(file, []byte(pasted))
	if err != nil {
		t.Fatalf("%v\n%s", err, pasted)
	}
	for name, want := range map[string]generate.Camera{
		"coast-low": found.Low, "coast-high": found.High} {
		shot, err := read.Shot(name, nil)
		if err != nil {
			t.Fatal(err)
		}
		at, look := shot.Eye().Origin, shot.Focus()
		if at.Sub(want.At.Add(middle)).Len() > placeTolerance ||
			look.Sub(want.LookAt.Add(middle)).Len() > placeTolerance {
			t.Errorf("%s reads back at %v looking at %v, found at %v "+
				"looking at %v", name, at, look, want.At, want.LookAt)
		}
	}
}

// Anti-test: a set whose planet has no sea prints no camera at all, and
// says there is no coast.
func TestADrySetPrintsNoCameras(t *testing.T) {
	code, out := run("cameras", setFile(t, "0", "0.6800"))
	if code == 0 || !strings.Contains(out, "no coast") {
		t.Errorf("exit %d for a dry set: %s", code, out)
	}
	if strings.Contains(out, "coast-low:") {
		t.Errorf("a dry set prints a camera:\n%s", out)
	}
}

// Anti-test: a direction that is not three numbers, a body the set does
// not have, and no set at all are refused, and print no camera.
func TestCamerasRefusesWhatItCannotRead(t *testing.T) {
	file := setFile(t, "0.0002", "0.6798")
	for _, args := range [][]string{
		{"cameras", file, "--from", "1,0"},
		{"cameras", file, "--from", "a,b,c"},
		{"cameras", file, "--body", "nobody"},
		{"cameras", file, "--height", "tall"},
		{"cameras"},
	} {
		code, out := run(args...)
		if code == 0 || strings.Contains(out, "coast-low:") {
			t.Errorf("%v: exit %d, %s", args, code, out)
		}
	}
}
