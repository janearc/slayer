//go:build draw

package ocean

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// TestWaveHeightFieldValidationDraw makes a small CPU tracer picture of the
// vertical wave adapter. Set SLAYER_OCEAN_STUDY_OUT to a new PNG path to retain
// the picture; without it, the render assertion still runs and the file is not
// written.
func TestWaveHeightFieldValidationDraw(t *testing.T) {
	const (
		width, height = 256, 144
		extent        = 81.2
		grid          = 257
		seconds       = 18.0
		seed          = 41
	)
	sea := oceantex.Sea{Wind: 12, Fetch: 100000, Toward: 0.2, Depth: 200}
	field, err := NewHeightField(HeightFieldConfig{
		Sea: sea, Extent: extent, Samples: grid, Seconds: seconds, Seed: seed,
	})
	if err != nil {
		t.Fatal(err)
	}

	world := thing.NewWorld()
	world.Sky = sky.Gradient{
		Horizon: maths.RGB(0.52, 0.68, 0.78),
		Zenith:  maths.RGB(0.10, 0.23, 0.40),
		Ground:  maths.RGB(0.12, 0.18, 0.22),
	}
	world.Add(thing.Thing{Shape: field, Material: &thing.Material{
		Name:   "study-surface-colour",
		Colour: maths.RGB(0.035, 0.20, 0.25),
	}})
	world.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 2.7},
		Material: &thing.Material{
			Name:   "study-marker-colour",
			Colour: maths.RGB(0.78, 0.30, 0.075),
		},
		Motion: thing.Still(maths.NewSubspace(maths.V(0, 7.0, -22),
			maths.Identity, 1)),
	})
	world.Lamp(thing.Sun(maths.V(-0.45, 0.30, 0.65),
		maths.RGB(1.8, 1.65, 1.45)))

	eyeAt := maths.V(0, 13.0, 54.0)
	eye := maths.NewSubspace(eyeAt, maths.LookAt(eyeAt, maths.V(0, 1.2, -15)), 1)
	lens := film.Pinhole{Wide: 49 * maths.Degree}
	frame, err := trace.Sampled(world, eye, lens, 0, width, height, 1)
	if err != nil {
		t.Fatal(err)
	}
	lit := false
	for _, pixel := range frame.Pix {
		if pixel.R+pixel.G+pixel.B > 0 {
			lit = true
			break
		}
	}
	if !lit {
		t.Fatal("validation picture is entirely black")
	}

	path := os.Getenv("SLAYER_OCEAN_STUDY_OUT")
	if path == "" {
		t.Log("rendered CPU validation frame; SLAYER_OCEAN_STUDY_OUT is unset")
		return
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("SLAYER_OCEAN_STUDY_OUT must end in .png: %q", path)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, colour.Tone{Exposure: 1.15, Knee: 0.8, Top: 2.2}.Develop(frame)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s at t=%.1fs, sea=%+v, seed=%d, extent=%.1fm, grid=%dx%d, %dx%d, one ray", path, seconds, sea, seed, extent, grid, grid, width, height)
}
