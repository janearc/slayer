package preview

import (
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

// InspectFull resolves the authored still contract without creating a snapshot.
func InspectFull(path, source string, profile Profile) (Profile, error) {
	full, err := ReadFullFilm(path, source)
	if err != nil {
		return Profile{}, err
	}
	return full.Resolve(profile)
}

// Resolve checks selected output and sampling against the authored still.
func (full FullFilm) Resolve(profile Profile) (Profile, error) {
	profile.Full = true
	profile, err := profile.resolved()
	if err != nil {
		return Profile{}, err
	}
	profile.rig, profile.stock, profile.reference = full.rig, full.stock, full.reference
	profile.region = full.region
	if profile.CameraPathRecipe != "" && (full.rig != nil || full.region != nil) {
		return Profile{}, fmt.Errorf("an inherited job camera path requires an uncropped set without a physical rig")
	}
	if err := full.resolveSampling(&profile); err != nil {
		return Profile{}, err
	}
	if profile.SizePreset == "" {
		profile.Width, err = full.authoredDimension("width", full.preset.Width)
		if err != nil {
			return Profile{}, err
		}
	}
	if full.rig != nil {
		if profile.CameraWide != 0 {
			return Profile{}, fmt.Errorf("a physical camera rig cannot use an inherited job zoom")
		}
		if full.region != nil {
			return Profile{}, fmt.Errorf("a physical camera rig does not support a film region")
		}
		return full.riggedProfile(profile)
	}
	if profile.SizePreset == "" {
		profile.Height, err = full.authoredDimension("height", full.preset.Height)
		if err != nil {
			return Profile{}, err
		}
	} else if profile.CameraPathRecipe != "" {
		width, err := full.authoredDimension("width", full.preset.Width)
		if err != nil {
			return Profile{}, err
		}
		height, err := full.authoredDimension("height", full.preset.Height)
		if err != nil {
			return Profile{}, err
		}
		w, h, err := fitCameraAspect(width, height, profile.Width, profile.Height)
		if err != nil || w != profile.Width || h != profile.Height {
			return Profile{}, fmt.Errorf("use the authored aspect ratio to keep the job camera path framing")
		}
	}
	return full.resolveRegion(profile)
}

// resolveRegion checks the crop against the frame after all size overrides.
func (full FullFilm) resolveRegion(profile Profile) (Profile, error) {
	region := full.region
	if region == nil {
		return profile, nil
	}
	if profile.NativeGPU {
		return Profile{}, fmt.Errorf("a film region requires CPU or software GPU")
	}
	if region.Left < 0 || region.Top < 0 || region.Width <= 0 || region.Height <= 0 ||
		region.Left > profile.Width-region.Width || region.Top > profile.Height-region.Height {
		return Profile{}, fmt.Errorf("film region is outside the resolved %d × %d frame", profile.Width, profile.Height)
	}
	return profile, nil
}

// resolveSampling preserves the job's rays-per-axis grid unless a total is supplied.
func (full FullFilm) resolveSampling(profile *Profile) error {
	if node := field(full.film, "samples"); node != nil {
		if field(full.film, "rays") != nil {
			return fmt.Errorf("say film.samples or film.rays, not both")
		}
		if profile.Samples == 0 {
			samples, err := filmInteger(node, "samples")
			if err != nil || samples > 64 {
				return fmt.Errorf("authored exact samples require a count from 1 to 64")
			}
			profile.Samples = samples
		}
	}
	if profile.Samples != 0 {
		if full.rig != nil {
			return fmt.Errorf("exact samples do not yet support physical camera rigs")
		}
		if full.exactError != "" {
			return fmt.Errorf("%s", full.exactError)
		}
		profile.CPURenderer = !profile.GPU && !profile.NativeGPU
		return nil
	}
	var err error
	profile.Rays, err = filmInteger(full.setting("rays"), "rays")
	return err
}

// authoredDimension gives an explicit job dimension precedence over its
// preset, with the set supplying anything neither specifies.
func (full FullFilm) authoredDimension(name string, preset int) (int, error) {
	if node := field(full.film, name); node != nil {
		return filmInteger(node, name)
	}
	if preset != 0 {
		return preset, nil
	}
	return filmInteger(field(full.setFilm, name), name)
}

// setting follows producer precedence from job to set.
func (full FullFilm) setting(name string) *yaml.Node {
	if node := field(full.film, name); node != nil {
		return node
	}
	return field(full.setFilm, name)
}

// filmInteger reports a missing or nonpositive size or sample count.
func filmInteger(node *yaml.Node, name string) (int, error) {
	if node == nil {
		return 0, fmt.Errorf("full render requires explicit authored film.%s", name)
	}
	var value int
	if err := node.Decode(&value); err != nil || value <= 0 {
		return 0, fmt.Errorf("full render requires positive authored film.%s", name)
	}
	return value, nil
}

// riggedProfile follows the producer's square-pixel rows and one-row tolerance.
func (full FullFilm) riggedProfile(profile Profile) (Profile, error) {
	for _, key := range []string{"exposure", "wide", "tall"} {
		if field(full.film, key) != nil {
			return Profile{}, fmt.Errorf("a physical film takes its exposure and angle from the rig")
		}
	}
	height := profile.Height
	if profile.SizePreset == "" {
		height = full.preset.Height
		if node := field(full.film, "height"); node != nil {
			var err error
			height, err = filmInteger(node, "height")
			if err != nil {
				return Profile{}, err
			}
		}
	}
	// The producer converts both frame dimensions to metres before deriving rows.
	const millimetre = 0.001
	rows := int(math.Round(float64(profile.Width) * (full.rig.HeightMM * millimetre) / (full.rig.WidthMM * millimetre)))
	if rows <= 0 {
		return Profile{}, fmt.Errorf("physical frame produces no image rows")
	}
	if height > 0 && math.Abs(float64(height)-float64(rows)) > 1 {
		return Profile{}, fmt.Errorf("output size %d × %d does not match the physical frame; at this width it needs %d rows", profile.Width, height, rows)
	}
	profile.Height = rows
	return profile, nil
}
