package manufacture

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// BurnSpec describes a volumetric fire independently of any fuel geometry.
// Flame and Smoke retain their packages' documented physical units. Smoke is
// optional. Albedo is smoke's authored scattering share, not a measured fuel
// property. LightCells is the source quadrature resolution, three to eight.
type BurnSpec struct {
	Sparks     *embers.Spec
	Layout     FireLayout
	Flame      flame.Fire
	Smoke      *plume.Fire
	Albedo     float64
	LightCells int
}

type burner struct {
	spec   BurnSpec
	bounds maths.Box
}

// Burn checks a fire recipe. It copies the optional smoke description.
func Burn(spec BurnSpec) (burner, error) {
	if spec.LightCells < 3 || spec.LightCells > 8 {
		return burner{}, errors.New(
			"manufacture: light cells must be from 3 to 8")
	}
	body, err := makeBurningField(spec, 1)
	if err != nil {
		return burner{}, err
	}
	bounds := body.Bounds()
	if spec.Sparks != nil {
		copied := *spec.Sparks
		spec.Sparks = &copied
		if _, err := embers.New(copied); err != nil {
			return burner{}, err
		}
	}
	if spec.Smoke != nil {
		if !(spec.Albedo >= 0 && spec.Albedo <= 1) {
			return burner{}, errors.New(
				"manufacture: smoke albedo must lie in [0, 1]")
		}
		copied := *spec.Smoke
		spec.Smoke = &copied
		column, err := plume.New(copied)
		if err != nil {
			return burner{}, err
		}
		bounds = enclosing(bounds, column.Bounds())
	}
	if !burnFinite(bounds.Min.Len()) || !burnFinite(bounds.Max.Len()) {
		return burner{},
			errors.New("manufacture: nonfinite fire extent")
	}
	return burner{spec: spec, bounds: bounds}, nil
}

// Bounds encloses the local flame and optional smoke, before placement.
func (made burner) Bounds() maths.Box {
	return made.bounds
}

// Place makes time-aware volumes and source descriptions at the supplied time.
// Rebuild the light samples for each animation frame. A placement's uniform
// scale multiplies each source's physical cell volume by its cube.
func (made burner) Place(at maths.Subspace, exposure float64,
	when time.Duration) ([]thing.Volume, []flame.Source, error) {
	if !burnPlacement(at) {
		return nil, nil,
			errors.New("manufacture: invalid fire placement")
	}
	body, err := makeBurningField(made.spec, exposure)
	if err != nil {
		return nil, nil, err
	}
	volumes := []thing.Volume{{Medium: body, Material: &thing.Material{
		Name: "flame", Shine: body.Shine()}, Motion: thing.Still(at)}}
	if made.spec.Smoke != nil {
		column, err := plume.New(*made.spec.Smoke)
		if err != nil {
			return nil, nil, err
		}
		volumes = append(volumes, thing.Volume{Medium: column,
			Material: &thing.Material{
				Name:   "fire smoke",
				Colour: maths.Grey(made.spec.Albedo)},
			Motion: thing.Still(at)})
	}
	var lights []flame.Source
	for _, source := range body.Tick(when).(burningField).Sources(
		made.spec.LightCells) {
		lights = append(lights, source.Placed(at))
	}
	return volumes, lights, nil
}

// burnFinite rejects nonfinite model and placement values.
func burnFinite(value float64) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0)
}

// A fire uses a positive uniform scale and orthonormal axes, like a mesh group.
func burnPlacement(space maths.Subspace) bool {
	if !burnFinite(space.Origin.Len()) || !burnFinite(space.Scale) ||
		space.Scale < 0 {
		return false
	}
	x, y, z := space.Orientation.Out(maths.XAxis),
		space.Orientation.Out(maths.YAxis),
		space.Orientation.Out(maths.ZAxis)
	return math.Abs(x.Len()-1) < 1e-9 && math.Abs(y.Len()-1) < 1e-9 &&
		math.Abs(z.Len()-1) < 1e-9 &&
		math.Abs(x.Dot(y)) < 1e-9 && math.Abs(x.Dot(z)) < 1e-9 &&
		math.Abs(y.Dot(z)) < 1e-9 && x.Cross(y).Dot(z) > 0
}
