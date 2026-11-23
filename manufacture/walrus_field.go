package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// The face is cut into the outline: a shallow hollow round each eye, a
// pit for each nostril, and the groove between the two muzzle pads. Each
// is placed on the outline's own surface, so a head of another size
// carries them with it. Widths and depths are picked; a depth is in the
// density's units, and walrusCutDepth says how far in metres it moves
// the hide.
const (
	// walrusEyeStation, walrusEyeDrop and walrusEyeOut place the eye in the
	// skull's frame: its share of the skull's length forward of the
	// condyle, its drop below the axis (above it, so less than nought) and
	// about how far out from the middle it lies, in metres. Picked: small
	// eyes set high and wide, behind the pads.
	walrusEyeStation = .47
	walrusEyeDrop    = -.08
	walrusEyeOut     = .17
	// walrusNostrilStation, walrusNostrilDrop and walrusNostrilOut place a
	// nostril on top of the muzzle, behind the pads' face. Picked.
	walrusNostrilStation = 1.04
	walrusNostrilDrop    = -.06
	walrusNostrilOut     = .04
	// walrusEyeWidth and walrusEyeDepth are the hollow round an eye.
	walrusEyeWidth = .05
	walrusEyeDepth = .10
	// walrusNostrilWidth and walrusNostrilDepth are a nostril's pit.
	walrusNostrilWidth = .020
	walrusNostrilDepth = .45
)

// walrusEyeSocket is about where each eye lies, in the skull's frame:
// high on the side of the head. The eye itself is set on the hide near
// here, by walrusEyeOn and walrusEyeAt.
func walrusEyeSocket(side float64) maths.Vec {
	return walrusFrame.at(walrusEyeStation, side*walrusEyeOut, walrusEyeDrop)
}

// walrusNostril is about where each nostril opens, in the skull's frame:
// on the top of the muzzle, over the front of the premaxilla.
func walrusNostril(side float64) maths.Vec {
	return walrusFrame.at(walrusNostrilStation, side*walrusNostrilOut,
		walrusNostrilDrop)
}

// walrusEyeOn is the eye's place on the outline's own surface: at the
// socket's place along the body and its height, as far out as the
// outline's flank lies there. It refuses a socket the outline has no
// flank beside.
func walrusEyeOn(outline *walrusOutline, side float64) (maths.Vec, error) {
	socket := walrusEyeSocket(side)
	out, ok := outline.flank(socket.X, socket.Y)
	if !ok {
		return maths.Vec{}, fmt.Errorf("%w: the outline has no flank beside "+
			"the eye on side %g", errWalrusLandmark, side)
	}
	return maths.V(socket.X, socket.Y, side*out), nil
}

// walrusNostrilOn is a nostril's place on the outline's own surface: at
// the pit's place along the body and across it, on the outline's back. It
// refuses a pit the outline has no back over.
func walrusNostrilOn(outline *walrusOutline, side float64) (maths.Vec, error) {
	pit := walrusNostril(side)
	top, ok := outline.back(pit.X, pit.Z)
	if !ok {
		return maths.Vec{}, fmt.Errorf("%w: the outline has no back over "+
			"the nostril on side %g", errWalrusLandmark, side)
	}
	return maths.V(pit.X, top, pit.Z), nil
}

// walrusPit is a round dip of the given width and depth at centre.
func walrusPit(p, centre maths.Vec, width, depth float64) float64 {
	far := p.Sub(centre).Len() / width
	return depth * math.Exp(-far*far)
}

// walrusCleft cuts the shallow groove down the middle of the muzzle that
// divides the two upper lip pads, in the skull's frame. Picked.
func walrusCleft(p maths.Vec) float64 {
	const (
		start, width, depth = .85, .016, .14
		high, low           = -.12, .21
	)
	t, side, down := walrusFrame.local(p)
	if t < start || down < high || down > low {
		return 0
	}
	off := side / width
	return depth * math.Exp(-off*off)
}

