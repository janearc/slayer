package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

// TestStretchArgsRewrites checks, for hand-worked argument lists, where the
// stretch verb is put in and which words are left as they came.
func TestStretchArgsRewrites(t *testing.T) {
	for _, c := range []struct{ in, want []string }{
		{nil, nil},
		{[]string{"compare", "-from", "1"}, []string{"compare", "-from", "1"}},
		{[]string{"-unknown", "1"}, []string{"-unknown", "1"}},
		{[]string{"-png", "-from", "3"}, []string{"stretch", "--png", "--from", "3"}},
		{[]string{"-from=3", "-redraw", "-to", "9"}, []string{"stretch", "--from=3", "--redraw", "--to", "9"}},
		{[]string{"stretch", "-out", "-x", "-png"}, []string{"stretch", "--out", "-x", "--png"}},
		{[]string{"--stretch", "-experimental-gpu", "-experimental-gpu-ocean", "-trace", "t"}, []string{"--stretch", "--experimental-gpu", "--experimental-gpu-ocean", "--trace", "t"}},
		{[]string{"-png", "--", "-from", "-png"}, []string{"stretch", "--png", "--", "-from", "-png"}},
		{[]string{"-png", "stray", "-other"}, []string{"stretch", "--png", "stray", "-other"}},
	} {
		if got := stretchArgs(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v: got %v, want %v", c.in, got, c.want)
		}
	}
}

// TestStretchOptionRefusals checks that each bad flag is refused with its
// reason, through the dry run so that nothing is loaded.
func TestStretchOptionRefusals(t *testing.T) {
	base := []string{"stretch", "--settings", "s.yaml", "--out", "film"}
	with := func(extra ...string) []string { return append(append([]string{}, base...), extra...) }
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"from not a number", with("--png", "--from", "abc"), "from:"},
		{"to not a number", with("--png", "--to", "abc"), "to:"},
		{"fps not a number", with("--png", "--fps", "abc"), "fps must be finite and positive"},
		{"fps zero", with("--png", "--fps", "0"), "fps must be finite and positive"},
		{"fps infinite", with("--png", "--fps", "Inf"), "fps must be finite and positive"},
		{"negative from", with("--png", "--from=-1"), "nonnegative half-open"},
		{"empty range", with("--png", "--from", "5", "--to", "5"), "nonnegative half-open"},
		{"range too long", with("--png", "--from", "0", "--to", "2000000"), "nonnegative half-open"},
		{"positional argument", with("--png", "extra"), "no positional arguments"},
		{"no output kind", with(), "at least one of png or encode"},
		{"no settings", []string{"stretch", "--out", "film", "--png"}, "settings, out"},
		{"gpu with png", with("--png", "--experimental-gpu"), "refuses --png"},
		{"empty trace", with("--experimental-gpu", "--encode", "m.mp4", "--ffmpeg", "/x/ffmpeg", "--trace", ""), "trace requires"},
		{"trace over limit", with("--experimental-gpu", "--encode", "m.mp4", "--ffmpeg", "/x/ffmpeg", "--trace", "t.json", "--to", "149797"), "trace requires"},
		{"empty movie path", with("--png", "--encode", ""), "encode path must not be empty"},
		{"movie without ffmpeg", with("--encode", "m.mp4"), "absolute --ffmpeg"},
		{"movie with relative ffmpeg", with("--encode", "m.mp4", "--ffmpeg", "ffmpeg"), "absolute --ffmpeg"},
	} {
		code, out := run(append(c.args, "--dry-run")...)
		if code == 0 || !strings.Contains(out, c.want) {
			t.Errorf("%s: %d %q", c.name, code, out)
		}
	}
	code, out := run(with("--png", "--from", "3", "--to", "5", "--dry-run")...)
	if code != 0 || !strings.Contains(out, "CPU film frames [3,5)") {
		t.Errorf("CPU dry run: %d %q", code, out)
	}
	code, out = run(with("--encode", "m.mp4", "--ffmpeg", "/x/ffmpeg", "--dry-run")...)
	if code != 0 || !strings.Contains(out, `movie: "m.mp4"`) || !strings.Contains(out, "[0,72)") {
		t.Errorf("default range dry run: %d %q", code, out)
	}
	code, errs := runInto(&failAfter{}, append(with("--png"), "--dry-run")...)
	if code == 0 || !strings.Contains(errs, "write refused") {
		t.Errorf("CPU dry run write: %d %q", code, errs)
	}
	code, errs = runInto(&failAfter{}, append(with("--experimental-gpu", "--encode", "m.mp4", "--ffmpeg", "/x/ffmpeg"), "--dry-run")...)
	if code == 0 || !strings.Contains(errs, "write refused") {
		t.Errorf("GPU dry run write: %d %q", code, errs)
	}
}

