package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

type clusteredCityMaker struct{ spec ClusteredCitySpec }

// made constructs a separate clustered city without changing the grid kind.
func (row clusteredCityMaker) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// madeAt batches all building, park, street and highway geometry by material.
func (row clusteredCityMaker) madeAt(from seed, detail string) (good, error) {
	if !cityLevel(detail) {
		return good{}, fmt.Errorf("manufacture: clustered city has no %s representation", detail)
	}
	layout, err := PlanClusteredCity(row.spec, from)
	if err != nil {
		return good{}, err
	}
	batch := newCityBatch()
	batch.placement = row.spec.Grid.BatchPlacement
	for _, building := range layout.base.buildings {
		choice := detail
		if row.spec.DetailView != nil {
			choice = row.spec.DetailView.detail(building, detail, row.spec.Night)
		}
		batch.addBuilding(building, choice, row.spec.Night)
	}
	var infill []CityBuilding
	if row.spec.ParkRelief {
		var err error
		infill, _, err = layout.vacantInfillPlan(from, row.spec.Process.Spread.InMetres())
		if err != nil {
			return good{}, err
		}
		for _, building := range infill {
			batch.add(districtBuildingParts(building, detail, row.spec.Night), building.Place)
		}
		layout.addUrbanTerraces(batch, infill)
		layout.addParkInfill(batch, from, detail, row.spec.Night)
	}
	layout.base.paving(batch)
	if row.spec.Trees != nil {
		if err := layout.vacantLandscape(batch, from); err != nil {
			return good{}, err
		}
	}
	if row.spec.ParkRelief {
		if err := layout.addUrbanService(batch, from, infill); err != nil {
			return good{}, err
		}
	}
	layout.landscape(batch)
	layout.infrastructure(batch)
	if row.spec.Night {
		layout.nightPoles(batch)
	}
	made, err := batch.finish()
	if err != nil {
		return good{}, err
	}
	if row.spec.Night {
		made.dressed = nightCityLooks()
	}
	if row.spec.Trees != nil {
		made.dressed[cityVacantSoil] = plain("city vacant soil", .13, .12, .085)
		made.dressed[cityVacantMeadow] = plain("city vacant meadow", .13, .22, .075)
	}
	if row.spec.ParkRelief {
		parkTerrainLooks(made.dressed)
	}
	if row.spec.Trees != nil {
		made, err = layout.treeGroups(made, *row.spec.Trees, from)
		if err != nil {
			return good{}, err
		}
	}
	if row.spec.Night {
		return layout.nightLights(made)
	}
	return made, nil
}

// landscape makes parks continuous over former street/lot surface patches.
// Paths form a connected cross, split at its junction to avoid overlapping
// coplanar triangles. Native trees are an explicit population option.
func (city ClusteredCityLayout) landscape(batch *cityBatch) {
	a := instrumentAssembly{}
	for _, park := range city.parks {
		w, d := park.Width.InMetres(), park.Depth.InMetres()
		if city.parkRelief {
			parkUrbanTerrain(batch, park, city.trees)
		} else {
			base := park.At
			base.Y = -.06
			districtBox(&a, w, .12, d, base, cityIsland)
		}
		if len(city.trees) > 0 || city.parkRelief {
			for pathIndex, path := range parkPaths(park) {
				if city.parkRelief {
					for i := range path {
						path[i].Y = .001 + float64(pathIndex)*.0001
					}
				}
				width := city.parkPathWidth.InMetres()
				if width == 0 {
					width = 4
				}
				batch.skin(cityPavement, parkRibbon(path, width))
			}
		} else {
			x, z := park.At.X, park.At.Z
			batch.rectangle(cityPavement, x-w/2, z-2, w, 4, .075)
			batch.rectangle(cityPavement, x-2, z-d/2, 4, d/2-2, .075)
			batch.rectangle(cityPavement, x-2, z+2, 4, d/2-2, .075)
		}
	}
	batch.add(a.object.parts, maths.Local())
}

// greenClusteredCity opts into native foliage while preserving prior defaults.
func greenClusteredCity(night bool) ClusteredCitySpec {
	spec := DefaultClusteredCity()
	trees := DefaultCityTrees()
	spec.Trees = &trees
	spec.Night = night
	return spec
}
