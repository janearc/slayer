package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// -cpuprofile writes a profile of the drawing where it is told, and a
// profile with nowhere to go is a hillside not drawn, which gives back
// one.
func TestTheDrawingIsProfiledWhereAsked(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "draw.pprof")
	var said, wrong bytes.Buffer
	small := append(only(map[string]string{"boulder": "2"}), "-w", "16",
		"-h", "10", "-n", "1", "-out", filepath.Join(dir, "hill.png"),
		"-cpuprofile")
	if code := run(append(small, profile), &said, &wrong); code != 0 {
		t.Fatalf("it gave back %d: %s", code, wrong.String())
	}
	if file, err := os.Stat(profile); err != nil || file.Size() == 0 {
		t.Errorf("no profile at %s: %v", profile, err)
	}
	nowhere := filepath.Join(dir, "no", "such", "draw.pprof")
	if code := run(append(small, nowhere), &said, &wrong); code != 1 {
		t.Errorf("a profile with nowhere to go gave back %d", code)
	}
}
