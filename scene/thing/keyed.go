package thing

import (
	"slices"
	"sort"
	"time"

	"github.com/janearc/slayer/maths"
)

// A Keyframe is where a keyed thing is at a share of its run: the share,
// from nought to one, and its place there.
type Keyframe struct {
	Share float64
	At    maths.Vec
}

// Keyed is a thing carried through keyframes, as css animates one: the
// frames in order of their share, how long the run is, the ease it runs
// by, and whether it holds its last place after the run.
//
// without Hold it goes back to Base, where it stands before the run,
// which is what css does and is seldom what is meant.
type Keyed struct {
	Frames []Keyframe
	Length time.Duration
	Ease   string // linear, ease, ease-in, ease-out, ease-in-out
	Hold   bool
	Base   maths.Vec
}

// At is the thing's subspace at a time: the share of the run is eased
// once, over the whole run, and the keyframes are blended at the eased
// share. css eases each stretch between keyframes in turn; this does
// not. a run of no length, or of no frames, stands at its base.
func (keyed Keyed) At(t time.Duration) maths.Subspace {
	if keyed.Length <= 0 || len(keyed.Frames) == 0 {
		return maths.Subspace{Origin: keyed.Base}
	}
	x := float64(t) / float64(keyed.Length)
	if x < 0 {
		x = 0
	}
	if x > 1 {
		if !keyed.Hold {
			return maths.Subspace{Origin: keyed.Base}
		}
		x = 1
	}
	x = Eased(keyed.Ease, x)
	frames := keyed.Frames
	if x <= frames[0].Share {
		return maths.Subspace{Origin: frames[0].At}
	}
	last := frames[len(frames)-1]
	if x >= last.Share {
		return maths.Subspace{Origin: last.At}
	}
	i := sort.Search(len(frames), func(i int) bool {
		return frames[i].Share > x
	}) - 1
	p, q := frames[i], frames[i+1]
	k := (x - p.Share) / (q.Share - p.Share)
	return maths.Subspace{Origin: p.At.Add(q.At.Sub(p.At).Scale(k))}
}

// Eases are the eases a run may be named by.
var Eases = []string{"linear", "ease", "ease-in", "ease-out", "ease-in-out"}

// IsEase is whether a name is one of them.
func IsEase(name string) bool { return slices.Contains(Eases, name) }

// Eased is a share of a run eased by the cubic bezier each name stands
// for, with css's control points, solved for the time axis by forty
// halvings. no name is ease; a name that is none of them is linear.
func Eased(name string, x float64) float64 {
	var p1x, p1y, p2x, p2y float64
	switch name {
	case "", "ease":
		p1x, p1y, p2x, p2y = 0.25, 0.1, 0.25, 1
	case "ease-in":
		p1x, p1y, p2x, p2y = 0.42, 0, 1, 1
	case "ease-out":
		p1x, p1y, p2x, p2y = 0, 0, 0.58, 1
	case "ease-in-out":
		p1x, p1y, p2x, p2y = 0.42, 0, 0.58, 1
	default:
		return x
	}
	bez := func(a, b, t float64) float64 {
		u := 1 - t
		return 3*u*u*t*a + 3*u*t*t*b + t*t*t
	}
	lo, hi := 0.0, 1.0
	for range 40 {
		mid := (lo + hi) / 2
		if bez(p1x, p2x, mid) < x {
			lo = mid
		} else {
			hi = mid
		}
	}
	return bez(p1y, p2y, (lo+hi)/2)
}
