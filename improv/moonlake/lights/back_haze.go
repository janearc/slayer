package lights

import "math"

type BackHazeInput struct {
	Camera        Camera
	Moon          [3]float64
	Positions     [2]Vec
	Levels        [2]float64
	Gain, Seconds float64
}

// backlights returns the positions of the two lights behind the trees, as
// offsets from the eye. Only their height depends on eyeOverLake, the eye's
// height over the lake. The first goes with white, the second with magenta.
func backlights(eyeOverLake float64) [2]Vec {
	z := 1126.5 - (1076 + eyeOverLake)
	return [2]Vec{{46.4, -3.8, z}, {38.3, -26.4, z}}
}

// AddBackHaze adds mist lit from the back lights to front. A trees image with
// no light in it means no trees and nothing is added. Pixels covered by trees
// are left alone.
func AddBackHaze(front, trees Image, input BackHazeInput) {
	most := 0.0
	for _, v := range trees.Light {
		most = math.Max(most, v)
	}
	if most == 0 {
		return
	}
	mask, _ := coveredBy(trees)
	addBackHaze(front, mask, input)
}

// addBackHaze adds mist lit from each back light to front. A light is skipped
// when its level is under 0.01 or it is behind the camera. Each pixel not in
// mask gathers light along the line to the light, which stops at a tree pixel.
func addBackHaze(front Image, mask []bool, input BackHazeInput) {
	tree := func(x, y int) bool { return mask[y*front.Width+x] }
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	colours := [2][3]float64{{1, 1, 1}, {1, 0.25, 0.85}}
	for side, at := range input.Positions {
		level := input.Levels[side]
		if level < 0.01 {
			continue
		}
		sx, sy, ahead := input.Camera.Project(at)
		if !ahead {
			continue
		}
		const steps = 32
		for y := 0; y < front.Height; y++ {
			for x := 0; x < front.Width; x++ {
				if tree(x, y) {
					continue
				}
				dx, dy := sx-float64(x), sy-float64(y)
				jitter := Lattice(int64(x), int64(y), 41)
				gathered := 0.0
				for k := 0; k < steps; k++ {
					f := (float64(k) + jitter) / steps
					px, py := int(float64(x)+dx*f), int(float64(y)+dy*f)
					if px < 0 || py < 0 || px >= front.Width || py >= front.Height {
						gathered += 1.0 / steps
						continue
					}
					if tree(px, py) {
						break
					}
					gathered += 1.0 / steps
				}
				scale := float64(front.Width) / 1280
				u := float64(x)/(240*scale) + 0.15*input.Seconds
				v := float64(y)/(90*scale) - 0.05*input.Seconds
				mist := 0.35 + 0.65*Smooth(0.35, 0.75, Curled(u, v, input.Seconds, 23))
				reach := 0.25 * float64(front.Width)
				d2 := (dx*dx + dy*dy) / (reach * reach)
				glow := input.Gain * lum * level * gathered * mist / (1 + d2)
				i := front.At(x, y)
				for c := range 3 {
					front.Light[i+c] += glow * colours[side][c]
				}
			}
		}
	}
}
