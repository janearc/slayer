package set

import (
	"github.com/janearc/slayer/scene/thing"
	yaml "gopkg.in/yaml.v3"
)

// VolumeIllumination is the checked source model for painted weather and smoke.
// It is independent of VolumeLighting, which selects the smoke cache backend.
func (made Set) VolumeIllumination() thing.VolumeIllumination { return made.illumination }

// volumeIllumination preserves the legacy scene when the block is absent.
// The shared constructor holds its even angular-work rule in one place.
func (reading reader) volumeIllumination(_, value *yaml.Node) error {
	var count float64
	err := reading.keys(value, map[string]step{
		"model":       reading.only("isotropic-single"),
		"sky-samples": reading.whole(&count, 2, 64),
	}, nil)
	if err != nil {
		return err
	}
	if !says(value, "model") || !says(value, "sky-samples") {
		return reading.refuse(value, "volume-illumination needs model and sky-samples")
	}
	checked, err := thing.NewVolumeIllumination(int(count))
	if err != nil {
		return reading.refuse(value, "%s", err)
	}
	reading.set.illumination = checked
	return nil
}
