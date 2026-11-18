package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// urbanServicePlan restores paved service courts over otherwise unbuilt vacant
// lots. Each tile stays inside a reserved-safe lot and clear of every tree's
// complete bounding box; the original meadow remains beneath the new surface.
func (city ClusteredCityLayout) urbanServicePlan(from seed, infill []CityBuilding) ([]maths.Box, error) {
	lots, err := city.urbanVacantLots(from)
	if err != nil {
		return nil, err
	}
	occupied := make(map[uint64]bool, len(infill))
	for _, b := range infill {
		occupied[b.Seed] = true
	}
	trees := parkInfillTreeBuckets(city.trees)
	var service []maths.Box
	for _, lot := range lots {
		if occupied[lot.seed] {
			continue
		}
		w, d := lot.lot.Max.X-lot.lot.Min.X, lot.lot.Max.Z-lot.lot.Min.Z
		nx, nz := int(math.Ceil(w/4)), int(math.Ceil(d/4))
		if nx <= 0 || nz <= 0 {
			continue
		}
		for j := 0; j < nz; j++ {
			for i := 0; i < nx; i++ {
				box := maths.Box{Min: maths.V(lot.lot.Min.X+float64(i)*w/float64(nx), 0, lot.lot.Min.Z+float64(j)*d/float64(nz)), Max: maths.V(lot.lot.Min.X+float64(i+1)*w/float64(nx), 0, lot.lot.Min.Z+float64(j+1)*d/float64(nz))}
				if !parkInfillMeetsTrees(box, trees) {
					service = append(service, box)
				}
			}
		}
	}
	return service, nil
}

// addUrbanService adds the service paving of the urban plan to batch.
func (city ClusteredCityLayout) addUrbanService(batch *cityBatch, from seed, infill []CityBuilding) error {
	service, err := city.urbanServicePlan(from, infill)
	if err != nil {
		return err
	}
	for _, box := range service {
		batch.rectangle(parkService, box.Min.X, box.Min.Z, box.Max.X-box.Min.X, box.Max.Z-box.Min.Z, -.004)
	}
	return nil
}
