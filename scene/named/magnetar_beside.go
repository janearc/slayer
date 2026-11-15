package named

import (
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Zappy beside the real one, both at their true sizes: zappy's star of
// 9,000 km below the real star, its beams off since it points at
// something else, its glass shells left out since from where the eye
// must stand they surround it and hide the dot.
//
// An eye that sees zappy fill the frame sees the real one as the dot
// it is, above it; it must come in by the ratio of their radii to see
// the real one as large.

// how far below the real star zappy's middle is, in zappy's radii, and
// how far the eye stands from zappy's middle, in zappy's radii: zappy's
// star fills four fifths of the frame's width and the real one sits
// above its limb. Picked by the author of this scene, 2026-11-12.
const (
	zappyBelow    = 1.35
	zappyStandOff = 3.5
)

// placeBeside puts zappy in the world below the real star when asked,
// its star only.
func placeBeside(world *thing.World, built *Built, held values) error {
	if !held.toggle("zappy") {
		return nil
	}
	built.clock = new(time.Duration)
	made, err := zappyDefined.Lantern(built.clock)
	if err != nil {
		return err
	}
	made.Radius = zappyKm()
	made.At = zappyMiddle()
	made.Beams = nil
	made.Orbits = nothing
	_, err = made.Place(world)
	return err
}

// zappyMiddle is where zappy's middle is, in km: below the real star.
func zappyMiddle() maths.Vec {
	return maths.V(0, -zappyBelow*zappyKm(), 0)
}

// besideEye is the eye that sees zappy fill the frame, looking at its
// middle, the real star above; or, at a distance asked, in km, the eye
// that far from the real star, looking at it from the same side.
func besideEye(asked float64) maths.Subspace {
	if asked > nothing {
		from := maths.V(0, 0, asked)
		return maths.Subspace{Origin: from,
			Orientation: maths.LookAtUp(from, maths.V(0, 0, 0), maths.Up)}
	}
	middle := zappyMiddle()
	from := middle.Add(maths.V(0, 0, zappyStandOff*zappyKm()))
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, middle, maths.Up)}
}
