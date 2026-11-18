package manufacture

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// highwayRamps connects both carriageways to two actual local grid streets.
func (city ClusteredCityLayout) highwayRamps(road CityHighway) []CityRamp {
	run := road.To.Sub(road.From)
	direction := run.Unit()
	normal := maths.V(-direction.Z, 0, direction.X)
	width := (road.Width.InMetres() - road.Median.InMetres()) / 2
	offset := road.Median.InMetres()/2 + width/2
	pitch := city.base.grid.BlockLength.InMetres()
	origin := maths.V(-float64(city.base.grid.BlocksX)*pitch/2, 0, -float64(city.base.grid.BlocksZ)*pitch/2)
	var ramps []CityRamp
	for _, share := range []float64{.32, .66} {
		cross := road.From.Add(run.Scale(share))
		street := maths.ZAxis
		if math.Abs(direction.X) >= math.Abs(direction.Z) {
			x := origin.X + math.Round((cross.X-origin.X)/pitch)*pitch
			cross = road.From.Add(direction.Scale((x - road.From.X) / direction.X))
		} else {
			z := origin.Z + math.Round((cross.Z-origin.Z)/pitch)*pitch
			cross = road.From.Add(direction.Scale((z - road.From.Z) / direction.Z))
			street = maths.XAxis
		}
		for _, side := range []float64{-1, 1} {
			start := cross.Sub(direction.Scale(side * 140)).Add(normal.Scale(side * offset))
			start.Y = road.Elevation.InMetres()
			end := cross.Add(street.Scale(side * pitch))
			end.Y = .04
			first := start.Add(direction.Scale(side * 75))
			last := end.Sub(street.Scale(side * 70))
			ramps = append(ramps, CityRamp{Points: [4]maths.Vec{start, first, last, end}, Width: measure.Metres(6)})
		}
	}
	return ramps
}

// at evaluates the actual ramp centreline, including a smooth descent.
func (ramp CityRamp) at(t float64) maths.Vec {
	u := 1 - t
	return ramp.Points[0].Scale(u * u * u).Add(ramp.Points[1].Scale(3 * u * u * t)).Add(ramp.Points[2].Scale(3 * u * t * t)).Add(ramp.Points[3].Scale(t * t * t))
}

// streetClear keeps elevated supports off carriageways and cross alleys.
func (city ClusteredCityLayout) streetClear(at maths.Vec) bool {
	grid := city.base.grid
	pitch := grid.BlockLength.InMetres()
	for axis := range 2 {
		value := at.X + float64(grid.BlocksX)*pitch/2
		if axis == 1 {
			value = at.Z + float64(grid.BlocksZ)*pitch/2
		}
		within := math.Mod(value, pitch)
		if within < 0 {
			within += pitch
		}
		if math.Min(within, pitch-within) < grid.RoadWidth.InMetres()/2+3 || math.Abs(within-pitch/2) < grid.AlleyWidth.InMetres()/2+3 {
			return false
		}
	}
	return true
}

