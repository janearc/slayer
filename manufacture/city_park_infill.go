package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

const parkInfillPitch = 28.0
const parkInfillTreeBucket = 64.0

type parkInfillBuilding struct {
	building CityBuilding
	base     float64
}

// parkInfillPlan adds independently sown buildings only to land that is
// already reserved for parks. The original city population is never edited.
// All tree hulls, paths, city roads and the camera grove remain disjoint.
func (city ClusteredCityLayout) parkInfillPlan(from seed) []parkInfillBuilding {
	buckets := parkInfillTreeBuckets(city.trees)
	var result []parkInfillBuilding
	// Parents and the small grove keep their original placement and identities.
	withoutParks := city
	withoutParks.parks = nil
	families := []string{"city-rowhouse", "city-apartment", "city-tower"}
	for pindex, park := range city.parks {
		w, d := park.Width.InMetres(), park.Depth.InMetres()
		paths := parkPaths(park)
		nx, nz := int(math.Floor(w/parkInfillPitch)), int(math.Floor(d/parkInfillPitch))
		startX, startZ := park.At.X-w/2, park.At.Z-d/2
		for iz := 0; iz < nz; iz++ {
			for ix := 0; ix < nx; ix++ {
				address := uint64(pindex+1)<<48 | uint64(iz+1)<<24 | uint64(ix+1)
				number := citySeed(from.number ^ address ^ 0x434f5255534341)
				own := sow(number)
				// Distinct families, irregular footprint sizes, rotations, heights,
				// material and occupied windows all remain the factory's own recipes.
				family := families[min(int(own.draw(210, 0, 0)*3), 2)]
				size := .62 + .47*own.draw(210, 0, 1)
				x := startX + (float64(ix)+.5)*parkInfillPitch + own.swing(210, 0, 2)*1.5
				z := startZ + (float64(iz)+.5)*parkInfillPitch + own.swing(210, 0, 3)*1.5
				angle := float64(int(own.draw(210, 0, 4)*4)) * math.Pi / 2
				design := cityBuildingDesign(family, own)
				place := maths.NewSubspace(maths.V(x, 0, z), maths.About(maths.YAxis, angle), size)
				box := transformed(design.envelope(), place)
				// Night windows protrude slightly from the original facade envelope.
				// Reserve their full reach before checking paths, trees and neighbors.
				box.Min.X -= .02 * size
				box.Min.Z -= .02 * size
				box.Max.X += .02 * size
				box.Max.Z += .02 * size
				if box.Min.X <= startX || box.Max.X >= startX+w || box.Min.Z <= startZ || box.Max.Z >= startZ+d {
					continue
				}
				// Bounding circle is conservative for the entire oriented rectangle.
				radius := .5 * math.Hypot(box.Max.X-box.Min.X, box.Max.Z-box.Min.Z)
				if parkTerrainPathDistance(paths, x, z) <= radius+.05 {
					continue
				}
				if pindex == 0 && boxesOverlapXZ(box, maths.Box{Min: park.At.Add(maths.V(-4.2, 0, -3.2)), Max: park.At.Add(maths.V(4.2, 0, 3.2))}) {
					continue
				}
				if withoutParks.reserved(box, 0) || city.parkInfillMeetsRoad(box) || city.parkInfillMeetsCircle(box) || parkInfillMeetsTrees(box, buckets) {
					continue
				}
				base := parkInfillHighPoint(park, paths, box)
				place.Origin.Y = base
				box.Min.Y += base
				box.Max.Y += base
				result = append(result, parkInfillBuilding{building: CityBuilding{Seed: number, Kind: family, Place: place, Lot: box, Bounds: box}, base: base})
			}
		}
	}
	return result
}

// boxesOverlapXZ reports whether two boxes overlap when seen from above.
func boxesOverlapXZ(a, b maths.Box) bool {
	return a.Min.X < b.Max.X && a.Max.X > b.Min.X && a.Min.Z < b.Max.Z && a.Max.Z > b.Min.Z
}

