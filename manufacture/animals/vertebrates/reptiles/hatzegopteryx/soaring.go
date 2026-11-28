package hatzegopteryx

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// NewSoaring builds an authored gliding pose. Its long rostrum and hindlimbs
// follow azhdarchid comparisons in Witton and Naish (2008); the robust neck is
// informed by Naish and Witton (2017). The complete animal, membrane outline,
// colours and pose remain artistic hypotheses, not measured Hatzegopteryx.
func NewSoaring(profile reptiles.Profile, at maths.Subspace) ([]thing.Thing, error) {
	hide := &thing.Material{Name: "soaring pterosaur invented hide", Colour: maths.RGB(.31, .25, .19), Satin: .4, Gloss: .05}
	wing := &thing.Material{Name: "soaring pterosaur inferred membrane", Colour: maths.RGB(.25, .15, .105), Satin: .4, Gloss: .04}
	beak := &thing.Material{Name: "soaring pterosaur invented beak", Colour: maths.RGB(.18, .12, .08), Satin: .3}
	eye := &thing.Material{Name: "soaring pterosaur eye", Colour: maths.RGB(.012, .009, .007), Gloss: .3}
	var parts []thing.Thing
	loft := func(points []maths.Vec, radii [][2]float64, material *thing.Material) error {
		piece, err := geometry.JointLoft(points, radii, material, at, func(p maths.Vec) maths.Vec { return p })
		if err == nil {
			parts = append(parts, piece)
		}
		return err
	}
	v := maths.V
	// A compact body, distinct stout neck, and separate tapered skull are
	// picked shapes, not fossil-derived measurements.
	if err := loft([]maths.Vec{v(-1.25, 2.66, 0), v(-.8, 2.68, 0), v(-.25, 2.7, 0), v(.35, 2.7, 0), v(.68, 2.73, 0), v(1.15, 2.73, 0), v(1.65, 2.72, 0)}, [][2]float64{{.025, .025}, {.25, .21}, {.36, .31}, {.29, .26}, {.17, .16}, {.14, .13}, {.12, .11}}, hide); err != nil {
		return nil, err
	}
	if err := loft([]maths.Vec{v(1.58, 2.72, 0), v(1.9, 2.76, 0), v(2.3, 2.75, 0), v(3.05, 2.69, 0), v(4.15, 2.58, 0)}, [][2]float64{{.12, .11}, {.22, .19}, {.19, .14}, {.095, .07}, {.004, .004}}, beak); err != nil {
		return nil, err
	}
	for _, side := range []float64{-1, 1} {
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .045}, Material: eye, Motion: thing.Still(maths.NewSubspace(v(1.99, 2.83, side*.17), maths.Identity, 1).Within(at))})
		shoulder, elbow, wrist, tip, inner := hatzWingControlsWithProfile(profile, 0, 0, side)
		if err := loft([]maths.Vec{shoulder, elbow, wrist, tip}, [][2]float64{{.10, .09}, {.075, .065}, {.06, .05}, {.012, .012}}, hide); err != nil {
			return nil, err
		}
		membrane, err := hatzWingSurfaceWithProfile(profile, shoulder, elbow, wrist, tip, inner, 0, side)
		if err != nil {
			return nil, err
		}
		parts = append(parts, thing.Thing{Shape: membrane, Material: wing, Motion: thing.Still(at)})
		// Trailing legs and toes make the silhouette legible in flight; their
		// position and the separate surfaces are chosen rather than measured.
		hip, knee, foot := v(-.72, 2.56, side*.20), v(-1.23, 2.43, side*.35), v(-1.91, 2.44, side*.40)
		if err := loft([]maths.Vec{hip, knee, foot}, [][2]float64{{.10, .09}, {.065, .055}, {.035, .03}}, hide); err != nil {
			return nil, err
		}
		for digit := -1; digit <= 1; digit++ {
			start := foot.Add(v(0, 0, float64(digit)*.035))
			if err := loft([]maths.Vec{start, start.Add(v(-.17, -.025, float64(digit)*.035))}, [][2]float64{{.022, .018}, {.004, .004}}, beak); err != nil {
				return nil, err
			}
		}
	}
	return parts, nil
}
