package gpu

import (
	"fmt"

	"github.com/janearc/slayer/film"
)

// CompositePass selects the primary path's contribution. SurfacePlate keeps
// surface lighting, reflections and their volume shadows, but leaves the
// primary ray's foreground volume for VolumeTransport to draw separately.
// A reused plate freezes that surface lighting at its original scene time.
type CompositePass uint8

const (
	// CompletePicture is the ordinary renderer, with no decomposition.
	CompletePicture CompositePass = iota
	// SurfacePlate omits only the primary ray's foreground volume.
	SurfacePlate
	// VolumeTransport returns scattered light and RGB transmission, stopped
	// at the first surface. It does not trace reflected or refracted paths.
	VolumeTransport
	// WithoutPrimaryBeams keeps the picture except direct beam scattering
	// on the camera ray. Beam lighting on surfaces and secondary paths stays.
	WithoutPrimaryBeams
	// PrimaryBeams contains only direct beam scattering on the camera ray,
	// stopped at the first surface and attenuated by every crossed volume.
	// Add its linear RGB to WithoutPrimaryBeams before tone mapping.
	// The two beam passes refuse CachedLighting's mixed source grid.
	PrimaryBeams
)

// passSource gives a pass its own kernel identity. The normal draw adds no
// selector; per-call renderers also keep the complete-picture path.
func passSource(source string, pass CompositePass) string {
	if pass == CompletePicture {
		return source
	}
	return fmt.Sprintf("#define SLAYER_COMPOSITE_PASS %d\n", pass) + source
}

// validPass refuses unknown selectors and a source cache that cannot be
// separated into complementary contributions before opening a device.
func validPass(mode Mode, pass CompositePass) error {
	if pass > PrimaryBeams {
		return fmt.Errorf("gpu: unknown composite pass %d", pass)
	}
	if mode == CachedLighting && pass >= WithoutPrimaryBeams {
		return fmt.Errorf("gpu: beam passes cannot separate " +
			"CachedLighting's mixed source grid; use NativeMeshes " +
			"or a per-lamp lighting mode")
	}
	return nil
}

// compositePinhole admits the public constructor's pointer and a value,
// refusing a typed nil before the camera or native drawer can use it.
func compositePinhole(lens film.Sensor) bool {
	switch pinhole := lens.(type) {
	case film.Pinhole:
		return true
	case *film.Pinhole:
		return pinhole != nil
	default:
		return false
	}
}
