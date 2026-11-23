package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// HatzFeedbackThree combines newly authored rest-space anatomical volumes
// and thin wings into one implicit skin. Its flight is illustrative.
func HatzFeedbackThree(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("hatz 03: nonfinite phase")
	}
	ell := func(p, c, r maths.Vec) float64 {
		q := p.Sub(c)
		return (math.Sqrt((q.X/r.X)*(q.X/r.X)+(q.Y/r.Y)*(q.Y/r.Y)+(q.Z/r.Z)*(q.Z/r.Z)) - 1) * math.Min(r.X, math.Min(r.Y, r.Z))
	}
	blend := func(a, b, k float64) float64 {
		h := math.Max(0, k-math.Abs(a-b)) / k
		return math.Min(a, b) - .25*k*h*h
	}
	field := func(p maths.Vec) float64 {
		d := ell(p, maths.V(-.35, 0, 0), maths.V(.60, .23, .18))
		d = blend(d, ell(p, maths.V(.30, .02, 0), maths.V(.28, .14, .10)), .10)
		d = blend(d, ell(p, maths.V(.65, .045, 0), maths.V(.32, .105, .075)), .08)
		d = blend(d, ell(p, maths.V(1.03, .055, 0), maths.V(.28, .15, .095)), .07)
		d = blend(d, ell(p, maths.V(1.30, .045, 0), maths.V(.24, .075, .055)), .05)
		// Flattened broad membranes grow outward from each flank.
		span := (math.Abs(p.Z) - .12) / .72
		t := math.Max(0, math.Min(1, span))
		leading := .35 - .22*t
		trailing := -.58 + .34*t
		half := (leading - trailing) * .5
		center := (leading + trailing) * .5
		wing := math.Max(math.Abs(p.X-center)-half, math.Max(math.Abs(p.Y+.025)-.040, math.Max(-span, span-1)*.10))
		d = blend(d, wing, .08)
		// Two independently placed hindlimbs; small lobes are not a skeletal fit.
		for _, s := range []float64{-1, 1} {
			d = blend(d, ell(p, maths.V(-.67, -.29, s*.12), maths.V(.11, .25, .055)), .075)
			d = blend(d, ell(p, maths.V(-.64, -.50, s*.14), maths.V(.13, .10, .05)), .055)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		s := 1.
		if p.Z < 0 {
			s = -1
		}
		u := math.Max(0, math.Min(1, (math.Abs(p.Z)-.12)/.70))
		w := u * u * (3 - 2*u)
		angle := .48 * w * math.Sin(phase-.35*u)
		dy := 2 * p.Y
		dz := 5 * (math.Abs(p.Z) - .12)
		z := s * (.6 + dy*math.Sin(angle) + dz*math.Cos(angle))
		y := dy*math.Cos(angle) - dz*math.Sin(angle)
		if math.Abs(p.Z) <= .12 {
			z = 5 * p.Z
			y = dy
		}
		return maths.V(2*p.X, y, z)
	}
	skin, err := bioformFieldMesh(field, warp)
	if err != nil {
		return nil, err
	}
	mat := &thing.Material{Name: "hatz 03 integrated anatomical study", Colour: maths.RGB(.42, .33, .26), Satin: .43, Gloss: .06}
	return []thing.Thing{{Shape: skin, Material: mat, Motion: thing.Still(at)}}, nil
}
