//go:build darwin

package main

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestVisibleMoonIlluminatesOnBothGPUBackends(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "software", true: "native"}[native], func(t *testing.T) {
			world, eye, lens := moonDiscLightFixture(t)
			sample := gpu.SampledMeasured
			if native {
				sample = gpu.SampledAccelerated
			}
			lit, _, err := sample(world, eye, lens, 0, 1, 1, 1)
			if errors.Is(err, metal.ErrNoDevice) {
				t.Skip("no gpu here")
			}
			if err != nil {
				t.Fatal(err)
			}
			if p := lit.Pix[0]; p.R <= 0 || p.G <= 0 || p.B <= 0 {
				t.Fatal("Moon blocked its paired light", p)
			}
			world.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}, Material: &thing.Material{},
				Motion: thing.Still(maths.Subspace{Origin: skyward(115, 28).Scale(10)})})
			shadow, _, err := sample(world, eye, lens, 0, 1, 1, 1)
			if err != nil {
				t.Fatal(err)
			}
			if p := shadow.Pix[0]; p.R != 0 || p.G != 0 || p.B != 0 {
				t.Fatal("ordinary geometry stopped occluding moonlight", p)
			}
		})
	}
}