// walrusFaceOf is every pit and dip of the head on the outline given: the
// hollows round the eyes, the nostrils and the cleft between the pads.
// Nothing behind the back of the skull is cut. It refuses an outline that
// one of them does not lie on.
func walrusFaceOf(outline *walrusOutline) (func(maths.Vec) float64, error) {
	const behind = -.25
	var eyes, nostrils [2]maths.Vec
	for i, side := range []float64{-1, 1} {
		eye, err := walrusEyeOn(outline, side)
		if err != nil {
			return nil, err
		}
		nostril, err := walrusNostrilOn(outline, side)
		if err != nil {
			return nil, err
		}
		eyes[i], nostrils[i] = eye, nostril
	}
	return func(p maths.Vec) float64 {
		if t, _, _ := walrusFrame.local(p); t < behind {
			return 0
		}
		total := walrusCleft(p)
		for i := range eyes {
			total += walrusPit(p, eyes[i], walrusEyeWidth, walrusEyeDepth)
			total += walrusPit(p, nostrils[i], walrusNostrilWidth,
				walrusNostrilDepth)
		}
		return total
	}, nil
}

// walrusDensity is the summed density of every mass at p.
func walrusDensity(masses []walrusMass, p maths.Vec) float64 {
	density := 0.
	for _, mass := range masses {
		density += mass.density(p)
	}
	return density
}

// walrusOutlineReach is how far, in metres, the outline's density falls
// by a factor of e outside it: the width over which a flipper blends into
// the body. Chosen.
const walrusOutlineReach = .04

// walrusBodyDensity is the density of the body at p: the outline's, which
// is walrusThreshold on the outline itself and falls off outside it, and
// the masses' added to it, so that the masses join the outline smoothly
// and leave it alone where they are not.
func walrusBodyDensity(outline *walrusOutline, masses []walrusMass,
	p maths.Vec) float64 {
	gap := math.Max(outline.distance(p), -walrusOutlineReach*walrusOutlineDeep)
	return walrusThreshold*math.Exp(-gap/walrusOutlineReach) +
		walrusDensity(masses, p)
}

// walrusOutlineDeep is how many reaches inside the outline its density
// stops growing, so that it stays a finite number. Chosen.
const walrusOutlineDeep = 20

const (
	// walrusFaint is the least density the height is taken of, so that
	// far from the body it stays a finite number. Chosen.
	walrusFaint = 1e-12
	// walrusFloorRim is the width over which the edge is rounded where
	// the belly meets the ground, in metres. Chosen.
	walrusFloorRim = .02
)

// walrusHeight turns a density into the field the hide is meshed from:
// nought where the density is walrusThreshold, negative inside. Where
// only the outline gives the density it is the distance outside the
// outline, in metres, so it runs evenly through the hide and the mesh's
// straight-line placing of each point leaves no ripple at the grid's
// spacing, as the density itself, which is curved there, would.
func walrusHeight(density float64) float64 {
	return -walrusOutlineReach *
		math.Log(math.Max(density, walrusFaint)/walrusThreshold)
}

// walrusCutDepth is how far, in metres, a cut of the given depth moves
// the hide in: the height at which the density is walrusThreshold and
// the cut's depth together.
func walrusCutDepth(cut float64) float64 {
	return walrusOutlineReach * math.Log(1+cut/walrusThreshold)
}

// walrusOnGround is the field of a body lying on the ground: hide, a
// height, with everything under the ground cut away and the edge
// between them rounded.
func walrusOnGround(hide float64, p maths.Vec) float64 {
	return walrusSoftMax(hide, walrusFloor-p.Y, walrusFloorRim)
}

// walrusVolumeField is the field of the outline and the masses, flattened
// on the ground: no face cut. It shows the form before any detail.
func walrusVolumeField(outline *walrusOutline,
	masses []walrusMass) func(maths.Vec) float64 {
	return func(p maths.Vec) float64 {
		return walrusOnGround(
			walrusHeight(walrusBodyDensity(outline, masses, p)), p)
	}
}

// walrusSoftMax is the larger of two numbers with the corner between them
// rounded over round.
func walrusSoftMax(a, b, round float64) float64 {
	return .5 * (a + b + math.Sqrt((a-b)*(a-b)+round*round))
}

// walrusField is the field whose zero is the hide: negative inside. It is
// the outline with the masses joined to it, the face's pits cut and the
// neck and shoulders creased; on the ground, it flattens the underside
// where the body's weight rests.
func walrusField(outline *walrusOutline, masses []walrusMass,
	ground bool) (func(maths.Vec) float64, error) {
	face, err := walrusFaceOf(outline)
	if err != nil {
		return nil, err
	}
	creases := walrusCreasesOf(outline)
	return func(p maths.Vec) float64 {
		hide := walrusHeight(walrusBodyDensity(outline, masses, p)) +
			walrusCutDepth(face(p)+creases(p))
		if !ground {
			return hide
		}
		return walrusOnGround(hide, p)
	}, nil
}
