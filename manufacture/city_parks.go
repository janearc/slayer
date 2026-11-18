package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

const (
	cityVacantSoil   slot = 12
	cityVacantMeadow slot = 13
)

// parkPaths supplies a perimeter loop and two winding through routes.
// Crossings differ by a centimetre vertically, avoiding coplanar overlaps.
func parkPaths(park CityPark) [][]maths.Vec {
	w, d := park.Width.InMetres(), park.Depth.InMetres()
	var loop, long, cross []maths.Vec
	for i := 0; i <= 64; i++ {
		t := float64(i) / 64
		angle := 2 * math.Pi * t
		loop = append(loop, park.At.Add(maths.V(w*.38*math.Cos(angle), .075, d*.42*math.Sin(angle))))
		long = append(long, park.At.Add(maths.V(w*.15*math.Sin(2*math.Pi*t), .085, (t-.5)*d*.84)))
		cross = append(cross, park.At.Add(maths.V((t-.5)*w*.76, .095, d*.11*math.Sin(2*math.Pi*t))))
	}
	loop[len(loop)-1] = loop[0]
	return [][]maths.Vec{loop, long, cross}
}

// parkRibbon is ordinary open upward geometry, with shared section vertices.
func parkRibbon(points []maths.Vec, width float64) surface {
	var left, right []maths.Vec
	closed := points[0] == points[len(points)-1]
	for i, at := range points {
		low, high := max(0, i-1), min(len(points)-1, i+1)
		if closed && (i == 0 || i == len(points)-1) {
			low, high = len(points)-2, 1
		}
		direction := points[high].Sub(points[low])
		direction.Y = 0
		normal := maths.V(-direction.Z, 0, direction.X).Unit().Scale(width / 2)
		left = append(left, at.Add(normal))
		right = append(right, at.Sub(normal))
	}
	var skin surface
	for i := 0; i < len(points)-1; i++ {
		pair := surface{{left[i], right[i], left[i+1]}, {right[i], right[i+1], left[i+1]}}
		for _, triangle := range pair {
			if triangle[1].Sub(triangle[0]).Cross(triangle[2].Sub(triangle[0])).Y < 0 {
				triangle[1], triangle[2] = triangle[2], triangle[1]
			}
			skin = append(skin, triangle)
		}
	}
	return skin
}

// parkPathDistance protects walking circulation from planted trunks.
func parkPathDistance(park CityPark, at maths.Vec) float64 {
	nearest := math.Inf(1)
	for _, path := range parkPaths(park) {
		for i := 1; i < len(path); i++ {
			a, b := path[i-1], path[i]
			a.Y, b.Y, at.Y = 0, 0, 0
			run := b.Sub(a)
			t := math.Max(0, math.Min(1, at.Sub(a).Dot(run)/run.Dot(run)))
			nearest = math.Min(nearest, at.Sub(a.Add(run.Scale(t))).Len())
		}
	}
	return nearest
}

// parkGroveWeight makes seeded woodland masses around open meadow clearings.
func parkGroveWeight(park CityPark, at maths.Vec, from seed) float64 {
	w, d := park.Width.InMetres(), park.Depth.InMetres()
	meadow := math.Pow((at.X-park.At.X-w*.1)/(w*.2), 2) + math.Pow((at.Z-park.At.Z)/(d*.13), 2)
	if meadow < 1 {
		return 0
	}
	weight := 0.0
	for grove := range 9 {
		centre := park.At.Add(maths.V(from.swing(140, grove, 0)*w*.36, 0, from.swing(140, grove, 1)*d*.4))
		radius := math.Min(w, d) * (.12 + .13*from.draw(140, grove, 2))
		distance := at.Sub(centre)
		distance.Y = 0
		weight += math.Exp(-distance.Dot(distance) / (2 * radius * radius))
	}
	return math.Min(1, weight)
}

// vacantLandscape replaces unused paved lot centres with soil and meadow.
func (city ClusteredCityLayout) vacantLandscape(batch *cityBatch, from seed) error {
	vacant := city.vacant
	if !city.vacantIndexed {
		all, err := PlanCity(city.base.grid, from)
		if err != nil {
			return err
		}
		occupied := make(map[uint64]bool, len(city.base.buildings))
		for _, building := range city.base.buildings {
			occupied[building.Seed] = true
		}
		vacant = nil
		for _, building := range all.buildings {
			if !occupied[building.Seed] && !city.reserved(building.Lot, 0) {
				vacant = append(vacant, cityVacantLot{seed: building.Seed, lot: building.Lot})
			}
		}
	}
	for _, lot := range vacant {
		role := cityVacantSoil
		if sow(lot.seed).draw(141, 0, 0) < .65 {
			role = cityVacantMeadow
		}
		batch.rectangle(role, lot.lot.Min.X, lot.lot.Min.Z,
			lot.lot.Max.X-lot.lot.Min.X, lot.lot.Max.Z-lot.lot.Min.Z, -.005)
	}
	return nil
}
