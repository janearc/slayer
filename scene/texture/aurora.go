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

// auroraFringeTop is how far up the shell, as a share of it, the fringe
// has blended into the bottom colour.
var auroraFringeTop = pickedNumber("aurora-fringe-top")

// auroraBottomTop is how far up the shell, as a share of it, the bottom
// colour begins to blend into the top.
var auroraBottomTop = pickedNumber("aurora-bottom-top")

// At is the colour at the point's height.
func (aurora Aurora) At(point maths.Vec) maths.Light {
	if aurora.High <= aurora.Low {
		return aurora.Bottom
	}
	height := (point.Len() - aurora.Radius - aurora.Low) /
		(aurora.High - aurora.Low)
	switch {
	case height < auroraFringeTop:
		share := maths.Smooth(nought, auroraFringeTop, height)
		return aurora.Fringe.Scale(unity - share).
			Add(aurora.Bottom.Scale(share))
	case height < auroraBottomTop:
		return aurora.Bottom
	default:
		share := maths.Smooth(auroraBottomTop, unity, height)
		return aurora.Bottom.Scale(unity - share).Add(aurora.Top.Scale(share))
	}
}
