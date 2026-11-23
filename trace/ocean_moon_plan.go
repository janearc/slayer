package trace

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// OceanMoonPlan is the validated calm-water subset of the CPU lunar
// route. Light is copied from the actual staged emitter, not a second lunar
// brightness model. Coefficients remain on its named optical medium.
type OceanMoonPlan struct {
	Medium                 *thing.ClearMedium
	Boundary               *thing.Thing
	AirToward              maths.Vec
	Strength               maths.Light
	Centre                 maths.Vec
	HalfExtent, Index      float64
	LampIndex              int
	MixedCloud, RaySpacing bool
	Unbounded              bool
}

// PlanOceanMoon prepares the first kernel-supported lunar interface. More
// general displaced CPU water and sky quadrature remain explicit
// refusals rather than silently changing their source model.
func PlanOceanMoon(world *thing.World, at time.Duration) (*OceanMoonPlan, error) {
	if world == nil || world.OceanSurface == nil || !world.OceanSurface.DirectMoonMeanPlane {
		return nil, nil
	}
	if err := validateOceanDirectSunWorld(world); err != nil {
		return nil, err
	}
	ocean := world.OceanSurface
	if (len(world.Volumes) != 0 && !clearCloudOnly(world)) ||
		ocean.ClearCloudSkySamples != 0 || ocean.SpotInscatter ||
		ocean.Boundary.Material.ClearMedium.DiffuseSkyInscatterSamples() != 0 {
		return nil, fmt.Errorf("trace: kernel Moon source needs finite calm heightfield water without cloud or sky quadrature")
	}
	level, extent, unbounded, err := kernelMoonPlane(ocean)
	if err != nil {
		return nil, err
	}
	space := ocean.Boundary.At(at)
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	centre := space.Out(maths.V(0, level, 0))
	x := space.Out(maths.V(1, level, 0)).Sub(centre)
	z := space.Out(maths.V(0, level, 1)).Sub(centre)
	if !finite(scale) || scale <= 0 || space.Orientation.Out(maths.Up).Unit() != maths.Up ||
		(!unbounded && (x.Y != 0 || z.Y != 0 || x.Z != 0 || z.X != 0 || x.X <= 0 || z.Z != x.X)) ||
		space.Origin != ocean.Mean {
		return nil, fmt.Errorf("trace: kernel Moon source requires an axis-aligned horizontal mean interface")
	}
	source := oceanDirectSunOf(world)
	if source == nil || !finite(source.strength.R, source.strength.G, source.strength.B) {
		return nil, fmt.Errorf("trace: kernel Moon source has no finite staged lunar emitter")
	}
	for _, object := range world.Things {
		if object != ocean.Boundary && object.Material != nil &&
			object.Material.ClearMedium.Defined() &&
			object.Material.ClearMedium.Name() == source.medium.Name() {
			return nil, fmt.Errorf("trace: kernel Moon source refuses its water medium on any object outside the tagged interface")
		}
	}
	halfExtent := extent * x.X / 2
	if unbounded {
		halfExtent = -1
	}
	for _, volume := range world.Volumes {
		if volume.Hidden() {
			continue
		}
		cloud, ok := volume.Medium.(medium.ClearCloud)
		if !ok {
			return nil, fmt.Errorf("trace: kernel lunar mixed transport requires explicit ClearCloud wrappers")
		}
		field := cloud.DensityField()
		if timed, ok := field.(medium.Timed); ok {
			field = timed.Tick(at)
		}
		switch field.(type) {
		case medium.PropellerCavitation, medium.Fizz:
		default:
			return nil, fmt.Errorf("trace: kernel lunar mixed transport " +
				"only supports native propeller or wrapped Fizz density")
		}
		if made := volume.Material; made != nil && (made.Texture != nil || made.Shine != nil) {
			return nil, fmt.Errorf("trace: kernel lunar cloud refuses spatial paint or emission")
		}
		bounds := field.Bounds()
		space := volume.At(at)
		for _, east := range []float64{bounds.Min.X, bounds.Max.X} {
			for _, up := range []float64{bounds.Min.Y, bounds.Max.Y} {
				for _, south := range []float64{bounds.Min.Z, bounds.Max.Z} {
					point := space.Out(maths.V(east, up, south))
					if point.Y >= centre.Y || (!unbounded && (math.Abs(point.X-centre.X) >= halfExtent || math.Abs(point.Z-centre.Z) >= halfExtent)) {
						return nil, fmt.Errorf("trace: kernel lunar cloud must remain wholly submerged inside the finite interface")
					}
				}
			}
		}
	}
	return &OceanMoonPlan{Medium: source.medium, Boundary: ocean.Boundary, AirToward: source.sunToward.Unit(),
		Strength: source.strength, Centre: centre, HalfExtent: halfExtent,
		Index: source.material.Dense(), LampIndex: source.sunIndex,
		MixedCloud: len(world.Volumes) > 0, RaySpacing: ocean.ClearCloudWaterStepMode == "ray-spacing", Unbounded: unbounded}, nil
}

// kernelMoonPlane returns the level and extent of the Moon plane under the
// ocean boundary, and whether it is unbounded. It fails for displaced water
// or for an infinite source without a continuation.
func kernelMoonPlane(ocean *thing.OceanSurface) (level, extent float64, unbounded bool, err error) {
	switch field := ocean.Boundary.Shape.(type) {
	case *shape.HeightField:
		if ocean.InfiniteMeanPlane {
			return 0, 0, false, fmt.Errorf("trace: infinite lunar source needs an actual continuation")
		}
		heights, _, size, _, _, _ := field.Pack()
		for _, height := range heights {
			if height != heights[0] {
				return 0, 0, false, fmt.Errorf("trace: kernel Moon source does not yet support displaced water")
			}
		}
		return heights[0], size, false, nil
	case *arid.InfiniteMeanContinuation:
		if mean, ok := field.ExactPlaneLevel(); ok && ocean.InfiniteMeanPlane {
			return mean, 0, true, nil
		}
		return 0, 0, false, fmt.Errorf("trace: kernel infinite lunar source requires exactly constant core and collar")
	default:
		return 0, 0, false, fmt.Errorf("trace: kernel Moon source requires a flat heightfield or exact calm infinite plane")
	}
}
