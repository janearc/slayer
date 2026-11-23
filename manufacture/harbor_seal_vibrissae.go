package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The mystacial vibrissae grow from a pad in rows and columns, one strand
// to a follicle. A harbor seal's array has "~44 whiskers in seven rows",
// "a grid-like pattern that includes 7 rows and 9 columns", arclength
// falling from caudal to rostral (Graff and others 2024, Journal of
// Experimental Biology 227(22), Europe PMC PMC11634035), read here as
// one side's array.
//
// The longest "tended to be found in the lower rows
// and posterior columns", and phocid whiskers run "from about 1.5 cm to
// 10 cm" (Ginter Summarell and others 2015, PLoS ONE 10(7) e0127941).
// Each row's count, the places, lengths and angles are chosen.
const (
	sealWhiskerRows    = 7
	sealWhiskerColumns = 9
	// sealMystacials is the strands on one side: the row counts' sum.
	sealMystacials = 44
	// sealPadX is the X of the rearmost column, sealPadStep the gap between
	// columns, sealPadY the height of the lowest row and sealPadRise the
	// gap between rows, all in metres.
	sealPadX, sealPadStep = .745, .0065
	sealPadY, sealPadRise = .030, .008
	// sealWhiskerLong is the rear lower strand's length, shortened by
	// sealWhiskerLess per column and sealWhiskerRowLess per row.
	sealWhiskerLong    = .082
	sealWhiskerLess    = .0050
	sealWhiskerRowLess = .0040
	// sealWhiskerFan tilts each row's strands up or down.
	sealWhiskerFan = .09
	// sealWhiskerOut and sealWhiskerAhead weigh the skin's normal and the
	// forward axis in a strand's first direction.
	sealWhiskerOut, sealWhiskerAhead = .50, .80
	// sealWhiskerSag is how far a strand's tip hangs below its straight
	// line, as a share of its length, and sealWhiskerSweep how far it
	// sweeps forward.
	sealWhiskerSag, sealWhiskerSweep = .30, .18
	// sealWhiskerBead is the amplitude, in metres, of the wave a strand
	// makes along its length, and sealWhiskerBeads how many waves it makes.
	sealWhiskerBead  = .0012
	sealWhiskerBeads = 5
	// sealWhiskerRoot and sealWhiskerTip are a strand's radius at the skin
	// and at its end, in metres. sealWhiskerSteps is the segments in one.
	sealWhiskerRoot, sealWhiskerTip = .00042, .00008
	sealWhiskerSteps                = 7
	// sealWhiskerBury is how deep a strand's root lies in the skin.
	sealWhiskerBury = .002
)

// sealStrand is one vibrissa: its root on the skin, its first direction,
// its length, and the phase of its wave.
type sealStrand struct {
	anchor    sealAnchor
	direction maths.Vec
	length    float64
	phase     float64
}

// vibrissa adds one strand to the mold as its own thin tube. It leaves
// the skin along its direction, sags and sweeps forward and beads.
func (mold *sealMold) vibrissa(strand sealStrand) {
	root := strand.anchor.point.Sub(strand.anchor.normal.Scale(sealWhiskerBury))
	perpendicular := strand.direction.Cross(strand.anchor.up).Unit()
	droop := strand.anchor.up.Scale(-sealWhiskerSag).
		Add(strand.anchor.forward.Scale(sealWhiskerSweep)).Scale(strand.length)
	path := make([]maths.Vec, sealWhiskerSteps+1)
	radius := make([]float64, len(path))
	for step := range path {
		share := float64(step) / sealWhiskerSteps
		wave := math.Sin(2*math.Pi*sealWhiskerBeads*share+strand.phase) * sealWhiskerBead * share
		path[step] = root.Add(strand.direction.Scale(strand.length * share)).
			Add(droop.Scale(share * share)).Add(perpendicular.Scale(wave))
		radius[step] = sealWhiskerRoot + (sealWhiskerTip-sealWhiskerRoot)*share
	}
	mold.strand(path, radius)
}

// sealWhiskersInRow is how many strands each row holds, from the lowest,
// filling its columns from the rearmost forward. They sum to
// sealMystacials; chosen.
var sealWhiskersInRow = [sealWhiskerRows]int{7, 8, 7, 7, 6, 5, 4}

