package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// How the lofted skin is held by the bones. A point of the trunk follows
// the skull and the vertebrae near it along the body, each by a share
// that falls from one at the bone's middle to nothing a reach away; a
// flipper follows one bone whole. The numbers are picked: a reach wide
// enough that a bent back shows no crease.
const (
	// sealHoldReach is a vertebra's hold in its own lengths, each side.
	sealHoldReach = 1.5
	// sealSkullHold is how far ahead of the condyles the skull's hold is
	// whole, and sealSkullRun the length it is given for its reach.
	sealSkullHold, sealSkullRun = .03, .05
)

// sealSpan is one bone of the axis as the skin is held by it: its place
// in the skeleton, the X of its hold's middle, and its length along X.
type sealSpan struct {
	bone        int
	middle, run float64
}

// sealAxis is the skull and the column's vertebrae as spans, the skull
// first and the tail's tip last.
func sealAxis(made *sealBones) []sealSpan {
	var spans []sealSpan
	at, _ := made.bones.Find("skull")
	skull := made.bones.Bone(at)
	spans = append(spans, sealSpan{bone: at,
		middle: skull.Head().X + sealSkullHold, run: sealSkullRun})
	neck := sealNamesWith(made.names(), "neck")
	for index := len(neck) - 1; index >= 0; index-- {
		spans = append(spans, made.span(neck[index]))
	}
	for _, name := range sealNamesWith(made.names(), "thoracic",
		"lumbar", "sacral", "caudal") {
		spans = append(spans, made.span(name))
	}
	return spans
}

// span is a named vertebra as a span of the axis.
func (made *sealBones) span(name string) sealSpan {
	at, _ := made.bones.Find(name)
	bone := made.bones.Bone(at)
	head, tail := bone.Head().X, bone.Tail().X
	return sealSpan{bone: at, middle: (head + tail) / 2,
		run: math.Abs(head - tail)}
}

// sealAxialHold is the bones of the axis that carry a point at x, each
// with its share. Ahead of the skull's middle the skull carries it
// alone; behind the last vertebra's middle, the last vertebra.
func sealAxialHold(spans []sealSpan, x float64) []skeleton.Influence {
	first, last := spans[0], spans[len(spans)-1]
	if x >= first.middle {
		return []skeleton.Influence{skeleton.NewInfluence(first.bone, 1)}
	}
	if x <= last.middle {
		return []skeleton.Influence{skeleton.NewInfluence(last.bone, 1)}
	}
	var hold []skeleton.Influence
	for _, span := range spans {
		share := 1 - math.Abs(x-span.middle)/(sealHoldReach*span.run)
		if share > 0 {
			hold = append(hold, skeleton.NewInfluence(span.bone, share))
		}
	}
	return hold
}

// sealHeld is a surface with the bones that carry each of its points.
type sealHeld struct {
	surface sealSurface
	holds   [][]skeleton.Influence
}

// sealHeldTrunk is an outline's lofted trunk held along the axis.
func sealHeldTrunk(made *sealBones, outline sealOutline) sealHeld {
	spans := sealAxis(made)
	surface := outline.surface()
	holds := make([][]skeleton.Influence, len(surface.points))
	for index, point := range surface.points {
		holds[index] = sealAxialHold(spans, point.X)
	}
	return sealHeld{surface: surface, holds: holds}
}

// heldPaddle is a flipper's cushion over the digits whose bones' names
// bones gives, carried whole by the bone named carrier.
func (made *sealBones) heldPaddle(bones func(digit int) []string,
	shape sealPaddleShape, carrier string,
) (sealHeld, error) {
	lines, err := made.digitLines(bones, shape)
	if err != nil {
		return sealHeld{}, err
	}
	at, found := made.bones.Find(carrier)
	if !found {
		return sealHeld{}, errSealCarrier
	}
	surface := sealPaddleSurface(lines, shape)
	hold := []skeleton.Influence{skeleton.NewInfluence(at, 1)}
	holds := make([][]skeleton.Influence, len(surface.points))
	for index := range holds {
		holds[index] = hold
	}
	return sealHeld{surface: surface, holds: holds}, nil
}

