package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/manufacture/geometry"
	"math"

	"github.com/janearc/slayer/maths"
)

// The Alfa's sail, the rescue sphere inside it, and the lofted solid
// and foil section the sail and later the fins and blades are made of.

// alfaSection is one level of the sail: so far up over the hull's top,
// its leading edge so far aft of the sail's front, its chord and how
// wide it is there.
type alfaSection struct{ Up, Lead, Chord, Wide float64 }

const (
	alfaFoilSteps     = 16   // chosen: corners down each side of a foil's section; enough to read as round
	alfaRound         = 0.35 // chosen: the share of a foil's chord over which its nose is round
	alfaSailTail      = 0.25 // chosen: the share of its widest a sail section keeps at its blunt trailing edge
	alfaLeastSections = 2    // defined: a loft needs two sections, one at each end
	alfaBallBands     = 12   // chosen: bands from pole to pole on the sphere; enough to read as round
	alfaFitSteps      = 12   // chosen: heights, and places along each, the sphere's fit is checked at
)

// alfaLoft is a solid lofted through rings of points, each ring a
// closed loop with as many corners as the next, capped at both ends.
// Its skin is wound outward whichever way its rings run.
type alfaLoft struct{ rings [][]maths.Vec }

// skin joins each ring to the next, corner to corner, and caps the
// first and last rings with fans from their middles.
func (shape alfaLoft) skin(cut) surface { return surface(geometry.Loft[face](shape.rings)) }

func (shape alfaLoft) normals(skin surface) [][3]maths.Vec { return geometry.Creased(skin) }

func creased(skin surface) [][3]maths.Vec { return geometry.Creased(skin) }

// held is the signed volume a closed skin holds.
func (skin surface) held() float64 { return geometry.Volume(skin) }

// alfaHalf is a foil's half width a share of the way along its chord:
// a quarter ellipse over its round nose, then tapering aft to its tail's
// share of its widest.
func alfaHalf(share, wide, tail float64) float64 {
	half := wide * spaceHalf
	if share < alfaRound {
		ahead := (alfaRound - share) / alfaRound
		return half * math.Sqrt(spaceOne-ahead*ahead)
	}
	aft := (share - alfaRound) / (spaceOne - alfaRound)
	return half * (spaceOne - (spaceOne-tail)*aft*aft)
}

// alfaFoil is the section alfaHalf gives, as alfaLoop lays it.
func alfaFoil(chord, wide, tail float64) [][coordinatePair]float64 {
	return alfaLoop(chord, func(share float64) float64 {
		return alfaHalf(share, wide, tail)
	})
}

// alfaLoop is a foil's section as a loop of points, each so far along
// its chord and so far across it, by a half thickness at each share of
// its chord: its leading point, down one side to the trailing edge, and
// back up the other, closer together at its nose.
func alfaLoop(chord float64,
	thickness func(float64) float64,
) [][coordinatePair]float64 {
	loop := [][coordinatePair]float64{{spaceZero, spaceZero}}
	var back [][coordinatePair]float64
	for step := spaceOne; step <= alfaFoilSteps; step++ {
		share := spaceOne - math.Cos(math.Pi*spaceHalf*float64(step)/
			alfaFoilSteps)
		half := thickness(share)
		loop = append(loop, [coordinatePair]float64{chord * share, half})
		back = append(back, [coordinatePair]float64{chord * share, -half})
	}
	for at := len(back) - spaceOne; at >= spaceZero; at-- {
		loop = append(loop, back[at])
	}
	return loop
}

// alfaHermite is a cubic through two values with the slopes a uniform
// Catmull-Rom spline gives them from their neighbours, a share of the
// way from the first to the second.
func alfaHermite(before, low, high, after, share float64) float64 {
	return geometry.CatmullRom(before, low, high, after, share)
}

const alfaSailSteps = 4 // chosen: interpolated levels between authored sections

// sailLevels is the sail's levels as drawn: the file's, and between each
// two, levels on a Catmull-Rom spline through them, so its height,
// lead, chord and width change smoothly from its foot to its crown.
func (size alfaSizes) sailLevels() []alfaSection {
	sections := size.Sail.Sections
	last := len(sections) - spaceOne
	var levels []alfaSection
	for at := spaceOne; at <= last; at++ {
		before := sections[max(at-coordinatePair, spaceZero)]
		low, high := sections[at-spaceOne], sections[at]
		after := sections[min(at+spaceOne, last)]
		for step := range alfaSailSteps {
			share := float64(step) / alfaSailSteps
			curve := func(of func(alfaSection) float64) float64 {
				return alfaHermite(of(before), of(low), of(high), of(after),
					share)
			}
			levels = append(levels, alfaSection{
				Up:    curve(func(one alfaSection) float64 { return one.Up }),
				Lead:  curve(func(one alfaSection) float64 { return one.Lead }),
				Chord: curve(func(one alfaSection) float64 { return one.Chord }),
				Wide:  curve(func(one alfaSection) float64 { return one.Wide })})
		}
	}
	return append(levels, sections[last])
}

