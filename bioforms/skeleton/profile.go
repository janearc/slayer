package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// profileRings is the number of rings along a profiled bone; chosen,
// enough to follow a skull's changes of section.
const profileRings = 16

// Profile is a straight bone whose section changes along it: an ellipse
// wide across and deep square to that, given at evenly spaced stations
// from its start to its end and eased linearly between. A skull's
// braincase and rostrum, a sternum. Across is square to the bone and to
// up; deep is square to both.
type Profile struct {
	from, to   maths.Vec
	up         maths.Vec
	wide, deep []float64
}

// NewProfile is a profiled bone from from to to, its half sizes wide and
// deep at evenly spaced stations, the first at from and the last at to.
func NewProfile(from, to, up maths.Vec, wide, deep []float64) Profile {
	return Profile{from: from, to: to, up: up, wide: wide, deep: deep}
}

// At is the point t along the bone's middle line, from nought to one.
func (profile Profile) At(t float64) maths.Vec {
	return profile.from.Add(profile.to.Sub(profile.from).Scale(t))
}

// Size is the half sizes of the section t along the bone.
func (profile Profile) Size(t float64) (wide, deep float64) {
	return stationAt(profile.wide, t), stationAt(profile.deep, t)
}

// stationAt is the value of stations, evenly spaced over nought to one,
// at t, eased linearly between them and held at the ends. No stations
// give nought.
func stationAt(stations []float64, t float64) float64 {
	if len(stations) == 0 {
		return 0
	}
	if len(stations) == 1 {
		return stations[0]
	}
	place := math.Min(math.Max(t, 0), 1) * float64(len(stations)-1)
	low := min(int(place), len(stations)-2)
	share := place - float64(low)
	return stations[low] + share*(stations[low+1]-stations[low])
}

// Middle is the bone's line, end to end.
func (profile Profile) Middle() []maths.Vec {
	return []maths.Vec{profile.from, profile.to}
}

// Build adds the bone to the mesh as one closed tube.
func (profile Profile) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	return Sweep(points, faces, profileRings, profile.At, profile.Size,
		profile.up, place)
}
