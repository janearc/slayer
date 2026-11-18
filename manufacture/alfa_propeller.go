package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// The Alfa's one fixed-pitch propeller (russianships line 289): its
// blades on the boss, each lofted from its root to its tip through
// foil sections wrapped round the axis on a helix of the file's pitch,
// its chord swelling between root and tip and its tip skewed round.

const (
	alfaBladeRings  = 7    // chosen: sections from a blade's root to its tip
	alfaBladeEnds   = 0.7  // chosen: a blade's chord at its root and tip, as a share of the file's
	alfaBladeSwell  = 0.5  // chosen: how much wider a blade's middle is than its ends, as a share of the file's chord
	alfaBladeThin   = 0.75 // chosen: the share of its root's thickness a blade has lost by its tip
	alfaBladeTail   = 0.05 // chosen: the share of its thickness a blade keeps at its trailing edge
	alfaLeastBlades = 1    // defined: a propeller has a blade or it is none
)

// bladeRoot and bladeTip are how far out from the axis a blade's root,
// sunk into the boss, and its tip stand.
func (size alfaSizes) bladeRoot() float64 {
	return size.Hull.Stern - alfaSink
}

// bladeTip is half the propeller's diameter.
func (size alfaSizes) bladeTip() float64 {
	return size.Propeller.Diameter * spaceHalf
}

// screwAft is how far aft of the bow the blades' middle stands: the
// file's distance aft of the hull's end, on the boss.
func (size alfaSizes) screwAft() float64 {
	return size.hullEnd() + size.Propeller.Aft
}

// widest is a blade's chord where it is widest, at its middle.
func (size alfaSizes) widest() float64 {
	return size.Propeller.Chord * (alfaBladeEnds + alfaBladeSwell)
}

// propellerFits refuses a propeller of no blades, one whose tip does
// not clear its root or reaches past the keel, one whose blades stand
// off the boss's straight part, and one that reaches the rudders.
func (size alfaSizes) propellerFits() error {
	screw := size.Propeller
	half := size.widest() * spaceHalf
	if screw.Blades < alfaLeastBlades ||
		size.bladeTip() <= size.Hull.Stern ||
		size.bladeTip() > size.radius() ||
		size.screwAft()-half <= size.hullEnd() ||
		size.screwAft()+half >= size.Length-size.Hull.Stern ||
		size.screwAft()-half <= size.flapLead()+math.Max(size.Stern.Rudder,
			size.Stern.Flap) {
		return fmt.Errorf("manufacture: the alfa's propeller, %d blades "+
			"%v across at %v aft of the bow, does not fit its boss",
			screw.Blades, screw.Diameter, size.screwAft())
	}
	return nil
}

// bladeRing is one section of a blade a share of the way from its root
// to its tip, in the propeller's space: its foil wrapped round the axis
// at its radius, its chord on the helix of the propeller's pitch, and
// turned round by its skew.
func (size alfaSizes) bladeRing(share float64) []maths.Vec {
	screw := size.Propeller
	radius := size.bladeRoot() + (size.bladeTip()-size.bladeRoot())*share
	chord := screw.Chord * (alfaBladeEnds +
		alfaBladeSwell*math.Sin(math.Pi*share))
	thick := screw.Thick * (spaceOne - alfaBladeThin*share)
	pitch := math.Atan(screw.Pitch * screw.Diameter / (fullTurn * radius))
	skew := screw.Skew * maths.Degree * share * share
	var ring []maths.Vec
	for _, corner := range alfaFoil(chord, thick, alfaBladeTail) {
		along := corner[spaceZero] - chord*spaceHalf
		round := along*math.Cos(pitch) - corner[spaceOne]*math.Sin(pitch)
		aft := along*math.Sin(pitch) + corner[spaceOne]*math.Cos(pitch)
		turn := skew + round/radius
		ring = append(ring, maths.V(-radius*math.Sin(turn),
			radius*math.Cos(turn), aft))
	}
	return ring
}

// blade is one blade's loft, standing up from the axis in the
// propeller's space.
func (size alfaSizes) blade() alfaLoft {
	var rings [][]maths.Vec
	for ring := range alfaBladeRings {
		rings = append(rings, size.bladeRing(float64(ring)/
			(alfaBladeRings-spaceOne)))
	}
	return alfaLoft{rings: rings}
}

// propeller sets the blades evenly round the axis at the propeller's
// place on the boss.
func (size alfaSizes) propeller(assembly *instrumentAssembly) {
	screw := size.Propeller
	blade := size.blade()
	middle := size.at(spaceZero, size.radius(), size.screwAft())
	for at := range screw.Blades {
		turn := fullTurn * float64(at) / float64(screw.Blades)
		assembly.add(blade, maths.NewSubspace(middle,
			maths.Rotation(maths.ZAxis, turn), spaceOne), alfaBlade)
	}
}

// auxiliaryPropellers puts the two sourced two-blade manoeuvring screws
// on the ends of the stern-plane motor pods. Their size and exact seat are
// authored approximations; the article gives their count and location, not
// dimensions. They use the same native foil blade as the main screw.
func (size alfaSizes) auxiliaryPropellers(assembly *instrumentAssembly) {
	if size.Auxiliary.Blades <= 0 || size.Auxiliary.Scale <= 0 {
		return
	}
	stern := size.Stern
	for _, side := range []float64{-spaceOne, spaceOne} {
		centre := size.at(side*stern.Across, size.radius(),
			size.podMiddle()+size.Pods.Long*spaceHalf)
		for blade := range size.Auxiliary.Blades {
			turn := fullTurn * float64(blade) / float64(size.Auxiliary.Blades)
			assembly.add(size.blade(), maths.NewSubspace(centre,
				maths.Rotation(maths.ZAxis, turn), size.Auxiliary.Scale), alfaAuxBlade)
		}
	}
}
