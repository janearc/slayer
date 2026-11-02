package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// shadowSeed is the seed of the noise that staggers a march toward a lamp, the
// same for every volume so neighbouring points agree.
const shadowSeed = 77

// opticalDepth is the volume's density added up along the way from a point of
// the world toward a lamp far away: density times length, in the medium's own
// units, over shades steps.
//
// It is zero where the way misses the volume, and the march stops at the lamp:
// what lies beyond it is behind the light and takes none.
//
// The samples start part way along the first step, by a smooth noise over the
// point, two wobbles to a step. A coarse march then does not lay its steps down
// as tiers of shadow across a volume, and neighbouring points agree, so there
// is no grain.
func (volume placedVolume) opticalDepth(point, toward maths.Vec,
	far float64) float64 {
	ray := volume.at.RayIn(maths.Ray{At: point, Dir: toward})
	inside, crossed := volume.span(ray, far/volume.scale)
	if !crossed {
		return 0
	}
	step := (inside.leave - inside.enter) / shades
	jitter := volume.stagger(point, step)
	depth := 0.0
	for i := 0; i < shades; i++ {
		at := ray.Point(inside.enter + (float64(i)+jitter)*step)
		depth += volume.med.Density(at) * step
	}
	return depth
}

// wobblesPerStep is how many times the stagger's noise turns over in
// the length of one step toward a lamp.
const wobblesPerStep = 2

// stagger is how far into its first step a march toward a lamp starts, from
// zero to one, by a smooth noise over the point of the world.
//
// step is the step's length in the medium's own units, which in the world is
// that times the volume's scale, so the noise turns over wobblesPerStep times
// in a step whatever the volume's size.
func (volume placedVolume) stagger(point maths.Vec, step float64) float64 {
	wobble := wobblesPerStep / (step * volume.scale)
	return 0.5 + 0.5*noise.Gradient(shadowSeed, point.X*wobble,
		point.Y*wobble, point.Z*wobble)
}

// shone is how much lamplight reaches a point in the world, as one
// number: the luminance of each lamp's light after the surfaces,
// volumes and gases in its way have taken their share of each channel.
func shone(scene staged, point, facing maths.Vec) float64 {
	sum := 0.0
	for _, lamp := range scene.lamps {
		sum += scene.reaching(lamp, point, facing).light.Luminance()
	}
	return sum
}

// paintedClear is how much of a lamp's light reaches a point in the
// world through every painted volume between: e to the minus their
// optical depths toward the lamp, added up. A gas's density is a
// share and not an extinction, so gas volumes are gasClear's.
//
// a tinted volume's depth is its tint's times in each channel. with
// none tinted, every channel is the one number it always was.
func paintedClear(volumes []placedVolume, point, toward maths.Vec,
	far float64) maths.Light {
	depth, tinted, any := 0.0, maths.Light{}, false
	for _, volume := range volumes {
		if volume.gas != nil {
			continue
		}
		thick := volume.opticalDepth(point, toward, far)
		if volume.tint == nil {
			depth += thick
			continue
		}
		tinted, any = tinted.Add(volume.tint.Scale(thick)), true
	}
	if !any {
		return maths.Grey(math.Exp(-depth))
	}
	return maths.RGB(math.Exp(-depth-tinted.R), math.Exp(-depth-tinted.G),
		math.Exp(-depth-tinted.B))
}