// settingsYAML is a valid settings file; paths are relative to its own
// directory and name files that the tests do not create unless they say so.
const settingsYAML = `frame: {width: 4096, height: 3002}
score: score.yaml
cues: cues.yaml
passes:
  nodisc: passes/nodisc.pfm
  disc: passes/disc.pfm
  mask: passes/mask.pfm
  water: passes/water.pfm
  trees: passes/trees.pfm
  back_left: passes/back-left.pfm
  back_right: passes/back-right.pfm
  overhead: passes/overhead.pfm
fire:
  frames: {off: true}
  swell: {off: true}
  burst: {off: true}
  gain: 0.1
  decay: 1
  fps: 24
  steady_from: 42
  pulse: 3
  flash: 8
  snap: 0.12
  swell_fall: 0.3
  burst_gain: 1
  top: 0.6
song: {off: true}
camera:
  eye: 71.7
  look: -6
  focal: 85
  frame_height: 39.61
  moon_alt: 2.3
  moon_scale: 6
  shore_distance: 1660
  moon_radius: 1.554
  moon_disc_alt: 5.45
night:
  haze_gain: 2.5
  flood_gain: 400
  spot_gain: 150
  search_gain: 0.9
  sky_glow: 0.02
  overhead_gain: 0.002
  overhead_mist: 3
  mist_thick: 12
  rig_deck: 20.4
  bloom: 1.6
  exposure: 23
  wind: 8
  mist_seed: 3
  strobe_mist: 0.25
  search_count: 8
  flood_passes: [back-left, back-right]
  mist_colour: [0.78, 0.88, 1.08]
beam: {out: 900, radius: 90, apex: 450, gain: 40, halo: 0.1}
gates: {load_limit: 1000000, swap_limit_bytes: 9000000000000000000}
fps: 24
disk_floor_bytes: 1
output: {memory_cache_bytes: 536870912}
`

// stretchFixture writes a settings file with the given text replacements
// applied, and valid score and cue files beside it, and returns the directory
// and the settings path.
func stretchFixture(t *testing.T, replace ...string) (string, string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("the host gate reads Darwin statistics")
	}
	dir := t.TempDir()
	text := strings.NewReplacer(replace...).Replace(settingsYAML)
	writeFile(t, dir, "score.yaml", "score: {}")
	writeFile(t, dir, "cues.yaml", "cues: [{at_sec: 0.4, moon_alt: 7}]")
	return dir, writeFile(t, dir, "settings.yaml", text)
}

