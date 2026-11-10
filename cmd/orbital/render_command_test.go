package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/raw"
)

func TestRealOrbitalWritesNumberedFramesAndLinearRaws(t *testing.T) {
	dir := t.TempDir()
	args := []string{"-far", "-still", "-gas", "0", "-w", "2", "-h", "2", "-n", "1", "-first", "1", "-frames", "2", "-raw", "-eye", "0,0,3600", "-up", "0,1,0", "-out", filepath.Join(dir, "orbit.png")}
	out, err := runCoverageCommand(t, args...)
	if err != nil {
		t.Fatalf("%v %q", err, out)
	}
	for _, stem := range []string{"orbit-0001", "orbit-0002"} {
		f, err := os.Open(filepath.Join(dir, stem+".png"))
		if err != nil {
			t.Fatal(err)
		}
		picture, err := png.Decode(f)
		f.Close()
		if err != nil || picture.Bounds() != image.Rect(0, 0, 2, 2) {
			t.Fatalf("%v %v", picture, err)
		}
		linear, err := raw.Read(filepath.Join(dir, stem+".pfm"))
		if err != nil {
			t.Fatal(err)
		}
		if linear.W != 2 || linear.H != 2 {
			t.Fatal("raw dimensions")
		}
	}
}
func TestOrbitalRejectsBadTerminalRequestsBeforeBuildingAWorld(t *testing.T) {
	out, err := runCoverageCommand(t, "-cells", "bad")
	if err == nil || !strings.Contains(string(out), "orbital:") {
		t.Fatalf("%v %q", err, out)
	}
}
