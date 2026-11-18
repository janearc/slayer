package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

type urbanTerrace struct {
	footprint maths.Box
	height    float64
	role      slot
}

// urbanTerracePlan makes four disjoint strips in each existing and infill lot
// around its admitted building. Adjacent lots can meet into built districts;
// roads, alleys, parks, the building itself and tree hulls stay untouched.
func (city ClusteredCityLayout) urbanTerracePlan(infill []CityBuilding) []urbanTerrace {
	trees := parkInfillTreeBuckets(city.trees)
	var terraces []urbanTerrace
	add := func(building CityBuilding) {
		lot, b := building.Lot, building.Bounds
		if lot.Min.X >= lot.Max.X || lot.Min.Z >= lot.Max.Z || b.Min.X < lot.Min.X || b.Max.X > lot.Max.X || b.Min.Z < lot.Min.Z || b.Max.Z > lot.Max.Z {
			return
		}
		strips := []maths.Box{
			{Min: maths.V(lot.Min.X, 0, lot.Min.Z), Max: maths.V(b.Min.X, 0, lot.Max.Z)},
			{Min: maths.V(b.Max.X, 0, lot.Min.Z), Max: maths.V(lot.Max.X, 0, lot.Max.Z)},
			{Min: maths.V(b.Min.X, 0, lot.Min.Z), Max: maths.V(b.Max.X, 0, b.Min.Z)},
			{Min: maths.V(b.Min.X, 0, b.Max.Z), Max: maths.V(b.Max.X, 0, lot.Max.Z)},
		}
		own := sow(building.Seed)
		for i, box := range strips {
			if box.Max.X-box.Min.X < .10 || box.Max.Z-box.Min.Z < .10 || parkInfillMeetsTrees(box, trees) {
				continue
			}
			// Low, asymmetrically stepped podiums support existing architecture;
			// no part intrudes on a window or tree or changes the original building.
			height := .35 + 2.25*own.draw(212, i, 0)
			role := cityPavement
			if own.draw(212, i, 1) > .6 {
				role = cityRoof
			}
			terraces = append(terraces, urbanTerrace{footprint: box, height: height, role: role})
		}
	}
	for _, building := range city.base.buildings {
		add(building)
	}
	for _, building := range infill {
		add(building)
	}
	return terraces
}

// addUrbanTerraces adds the terrace podiums of the urban plan to batch. A
// podium without a finite, positive height is skipped.
func (city ClusteredCityLayout) addUrbanTerraces(batch *cityBatch, infill []CityBuilding) {
	for _, terrace := range city.urbanTerracePlan(infill) {
		b := terrace.footprint
		if !finite(terrace.height) || terrace.height <= 0 || math.IsInf(terrace.height, 0) {
			continue
		}
		// Separate independently closed podiums by 2.5 mm at the orbital's
		// 250x scale. Coincident edges between same-material solids otherwise
		// make their combined native mesh non-manifold and fail validation.
		const seam = .00001
		b.Min.X += seam
		b.Min.Z += seam
		b.Max.X -= seam
		b.Max.Z -= seam
		a := instrumentAssembly{}
		a.box(b.Max.X-b.Min.X, terrace.height, b.Max.Z-b.Min.Z, b.Min, terrace.role)
		batch.add(a.object.parts, maths.Local())
	}
}
