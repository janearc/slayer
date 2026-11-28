package set

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	yaml "gopkg.in/yaml.v3"
)

// Recipe is a reusable construction specification, independent of placement.
type Recipe struct {
	profileKind                                    *manufacture.Kind
	profileFamily                                  string
	profileSpecies                                 string
	profileDigest                                  [32]byte
	profilePath                                    string
	profileBuiltin                                 string
	treeProfiles                                   *manufacture.PlantProfiles
	composition                                    *cities.Composition
	compositionDigest                              [32]byte
	compositionPath                                string
	burn                                           *manufacture.BurnSpec
	kind                                           string
	batchPlacement                                 manufacture.CityBatchPlacementMode
	vacantLotIndex                                 manufacture.CityVacantLotIndexMode
	design                                         string
	season                                         string
	script                                         string
	carries                                        []string
	diameter, aperture, opening, azimuth, altitude float64
	segmented                                      bool
	poseClip                                       bool
	tower                                          manufacture.TowerSpec
	bridge                                         manufacture.BridgeSpec
	street                                         manufacture.StreetSpec
	lashings                                       manufacture.LashingsSpec
	parcel                                         superblockAsk
	digest                                         string
}

// Kind names the construction rule shared by these specimens.
func (recipe Recipe) Kind() string { return recipe.kind }

// HasTreeProfiles reports a city or tower recipe with a checked tree book.
func (recipe Recipe) HasTreeProfiles() bool { return recipe.treeProfiles != nil }

// HasPoseClip reports a recipe with a checked authored bioform clip.
func (recipe Recipe) HasPoseClip() bool { return recipe.poseClip }

// IsPlant reports a checked plant profile recipe.
func (recipe Recipe) IsPlant() bool {
	return recipe.profileKind != nil && recipe.kind == "plant-profile"
}

// IsAnimal reports a checked animal profile recipe.
func (recipe Recipe) IsAnimal() bool {
	return recipe.profileKind != nil && recipe.kind == "animal-profile"
}

// IsAlga reports a checked alga profile recipe.
func (recipe Recipe) IsAlga() bool {
	return recipe.profileKind != nil && recipe.kind == "alga-profile"
}

// AlgaSpecies names the species in a checked alga recipe, or empty.
func (recipe Recipe) AlgaSpecies() string {
	if !recipe.IsAlga() {
		return ""
	}
	return recipe.profileSpecies
}

// AnimalSpecies names the species in a checked animal recipe, or empty.
func (recipe Recipe) AnimalSpecies() string {
	if !recipe.IsAnimal() {
		return ""
	}
	return recipe.profileSpecies
}

// PlantFamily names the family in a checked plant recipe, or empty.
func (recipe Recipe) PlantFamily() string {
	if !recipe.IsPlant() {
		return ""
	}
	return recipe.profileFamily
}

// PlantSpecies names the species in a checked plant recipe, or empty.
func (recipe Recipe) PlantSpecies() string {
	if !recipe.IsPlant() {
		return ""
	}
	return recipe.profileSpecies
}

// BatchPlacement is the city's explicit material-batch placement mode, if set.
func (recipe Recipe) BatchPlacement() manufacture.CityBatchPlacementMode {
	return recipe.batchPlacement
}

// VacantLotIndex is the clustered city's vacant-lot enumeration mode, if set.
func (recipe Recipe) VacantLotIndex() manufacture.CityVacantLotIndexMode {
	return recipe.vacantLotIndex
}

// Script is the script the recipe asks the words of its signs to be in,
// or nothing if it says none, which is Latin capitals.
func (recipe Recipe) Script() string { return recipe.script }

// Carries are the trades the recipe asks every building it makes to
// carry a sign of, once each, whatever the weights of the sign culture say.
func (recipe Recipe) Carries() []string { return slices.Clone(recipe.carries) }