// TestStretchRefusesBeforeRendering checks each refusal that the stretch verb
// makes after reading its settings and before it draws anything.
func TestStretchRefusesBeforeRendering(t *testing.T) {
	ffmpeg := []string{"--ffmpeg", "/x/ffmpeg"}
	for _, c := range []struct {
		name    string
		replace []string
		extra   []string
		want    string
	}{
		{"song for a movie", []string{"song: {off: true}", "song: {path: song.wav}"}, []string{"--encode", "OUT/m.mp4"}, "audio muxing is unsupported"},
		{"odd width for a movie", []string{"width: 4096", "width: 4095"}, []string{"--encode", "OUT/m.mp4"}, "even frame width"},
		{"frame rate over 240 for a movie", nil, []string{"--encode", "OUT/m.mp4", "--fps", "300"}, "fps at most 240"},
		{"movie over the settings", nil, []string{"--encode", "SETTINGS"}, "output would replace input"},
		{"debug mask", []string{"fps: 24\ndisk", "debug_mask: {path: mask.pfm}\nfps: 24\ndisk"}, []string{"--png"}, "debug_mask is unsupported"},
		{"shore distance", []string{"shore_distance: 1660", "shore_distance: 1000"}, []string{"--png"}, "requires 1660"},
		{"disk floor", []string{"disk_floor_bytes: 1", "disk_floor_bytes: 9000000000000000000"}, []string{"--png"}, "disk resource refused"},
		{"existing movie", nil, []string{"--encode", "OUT/m.mp4"}, "use --redraw"},
		{"missing score", nil, []string{"--png"}, "score.yaml"},
		{"missing cues", nil, []string{"--png"}, "cues.yaml"},
		{"missing pass for the CPU route", nil, []string{"--encode", "OUT/m.mp4"}, "passes.nodisc"},
		{"missing pass for the GPU route", nil, []string{"--experimental-gpu", "--encode", "OUT/m.mp4"}, "passes.nodisc"},
		{"missing pass for the PNG route", nil, []string{"--png"}, "passes.nodisc"},
		{"bad score for the CPU route", nil, []string{"--encode", "OUT/m.mp4"}, "yaml:"},
		{"bad score for the GPU route", nil, []string{"--experimental-gpu", "--encode", "OUT/m.mp4"}, "yaml:"},
		{"bad score for the PNG route", nil, []string{"--png"}, "execution: show"},
	} {
		dir, settingsPath := stretchFixture(t, c.replace...)
		out := filepath.Join(dir, "film")
		args := []string{"stretch", "--settings", settingsPath, "--out", out}
		for _, word := range c.extra {
			word = strings.ReplaceAll(word, "OUT", out)
			args = append(args, strings.ReplaceAll(word, "SETTINGS", settingsPath))
		}
		if strings.Contains(c.name, "movie") || strings.Contains(c.name, "song") || strings.Contains(c.name, "frame rate") || strings.Contains(c.name, "GPU") || strings.Contains(c.name, "CPU") {
			args = append(args, ffmpeg...)
		}
		switch {
		case c.name == "existing movie":
			if err := os.MkdirAll(out, 0700); err != nil {
				t.Fatal(err)
			}
			writeFile(t, out, "m.mp4", "old movie")
		case c.name == "missing score":
			if err := os.Remove(filepath.Join(dir, "score.yaml")); err != nil {
				t.Fatal(err)
			}
		case c.name == "missing cues":
			if err := os.Remove(filepath.Join(dir, "cues.yaml")); err != nil {
				t.Fatal(err)
			}
		case strings.HasPrefix(c.name, "bad score"):
			writeFile(t, dir, "score.yaml", "not: [valid")
		}
		code, output := run(args...)
		if code == 0 || !strings.Contains(output, c.want) {
			t.Errorf("%s: %d %q", c.name, code, output)
		}
	}
}

// TestStretchRefusesBadSettings checks that a settings file that is missing
// or that does not parse stops the verb before anything is made.
func TestStretchRefusesBadSettings(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "film")
	code, output := run("stretch", "--settings", filepath.Join(dir, "none.yaml"), "--out", out, "--png")
	if code == 0 || !strings.Contains(output, "none.yaml") {
		t.Errorf("missing settings: %d %q", code, output)
	}
	bad := writeFile(t, dir, "bad.yaml", "frame: {width: 4}\n")
	code, output = run("stretch", "--settings", bad, "--out", out, "--png")
	if code == 0 || !strings.Contains(output, "settings") {
		t.Errorf("bad settings: %d %q", code, output)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a refused run made its output directory: %v", err)
	}
}

// TestResolvePathsLeavesOptionalAlone checks that relative paths are joined to
// the base, and absolute, empty and switched-off paths are not.
func TestResolvePathsLeavesOptionalAlone(t *testing.T) {
	base := t.TempDir()
	absolute := filepath.Join(t.TempDir(), "abs.pfm")
	text := strings.NewReplacer("song: {off: true}", "song: {path: song.wav}", "water: passes/water.pfm", "water: "+absolute, "frames: {off: true}", "frames: {path: fire}").Replace(settingsYAML)
	path := writeFile(t, base, "settings.yaml", text)
	// Read the file back through the verb's own reader to get a settings value.
	s := mustReadSettings(t, path)
	resolvePaths(&s, base)
	if s.Passes.Water != absolute {
		t.Errorf("absolute path changed: %s", s.Passes.Water)
	}
	if s.Passes.Disc != filepath.Join(base, "passes/disc.pfm") || s.Song.Path != filepath.Join(base, "song.wav") || s.Fire.Frames.Path != filepath.Join(base, "fire") {
		t.Errorf("relative paths not joined: %+v", s)
	}
	if s.Fire.Swell.Path != "" || !s.Fire.Swell.Off {
		t.Errorf("switched-off path touched: %+v", s.Fire.Swell)
	}
}

