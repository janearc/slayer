package manufacture

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// CityTreeSpec uses an existing native plant constructor. Complexity selects
// actual native foliage detail; placement always uses its original physical
// hull bounds. Four shared prototypes retain foliage-cell mesh sharing.
type CityTreeSpec struct {
	Kind                         string
	Complexity                   string
	Height, Spacing, ParkSpacing measure.Length
	Prototypes                   int
	// ParkEnrichment adds seeded groves without removing any existing plant.
	ParkEnrichment bool
	// UrbanRows adds physically spaced avenue plantings after the grove.
	UrbanRows bool
	// SizeScale changes trees after their established locations are selected.
	// Zero means one; an orbital magnified 250 times uses 1/250 here to keep
	// ordinary-sized trees while preserving the original city placement.
	SizeScale float64
}

// DefaultCityTrees supplies native valley oaks for rows and park groves.
func DefaultCityTrees() CityTreeSpec {
	return CityTreeSpec{
		Kind: "valley-oak", Complexity: "quick",
		Height:      measure.Metres(districtValue("tree-height")),
		Spacing:     measure.Metres(districtValue("tree-spacing")),
		ParkSpacing: measure.Metres(districtValue("park-tree-spacing")),
		Prototypes:  int(districtValue("tree-prototypes")),
	}
}

// CityTree records planting identity separately from its shared geometry seed.
type CityTree struct {
	Kind                string
	Seed, PrototypeSeed uint64
	Place               maths.Subspace
	Bounds              maths.Box
}

// Trees returns owned native tree placements for inspection or scene reuse.
func (city ClusteredCityLayout) Trees() []CityTree { return slices.Clone(city.trees) }

// check admits bounded, supported tree rows before generating the city.
func (spec CityTreeSpec) check() error {
	detail := spec.complexity()
	if detail != "hull" && detail != "clumps" && detail != "quick" && detail != "demo" {
		return fmt.Errorf("manufacture: unsupported city foliage complexity %q", detail)
	}
	if spec.Prototypes < 1 || spec.Prototypes > 16 || (!positive(spec.Height.InMetres()) || !positive(spec.Spacing.InMetres()) || !positive(spec.ParkSpacing.InMetres())) || spec.Height.InMetres() > 30 || spec.Spacing.InMetres() < 24 || spec.ParkSpacing.InMetres() < 24 || spec.Spacing.InMetres() > 256 || spec.ParkSpacing.InMetres() > 256 {
		return fmt.Errorf("manufacture: invalid native city tree dimensions/budget")
	}
	if math.IsNaN(spec.SizeScale) || math.IsInf(spec.SizeScale, 0) || spec.SizeScale < 0 || spec.SizeScale > 1 {
		return fmt.Errorf("manufacture: city tree size scale must be zero or in (0,1]")
	}
	if spec.UrbanRows && !spec.ParkEnrichment {
		return fmt.Errorf("manufacture: urban rows require park enrichment")
	}
	row, err := KindNamed(spec.Kind)
	if err != nil {
		return err
	}
	if _, known := row.how.(seasonalPlant); !known {
		return fmt.Errorf("manufacture: city trees need an existing seasonal plant recipe")
	}
	return nil
}

type cityTreePrototype struct {
	seed uint64
	kind string
	made good
}

// treePrototypes retains the original physical placement specimens.
func treePrototypes(spec CityTreeSpec, from seed) ([]cityTreePrototype, error) {
	return treePrototypesAt(spec, from, "hull")
}

