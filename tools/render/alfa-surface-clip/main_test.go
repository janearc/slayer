package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	out := filepath.Join(root, "output")
	if err := os.MkdirAll(filepath.Join(out, "inputs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "inputs", "slayer"), []byte("pinned renderer"), 0644); err != nil {
		t.Fatal(err)
	}
	base, err := os.ReadFile("scene.template.yaml")
	if err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(root, "template")
	if err = os.WriteFile(template, base, 0644); err != nil {
		t.Fatal(err)
	}
	return out, template
}
func call(t *testing.T, args ...string) int {
	t.Helper()
	var out, err bytes.Buffer
	return run(args, &out, &err)
}
func TestCLIRefusesUnknownOrUnboundedArgumentsBeforeWrites(t *testing.T) {
	for _, suffix := range [][]string{{"unknown"}, {"prepare", "extra"}, {"prepare", "--camera-east", "NaN"}, {"prepare", "--camera-east", "Inf"}, {"prepare", "--camera-east", "1000001"}, {"prepare", "--aim-east", "-1000001"}, {"prepare", "--unknown"}} {
		t.Run(strings.Join(suffix, "_"), func(t *testing.T) {
			root := t.TempDir()
			out := filepath.Join(root, "must-not-exist")
			args := append([]string{suffix[0], out, "absent-template"}, suffix[1:]...)
			if call(t, args...) == 0 {
				t.Fatal("invalid invocation succeeded")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("invalid CLI wrote output: %v", err)
			}
		})
	}
}
func TestPrepareExactRerunAndChangedInputRefusal(t *testing.T) {
	out, template := fixture(t)
	args := []string{"prepare", out, template}
	if call(t, args...) != 0 {
		t.Fatal("prepare failed")
	}
	first := filepath.Join(out, "jobs", "0000.yaml")
	before, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if call(t, args...) != 0 {
		t.Fatal("exact rerun failed")
	}
	after, _ := os.Stat(first)
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("exact rerun rewrote matching input")
	}
	if err = os.Remove(first); err != nil {
		t.Fatal(err)
	}
	last := filepath.Join(out, "jobs", "0119.yaml")
	if err = os.WriteFile(last, []byte("different recipe"), 0644); err != nil {
		t.Fatal(err)
	}
	if call(t, args...) == 0 {
		t.Fatal("changed recipe was accepted")
	}
	if _, err = os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("refusal happened after earlier inputs were written")
	}
	got, _ := os.ReadFile(last)
	if string(got) != "different recipe" {
		t.Fatal("changed existing input overwritten")
	}
}
func TestExistingPictureWithoutPinRefused(t *testing.T) {
	out, template := fixture(t)
	if err := os.MkdirAll(filepath.Join(out, "frames"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "frames", "0000.png"), []byte("stale picture"), 0644); err != nil {
		t.Fatal(err)
	}
	if call(t, "prepare", out, template) == 0 {
		t.Fatal("untracked existing picture accepted")
	}
	if _, err := os.Stat(filepath.Join(out, "jobs")); !os.IsNotExist(err) {
		t.Fatal("wrote inputs before refusing stale picture")
	}
}
func TestRendererChangeRefused(t *testing.T) {
	out, template := fixture(t)
	if call(t, "prepare", out, template) != 0 {
		t.Fatal("prepare failed")
	}
	if err := os.WriteFile(filepath.Join(out, "inputs", "slayer"), []byte("other renderer"), 0644); err != nil {
		t.Fatal(err)
	}
	if call(t, "prepare", out, template) == 0 {
		t.Fatal("renderer pin change accepted")
	}
}

func TestDryRunDoesNotWritePreparedInputs(t *testing.T) {
	out, template := fixture(t)
	if call(t, "prepare", out, template, "--dry-run") != 0 {
		t.Fatal("dry run failed")
	}
	if _, err := os.Stat(filepath.Join(out, "jobs")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote jobs")
	}
	if _, err := os.Stat(filepath.Join(out, "inputs", "film-pin.json")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote input pin")
	}
}
func TestNativeTrajectoryAndNamedHalfPortrait(t *testing.T) {
	base, err := os.ReadFile("scene.template.yaml")
	if err != nil {
		t.Fatal(err)
	}
	frames, err := plan(config{mode: "prepare", camera: -1650, aim: -1535}, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 120 {
		t.Fatalf("frames=%d", len(frames))
	}
	for _, i := range []int{0, 60, 119} {
		one := frames[i]
		seconds := float64(i) / 12
		for _, want := range []string{fmt.Sprintf("east: {value: %.9f", -1500+15*seconds), fmt.Sprintf("location: {at: [%.9f, -50", -1650+15*seconds), fmt.Sprintf("look_at: [%.9f, 0.000000, 0.000000]", -1535+15*seconds), fmt.Sprintf("time: {value: %.9f", seconds), "origin: [-1545.902645, 0.257687, 0]"} {
			if !strings.Contains(string(one.set), want) {
				t.Fatalf("frame%d missing %q", i, want)
			}
		}
		if !strings.Contains(string(one.job), "preset: half-portrait, width: 1080, height: 960") {
			t.Fatal("named half-portrait dimensions missing")
		}
	}
}
