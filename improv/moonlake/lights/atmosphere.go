package lights

import "math"

type SkyGlowInput struct {
	Open                                                      []bool
	Rim                                                       []int
	Camera                                                    Camera
	Moon                                                      [3]float64
	Gain, Azimuth, MoonAltitude, MoonScale, MoonLevel, Master float64
}

type StrobeInput struct {
	Open, Near                           []bool
	Rim                                  []int
	Camera                               Camera
	Moon                                 [3]float64
	FarTree                              Vec
	Azimuth, Top, Bottom, Shore, Seconds float64
	White, Magenta                       int
}

// AddSkyGlow adds a blue-white glow to the open sky pixels of sky: strongest
// at the rim, fading upward, plus a halo around the moon when it is in front
// of the camera. It does nothing when Gain is zero or less.
func AddSkyGlow(sky Image, input SkyGlowInput) {
	if input.Gain <= 0 {
		return
	}
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	mx, my, ahead := input.Camera.Project(Toward(input.Azimuth, input.MoonAltitude).Scale(1e6))
	radius := math.Tan(input.MoonScale*0.259*math.Pi/180) * input.Camera.Focal
	colour := [3]float64{0.55, 0.66, 0.95}
	for y := 0; y < sky.Height; y++ {
		for x := 0; x < sky.Width; x++ {
			if !input.Open[y*sky.Width+x] {
				continue
			}
			above := float64(input.Rim[x] - y)
			band := 0.4 + 0.6*math.Exp(-above/(0.18*float64(sky.Height)))
			halo := 0.0
			if ahead {
				d := math.Hypot(float64(x)-mx, float64(y)-my) / radius
				halo = 3 * math.Exp(-math.Max(0, d-1)/3)
			}
			glow := input.Gain * lum * (band + halo*input.MoonLevel) * input.Master
			i := (y*sky.Width + x) * 3
			for c := range 3 {
				sky.Light[i+c] += glow * colour[c]
			}
		}
	}
}

