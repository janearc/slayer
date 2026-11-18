package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// The Alfa's bow planes and its cruciform stern: an upright fin above
// and below with a rudder behind each, a stabiliser to either side with
// a stern plane behind each, and a thruster pod on each stabiliser's
// tip. Each is a foil lofted from a root inside the hull to its tip.

// The fins, rudders, stern planes and bow planes are NACA four-digit
// symmetrical sections, a choice: half the section's thickness at a
// share x of its chord, over its whole thickness, is five times
// 0.2969 root x - 0.1260 x - 0.3516 x^2 + 0.2843 x^3 - 0.1015 x^4.
const (
	alfaNacaScale  = 5      // defined: NACA Report 460, Jacobs, Ward and Pinkerton, 1933; read, not shelved
	alfaNacaRoot   = 0.2969 // defined: the same, the term in the root of x
	alfaNacaLinear = 0.1260 // defined: the same, the term in x
	alfaNacaSquare = 0.3516 // defined: the same, the term in x squared
	alfaNacaCube   = 0.2843 // defined: the same, the term in x cubed
	alfaNacaFourth = 0.1015 // defined: the same, the term in x to the fourth
)

// alfaNaca is a NACA four-digit section's half thickness a share of the
// way along its chord, over its whole thickness: nought at its nose,
// a half at three tenths, and a hundredth at its trailing edge.
func alfaNaca(share float64) float64 {
	tail := share * (alfaNacaLinear + share*(alfaNacaSquare-
		share*(alfaNacaCube-alfaNacaFourth*share)))
	return alfaNacaScale * (alfaNacaRoot*math.Sqrt(share) - tail)
}

const (
	alfaFlapThick = 0.6  // chosen: a rudder or stern plane is this share as thick as its fin
	alfaFlapGap   = 0.05 // chosen: metres between a fin's trailing edge and its rudder or plane
	alfaPodCap    = 2    // chosen: a pod's rounded ends are this many of its radii long
	alfaQuarters  = 4    // defined: the four fins stand a quarter turn apart
	alfaPortSide  = 1    // defined: the quarter turn from the top that points to port
	alfaStarboard = 3    // defined: the quarter turn from the top that points to starboard
	alfaPair      = 2    // defined: the fins alternate, upright then level, round the axis
)

// alfaWing is a hydrofoil lofted out from its root to its tip, its
// section NACA's of its thickness, in a space whose y runs out from the
// hull's axis along its span, whose z runs aft from its root's leading
// edge, and whose x is across its thickness.
func alfaWing(root, tip, rootChord, tipChord, tipLead,
	thick float64,
) alfaLoft {
	levels := []struct{ out, lead, chord float64 }{{root, spaceZero,
		rootChord}, {tip, tipLead, tipChord}}
	half := func(share float64) float64 { return thick * alfaNaca(share) }
	var rings [][]maths.Vec
	for _, level := range levels {
		var ring []maths.Vec
		for _, corner := range alfaLoop(level.chord, half) {
			ring = append(ring, maths.V(corner[spaceOne], level.out,
				level.lead+corner[spaceZero]))
		}
		rings = append(rings, ring)
	}
	return alfaLoft{rings: rings}
}

// wingSpace is the space a wing is made in, its root's leading edge so
// far aft of the bow on the hull's axis, turned a number of quarter
// turns about the axis from the top toward port.
func (size alfaSizes) wingSpace(quarter int, aft float64) maths.Subspace {
	turn := fullTurn * float64(quarter) / alfaQuarters
	return maths.NewSubspace(size.at(spaceZero, size.radius(), aft),
		maths.Rotation(maths.ZAxis, turn), spaceOne)
}

// rootIn is how far out from the axis a wing's root can stand and keep
// its whole thickness inside a hull of a radius, less the sink.
func rootIn(radius, thick float64) float64 {
	half := thick * spaceHalf
	return math.Sqrt((radius-half)*(radius+half)) - alfaSink
}

