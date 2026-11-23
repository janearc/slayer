package thing

import (
	"slices"
	"sort"
	"time"

	"github.com/janearc/slayer/maths"
)

// The numbers a keyed run is worked in, each true by definition.
const (
	keyingNought = 0 // defined: no amount, and the first place in a list
	keyingUnity  = 1 // defined: one whole, the end of a run
	keyingPair   = 2 // defined: two, the halves a span is cut into
	keyingTriple = 3 // defined: the weight of a cubic bezier's middle terms
	keyingFirst  = 0 // defined: the first place in a list
	keyingNext   = 1 // defined: the place after another
)

// The control points of css's eases, and how many halvings find the time
// on a curve.
var (
	keyingNear     = pick("keyed-ease-near")
	keyingLift     = pick("keyed-ease-lift")
	keyingInStart  = pick("keyed-ease-in-start")
	keyingOutEnd   = pick("keyed-ease-out-end")
	keyingHalvings = int(pick("keyed-halvings"))
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

// keyingPlaced is a subspace at a place, unturned and of no scale named.
func keyingPlaced(place maths.Vec) maths.Subspace {
	var orientation maths.Orientation
	return maths.NewSubspace(place, orientation, keyingNought)
}

// At is the thing's subspace at a time: the share of the run is eased
// once, over the whole run, and the keyframes are blended at the eased
// share. css eases each stretch between keyframes in turn; this does
// not. a run of no length, or of no frames, stands at its base.
func (keyed Keyed) At(elapsed time.Duration) maths.Subspace {
	if keyed.Length <= keyingNought || len(keyed.Frames) == keyingNought {
		return keyingPlaced(keyed.Base)
	}
	share := float64(elapsed) / float64(keyed.Length)
	if share > keyingUnity && !keyed.Hold {
		return keyingPlaced(keyed.Base)
	}
	share = eased(keyed.Ease, share)
	frames := keyed.Frames
	first := frames[keyingFirst]
	if share <= first.Share {
		return keyingPlaced(first.At)
	}
	last := frames[len(frames)-keyingUnity]
	if share >= last.Share {
		return keyingPlaced(last.At)
	}
	return keyingBetween(frames, share)
}

// keyingPast asks, of the frames, whether a frame is past a share.
type keyingPast struct {
	frames []Keyframe
	share  float64
}

// past is whether the frame at a place is further on than the share.
func (asked keyingPast) past(place int) bool {
	return asked.frames[place].Share > asked.share
}

// keyingBetween is the place a share of the run has reached, blended from
// the two frames it falls between.
func keyingBetween(frames []Keyframe, share float64) maths.Subspace {
	asked := keyingPast{frames: frames, share: share}
	i := sort.Search(len(frames), asked.past) - keyingUnity
	before, after := frames[i], frames[i+keyingNext]
	k := (share - before.Share) / (after.Share - before.Share)
	step := after.At.Sub(before.At).Scale(k)
	return keyingPlaced(before.At.Add(step))
}

// easeNames are the eases a run may be named by.
var easeNames = []string{"linear", "ease", "ease-in", "ease-out",
	"ease-in-out"}

// Eases are the eases a run may be named by, a copy a caller may change.
func Eases() []string { return slices.Clone(easeNames) }

// keyingCurve is the two control points of a cubic bezier from nought to
// one, as css names them: the first's x and y, then the second's.
type keyingCurve struct {
	firstX, firstY, secondX, secondY float64
}

// keyingCurveOf is the curve a name stands for, with css's control
// points, and whether the name stands for one. no name is ease.
func keyingCurveOf(name string) (keyingCurve, bool) {
	switch name {
	case "", "ease":
		return keyingCurve{keyingNear, keyingLift, keyingNear,
			keyingUnity}, true
	case "ease-in":
		return keyingCurve{keyingInStart, keyingNought, keyingUnity,
			keyingUnity}, true
	case "ease-out":
		return keyingCurve{keyingNought, keyingNought, keyingOutEnd,
			keyingUnity}, true
	case "ease-in-out":
		return keyingCurve{keyingInStart, keyingNought, keyingOutEnd,
			keyingUnity}, true
	}
	return keyingCurve{}, false
}

// keyingBezier is a cubic bezier's value at a place along it, from nought
// to one, for two control values. Each term is rounded as a float64 before
// they are added, as the language says a conversion does, so no machine
// fuses a multiply into an add and the curve is the same on every one: at a
// half with css's control points it is exactly a half, where the solving
// for a share meets it.
func keyingBezier(near, far, along float64) float64 {
	remaining := keyingUnity - along
	first := float64(keyingTriple * remaining * remaining * along * near)
	second := float64(keyingTriple * remaining * along * along * far)
	return first + second + float64(along*along*along)
}

// solved is the curve's value at a share on its time axis, which is
// found by halving the span the share is in.
func (curve keyingCurve) solved(share float64) float64 {
	low, high := float64(keyingNought), float64(keyingUnity)
	for range keyingHalvings {
		middle := (low + high) / keyingPair
		if keyingBezier(curve.firstX, curve.secondX, middle) < share {
			low = middle
		} else {
			high = middle
		}
	}
	return keyingBezier(curve.firstY, curve.secondY, (low+high)/keyingPair)
}

// eased is a share of a run eased by the cubic bezier each name stands
// for, with css's control points, solved for the time axis by forty
// halvings. no name is ease; a name that is none of them is linear.
//
// A share at or below nought is nought and a share at or above one is
// one, for every name: the ends of a run are its ends.
func eased(name string, share float64) float64 {
	if share <= keyingNought {
		return keyingNought
	}
	if share >= keyingUnity {
		return keyingUnity
	}
	curve, known := keyingCurveOf(name)
	if !known {
		return share
	}
	return curve.solved(share)
}
