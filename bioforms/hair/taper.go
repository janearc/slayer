package hair

import "github.com/janearc/slayer/maths"

// ShaftTaper controls where a strand begins to narrow and the share
// of its radius the profile tends towards at the tip. A zero value
// uses the original narrow, pointed shaft. The final cap meets the
// curve centre in either case.
type ShaftTaper struct {
	Start, TipShare float64
}

func (taper ShaftTaper) valid() bool {
	return taper.Start >= 0 && taper.Start < 1 &&
		taper.TipShare >= 0 && taper.TipShare <= 1
}

func (taper ShaftTaper) scale(t float64) float64 {
	if taper == (ShaftTaper{}) {
		return 1 - .95*maths.Smooth(.4, 1, t)
	}
	return 1 - (1-taper.TipShare)*maths.Smooth(taper.Start, 1, t)
}
