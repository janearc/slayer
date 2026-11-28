package hair

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// GuideWave adds a bounded oscillation to the free part of a guide.
// Width and Height are fractions of the shaft's length. Turns is the
// number of cycles along the guide and Phase is in radians. The
// oscillation ramps in beyond the root and eases toward the tip.
type GuideWave struct {
	Width, Height, Turns, Phase float64
}

func (wave GuideWave) valid() bool {
	return wave.Width >= 0 && wave.Width <= .18 &&
		wave.Height >= 0 && wave.Height <= .18 &&
		wave.Turns >= 0 && wave.Turns <= 3 && finite(wave.Phase)
}

func (wave GuideWave) offset(t float64) maths.Vec {
	if wave.Width == 0 && wave.Height == 0 {
		return maths.Vec{}
	}
	envelope := maths.Smooth(.03, .44, t) * (1 - .48*maths.Smooth(.70, 1, t))
	sine, cosine := math.Sincos(2*math.Pi*wave.Turns*t + wave.Phase)
	return maths.V(0, envelope*wave.Height*cosine, envelope*wave.Width*sine)
}
