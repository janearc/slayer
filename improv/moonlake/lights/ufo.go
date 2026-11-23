package lights

import "math"

type UFOConfig struct {
	Out, Radius, Apex, Gain, Halo float64
	Azimuth, WaterZ               float64
}

type UFOCues struct {
	Overhead float64
}

type UFOInput struct {
	Trees, Water Image
	Camera       Camera
	Moon         [3]float64
	Seconds      float64
	Cues         UFOCues
}

// AddUFOBeam paints the UFO's beam of light into sky, and into front where the
// water mask is set. It makes the tree and water masks from input.Trees and
// input.Water, then calls addUFOBeam.
func AddUFOBeam(sky, front Image, config UFOConfig, input UFOInput) {
	addUFOBeam(sky, front, config, input, classifyBeamMask(input.Trees), classifyBeamMask(input.Water))
}

// classifyBeamMask marks the pixels of image whose channel mean is over 0.3
// times the largest channel value. It returns nil for an image with no Light
// and all false for one that is all zero.
func classifyBeamMask(image Image) []bool {
	if image.Light == nil {
		return nil
	}
	most := 0.0
	for _, value := range image.Light {
		most = math.Max(most, value)
	}
	mask := make([]bool, image.Width*image.Height)
	if most == 0 {
		return mask
	}
	for i := range mask {
		j := i * 3
		mask[i] = (image.Light[j]+image.Light[j+1]+image.Light[j+2])/3 > 0.3*most
	}
	return mask
}

// addUFOBeam paints the beam for each pixel not under treeMask: a halo near
// the foot, three nested cone shells in white-blue, red and violet lit by the
// fog, and a bright patch where the cone meets the water. Everything is scaled
// by the overhead cue and the moon. A nil mask means no pixels.
func addUFOBeam(sky, front Image, config UFOConfig, input UFOInput, treeMask, waterMask []bool) {
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	axis := Toward(config.Azimuth, 0).Scale(config.Out)
	apex := axis.Add(Vec{0, 0, config.WaterZ + config.Apex})
	slope := config.Radius / config.Apex
	layers := []struct {
		slope  float64
		colour [3]float64
		gain   float64
	}{
		{slope, [3]float64{0.82, 0.9, 1.0}, 1},
		{slope * 1.07, [3]float64{1, 0.12, 0.08}, 0.5},
		{slope * 1.16, [3]float64{0.55, 0.15, 1}, 0.3},
	}
	cam, seconds := input.Camera, input.Seconds
	on := func(mask []bool, x, y int) bool {
		return mask != nil && mask[y*sky.Width+x]
	}
	footX, footY, _ := cam.Project(axis.Add(Vec{0, 0, config.WaterZ}))
	headX, headY, _ := cam.Project(apex)
	edgeX, _, _ := cam.Project(axis.Add(Vec{0, 0, config.WaterZ}).Add(Toward(config.Azimuth+90, 0).Scale(config.Radius)))
	footHalf := math.Max(1, math.Abs(edgeX-footX))
	for y := 0; y < sky.Height; y++ {
		for x := 0; x < sky.Width; x++ {
			if on(treeMask, x, y) {
				continue
			}
			if below := float64(y) - footY; below < 4*footHalf {
				fade := 1.0
				if below > 0 {
					fade = math.Exp(-below / (0.5 * footHalf))
				}
				f := 0.0
				if headY != footY {
					f = math.Max(0, (float64(y)-footY)/(headY-footY))
				}
				ax := footX + f*(headX-footX)
				halfWidth := math.Max(1, footHalf*(1-f))
				across := math.Abs(float64(x)-ax)/halfWidth - 1
				if below > 0 && across < 0 {
					across = 0
				}
				if across > 0 || below > 0 {
					u := float64(x)/(60*float64(sky.Width)/1280) + 0.2*seconds
					v := float64(y)/(25*float64(sky.Width)/1280) - 0.1*seconds
					wisps := 0.4 + 0.6*Smooth(0.3, 0.75, Curled(u, v, seconds, 29))
					radius := config.Radius * (1 - f)
					halo := config.Halo * config.Gain * input.Cues.Overhead * lum *
						2 * radius * wisps * math.Exp(-across/0.9) * fade
					i := sky.At(x, y)
					wet := on(waterMask, x, y)
					for c := range 3 {
						tint := [3]float64{0.7, 0.8, 1.0}[c]
						sky.Light[i+c] += halo * tint
						if wet {
							front.Light[i+c] += halo * tint
						}
					}
				}
			}
			d := cam.Ray(x, y)
			hitWater := math.Inf(1)
			if d.Z < 0 {
				hitWater = config.WaterZ / d.Z
			}
			var light [3]float64
			inner := 0.0
			for _, layer := range layers {
				in, out, crosses := coneChord(d, apex, layer.slope, config.WaterZ)
				if !crosses || in >= hitWater {
					continue
				}
				out = math.Min(out, hitWater)
				mist := 0.0
				for _, f := range []float64{0.2, 0.5, 0.8} {
					p := d.Scale(in + f*(out-in))
					mist += (0.35 + 6*laserFog(p, seconds, config.WaterZ)) / 3
				}
				length := out - in
				shell := length - inner
				inner = length
				if shell <= 0 {
					continue
				}
				amount := config.Gain * input.Cues.Overhead * lum * mist * shell * layer.gain
				for c := range 3 {
					light[c] += amount * layer.colour[c]
				}
			}
			if !math.IsInf(hitWater, 1) {
				p := d.Scale(hitWater)
				r := math.Hypot(p.X-axis.X, p.Y-axis.Y)
				if r < config.Radius*1.3 {
					edge := Smooth(config.Radius*1.3, config.Radius*0.9, r)
					amount := config.Gain * input.Cues.Overhead * lum * 60 * edge
					for c := range 3 {
						light[c] += amount * layers[0].colour[c]
					}
				}
			}
			if light == [3]float64{} {
				continue
			}
			i := sky.At(x, y)
			wet := on(waterMask, x, y)
			for c := range 3 {
				sky.Light[i+c] += light[c]
				if wet {
					front.Light[i+c] += light[c]
				}
			}
		}
	}
}
