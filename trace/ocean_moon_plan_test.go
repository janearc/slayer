package trace

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	seaocean "github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestOceanMoonPlanCopiesTaggedEmitterAndRejectsDisplacement(t *testing.T) {
	world, water, boundary, _, toward := oceanSunFixture(t, .5)
	world.Lamps = world.Lamps[:1]
	ocean := world.OceanSurface
	ocean.DirectSunMeanPlane = false
	ocean.DirectMoonMeanPlane = true
	ocean.SunRoute = thing.OceanSunRouteMoon
	ocean.FillLampIndex = -1
	plan, err := PlanOceanMoon(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	if plan == nil || plan.Medium != water || plan.AirToward != toward ||
		plan.Strength.R != 10 || plan.HalfExtent != 10 || plan.LampIndex != 0 {
		t.Fatalf("plan did not preserve actual staged source: %+v", plan)
	}
	// Even an opaque foreign field must not acquire the interface exclusion
	// merely by naming the same medium. Public GPU preflight calls this plan.
	world.Add(thing.Thing{Shape: boundary.Shape,
		Material: &thing.Material{Through: 0, ClearMedium: water}})
	if _, err := PlanOceanMoon(world, 0); err == nil ||
		!strings.Contains(err.Error(), "outside the tagged interface") {
		t.Fatalf("foreign same-medium blocker gained the interface exemption: %v", err)
	}
	world.Things = world.Things[:len(world.Things)-1]
	field, err := shape.NewHeightField([]float64{0, .1, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Shape = field
	if _, err := PlanOceanMoon(world, 0); err == nil || !strings.Contains(err.Error(), "displaced water") {
		t.Fatalf("unsupported interface passed kernel plan: %v", err)
	}
	continuation, err := seaocean.NewInfiniteMeanContinuation(field, 0, 4)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Shape = continuation
	ocean.InfiniteMeanPlane = true
	if _, err := PlanOceanMoon(world, 0); err == nil || !strings.Contains(err.Error(), "exactly constant") {
		t.Fatalf("displaced infinite continuation was flattened: %v", err)
	}
	flat, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Shape, err = seaocean.NewInfiniteMeanContinuation(flat, 0, 4)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Motion = thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, -1))
	if _, err := PlanOceanMoon(world, 0); err == nil {
		t.Fatal("negative plane scale lost its orientation")
	}
}