// treePrototypesAt changes foliage geometry independently of planting.
func treePrototypesAt(spec CityTreeSpec, from seed, detail string) ([]cityTreePrototype, error) {
	row, err := KindNamed(spec.Kind)
	if err != nil {
		return nil, err
	}
	var prototypes []cityTreePrototype
	for i := range spec.Prototypes {
		own := citySeed(from.number ^ uint64(i) ^ 0x54524545)
		made, err := MakeWithComplexity(row, Sow(own), detail)
		if err != nil {
			return nil, err
		}
		prototypes = append(prototypes, cityTreePrototype{own, spec.Kind, made})
	}
	if spec.ParkEnrichment {
		for family, name := range []string{"alder", "poplar", "joshua-tree"} {
			row, err := KindNamed(name)
			if err != nil {
				return nil, err
			}
			for i := range 2 {
				own := citySeed(from.number ^ 0x4558545241545245 ^ uint64(family+1)*100003 ^ uint64(i+1)*700001)
				made, err := MakeWithComplexity(row, Sow(own), detail)
				if err != nil {
					return nil, err
				}
				prototypes = append(prototypes, cityTreePrototype{own, name, made})
			}
		}
	}
	return prototypes, nil
}

// plantTrees scatters groves inside parks and regular rows beside streets.
// Spatial buckets keep tree/building clearance proportional to city size.
func (city *ClusteredCityLayout) plantTrees(spec CityTreeSpec, from seed) error {
	prototypes, err := treePrototypes(spec, from)
	if err != nil {
		return err
	}
	buckets := map[[2]int][]maths.Box{}
	for _, building := range city.base.buildings {
		box := building.Bounds
		for x := int(math.Floor(box.Min.X / 64)); x <= int(math.Floor(box.Max.X/64)); x++ {
			for z := int(math.Floor(box.Min.Z / 64)); z <= int(math.Floor(box.Max.Z/64)); z++ {
				buckets[[2]int{x, z}] = append(buckets[[2]int{x, z}], box)
			}
		}
	}
	address := uint64(0)
	var treeBuckets map[[2]int][]maths.Box
	plant := func(at maths.Vec, park *CityPark, enriched bool) {
		address++
		if park != nil && city.parkRelief {
			at.Y = parkTerrainHeight(*park, at)
		} else if park != nil && enriched {
			at.Y *= spec.sizeScale()
		}
		own := sow(citySeed(from.number ^ address ^ 0x504c414e54))
		pool := prototypes[:spec.Prototypes]
		height := spec.Height.InMetres()
		if enriched {
			pool = prototypes[spec.Prototypes:]
			height = math.Max(height, 24) * spec.sizeScale()
		}
		prototype := pool[int(own.draw(90, 0, 0)*float64(len(pool)))]
		scale := height / prototype.made.Bounds().Max.Y * (.85 + .3*own.draw(90, 0, 1))
		place := maths.NewSubspace(at, maths.About(maths.YAxis, 2*math.Pi*own.draw(90, 0, 2)), scale)
		box := transformed(prototype.made.Bounds(), place)
		reservations := *city
		if park != nil {
			reservations.parks = nil
		}
		if reservations.reserved(box, 1) {
			return
		}
		if park == nil {
			for _, circle := range city.base.circles {
				if at.Sub(circle.At).Len() < circle.Radius.InMetres()+10 {
					return
				}
			}
		}
		if enriched {
			radius := math.Max(math.Max(math.Abs(box.Min.X-at.X), math.Abs(box.Max.X-at.X)), math.Max(math.Abs(box.Min.Z-at.Z), math.Abs(box.Max.Z-at.Z)))
			if parkPathDistance(*park, at) < radius+spec.sizeScale() {
				return
			}
			if spec.UrbanRows && math.Abs(at.X-park.At.X) > 1000*spec.sizeScale() {
				// Avenues flank the neighbourhood at ±1,045 m; account for
				// each crown, not just the centre of its trunk.
				for _, side := range []float64{-1, 1} {
					x := park.At.X + side*1045*spec.sizeScale()
					width := 2 * spec.sizeScale()
					if box.Min.X < x+width && box.Max.X > x-width && box.Min.Z < park.At.Z+750*spec.sizeScale() && box.Max.Z > park.At.Z-750*spec.sizeScale() {
						return
					}
				}
			}
			// Keep the authored camera passage clear, including whole tree
			// bounds. The short segment follows the park's long path, at
			// u=0.5..0.5003; its 25 m margin is deliberately conservative.
			clear := 25 * spec.sizeScale()
			endX := park.At.X + park.Width.InMetres()*.15*math.Sin(2*math.Pi*.5003)
			endZ := park.At.Z + park.Depth.InMetres()*.84*.0003
			if box.Min.X < math.Max(park.At.X, endX)+clear && box.Max.X > math.Min(park.At.X, endX)-clear && box.Min.Z < math.Max(park.At.Z, endZ)+clear && box.Max.Z > math.Min(park.At.Z, endZ)-clear {
				return
			}
		}
		for x := int(math.Floor(box.Min.X / 64)); x <= int(math.Floor(box.Max.X/64)); x++ {
			for z := int(math.Floor(box.Min.Z / 64)); z <= int(math.Floor(box.Max.Z/64)); z++ {
				for _, building := range buckets[[2]int{x, z}] {
					if box.Min.X < building.Max.X && box.Max.X > building.Min.X && box.Min.Z < building.Max.Z && box.Max.Z > building.Min.Z {
						return
					}
				}
				if enriched {
					for _, previous := range treeBuckets[[2]int{x, z}] {
						if box.Min.X < previous.Max.X && box.Max.X > previous.Min.X && box.Min.Z < previous.Max.Z && box.Max.Z > previous.Min.Z {
							return
						}
					}
				}
			}
		}
		city.trees = append(city.trees, CityTree{Kind: prototype.kind, Seed: own.number, PrototypeSeed: prototype.seed, Place: place, Bounds: box})
		if enriched {
			for x := int(math.Floor(box.Min.X / 64)); x <= int(math.Floor(box.Max.X/64)); x++ {
				for z := int(math.Floor(box.Min.Z / 64)); z <= int(math.Floor(box.Max.Z/64)); z++ {
					treeBuckets[[2]int{x, z}] = append(treeBuckets[[2]int{x, z}], box)
				}
			}
		}
	}
	for _, park := range city.parks {
		spacing := spec.ParkSpacing.InMetres()
		for x := park.At.X - park.Width.InMetres()/2 + 16; x < park.At.X+park.Width.InMetres()/2-16; x += spacing {
			for z := park.At.Z - park.Depth.InMetres()/2 + 16; z < park.At.Z+park.Depth.InMetres()/2-16; z += spacing {
				own := sow(citySeed(from.number ^ uint64(address) ^ 0x5041524b))
				at := maths.V(x+own.swing(91, 0, 0)*spacing*.2, .06, z+own.swing(91, 0, 1)*spacing*.2)
				if parkPathDistance(park, at) < 5 || own.draw(92, 0, 0) > parkGroveWeight(park, at, from) {
					continue
				}
				plant(at, &park, false)
			}
		}
	}
	grid := city.base.grid
	pitch := grid.BlockLength.InMetres()
	startX, startZ := -float64(grid.BlocksX)*pitch/2, -float64(grid.BlocksZ)*pitch/2
	for line := 0; line <= grid.BlocksZ; line++ {
		for _, side := range []float64{-1, 1} {
			z := startZ + float64(line)*pitch + side*(grid.RoadWidth.InMetres()/2+5.5)
			for x := startX + spec.Spacing.InMetres()/2; x < -startX; x += spec.Spacing.InMetres() {
				y := -.02 * spec.sizeScale()
				plant(maths.V(x, y, z), nil, false)
			}
		}
	}
	// Ordinary-sized trees stay at the original city locations. Scaling
	// after selection keeps even previously admitted collision decisions.
	scale := spec.sizeScale()
	if scale != 1 {
		for i := range city.trees {
			tree := &city.trees[i]
			origin := tree.Place.Origin
			tree.Place.Scale *= scale
			tree.Bounds.Min = origin.Add(tree.Bounds.Min.Sub(origin).Scale(scale))
			tree.Bounds.Max = origin.Add(tree.Bounds.Max.Sub(origin).Scale(scale))
		}
	}
	if spec.ParkEnrichment {
		treeBuckets = make(map[[2]int][]maths.Box)
		for _, tree := range city.trees {
			box := tree.Bounds
			for x := int(math.Floor(box.Min.X / 64)); x <= int(math.Floor(box.Max.X/64)); x++ {
				for z := int(math.Floor(box.Min.Z / 64)); z <= int(math.Floor(box.Max.Z/64)); z++ {
					treeBuckets[[2]int{x, z}] = append(treeBuckets[[2]int{x, z}], box)
				}
			}
		}
		// A compact walkable grove, a few kilometres wide after orbital scaling,
		// supplements rather than replaces the sparse city-wide plantings.
		// Candidate noise depends on its coordinates, even on misses.
		for pi, park := range city.parks {
			if pi > 0 {
				break
			}
			spacing := 25 * scale
			for ix := 0; ix < 80; ix++ {
				x := park.At.X + (float64(ix)-39.5)*spacing
				for iz := 0; iz < 60; iz++ {
					z := park.At.Z + (float64(iz)-29.5)*spacing
					own := sow(citySeed(from.number ^ uint64(pi+1)*100003 ^ uint64(ix+1)*700001 ^ uint64(iz+1)*1100003))
					at := maths.V(x+own.swing(91, 0, 0)*spacing*.2, .06, z+own.swing(91, 0, 1)*spacing*.2)
					if math.Abs(at.X-park.At.X) >= park.Width.InMetres()/2 || math.Abs(at.Z-park.At.Z) >= park.Depth.InMetres()/2 {
						continue
					}
					if parkPathDistance(park, at) < 7*scale || own.draw(92, 0, 0) > .5 {
						continue
					}
					plant(at, &park, true)
				}
			}
			if spec.UrbanRows {
				// Rows flank the reserved central neighbourhood, with 25 m
				// between trunks after city scaling. Building generation runs
				// later and reserves every resulting tree's complete bounds.
				for _, side := range []float64{-1, 1} {
					for index := -30; index <= 30; index++ {
						at := maths.V(park.At.X+side*1062.5*scale, .06,
							park.At.Z+float64(index)*25*scale)
						if math.Abs(at.X-park.At.X) >= park.Width.InMetres()/2 || math.Abs(at.Z-park.At.Z) >= park.Depth.InMetres()/2 {
							continue
						}
						plant(at, &park, true)
					}
				}
			}
		}
	}
	return nil
}

