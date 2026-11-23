package lights

import "math"

type LaserConfig struct {
	Azimuth, EyeOverLake, RigDeck, WaterZ, Reach float64
	FarTree, ShoreLeft, ShoreRight               Vec
}

type laserCues struct {
	Rig, Sweep, Gate, GateLeft, GateRight, FigurePhase float64
	Beams, Parity                                      int
	Colour                                             [3]float64
	HasColour                                          bool
	Figure                                             string
}

type LaserInput struct {
	Open, CloseTrees  []bool
	Rim               []int
	Camera            Camera
	Moon              [3]float64
	Seconds, ShoreRow float64
	Cues              laserCues
}

type laserHead struct {
	Origin            Vec
	Aim, Lift, Mirror float64
	Flip, Bank        int
}

// laserOnDeck moves head to the rig deck. When config.RigDeck is positive it
// sets the height to the ground height of tower, 0 to 3, relative to the eye,
// plus RigDeck. Otherwise it returns head as given.
func laserOnDeck(head Vec, tower int, config LaserConfig) Vec {
	if config.RigDeck <= 0 {
		return head
	}
	rigGround := [4]float64{1180.2, 1191.3, 1076.1, 1079.4}
	head.Z = rigGround[tower] - (1076 + config.EyeOverLake) + config.RigDeck
	return head
}

// laserHeads returns the four laser heads: two beside the far tree, 120 out to
// each side of the view and 6 up, aiming back at the viewer, and two at the
// shore points aiming at each other, tilted up 7 degrees. Heads 0 and 2 belong
// to bank 0 and heads 1 and 3 to bank 1. Mirror makes a pair swing opposite
// ways.
func laserHeads(config LaserConfig) [4]laserHead {
	return [4]laserHead{
		{laserOnDeck(config.FarTree.Add(Toward(config.Azimuth-90, 0).Scale(120)).Add(Vec{0, 0, 6}), 0, config), config.Azimuth + 180, 0, 1, 0, 0},
		{laserOnDeck(config.FarTree.Add(Toward(config.Azimuth+90, 0).Scale(120)).Add(Vec{0, 0, 6}), 1, config), config.Azimuth + 180, 0, -1, 1, 1},
		{laserOnDeck(config.ShoreLeft, 2, config), Bearing(config.ShoreLeft, config.ShoreRight), 7, 1, 1, 0},
		{laserOnDeck(config.ShoreRight, 3, config), Bearing(config.ShoreRight, config.ShoreLeft), 7, -1, 0, 1},
	}
}

// addLasers draws every lit laser head's beams and opening into sky and front.
// It does nothing when Rig is 0 or both gates are under 0.01. The colour is
// green unless HasColour, scaled by Rig when Rig is positive; GateLeft and
// GateRight, when not negative, replace Gate for their bank. Swing, tilt and
// spread follow Seconds, or Sweep, a phase, when that is not negative.
func addLasers(sky, front Image, config LaserConfig, input LaserInput) {
	heads := laserHeads(config)
	cue := input.Cues
	colour := [3]float64{0.08, 1, 0.12}
	if cue.Rig == 0 {
		return
	}
	if cue.HasColour {
		colour = cue.Colour
	}
	if cue.Rig > 0 {
		for c := range colour {
			colour[c] *= cue.Rig
		}
	}
	gates := [2]float64{cue.Gate, cue.Gate}
	if cue.GateLeft >= 0 {
		gates[0] = cue.GateLeft
	}
	if cue.GateRight >= 0 {
		gates[1] = cue.GateRight
	}
	if gates[0] < 0.01 && gates[1] < 0.01 {
		return
	}
	swing := 45 * math.Sin(input.Seconds*1.7)
	tilt := -4.5 + 2.2*math.Sin(input.Seconds*1.1+0.6)
	spread := 22 + 14*math.Sin(input.Seconds*0.9)
	if cue.Sweep >= 0 {
		w := 2 * math.Pi * cue.Sweep
		swing = 45 * math.Sin(w)
		tilt = -4.5 + 2.2*math.Sin(2*w+0.6)
		spread = 22 + 14*math.Sin(3*w)
	}
	for _, head := range heads {
		gate := gates[head.Bank]
		if gate < 0.01 {
			continue
		}
		lit := colour
		for c := range lit {
			lit[c] *= gate
		}
		heading := head.Aim + head.Mirror*swing
		ways, weight, split := figureWays(cue.Figure, heading, tilt+head.Lift, spread, cue.Beams, cue.FigurePhase)
		for b, way := range ways {
			if split && cue.Parity >= 0 && (b+head.Flip)%2 != cue.Parity {
				continue
			}
			var each [3]float64
			for c := range each {
				each[c] = lit[c] * weight
			}
			addLaserBeam(sky, front, config, input, head.Origin, way, each)
		}
		addLaserAperture(sky, front, input, head.Origin, lit)
	}
}