// narrowest is the hull's radius where it is narrowest between two
// distances aft; the hull's radius only rises on the nose and only
// falls on the tail, so that is at one end or the other.
func (size alfaSizes) narrowest(from, to float64) float64 {
	return math.Min(size.hullRadius(from), size.hullRadius(to))
}

// planeLead is how far aft of the bow the bow planes' leading edge is.
func (size alfaSizes) planeLead() float64 {
	return size.Planes.Aft - size.Planes.Chord*spaceHalf
}

// planeRoot and planeTip are how far out from the axis the bow planes'
// root and tip stand.
func (size alfaSizes) planeRoot() float64 {
	planes := size.Planes
	return rootIn(size.narrowest(size.planeLead(), size.planeLead()+
		planes.Chord), planes.Thick)
}

// planeTip is how far out from the axis the bow planes' tip stands: the
// span out from the hull at their middle.
func (size alfaSizes) planeTip() float64 {
	return size.hullRadius(size.Planes.Aft) + size.Planes.Span
}

// planesFit refuses bow planes off the hull's ends, or too thick to
// keep their root inside it.
func (size alfaSizes) planesFit() error {
	planes := size.Planes
	if size.planeLead() <= spaceZero ||
		size.planeLead()+planes.Chord >= size.hullEnd() ||
		!(size.planeRoot() > spaceZero) {
		return fmt.Errorf("manufacture: the alfa's bow planes, %v long "+
			"and %v thick round %v aft of the bow, do not fit on its hull",
			planes.Chord, planes.Thick, planes.Aft)
	}
	return nil
}

// planes sets the bow planes, one out to either side.
func (size alfaSizes) planes(assembly *instrumentAssembly) {
	planes := size.Planes
	for _, quarter := range []int{alfaPortSide, alfaStarboard} {
		assembly.add(alfaWing(size.planeRoot(), size.planeTip(),
			planes.Chord, planes.Chord, spaceZero, planes.Thick),
			size.wingSpace(quarter, size.planeLead()), alfaPlane)
	}
}

// trailing is how far aft of the bow the fins' trailing edge stands.
func (size alfaSizes) trailing() float64 {
	return size.Stern.Aft + size.Stern.Chord
}

// flapLead is how far aft of the bow the rudders' and stern planes'
// leading edge stands, a gap behind the fins.
func (size alfaSizes) flapLead() float64 {
	return size.trailing() + alfaFlapGap
}

// reaches is how far out from the axis each fin reaches, by quarter
// turns from the top toward port: the upper fin, the port stabiliser,
// the lower fin and the starboard stabiliser.
func (size alfaSizes) reaches() []float64 {
	stern := size.Stern
	return []float64{stern.Up, stern.Across, stern.Down, stern.Across}
}

// flapAt is the length of the rudder or stern plane behind a fin, its
// look, and how far out from the axis its root stands.
func (size alfaSizes) flapAt(quarter int) (float64, slot, float64) {
	stern := size.Stern
	long, of := stern.Rudder, alfaRudder
	if quarter%alfaPair == alfaPortSide%alfaPair {
		long, of = stern.Flap, alfaFlap
	}
	end := size.flapLead() + long
	return long, of, rootIn(size.hullRadius(end), stern.Thick*alfaFlapThick)
}

// finRoot is how far out from the axis the fins' root stands, inside
// the hull at their trailing edge, where it is narrowest.
func (size alfaSizes) finRoot() float64 {
	return rootIn(size.hullRadius(size.trailing()), size.Stern.Thick)
}

