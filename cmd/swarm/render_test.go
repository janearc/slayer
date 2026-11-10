package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func swarmArgs(out string) []string {
	return []string{"-w", "16", "-h", "12", "-n", "1", "-orbits", "2", "-panels", "3", "-out", out}
}

func TestActualSwarmRendersOnCPUAndGPU(t *testing.T) {
	for _, gpu := range []bool{false, true} {
		t.Run(map[bool]string{false: "cpu", true: "gpu"}[gpu], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "swarm.png")
			args := swarmArgs(path)
			if gpu {
				args = append(args, "-gpu")
			}
			out, err := runCoverageCommand(t, args...)
			if err != nil || !strings.Contains(string(out), "6 panels on 2 orbits") {
				t.Fatalf("%v %q", err, out)
			}
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			picture, err := png.Decode(f)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			if picture.Bounds() != image.Rect(0, 0, 16, 12) {
				t.Fatal(picture.Bounds())
			}
			lit := false
			for y := range 12 {
				for x := range 16 {
					r, g, b, a := picture.At(x, y).RGBA()
					if a != 65535 {
						t.Fatal("transparent print")
					}
					lit = lit || r+g+b > 0
				}
			}
			if !lit {
				t.Fatal("empty print")
			}
		})
	}
}

func TestSwarmPropagatesConstructionDrawingAndWriteFailures(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "held")
	if err := os.WriteFile(parent, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"mesh", append(swarmArgs(filepath.Join(t.TempDir(), "x.png")), "-panels", "0"), "mesh"},
		{"draw", append(swarmArgs(filepath.Join(t.TempDir(), "x.png")), "-gpu", "-w", "0"), "swarm:"},
		{"directory", swarmArgs(filepath.Join(parent, "x.png")), "not a directory"},
		{"file", swarmArgs(t.TempDir()), "swarm:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, err := runCoverageCommand(t, test.args...)
			if err == nil || !strings.Contains(string(out), test.want) {
				t.Fatalf("%v %q", err, out)
			}
		})
	}
}