// parkInfillTreeBuckets sorts the tree bounds into grid buckets, so a box is
// checked only against the trees near it.
func parkInfillTreeBuckets(trees []CityTree) map[[2]int][]maths.Box {
	buckets := map[[2]int][]maths.Box{}
	for _, tree := range trees {
		b := tree.Bounds
		for x := int(math.Floor(b.Min.X / parkInfillTreeBucket)); x <= int(math.Floor(b.Max.X/parkInfillTreeBucket)); x++ {
			for z := int(math.Floor(b.Min.Z / parkInfillTreeBucket)); z <= int(math.Floor(b.Max.Z/parkInfillTreeBucket)); z++ {
				buckets[[2]int{x, z}] = append(buckets[[2]int{x, z}], b)
			}
		}
	}
	return buckets
}

// parkInfillMeetsTrees reports whether box overlaps a tree in the buckets it
// covers.
func parkInfillMeetsTrees(box maths.Box, buckets map[[2]int][]maths.Box) bool {
	for x := int(math.Floor(box.Min.X / parkInfillTreeBucket)); x <= int(math.Floor(box.Max.X/parkInfillTreeBucket)); x++ {
		for z := int(math.Floor(box.Min.Z / parkInfillTreeBucket)); z <= int(math.Floor(box.Max.Z/parkInfillTreeBucket)); z++ {
			for _, tree := range buckets[[2]int{x, z}] {
				if boxesOverlapXZ(box, tree) {
					return true
				}
			}
		}
	}
	return false
}

// The unmodified city grid includes roads even where the park used to hide
// them, so new structures must not occupy either the roads or access alleys.
func (city ClusteredCityLayout) parkInfillMeetsRoad(box maths.Box) bool {
	grid := city.base.grid
	pitch := grid.BlockLength.InMetres()
	startX := -float64(grid.BlocksX) * pitch / 2
	startZ := -float64(grid.BlocksZ) * pitch / 2
	crosses := func(low, high, start, width float64) bool {
		return math.Ceil((low-start-width/2)/pitch) <= math.Floor((high-start+width/2)/pitch)
	}
	return crosses(box.Min.X, box.Max.X, startX, grid.RoadWidth.InMetres()) ||
		crosses(box.Min.Z, box.Max.Z, startZ, grid.RoadWidth.InMetres()) ||
		crosses(box.Min.X, box.Max.X, startX+pitch/2, grid.AlleyWidth.InMetres()) ||
		crosses(box.Min.Z, box.Max.Z, startZ+pitch/2, grid.AlleyWidth.InMetres())
}

// A circle can extend beyond its street's width; preserve its whole island.
func (city ClusteredCityLayout) parkInfillMeetsCircle(box maths.Box) bool {
	for _, circle := range city.base.circles {
		x := math.Max(box.Min.X, math.Min(box.Max.X, circle.At.X))
		z := math.Max(box.Min.Z, math.Min(box.Max.Z, circle.At.Z))
		if math.Hypot(x-circle.At.X, z-circle.At.Z) <= circle.Radius.InMetres() {
			return true
		}
	}
	return false
}

// All vertices of cells touching the footprint bound the piecewise-linear
// ground beneath it, including grid diagonals and footprint-edge crossings.
func parkInfillHighPoint(park CityPark, paths [][]maths.Vec, box maths.Box) float64 {
	nx, nz, sx, sz := parkTerrainGrid(park)
	x0, z0 := park.At.X-park.Width.InMetres()/2, park.At.Z-park.Depth.InMetres()/2
	firstX := max(0, int(math.Floor((box.Min.X-x0)/sx)))
	lastX := min(nx, int(math.Ceil((box.Max.X-x0)/sx)))
	firstZ := max(0, int(math.Floor((box.Min.Z-z0)/sz)))
	lastZ := min(nz, int(math.Ceil((box.Max.Z-z0)/sz)))
	top := parkTerrainBase
	for j := firstZ; j <= lastZ; j++ {
		for i := firstX; i <= lastX; i++ {
			top = math.Max(top, parkTerrainVertex(park, paths, x0+float64(i)*sx, z0+float64(j)*sz))
		}
	}
	return top
}

