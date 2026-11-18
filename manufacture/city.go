package manufacture

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// CityGrid describes authored, flat, metre-scale scenery, not a traffic or
// planning simulation. Blocks are bounded by streets and crossed by alleys.
// Two or four lots across each block leave every building an alley or street
// frontage. Setback is reserved beside streets, alleys and traffic circles.
type CityGrid struct {
	BlocksX, BlocksZ, LotsAcross       int
	BlockLength, RoadWidth, AlleyWidth measure.Length
	Setback, RoundaboutRadius          measure.Length
	RoundaboutEvery                    int
	// BatchPlacement selects the exact-equivalent material-batch append path.
	// The empty value selects CityBatchFused. This is a Go API setting only;
	// set and job YAML do not currently expose it.
	BatchPlacement CityBatchPlacementMode
}

// CityBatchPlacementMode selects how city parts are placed into material
// batches. Both modes preserve geometry and failure behavior exactly.
type CityBatchPlacementMode string

// CityVacantLotIndexMode selects the clustered city's vacant-lot enumeration.
type CityVacantLotIndexMode string

const (
	// CityBatchFused applies part and city transforms while appending faces,
	// avoiding temporary placed surfaces. It is the default.
	CityBatchFused CityBatchPlacementMode = "fused"
	// CityBatchNested uses the historical nested surface-placement path.
	CityBatchNested CityBatchPlacementMode = "nested"
	// CityVacantLotIndexOn caches vacant lot descriptors during planning.
	CityVacantLotIndexOn CityVacantLotIndexMode = "on"
	// CityVacantLotIndexOff recomputes the historical full-grid rescan.
	CityVacantLotIndexOff CityVacantLotIndexMode = "off"
)

// WithCityVacantLotIndex copies a tree-bearing clustered city kind with an
// explicit vacant-lot enumeration mode. Other kinds refuse the setting.
func WithCityVacantLotIndex(row Kind, mode CityVacantLotIndexMode) (Kind, error) {
	if mode != "" && mode != CityVacantLotIndexOn && mode != CityVacantLotIndexOff {
		return Kind{}, fmt.Errorf("manufacture: unknown vacant lot index mode %q", mode)
	}
	how, ok := row.how.(clusteredCityMaker)
	if !ok || how.spec.Trees == nil {
		return Kind{}, fmt.Errorf("manufacture: %s does not support vacant lot indexing", row.name)
	}
	how.spec.VacantLotIndex = mode
	return Kind{name: row.name, how: how}, nil
}

// WithCityBatchPlacement copies a named city kind with an explicit batch
// placement mode. Other kinds refuse the setting rather than ignoring it.
func WithCityBatchPlacement(row Kind, mode CityBatchPlacementMode) (Kind, error) {
	if mode != "" && mode != CityBatchFused && mode != CityBatchNested {
		return Kind{}, fmt.Errorf("manufacture: unknown city batch placement mode %q", mode)
	}
	switch how := row.how.(type) {
	case cityMaker:
		how.grid.BatchPlacement = mode
		return Kind{name: row.name, how: how}, nil
	case clusteredCityMaker:
		how.spec.Grid.BatchPlacement = mode
		return Kind{name: row.name, how: how}, nil
	default:
		return Kind{}, fmt.Errorf("manufacture: %s does not support city batch placement", row.name)
	}
}

// DefaultCityGrid has exactly 10,000 buildings: 25 by 25 blocks, each with
// sixteen lots. Dimensions, districts and junction spacing are authored
// visual choices, not measured city data or road engineering standards.
func DefaultCityGrid() CityGrid {
	return CityGrid{
		BlocksX:          int(cityValue("blocks-across")),
		BlocksZ:          int(cityValue("blocks-across")),
		LotsAcross:       int(cityValue("lots-across")),
		BlockLength:      measure.Metres(cityValue("block-length")),
		RoadWidth:        measure.Metres(cityValue("road-width")),
		AlleyWidth:       measure.Metres(cityValue("alley-width")),
		Setback:          measure.Metres(cityValue("setback")),
		RoundaboutRadius: measure.Metres(cityValue("roundabout-radius")),
		RoundaboutEvery:  int(cityValue("junction-interval")),
	}
}

// City makes a reusable factory recipe from checked layout dimensions.
// Make and MakeWithComplexity use the ordinary specimen/material pipeline.
func City(grid CityGrid) (Kind, error) {
	if err := grid.check(); err != nil {
		return kind{}, err
	}
	return kind{name: "city-grid", how: cityMaker{grid}}, nil
}

