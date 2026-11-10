package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// cactusStem sweeps a pleated column through an optional quarter-circle arm.
// Its upright run ends in a rounded cap; its base is closed inside the trunk.
type cactusStem struct {
	run, radius, bend, depth float64
	ribs                     int
	curve, cap               int
}

// cactusStation keeps each ring's centre, tangent and physical radius together.
type cactusStation struct {
	at, along maths.Vec
	radius    float64
}

// stations samples one fixed path without changing its dimensions by detail.
func (stem cactusStem) stations() []cactusStation {
	var rows []cactusStation
	if stem.bend > spaceZero {
		for index := spaceZero; index <= stem.curve; index++ {
			angle := math.Pi * spaceHalf * float64(index) / float64(stem.curve)
			sine, cosine := math.Sincos(angle)
			at := maths.V(stem.bend*sine, stem.bend*(spaceOne-cosine), spaceZero)
			rows = append(rows, cactusStation{at: at,
				along: maths.V(cosine, sine, spaceZero), radius: stem.radius})
		}
	} else {
		rows = append(rows, cactusStation{along: maths.YAxis, radius: stem.radius})
	}
	shoulder := maths.V(stem.bend, stem.bend+stem.run, spaceZero)
	rows = append(rows, cactusStation{at: shoulder, along: maths.YAxis, radius: stem.radius})
	return stem.rounded(rows, shoulder)
}

// rounded finishes with an exact single point, avoiding zero-area tip quads.
func (stem cactusStem) rounded(rows []cactusStation, shoulder maths.Vec) []cactusStation {
	for index := spaceOne; index <= stem.cap; index++ {
		angle := math.Pi * spaceHalf * float64(index) / float64(stem.cap)
		sine, cosine := math.Sincos(angle)
		if index == stem.cap {
			sine, cosine = spaceOne, spaceZero
		}
		at := shoulder.Add(maths.YAxis.Scale(stem.radius * sine))
		rows = append(rows, cactusStation{at: at, along: maths.YAxis,
			radius: stem.radius * cosine})
	}
	return rows
}

// ring keeps pleats longitudinal while the arm tangent turns upward.
func (stem cactusStem) ring(row cactusStation, sides int) []maths.Vec {
	side := maths.V(row.along.Y, -row.along.X, spaceZero)
	ring := make([]maths.Vec, sides)
	for index := range sides {
		angle := fullTurn * float64(index) / float64(sides)
		sine, cosine := math.Sincos(angle)
		rib := spaceOne - stem.depth + stem.depth*math.Cos(float64(stem.ribs)*angle)
		radial := side.Scale(cosine).Sub(maths.ZAxis.Scale(sine))
		ring[index] = row.at.Add(radial.Scale(row.radius * rib))
	}
	return ring
}

// skin closes the base and joins each sampled ring with consistent winding.
func (stem cactusStem) skin(fine cut) surface {
	rows := stem.stations()
	previous := stem.ring(rows[spaceZero], fine.around)
	var skin surface
	for index := range fine.around {
		next := (index + spaceOne) % fine.around
		skin = append(skin, face{rows[spaceZero].at, previous[next], previous[index]})
	}
	for _, row := range rows[spaceOne:] {
		current := stem.ring(row, fine.around)
		skin = cactusRingBand(skin, previous, current, row)
		previous = current
	}
	return skin
}

// cactusRingBand uses one face per side at the exact pointed cap endpoint.
func cactusRingBand(skin surface, previous, current []maths.Vec,
	row cactusStation,
) surface {
	for index := range previous {
		next := (index + spaceOne) % len(previous)
		if row.radius == spaceZero {
			skin = append(skin, face{previous[index], previous[next], row.at})
		} else {
			skin = quad(skin, previous[index], previous[next], current[next], current[index])
		}
	}
	return skin
}

// bounds includes spine tips even at representations which omit the spines.
func (stem cactusStem) bounds() maths.Box {
	radius := stem.radius + cactusValue("spine-length")
	bottom := float64(spaceZero)
	if stem.bend > spaceZero {
		bottom = -radius
	}
	low := maths.V(-radius, bottom, -radius)
	high := maths.V(stem.bend+radius, stem.bend+stem.run+radius, radius)
	return maths.SegmentBox(low, high, spaceZero)
}
