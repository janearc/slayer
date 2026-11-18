package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// macroArch sweeps one closed tube over a parabolic load path. Shared rings
// avoid coincident internal caps where independent straight tubes would meet.
type macroArch struct {
	span, rise, radius float64
	segments           int
}

// station gives the centre and a stable transverse axis at one arch station.
func (arch macroArch) station(t float64) (maths.Vec, maths.Vec) {
	x := arch.span * (t - .5)
	centre := maths.V(x, arch.rise*(1-4*x*x/(arch.span*arch.span)), 0)
	tangent := maths.V(1, -8*arch.rise*x/(arch.span*arch.span), 0).Unit()
	return centre, maths.V(tangent.Y, -tangent.X, 0)
}

// skin closes the swept wall only at its two ground-bearing ends.
func (arch macroArch) skin(fine cut) surface {
	rings := make([][]maths.Vec, arch.segments+1)
	for i := range rings {
		centre, normal := arch.station(float64(i) / float64(arch.segments))
		for side := range fine.around {
			angle := 2 * math.Pi * float64(side) / float64(fine.around)
			point := centre.Add(normal.Scale(arch.radius * math.Cos(angle))).Add(maths.V(0, 0,
				-arch.radius*math.Sin(angle)))
			rings[i] = append(rings[i], point)
		}
	}
	var skin surface
	for i := range arch.segments {
		low, high := rings[i], rings[i+1]
		for side, a := range low {
			next := (side + 1) % len(low)
			skin = append(skin, face{a, low[next], high[side]}, face{low[next], high[next], high[side]})
		}
	}
	first, _ := arch.station(0)
	last, _ := arch.station(1)
	for side, p := range rings[0] {
		next := (side + 1) % len(rings[0])
		skin = append(skin, face{first, rings[0][next], p}, face{last,
			rings[len(rings)-1][side], rings[len(rings)-1][next]})
	}
	return skin
}
