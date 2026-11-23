package pier

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/internal/beams"
	"github.com/janearc/slayer/scene/thing"
)

// Wheel describes an observation wheel placed on a deck. Radius reaches the
// gondola pivots. Phase rotates the wheel while the cabins stay upright.
type Wheel struct {
	Base                          maths.Vec
	Yaw, Phase, Radius, HubHeight float64
	Gondolas                      int
	Frame, Rim, Gondola, Windows  *thing.Material
}

// NewWheel makes the twin rims, spokes, axle, legs and hanging cabins.
func NewWheel(w Wheel) ([]thing.Thing, error) {
	if !positive(w.Radius, w.HubHeight) || !finite(w.Yaw) ||
		!finite(w.Phase) || !finite(w.Base.X) || !finite(w.Base.Y) ||
		!finite(w.Base.Z) || w.HubHeight < w.Radius*1.18 ||
		w.Gondolas < 8 || w.Gondolas > 64 ||
		w.Frame == nil || w.Rim == nil || w.Gondola == nil || w.Windows == nil {
		return nil, fmt.Errorf("pier: invalid observation wheel")
	}
	var a assembly
	yaw := maths.Rotation(maths.Up, w.Yaw)
	local := func(x, y, z float64) maths.Vec {
		return w.Base.Add(yaw.Out(maths.V(x, y, z)))
	}
	r := w.Radius
	h := w.HubHeight
	for _, side := range [...]float64{-1, 1} {
		z := r * .19 * side
		hub := local(0, h, z)
		for _, x := range [...]float64{-1, 1} {
			a.take(beams.Rod(local(.44*r*x, 0, 1.65*z), hub,
				.072*r, w.Frame))
		}
	}
	a.take(beams.Rod(local(0, h, -.22*r), local(0, h, .22*r), .13*r, w.Frame))
	for i := range w.Gondolas * 3 {
		angle := w.Phase + float64(i)*2*math.Pi/float64(w.Gondolas*3)
		next := angle + 2*math.Pi/float64(w.Gondolas*3)
		x, y := r*math.Cos(angle), h+r*math.Sin(angle)
		for _, side := range [...]float64{-1, 1} {
			z := side * r * .145
			at := local(x, y, z)
			a.take(beams.Rod(at, local(r*math.Cos(next), h+r*math.Sin(next), z),
				r*.029, w.Rim))
			if i%3 == 0 {
				a.take(beams.Rod(local(0, h, z), at, r*.017, w.Frame))
			}
		}
		if i%3 == 0 {
			a.cabin(w, yaw, local(x, y, 0))
		}
	}
	return a.parts, a.err
}

// cabin hangs a gondola from its pivot without applying the wheel phase.
func (a *assembly) cabin(w Wheel, yaw maths.Orientation, pivot maths.Vec) {
	r := w.Radius
	for _, side := range [...]float64{-1, 1} {
		z := side * r * .10
		top := pivot.Add(yaw.Out(maths.V(0, 0, z)))
		a.take(beams.Rod(top, top.Add(maths.V(0, -.11*r, 0)), .023*r, w.Frame))
	}
	a.take(beams.Box(pivot.Add(maths.V(0, -.147*r, 0)),
		maths.V(.175*r, .085*r, .23*r), yaw, w.Gondola))
	a.take(beams.Box(pivot.Add(maths.V(0, -.094*r, 0)),
		maths.V(.155*r, .071*r, .215*r), yaw, w.Windows))
	a.take(beams.Box(pivot.Add(maths.V(0, -.055*r, 0)),
		maths.V(.181*r, .024*r, .236*r), yaw, w.Gondola))
}
