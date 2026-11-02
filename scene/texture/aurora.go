package texture

import (
	"github.com/janearc/slayer/maths"
)

// Aurora is an aurora's colour by its height above a body: a Fringe at
// the very bottom, Bottom above it, blending into Top through the top
// of the shell, between Low and High above Radius.
//
// The earth's is purple at the fringe, the green of oxygen's low line
// wholly from a sixth to a third of the way up, and the red of its high
// line toward the top; a giant's is ultraviolet and is shown in blues.
// the fractions and the colours are chosen, not derived from the lines.
//
// it cuts nothing outside Low and High, which the aurora medium's density
// does; a shell of no height is all Bottom.
type Aurora struct {
	Radius              float64
	Low, High           float64
	Fringe, Bottom, Top maths.Light
}

// At is the colour at p's height.
func (a Aurora) At(p maths.Vec) maths.Light {
	if a.High <= a.Low {
		return a.Bottom
	}
	h := (p.Len() - a.Radius - a.Low) / (a.High - a.Low)
	switch {
	case h < 0.15:
		t := maths.Smooth(0, 0.15, h)
		return a.Fringe.Scale(1 - t).Add(a.Bottom.Scale(t))
	case h < 0.35:
		return a.Bottom
	default:
		t := maths.Smooth(0.35, 1, h)
		return a.Bottom.Scale(1 - t).Add(a.Top.Scale(t))
	}
}
