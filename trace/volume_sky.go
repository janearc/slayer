package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// volumeSkyAt estimates the single-scattering incident source. Uniform sphere
// sampling cancels the normalized isotropic phase, leaving a radiance mean.
// Unlike a surface integral, this has no cosine factor. Shadow queries include
// opaque geometry and transmission through every intervening volume.
func (scene *staged) volumeSkyAt(point maths.Vec) maths.Light {
	if scene.lighting == nil || len(scene.volumeSky) == 0 {
		return maths.Light{}
	}
	var sum maths.Light
	for _, toward := range scene.volumeSky {
		incoming := scene.lighting.Light(toward)
		if incoming == (maths.Light{}) {
			continue
		}
		passed := scene.transmit(point, toward, math.Inf(1))
		sum = sum.Add(incoming.Times(passed))
	}
	return sum.Scale(1 / float64(len(scene.volumeSky)))
}
