package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// childEnv, when set to one, makes the test binary run the command's main
// and exit, so the paths that stop the process can be tried in a child.
const childEnv = "LIGHTSHOW_TEST_CHILD"

// controls are the names the command reads from the environment. Several
// are read once, when the package starts, so a test cannot unset them
// after the fact.
var controls = []string{
	"BEAM_APEX", "BEAM_GAIN", "BEAM_HALO", "BEAM_OUT", "BEAM_RADIUS", "CUES",
	"DEBUGMASK", "FIRE_BURST", "FIRE_BURST_GAIN", "FIRE_DECAY", "FIRE_FLASH",
	"FIRE_FPS", "FIRE_FRAMES", "FIRE_GAIN", "FIRE_PULSE", "FIRE_SNAP",
	"FIRE_STEADY_FROM", "FIRE_SWELL", "FIRE_SWELL_FALL", "FIRE_TOP",
	"FLOOD_GAIN", "FLOOD_PASSES", "HAZE_GAIN", "MOON_DISC", "MOON_DISC_ALT",
	"MOONLAKE_EYE", "MOONLAKE_FOCAL", "MOONLAKE_FRAME_HEIGHT", "MOONLAKE_LOOK",
	"MOONLAKE_MOON_ALT", "MOONLAKE_MOON_SCALE", "OVERHEAD_GAIN", "PASSES",
	"RIG_DECK", "SEARCH_COUNT", "SEARCH_GAIN", "SKY_GLOW", "SPOT_GAIN",
	"STOPS", "TREEMASK", "WATERMASK",
}

// quiet is the environment given with every control of the command left
// out, so a setting in the caller's shell cannot steer a test.
func quiet(environment []string) []string {
	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		named := false
		for _, control := range controls {
			named = named || name == control
		}
		if !named {
			kept = append(kept, entry)
		}
	}
	return kept
}