// AddStrobes adds the glow of two far strobes, one white and one magenta, to
// sky. White and Magenta force a strobe on (positive), off (zero) or leave it
// flickering by frame at 12 a second (negative). Light is gathered along the
// line from each pixel to the strobe and stops at blocked pixels. It returns
// 0.5 for each strobe that is lit.
func AddStrobes(sky Image, input StrobeInput) float64 {
	frame := int(math.Round(input.Seconds * 12))
	lights := []struct {
		at     Vec
		colour [3]float64
		seed   uint64
	}{
		{input.FarTree.Add(Toward(input.Azimuth, 0).Scale(160)).Add(Vec{0, 0, 20}),
			[3]float64{1, 1, 1}, 1},
		{input.FarTree.Add(Toward(input.Azimuth, 0).Scale(220)).Add(
			Toward(input.Azimuth-90, 0).Scale(260)).Add(Vec{0, 0, 30}),
			[3]float64{1, 0.25, 0.85}, 2},
	}
	shoreRow := (input.Top - input.Shore) / (input.Top - input.Bottom) * float64(sky.Height)
	total := 0.0
	for _, light := range lights {
		on := Lattice(int64(frame), 0, light.seed) < 0.28
		forced := input.White
		if light.seed == 2 {
			forced = input.Magenta
		}
		if forced == 0 || (forced < 0 && !on) {
			continue
		}
		total += 0.5
		sx, sy, ahead := input.Camera.Project(light.at)
		if !ahead {
			continue
		}
		const steps = 48
		for y := 0; y < sky.Height; y++ {
			for x := 0; x < sky.Width; x++ {
				dx, dy := sx-float64(x), sy-float64(y)
				gathered := 0.0
				jitter := Lattice(int64(x), int64(y), 99)
				for k := 0; k < steps; k++ {
					f := (float64(k) + jitter) / steps
					px := int(float64(x) + dx*f)
					py := int(float64(y) + dy*f)
					if px < 0 || py < 0 || px >= sky.Width || py >= sky.Height {
						continue
					}
					j := py*sky.Width + px
					if !input.Open[j] && !input.Near[j] {
						break
					}
					gathered += 1.0 / steps
				}
				h := Haze(float64(x), float64(y), float64(input.Rim[x]), shoreRow,
					float64(sky.Height), input.Seconds)
				reach := 420 * float64(sky.Width) / 1280
				d2 := (dx*dx + dy*dy) / (reach * reach)
				glow := 1.4 * gathered * h / (1 + d2)
				if !input.Open[y*sky.Width+x] && y < input.Rim[x]+6 {
					glow *= 0.08
				}
				i := (y*sky.Width + x) * 3
				for c := range 3 {
					sky.Light[i+c] += glow * light.colour[c] * ((input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3)
				}
			}
		}
	}
	return total
}

// Haze is how thick the mist is at pixel x, y, never negative. It collects
// just above the rim row and over the lake below shoreRow, and drifts with
// seconds. Its scales follow height, taking 720 rows as the base.
func Haze(x, y, rim, shoreRow, height float64, seconds float64) float64 {
	k := height / 720
	above := rim - y
	behind := 0.0
	if above > -40*k {
		behind = math.Exp(-math.Max(0, above)/(90*k)) * Smooth(-40*k, 0, above)
	}
	lake := 0.0
	if y > shoreRow-30*k {
		lake = Smooth(shoreRow-30*k, shoreRow, y) * (0.5 + 0.5*math.Exp(-(y-shoreRow)/(height-shoreRow)*1.5))
	}
	u := x/(220*k) + 0.1*seconds
	v := y/(60*k) - 0.03*seconds
	wisps := 0.35 + 0.65*Smooth(0.35, 0.75, Curled(u, v, seconds, 7))
	return (behind + lake) * wisps
}

// Curled is drifting noise from 0 to 1 at u, v, made by bending FBM's
// sampling coordinates with two other FBM fields so the pattern swirls.
func Curled(u, v, seconds float64, seed uint64) float64 {
	qx := FBM(u+0.09*seconds, v, seed)
	qy := FBM(u+5.2-0.07*seconds, v+1.3, seed+1)
	return FBM(u+1.9*qx, v+0.8*qy, seed+2)
}

// FBM is the sum of five octaves of Noise, each with twice the frequency and
// half the weight of the one before, scaled back to the range 0 to 1.
func FBM(u, v float64, seed uint64) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for octave := range 5 {
		sum += amp * Noise(u, v, seed+uint64(octave)*7919)
		norm += amp
		u, v, amp = u*2.03, v*2.03, amp*0.5
	}
	return sum / norm
}

// Noise is smooth noise from 0 to 1 at u, v: the four lattice values around
// the point, blended with smoothstep weights. The same seed gives the same
// field.
func Noise(u, v float64, seed uint64) float64 {
	x0, y0 := math.Floor(u), math.Floor(v)
	fx, fy := u-x0, v-y0
	sx, sy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	ix, iy := int64(x0), int64(y0)
	a := Lattice(ix, iy, seed)
	b := Lattice(ix+1, iy, seed)
	c := Lattice(ix, iy+1, seed)
	d := Lattice(ix+1, iy+1, seed)
	return a + (b-a)*sx + (c-a)*sy + (a-b-c+d)*sx*sy
}

// Lattice is a repeatable pseudo-random number from 0 up to but not including
// 1 for the whole-number point x, y and seed. It keeps no state.
func Lattice(x, y int64, seed uint64) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ seed*0x165667B19E3779F9
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	return float64(h>>11) / float64(1<<53)
}

// Smooth is a smooth ramp that is 0 at a, 1 at b and S-shaped between, held at
// 0 and 1 outside. Pass a larger than b for a ramp that falls.
func Smooth(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}