// figureWays returns the beam directions for the named figure, the brightness
// weight for each, and whether addLasers may thin them by Parity. Figures are
// beam, pair, fan64, scan, cone and wave; any other name gives beams beams
// spread across spread degrees. Angles are in degrees; phase turns cone and
// wave.
func figureWays(figure string, heading, tilt, spread float64, beams int, phase float64) ([]Vec, float64, bool) {
	across := func(n int, wide float64, lift func(f float64) float64) []Vec {
		ways := make([]Vec, n)
		for b := range ways {
			f := 0.0
			if n > 1 {
				f = float64(b)/float64(n-1) - 0.5
			}
			ways[b] = Toward(heading+f*wide, tilt+lift(f))
		}
		return ways
	}
	level := func(float64) float64 { return 0 }
	switch figure {
	case "beam":
		return across(1, 0, level), 1.8, false
	case "pair":
		return []Vec{Toward(heading-spread/4, tilt), Toward(heading+spread/4, tilt)}, 1.4, false
	case "fan64":
		ways := make([]Vec, 0, 64)
		for row := range 8 {
			up := 6 * (float64(row)/7 - 0.5)
			ways = append(ways, across(8, spread*1.3, func(float64) float64 { return up })...)
		}
		return ways, 0.45, true
	case "scan":
		const sweeps = 48
		return across(sweeps, spread*1.4, level), 5.0 / sweeps, false
	case "cone":
		const round, half = 40, 7 * math.Pi / 180
		axis := Toward(heading, tilt)
		u := axis.Cross(Vec{0, 0, 1}).Unit()
		v := u.Cross(axis)
		ways := make([]Vec, round)
		for k := range ways {
			a := 2*math.Pi*float64(k)/round + 2*math.Pi*phase
			ring := u.Scale(math.Cos(a)).Add(v.Scale(math.Sin(a)))
			ways[k] = axis.Scale(math.Cos(half)).Add(ring.Scale(math.Sin(half)))
		}
		return ways, 5.0 / round, false
	case "wave":
		return across(16, spread, func(f float64) float64 { return 3 * math.Sin(2*math.Pi*(1.5*f+phase)) }), 0.9, false
	}
	return across(beams, spread, level), 1, true
}

// addLaserAperture draws the bright spot where a laser leaves its head at at:
// a tight core and a wider glow in colour. Nothing is drawn when the head is
// behind the camera, off the picture or under CloseTrees.
func addLaserAperture(sky, front Image, input LaserInput, at Vec, colour [3]float64) {
	cam := input.Camera
	x, y, ahead := cam.Project(at)
	if !ahead || x < 0 || y < 0 || x >= cam.Width || y >= cam.Height {
		return
	}
	if input.CloseTrees != nil && input.CloseTrees[int(y)*sky.Width+int(x)] {
		return
	}
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	k := cam.Width / 1280
	most := math.Max(colour[0], math.Max(colour[1], colour[2]))
	core := [3]float64{}
	for c := range core {
		core[c] = 0.5*colour[c] + 0.5*most
	}
	Splat(sky, front, x, y, 0.8*k, lum*40*k, core, input.ShoreRow)
	Splat(sky, front, x, y, 5*k, lum*25*k, colour, input.ShoreRow)
}

