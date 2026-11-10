package set

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// Recipe is a reusable construction specification, independent of placement.
type Recipe struct {
	burn                                           *manufacture.BurnSpec
	kind                                           string
	design                                         string
	season                                         string
	diameter, aperture, opening, azimuth, altitude float64
	segmented                                      bool
}

// Kind names the construction rule shared by these specimens.
func (recipe Recipe) Kind() string { return recipe.kind }

// Construction selects the checked solid recipe. Volumetric fire uses Fire.
func (recipe Recipe) Construction() (*manufacture.Kind, error) {
	if recipe.kind == "space-observatory" {
		aperture := measure.Metres(recipe.aperture)
		return manufacture.Spacecraft(recipe.design, aperture)
	}
	row, err := manufacture.KindNamed(recipe.kind)
	if recipe.kind == "optical-observatory" {
		row, err = manufacture.Observatory(recipe.Optical())
	}
	if err != nil {
		return nil, err
	}
	if recipe.season != "" {
		row, err = manufacture.WithSeason(row, recipe.season)
		if err != nil {
			return nil, err
		}
	}
	return &row, nil
}

// Optical is the checked optical specification, in physical units.
func (recipe Recipe) Optical() manufacture.Optical {
	return manufacture.Optical{Diameter: measure.Metres(recipe.diameter),
		Aperture: measure.Metres(recipe.aperture), Opening: measure.Degrees(recipe.opening),
		Azimuth: measure.Degrees(recipe.azimuth), Altitude: measure.Degrees(recipe.altitude),
		Segmented: recipe.segmented}
}

// Recipe finds a construction recipe by its YAML name.
func (made Set) Recipe(name string) (Recipe, bool) { r, ok := made.recipes[name]; return r, ok }

// Renderer names cpu, gpu, gpu-native-shadows or gpu-native.
// Empty lets the caller choose.
func (made Set) Renderer() string { return made.renderer }

// Seed is the scene construction seed, zero when unspecified.
func (made Set) Seed() uint64 { return uint64(made.seed) }

// HasSeed distinguishes an explicit zero seed from an unspecified seed.
func (made Set) HasSeed() bool { return made.hasSeed }

// Complexity is the representation inherited by placements.
func (made Set) Complexity() string { return made.complexity }

// Recipe is the referenced recipe, or empty for a direct kind.
func (made Thing) Recipe() string { return made.recipe }

// Complexity is the specimen representation after inheritance.
func (made Thing) Complexity() string { return made.complexity }

// Recipe is the construction repeated in this stand, if any.
func (made Stand) Recipe() string { return made.recipe }

// Complexity is the stand representation after inheritance.
func (made Stand) Complexity() string { return made.complexity }

// Exclude names the object and clearance beyond it, in metres.
func (made Stand) Exclude() (string, float64) { return made.excluded, made.clearance }

// Atmosphere names shared weather and whether its boundary fades.
func (made Stand) Atmosphere() (string, bool) { return made.weather, made.fade }

// complexity accepts representations the manufactory exposes.
func (reading reader) complexity(target *string) step {
	return reading.word(target, manufacture.Complexities()...)
}

// quantity checks both the value and its unit; a missing half is an error.
func (reading reader) quantity(target *float64, unit string, lo, hi float64) step {
	return func(_, node *yaml.Node) error {
		err := reading.keys(node, map[string]step{"value": reading.number(target, lo, hi),
			"unit": reading.only(unit)}, nil)
		if err != nil {
			return err
		}
		return reading.all(node, "a quantity", "value", "unit")
	}
}

// recipes reads construction independently of placement.
func (reading reader) recipes(_, node *yaml.Node) error {
	return reading.each(node, func(name, block *yaml.Node) error {
		if valueOf(block, "kind") == "fire" {
			return reading.fireRecipe(name, block)
		}
		if valueOf(block, "kind") == "space-observatory" {
			return reading.spacecraftRecipe(name, block)
		}
		r := Recipe{}
		err := reading.keys(block, map[string]step{
			"kind":   reading.word(&r.kind, manufacture.KindNames()...),
			"season": reading.word(&r.season, manufacture.SeasonNames()...),
			"enclosure": reading.block(map[string]step{
				"kind":     reading.only("dome"),
				"diameter": reading.quantity(&r.diameter, "m", 1, 200),
				"opening":  reading.quantity(&r.opening, "deg", 5, 90)}),
			"telescope": reading.block(map[string]step{
				"aperture": reading.quantity(&r.aperture, "m", 0.1, 120),
				"mount":    reading.only("alt-azimuth"),
				"primary": func(key, value *yaml.Node) error {
					if err := reading.only("segmented", "monolithic")(key, value); err != nil {
						return err
					}
					r.segmented = value.Value == "segmented"
					return nil
				},
				"pointing": reading.block(map[string]step{
					"azimuth":  reading.number(&r.azimuth, 0, 360),
					"altitude": reading.number(&r.altitude, 0, 90)})}),
		}, nil)
		if err != nil {
			return err
		}
		if r.kind == "" {
			return reading.refuse(block, "recipe %s needs a kind", name.Value)
		}
		if r.kind == "optical-observatory" {
			if _, err := manufacture.Observatory(r.Optical()); err != nil {
				return reading.refuse(block, "%v", err)
			}
		} else if says(block, "enclosure") || says(block, "telescope") {
			return reading.refuse(block, "recipe %s is not an optical observatory", name.Value)
		}
		if r.season != "" {
			if _, err := r.Construction(); err != nil {
				return reading.refuse(block, "%v", err)
			}
		}
		reading.set.recipes[name.Value] = r
		return nil
	})
}

// resolvedRecipes runs after all blocks so references work in either order.
func (reading reader) resolvedRecipes() error {
	resolve := func(name, kind, recipe, detail string) (string, string, error) {
		if (kind == "") == (recipe == "") {
			return "", "", fmt.Errorf("%s: %s needs exactly one of kind or recipe", reading.file, name)
		}
		if recipe != "" {
			r, ok := reading.set.recipes[recipe]
			if !ok {
				return "", "", fmt.Errorf("%s: %s names unknown recipe %q", reading.file, name, recipe)
			}
			kind = r.kind
		}
		if detail == "" {
			detail = reading.set.complexity
		}
		return kind, detail, nil
	}
	for i := range reading.set.things {
		t := &reading.set.things[i]
		kind, detail, err := resolve(t.name, t.kind, t.recipe, t.complexity)
		if err != nil {
			return err
		}
		if kind == "fire" && t.alight {
			return fmt.Errorf("%s: fire %s is already alight", reading.file, t.name)
		}
		t.kind, t.complexity = kind, detail
	}
	for i := range reading.set.stands {
		s := &reading.set.stands[i]
		kind, detail, err := resolve(s.name, s.kind, s.recipe, s.complexity)
		if err != nil {
			return err
		}
		if kind == "fire" && s.alight != 0 {
			return fmt.Errorf("%s: fire stand %s is already alight", reading.file, s.name)
		}
		s.kind, s.complexity = kind, detail
		if s.excluded != "" && !reading.set.named(s.excluded) {
			return fmt.Errorf("%s: stand %s excludes unknown object %q", reading.file, s.name, s.excluded)
		}
		if s.weather != "" {
			if _, ok := reading.set.weather[s.weather]; !ok {
				return fmt.Errorf("%s: unknown weather %q", reading.file, s.weather)
			}
		}
	}
	return nil
}

// VolumeLighting selects direct lighting or the approximate GPU smoke cache.
// Empty retains direct sampling.
func (made Set) VolumeLighting() string { return made.volumeLighting }