// infrastructure builds separated decks, real ramps and street-clearing piers.
func (city ClusteredCityLayout) infrastructure(batch *cityBatch) {
	a := instrumentAssembly{}
	for _, road := range city.highways {
		run := road.To.Sub(road.From)
		direction := run.Unit()
		normal := maths.V(-direction.Z, 0, direction.X)
		carriage := (road.Width.InMetres() - road.Median.InMetres()) / 2
		offset := road.Median.InMetres()/2 + carriage/2
		angle := -math.Atan2(run.Z, run.X)
		for _, side := range []float64{-1, 1} {
			centre := road.From.Add(road.To).Scale(.5).Add(normal.Scale(side * offset))
			centre.Y = road.Elevation.InMetres() - 1
			a.add(boxShape(run.Len(), 1, carriage), maths.NewSubspace(centre, maths.About(maths.YAxis, angle), 1), cityRoad)
			for _, edge := range []float64{-1, 1} {
				barrier := centre.Add(normal.Scale(edge * (carriage/2 - .35)))
				barrier.Y = road.Elevation.InMetres()
				a.add(boxShape(run.Len(), .6, .35), maths.NewSubspace(barrier, maths.About(maths.YAxis, angle), 1), cityPavement)
			}
			stations := int(run.Len() / 96)
			for i := 1; i < stations; i++ {
				at := road.From.Add(run.Scale(float64(i) / float64(stations))).Add(normal.Scale(side * offset))
				for _, shift := range []float64{0, 12, -12, 24, -24} {
					place := at.Add(direction.Scale(shift))
					if city.streetClear(place) {
						districtBox(&a, 3, road.Elevation.InMetres()-1, 3, place, cityPavement)
						break
					}
				}
			}
		}
	}
	for _, ramp := range city.ramps {
		a.add(cityRampDeck{ramp}, maths.Local(), cityRoad)
		for _, t := range []float64{.25, .5, .75} {
			at := ramp.at(t)
			height := at.Y - .7
			at.Y = 0
			if height > 0 && city.streetClear(at) {
				districtBox(&a, 2, height, 2, at, cityPavement)
			}
		}
	}
	batch.add(a.object.parts, maths.Local())
}

// districtBox is the ordinary centred box primitive in metre coordinates.
func districtBox(a *instrumentAssembly, w, h, d float64, at maths.Vec, material slot) {
	a.box(w, h, d, at.Sub(maths.V(w/2, 0, d/2)), material)
}

// cityRampDeck is one closed swept deck with no coincident segment caps.
type cityRampDeck struct{ ramp CityRamp }

// skin joins shared ramp sections and closes only the two connection ends.
func (deck cityRampDeck) skin(cut) surface {
	const steps = 16
	var rings [steps + 1][4]maths.Vec
	for i := range rings {
		t := float64(i) / steps
		centre := deck.ramp.at(t)
		direction := deck.ramp.at(math.Min(1, t+.001)).Sub(deck.ramp.at(math.Max(0, t-.001)))
		direction.Y = 0
		normal := maths.V(-direction.Z, 0, direction.X).Unit().Scale(deck.ramp.Width.InMetres() / 2)
		left, right := centre.Add(normal), centre.Sub(normal)
		rings[i] = [4]maths.Vec{left, right, right.Sub(maths.V(0, .7, 0)), left.Sub(maths.V(0, .7, 0))}
	}
	var skin surface
	for i := range steps {
		axis := deck.ramp.at((float64(i) + .5) / steps).Sub(maths.V(0, .35, 0))
		for side, a := range rings[i] {
			next := (side + 1) % 4
			pair := surface{{a, rings[i][next], rings[i+1][side]}, {rings[i][next], rings[i+1][next], rings[i+1][side]}}
			for _, triangle := range pair {
				centre := triangle[0].Add(triangle[1]).Add(triangle[2]).Scale(1.0 / 3)
				if triangle[1].Sub(triangle[0]).Cross(triangle[2].Sub(triangle[0])).Dot(centre.Sub(axis)) < 0 {
					triangle[1], triangle[2] = triangle[2], triangle[1]
				}
				skin = append(skin, triangle)
			}
		}
	}
	for _, end := range []int{0, steps} {
		ring := rings[end]
		centre := maths.Vec{}
		for _, p := range ring {
			centre = centre.Add(p.Scale(.25))
		}
		direction := deck.ramp.Points[1].Sub(deck.ramp.Points[0]).Neg()
		if end == steps {
			direction = deck.ramp.Points[3].Sub(deck.ramp.Points[2])
		}
		for i, p := range ring {
			triangle := face{centre, p, ring[(i+1)%4]}
			if triangle[1].Sub(triangle[0]).Cross(triangle[2].Sub(triangle[0])).Dot(direction) < 0 {
				triangle[1], triangle[2] = triangle[2], triangle[1]
			}
			skin = append(skin, triangle)
		}
	}
	return skin
}
