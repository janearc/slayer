package thing

import "fmt"

// SubsurfaceProfile describes lateral diffusion on an opaque surface. Radius
// gives the Gaussian width in mesh-local units for each linear RGB channel.
// Share is the fraction of diffuse reflection that uses this profile.
type SubsurfaceProfile struct {
	Radius [3]float64
	Share  float64
}

// Subsurface is an optional visible-light diffusion response. Its zero value
// keeps all diffuse reflection local.
type Subsurface struct {
	profile SubsurfaceProfile
}

// NewSubsurface validates the profile. A zero share disables diffusion.
func NewSubsurface(profile SubsurfaceProfile) (Subsurface, error) {
	if !thingShare(profile.Share) {
		return Subsurface{}, fmt.Errorf("subsurface: share must be between zero and one")
	}
	if profile.Share == 0 {
		return Subsurface{}, nil
	}
	for _, radius := range profile.Radius {
		if !(radius > 0) || !thingFinite(3*radius) {
			return Subsurface{}, fmt.Errorf("subsurface: radii must be positive and finite through three widths")
		}
	}
	return Subsurface{profile: profile}, nil
}

// Enabled reports whether the material diffuses a positive share.
func (made Subsurface) Enabled() bool { return made.profile.Share > 0 }

// Profile returns the authored widths and diffuse share.
func (made Subsurface) Profile() SubsurfaceProfile { return made.profile }
