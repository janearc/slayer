package flame

import (
	"errors"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// Sampled is a flame's immutable density grid at one time.
// Spacing is chosen in local units from the image's pixel footprint.
// Its temperature, tint and emission remain the original flame's.
type Sampled struct {
	body       *Body
	bounds     maths.Box
	spacing    float64
	nx, ny, nz int
	values     []float64
}

// mostSamples bounds the memory of a requested density grid.
const mostSamples = 1 << 20

// NewSampled caches density at spacing or finer, with four intervals
// across the smallest extent. It refuses invalid or oversized grids.
func NewSampled(body *Body, spacing float64) (*Sampled, error) {
	if body == nil || !sane(spacing) {
		return nil, errors.New(
			"flame: invalid sampled body or spacing")
	}
	box := body.Bounds()
	extent := box.Max.Sub(box.Min)
	narrow := math.Min(extent.X, math.Min(extent.Y, extent.Z))
	spacing = math.Min(spacing, narrow/4)
	x := math.Ceil(extent.X/spacing) + 1
	y := math.Ceil(extent.Y/spacing) + 1
	z := math.Ceil(extent.Z/spacing) + 1
	if !(x >= 2 && y >= 2 && z >= 2 && x*y*z <= mostSamples) {
		return nil, errors.New(
			"flame: density grid exceeds its sample budget")
	}
	sampled := &Sampled{body: body, bounds: box, spacing: spacing,
		nx: int(x), ny: int(y), nz: int(z)}
	sampled.fill()
	return sampled, nil
}

// fill evaluates each grid point once before any ray can read it.
func (s *Sampled) fill() {
	s.values = make([]float64, s.nx*s.ny*s.nz)
	var workers sync.WaitGroup
	count := min(runtime.GOMAXPROCS(0), s.nz)
	for worker := range count {
		workers.Go(func() {
			for z := worker; z < s.nz; z += count {
				s.fillPlane(z)
			}
		})
	}
	workers.Wait()
}

// fillPlane writes one disjoint slice of the density grid.
func (s *Sampled) fillPlane(z int) {
	for y := range s.ny {
		for x := range s.nx {
			offset := maths.V(float64(x), float64(y), float64(z))
			p := s.bounds.Min.Add(offset.Scale(s.spacing))
			index := x + s.nx*(y+s.ny*z)
			s.values[index] = s.body.Density(p)
		}
	}
}

// Bounds remains the flame's conservative extent.
func (s *Sampled) Bounds() maths.Box { return s.bounds }

// Tint preserves wavelength-dependent soot absorption.
func (s *Sampled) Tint() maths.Light { return s.body.Tint() }

// Heat preserves the original temperature reported by a ray replay.
func (s *Sampled) Heat(p maths.Vec) measure.Temperature {
	return s.body.Heat(p)
}

// Tick creates a separate grid; it never changes a frame being rendered.
func (s *Sampled) Tick(at time.Duration) medium.Medium {
	if at == s.body.at {
		return s
	}
	next := *s
	next.body = s.body.Tick(at).(*Body)
	next.fill()
	return &next
}

// at reads a validated grid index.
func (s *Sampled) at(x, y, z int) float64 { return s.values[x+s.nx*(y+s.ny*z)] }

// densityMix interpolates density along one grid axis.
func densityMix(t, a, b float64) float64 { return a + t*(b-a) }

// Density interpolates cached values and never evaluates noise per ray.
func (s *Sampled) Density(p maths.Vec) float64 {
	b := s.bounds
	if !(p.X >= b.Min.X && p.Y >= b.Min.Y && p.Z >= b.Min.Z &&
		p.X <= b.Max.X && p.Y <= b.Max.Y && p.Z <= b.Max.Z) {
		return 0
	}
	u := p.Sub(b.Min).Scale(1 / s.spacing)
	x, y, z := min(int(u.X), s.nx-2),
		min(int(u.Y), s.ny-2), min(int(u.Z), s.nz-2)
	a, beta, c := u.X-float64(x), u.Y-float64(y), u.Z-float64(z)
	low := densityMix(beta,
		densityMix(a, s.at(x, y, z), s.at(x+1, y, z)),
		densityMix(a, s.at(x, y+1, z), s.at(x+1, y+1, z)))
	high := densityMix(beta,
		densityMix(a, s.at(x, y, z+1), s.at(x+1, y, z+1)),
		densityMix(a, s.at(x, y+1, z+1), s.at(x+1, y+1, z+1)))
	return densityMix(c, low, high)
}
