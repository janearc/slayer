package hair

import (
	"math"

	"github.com/janearc/slayer/maths/noise"
)

// FollicleUnits places nearby roots around a shared candidate. Spread
// is the maximum offset in metres. Density still counts individual
// shafts. The zero value places independent roots.
type FollicleUnits struct {
	Size   int
	Spread float64
}

func (units FollicleUnits) valid() bool {
	return units == (FollicleUnits{}) ||
		(units.Size >= 2 && units.Size <= 4 && units.Spread > 0 && finite(units.Spread))
}

func (units FollicleUnits) member(center [3]float64, basis triangle, seed uint64, face, sample, index int) [3]float64 {
	if index == 0 {
		return center
	}
	seed ^= 0x666f6c6c69636c65
	angle := 2 * math.Pi * noise.Hash3(seed, int64(face), int64(sample), 0)
	radius := units.Spread * math.Sqrt(noise.Hash3(seed, int64(face), int64(sample), 1))
	sine, cosine := math.Sincos(angle)
	along := basis.first.Unit()
	area := basis.area()
	normal := area.Unit()
	size := area.Len()
	across := normal.Cross(along)
	offset := along.Scale(radius * cosine).Add(across.Scale(radius * sine))
	u := offset.Cross(basis.second).Dot(normal) / size
	v := basis.first.Cross(offset).Dot(normal) / size
	if !finite(u) || !finite(v) || !finite(u+v) {
		return center
	}
	delta := [3]float64{-u - v, u, v}
	// Shorten the displacement at a triangle edge so every member
	// can keep its own barycentric attachment on this face.
	scale := 1.
	for i, change := range delta {
		if change < -center[i] {
			scale = min(scale, .90*center[i]/-change)
		}
	}
	for i, change := range delta {
		center[i] += scale * change
	}
	return center
}
