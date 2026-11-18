package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderDryRun checks the one-line summary of a render that is not run.
func TestRenderDryRun(t *testing.T) {
	dir := t.TempDir()
	scene := writeFile(t, dir, "scene.yaml", minimalScene)
	out := filepath.Join(dir, "frames")
	code, stdout, stderr := runTool(t, "render", "--scene", scene, "--out", out, "--dry-run")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if want := "640 x 360, 4 samples/pixel, 1 sources -> " + out + "\n"; stdout != want {
		t.Errorf("output = %q, want %q", stdout, want)
	}
}

// TestRenderRefusals checks the argument and scene refusals that precede any
// device work.
func TestRenderRefusals(t *testing.T) {
	dir := t.TempDir()
	scene := writeFile(t, dir, "scene.yaml", minimalScene)
	bad := writeFile(t, dir, "bad.yaml", "{")
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"positional argument", []string{"render", "extra", "--scene", scene, "--out", "x"}, "without positional arguments"},
		{"scene omitted", []string{"render", "--out", "x"}, "island needs --scene"},
		{"scene empty", []string{"render", "--scene", "", "--out", "x"}, "island needs --scene"},
		{"out omitted", []string{"render", "--scene", scene}, "island needs --out"},
		{"scene unreadable", []string{"render", "--scene", bad, "--out", "x"}, "render:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := runTool(t, tc.args...)
			if code == 0 {
				t.Fatal("exit code 0")
			}
			mustContain(t, stderr, tc.want)
		})
	}
}

// TestSequenceDryRun checks that every frame is sampled and the summary
// reports the track and the scene's film size.
func TestSequenceDryRun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "scene.yaml", minimalScene)
	track := writeFile(t, dir, "track.yaml", sequenceText("continuous", "linear", "scene.yaml"))
	out := filepath.Join(dir, "frames")
	code, stdout, stderr := runTool(t, "sequence", "--plan", track, "--out", out, "--dry-run")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if want := "5 frames at 10 fps, 20.000 scene seconds, 640 x 360 -> " + out + "\n"; stdout != want {
		t.Errorf("output = %q, want %q", stdout, want)
	}
}

// TestSequenceRefusals checks the refusals that precede any device work.
func TestSequenceRefusals(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "scene.yaml", minimalScene)
	track := writeFile(t, dir, "track.yaml", sequenceText("", "linear", "scene.yaml"))
	lost := writeFile(t, dir, "lost.yaml", sequenceText("", "linear", "absent.yaml"))
	late := writeFile(t, dir, "late.yaml", sequenceText("", "linear", "late-scene.yaml"))
	writeFile(t, dir, "late-scene.yaml", strings.Replace(minimalScene, "exposure: 1.5", "exposure: 1.5\ntime-seconds: 86390", 1))
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"positional argument", []string{"sequence", "extra", "--plan", track, "--out", "x"}, "without positional arguments"},
		{"plan omitted", []string{"sequence", "--out", "x"}, "island needs --plan"},
		{"out omitted", []string{"sequence", "--plan", track}, "island needs --out"},
		{"plan unreadable", []string{"sequence", "--plan", filepath.Join(dir, "none.yaml"), "--out", "x"}, "sequence:"},
		{"scene missing", []string{"sequence", "--plan", lost, "--out", "x"}, "absent.yaml"},
		{"clock exceeds a day", []string{"sequence", "--plan", late, "--out", "x"}, "exceeds one day"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := runTool(t, tc.args...)
			if code == 0 {
				t.Fatal("exit code 0")
			}
			mustContain(t, stderr, tc.want)
		})
	}
}
