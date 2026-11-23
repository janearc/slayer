package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

type hatzFourMass struct {
	center, scale maths.Vec
	weight        float64
}

// HatzFeedbackFour is an original density sculpture and qualitative wing
// deformation. Appearance, tissue and motion are not fossil measurements.
func HatzFeedbackFour(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	return hatzFeedbackFour(phase, true, at)
}

// hatzFeedbackFour is the fourth study, a pterosaur drawn as masses, at a
// phase of its wing stroke in radians. It refuses a phase that is not
// finite.
func hatzFeedbackFour(phase float64, animated bool, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("hatz 04: nonfinite phase")
	}
	v := maths.V
	masses := []hatzFourMass{
		{v(-.80, 0, 0), v(.35, .20, .16), .95}, {v(-.38, .015, 0), v(.45, .24, .21), 1.1},
		{v(.02, .025, 0), v(.30, .20, .16), .95}, {v(.32, .07, 0), v(.28, .13, .10), .90},
		{v(.63, .11, 0), v(.25, .11, .085), .90}, {v(.88, .14, 0), v(.23, .15, .11), 1.02},
		{v(1.10, .11, 0), v(.24, .105, .08), .85}, {v(1.33, .08, 0), v(.20, .065, .055), .80},
	}
	for _, s := range []float64{-1, 1} {
		// Two distinct leg lobes merge to the body and to one another.
		masses = append(masses, hatzFourMass{v(-.63, -.23, s*.10), v(.10, .23, .055), .85}, hatzFourMass{v(-.58, -.47, s*.115), v(.08, .19, .046), .72}, hatzFourMass{v(-.48, -.59, s*.12), v(.13, .055, .05), .75})
		// Individually authored membrane masses; a summed field is not a measured wing.
		masses = append(masses, hatzFourMass{v(-.12, -.01, s*.21), v(.34, .085, .22), .95}, hatzFourMass{v(-.16, -.01, s*.45), v(.31, .065, .24), .90}, hatzFourMass{v(-.21, 0, s*.68), v(.24, .055, .22), .86}, hatzFourMass{v(-.24, .015, s*.84), v(.12, .045, .11), .80})
	}
	field := func(p maths.Vec) float64 {
		sum := 0.
		for _, m := range masses {
			q := p.Sub(m.center)
			x, y, z := q.X/m.scale.X, q.Y/m.scale.Y, q.Z/m.scale.Z
			if math.Abs(x) < 3 && math.Abs(y) < 3 && math.Abs(z) < 3 {
				sum += m.weight * math.Exp(-(x*x + y*y + z*z))
			}
		}
		return .42 - sum
	}
	original, err := bioformFieldMesh(field, func(p maths.Vec) maths.Vec { return p })
	if err != nil {
		return nil, err
	}
	cubic := func(x float64) float64 {
		a := math.Abs(x)
		if a < 1 {
			return (4 - 6*a*a + 3*a*a*a) / 6
		}
		if a < 2 {
			return math.Pow(2-a, 3) / 6
		}
		return 0
	}
	rest := make([]maths.Vec, len(original.Vertices))
	posed := make([]maths.Vec, len(rest))
	for i, p := range original.Vertices {
		rest[i] = maths.V(2*p.X, 2*p.Y, 5*p.Z)
		q := rest[i]
		span := math.Abs(p.Z)
		t := math.Max(0, math.Min(1, (span-.12)/.20))
		root := t * t * (3 - 2*t)
		if animated {
			q.Y += root * (.25*cubic((span-.30)/.30)*math.Sin(phase+.35) + .85*cubic((span-.58)/.30)*math.Sin(phase-.22) + 1.30*cubic((span-.82)/.30)*math.Sin(phase-.55))
		}
		posed[i] = q
	}
	normals := make([]maths.Vec, len(posed))
	for _, f := range original.Faces {
		n := posed[f[1]].Sub(posed[f[0]]).Cross(posed[f[2]].Sub(posed[f[0]]))
		for _, i := range f {
			normals[i] = normals[i].Add(n)
		}
	}
	skin, err := mesh.NewWithRestPoints(posed, rest, original.Faces, normals, true)
	if err != nil {
		return nil, err
	}
	mat := &thing.Material{Name: "hatz 04 speculative skin", Texture: hatzFourPaint{}, Colour: maths.RGB(.42, .33, .26), Satin: .42, Gloss: .06}
	return []thing.Thing{{Shape: skin, Material: mat, Motion: thing.Still(at)}}, nil
}

type hatzFourPaint struct{}

// At is the study's colour at a point: a dark eye on each side of the
// head, and the body's colour elsewhere.
func (hatzFourPaint) At(p maths.Vec) maths.Light {
	x, y, z := p.X/2, p.Y/2, p.Z/5
	eye := (x-.91)*(x-.91)/(.035*.035) + (y-.17)*(y-.17)/(.025*.025)
	if eye < 1 && math.Abs(z) > .07 {
		return maths.RGB(.018, .016, .014)
	}
	if x > 1.03 && x < 1.45 && math.Abs(y-.075) < .004 && math.Abs(z) > .025 {
		return maths.RGB(.16, .11, .08)
	}
	return maths.RGB(.42, .33, .26)
}
