package fishes

import (
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture/profilecheck"
)

// FishProfiles is a checked, immutable book of fish species.
type FishProfiles struct{ file fishFile }

// FishOptions chooses a single swimming fish, a glide, its bones, or a school.
// A school count is zero for one fish; schools cannot be combined with poses.
type FishOptions struct {
	Glide  bool
	Bones  bool
	School int
}

// ReadFishProfiles checks the entire authored book and retains its own data.
func ReadFishProfiles(file string, raw []byte) (*FishProfiles, error) {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return nil, fmt.Errorf("%s: empty or oversized fish book", file)
	}
	if err := profilecheck.SpeciesNumbersPresent(file, raw, []profilecheck.Section{
		{Name: "figures", Kind: reflect.TypeOf(fishFigure{})}, {Name: "looks", Kind: reflect.TypeOf(fishLook{})},
	}); err != nil {
		return nil, err
	}
	var data fishFile
	if err := yamlfile.Read(file, raw, &data); err != nil {
		return nil, err
	}
	if len(data.Species) > 256 {
		return nil, fmt.Errorf("%s: too many fish profiles", file)
	}
	if err := data.checked(); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	for name, row := range data.Species {
		if name == "" {
			return nil, fmt.Errorf("%s: empty fish species name", file)
		}
		if err := row.profileBounds(name); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
	}
	return &FishProfiles{file: data}, nil
}

func (species fishSpecies) profileBounds(name string) error {
	counts := []string{"first-dorsal-spines", "first-dorsal-rays", "second-dorsal-spines", "second-dorsal-rays", "anal-spines", "anal-rays", "dorsal-finlets", "anal-finlets", "caudal-rays", "pectoral-rays", "pelvic-rays", "vertebrae", "precaudal", "keels", "spot-rows", "ribs-from", "bar-count", "school-variants"}
	for key, row := range species.Figures {
		limit := 10.
		if key == "length" {
			limit = 100
		}
		if key == "school-heading" {
			limit = 180
		}
		if slices.Contains(counts, key) {
			limit = 128
			if row.Least < 0 || math.Trunc(row.Least) != row.Least || math.Trunc(row.Most) != row.Most {
				return fmt.Errorf("%s: invalid count %s", name, key)
			}
		}
		if math.Abs(row.Least) > limit || math.Abs(row.Most) > limit {
			return fmt.Errorf("%s: excessive %s", name, key)
		}
	}
	for _, key := range []string{"length", "depth", "deepest", "width", "head", "snout", "stalk", "stalk-width", "eye", "gape", "tail-span", "pectoral", "pelvic", "wavelength", "fin-thickness"} {
		if species.Figures[key].Least < 0.0001 {
			return fmt.Errorf("%s: %s must be positive", name, key)
		}
	}
	if species.Figures["gape"].Most > 1 {
		return fmt.Errorf("%s: gape must lie within the head", name)
	}
	eyeAt, eyeHeight := species.Figures["eye-at"], species.Figures["eye-height"]
	if eyeAt.Least < 0.0001 || eyeAt.Most >= 1 || eyeHeight.Least <= -1 || eyeHeight.Most >= 1 || species.Figures["eye"].Most > 1 {
		return fmt.Errorf("%s: invalid eye position or size", name)
	}
	for _, key := range []string{"pectoral-at", "pelvic-at"} {
		position := species.Figures[key]
		if position.Least < 0.0001 || position.Most >= species.Figures["stalk-at"].Least {
			return fmt.Errorf("%s: %s must lie within the trunk", name, key)
		}
	}
	pectoralHeight := species.Figures["pectoral-height"]
	if pectoralHeight.Least <= -1 || pectoralHeight.Most >= 1 {
		return fmt.Errorf("%s: pectoral height must lie on the body", name)
	}
	lastDorsal := "first-dorsal"
	if species.Figures["second-dorsal-rays"].Most > 0 {
		lastDorsal = "second-dorsal"
	}
	for _, pair := range [][2]string{{"dorsal-finlets", lastDorsal}, {"anal-finlets", "anal"}} {
		if species.Figures[pair[0]].Most > 0 && species.Figures[pair[1]+"-end"].Most+fishFinletGap >= species.Figures["stalk-at"].Least {
			return fmt.Errorf("%s: no room for %s", name, pair[0])
		}
	}
	if species.Figures["bend-from"].Least < 0 || species.Figures["bend-from"].Most >= 1 {
		return fmt.Errorf("%s: invalid bend start", name)
	}
	if species.Figures["spot-rows"].Most > 0 && species.Figures["spot-pitch"].Least < 0.0001 {
		return fmt.Errorf("%s: spotted fish need a positive spot pitch", name)
	}
	for _, key := range []string{"stripe-width", "spot-size", "bar-width", "amplitude", "glide"} {
		if species.Figures[key].Least < 0 {
			return fmt.Errorf("%s: %s cannot be negative", name, key)
		}
	}
	for _, key := range []string{"back-line", "stripe-height", "spot-height"} {
		value := species.Figures[key]
		if value.Least < -1 || value.Most > 1 {
			return fmt.Errorf("%s: %s is outside the body", name, key)
		}
	}
	if species.Figures["bar-count"].Most > 0 {
		bar := species.Figures["bar-from"]
		if bar.Least < 0 || bar.Most >= species.Figures["stalk-at"].Least {
			return fmt.Errorf("%s: bars must begin on the body", name)
		}
	}
	for _, key := range []string{"vertebrae", "precaudal", "caudal-rays", "pectoral-rays", "pelvic-rays"} {
		if species.Figures[key].Least < 3 {
			return fmt.Errorf("%s: too few %s", name, key)
		}
	}
	for key := range species.Looks {
		if !slices.Contains(fishLookNames, key) {
			return fmt.Errorf("%s: unknown look %s", name, key)
		}
	}
	if len(species.Schools) > 0 {
		for _, key := range []string{"school-long", "school-wide", "school-high"} {
			r := species.Figures[key]
			if r.Least < .5 || r.Most > 5 {
				return fmt.Errorf("%s: invalid %s", name, key)
			}
		}
		if species.Figures["school-spacing"].Least < .05 || species.Figures["school-variants"].Least < 1 || species.Figures["school-variants"].Most > 16 {
			return fmt.Errorf("%s: invalid school dimensions or variants", name)
		}
		if species.Figures["school-scatter"].Least < 0 || species.Figures["school-heading"].Least < 0 {
			return fmt.Errorf("%s: invalid school scatter or heading", name)
		}
	}
	return nil
}

// NewFish retains the selected profile and options as a reusable recipe.
func (book *FishProfiles) NewFish(name string, options FishOptions) (*Recipe, error) {
	if book == nil {
		return nil, fmt.Errorf("manufacture: no fish profile book")
	}
	profile, found := book.file.Species[name]
	if !found {
		return nil, fmt.Errorf("manufacture: no fish profile %q", name)
	}
	if options.School < 0 || options.School > mostSchool || (options.Bones && options.Glide) || (options.School > 0 && (options.Bones || options.Glide)) {
		return nil, fmt.Errorf("manufacture: invalid fish options")
	}
	if options.School > 0 && len(profile.Schools) == 0 {
		return nil, fmt.Errorf("manufacture: fish %q has no school profile", name)
	}
	return &Recipe{name: name, profile: profile, options: options}, nil
}