// CityBuilding records one independently seeded lot and its actual envelope.
// Seed can be passed to Sow and Kind to KindNamed to recreate the specimen;
// Place transforms that specimen into city coordinates. Lot excludes roads
// and their reserved setback, and bounds include every architectural part.
type CityBuilding struct {
	Seed        uint64
	Kind        string
	Place       maths.Subspace
	Lot, Bounds maths.Box
}

// CityRoundabout is a junction's centre and outside carriageway radius.
// Its raised island occupies the inner half of the radius.
type CityRoundabout struct {
	At     maths.Vec
	Radius measure.Length
}

// CityLayout retains a checked plan without allocating building geometry.
// Its accessors return owned slices; a city's seed and lot addresses are
// independent of traversal order and extending the grid's far edges.
type CityLayout struct {
	grid      CityGrid
	buildings []CityBuilding
	circles   []CityRoundabout
}

// Buildings returns the individual, reproducible placements in lot order.
func (city CityLayout) Buildings() []CityBuilding { return slices.Clone(city.buildings) }

// Roundabouts returns the traffic circles on interior street junctions.
func (city CityLayout) Roundabouts() []CityRoundabout { return slices.Clone(city.circles) }

// check bounds population and rejects dimensions before allocating a plan.
func (grid CityGrid) check() error {
	if grid.BatchPlacement != "" && grid.BatchPlacement != CityBatchFused &&
		grid.BatchPlacement != CityBatchNested {
		return fmt.Errorf("manufacture: unknown city batch placement mode %q", grid.BatchPlacement)
	}
	if grid.BlocksX < 1 || grid.BlocksZ < 1 ||
		grid.BlocksX > 25_000 || grid.BlocksZ > 25_000 ||
		(grid.LotsAcross != 2 && grid.LotsAcross != 4) ||
		int64(grid.BlocksX)*int64(grid.BlocksZ)*
			int64(grid.LotsAcross*grid.LotsAcross) >
			int64(cityValue("population-limit")) {
		return fmt.Errorf("manufacture: city needs 2 or 4 lots across and 1 to 100000 buildings")
	}
	for _, length := range []measure.Length{grid.BlockLength, grid.RoadWidth,
		grid.AlleyWidth, grid.Setback, grid.RoundaboutRadius} {
		if !positive(length.InMetres()) {
			return fmt.Errorf("manufacture: city dimensions must be finite and positive")
		}
	}
	if grid.RoundaboutEvery < 0 || grid.RoundaboutEvery == 1 ||
		grid.RoundaboutRadius.InMetres() <= grid.RoadWidth.InMetres()/math.Sqrt2 {
		return fmt.Errorf("manufacture: roundabouts need separated junctions " +
			"and radius beyond the road corners")
	}
	span := grid.BlockLength.InMetres() *
		float64(max(grid.BlocksX, grid.BlocksZ))
	if !positive(span) || grid.lotWidth() < cityValue("minimum-lot") {
		return fmt.Errorf("manufacture: city streets, circles and setbacks " +
			"leave lots narrower than 20 metres")
	}
	return nil
}

// margin conservatively reserves a traffic circle's diagonal corners in
// every block, so an extension need not move previously addressed buildings.
func (grid CityGrid) margin() float64 {
	return math.Max(grid.RoadWidth.InMetres()/2,
		grid.RoundaboutRadius.InMetres()/math.Sqrt2) + grid.Setback.InMetres()
}

// lotWidth divides the buildable area around the central alley cross.
func (grid CityGrid) lotWidth() float64 {
	return (grid.BlockLength.InMetres() - 2*grid.margin() -
		grid.AlleyWidth.InMetres() - 2*grid.Setback.InMetres()) / float64(grid.LotsAcross)
}

// circle identifies interior grid junctions, never adjacent circles.
func (grid CityGrid) circle(x, z int) bool {
	return grid.RoundaboutEvery > 1 && x > 0 && z > 0 &&
		x < grid.BlocksX && z < grid.BlocksZ &&
		x%grid.RoundaboutEvery == 0 && z%grid.RoundaboutEvery == 0
}