// whiskers sets the array of whiskers of one side on the pad, row by row,
// each strand at its anchor; side is the sign of z.
func (rig *SealRig) whiskers(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	for row, count := range sealWhiskersInRow {
		for column := range count {
			anchor, err := rig.anchor(pose, maths.V(sealPadX+sealPadStep*float64(column),
				sealPadY+sealPadRise*float64(row), side*sealProbe), maths.V(0, 0, -side))
			if err != nil {
				return err
			}
			tilt := (float64(row) - float64(sealWhiskerRows-1)/2) * sealWhiskerFan
			direction := anchor.normal.Scale(sealWhiskerOut).
				Add(anchor.forward.Scale(sealWhiskerAhead)).
				Add(anchor.up.Scale(tilt)).Unit()
			molds.hair.vibrissa(sealStrand{anchor, direction,
				sealWhiskerLong - sealWhiskerLess*float64(column) -
					sealWhiskerRowLess*float64(row),
				float64(row*sealWhiskerColumns+column) * sealWhiskerPhase})
		}
	}
	return nil
}

// sealWhiskerPhase offsets the wave of neighbouring strands so they do
// not bead in step, in radians.
const sealWhiskerPhase = 1.3

// The supraorbital vibrissae grow above each eye, a short row of stiff
// strands. Counts and numbers are chosen.
const (
	sealBrowCount = 4
	sealBrowX     = .700
	sealBrowStep  = .012
	sealBrowZ     = .05
	sealBrowLong  = .052
	sealBrowLess  = .004
	// sealBrowOut, sealBrowAhead and sealBrowAside weigh the skin's
	// normal, the forward axis and the outward side in a strand's first
	// direction.
	sealBrowOut, sealBrowAhead, sealBrowAside = .50, .70, .30
)

// brow sets the row of brow whiskers above the eye of one side; side is the
// sign of z.
func (rig *SealRig) brow(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	for index := range sealBrowCount {
		step := float64(index)
		anchor, err := rig.anchor(pose, maths.V(sealBrowX+sealBrowStep*step,
			sealProbe, side*sealBrowZ), maths.V(0, -1, 0))
		if err != nil {
			return err
		}
		direction := anchor.normal.Scale(sealBrowOut).
			Add(anchor.forward.Scale(sealBrowAhead)).
			Add(anchor.across.Scale(side * sealBrowAside)).Unit()
		molds.hair.vibrissa(sealStrand{anchor, direction,
			sealBrowLong - sealBrowLess*step, step * sealWhiskerPhase})
	}
	return nil
}

// The claws. Each digit tip carries one, a curved horn cone whose root
// lies inside the digit's end. Fore claws are larger. All chosen.
const (
	sealForeClawLong, sealForeClawRound = .017, .0042
	sealHindClawLong, sealHindClawRound = .009, .0028
	// sealClawBury is how far back along the claw, as a share of its
	// length, the root lies inside the digit.
	sealClawBury = .25
	// sealClawCurl is how far the tip curls down, as a share of length.
	sealClawCurl = .28
	// sealClawSteps and sealClawSides cut a claw along and round.
	sealClawSteps, sealClawSides = 6, 6
	// sealClawBlunt is the least radius a claw's tip keeps, in metres.
	sealClawBlunt = .00006
)

// claws adds a claw at each digit's end, found where the outermost ring's
// digit vertices lie in the posed skin and pointing along the digit.
func (rig *SealRig) claws(points []maths.Vec, molds *sealMolds) {
	for _, limb := range rig.limbs {
		long, round := sealForeClawLong, sealForeClawRound
		if limb.hind {
			long, round = sealHindClawLong, sealHindClawRound
		}
		for digit := range sealDigits {
			end := sealDigitMean(points, limb.outer, limb.digits, digit)
			back := sealDigitMean(points, limb.inner, limb.digits, digit)
			molds.horn.claw(end, end.Sub(back).Unit(), long, round)
		}
	}
}

// sealDigitMean is the mean of the posed points of a ring's vertices that
// end one digit.
func sealDigitMean(points []maths.Vec, ring, digits []int, digit int) maths.Vec {
	var sum maths.Vec
	var count float64
	for place, index := range ring {
		if digits[place] == digit {
			sum = sum.Add(points[index])
			count++
		}
	}
	return sum.Scale(1 / count)
}

// claw adds a curved cone from root along axis, curling toward the belly.
func (mold *sealMold) claw(root, axis maths.Vec, long, round float64) {
	path := make([]maths.Vec, sealClawSteps+1)
	radius := make([]float64, len(path))
	for step := range path {
		share := -sealClawBury + (1+sealClawBury)*float64(step)/sealClawSteps
		ahead := math.Max(share, 0)
		path[step] = root.Add(axis.Scale(long * share)).
			Add(maths.V(0, -sealClawCurl*long*ahead*ahead, 0))
		radius[step] = math.Max(round*math.Pow(1-ahead, .7), sealClawBlunt)
	}
	mold.tube(path, radius, sealClawSides)
}
