//go:build draw

package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/smoke"
)

// TestEachTracedShotWritesItsPictureAndLight draws a very small picture
// of each traced shot and checks that the PNG and the PFM beside it
// exist and that the run says what it drew.
func TestEachTracedShotWritesItsPictureAndLight(t *testing.T) {
	for _, name := range []string{"bent", "straight", "aimed"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "x.png")
			var out, problems bytes.Buffer
			args := []string{"-shot", name, "-out", path, "-width", "16",
				"-height", "9", "-rays", "1", "-aim", "0.01", "-reach", "20"}
			if run(args, &out, &problems) != ok {
				t.Fatal(problems.String())
			}
			if !strings.HasPrefix(out.String(), name+": 16x9 at 1 rays") {
				t.Errorf("run said %q", out.String())
			}
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			config, err := png.DecodeConfig(file)
			if err != nil || config.Width != 16 || config.Height != 9 {
				t.Errorf("picture is %+v (%v)", config, err)
			}
			if info, err := os.Stat(path + ".pfm"); err != nil || info.Size() == 0 {
				t.Errorf("no light written beside the picture: %v", err)
			}
		})
	}
}

// TestDrawRefusesAnUnwritableOutput checks the picture's own create error
// and the light's, and that a lens of nothing is refused.
func TestDrawRefusesAnUnwritableOutput(t *testing.T) {
	ring, _ := smoke.Get(photonRing)
	made := bent(ring, asked{width: 8, height: 4, rays: 1})
	missing := filepath.Join(t.TempDir(), "no", "such", "x.png")
	if err := draw(made, ring, missing); err == nil {
		t.Error("draw wrote into a directory that is not there")
	}
	blocked := filepath.Join(t.TempDir(), "x.png")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := draw(made, ring, blocked); err == nil {
		t.Error("draw wrote a picture over a directory")
	}
	made.wide = 0
	if err := draw(made, ring, filepath.Join(t.TempDir(), "y.png")); err == nil {
		t.Error("draw accepted a lens of no width")
	}
	made = bent(ring, asked{width: 0, height: 4, rays: 1})
	if err := draw(made, ring, filepath.Join(t.TempDir(), "z.png")); err == nil {
		t.Error("draw accepted a picture of no width")
	}
}

// TestRunSaysWhenATracedShotCannotBeWritten checks that run reports the
// draw's error and gives the failing code.
func TestRunSaysWhenATracedShotCannotBeWritten(t *testing.T) {
	var out, problems bytes.Buffer
	missing := filepath.Join(t.TempDir(), "no", "such", "x.png")
	args := []string{"-shot", "bent", "-out", missing, "-width", "8", "-height", "4",
		"-rays", "1", "-wide", "20"}
	if run(args, &out, &problems) != failed || problems.Len() == 0 {
		t.Errorf("run said %q", problems.String())
	}
	if out.Len() != 0 {
		t.Errorf("run claimed to have drawn: %q", out.String())
	}
}