// PlanCity constructs actual placements and envelopes without tessellating.
func PlanCity(grid CityGrid, from seed) (CityLayout, error) {
	if err := grid.check(); err != nil {
		return CityLayout{}, err
	}
	if !from.made() {
		return CityLayout{}, errGood
	}
	city := CityLayout{grid: grid}
	pitch := grid.BlockLength.InMetres()
	startX, startZ := -float64(grid.BlocksX)*pitch/2, -float64(grid.BlocksZ)*pitch/2
	for z := 0; z <= grid.BlocksZ; z++ {
		for x := 0; x <= grid.BlocksX; x++ {
			if grid.circle(x, z) {
				city.circles = append(city.circles, CityRoundabout{
					At:     maths.V(startX+float64(x)*pitch, 0, startZ+float64(z)*pitch),
					Radius: grid.RoundaboutRadius})
			}
		}
	}
	count := grid.BlocksX * grid.BlocksZ * grid.LotsAcross * grid.LotsAcross
	city.buildings = make([]CityBuilding, 0, count)
	for z := range grid.BlocksZ {
		for x := range grid.BlocksX {
			for lot := range grid.LotsAcross * grid.LotsAcross {
				city.buildings = append(city.buildings, grid.building(x, z, lot, startX, startZ, from))
			}
		}
	}
	return city, nil
}

// building gives each lot a stable address, family, dimensions and frontage.
func (grid CityGrid) building(x, z, lot int, startX, startZ float64, from seed) CityBuilding {
	address := uint64(z)<<32 | uint64(x)<<8 | uint64(lot)
	// SplitMix's invertible permutation keeps distinct lot addresses distinct.
	number := citySeed(from.number ^ address)
	own := sow(number)
	family := "city-rowhouse"
	if own.draw(0, 0, 0) > 0.65 || x%5 == 0 || z%5 == 0 {
		family = "city-apartment"
	}
	if x%5 == 2 && z%5 == 2 && own.draw(0, 0, 1) > 0.4 {
		family = "city-tower"
	}
	wide := grid.lotWidth()
	along := func(block, index int, start float64) float64 {
		value := start + float64(block)*grid.BlockLength.InMetres() + grid.margin() + float64(index)*wide
		if index >= grid.LotsAcross/2 {
			value += grid.AlleyWidth.InMetres() + 2*grid.Setback.InMetres()
		}
		return value
	}
	col, row := lot%grid.LotsAcross, lot/grid.LotsAcross
	lowX, lowZ := along(x, col, startX), along(z, row, startZ)
	at := maths.V(lowX+wide/2, 0, lowZ+wide/2)
	// Fronts look toward the nearest outer street or central alley.
	turn := 0.0
	if row == 0 || row == grid.LotsAcross/2 {
		turn = math.Pi
	}
	place := maths.NewSubspace(at, maths.About(maths.YAxis, turn), 1)
	design := cityBuildingDesign(family, own)
	bounds := transformed(design.envelope(), place)
	return CityBuilding{Seed: number, Kind: family, Place: place, Bounds: bounds,
		Lot: maths.Box{Min: maths.V(lowX, 0, lowZ), Max: maths.V(lowX+wide, bounds.Max.Y,
			lowZ+wide)}}
}

// citySeed is a bijection of the complete 64-bit lot seed space.
func citySeed(number uint64) uint64 {
	number += 0x9e3779b97f4a7c15
	number = (number ^ (number >> 30)) * 0xbf58476d1ce4e5b9
	number = (number ^ (number >> 27)) * 0x94d049bb133111eb
	return number ^ (number >> 31)
}

// cityRows exposes standalone architectural families and the full city.
func cityRows() map[string]maker {
	return map[string]maker{"city-rowhouse": cityFamily{"city-rowhouse"},
		"city-apartment": cityFamily{"city-apartment"}, "city-tower": cityFamily{"city-tower"},
		"city-grid-10000": cityMaker{DefaultCityGrid()}}
}

type cityMaker struct{ grid CityGrid }

// made bakes this city at its default architectural representation.
func (row cityMaker) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// madeAt keeps the same lot, envelope and seed at both supported levels.
func (row cityMaker) madeAt(from seed, detail string) (good, error) {
	if !cityLevel(detail) {
		return good{}, fmt.Errorf("manufacture: city has no %s representation", detail)
	}
	layout, err := PlanCity(row.grid, from)
	if err != nil {
		return good{}, err
	}
	batch := newCityBatch()
	batch.placement = row.grid.BatchPlacement
	for _, building := range layout.buildings {
		batch.addBuilding(building, detail, false)
	}
	layout.paving(batch)
	return batch.finish()
}