// sternFits refuses fins whose tip is longer than their root, whose
// rudders or stern planes reach the boss, whose roots cannot stay inside
// the hull, or whose tips stay inside it; and pods that cannot hold a
// stabiliser's tip.
func (size alfaSizes) sternFits() error {
	stern, pods := size.Stern, size.Pods
	refused := fmt.Errorf("manufacture: the alfa's stern fins, %v long at "+
		"the root and %v at the tip from %v aft of the bow, do not fit its "+
		"tail", stern.Chord, stern.Tip, stern.Aft)
	tipLead := size.trailing() - stern.Tip
	if stern.Tip > stern.Chord || !(size.finRoot() > spaceZero) ||
		size.flapLead()+math.Max(stern.Rudder, stern.Flap) >= size.hullEnd() {
		return refused
	}
	for quarter, reach := range size.reaches() {
		_, _, root := size.flapAt(quarter)
		if !(root > spaceZero) || reach <= size.hullRadius(tipLead) {
			return refused
		}
	}
	if pods.Radius*alfaPair <= stern.Thick || pods.Long <= stern.Tip ||
		pods.Long <= pods.Radius*alfaPodCap*alfaPair {
		return fmt.Errorf("manufacture: the alfa's thruster pods, %v long "+
			"and %v round, cannot hold the stabilisers' tips", pods.Long,
			pods.Radius)
	}
	return nil
}

// withinBeam refuses stabilisers and pods wider than the measured beam,
// and a lower fin below the keel: the beam and the keel are the hull's,
// and nothing fixed stands outside them. The bow planes, drawn out, do.
func (size alfaSizes) withinBeam() error {
	stern := size.Stern
	if stern.Across+size.Pods.Radius > size.radius() ||
		stern.Down > size.radius() {
		return fmt.Errorf("manufacture: the alfa's stabilisers and pods "+
			"reach %v to a side and its lower fin %v down, past its beam "+
			"or its keel, %v from the axis", stern.Across+size.Pods.Radius,
			stern.Down, size.radius())
	}
	return nil
}

// stern sets the four fins, a rudder or a stern plane behind each, and
// a thruster pod on each stabiliser's tip.
func (size alfaSizes) stern(assembly *instrumentAssembly) {
	stern := size.Stern
	thin := stern.Thick * alfaFlapThick
	for quarter, reach := range size.reaches() {
		assembly.add(alfaWing(size.finRoot(), reach, stern.Chord, stern.Tip,
			stern.Chord-stern.Tip, stern.Thick),
			size.wingSpace(quarter, stern.Aft), alfaFin)
		long, of, root := size.flapAt(quarter)
		assembly.add(alfaWing(root, reach, long, long, spaceZero, thin),
			size.wingSpace(quarter, size.flapLead()), of)
	}
	front := size.podMiddle() - size.Pods.Long*spaceHalf
	for _, side := range []float64{-spaceOne, spaceOne} {
		assembly.add(alfaTurned{turnedProfile{points: size.podProfile()}},
			size.axialAt(side*stern.Across, size.radius(), front), alfaPod)
	}
}

// podMiddle is how far aft of the bow a pod's middle stands: the middle
// of a stabiliser's tip.
func (size alfaSizes) podMiddle() float64 {
	return size.trailing() - size.Stern.Tip*spaceHalf
}

// podProfile is a thruster pod's turned profile: a rounded nose, a
// straight middle and a rounded tail, from its front to its back.
func (size alfaSizes) podProfile() [][coordinatePair]float64 {
	pods := size.Pods
	end := pods.Radius * alfaPodCap
	points := [][coordinatePair]float64{{spaceZero, spaceZero}}
	for step := spaceOne; step <= alfaCapSteps; step++ {
		turn := math.Pi * spaceHalf * float64(step) / alfaCapSteps
		points = append(points, [coordinatePair]float64{
			pods.Radius * math.Sin(turn), end * (spaceOne - math.Cos(turn))})
	}
	for step := range alfaCapSteps {
		turn := math.Pi * spaceHalf * float64(step) / alfaCapSteps
		points = append(points, [coordinatePair]float64{
			pods.Radius * math.Cos(turn), pods.Long - end + end*math.Sin(turn)})
	}
	return append(points, [coordinatePair]float64{spaceZero, pods.Long})
}