// sealHeldFlippers is the four flippers, each carried by its limb: a
// fore flipper by its middle metacarpal, a hind one by its ankle bone.
func sealHeldFlippers(made *sealBones) ([]sealHeld, error) {
	var flippers []sealHeld
	for _, side := range [...]float64{1, -1} {
		fore := "fore-" + sealSideName(side) + "-"
		hind := "hind-" + sealSideName(side) + "-"
		for _, limb := range [...]struct {
			bones   func(int) []string
			shape   sealPaddleShape
			carrier string
		}{
			{func(digit int) []string {
				return sealForeDigitBones(fore, digit)
			}, sealForePaddle, fore + "metacarpal3"},
			{func(digit int) []string {
				return sealHindDigitBones(hind, digit)
			}, sealHindPaddle, hind + "astragalus"},
		} {
			held, err := made.heldPaddle(limb.bones, limb.shape,
				limb.carrier)
			if err != nil {
				return nil, err
			}
			flippers = append(flippers, held)
		}
	}
	return flippers, nil
}

// How a seal lying on the ground settles. The ground is taken a sink
// above the lowest point of the posed animal; whatever lies below a
// band over it is pressed up toward it, most where it is lowest, so the
// belly flattens where it bears, and spreads sideways as it is pressed.
// All picked.
const (
	sealSlumpSink   = .045 // metres the lowest point lies under the ground
	sealSlumpBand   = .11  // metres over the ground that are pressed
	sealSlumpSpread = 3.   // sideways spread per metre a point is lifted
)

// sealStance is an animal's pose as the skin needs it: each bone's
// motion, the turn of the whole animal, for one lying down the height
// of the ground it settles on, and the animal's size over the rest
// space's, by which it is scaled last.
type sealStance struct {
	frames []skeleton.Frame
	lean   skeleton.Frame
	rests  bool
	ground float64
	scale  float64
}

// carried is a rest point moved by its bones and the whole animal's
// turn, before it settles.
func (stance sealStance) carried(rest maths.Vec,
	hold []skeleton.Influence,
) maths.Vec {
	return stance.lean.Apply(skeleton.Carry(rest, stance.frames, hold))
}

// sealPressed is the height over the ground a point at height over it
// is pressed to: itself above the band, and below it a height that
// falls ever more slowly toward the ground.
func sealPressed(height float64) float64 {
	if height >= sealSlumpBand {
		return height
	}
	under := (sealSlumpBand - height) / sealSlumpBand
	return sealSlumpBand * (1 - under/(1+under))
}

// settled is a carried point as it lies: unchanged for an animal that
// does not rest; else pressed onto the ground, which becomes height
// nought.
func (stance sealStance) settled(point maths.Vec) maths.Vec {
	if !stance.rests {
		return point
	}
	height := point.Y - stance.ground
	pressed := sealPressed(height)
	return maths.V(point.X, pressed-sealPressed(-sealSlumpSink),
		point.Z*(1+sealSlumpSpread*(pressed-height)))
}

// placed is a rest point where the skin's point held so ends up, at the
// animal's own size.
func (stance sealStance) placed(rest maths.Vec,
	hold []skeleton.Influence,
) maths.Vec {
	return stance.settled(stance.carried(rest, hold)).Scale(stance.scale)
}

// onGround is the stance with its ground set from the lowest carried
// point of the surfaces.
func (stance sealStance) onGround(parts []sealHeld) sealStance {
	lowest := math.Inf(1)
	for _, part := range parts {
		for index, rest := range part.surface.points {
			lowest = math.Min(lowest,
				stance.carried(rest, part.holds[index]).Y)
		}
	}
	stance.ground = lowest + sealSlumpSink
	return stance
}
