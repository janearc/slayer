package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The trunk and the neck are one lofted skin, so there is no seam between
// them: a closed tube from the tail stub through the hips, the deep chest
// and the neck to the skull's back. The wing, the legs and the skull are
// separate parts that meet it.

// The numbers the torso's surface uses.
const (
	// torsoSpacing is the distance between rings, in metres. It is fine
	// enough for the scale pattern below to read as surface.
	torsoSpacing = .01
	// torsoSides is how many points make a ring.
	torsoSides = 48
	// scaleRelief is the depth of the scale pattern, as a share of the
	// radius. Authored: about three millimetres on the neck.
	scaleRelief = .014
	// scaleCell is the pattern's size along the body, in metres, and
	// scaleAround how many cells ring the body.
	scaleCell   = .03
	scaleAround = 36
	// ridgeRelief and ridgeWidth shape the neck's dorsal ridge, and
	// ridgeBlend is how far down the neck it takes to reach full height.
	// Authored: the heavy neck's row of neural spines.
	ridgeRelief = .075
	ridgeWidth  = .22
	ridgeBlend  = .3
	// vertebraRelief is the swelling of each cervical, as a share of the
	// radius. Authored, after the elongate cervicals of azhdarchids.
	vertebraRelief = .03
	// keelRelief, keelWidth and keelReach shape the chest's keel, as a
	// share of the radius, an angle and a length in metres. Authored: the
	// flight muscles' attachment on the sternum.
	keelRelief = .07
	keelWidth  = .32
	keelReach  = .38
	// neckInto is how far the neck's spine runs on past the occiput into
	// the skull, in metres, so no cap shows at the back of the head.
	neckInto = .14
)

// torsoKeys are the control rings of the torso, trunk then neck, for a
// pose. The trunk's rings are fixed in the body's frame and lifted by the
// pose; the neck's follow its spine.
func (pose pterosaurPose) torsoKeys() []tubeKey {
	lift := maths.V(0, pose.lift, 0)
	trunk := []tubeKey{
		{maths.V(pterosaurTailEnd, -.02, 0), .035, .035, 2},
		{maths.V(-1.0, -.01, 0), .12, .12, 2},
		{maths.V(pterosaurHipAt, 0, 0), .24, .26, 2.2},
		{maths.V(-.30, .03, 0), .27, .28, 2.2},
		{maths.V(pterosaurChestAt, .08, 0), pterosaurTrunkHalfWidth,
			pterosaurTrunkHalfHeight, 2.2},
	}
	keys := make([]tubeKey, 0, len(trunk)+len(pose.neck)+1)
	for _, key := range trunk {
		key.centre = key.centre.Add(lift)
		keys = append(keys, key)
	}
	return append(keys, pose.neckKeys()...)
}

// neckKeys are the neck's control rings: its vertebral points, narrowing
// toward the head, then one more run on into the skull.
func (pose pterosaurPose) neckKeys() []tubeKey {
	last := len(pose.neck) - 1
	keys := make([]tubeKey, 0, len(pose.neck)+1)
	for i, point := range pose.neck {
		share := smoothShare(float64(i) / float64(last))
		base, head := pterosaurNeckBaseHalfWidth,
			pterosaurNeckHeadHalfWidth
		half := base + (head-base)*share
		keys = append(keys, tubeKey{point, half,
			half * pterosaurNeckHeightShare, 2.4})
	}
	heading := pose.neck[last].Sub(pose.neck[last-1]).Unit()
	end := keys[last]
	end.centre = end.centre.Add(heading.Scale(neckInto))
	return append(keys, end)
}

// torsoLength is the length of the torso's spine through its keys.
func torsoLength(keys []tubeKey) float64 {
	total := 0.0
	for i := 1; i < len(keys); i++ {
		total += keys[i].centre.Sub(keys[i-1].centre).Len()
	}
	return total
}

// torsoRelief is the multiplier of the torso's outline at a distance
// along the spine and an angle round it: the scale pattern, the neck's
// dorsal ridge and cervical swellings, and the chest's keel. neckStart is
// where the neck begins along the spine, in metres, and chestAt where the
// chest is.
func torsoRelief(along, angle, neckStart, chestAt float64) float64 {
	scale := scaleRelief * math.Cos(2*math.Pi*along/scaleCell) *
		math.Cos(scaleAround*angle)
	neck := smoothShare((along - neckStart) / ridgeBlend)
	top := (angle - math.Pi/2) / ridgeWidth
	ridge := ridgeRelief * math.Exp(-top*top) * neck
	step := pterosaurNeckLength / float64(pterosaurVertebrae)
	turn := 2 * math.Pi * (along - neckStart) / step
	vertebra := vertebraRelief * math.Cos(turn) * neck
	under := (angle - 1.5*math.Pi) / keelWidth
	reach := (along - chestAt) / keelReach
	keel := keelRelief * math.Exp(-under*under-reach*reach)
	return 1 + scale + ridge + vertebra + keel
}

// torsoSurface is the lofted skin of trunk and neck for a pose.
func (pose pterosaurPose) torsoSurface() tubeSurface {
	keys := pose.torsoKeys()
	trunkKeys := len(keys) - len(pose.neck) - 1
	neckStart := torsoLength(keys[:trunkKeys+1])
	chestAt := torsoLength(keys[:trunkKeys])
	relief := func(along, angle float64) float64 {
		return torsoRelief(along, angle, neckStart, chestAt)
	}
	up := maths.V(0, 1, 0)
	return tubeLoft(keys, torsoSpacing, torsoSides, up, relief)
}
