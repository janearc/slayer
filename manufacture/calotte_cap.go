package manufacture

import "math"

// calotteCap is a closed spherical wall around a circular observing hole.
// The whole sphere overlaps the supporting cylinder, avoiding an open belt
// when its observing pupil tilts. This is a scenery shell, not its mechanism.
type calotteCap struct{ radius, wall, opening float64 }

// skin follows the outer sphere upward, then returns along the inner sphere.
func (cap calotteCap) skin(fine cut) surface {
	levels := int(summitValue("calotte-levels"))
	points := make([][coordinatePair]float64, spaceZero)
	for index := range levels + spaceOne {
		points = append(points, cap.point(index, levels, cap.radius))
	}
	for index := levels; index >= spaceZero; index-- {
		points = append(points, cap.point(index, levels, cap.radius-cap.wall))
	}
	return (turnedProfile{points: points}).skin(fine)
}

// point says the pole exactly; sin(pi)'s tiny residue is not a real ring.
func (cap calotteCap) point(index, levels int,
	radius float64,
) [coordinatePair]float64 {
	if index == spaceZero {
		return [coordinatePair]float64{spaceZero, -radius}
	}
	share := float64(index) / float64(levels)
	angle := math.Pi + (cap.opening-math.Pi)*share
	sine, cosine := math.Sincos(angle)
	return [coordinatePair]float64{radius * sine, radius * cosine}
}
