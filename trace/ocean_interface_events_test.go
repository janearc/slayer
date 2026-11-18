package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	seaocean "github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

func oceanInterfaceEscapeFixture(t *testing.T) (*thing.World, look) {
	t.Helper()
	// The left face rises with slope 0.2 and the right face falls with slope 4.
	// The grazing incoming ray TIR-reflects from the left, then encounters the
	// right near its normal, where it must be able to refract back into air.
	field, err := shape.NewHeightField([]float64{
		-.4, 0, -8, -.4, 0, -8, -.4, 0, -8,
	}, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	interfaceShape, err := seaocean.NewInfiniteMeanContinuation(field, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	material := &thing.Material{Through: 1, Gloss: 1, Index: 1.333}
	boundary := &thing.Thing{Shape: interfaceShape, Material: material}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	world.Sky = sky.Gradient{Horizon: maths.Grey(1),
		Zenith: maths.Grey(1), Ground: maths.Light{}}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary}
	seen := look{ray: maths.Ray{At: maths.V(-1.5, -.35, 0),
		Dir: maths.V(1, .3, 0).Unit()}, holding: 1}
	seen.inside[0] = of(material)
	return world, seen
}

func TestOceanInterfaceEventsCanEscapeAfterTotalInternalReflection(t *testing.T) {
	world, seen := oceanInterfaceEscapeFixture(t)
	legacy := shade(stage(world, 0), seen).Light()
	world.OceanSurface.InterfaceEvents = 1
	limited := shade(stage(world, 0), seen).Light()
	world.OceanSurface.InterfaceEvents = 2
	escaped := shade(stage(world, 0), seen).Light()
	if legacy != (maths.Light{}) || limited != (maths.Light{}) {
		t.Fatalf("unfollowed paths should be black: legacy=%v one-event=%v", legacy, limited)
	}
	if escaped.R <= .1 || escaped.G <= .1 || escaped.B <= .1 {
		t.Fatalf("two-event water path cannot escape after TIR: %v", escaped)
	}
	leftNormal := maths.V(-.2, 1, 0).Unit()
	reflected := seen.ray.Dir.Sub(leftNormal.Scale(
		2 * seen.ray.Dir.Dot(leftNormal))).Unit()
	want := 1 - maths.Fresnel(reflected.Dot(maths.V(4, 1, 0).Unit()),
		1/1.333)
	for _, got := range []float64{escaped.R, escaped.G, escaped.B} {
		if math.Abs(got-want) > 1e-6 {
			t.Fatalf("escape radiance=%v, want Fresnel transmission %g", escaped, want)
		}
	}
}

func TestOceanInterfaceEventsRefuseInvalidBoundaryAndBudget(t *testing.T) {
	for _, fault := range []string{"negative", "too many", "missing", "opaque", "finite"} {
		t.Run(fault, func(t *testing.T) {
			world, _ := oceanInterfaceEscapeFixture(t)
			world.OceanSurface.InterfaceEvents = 2
			switch fault {
			case "negative":
				world.OceanSurface.InterfaceEvents = -1
			case "too many":
				world.OceanSurface.InterfaceEvents = thing.MaxOceanInterfaceEvents + 1
			case "missing":
				world.OceanSurface.Boundary = nil
			case "opaque":
				world.OceanSurface.Boundary.Material.Through = 0
			case "finite":
				field, err := shape.NewHeightField(make([]float64, 9), 3, 4)
				if err != nil {
					t.Fatal(err)
				}
				world.OceanSurface.Boundary.Shape = field
			}
			if err := validateOceanInterfaceEvents(world); err == nil {
				t.Fatal("accepted invalid ocean interface-event policy")
			}
		})
	}
}

func TestOceanInterfaceEventsLeaveUntaggedSurfaceAndDefaultAlone(t *testing.T) {
	world, seen := oceanInterfaceEscapeFixture(t)
	legacy := shade(stage(world, 0), seen).Light()
	world.OceanSurface = nil
	withoutOcean := shade(stage(world, 0), seen).Light()
	world.OceanSurface = &thing.OceanSurface{Boundary: &thing.Thing{},
		InterfaceEvents: 2}
	otherBoundary := shade(stage(world, 0), seen).Light()
	if legacy != withoutOcean || otherBoundary != withoutOcean {
		t.Fatalf("ocean policy changed an untagged surface: %v/%v/%v",
			legacy, withoutOcean, otherBoundary)
	}
}