// TestSceneRefusals checks the scene verb's refusals: its arguments, a plan
// that is missing, an output directory that exists, a movie without a frame
// rate, a movie that cannot be set up, and a dry run that cannot report.
func TestSceneRefusals(t *testing.T) {
	dir := t.TempDir()
	job := "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: summit.yaml}\nfilm: {width: 320, height: 180, rays: 1}\nrenderer: gpu\noutputs: {picture: unused.png}\n"
	planText := "version: 1\njob: job.yaml\nfrom: 0\nto: 1\ntransport-scale: 2\ntransport-rays: 1\nplate-every: 1\n"
	writeFile(t, dir, "job.yaml", job)
	plan := writeFile(t, dir, "plan.yaml", planText)
	if code, out := run("scene", plan); code == 0 || !strings.Contains(out, "give PLAN and --out") {
		t.Errorf("no out: %d %q", code, out)
	}
	if code, out := run("scene", "--out", filepath.Join(dir, "o")); code == 0 || !strings.Contains(out, "give PLAN and --out") {
		t.Errorf("no plan: %d %q", code, out)
	}
	if code, out := run("scene", filepath.Join(dir, "none.yaml"), "--out", filepath.Join(dir, "o")); code == 0 || !strings.Contains(out, "none.yaml") {
		t.Errorf("missing plan: %d %q", code, out)
	}
	if code, errs := runInto(&failAfter{}, "scene", plan, "--out", filepath.Join(dir, "o"), "--dry-run"); code == 0 || !strings.Contains(errs, "write refused") {
		t.Errorf("dry run write: %d %q", code, errs)
	}
	existing := filepath.Join(dir, "existing")
	if err := os.Mkdir(existing, 0700); err != nil {
		t.Fatal(err)
	}
	if code, out := run("scene", plan, "--out", existing); code == 0 || !strings.Contains(out, "exists") {
		t.Errorf("existing output: %d %q", code, out)
	}
	fresh := filepath.Join(dir, "fresh")
	code, out := run("scene", plan, "--out", fresh, "--movie")
	if code == 0 || !strings.Contains(out, "a movie needs a film job with fps") {
		t.Fatalf("movie without fps: %d %q", code, out)
	}
	got, err := os.ReadFile(filepath.Join(fresh, "plan.txt"))
	if err != nil || string(got) != planText {
		t.Errorf("plan copy: %q %v", got, err)
	}
	if data, err := os.ReadFile(filepath.Join(fresh, "job.json")); err != nil || !strings.Contains(string(data), "320") {
		t.Errorf("job copy: %q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(fresh, "render.json")); !os.IsNotExist(err) {
		t.Errorf("a refused run wrote a receipt: %v", err)
	}
	writeFile(t, dir, "job.yaml", strings.Replace(strings.Replace(job, "width: 320, height: 180", "width: 4096, height: 4096", 1), "outputs: {picture: unused.png}", "outputs: {frames: unused}", 1)+"fps: 120\nduration: {value: 1, unit: s}\n")
	code, out = run("scene", plan, "--out", filepath.Join(dir, "fast"), "--movie")
	if code == 0 || !strings.Contains(out, "movie:") {
		t.Errorf("movie with a size no level can carry: %d %q", code, out)
	}
}

// TestSceneBuildNamesTheRevision checks that a link-time stamp is returned
// as it is, and that without one the answer is not empty.
func TestSceneBuildNamesTheRevision(t *testing.T) {
	saved := build
	t.Cleanup(func() { build = saved })
	build = "stamp-for-test"
	if got := sceneBuild(); got != "stamp-for-test" {
		t.Errorf("stamped build: %q", got)
	}
	build = ""
	if got := sceneBuild(); got == "" {
		t.Error("an unstamped build gave no name")
	}
}

// mustReadSettings reads and checks the settings file at path.
func mustReadSettings(t *testing.T, path string) settings.Settings {
	t.Helper()
	s, err := settings.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestStretchRefusesUnwritableOutputParent checks that an output directory
// that cannot be made is an error and that nothing is drawn.
func TestStretchRefusesUnwritableOutputParent(t *testing.T) {
	dir, settingsPath := stretchFixture(t)
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0700) })
	if f, err := os.Create(filepath.Join(locked, "probe")); err == nil {
		f.Close()
		t.Skip("directory permissions are not enforced for this user")
	}
	code, output := run("stretch", "--settings", settingsPath, "--out", filepath.Join(locked, "film"), "--png")
	if code == 0 || !strings.Contains(output, "permission denied") {
		t.Errorf("unwritable parent: %d %q", code, output)
	}
}
