//go:build draw && darwin

package gpu

import (
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGravityCompleteImageMatchesCPU(t *testing.T) {
	world := thing.NewWorld()
	world.Fields = append(world.Fields, &thing.Field{Mass: 1, Reach: 40})
	world.Sky = sky.Stars{Dark: maths.Grey(.15), Seed: 3}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 3}, Material: &thing.Material{Glow: maths.RGB(.8, .3, .1)}, Motion: at(7, 0, -9)})
	// Foreground surfaces must win before entry; background ones must be bent.
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Glow: maths.RGB(.2, .8, .3)}, Motion: at(-3, 2, 45)})
	eye := maths.Subspace{Origin: maths.V(0, 0, 60)}
	within(t, world, eye, 48, 32, 1, tolerance{mean: .003, largest: .03})
}

func TestGravityCompleteRingMovesOnGPU(t *testing.T) {
	world := thing.NewWorld()
	body, err := template.Quasar()
	if err != nil {
		t.Fatal(err)
	}
	body = body.WithoutJets()
	if err := body.Place(world, maths.Vec{}, 1/body.Radius(), 0); err != nil {
		t.Fatal(err)
	}
	for _, v := range world.Volumes {
		if thin, ok := v.Material.Shine.(texture.Thin); ok {
			thin.Streaks = .45
			thin.Omega *= 1000000
			v.Material.Shine = thin
		}
	}
	from := maths.V(0, 8, 45)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 2.6, 0))}
	lens := film.Pinhole{Wide: 1 * maths.Degree}
	renderer, err := NewRenderer(32)
	if err != nil {
		t.Fatal(err)
	}
	var previous []float32
	for _, moment := range []time.Duration{0, 2 * time.Second} {
		drawn, err := renderer.Sampled(world, eye, lens, moment, 48, 32, 1)
		if err != nil {
			t.Fatal(err)
		}
		if timing := renderer.LastTiming(); timing.Encode <= 0 || timing.Compile <= 0 {
			t.Fatal("renderer stage timing not recorded", timing)
		}
		if renderer.LastTiming().SeedsCached != (moment > 0) {
			t.Fatal("stationary camera seed reuse lost", moment)
		}
		truth, err := slayer.Sampled(world, eye, lens, moment, 48, 32, 1)
		if err != nil {
			t.Fatal(err)
		}
		if dir := os.Getenv("SLAYER_GRAVITY_CAPTURE_DIR"); dir != "" {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			for name, picture := range map[string]*frame.Frame{"cpu": truth, "metal": drawn} {
				file, err := os.Create(filepath.Join(dir, "ring-"+moment.String()+"-"+name+".png"))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, (colour.Tone{Knee: .8, Top: 16, Bloom: 1.5}).Develop(picture))
				closeErr := file.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
			}
		}
		found := compare(truth, drawn, 48, 32)
		for y := range 32 {
			for x := range 48 {
				a, b := truth.At(x, y), drawn.At(x, y)
				if !sameDepth(a.Depth, b.Depth) {
					t.Logf("ring depth %d,%d CPU=%g GPU=%g lightCPU=%g GPU=%g", x, y, a.Depth, b.Depth, a.R, b.R)
				}
			}
		}
		t.Logf("ring at %s: %+v", moment, found)
		if found.mean > .01 || found.largest > .05 || found.cover > 0 || found.depth > 0 {
			t.Fatal("GPU ring differs from CPU", found)
		}
		var samples []float32
		changed := 0
		for y := range 32 {
			for x := range 48 {
				p := drawn.At(x, y)
				samples = append(samples, p.R)
				if previous != nil && p.R != previous[y*48+x] {
					changed++
				}
			}
		}
		if previous != nil && changed < 100 {
			t.Fatal("GPU disc light did not move", changed)
		}
		previous = samples
	}
}

func TestGravityCompleteDiscMatchesCPU(t *testing.T) {
	world := thing.NewWorld()
	body, err := template.Quasar()
	if err != nil {
		t.Fatal(err)
	}
	body = body.WithoutJets()
	if err := body.Place(world, maths.Vec{}, 1/body.Radius(), 0); err != nil {
		t.Fatal(err)
	}
	from := maths.V(0, 8, 45)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
	within(t, world, eye, 48, 32, 1, tolerance{mean: .02, largest: .08})
}
