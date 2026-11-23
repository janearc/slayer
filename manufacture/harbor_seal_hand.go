package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// sealDigits is how many digits a seal's flipper has, thumb to fifth.
const sealDigits = 5

// sealHand lays out one flipper's five digits. reach is how far each
// digit runs past the wrist, in metres, from the rear chord station to
// the front one. splay is half the fan's spread in metres. web is the
// share of its two digits' mean reach that the skin between them keeps,
// and thin the share of a digit's thickness that web keeps. All of them
// are chosen to read as a fan with a scalloped edge, not measured.
type sealHand struct {
	reach            [sealDigits]float64
	splay, web, thin float64
}

// foreHand has the thumb at the front station and the longest, each
// digit after it shorter.
var foreHand = sealHand{
	reach: [sealDigits]float64{.040, .048, .057, .065, .072},
	splay: .040, web: .70, thin: .45,
}

// hindHand has the first and fifth digits longest, which is the shape of
// a seal's trailing flipper.
var hindHand = sealHand{
	reach: [sealDigits]float64{.080, .062, .054, .060, .082},
	splay: .052, web: .68, thin: .45,
}

// sealDigitStations names each outline vertex of a flipper opening by
// the chord station it lies on, from zero at the rear to sealDigits-1 at
// the front. The outline runs along the rear edge, down the lower face,
// along the front edge and back up the upper face; every opening is
// sealDigits-1 rows long.
func sealDigitStations(loop int) []int {
	rows := sealDigits - 1
	columns := loop/2 - rows
	stations := make([]int, loop)
	for index := range stations {
		switch {
		case index < columns:
			stations[index] = 0
		case index < columns+rows:
			stations[index] = index - columns
		case index < 2*columns+rows:
			stations[index] = rows
		default:
			stations[index] = rows - (index - 2*columns - rows)
		}
	}
	return stations
}

// labels is each vertex of a digit ring: the digit it ends, or minus one
// where the vertex is web between two digits. Even entries are outline
// vertices and odd ones the web vertices inserted between them.
func (hand sealHand) labels(loop int) []int {
	stations := sealDigitStations(loop)
	labels := make([]int, 2*loop)
	for index := range labels {
		labels[index] = -1
		if index%2 == 0 {
			labels[index] = stations[index/2]
		}
	}
	return labels
}

// reachAt is how far a point at a chord station runs past the wrist. A
// station between two digits is web, which falls short of both.
func (hand sealHand) reachAt(station float64) float64 {
	low := int(math.Floor(station))
	if float64(low) == station {
		return hand.reach[low]
	}
	return hand.web * (hand.reach[low] + hand.reach[low+1]) / 2
}

// thickness is the share of a section's thickness kept at a station.
func (hand sealHand) thickness(station float64) float64 {
	if station == math.Floor(station) {
		return 1
	}
	return hand.thin
}

// spread is a point's offset in the flipper's chord, thickness and span
// axes, at a station and a fan share. The span axis points away from the
// body, so side signs it.
func (hand sealHand) spread(station, fan, side float64) maths.Vec {
	middle := float64(sealDigits-1) / 2
	chord := (station - middle) / middle
	return maths.V(chord*hand.splay*fan, 0, side*hand.reachAt(station)*fan)
}

// fanRing places one digit ring: every outline vertex of the opening and
// a web vertex halfway to the next. Digits spread across the chord and
// run out by their reach, and web is thinner and shorter between them.
func (rig *SealRig) fanRing(limb int, boundary []int, section sealBranchRing,
	hand sealHand,
) ([]int, maths.Vec) {
	joint := rig.limbs[limb]
	to := joint.root.Add(maths.V(section.dx, section.dy, joint.side*section.dz))
	stations := sealDigitStations(len(boundary))
	ring := make([]int, 0, 2*len(boundary))
	for index := range boundary {
		next := (index + 1) % len(boundary)
		for half := range 2 {
			base := rig.vertices[boundary[index]].point
			station := float64(stations[index])
			if half == 1 {
				base = base.Add(rig.vertices[boundary[next]].point).Scale(.5)
				station = (station + float64(stations[next])) / 2
			}
			point, spine := rig.sectionPoint(limb, base, section,
				hand.thickness(station), hand.spread(station, section.fan, joint.side))
			ring = append(ring, rig.add(point, spine, limb, section.influence))
		}
	}
	return ring, to
}