// TestMain runs the command itself in a child, and the tests otherwise.
// If the caller's environment sets any control, the tests are run again
// in a process without them, because the package has already read some.
// That second process's coverage is not counted.
func TestMain(m *testing.M) {
	if os.Getenv(childEnv) == "1" {
		main()
		os.Exit(0)
	}
	if clean := quiet(os.Environ()); len(clean) != len(os.Environ()) {
		again := exec.Command(os.Args[0], os.Args[1:]...)
		again.Env, again.Stdout, again.Stderr = clean, os.Stdout, os.Stderr
		if err := again.Run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// child runs the command in a child process with arguments and extra
// environment, and says its exit status and what it wrote to stderr. The
// child starts with none of the command's controls set but those given.
func child(t *testing.T, args []string, env ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	inherited := quiet(os.Environ())
	cmd.Env = append(inherited, childEnv+"=1")
	cmd.Env = append(cmd.Env, env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), stderr.String()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0, stderr.String()
}

// TestCommandStopsOnBadInput: a wrong count of arguments, a seconds that
// is not a number, a missing plate and a bad number in the environment
// each stop with the status and message the command promises.
func TestCommandStopsOnBadInput(t *testing.T) {
	dir := t.TempDir()
	plate := keep(t, dir, "plate.pfm", flat(2, 2, 1), -1)
	out := filepath.Join(dir, "out.pfm")
	good := []string{plate, plate, out, out}
	for _, c := range []struct {
		name    string
		args    []string
		env     []string
		status  int
		mention string
	}{
		{"no arguments", nil, nil, 2, "plate.pfm mask.pfm"},
		{"seconds", append(append([]string{}, good...), "soon"), nil, 1,
			"lightshow:"},
		{"plate", []string{filepath.Join(dir, "none.pfm"), plate, out, out, "1"},
			nil, 1, "lightshow:"},
		{"mask", []string{plate, filepath.Join(dir, "none.pfm"), out, out, "1"},
			nil, 1, "lightshow:"},
		{"cues", append(append([]string{}, good...), "1"),
			[]string{"CUES=" + filepath.Join(dir, "none")}, 1, "lightshow:"},
		{"focal", append(append([]string{}, good...), "1"),
			[]string{"MOONLAKE_FOCAL=wide"}, 2, "MOONLAKE_FOCAL=\"wide\" is not a number"},
		{"disc", append(append([]string{}, good...), "1"),
			[]string{"MOON_DISC=" + filepath.Join(dir, "none.pfm")}, 1,
			"lightshow:"},
		{"disc size", append(append([]string{}, good...), "1"),
			[]string{"MOON_DISC=" + keep(t, dir, "big.pfm", flat(3, 3, 1), -1)},
			1, "2x2"},
		{"trees", append(append([]string{}, good...), "1"),
			[]string{"TREEMASK=" + filepath.Join(dir, "none.pfm")}, 1,
			"lightshow:"},
		{"debug", append(append([]string{}, good...), "1"),
			[]string{"DEBUGMASK=" + filepath.Join(dir, "no", "such", "d.pfm")},
			1, "lightshow:"},
		{"water", append(append([]string{}, good...), "1"),
			[]string{"WATERMASK=" + filepath.Join(dir, "none.pfm")}, 1,
			"lightshow:"},
		{"passes", append(append([]string{}, good...), "1"),
			[]string{"PASSES=" + filepath.Join(dir, "none"),
				"CUES=" + cueFile(t, "flood_left=1\n")}, 1, "lightshow:"},
		{"overhead", append(append([]string{}, good...), "1"),
			[]string{"PASSES=" + t.TempDir(), "CUES=" + cueFile(t, "overhead=1\n")}, 1,
			"lightshow:"},
		{"sky", []string{plate, plate, filepath.Join(dir, "no", "s.pfm"), out,
			"1"}, nil, 1, "lightshow:"},
		{"front", []string{plate, plate, out, filepath.Join(dir, "no", "f.pfm"),
			"1"}, nil, 1, "lightshow:"},
	} {
		status, stderr := child(t, c.args, c.env...)
		if status != c.status || !strings.Contains(stderr, c.mention) {
			t.Errorf("%s: status %d, stderr %q; want %d and %q", c.name, status,
				stderr, c.status, c.mention)
		}
	}
}

// scenePFMs writes a 64 by 36 plate with a moon and a mask that shows sky
// above row 12, and says where they are.
func scenePFMs(t *testing.T, dir string) (plate, mask string) {
	t.Helper()
	p := flat(frameW, frameH, 0.001)
	for x := 30; x < 34; x++ {
		p.light[p.at(x, 5)] = 10
		p.light[p.at(x, 5)+1] = 10
		p.light[p.at(x, 5)+2] = 10
	}
	m := flat(frameW, frameH, 0)
	for y := 0; y < 12; y++ {
		for x := 0; x < frameW; x++ {
			m.light[m.at(x, y)] = 1
			m.light[m.at(x, y)+1] = 1
			m.light[m.at(x, y)+2] = 1
		}
	}
	return keep(t, dir, "plate.pfm", p, -1), keep(t, dir, "mask.pfm", m, -1)
}

// runMain runs the command's main in this process with arguments, and
// says what it printed on standard output.
func runMain(t *testing.T, args ...string) string {
	t.Helper()
	savedArgs, savedOut := os.Args, os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Args, os.Stdout = append([]string{"lightshow"}, args...), writer
	defer func() { os.Args, os.Stdout = savedArgs, savedOut }()
	main()
	writer.Close()
	printed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(printed)
}

// TestMainWritesTheSkyAndTheFront: a run with every optional input names
// the strobes' level on standard output, writes the debug mask, and
// writes a sky no darker than the plate at the master's level and a front
// the same size.
func TestMainWritesTheSkyAndTheFront(t *testing.T) {
	guard(t)
	dir := t.TempDir()
	plate, mask := scenePFMs(t, dir)
	disc := flat(frameW, frameH, 0)
	for x := 0; x < frameW; x++ {
		disc.light[disc.at(x, 5)] = 1
	}
	passes := filepath.Join(dir, "passes")
	fire := filepath.Join(dir, "fire")
	for _, d := range []string{passes, fire} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	pass(t, passes, "overhead.pfm", frameW, frameH, 0.01)
	pass(t, passes, "flood-left.pfm", frameW, frameH, 0.001)
	pass(t, fire, "000.pfm", frameW, frameH, 0.5)
	trees := flat(frameW, frameH, 0)
	for y := 0; y < frameH; y++ {
		trees.light[trees.at(3, y)] = 1
	}
	debug := filepath.Join(dir, "debug.pfm")
	t.Setenv("CUES", cueFile(t, "master=0.5\nwhite=1\nmagenta=0\nrig=1\n"+
		"search=1\noverhead=1\nfloodLeft=1\nflood_left=1\nfire=1\n"+
		"fire_since=0\nfire_lit=0\nmoon_alt=5.45\n"))
	t.Setenv("MOON_DISC", keep(t, dir, "disc.pfm", disc, -1))
	t.Setenv("TREEMASK", keep(t, dir, "trees.pfm", trees, -1))
	water := flat(frameW, frameH, 1)
	t.Setenv("WATERMASK", keep(t, dir, "water.pfm", water, -1))
	t.Setenv("DEBUGMASK", debug)
	t.Setenv("PASSES", passes)
	t.Setenv("FIRE_FRAMES", fire)
	sky, front := filepath.Join(dir, "sky.pfm"), filepath.Join(dir, "front.pfm")
	printed := runMain(t, plate, mask, sky, front, "1")
	if printed != "0.5000\n" {
		t.Errorf("standard output was %q, want the level 0.5000", printed)
	}
	skyOut, _, err := read(sky)
	if err != nil {
		t.Fatal(err)
	}
	plateIn, _, _ := read(plate)
	for i := range plateIn.light {
		if skyOut.light[i] < 0.5*plateIn.light[i]-1e-6 {
			t.Fatalf("sky number %d is %v, under half the plate's %v", i,
				skyOut.light[i], plateIn.light[i])
		}
	}
	frontOut, _, err := read(front)
	if err != nil || frontOut.width != frameW || frontOut.height != frameH ||
		total(frontOut) <= 0 {
		t.Errorf("the front is %v, %v", frontOut, err)
	}
	shown, _, err := read(debug)
	if err != nil {
		t.Fatal(err)
	}
	if shown.light[shown.at(0, 0)] != 1 || shown.light[shown.at(0, 30)] != 0 {
		t.Errorf("the debug mask has sky %v at the top and %v on land",
			shown.light[shown.at(0, 0)], shown.light[shown.at(0, 30)])
	}
	if shown.light[shown.at(0, 12)+2] != 1 {
		t.Errorf("the rim line is not marked at row 12, under the last sky")
	}
}

// TestMainPercussiveFire: with a swell the fire is struck,
// and passes of the wrong size or a missing fire directory stop the run
// in a child.
func TestMainPercussiveFire(t *testing.T) {
	guard(t)
	dir := t.TempDir()
	plate, mask := scenePFMs(t, dir)
	fire := filepath.Join(dir, "fire")
	swell := filepath.Join(dir, "swell")
	for _, d := range []string{fire, swell} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		pass(t, fire, "00"+strconv.Itoa(i)+".pfm", frameW, frameH, 1e-4)
	}
	pass(t, swell, "000.pfm", frameW, frameH, 1e-4)
	fireSwellDir = swell
	t.Setenv("CUES", cueFile(t, "fire=1\nfire_since=0\nfire_lit=0\n"))
	t.Setenv("FIRE_FRAMES", fire)
	sky, front := filepath.Join(dir, "sky.pfm"), filepath.Join(dir, "front.pfm")
	runMain(t, plate, mask, sky, front, "1")
	frontOut, _, err := read(front)
	if err != nil || !finite(frontOut) || total(frontOut) <= 0 {
		t.Fatalf("the struck fire laid %v, %v", total(frontOut), err)
	}
	// with the fire off the same run lays less in the front
	t.Setenv("CUES", cueFile(t, "fire=0\nfire_since=0\nfire_lit=0\n"))
	runMain(t, plate, mask, sky, front, "1")
	base, _, err := read(front)
	if err != nil {
		t.Fatal(err)
	}
	more := false
	for i := range base.light {
		if frontOut.light[i] < base.light[i] {
			t.Fatalf("number %d is darker with the fire on", i)
		}
		more = more || frontOut.light[i] > base.light[i]
	}
	if !more {
		t.Error("the fire laid no light in the front")
	}
	wrong := filepath.Join(dir, "wrong")
	os.Mkdir(wrong, 0o755)
	pass(t, wrong, "000.pfm", 3, 3, 1)
	if status, stderr := child(t, []string{plate, mask, sky, front, "1"},
		"CUES="+cueFile(t, "fire=1\nfire_since=0\n"), "FIRE_FRAMES="+wrong); status != 1 ||
		!strings.Contains(stderr, "3x3") {
		t.Errorf("a fire frame of the wrong size: %d, %q", status, stderr)
	}
}

// TestQuietDropsOnlyTheControls checks that a control is left out of the
// environment a child is given, and that everything else is kept.
func TestQuietDropsOnlyTheControls(t *testing.T) {
	got := quiet([]string{"CUES=/nowhere", "PATH=/bin", "MOON_DISC=x", "CUESHEET=kept"})
	if len(got) != 2 || got[0] != "PATH=/bin" || got[1] != "CUESHEET=kept" {
		t.Errorf("quiet kept %v", got)
	}
}
