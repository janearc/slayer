package preview

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"gopkg.in/yaml.v3"
)

// InspectFull resolves the authored still contract without creating a
// snapshot or starting a render. A selected preset supplies its frozen size.
func InspectFull(path, source string, profile Profile) (Profile, error) {
	profile.Full = true
	profile, err := profile.resolved()
	if err != nil {
		return Profile{}, err
	}
	document, err := draft.Open(filepath.Base(path), []byte(source))
	if err != nil {
		return Profile{}, err
	}
	tree, err := document.Tree()
	if err != nil {
		return Profile{}, err
	}
	return authoredProfile(tree, document.Kind(), profile)
}

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
	if (kind == "set" || kind == "legacy-set") && field(film, "frame") != nil {
		if profile.Samples > 0 {
			return Profile{}, fmt.Errorf("exact samples do not yet support physical camera rigs")
		}
		return riggedProfile(film, profile)
	}
	values := []*int{&profile.Width, &profile.Height, &profile.Rays}
	for index, name := range []string{"width", "height", "rays"} {
		if name == "rays" && profile.Samples > 0 {
			continue
		}
		if name == "rays" && kind == "job" && field(film, "samples") != nil {
			value, err := strconv.Atoi(field(film, "samples").Value)
			if err != nil || value < 1 || value > 64 {
				return Profile{}, fmt.Errorf("authored exact samples require a count from 1 to 64")
			}
			profile.Samples = value
			if !profile.GPU {
				profile.CPURenderer = true
			}
			continue
		}
		if index < 2 && profile.SizePreset != "" {
			continue
		}
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

// riggedProfile follows the producer's square-pixel rows and size tolerance.
// A rig owns its height, so no height override is invented for an authored size.
func riggedProfile(film *yaml.Node, profile Profile) (Profile, error) {
	frame := field(film, "frame")
	dimensions := [2]float64{}
	for i, name := range []string{"width", "height"} {
		value := field(frame, name)
		if value == nil {
			return Profile{}, fmt.Errorf("physical frame requires %s", name)
		}
		parsed, err := strconv.ParseFloat(value.Value, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 1 || parsed > 1000 {
			return Profile{}, fmt.Errorf("physical frame %s must be between 1 and 1000 mm", name)
		}
		dimensions[i] = parsed
	}
	rays := field(film, "rays")
	if rays == nil {
		return Profile{}, fmt.Errorf("full render requires explicit authored film.rays")
	}
	var err error
	profile.Rays, err = strconv.Atoi(rays.Value)
	if err != nil || profile.Rays <= 0 || profile.Rays > 64 {
		return Profile{}, fmt.Errorf("full render requires film.rays from 1 to 64")
	}
	if profile.SizePreset == "" {
		width := field(film, "width")
		if width == nil {
			return Profile{}, fmt.Errorf("full render requires explicit authored film.width")
		}
		profile.Width, err = strconv.Atoi(width.Value)
		if err != nil || profile.Width <= 0 {
			return Profile{}, fmt.Errorf("full render requires positive authored film.width")
		}
	}
	rows := int(math.Round(float64(profile.Width) * dimensions[1] / dimensions[0]))
	if rows <= 0 {
		return Profile{}, fmt.Errorf("physical frame produces no image rows")
	}
	if profile.SizePreset != "" && absInt(profile.Height-rows) > 1 {
		return Profile{}, fmt.Errorf("output size %d × %d does not match the physical frame; at this width it needs %d rows", profile.Width, profile.Height, rows)
	}
	profile.Height = rows
	return profile, nil
}

// absInt measures a difference in nonnegative image dimensions.
func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
