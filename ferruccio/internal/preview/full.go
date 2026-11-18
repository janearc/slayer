package preview

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"strconv"
)

// authoredProfile refuses formats whose complete still contract is not connected.
// Sampling and dimensions stay in the frozen YAML; no CLI size overrides are used.
func authoredProfile(tree *yaml.Node, kind string, profile Profile) (Profile, error) {
	if kind == "catalogue" {
		return Profile{}, fmt.Errorf("full render is not supported for catalogue plans; use the catalogue producer")
	}
	root := tree.Content[0]
	if kind == "set" || kind == "legacy-set" {
		root = field(root, "scene")
	}
	film := field(root, "film")
	values := []*int{&profile.Width, &profile.Height, &profile.Rays}
	for index, name := range []string{"width", "height", "rays"} {
		value := field(film, name)
		if value == nil {
			return Profile{}, fmt.Errorf("full render requires explicit authored film.%s; presets and inherited film settings are not connected", name)
		}
		number, err := strconv.Atoi(value.Value)
		if err != nil || number <= 0 {
			return Profile{}, fmt.Errorf("full render requires positive authored film.%s", name)
		}
		*values[index] = number
	}
	return profile, nil
}