// A solid plinth joins any sloping ground to the level building base. Its
// extent is the already-admitted building footprint, so it cannot reach a
// neighbouring tree, path, road or building.
func (city ClusteredCityLayout) addParkInfill(batch *cityBatch, from seed, detail string, night bool) {
	if len(city.parks) > 0 {
		park := city.parks[0]
		for _, side := range []float64{-1, 1} {
			// Four physical metres after the orbital scales its factory coordinates.
			a := park.At.Add(maths.V(side*4.18, .0013, -3))
			b := park.At.Add(maths.V(side*4.18, .0013, 3))
			batch.skin(cityPavement, parkRibbon([]maths.Vec{a, b}, 4.0/250))
		}
	}
	for _, infill := range city.parkInfillPlan(from) {
		building := infill.building
		if infill.base > parkTerrainBase+.00001 {
			b := building.Bounds
			a := instrumentAssembly{}
			a.box(b.Max.X-b.Min.X, infill.base, b.Max.Z-b.Min.Z, maths.V(b.Min.X, 0, b.Min.Z), cityPavement)
			batch.add(a.object.parts, maths.Local())
		}
		batch.add(districtBuildingParts(building, detail, night), building.Place)
	}
}

// vacantInfillPlan fills a seeded, cluster-weighted share of already planned
// empty lots. Their stable addresses and all original city buildings remain
// unchanged; candidate footprints must fit their lots and avoid every tree.
func (city ClusteredCityLayout) urbanVacantLots(from seed) ([]cityVacantLot, error) {
	vacant := city.vacant
	if !city.vacantIndexed || len(vacant) == 0 {
		all, err := PlanCity(city.base.grid, from)
		if err != nil {
			return nil, err
		}
		occupied := make(map[uint64]bool, len(city.base.buildings))
		for _, building := range city.base.buildings {
			occupied[building.Seed] = true
		}
		for _, building := range all.buildings {
			if !occupied[building.Seed] && !city.reserved(building.Lot, 0) {
				vacant = append(vacant, cityVacantLot{seed: building.Seed, lot: building.Lot})
			}
		}
	}
	return vacant, nil
}

// vacantInfillPlan places infill buildings on the vacant lots, each inside its
// lot and clear of the trees. It returns the buildings and the number of
// vacant lots.
func (city ClusteredCityLayout) vacantInfillPlan(from seed, spread float64) ([]CityBuilding, int, error) {
	vacant, err := city.urbanVacantLots(from)
	if err != nil {
		return nil, 0, err
	}
	trees := parkInfillTreeBuckets(city.trees)
	var made []CityBuilding
	for _, lot := range vacant {
		own := sow(lot.seed)
		at := lot.lot.Min.Add(lot.lot.Max).Scale(.5)
		at.Y = 0
		nearest := math.Inf(1)
		for _, parent := range city.parents {
			nearest = math.Min(nearest, at.Sub(parent.At).Len())
		}
		intensity := 0.0
		if spread > 0 && !math.IsInf(nearest, 0) {
			intensity = math.Exp(-.5 * nearest * nearest / (spread * spread))
		}
		if own.draw(211, 0, 0) > .85+.13*intensity {
			continue
		}
		family := "city-rowhouse"
		choice := own.draw(211, 0, 1)
		if intensity > .35 && choice > .35 {
			family = "city-tower"
		} else if choice > .3 {
			family = "city-apartment"
		}
		scale := .90 + .13*own.draw(211, 0, 2)
		angle := float64(int(own.draw(211, 0, 3)*4)) * math.Pi / 2
		place := maths.NewSubspace(at, maths.About(maths.YAxis, angle), scale)
		box := transformed(cityBuildingDesign(family, own).envelope(), place)
		// Include the night window projection regardless of current light setting.
		box.Min.X -= .02
		box.Min.Z -= .02
		box.Max.X += .02
		box.Max.Z += .02
		if box.Min.X < lot.lot.Min.X || box.Max.X > lot.lot.Max.X || box.Min.Z < lot.lot.Min.Z || box.Max.Z > lot.lot.Max.Z || parkInfillMeetsTrees(box, trees) {
			continue
		}
		made = append(made, CityBuilding{Seed: lot.seed, Kind: family, Place: place, Lot: lot.lot, Bounds: box})
	}
	return made, len(vacant), nil
}
