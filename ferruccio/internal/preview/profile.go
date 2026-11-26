package preview

import "fmt"

// resolved preserves the original profile unless both dimensions are supplied.
func (profile Profile) resolved() (Profile, error) {
	if profile.Samples < 0 || profile.Samples > 64 || (profile.Samples > 0 && (!profile.Full || (profile.GPU && profile.CPURenderer))) {
		return Profile{}, fmt.Errorf("exact samples require a full render, one backend, and a count from 1 to 64")
	}
	if profile.Samples > 0 && !profile.GPU {
		profile.CPURenderer = true
	}
	if profile.SizePreset != "" && (!profile.Full || profile.Width <= 0 || profile.Height <= 0) {
		return Profile{}, fmt.Errorf("an output-size preset needs full render dimensions")
	}
	if profile.Full && profile.Width == 0 && profile.Height == 0 {
		return profile, nil
	}
	if profile.Width == 0 && profile.Height == 0 {
		profile.Width, profile.Height = 640, 360
	}
	if profile.Width <= 0 || profile.Height <= 0 {
		return Profile{}, fmt.Errorf("preview width and height must both be positive")
	}
	return profile, nil
}

// lane follows the explicit CPU override; catalogue keeps its own renderer.
func (plan *Plan) lane() string {
	if plan.Profile.CPURenderer && plan.Kind != "catalogue" {
		return "tests"
	}
	return "gpu"
}

// quality names the requested producer mode, independently of authored sampling.
func (profile Profile) quality() string {
	if profile.Full {
		return "exact"
	}
	return "preview"
}

// samples is rays per axis, so the sample count per pixel is its square.
func (profile Profile) samples() int {
	if profile.Full {
		return profile.Rays
	}
	return 1
}
