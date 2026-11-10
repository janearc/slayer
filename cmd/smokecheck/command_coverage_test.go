package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/commandtest"
)

var coverageCommand string

// TestMain builds the real command once, with the test's coverage mode.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "slayer-smokecheck-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	coverageCommand = filepath.Join(directory, "smokecheck")
	if err = commandtest.Build(".", ".", coverageCommand, testing.CoverMode()); err != nil {
		os.RemoveAll(directory)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

// runCoverageCommand bounds a real invocation and retains its diagnostics.
func runCoverageCommand(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	return exec.CommandContext(ctx, coverageCommand, args...).CombinedOutput()
}

func TestRealSmokecheckRendersThenServesAndCopiesTheSameImages(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir := t.TempDir()
	for i, want := range []string{"drew 1, served 0", "drew 0, served 1"} {
		out, err := runCoverageCommand(t, "smoke", "1", "-k", "1", "-out", dir)
		if err != nil || !strings.Contains(string(out), want) || !strings.Contains(string(out), "honest") {
			t.Fatalf("run%d %v %q", i, err, out)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 3 {
		t.Fatalf("copied outputs %v %v", entries, err)
	}
	decoded := 0
	verdicts := 0
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if filepath.Ext(path) == ".png" {
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			image, err := png.Decode(f)
			f.Close()
			if err != nil || image.Bounds().Dx() < 358 || image.Bounds().Dy() < 150 {
				t.Fatalf("bad picture %v %v", image, err)
			}
			decoded++
		} else if filepath.Ext(path) == ".json" {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var report struct{ Compared, Pixels int }
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Compared < report.Pixels/2 || report.Pixels == 0 {
				t.Fatal("empty verdict")
			}
			verdicts++
		}
	}
	if decoded != 2 || verdicts != 1 {
		t.Fatal("missing images or verdict")
	}
}

func TestRealSmokecheckReportsBadRequestsAndOutputPaths(t *testing.T) {
	file := filepath.Join(t.TempDir(), "held")
	if err := os.WriteFile(file, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"smoke", "9999"}, {"smoke", "1", "-k", "0"}, {"smoke", "1", "-unknown"}, {"smoke", "1", "-out", filepath.Join(file, "copies")}} {
		out, err := runCoverageCommand(t, args...)
		if err == nil || !(strings.Contains(string(out), "smokecheck") || strings.Contains(string(out), "usage:")) {
			t.Fatalf("%v %q", err, out)
		}
	}
}
