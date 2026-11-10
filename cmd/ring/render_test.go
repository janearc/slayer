package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ringArgs(out string) []string {
	return []string{"-around", "8", "-w", "8", "-h", "6", "-n", "1", "-out", out}
}
func ringPrint(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if picture.Bounds() != image.Rect(0, 0, 8, 6) {
		t.Fatal(picture.Bounds())
	}
}
func TestRingCommandDrawsVisibleAndFalseColour(t *testing.T) {
	for _, radiation := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "ring.png")
		args := append(ringArgs(path), "-bands")
		if radiation {
			args = append(args, "-radiation")
		}
		out, err := runCoverageCommand(t, args...)
		if err != nil {
			t.Fatalf("%v %q", err, out)
		}
		for _, band := range []string{"x-ray", "gamma ray", "infrared"} {
			if !strings.Contains(string(out), band) {
				t.Fatalf("missing %s: %q", band, out)
			}
		}
		ringPrint(t, path)
		ringPrint(t, strings.TrimSuffix(path, ".png")+"-bands.png")
	}
}
func TestRingCommandReportsBothWriteStages(t *testing.T) {
	file := filepath.Join(t.TempDir(), "held")
	if err := os.WriteFile(file, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ring.png")
	if err := os.Mkdir(strings.TrimSuffix(path, ".png")+"-bands.png", 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{ringArgs(filepath.Join(file, "x.png")), ringArgs(t.TempDir()), append(ringArgs(path), "-bands")} {
		out, err := runCoverageCommand(t, args...)
		if err == nil || !strings.Contains(string(out), "ring:") {
			t.Fatalf("%v %q", err, out)
		}
	}
}

func TestRingDressAndNightKeepAClosedHullAtLowResolution(t *testing.T) {
	for _, night := range []bool{false, true} {
		world, _, err := scene(8, true, true, night)
		if err != nil {
			t.Fatal(err)
		}
		if len(world.Things) < 3 || len(world.Lamps) == 0 {
			t.Fatal("dressed world lost geometry or light")
		}
	}
	for _, around := range []int{0, 8, 32} {
		solid, err := band(around, func(int) bool { return true })
		if err != nil {
			t.Fatal(err)
		}
		if !solid.Closed() || solid.FaceCount() != max(8, around)*20 {
			t.Fatalf("incomplete ring: %d", solid.FaceCount())
		}
	}
	world, _, err := scene(8, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Lamps) != 0 {
		t.Fatal("sun remained on")
	}
}
