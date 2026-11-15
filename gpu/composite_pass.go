package gpu

import "fmt"

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
)

// passSource gives a pass its own kernel identity. The normal draw adds no
// selector; per-call renderers also keep the complete-picture path.
func passSource(source string, pass CompositePass) string {
	if pass == CompletePicture {
		return source
	}
	return fmt.Sprintf("#define SLAYER_COMPOSITE_PASS %d\n", pass) + source
}

// validPass refuses unknown pass selectors before opening a device.
func validPass(pass CompositePass) error {
	if pass > VolumeTransport {
		return fmt.Errorf("gpu: unknown composite pass %d", pass)
	}
	return nil
}
