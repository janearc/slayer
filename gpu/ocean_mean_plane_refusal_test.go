//go:build darwin

package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

func TestOceanInterfaceEventsAreRefusedBeforeGPUDeviceUse(t *testing.T) {
	core, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	continued, err := ocean.NewInfiniteMeanContinuation(core, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	boundary := world.Add(thing.Thing{Shape: continued,
		Material: &thing.Material{Through: 1, Gloss: 1, Index: 1.333}})
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		InterfaceEvents: 2}
	_, err = Sampled(world, maths.Local(), film.Pinhole{}, 0, 1, 1, 1)
	if err == nil || !strings.Contains(err.Error(),
		"CPU-only ocean interface-event transport") {
		t.Fatalf("GPU ocean-event refusal=%v", err)
	}
}

func TestGPURefusesUnvalidatedDirectMoonBeforeDeviceUse(t *testing.T) {
	world := thing.NewWorld()
	world.OceanSurface = &thing.OceanSurface{DirectMoonMeanPlane: true}
	_, err := Sampled(world, maths.Local(), film.Pinhole{}, 0, 1, 1, 1)
	if err == nil || !strings.Contains(err.Error(),
		"source route does not match its Sun/Moon selection") {
		t.Fatalf("direct Moon GPU refusal = %v", err)
	}
}

// This uses an encoder-unsupported shape as a safety guard: if a public API
// loses the mean-plane refusal, encoding still fails before opening Metal.
func TestPublicGPUAPIsRefuseMeanPlaneDirectSunBeforeMetal(t *testing.T) {
	core, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	continuation, err := ocean.NewFlatMeanContinuation(core, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.OceanSurface = &thing.OceanSurface{DirectSunMeanPlane: true}
	world.Add(thing.Thing{Shape: continuation,
		Material: &thing.Material{Name: "unused guard shape"}})
	eye, lens := maths.Local(), film.Pinhole{}
	check := func(label string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(),
			"CPU-only ocean mean-plane direct-sun transport") {
			t.Errorf("%s error = %v, want direct-sun refusal before device use",
				label, err)
		}
	}

	_, err = Sampled(world, eye, lens, 0, 1, 1, 1)
	check("Sampled", err)
	_, _, err = SampledWithInitialMedia(Software, world, eye, lens, 0,
		1, 1, 1, []trace.InitialMedium{{Index: 1.33}})
	check("SampledWithInitialMedia", err)

	renderer, err := NewRenderer(16)
	if err != nil {
		t.Fatal(err)
	}
	_, err = renderer.Sampled(world, eye, lens, 0, 1, 1, 1)
	check("Renderer.Sampled", err)
	_, err = renderer.SampledWithInitialMedia(world, eye, lens, 0,
		1, 1, 1, []trace.InitialMedium{{Index: 1.33}})
	check("Renderer.SampledWithInitialMedia", err)

	frames, err := NewFrames(Software)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = frames.Draw(world, 1, eye, lens, 0, 1, 1, 1)
	check("Frames.Draw", err)
	frames.Close()

	frames, err = NewFrames(Software)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = frames.DrawWithInitialMedia(world, 1, eye, lens, 0,
		1, 1, 1, []trace.InitialMedium{{Index: 1.33}})
	check("Frames.DrawWithInitialMedia", err)
	frames.Close()
}
