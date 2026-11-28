package island

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// ProceduralHeights is an art-directed, seeded radial height provider. Its
// three-dimensional noise is fixed on the sphere, independent of camera and
// geographic chart. It is scenery, not surveyed or simulated geology.
type ProceduralHeights struct {
	seed                                   uint64
	radius, base, height, wavelength, fine float64
}

// ProceduralGround supplies a radial base, mountain amplitude and wavelength,
// and a smaller fine-detail amplitude, all in metres. The envelope, octave
// counts and ridge shaping are artistic choices. The source's conservative
// height envelope is [Base, Base+Height+Fine].
type ProceduralGround struct {
	Seed                                   uint64
	Radius, Base, Height, Wavelength, Fine float64
}

// NewProceduralHeights constructs a fixed global field. It does not choose a
// view, mesh, sea or colour, and does not implement terrain erosion.
func NewProceduralHeights(g ProceduralGround) (*ProceduralHeights, error) {
	for _, n := range []float64{g.Radius, g.Base, g.Height, g.Wavelength, g.Fine} {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, fmt.Errorf("island: procedural ground needs finite dimensions")
		}
	}
	if g.Radius <= 0 || g.Radius+g.Base <= 0 || math.IsInf(g.Radius+g.Base, 0) || g.Height < 0 || g.Fine < 0 ||
		g.Wavelength <= 0 || g.Radius/g.Wavelength > 1e9 ||
		math.IsInf(g.Base+g.Height+g.Fine, 0) {
		return nil, fmt.Errorf("island: invalid procedural ground dimensions")
	}
	return &ProceduralHeights{seed: g.Seed, radius: g.Radius, base: g.Base,
		height: g.Height, wavelength: g.Wavelength, fine: g.Fine}, nil
}

// Height returns radial metres at longitude and latitude in degrees.
func (ground *ProceduralHeights) Height(longitude, latitude float64) (float64, bool, error) {
	if ground == nil || math.IsNaN(longitude) || math.IsInf(longitude, 0) ||
		math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 {
		return 0, false, fmt.Errorf("island: invalid procedural ground coordinate")
	}
	lon, lat := math.Remainder(longitude, 360)*maths.Degree, latitude*maths.Degree
	sinLat, cosLat := math.Sincos(lat)
	if math.Abs(latitude) == 90 {
		cosLat = 0
	}
	sinLon, cosLon := math.Sincos(lon)
	p := maths.V(cosLat*cosLon, sinLat, cosLat*sinLon).Scale(ground.radius / ground.wavelength)
	clamp := func(x float64) float64 { return math.Max(-1, math.Min(1, x)) }
	// The lower-frequency field gathers ridges into larger regions; the
	// ridged and fine fields provide distinct shorter wavelengths. Their
	// amplitudes and shaping are deliberately invented for visual direction.
	macro := clamp(noise.Fractal(ground.seed+1, p.X/5, p.Y/5, p.Z/5, 4))
	mask := maths.Smooth(-0.35, 0.25, macro)
	ridge := 1 - math.Abs(clamp(noise.Fractal(ground.seed+2, p.X, p.Y, p.Z, 5)))
	ridges := ridge * ridge * ridge * ridge
	fine := 0.5 + 0.5*clamp(noise.Fractal(ground.seed+3, p.X*4, p.Y*4, p.Z*4, 3))
	return ground.base + ground.height*mask*ridges + ground.fine*mask*fine, true, nil
}
