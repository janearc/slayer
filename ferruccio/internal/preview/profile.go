package preview

import (
	"fmt"
	"math"
	"math/big"
)

// resolved preserves the original profile unless both dimensions are supplied.
func (profile Profile) resolved() (Profile, error) {
	if profile.CameraWide != 0 && (profile.Camera == "" || math.IsNaN(profile.CameraWide) ||
		profile.CameraWide <= 0 || profile.CameraWide >= 180) {
		return Profile{}, fmt.Errorf("a camera angle requires a named camera and a value between zero and 180 degrees")
	}
	if profile.CameraPathRecipe != "" && (profile.Camera != "" || profile.CameraWide != 0) {
		return Profile{}, fmt.Errorf("a camera path and a named camera cannot be used together")
	}
	if profile.StillAt != 0 && (profile.Camera == "" || profile.CameraPathRecipe != "") {
		return Profile{}, fmt.Errorf("an independent still time requires an explicit scene camera")
	}
	if (profile.CPURenderer && (profile.GPU || profile.NativeGPU)) ||
		(profile.GPU && profile.NativeGPU) {
		return Profile{}, fmt.Errorf("choose only one render backend")
	}
	if profile.NativeGPU && !profile.Full {
		return Profile{}, fmt.Errorf("native GPU requires a full render")
	}
	if profile.Samples < 0 || profile.Samples > 64 || (profile.Samples > 0 && !profile.Full) {
		return Profile{}, fmt.Errorf("exact samples require a full render and a count from 1 to 64")
	}
	if profile.Samples > 0 && !profile.GPU && !profile.NativeGPU {
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

// fitCameraAspect preserves the set's exact aspect within the Preview bounds.
func fitCameraAspect(width, height, maxWidth, maxHeight int) (int, int, error) {
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("an inherited camera path requires an authored frame size")
	}
	divisor, rest := width, height
	for rest != 0 {
		divisor, rest = rest, divisor%rest
	}
	width, height = width/divisor, height/divisor
	scale := min(maxWidth/width, maxHeight/height)
	if scale == 0 {
		return 0, 0, fmt.Errorf("the authored frame ratio does not fit Preview; use Full Render")
	}
	return width * scale, height * scale, nil
}

// rendererOverride names the renderer sent to Slayer.
func (profile Profile) rendererOverride() string {
	switch {
	case profile.CPURenderer:
		return "cpu"
	case profile.GPU:
		return "gpu"
	case profile.NativeGPU:
		return "gpu-native"
	}
	return ""
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

// SamplesPerPixel counts the spatial samples in one exposure glimpse.
// An explicit sample count takes precedence over the authored ray grid.
func (profile Profile) SamplesPerPixel() *big.Int {
	if profile.Full && profile.Samples > 0 {
		return big.NewInt(int64(profile.Samples))
	}
	side := big.NewInt(int64(profile.samples()))
	return new(big.Int).Mul(side, side)
}

// Glimpses is the number of temporal shutter samples for an exposure.
func (profile Profile) Glimpses() int {
	if profile.rig != nil {
		return profile.rig.Glimpses
	}
	return 1
}

// OutputSize returns the PNG dimensions, including an authored crop.
func (profile Profile) OutputSize() (width, height int) {
	if profile.region != nil {
		return profile.region.Width, profile.region.Height
	}
	return profile.Width, profile.Height
}

// PrimarySamples counts every spatial and temporal sample across the output.
func (profile Profile) PrimarySamples() *big.Int {
	width, height := profile.OutputSize()
	total := new(big.Int).Mul(big.NewInt(int64(width)), big.NewInt(int64(height)))
	total.Mul(total, profile.SamplesPerPixel())
	return total.Mul(total, big.NewInt(int64(profile.Glimpses())))
}
