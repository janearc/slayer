package set

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/biome"
	yaml "gopkg.in/yaml.v3"
)

// Ecosystem is an authored mixture over a fixed projected source region.
// Its density is candidates per hectare, before ground and clearance filters.
type Ecosystem struct {
	name, detail, weather        string
	seed, cell, density, variety float64
	west, north, east, south     float64
	mixture                      []biome.Species
	above, below                 float64
	excluded                     string
	clearance                    float64
	fade                         bool
}

// Ecosystems returns immutable region descriptions in declaration order.
func (made Set) Ecosystems() []Ecosystem { return slices.Clone(made.ecosystems) }

// Name identifies the population independently of any query or terrain crop.
func (region Ecosystem) Name() string { return region.name }

// Seed fixes population draws independently of the visible query.
func (region Ecosystem) Seed() uint64 { return uint64(region.seed) }

// Complexity selects a physical representation without changing root identity.
func (region Ecosystem) Complexity() string { return region.detail }

// Variety caps distinct geometry prototypes per component recipe in this
// region.
func (region Ecosystem) Variety() int { return int(region.variety) }

// Band gives source-height habitat limits; these do not simulate plant ecology.
func (region Ecosystem) Band() (float64, float64) { return region.above, region.below }

// Exclude names a placed object's enclosing clearance circle.
func (region Ecosystem) Exclude() (string, float64) { return region.excluded, region.clearance }

// Atmosphere selects shared prescribed mist over the resulting population.
func (region Ecosystem) Atmosphere() (string, bool) { return region.weather, region.fade }

// Population constructs the bounded spatial generator without enumerating
// roots.
func (region Ecosystem) Population() (*biome.Region, error) {
	return biome.New(biome.Definition{Name: region.name, Seed: uint64(region.seed),
		Bounds: biome.Rect{MinEast: region.west, MaxEast: region.east,
			MinSouth: -region.north, MaxSouth: -region.south},
		Cell: measure.Metres(region.cell), PerHectare: region.density, Species: region.mixture})
}

// ecosystems reads each mixed population through the ordinary checked reader.
func (reading reader) ecosystems(_, block *yaml.Node) error {
	if err := reading.each(block, reading.ecosystem); err != nil {
		return err
	}
	if len(block.Content) == 0 {
		return reading.refuse(block, "ecosystems needs at least one region")
	}
	return nil
}

// ecosystem keeps population controls distinct from individual constructions.
func (reading reader) ecosystem(name, block *yaml.Node) error {
	region := Ecosystem{name: name.Value, above: math.Inf(-1), below: math.Inf(1)}
	err := reading.keys(block, map[string]step{
		"seed":       reading.whole(&region.seed, 0, mostSeed),
		"cell":       reading.quantity(&region.cell, "m", math.SmallestNonzeroFloat64, math.MaxFloat64),
		"density":    reading.quantity(&region.density, "ha^-1", 0, math.Inf(1)),
		"variety":    reading.whole(&region.variety, 1, mostStand),
		"complexity": reading.complexity(&region.detail),
		"above":      reading.number(&region.above, math.Inf(-1), math.Inf(1)),
		"below":      reading.number(&region.below, math.Inf(-1), math.Inf(1)),
		"bounds": reading.block(map[string]step{
			"west":  reading.number(&region.west, math.Inf(-1), math.Inf(1)),
			"east":  reading.number(&region.east, math.Inf(-1), math.Inf(1)),
			"north": reading.number(&region.north, math.Inf(-1), math.Inf(1)),
			"south": reading.number(&region.south, math.Inf(-1), math.Inf(1))}),
		"exclude": reading.block(map[string]step{
			"around":    reading.text(&region.excluded),
			"clearance": reading.quantity(&region.clearance, "m", 0, math.Inf(1))}),
		"atmosphere": reading.block(map[string]step{
			"weather":          reading.text(&region.weather),
			"extent":           reading.only("canopy-and-ground"),
			"fade-at-boundary": reading.yes(&region.fade)}),
		"mixture": reading.mixture(&region.mixture),
	}, nil)
	if err != nil {
		return err
	}
	return reading.keepEcosystem(region, block)
}

// mixture reads recipe names with relative positive weights.
func (reading reader) mixture(target *[]biome.Species) step {
	return func(_, block *yaml.Node) error {
		return reading.each(block, func(name, value *yaml.Node) error {
			member := biome.Species{Recipe: name.Value}
			if err := reading.number(&member.Weight, 0, math.Inf(1))(name, value); err != nil {
				return err
			}
			*target = append(*target, member)
			return nil
		})
	}
}

// keepEcosystem requires complete bounds and validates construction-independent
// data.
func (reading reader) keepEcosystem(region Ecosystem, block *yaml.Node) error {
	if err := reading.all(block, "ecosystem", "seed", "cell", "density", "variety", "bounds", "mixture"); err != nil {
		return err
	}
	for _, key := range []string{"bounds", "exclude", "atmosphere"} {
		if err := reading.ecosystemBlock(block, key); err != nil {
			return err
		}
	}
	if region.above >= region.below {
		return reading.refuse(block, "ecosystem habitat height band is empty")
	}
	if _, err := region.Population(); err != nil {
		return reading.refuse(block, "%v", err)
	}
	reading.set.ecosystems = append(reading.set.ecosystems, region)
	return nil
}

// ecosystemBlock checks required subkeys that simple block readers cannot
// infer.
func (reading reader) ecosystemBlock(block *yaml.Node, key string) error {
	for index := 0; index+1 < len(block.Content); index += 2 {
		if block.Content[index].Value != key {
			continue
		}
		value := block.Content[index+1]
		switch key {
		case "bounds":
			return reading.all(value, "ecosystem bounds", "west", "north", "east", "south")
		case "exclude":
			return reading.all(value, "ecosystem exclusion", "around", "clearance")
		case "atmosphere":
			return reading.all(value, "ecosystem atmosphere", "weather", "extent", "fade-at-boundary")
		}
	}
	return nil
}

// foundEcosystems resolves recipes, exclusions and weather after all scene
// blocks.
func (reading reader) foundEcosystems() error {
	names := map[string]bool{}
	for _, stand := range reading.set.stands {
		names[stand.Name()] = true
	}
	for index := range reading.set.ecosystems {
		region := &reading.set.ecosystems[index]
		if names[region.name] {
			return fmt.Errorf("%s: ecosystem name %s is already used by a population", reading.file, region.name)
		}
		names[region.name] = true
		if region.detail == "" {
			region.detail = reading.set.complexity
		}
		if reading.set.terrain == nil {
			return fmt.Errorf("%s: ecosystem %s needs terrain", reading.file, region.name)
		}
		if region.excluded != "" && !reading.set.named(region.excluded) {
			return fmt.Errorf("%s: ecosystem %s names unknown exclusion %s", reading.file, region.name, region.excluded)
		}
		if region.weather != "" {
			if _, found := reading.set.Mist(region.weather); !found {
				return fmt.Errorf("%s: ecosystem %s names unknown mist %s", reading.file, region.name, region.weather)
			}
		}
		for _, member := range region.mixture {
			row, found := reading.set.Recipe(member.Recipe)
			if !found || row.Kind() == "fire" {
				return fmt.Errorf("%s: ecosystem %s needs a solid recipe %s", reading.file, region.name, member.Recipe)
			}
			if _, err := row.Construction(); err != nil {
				return fmt.Errorf("%s: ecosystem %s: %w", reading.file, region.name, err)
			}
		}
	}
	return nil
}
