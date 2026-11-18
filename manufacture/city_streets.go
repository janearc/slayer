package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// paving builds one connected street fabric. Straight road sections stop at
// each junction. A circle's necks meet its four approaches along chords;
// its central island is a separate raised solid, never a through road.
func (city CityLayout) paving(batch *cityBatch) {
	grid := city.grid
	pitch, half := grid.BlockLength.InMetres(), grid.RoadWidth.InMetres()/2
	startX, startZ := -float64(grid.BlocksX)*pitch/2, -float64(grid.BlocksZ)*pitch/2
	reach := func(x, z int) float64 {
		if grid.circle(x, z) {
			radius := grid.RoundaboutRadius.InMetres()
			return math.Sqrt(radius*radius - half*half)
		}
		return half
	}
	for z := 0; z <= grid.BlocksZ; z++ {
		for x := 0; x <= grid.BlocksX; x++ {
			at := maths.V(startX+float64(x)*pitch, 0, startZ+float64(z)*pitch)
			if grid.circle(x, z) {
				city.trafficCircle(batch, at)
			} else {
				batch.rectangle(cityRoad, at.X-half, at.Z-half, 2*half, 2*half, 0)
			}
			if x < grid.BlocksX {
				left, right := reach(x, z), reach(x+1, z)
				batch.rectangle(cityRoad, at.X+left, at.Z-half, pitch-left-right, 2*half, 0)
			}
			if z < grid.BlocksZ {
				low, high := reach(x, z), reach(x, z+1)
				batch.rectangle(cityRoad, at.X-half, at.Z+low, 2*half, pitch-low-high, 0)
			}
			if x < grid.BlocksX && z < grid.BlocksZ {
				city.blockPaving(batch, at)
			}
		}
	}
}

// rectangle is an open upward surface; roads have no hidden closed boxes
// touching one another or coincident horizontal faces at intersections.
func (batch *cityBatch) rectangle(material slot, x, z, width, depth, y float64) {
	a, b := maths.V(x, y, z), maths.V(x+width, y, z)
	c, d := maths.V(x+width, y, z+depth), maths.V(x, y, z+depth)
	batch.skin(material, surface{{a, c, b}, {a, d, c}})
}

// blockPaving leaves a cross of four-metre access alleys between lot groups.
// Paving lies below road surfaces and buildings to avoid coincident floors.
func (city CityLayout) blockPaving(batch *cityBatch, at maths.Vec) {
	pitch := city.grid.BlockLength.InMetres()
	half := city.grid.RoadWidth.InMetres() / 2
	alley := city.grid.AlleyWidth.InMetres()
	arm := pitch/2 - half - alley/2
	for _, dx := range []float64{half, pitch/2 + alley/2} {
		for _, dz := range []float64{half, pitch/2 + alley/2} {
			batch.rectangle(cityPavement, at.X+dx, at.Z+dz, arm, arm, -cityValue("pavement-drop"))
		}
	}
	middle := pitch/2 - alley/2
	batch.rectangle(cityRoad, at.X+half, at.Z+middle, pitch-2*half, alley, 0)
	batch.rectangle(cityRoad, at.X+middle, at.Z+half, alley, arm, 0)
	batch.rectangle(cityRoad, at.X+middle, at.Z+middle+alley, alley, arm, 0)
}

// trafficCircle clips four circular arcs to exact approach-width chords.
// Angular subdivision is an authored distance-detail budget. The radius
// describes the circular outside envelope, independent of those chords.
func (city CityLayout) trafficCircle(batch *cityBatch, at maths.Vec) {
	radius := city.grid.RoundaboutRadius.InMetres()
	half := city.grid.RoadWidth.InMetres() / 2
	angle := math.Asin(half / radius)
	arcSteps := int(cityValue("circle-arc-steps"))
	var outer, inner []maths.Vec
	for quadrant := range 4 {
		for step := 0; step <= arcSteps; step++ {
			turn := float64(quadrant)*math.Pi/2 + angle +
				float64(step)/float64(arcSteps)*(math.Pi/2-2*angle)
			radial := maths.V(math.Cos(turn), 0, math.Sin(turn))
			outer = append(outer, at.Add(radial.Scale(radius)))
			inner = append(inner, at.Add(radial.Scale(radius*cityValue("island-radius-share"))))
		}
	}
	var asphalt, island surface
	topCentre := at.Add(maths.V(0, cityValue("island-height"), 0))
	for i, a := range outer {
		j := (i + 1) % len(outer)
		b, c, d := inner[i], inner[j], outer[j]
		asphalt = append(asphalt, face{a, b, c}, face{a, c, d})
		high, next := b.Add(maths.V(0, cityValue("island-height"), 0)), c.Add(maths.V(0,
			cityValue("island-height"), 0))
		island = append(island, face{topCentre, next, high}, face{at, b, c},
			face{b, high, next}, face{b, next, c})
	}
	batch.skin(cityRoad, asphalt)
	batch.skin(cityIsland, island)
}
