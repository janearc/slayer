package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// baseSet is the smallest moonlake set that holds every line the passes swap, each once.
const baseSet = `scene:
  light: moon
  moon:
    face: svs-moon-kit-color
    altitude: 2.3
  sun:
    azimuth: 304
    altitude: -12
  set:
    things:
      close left:
        kind: spruce
        complexity: demo
        seed: 51
        east: 263115.3
        north: 2148423.9
      close right:
        kind: spruce
        complexity: demo
        seed: 52
        east: 263114.1
        north: 2148414.8
      close edge:
        kind: spruce
        complexity: demo
        seed: 53
        east: 263113.8
        north: 2148428.7
`

// runMain runs main with the given arguments, returning what it printed.
func runMain(t *testing.T, args ...string) string {
	t.Helper()
	saved, savedOut := os.Args, os.Stdout
	out, err := os.Create(filepath.Join(t.TempDir(), "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	os.Args, os.Stdout = args, out
	defer func() { os.Args, os.Stdout = saved, savedOut }()
	main()
	text, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// TestMainWritesEveryPass checks each pass file exists, is named on stdout, and holds the swaps that make it that pass.
func TestMainWritesEveryPass(t *testing.T) {
	dir := t.TempDir()
	setPath := filepath.Join(dir, "moonlake.yaml")
	if err := os.WriteFile(setPath, []byte(baseSet), 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	printed := runMain(t, "passes", setPath, out)
	read := func(name string) string {
		raw, err := os.ReadFile(filepath.Join(out, "moonlake-pass-"+name+".yaml"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return string(raw)
	}
	names := []string{"flood-left", "flood-right", "spot-left-0", "spot-left-1", "spot-left-2",
		"spot-right-0", "spot-right-1", "spot-right-2", "back-left", "back-right",
		"overhead", "fire", "fire-swell", "fire-burst", "nodisc", "disc"}
	for _, name := range names {
		if !strings.Contains(printed, "moonlake-pass-"+name+".yaml\n") {
			t.Errorf("%s was not printed", name)
		}
		read(name)
	}
	if lines := strings.Count(printed, "\n"); lines != len(names) {
		t.Errorf("%d lines printed, want %d", lines, len(names))
	}
	// A flood aimed at 152 is turned by 152 - 90 = 62 degrees, and stands where the table says.
	flood := read("flood-left")
	for _, want := range []string{"  light: none\n", "    disc: false\n", "      flood-left:\n        kind: strobe\n        seed: 61\n",
		"        east: 263120.6\n        north: 2148439.0\n", "        heading: {value: 62, unit: deg}\n", "aimed at azimuth 152"} {
		if !strings.Contains(flood, want) {
			t.Errorf("flood-left lacks %q", want)
		}
	}
	// A spot aimed at 70 is turned by -20, which is brought round to 340.
	if spot := read("spot-right-0"); !strings.Contains(spot, "heading: {value: 340, unit: deg}") {
		t.Error("spot-right-0 heading is not 340")
	}
	overhead := read("overhead")
	if !strings.Contains(overhead, "  light: sun\n") || !strings.Contains(overhead, "azimuth: 109.65\n    altitude: 86\n") || strings.Contains(overhead, "light: none") {
		t.Error("overhead is not lit by a sun nearly overhead")
	}
	fire := read("fire")
	if !strings.Contains(fire, "fire rim left:") || !strings.Contains(fire, "fire shore right:") || !strings.Contains(fire, "count: 8\n") ||
		strings.Contains(fire, "close left") || strings.Contains(fire, "close right") || strings.Contains(fire, "close edge") {
		t.Error("fire pass lacks the fires or still holds the close spruces")
	}
	if swell := read("fire-swell"); !strings.Contains(swell, "count: 12\n") || strings.Contains(swell, "count: 8\n") {
		t.Error("fire-swell does not carry the swell recipe")
	}
	if burst := read("fire-burst"); !strings.Contains(burst, "rate: {value: 5000, unit: s^-1}") || strings.Contains(burst, "count: 8\n") {
		t.Error("fire-burst does not carry the burst recipe")
	}
	if nodisc := read("nodisc"); !strings.Contains(nodisc, "    disc: false\n") || !strings.Contains(nodisc, "  light: moon\n") {
		t.Error("nodisc should keep the moon's light and drop its disc")
	}
	if disc := read("disc"); !strings.Contains(disc, "    altitude: 5.45\n") || strings.Contains(disc, "altitude: 2.3") {
		t.Error("disc pass does not move the disc to 5.45 degrees")
	}
}

// TestSwappedReplacesEachOnce checks swaps are made in order and only where the text holds the original once.
func TestSwappedReplacesEachOnce(t *testing.T) {
	got := swapped("a b c", "test", []struct{ from, to string }{{"a", "x"}, {"x b", "y"}})
	if got != "y c" {
		t.Fatalf("swapped=%q, want %q", got, "y c")
	}
}

// TestFailingRunsStopWithStatusOne runs the program as a child with bad arguments and a set that lacks the lines to swap.
func TestFailingRunsStopWithStatusOne(t *testing.T) {
	if os.Getenv("PASSES_TEST_CHILD") == "1" {
		args := strings.Split(os.Getenv("PASSES_TEST_ARGS"), "|")
		os.Args = append([]string{"passes"}, args...)
		if os.Getenv("PASSES_TEST_ARGS") == "" {
			os.Args = []string{"passes"}
		}
		main()
		return
	}
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(empty, []byte("nothing to swap\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "good.yaml")
	if err := os.WriteFile(good, []byte(baseSet), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, args, want string }{
		{"no arguments", "", "passes sets/moonlake.yaml sets"},
		{"missing set", filepath.Join(dir, "absent.yaml") + "|" + dir, "no such file"},
		{"set without the lines", empty + "|" + dir, `the base: the set does not hold "  light: moon\n" once`},
		{"missing output directory", good + "|" + filepath.Join(dir, "absent"), "no such file"},
	}
	for _, tc := range cases {
		cmd := exec.Command(os.Args[0], "-test.run=^TestFailingRunsStopWithStatusOne$")
		cmd.Env = append(os.Environ(), "PASSES_TEST_CHILD=1", "PASSES_TEST_ARGS="+tc.args)
		out, err := cmd.CombinedOutput()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 1 {
			t.Errorf("%s: err=%v, want exit status 1", tc.name, err)
		}
		if !strings.Contains(string(out), "passes: ") || !strings.Contains(string(out), tc.want) {
			t.Errorf("%s: output %q does not say %q", tc.name, out, tc.want)
		}
	}
}
