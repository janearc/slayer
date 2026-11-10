//go:build darwin

package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const deepWorkingDirectoryProbe = "SLAYER_DEEP_WORKING_DIRECTORY_PROBE"

// Darwin's fixed-size getcwd buffer and Go's bounded parent walk can both
// be exceeded by a real directory. Keep chdir and all cleanup in a child.
func TestNearestPreservesTheDeepWorkingDirectoryRefusal(t *testing.T) {
	if os.Getenv(deepWorkingDirectoryProbe) == "1" {
		probeNearestDeepWorkingDirectory(t)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	args := []string{"-test.run=^TestNearestPreservesTheDeepWorkingDirectoryRefusal$"}
	if dir := flag.Lookup("test.gocoverdir"); testing.CoverMode() != "" &&
		dir != nil && dir.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+dir.Value.String())
	}
	child := exec.CommandContext(ctx, os.Args[0], args...)
	child.Env = append(os.Environ(), deepWorkingDirectoryProbe+"=1")
	out, err := child.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("deep directory probe did not finish: %v", ctx.Err())
	}
	if err != nil {
		t.Fatalf("deep directory probe: %v\n%s", err, out)
	}
}

// probeNearestDeepWorkingDirectory owns the child process's directory walk.
func probeNearestDeepWorkingDirectory(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	path := filepath.Join(base, "sources.yaml")
	if err := os.WriteFile(path, []byte("sources: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := nearest(base, "sources.yaml"); err != nil || got != path {
		t.Fatalf("ordinary nearest lookup: %q %v", got, err)
	}
	origin, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PWD", "")
	if err := os.Chdir(base); err != nil {
		t.Fatal(err)
	}
	depth := 0
	defer func() {
		for range depth {
			if err := os.Chdir(".."); err != nil {
				t.Errorf("leave owned child directory: %v", err)
				break
			}
			if err := os.Remove("step"); err != nil {
				t.Errorf("remove owned child directory: %v", err)
			}
		}
		if err := os.Chdir(origin); err != nil {
			t.Errorf("restore child working directory: %v", err)
		}
	}()
	for range 350 {
		if err := os.Mkdir("step", 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir("step"); err != nil {
			t.Fatal(err)
		}
		depth++
	}
	if _, err := os.Getwd(); !errors.Is(err, syscall.ENAMETOOLONG) {
		t.Fatalf("real directory did not exceed getcwd: %v", err)
	}
	got, err := nearest(".", "sources.yaml")
	if got != "" || !errors.Is(err, syscall.ENAMETOOLONG) {
		t.Fatalf("nearest lost getcwd refusal: %q %v", got, err)
	}
}