// laserFog is how thick the fog is at p, from 0 to 1. It is 0 below waterZ and
// above that thins with height, broken into blobs by noise that drifts with
// seconds.
func laserFog(p Vec, seconds, waterZ float64) float64 {
	above := p.Z - waterZ
	if above < 0 {
		return 0
	}
	layer := math.Exp(-above / 90)
	blobs := fbm3(p.X/45+0.6*seconds, p.Y/45-0.2*seconds, p.Z/18+0.1*seconds, 17)
	return layer * (0.03 + 0.97*Smooth(0.45, 0.7, blobs))
}

// fbm3 is the three-dimensional form of FBM: four octaves of noise3, scaled
// back to the range 0 to 1.
func fbm3(u, v, w float64, seed uint64) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for octave := range 4 {
		sum += amp * noise3(u, v, w, seed+uint64(octave)*7919)
		norm += amp
		u, v, w, amp = u*2.03, v*2.03, w*2.03, amp*0.5
	}
	return sum / norm
}

// noise3 is smooth noise from 0 to 1 at u, v, w: the eight lattice values
// around the point, blended with smoothstep weights.
func noise3(u, v, w float64, seed uint64) float64 {
	x0, y0, z0 := math.Floor(u), math.Floor(v), math.Floor(w)
	fx, fy, fz := u-x0, v-y0, w-z0
	sx, sy, sz := fx*fx*(3-2*fx), fy*fy*(3-2*fy), fz*fz*(3-2*fz)
	ix, iy, iz := int64(x0), int64(y0), int64(z0)
	at := func(dx, dy, dz int64) float64 { return Lattice(ix+dx, iy+dy*7349+(iz+dz)*104729, seed) }
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	bottom := lerp(lerp(at(0, 0, 0), at(1, 0, 0), sx), lerp(at(0, 1, 0), at(1, 1, 0), sx), sy)
	top := lerp(lerp(at(0, 0, 1), at(1, 0, 1), sx), lerp(at(0, 1, 1), at(1, 1, 1), sx), sy)
	return lerp(bottom, top, sz)
}

// addLaserBeam draws one beam from origin along dir, stepping out to
// config.Reach in steps of half a pixel's width. Each step adds two blurred
// spots lit by the fog there. Steps that are off the picture or hidden by the
// rim (beyond 80) or by CloseTrees are skipped. At the water it adds one last
// spot and stops.
func addLaserBeam(sky, front Image, config LaserConfig, input LaserInput, origin, dir Vec, colour [3]float64) {
	s := 0.0
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	cam := input.Camera
	for s < config.Reach {
		p := origin.Add(dir.Scale(s))
		if p.Z < config.WaterZ {
			if x, y, ahead := cam.Project(p); ahead {
				Splat(sky, front, x, y, 1.2, lum*60*laserFog(p, input.Seconds, config.WaterZ), colour, input.ShoreRow)
			}
			return
		}
		x, y, ahead := cam.Project(p)
		dist := math.Sqrt(p.Dot(p))
		step := math.Max(0.05, 0.5*dist/cam.Focal)
		s += step
		if !ahead || x < -4 || y < -4 || x >= cam.Width+4 || y >= cam.Height+4 {
			continue
		}
		xi, yi := int(x), int(y)
		if xi >= 0 && yi >= 0 && xi < sky.Width && yi < sky.Height && !input.Open[yi*sky.Width+xi] && yi < input.Rim[xi] && dist > 80 {
			continue
		}
		if input.CloseTrees != nil && xi >= 0 && yi >= 0 && xi < sky.Width && yi < sky.Height && input.CloseTrees[yi*sky.Width+xi] {
			continue
		}
		bright := lum * (0.12 + 24*laserFog(p, input.Seconds, config.WaterZ)) * 2.2 * math.Min(1, float64(sky.Width)/1280)
		pixels := step / dist * cam.Focal
		Splat(sky, front, x, y, 0.5, bright*pixels*0.88, colour, input.ShoreRow)
		Splat(sky, front, x, y, 1.6, bright*pixels*0.12, colour, input.ShoreRow)
	}
}
