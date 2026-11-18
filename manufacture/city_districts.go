package manufacture

import (
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// CityPark reserves a continuous rectangular landscape, in metre city space.
type CityPark struct {
	Name         string
	At           maths.Vec
	Width, Depth measure.Length
}

// CityHighway has separated elevated carriageways and ground connections.
// Width includes the median gap. Clearance reserves land beside its decks.
type CityHighway struct {
	From, To                            maths.Vec
	Width, Median, Clearance, Elevation measure.Length
}

// CityRamp is a cubic elevated-to-ground connection to a local grid street.
// Controls are physical metre positions; Width is its carriageway width.
type CityRamp struct {
	Points [4]maths.Vec
	Width  measure.Length
}

// ClusteredCitySpec conditions a cluster field on streets, lots, parks and
// highways. Count is exact; this weighted fixed-count realization is not the
// unconstrained Thomas/Neyman–Scott offspring process returned by its sampler.
type ClusteredCitySpec struct {
	Grid     CityGrid
	Count    int
	Process  NeymanScottSpec
	Parks    []CityPark
	Highways []CityHighway
	Trees    *CityTreeSpec
	// ParkRelief adds shaped, planted terrain within park reservations.
	ParkRelief bool
	// ParkPathWidth optionally selects a path width in factory metres.
	// Zero retains the established four-metre path.
	ParkPathWidth measure.Length
	// VacantLotIndex selects cached vacant-lot records or the historical
	// full-grid rescan used when constructing the vacant landscape. Empty uses
	// the cached default. An explicit mode requires Trees.
	VacantLotIndex CityVacantLotIndexMode
	Night          bool
}

// DefaultClusteredCity reserves large parks and a diagonal freeway while
// leaving room for exactly ten thousand clustered, individually seeded lots.
func DefaultClusteredCity() ClusteredCitySpec {
	grid := DefaultCityGrid()
	grid.BlocksX, grid.BlocksZ = int(districtValue("blocks-across")), int(districtValue("blocks-across"))
	extent := grid.BlockLength.Times(float64(grid.BlocksX))
	return ClusteredCitySpec{Grid: grid, Count: int(districtValue("building-count")),
		Process: NeymanScottSpec{Pattern: ThomasGaussian, Width: extent, Depth: extent,
			ParentSpacing: measure.Metres(districtValue("parent-spacing")), Spread: measure.Metres(districtValue("cluster-spread")), MeanOffspring: districtValue("mean-offspring")},
		Parks: []CityPark{
			{Name: "long central park", At: maths.V(-900, 0, 200), Width: measure.Metres(districtValue("long-park-width")), Depth: measure.Metres(districtValue("long-park-depth"))},
			{Name: "western landscape park", At: maths.V(850, 0, -1500), Width: measure.Metres(districtValue("wide-park-width")), Depth: measure.Metres(districtValue("wide-park-depth"))},
		}, Highways: []CityHighway{{From: maths.V(-2600, 0, -2200), To: maths.V(2600, 0, 800),
			Width: measure.Metres(districtValue("highway-width")), Median: measure.Metres(districtValue("highway-median")), Clearance: measure.Metres(districtValue("highway-clearance")), Elevation: measure.Metres(districtValue("highway-elevation"))}},
	}
}

// ClusteredCity returns a checked recipe without altering the grid default.
func ClusteredCity(spec ClusteredCitySpec) (Kind, error) {
	if err := spec.check(); err != nil {
		return kind{}, err
	}
	if spec.Trees != nil {
		copy := *spec.Trees
		spec.Trees = &copy
	}
	spec.Parks = slices.Clone(spec.Parks)
	spec.Highways = slices.Clone(spec.Highways)
	return kind{name: "city-clustered-10000", how: clusteredCityMaker{spec}}, nil
}

// ClusteredCityLayout owns selected lots and reserved features for inspection.
type ClusteredCityLayout struct {
	base          CityLayout
	vacant        []cityVacantLot
	vacantIndexed bool
	parks         []CityPark
	parkRelief    bool
	parkPathWidth measure.Length
	highways      []CityHighway
	ramps         []CityRamp
	parents       []ClusterPoint
	trees         []CityTree
}

type cityVacantLot struct {
	seed uint64
	lot  maths.Box
}

// Buildings returns exactly the admitted population in stable seed order.
func (city ClusteredCityLayout) Buildings() []CityBuilding { return city.base.Buildings() }

// Parks returns owned landscape reservations.
func (city ClusteredCityLayout) Parks() []CityPark { return slices.Clone(city.parks) }

// Highways returns owned elevated highway corridors.
func (city ClusteredCityLayout) Highways() []CityHighway { return slices.Clone(city.highways) }

// Ramps returns actual highway-to-local-street connections.
func (city ClusteredCityLayout) Ramps() []CityRamp { return slices.Clone(city.ramps) }

// Parents returns the Poisson centres used by the conditioned district field.
func (city ClusteredCityLayout) Parents() []ClusterPoint { return slices.Clone(city.parents) }

// check validates feature dimensions before sampling or geometry allocation.
func (spec ClusteredCitySpec) check() error {
	if width := spec.ParkPathWidth.InMetres(); width != 0 && (!positive(width) || spec.ParkRelief && width > 4) {
		return fmt.Errorf("manufacture: invalid park path width")
	}
	if spec.VacantLotIndex != "" && spec.VacantLotIndex != CityVacantLotIndexOn && spec.VacantLotIndex != CityVacantLotIndexOff {
		return fmt.Errorf("manufacture: unknown vacant lot index mode %q", spec.VacantLotIndex)
	}
	if spec.VacantLotIndex != "" && spec.Trees == nil {
		return fmt.Errorf("manufacture: vacant lot index mode requires clustered city trees")
	}
	if spec.Trees != nil {
		if err := spec.Trees.check(); err != nil {
			return err
		}
	}
	if err := spec.Grid.check(); err != nil {
		return err
	}
	if err := spec.Process.check(); err != nil {
		return err
	}
	if spec.Count < 1 || spec.Count > spec.Grid.BlocksX*spec.Grid.BlocksZ*spec.Grid.LotsAcross*spec.Grid.LotsAcross {
		return fmt.Errorf("manufacture: clustered city count exceeds grid capacity")
	}
	if spec.Process.Width != spec.Grid.BlockLength.Times(float64(spec.Grid.BlocksX)) || spec.Process.Depth != spec.Grid.BlockLength.Times(float64(spec.Grid.BlocksZ)) {
		return fmt.Errorf("manufacture: cluster parent window must match city extent")
	}
	for _, park := range spec.Parks {
		if !positive(park.Width.InMetres()) || !positive(park.Depth.InMetres()) || park.Width.InMetres() < 32 || park.Depth.InMetres() < 32 || !cityPoint(park.At) {
			return fmt.Errorf("manufacture: invalid city park")
		}
	}
	for _, road := range spec.Highways {
		if !cityPoint(road.From) || !cityPoint(road.To) || !positive(road.To.Sub(road.From).Len()) || !positive(road.Width.InMetres()) || !positive(road.Median.InMetres()) || road.Median.InMetres() >= road.Width.InMetres() || !positive(road.Clearance.InMetres()) || !positive(road.Elevation.InMetres()) || road.Elevation.InMetres() < 4 || road.Elevation.InMetres() > 50 {
			return fmt.Errorf("manufacture: invalid elevated highway corridor")
		}
	}
	return nil
}

// cityPoint requires a finite ground position in the city's metre frame.
func cityPoint(at maths.Vec) bool {
	return !math.IsNaN(at.Len()) && !math.IsInf(at.Len(), 0) && at.Y == 0
}

// PlanClusteredCity selects lots without replacement under the cluster field.
// Exponential races provide weighted selection; a small uniform floor permits
// exact requested populations even in sparse tails or an empty parent draw.
func PlanClusteredCity(spec ClusteredCitySpec, from seed) (ClusteredCityLayout, error) {
	if err := spec.check(); err != nil {
		return ClusteredCityLayout{}, err
	}
	base, err := PlanCity(spec.Grid, from)
	if err != nil {
		return ClusteredCityLayout{}, err
	}
	process, err := SampleNeymanScott(spec.Process, from)
	if err != nil {
		return ClusteredCityLayout{}, err
	}
	city := ClusteredCityLayout{base: base, parks: slices.Clone(spec.Parks), parkRelief: spec.ParkRelief, parkPathWidth: spec.ParkPathWidth, highways: slices.Clone(spec.Highways), parents: process.Parents(),
		vacantIndexed: spec.VacantLotIndex != CityVacantLotIndexOff}
	for _, road := range city.highways {
		city.ramps = append(city.ramps, city.highwayRamps(road)...)
	}
	type candidate struct {
		building      CityBuilding
		key, distance float64
	}
	var available []candidate
	spread := spec.Process.Spread.InMetres()
	for _, building := range base.buildings {
		if city.reserved(building.Lot, spec.Grid.Setback.InMetres()) {
			continue
		}
		weight, nearest := .001, math.Inf(1)
		for _, parent := range city.parents {
			distance := building.Place.Origin.Sub(parent.At).Len()
			nearest = math.Min(nearest, distance)
			if spec.Process.Pattern == ThomasGaussian {
				weight += math.Exp(-.5 * distance * distance / (spread * spread))
			} else if distance < spread {
				weight++
			}
		}
		own := sow(building.Seed)
		key := -math.Log(math.Max(own.draw(80, 0, 0), 1e-15)) / weight
		available = append(available, candidate{building, key, nearest})
	}
	if len(available) < spec.Count {
		return ClusteredCityLayout{}, fmt.Errorf("manufacture: parks/highways leave %d lots for %d buildings", len(available), spec.Count)
	}
	sort.Slice(available, func(i, j int) bool {
		if available[i].key == available[j].key {
			return available[i].building.Seed < available[j].building.Seed
		}
		return available[i].key < available[j].key
	})
	available = available[:spec.Count]
	sort.Slice(available, func(i, j int) bool { return available[i].building.Seed < available[j].building.Seed })
	if spec.Trees != nil && city.vacantIndexed {
		occupied := make(map[uint64]bool, len(available))
		for _, one := range available {
			occupied[one.building.Seed] = true
		}
		for _, building := range base.buildings {
			if !occupied[building.Seed] && !city.reserved(building.Lot, 0) {
				city.vacant = append(city.vacant, cityVacantLot{
					seed: building.Seed, lot: building.Lot})
			}
		}
	}
	city.base.buildings = make([]CityBuilding, 0, spec.Count)
	for _, one := range available {
		building := one.building
		if one.distance < spread*.6 && sow(building.Seed).draw(81, 0, 0) > .55 {
			building.Kind = "city-tower"
		} else if one.distance < spread*1.2 {
			building.Kind = "city-apartment"
		} else {
			building.Kind = "city-rowhouse"
		}
		building.Bounds = transformed(cityBuildingDesign(building.Kind, sow(building.Seed)).envelope(), building.Place)
		if spec.Night {
			building.Bounds.Min.X -= .02
			building.Bounds.Min.Z -= .02
			building.Bounds.Max.X += .02
			building.Bounds.Max.Z += .02
		}
		building.Lot.Max.Y = building.Bounds.Max.Y
		city.base.buildings = append(city.base.buildings, building)
	}
	if spec.Trees != nil {
		if err := city.plantTrees(*spec.Trees, from); err != nil {
			return ClusteredCityLayout{}, err
		}
	}
	return city, nil
}

// reserved excludes whole lots for uninterrupted landscapes and road margins.
func (city ClusteredCityLayout) reserved(box maths.Box, clearance float64) bool {
	for _, park := range city.parks {
		halfW, halfD := park.Width.InMetres()/2+clearance, park.Depth.InMetres()/2+clearance
		if box.Min.X < park.At.X+halfW && box.Max.X > park.At.X-halfW && box.Min.Z < park.At.Z+halfD && box.Max.Z > park.At.Z-halfD {
			return true
		}
	}
	for _, road := range city.highways {
		if cityCorridor(box, road.From, road.To, road.Width.InMetres()/2+road.Clearance.InMetres()) {
			return true
		}
	}
	for _, ramp := range city.ramps {
		previous := ramp.at(0)
		for step := 1; step <= 12; step++ {
			next := ramp.at(float64(step) / 12)
			if cityCorridor(box, previous, next, ramp.Width.InMetres()/2+clearance) {
				return true
			}
			previous = next
		}
	}
	return false
}

// cityCorridor conservatively intersects an AABB with an oriented corridor.
func cityCorridor(box maths.Box, from, to maths.Vec, half float64) bool {
	from.Y, to.Y = 0, 0
	run := to.Sub(from)
	length := run.Len()
	direction := run.Scale(1 / length)
	normal := maths.V(-direction.Z, 0, direction.X)
	centre := box.Min.Add(box.Max).Scale(.5)
	centre.Y = 0
	extent := box.Max.Sub(box.Min).Scale(.5)
	offset := centre.Sub(from)
	across := math.Abs(normal.X)*extent.X + math.Abs(normal.Z)*extent.Z
	along := math.Abs(direction.X)*extent.X + math.Abs(direction.Z)*extent.Z
	return math.Abs(offset.Dot(normal)) <= half+across && offset.Dot(direction) >= -half-along && offset.Dot(direction) <= length+half+along
}