// sizeScale returns the scale of the tree spec, or one when none is set.
func (spec CityTreeSpec) sizeScale() float64 {
	if spec.SizeScale == 0 {
		return 1
	}
	return spec.SizeScale
}

// treeGroups instances bounded prototype meshes by material, never one native
// acceleration structure per tree. Native groups contain only flat meshes.
func (city ClusteredCityLayout) treeGroups(made good, spec CityTreeSpec, from seed) (good, error) {
	prototypes, err := treePrototypesAt(spec, from, spec.complexity())
	if err != nil {
		return good{}, err
	}
	prepared, err := prepareCityFoliage(prototypes, made.dressed)
	if err != nil {
		return good{}, err
	}
	counts := map[slot]int{}
	for _, tree := range city.trees {
		for _, part := range prepared[tree.PrototypeSeed] {
			counts[part.role] += len(part.members)
		}
	}
	members := map[slot][]shape.MeshMember{}
	for role, count := range counts {
		members[role] = make([]shape.MeshMember, 0, count)
	}
	for _, tree := range city.trees {
		for _, part := range prepared[tree.PrototypeSeed] {
			for _, member := range part.members {
				member.Space = member.Space.Within(tree.Place)
				members[part.role] = append(members[part.role], member)
			}
		}
	}
	for role, places := range members {
		group, err := shape.NewMeshGroup(places)
		if err != nil {
			return good{}, err
		}
		made.groups[role] = group
		made.groupParts[role] = len(places)
	}
	return made, nil
}

// complexity leaves an explicit hull available for distant overview callers.
// Empty values select native quick foliage, as the supplied defaults do.
func (spec CityTreeSpec) complexity() string {
	if spec.Complexity == "" {
		return "quick"
	}
	return spec.Complexity
}