// sailRings is the sail's levels as rings in the boat's own space,
// each across to starboard and port and along from its leading edge
// aft.
func (size alfaSizes) sailRings() [][]maths.Vec {
	var rings [][]maths.Vec
	for _, section := range size.sailLevels() {
		var ring []maths.Vec
		for _, corner := range alfaFoil(section.Chord, section.Wide,
			alfaSailTail) {
			ring = append(ring, size.at(corner[spaceOne], size.Beam+section.Up,
				size.Sail.Front+section.Lead+corner[spaceZero]))
		}
		rings = append(rings, ring)
	}
	return rings
}

// sailSeated refuses a sail of fewer than two levels, levels out of
// order, a foot not below the hull's top or not wholly inside the hull,
// so that the sail would stand off it, and a top not above it.
func (size alfaSizes) sailSeated() error {
	sections := size.Sail.Sections
	refused := fmt.Errorf("manufacture: the alfa's sail, %d levels from "+
		"%v aft of the bow, does not stand on the hull", len(sections),
		size.Sail.Front)
	if len(sections) < alfaLeastSections ||
		sections[spaceZero].Up >= spaceZero ||
		sections[len(sections)-spaceOne].Up <= spaceZero {
		return refused
	}
	levels := size.sailLevels()
	for at := spaceOne; at < len(levels); at++ {
		if levels[at].Up <= levels[at-spaceOne].Up ||
			!sound(levels[at].Chord) || !sound(levels[at].Wide) {
			return refused
		}
	}
	foot := sections[spaceZero]
	for _, corner := range alfaFoil(foot.Chord, foot.Wide, alfaSailTail) {
		aft := size.Sail.Front + foot.Lead + corner[spaceZero]
		if math.Hypot(corner[spaceOne], size.radius()+foot.Up) >=
			size.hullRadius(aft) {
			return refused
		}
	}
	return nil
}

// sailHalf is how far the sail reaches to either side so high over the
// keel and so far aft of the bow: its levels' shapes, met between them
// as the loft meets them, and nought off it. The loft's flat faces
// stand a little inside this, by less than the sphere's clearance.
func (size alfaSizes) sailHalf(up, aft float64) float64 {
	sections := size.sailLevels()
	height := up - size.Beam
	for at := spaceOne; at < len(sections); at++ {
		low, high := sections[at-spaceOne], sections[at]
		if height < low.Up || height > high.Up {
			continue
		}
		mix := (height - low.Up) / (high.Up - low.Up)
		lead := low.Lead + (high.Lead-low.Lead)*mix
		chord := low.Chord + (high.Chord-low.Chord)*mix
		share := (aft - size.Sail.Front - lead) / chord
		if share < spaceZero || share > spaceOne {
			return spaceZero
		}
		return alfaHalf(share, low.Wide+(high.Wide-low.Wide)*mix, alfaSailTail)
	}
	return spaceZero
}

// sphereCentre is where the rescue sphere's middle stands.
func (size alfaSizes) sphereCentre() maths.Vec {
	return size.at(spaceZero, size.Beam+size.Sphere.Up,
		size.Sail.Front+size.Sphere.Aft)
}

// sphereFits refuses a rescue sphere that does not fit inside the sail
// by its clearance: at each height through it, every point of its round
// cut there is inside the sail's section there, clear of its skin.
func (size alfaSizes) sphereFits() error {
	sphere := size.Sphere
	radius := sphere.Diameter * spaceHalf
	centre := size.Beam + sphere.Up
	middle := size.Sail.Front + sphere.Aft
	for level := range alfaFitSteps + spaceOne {
		up := centre - radius + sphere.Diameter*float64(level)/alfaFitSteps
		round := math.Sqrt(math.Max(spaceZero, (radius-up+centre)*
			(radius+up-centre)))
		for place := range alfaFitSteps + spaceOne {
			along := round * (float64(place)*coordinatePair/alfaFitSteps -
				spaceOne)
			across := math.Sqrt(math.Max(spaceZero, (round-along)*
				(round+along)))
			if size.sailHalf(up, middle+along) < across+sphere.Clear {
				return fmt.Errorf("manufacture: the alfa's rescue sphere, "+
					"%v across, does not fit inside its sail", sphere.Diameter)
			}
		}
	}
	return nil
}

// alfaBall is a ball's turned profile, from its bottom pole to its top.
func alfaBall(radius float64) [][coordinatePair]float64 {
	var side [][coordinatePair]float64
	for band := range alfaBallBands + spaceOne {
		turn := math.Pi * float64(band) / alfaBallBands
		reach := radius * math.Sin(turn)
		if band == spaceZero || band == alfaBallBands {
			reach = spaceZero // the poles exactly, not sin's residue
		}
		side = append(side, [coordinatePair]float64{reach,
			-radius * math.Cos(turn)})
	}
	return side
}
