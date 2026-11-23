package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// The flippers' flesh is laid on their bones: the muscle of the upper arm
// and thigh inside the body wall, the forearm and shank, then on every
// bone of the palm or sole and of each digit a flat mass as wide as the
// space between rays, so neighbouring rays meet in one web, and a thin
// rim of web beyond the last bone. Sizes chosen.
const (
	// walrusWebThickness is the half thickness of the flipper over the
	// palm or sole; it thins to walrusWebTip at the end of each digit.
	walrusWebThickness = .028
	walrusWebTip       = .008
	// walrusWebShare is the half width of a ray's flesh as a share of the
	// spacing between rays: over one, so that neighbouring rays run into
	// one smooth paddle with no groove between them.
	walrusWebShare = 1.15
	// walrusWebRim is how far the web runs past the last bone of a digit.
	walrusWebRim = .05
)

// walrusLimbMasses is the flesh of all four flippers.
func walrusLimbMasses(body *walrusBones) []walrusMass {
	var masses []walrusMass
	for _, side := range []float64{-1, 1} {
		name := walrusSideName(side)
		masses = append(masses, walrusArmFlesh(body, name)...)
		masses = append(masses, walrusRays(body, "metacarpal", "fore", name,
			walrusForeDigits, walrusForeSpacing)...)
		masses = append(masses, walrusLegFlesh(body, name)...)
		masses = append(masses, walrusRays(body, "metatarsal", "hind", name,
			walrusHindDigits, walrusHindSpacing)...)
	}
	return masses
}

// walrusBoneLine is the two ends of the straight bone of that name.
func walrusBoneLine(body *walrusBones, name string) (maths.Vec, maths.Vec) {
	return body.rod(name)
}

// walrusFleshOn is a flat mass along the bone from a to b, its half
// thickness thick and half width wide, reaching past each end by over.
func walrusFleshOn(a, b maths.Vec, thick, wide, over, strength float64) walrusMass {
	length := b.Sub(a).Len()
	return walrusPaddle(a.Add(b).Scale(.5), b.Sub(a),
		maths.V(length/2+over, thick, wide), strength)
}

// walrusArmFlesh is the muscle round the forearm, and the pad over the
// wrist. The upper arm lies inside the body's outline and adds nothing to
// it.
func walrusArmFlesh(body *walrusBones, side string) []walrusMass {
	_, elbow := walrusBoneLine(body, "humerus "+side)
	_, wrist := walrusBoneLine(body, "radius "+side)
	return []walrusMass{
		walrusFleshOn(elbow, wrist, .10, .12, .03, 1),
		walrusFleshOn(wrist, wrist.Add(wrist.Sub(elbow).Unit().Scale(.10)),
			.06, .12, .02, 1),
	}
}

// walrusLegFlesh is the muscle round the shank, and the pad over the ankle
// and heel. The thigh lies inside the body's outline and adds nothing to
// it.
func walrusLegFlesh(body *walrusBones, side string) []walrusMass {
	_, knee := walrusBoneLine(body, "femur "+side)
	_, ankle := walrusBoneLine(body, "tibia "+side)
	heel, toe := walrusBoneLine(body, "calcaneus "+side)
	return []walrusMass{
		walrusFleshOn(knee, ankle, .10, .13, .03, 1),
		walrusFleshOn(heel, toe, .045, .09, .03, 1),
	}
}

// walrusRays is the flesh on every ray of one flipper: on its palm or
// sole bone and on each phalanx, thinning toward the digit's end, and the
// web's rim beyond the last bone.
func walrusRays(body *walrusBones, palm, kind, side string,
	digits [walrusDigitCount][]float64, spacing float64) []walrusMass {
	var masses []walrusMass
	wide := spacing * walrusWebShare
	for ray := 1; ray <= walrusDigitCount; ray++ {
		names := walrusRayNames(palm, kind, side, ray, digits)
		var tip, along maths.Vec
		for i, name := range names {
			share := float64(i) / float64(len(names))
			thick := walrusWebThickness + (walrusWebTip-walrusWebThickness)*share
			from, to := walrusBoneLine(body, name)
			masses = append(masses, walrusFleshOn(from, to, thick, wide, .01, 1))
			tip, along = to, to.Sub(from).Unit()
		}
		rim := tip.Add(along.Scale(walrusWebRim))
		masses = append(masses, walrusFleshOn(tip, rim, walrusWebTip, wide, 0, 1))
	}
	return masses
}

// walrusRayNames is the names of the bones of one ray, from its palm or
// sole bone to the last phalanx.
func walrusRayNames(palm, kind, side string, ray int,
	digits [walrusDigitCount][]float64) []string {
	names := []string{fmt.Sprintf("%s %d %s", palm, ray, side)}
	for bone := 1; bone <= len(digits[ray-1]); bone++ {
		names = append(names,
			fmt.Sprintf("%s phalanx %d.%d %s", kind, ray, bone, side))
	}
	return names
}