// Construction selects the checked solid recipe. Volumetric fire uses Fire.
func (recipe Recipe) Construction() (*manufacture.Kind, error) {
	if recipe.profileKind != nil {
		row := *recipe.profileKind
		return &row, nil
	}
	if recipe.composition != nil {
		row, err := manufacture.Composition(*recipe.composition)
		return &row, err
	}
	if recipe.kind == "space-observatory" {
		aperture := measure.Metres(recipe.aperture)
		return manufacture.Spacecraft(recipe.design, aperture)
	}
	row, err := manufacture.KindNamed(recipe.kind)
	if recipe.kind == "optical-observatory" {
		row, err = manufacture.Observatory(recipe.Optical())
	}
	if recipe.kind == "tower" && err == nil {
		row, err = manufacture.Tower(recipe.tower)
	}
	if recipe.kind == "bridge" && err == nil {
		row, err = manufacture.Bridge(recipe.bridge)
	}
	if recipe.kind == "street" && err == nil {
		row, err = manufacture.Street(recipe.street)
	}
	if recipe.kind == manufacture.LashedStreetKind && err == nil {
		row, err = manufacture.LashedStreet(recipe.lashings)
	}
	if recipe.kind == manufacture.VillageKind && err == nil {
		row, err = manufacture.Village(manufacture.VillageSpec{
			Width: recipe.parcel.width, Depth: recipe.parcel.depth})
	}
	if err != nil {
		return nil, err
	}
	if recipe.treeProfiles != nil && recipe.kind != "tower" {
		row, err = manufacture.WithCityTreeProfiles(row, recipe.treeProfiles)
		if err != nil {
			return nil, err
		}
	}
	if recipe.season != "" {
		row, err = manufacture.WithSeason(row, recipe.season)
		if err != nil {
			return nil, err
		}
	}
	if recipe.batchPlacement != "" {
		row, err = manufacture.WithCityBatchPlacement(row, recipe.batchPlacement)
		if err != nil {
			return nil, err
		}
	}
	if recipe.vacantLotIndex != "" {
		row, err = manufacture.WithCityVacantLotIndex(row, recipe.vacantLotIndex)
		if err != nil {
			return nil, err
		}
	}
	if recipe.script != "" || len(recipe.carries) > 0 {
		row, err = manufacture.WithScript(row, recipe.script,
			recipe.carries)
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

// placedComplexity accepts what complexity does, or levelled: a thing or
// a stand drawn as finely as the set's camera sees it.
func (reading reader) placedComplexity(target *string) step {
	return reading.word(target, append(manufacture.Complexities(),
		manufacture.Levelled)...)
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
	return reading.each(node, reading.digested(func(name, block *yaml.Node) error {
		if says(block, "animal") {
			return reading.animalRecipe(name, block)
		}
		if says(block, "pose-clip") {
			return reading.poseRecipe(name, block)
		}
		if says(block, "plant") {
			return reading.plantRecipe(name, block)
		}
		if says(block, "alga") {
			return reading.algaRecipe(name, block)
		}
		if says(block, "composition") {
			return reading.structureRecipe(name, block)
		}
		if valueOf(block, "kind") == "fire" {
			return reading.fireRecipe(name, block)
		}
		if valueOf(block, "kind") == "space-observatory" {
			return reading.spacecraftRecipe(name, block)
		}
		r := Recipe{}
		var treeFile string
		steps := map[string]step{
			"tree-profiles": func(_, value *yaml.Node) error {
				if err := reading.keys(value, map[string]step{
					"file": func(key, file *yaml.Node) error {
						if file.Kind != yaml.ScalarNode || file.ShortTag() != "!!str" || strings.TrimSpace(file.Value) == "" {
							return reading.refuse(file, "%s needs a nonempty file path", key.Value)
						}
						treeFile = file.Value
						return nil
					},
				}, nil); err != nil {
					return err
				}
				return reading.all(value, "tree profiles", "file")
			},
			"kind":         reading.word(&r.kind, manufacture.KindNames()...),
			"season":       reading.word(&r.season, manufacture.SeasonNames()...),
			"script":       reading.word(&r.script, manufacture.SignScripts()...),
			"carries":      reading.words(&r.carries),
			"irregularity": reading.irregularity(&r.tower, &r.bridge),
			"pins":         reading.towerPins(&r.tower),
			"span":         reading.quantity(&r.bridge.Span, "m", 1, bridgeMostSpan),
			"bare":         reading.boolean(&r.bridge.Bare),
			"hub":          reading.boolean(&r.bridge.Hub),
			"depth":        reading.bridgeDepth(&r.bridge),
			"wetness":      reading.number(&r.street.Wetness, 0, 1),
			"parcel":       reading.parcel(&r.parcel),
			"length":       reading.quantity(&r.street.Length, "m", 8, 100),
			"batch-placement": func(key, value *yaml.Node) error {
				if err := reading.only(string(manufacture.CityBatchFused),
					string(manufacture.CityBatchNested))(key, value); err != nil {
					return err
				}
				r.batchPlacement = manufacture.CityBatchPlacementMode(value.Value)
				return nil
			},
			"vacant-lot-index": func(key, value *yaml.Node) error {
				if err := reading.only(string(manufacture.CityVacantLotIndexOn),
					string(manufacture.CityVacantLotIndexOff))(key, value); err != nil {
					return err
				}
				r.vacantLotIndex = manufacture.CityVacantLotIndexMode(value.Value)
				return nil
			},
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
		}
		maps.Copy(steps, reading.lashingKeys(&r.lashings))
		if err := reading.keys(block, steps, nil); err != nil {
			return err
		}
		if err := reading.lashingsRefused(r, name, block); err != nil {
			return err
		}
		if r.kind == "" {
			return reading.refuse(block, "recipe %s needs a kind", name.Value)
		}
		if says(block, "tree-profiles") {
			if r.kind != "tower" && r.kind != "city-clustered-green-10000" && r.kind != "city-clustered-night-10000" {
				return reading.refuse(block, "recipe %s does not support tree profiles", name.Value)
			}
			path, raw, err := reading.profileFile(treeFile)
			if err != nil {
				return err
			}
			book, err := manufacture.ReadPlantProfiles(path, raw)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			r.profilePath, r.profileDigest, r.treeProfiles = path, sha256.Sum256(raw), book
			if r.kind == "tower" {
				r.tower.TreeProfiles = book
			}
			if _, err := r.Construction(); err != nil {
				return reading.refuse(block, "%v", err)
			}
		}
		if r.kind != "tower" && r.kind != "bridge" && says(block, "irregularity") {
			return reading.refuse(block, "recipe %s is not a tower or a bridge", name.Value)
		}
		if r.kind != "tower" && says(block, "pins") {
			return reading.refuse(block, "recipe %s is not a tower", name.Value)
		}
		if r.kind != "bridge" && (says(block, "span") ||
			says(block, "depth") || says(block, "bare") || says(block, "hub")) {
			return reading.refuse(block, "recipe %s is not a bridge", name.Value)
		}
		if r.bridge.Bare && says(block, "depth") {
			return reading.refuse(block, "recipe %s is a bare bridge, "+
				"which has no lots to nest, so no depth", name.Value)
		}
		if r.kind != "street" && says(block, "wetness") {
			return reading.refuse(block, "recipe %s is not a street", name.Value)
		}
		if r.kind != manufacture.VillageKind && says(block, "parcel") {
			return reading.refuse(block, "recipe %s is not a village, so "+
				"it has no parcel", name.Value)
		}
		if r.kind != "street" && r.kind != manufacture.LashedStreetKind &&
			says(block, "length") {
			return reading.refuse(block, "recipe %s is not a street or a "+
				"lashed street, so it has no length", name.Value)
		}
		if r.kind == "tower" || r.kind == "bridge" || r.kind == "street" ||
			r.kind == manufacture.VillageKind {
			if _, err := r.Construction(); err != nil {
				return reading.refuse(block, "%v", err)
			}
		}
		if r.kind == "optical-observatory" {
			if _, err := manufacture.Observatory(r.Optical()); err != nil {
				return reading.refuse(block, "%v", err)
			}
		} else if says(block, "enclosure") || says(block, "telescope") {
			return reading.refuse(block, "recipe %s is not an optical observatory", name.Value)
		}
		if r.season != "" || r.script != "" || len(r.carries) > 0 {
			if _, err := r.Construction(); err != nil {
				return reading.refuse(block, "%v", err)
			}
		}
		if r.batchPlacement != "" || r.vacantLotIndex != "" {
			if _, err := r.Construction(); err != nil {
				return reading.refuse(block, "%v", err)
			}
		}
		reading.set.recipes[name.Value] = r
		return nil
	}))
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
		if kind == "fire" && t.position != nil && t.scale != nil && reading.set.InSpace() {
			return fmt.Errorf("%s: scale is not supported for fire recipe placement %q", reading.file, t.name)
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
		if s.recipe != "" {
			config := reading.set.recipes[s.recipe]
			if config.batchPlacement != "" {
				return fmt.Errorf("%s: city batch placement recipe %q can only be used by an individual thing, not stand %s", reading.file, s.recipe, s.name)
			}
			if config.vacantLotIndex != "" {
				return fmt.Errorf("%s: vacant lot index recipe %q can only be used by an individual thing, not stand %s", reading.file, s.recipe, s.name)
			}
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
