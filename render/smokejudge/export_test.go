//go:build draw

package smokejudge

import (
	"image"

	"github.com/janearc/slayer/scene/smoke"
)

// the unexported words the tests in smokejudge_test check on their own
var (
	VerdictWord = verdictWord
	RaysFor     = raysFor
	SmallRays   = smallRays
)

// the two sizes, for RaysFor
var (
	FullSize  = fullSize
	SmallSize = smallSize
)

// Verdict is how a smoke fared when Judge judged it.
type Verdict struct {
	got *verdictOf
}

// Honest is true when the terminal's picture is the real one, coarser.
func (verdict *Verdict) Honest() bool { return verdict.got.honest() }

// Compared is how many terminal pixels were compared.
func (verdict *Verdict) Compared() int { return verdict.got.compared() }

// Pixels is how many terminal pixels the picture has.
func (verdict *Verdict) Pixels() int { return verdict.got.pixels() }

// Pictures is a smoke's two pictures, the terminal's and the real one, as
// both draws them, at full size or small, at the smoke's own fineness.
func Pictures(made smoke.Smoke, small bool) (terminalPicture,
	realPicture image.Image, err error) {
	at := scene{}
	at.world, at.eye = made.Scene()
	sz := fullSize
	if small {
		sz = smallSize
	}
	return both(made, at, sz, Fineness(at.world))
}

// Judge draws smoke made both ways and judges them, as judge does, and is
// nil when it cannot be drawn.
func Judge(made smoke.Smoke, small bool, k int, fresh bool) (*Verdict, error) {
	got, err := judge(made, small, k, fresh)
	if err != nil {
		return nil, err
	}
	return &Verdict{got}, nil
}
