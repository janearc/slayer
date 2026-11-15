package section

import (
	"math"
	"reflect"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// A Change is what differs between a world at one moment and a world at
// another, worked out once for a frame and asked of each section.
type Change struct {
	before, after *thing.World
	was, now      time.Duration
	same, bent    bool
	drift         float64
	steady        map[*thing.Thing]bool
	still         map[*thing.Volume]bool
	movers        []ball
	lit, unlit    []thing.Lamp
}

// a ball is a place a thing or a volume that changed is, at one of the
// two moments: all its bounds lie within it.
type ball struct {
	centre maths.Vec
	radius float64
}

// Between is the change from one world at one moment to another world at
// another: the worlds of two frames of a film, which may be one world.
func Between(before, after *thing.World, was, now time.Duration) Change {
	change := Change{before: before, after: after, was: was, now: now,
		steady: map[*thing.Thing]bool{}, still: map[*thing.Volume]bool{}}
	change.bent = present(len(before.Fields)) || present(len(after.Fields))
	change.sortThings()
	change.sortVolumes()
	change.sortLamps()
	change.same = !change.bent && !change.reset() &&
		!present(len(change.movers)) &&
		!present(len(change.unlit)) &&
		reflect.DeepEqual(before.Haze, after.Haze)
	return change
}

// nowhere is the vector with no length, defined as the origin.
var nowhere maths.Vec

// present is whether a count is above none.
func present(count int) bool { return count > nothing }

// absent is whether an amount is none.
func absent(amount float64) bool { return amount == nothing }

// Allowing is the change with a drift allowed: the share of a volume's
// light from the lamps that did not change that the lamps that did may
// come to, and the section still be kept. With none, the default, a
// section with a volume in sight is kept only when nothing changed.
func (change Change) Allowing(drift float64) Change {
	change.drift = drift
	return change
}

// sortThings finds the things that are in both worlds, in the same place
// at both moments and not painted by a texture that moves: the steady
// ones. Every other thing, at each moment it is there, is a mover.
func (change *Change) sortThings() {
	later := map[*thing.Thing]bool{}
	for _, each := range change.after.Things {
		later[each] = true
	}
	for _, each := range change.before.Things {
		if later[each] && !timed(each.Material) &&
			where(each.Motion, change.was) == where(each.Motion, change.now) {
			change.steady[each] = true
			continue
		}
		change.moves(each.Shape.Bounds(), where(each.Motion, change.was))
	}
	for _, each := range change.after.Things {
		if !change.steady[each] {
			change.moves(each.Shape.Bounds(), where(each.Motion, change.now))
		}
	}
}

// sortVolumes finds the volumes that are in both worlds, in the same
// place at both moments, holding a medium that does not move and painted
// by no texture that moves. Every other volume is a mover.
func (change *Change) sortVolumes() {
	later := map[*thing.Volume]bool{}
	for _, each := range change.after.Volumes {
		later[each] = true
	}
	for _, each := range change.before.Volumes {
		_, moving := each.Medium.(medium.Timed)
		if later[each] && !moving && !timed(each.Material) &&
			where(each.Motion, change.was) == where(each.Motion, change.now) {
			change.still[each] = true
			continue
		}
		change.moves(each.Medium.Bounds(), where(each.Motion, change.was))
	}
	for _, each := range change.after.Volumes {
		if !change.still[each] {
			change.moves(each.Medium.Bounds(), where(each.Motion, change.now))
		}
	}
}

// sortLamps pairs the two worlds' lamps in order: a pair that is the
// same lamp is lit, steadily; every other lamp of either world is unlit
// at one moment or changed between them.
func (change *Change) sortLamps() {
	before, after := change.before.Lamps, change.after.Lamps
	for at := range max(len(before), len(after)) {
		switch {
		case at < len(before) && at < len(after) &&
			reflect.DeepEqual(before[at], after[at]):
			change.lit = append(change.lit, before[at])
		default:
			if at < len(before) {
				change.unlit = append(change.unlit, before[at])
			}
			if at < len(after) {
				change.unlit = append(change.unlit, after[at])
			}
		}
	}
}

// moves notes a mover: the ball about a box placed in a subspace.
func (change *Change) moves(box maths.Box, placed maths.Subspace) {
	centre, radius := ballOf(box, placed)
	change.movers = append(change.movers, ball{centre, radius})
}

// timed is whether any texture of a material moves.
func timed(made *thing.Material) bool {
	if made == nil {
		return false
	}
	for _, each := range []texture.Texture{made.Texture, made.Shine,
		made.Rough} {
		if _, moving := each.(texture.Timed); moving {
			return true
		}
	}
	return false
}

// selfLit is whether a material shows only its own light: its glow and
// its shine, with nothing of it that a lamp, the sky or another thing
// could light, shadow or show through.
func selfLit(made *thing.Material) bool {
	var dark maths.Light
	return made != nil && made.Colour == dark &&
		made.Texture == nil && made.Rough == nil && made.Radiant == nil &&
		absent(made.Gloss) && absent(made.Through) && absent(made.Satin) &&
		made.Crinkle == nil
}

// keeps is whether a section with this sight shows at the second moment
// the picture it showed at the first, so that the first can be kept. It
// may say no for a section that did not change. With no drift allowed it
// does not say yes for one that did; with a drift, a kept section's
// volumes may differ by the light the drift lets through.
//
// A section is kept when nothing in the world changed. Failing that, it
// is kept when the sky is the same, nothing that changed can be met by
// its rays, and all they can meet is steady things that show only their
// own light, and steady volumes whose light did not change by more than
// the drift and which nothing that changed can shadow. A world whose
// rays bend, or one with a haze through all of it, is drawn again.
func (change Change) keeps(sight Sight) bool {
	if change.same && sameSky(change) {
		return true
	}
	if change.bent || !sameSky(change) || change.hazed() || change.reset() {
		return false
	}
	for _, mover := range change.movers {
		if unbounded(mover.radius) || sight.meets(mover.centre, mover.radius) {
			return false
		}
	}
	for each := range change.steady {
		if !selfLit(each.Material) && !each.Hidden() &&
			ballMeets(sight, each.Shape.Bounds(), where(each.Motion,
				change.now)) {
			return false
		}
	}
	for each := range change.still {
		placed := where(each.Motion, change.now)
		if ballMeets(sight, each.Medium.Bounds(), placed) &&
			!change.lights(each.Medium.Bounds(), placed) {
			return false
		}
	}
	return true
}

// reset is whether the two worlds are drawn by different settings: the
// pixel's width they say, how their volumes are lit, or where their rays
// are counted from. Any of them changes every pixel.
func (change Change) reset() bool {
	before, after := change.before, change.after
	return before.Pixel != after.Pixel ||
		!reflect.DeepEqual(before.VolumeIllumination,
			after.VolumeIllumination) ||
		!reflect.DeepEqual(before.RayOriginReference,
			after.RayOriginReference)
}

// sameSky is whether the sky is the same at both moments.
func sameSky(change Change) bool {
	return reflect.DeepEqual(change.before.Sky, change.after.Sky)
}

// hazed is whether either world has a haze through all of it.
func (change Change) hazed() bool {
	var clear thing.Haze
	return !reflect.DeepEqual(change.before.Haze, clear) ||
		!reflect.DeepEqual(change.after.Haze, clear)
}

// ballOf is the ball about a box placed in a subspace: its centre, and
// its radius, which is under zero for a box with no end.
func ballOf(box maths.Box, placed maths.Subspace) (maths.Vec, float64) {
	across := box.Max.Sub(box.Min)
	if !(across.Len() < math.MaxFloat64) {
		return nowhere, endless
	}
	inside := box.Min.Add(across.Scale(half))
	middle := placed.Out(inside)
	return middle, placed.Out(box.Max).Sub(middle).Len()
}

// endless is the radius ballOf gives a box with no end, defined as the
// one radius that is under zero, which no ball has.
const endless = -one

// unbounded is whether a radius is the one ballOf gives a box with no
// end.
func unbounded(radius float64) bool { return radius < nothing }

// ballMeets is whether a sight's rays can meet the ball about a box that
// is placed in a subspace. A box with no end is met by every sight.
func ballMeets(sight Sight, box maths.Box, placed maths.Subspace) bool {
	centre, radius := ballOf(box, placed)
	return unbounded(radius) || sight.meets(centre, radius)
}

// half is defined as one over two: the middle of a box is half across it.
const half = 0.5

// where is the subspace a motion puts its thing in at a moment; a thing
// with no motion is where the world is.
func where(motion thing.Motion, at time.Duration) maths.Subspace {
	if motion == nil {
		return maths.Local()
	}
	return motion.At(at)
}
