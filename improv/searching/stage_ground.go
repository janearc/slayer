package searching

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
	"math"
	"time"
)

type stageLayer struct {
	field  *medium.Layer
	volume *thing.Volume
}
type stageBeam struct {
	origin, pivot, at maths.Vec
	lamp              int
	exit              maths.Subspace
}

// makeGround uses the Cloudsea source terrain for Hawaii, with the new
// instruments. Chile and Los Angeles have explicitly authored ground.
func (s *heardStage) makeGround(p *stagePlan) error {
	s.ground = groundHeight
	summit := false
	if p.Terrain != "" {
		raster, err := terrain.Open(p.Terrain)
		if err != nil {
			return err
		}
		defer raster.Close()
		if raster.Reference().Model == 2 {
			if err := s.maunaKeaGround(p, raster); err != nil {
				return err
			}
			summit = true
		} else {
			window, err := raster.Window(263050, 2149520, 129, 129, 12.5)
			if err != nil {
				return err
			}
			ground, low, err := window.Ground()
			if err != nil {
				return err
			}
			zero := window.At(32, 35)
			s.world.Add(thing.Thing{Shape: ground, Material: &thing.Material{Name: "USGS Kilauea Cloudsea ground", Colour: maths.RGB(.09, .105, .08)}, Motion: thing.Still(maths.NewSubspace(maths.V(-400, low-zero, -437.5), maths.Identity, 1))})
			s.ground = func(x, z float64) float64 {
				a, b := math.Max(0, math.Min(127.999, (x+400)/12.5)), math.Max(0, math.Min(127.999, (z+437.5)/12.5))
				i, j := int(a), int(b)
				a -= float64(i)
				b -= float64(j)
				return (window.At(i, j)*(1-a)+window.At(i+1, j)*a)*(1-b) + (window.At(i, j+1)*(1-a)+window.At(i+1, j+1)*a)*b - zero
			}
		}
	} else if p.Kind == "owl" {
		if err := s.atacamaSite(); err != nil {
			return err
		}
	} else if p.Kind == "griffith" {
		if err := s.griffithHill(); err != nil {
			return err
		}
	} else {
		const n = 65
		h := make([]float64, n*n)
		for z := range n {
			for x := range n {
				h[z*n+x] = s.ground(float64(x)*12.5-400, float64(z)*12.5-400)
			}
		}
		ground, err := shape.NewHeightField(h, n, 800)
		if err != nil {
			return err
		}
		s.world.Add(thing.Thing{Shape: ground, Material: &thing.Material{Name: "authored Chile or Los Angeles relief", Colour: maths.RGB(.18, .12, .08)}})
	}
	// Low warm Moon, two interior lamps, and at most four beams fit the
	if !summit && p.Kind != "griffith" && p.Kind != "owl" {
		if err := s.distantGround(p.Terrain != ""); err != nil {
			return err
		}
	}
	// fastlight store's eight-lamp bound. Index zero is the common sky lamp.
	s.world.Lamp(film.Sun{})
	switch p.Kind {
	case "kecks":
		for i, name := range []string{"keck-i", "keck-ii"} {
			x, z := float64(i)*52-26, 0.0
			if summit {
				x = []float64{-28.669, 22.005}[i]
				z = []float64{18.299, -49.972}[i]
			}
			if err := s.groundRig(name, maths.V(x, s.ground(x, z), z), p); err != nil {
				return err
			}
		}
		if summit {
			if err := s.keckCampus(p); err != nil {
				return err
			}
		}
	case "subaru", "rubin", "elt":
		if err := s.groundRig(p.Kind, maths.V(0, s.ground(0, 0), 0), p); err != nil {
			return err
		}
		if p.Kind == "rubin" {
			if err := s.enrichRubin(); err != nil {
				return err
			}
		}
		if p.Kind == "elt" {
			if err := s.enrichELT(maths.V(0, s.ground(0, 0), 0)); err != nil {
				return err
			}
		}
	case "owl":
		if err := s.makeOWL(); err != nil {
			return err
		}
	case "griffith":
		if err := s.griffith(); err != nil {
			return err
		}
	case "compact-radio", "spread-radio":
		for i := range 12 {
			a := float64(i) * 2.399963229728653
			radius := 18 * math.Sqrt(float64(i))
			if p.Kind == "spread-radio" {
				radius *= 2.2
			}
			x, z := radius*math.Cos(a), radius*math.Sin(a)
			if err := s.radioRig(maths.V(x, s.ground(x, z), z)); err != nil {
				return err
			}
		}
	}
	if !p.Interior && !summit && p.Kind != "griffith" {
		temp := &set{World: s.world, prototypes: map[string][]thing.Thing{}}
		for i, name := range []string{"service-truck", "mobile-crane", "scaffold-tower"} {
			x, z := -45+float64(i)*43, 33.0
			if err := temp.scenery(name, uint64(i+5), maths.V(x, s.ground(x, z), z), 1); err != nil {
				return err
			}
		}
		if p.Terrain != "" && !summit {
			for i := range 700 {
				a := float64(i) * 2.399963229728653
				radius := 65 + 14*math.Sqrt(float64(i))
				x, z := radius*math.Cos(a), radius*math.Sin(a)
				if z < 40 && math.Abs(x) < 85 {
					continue
				}
				if err := temp.scenery("coast-redwood", 77, maths.V(x, s.ground(x, z), z), 1.5+.5*math.Sin(float64(i)*1.9)); err != nil {
					return err
				}
			}
			for _, at := range []maths.Vec{maths.V(20, 0, -145), maths.V(155, 0, -30)} {
				at.Y = s.ground(at.X, at.Z)
				if err := temp.scenery("coast-redwood", 77, at, 1.7); err != nil {
					return err
				}
			}
			if err := temp.scenery("boulder", 7, maths.V(58, s.ground(58, -108), -108), 3); err != nil {
				return err
			}
		}
	}
	if summit {
		if err := s.summitScenery(p); err != nil {
			return err
		}
	}
	// Cloudsea-inspired three layers; their local heights follow this
	// source window rather than an unrelated flat floor.
	for i, row := range [][4]float64{{.008, 95, .55, -28}, {.003, 70, .35, 35}, {.00012, 500, 1, -80}} {
		if p.Starlight || summit || p.Kind == "griffith" {
			continue
		}
		if p.Interior && i < 2 {
			continue
		}
		if p.Terrain == "" && i < 2 {
			row[0] *= .22
		}
		weather := medium.NewWeather(row[0], row[1], row[2], uint64(23+i), maths.V(2, 0, -1))
		weather.PatchScale = 130
		base := row[3]
		field, err := medium.NewLayer(weather, maths.Box{Min: maths.V(-400, -100, -400), Max: maths.V(650, 500, 650)}, func(x, z float64) float64 { return s.ground(x, z) + base })
		if err != nil {
			return err
		}
		v := s.world.AddVolume(thing.Volume{Medium: field, Material: &thing.Material{Name: "warm layered Cloudsea proof mist", Colour: maths.RGB(.95, .83, .65)}})
		s.layers = append(s.layers, stageLayer{field, v})
	}
	if p.Burst {
		for i := 0; i < 3; i++ {
			s.burstLamps = append(s.burstLamps, len(s.world.Lamps))
			s.world.Lamp(film.Sun{})
		}
	}
	return nil
}

// groundRig adds the named ground rig at at, with its beams and lamps.
func (s *heardStage) groundRig(name string, at maths.Vec, p *stagePlan) error {
	rig, err := manufacture.MakeGroundRigWithLaunchers(name, "demo")
	if err != nil {
		return err
	}
	for role, parts := range [][]thing.Thing{rig.Fixed, rig.Azimuth, rig.Optical, rig.Enclosure} {
		object := stageObject{at: at, pivot: rig.Pivot, role: role, scale: 1}
		for _, part := range parts {
			part.Motion = thing.Still(maths.NewSubspace(at, maths.Identity, 1))
			object.things = append(object.things, s.world.Add(part))
		}
		s.objects = append(s.objects, object)
	}
	// Authored launchers are attached to the optical pose. Their origins
	// are inside the retained shell, so the model must occlude the beam.
	height := rig.Pivot.Y + 8.0
	z := -5.0
	spacing := 2.5
	strength := 80.0
	if name == "subaru" {
		height, z = 24, -8
	}
	if name == "elt" {
		height, z, spacing, strength = 48, -20, 5, 500
	}
	for _, exit := range rig.Launchers {
		s.beams = append(s.beams, stageBeam{origin: exit.Origin, exit: exit, pivot: rig.Pivot, at: at, lamp: len(s.world.Lamps)})
		s.world.Lamp(film.Sun{})
	}
	if p.Interior && !(s.summit && p.Kind == "kecks") {
		for _, sign := range []float64{-1, 1} {
			s.world.Lamp(emit.Lit{At: at.Add(maths.V(sign*spacing*4, height-6, z+5)), Strength: maths.RGB(strength, strength*.6, strength*.25)})
		}
	}
	return nil
}

// factory adds one manufactured object of the named kind at at, scaled and
// tagged with its role.
func (s *heardStage) factory(name string, at maths.Vec, scale float64, role int) error {
	kind, err := manufacture.KindNamed(name)
	if err != nil {
		return err
	}
	made, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(5), "demo")
	if err != nil {
		return err
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		return err
	}
	obj := stageObject{at: at, scale: scale, role: role}
	if role == 1 {
		bounds := made.Bounds()
		obj.pivot = bounds.Min.Add(bounds.Max).Scale(.5)
	}
	for _, part := range parts {
		if name == "owl" && part.Material.Name == "primary mirror" {
			// A local coating override preserves the accepted campus and
			// leaves the shared catalogue's reference geometry unchanged.
			coating := *part.Material
			coating.Gloss = 1
			coating.Satin = 0
			coating.Metal = thing.Aluminium()
			part.Material = &coating
		}
		obj.things = append(obj.things, s.world.Add(part))
	}
	s.objects = append(s.objects, obj)
	return nil
}

// updateGround sets the poses, beams, layers, sun and sky for progress u.
func (s *heardStage) updateGround(p *stagePlan, state State, u float64) error {
	// One bounded correction in each shot, then a visible settled hold.
	progress := smooth(math.Min(1, u/.62))
	yaw := maths.About(maths.YAxis, mix(p.Aim, progress)*maths.Degree)
	tilt := maths.About(maths.XAxis, mix(p.Elevation, progress)*maths.Degree)
	for _, obj := range s.objects {
		place := maths.NewSubspace(obj.at, maths.Identity, obj.scale)
		if obj.role > 0 {
			place.Orientation = yaw
		}
		if obj.role == 2 {
			local := maths.NewSubspace(obj.pivot.Sub(tilt.Out(obj.pivot)), tilt, 1)
			place = local.Within(place)
		}
		for i, part := range obj.things {
			pose := place
			if len(obj.locals) > i {
				pose = obj.locals[i].Within(place)
			}
			part.Motion = thing.Still(pose)
		}
	}
	for _, beam := range s.beams {
		pose := s.beamPose(beam, p, u)
		source, err := emit.NewBeam(emit.Lit{At: pose.Origin, Toward: pose.Orientation.Out(maths.YAxis), Strength: maths.RGB(9000, 4500, 100)}, measure.Metres(.15), measure.Metres(800), measure.Degrees(.002))
		if err != nil {
			return err
		}
		s.world.Lamps[beam.lamp] = source
	}
	for _, layer := range s.layers {
		layer.volume.Medium = layer.field.Tick(time.Duration(state.At * 4 * float64(time.Second)))
	}
	altitude := mix(p.Sun, u) * maths.Degree
	sun := maths.V(.15, math.Sin(altitude), -math.Cos(altitude)).Unit()
	if s.summit {
		sun = maths.V(-math.Cos(altitude), math.Sin(altitude), .15).Unit()
	}
	turn := mix(p.SkyHours, u) * math.Pi / 12
	moon := heardMoon(turn)
	light := maths.RGB(.22, .17, .105)
	if s.summit {
		// Cool moonlit volcanic ground and cloud tops remain distinct from sunset.
		light = maths.RGB(.3, .36, .48)
	}
	toward := moon
	if p.Sun[0] != p.Sun[1] || p.Sun[0] > 0 {
		light = maths.RGB(.6, .24, .065).Scale(math.Max(.02, math.Sin(altitude)+.04))
		toward = sun
	}
	if s.summit && altitude >= 0 {
		light = maths.RGB(1.4, .72, .35).Scale(.35 + math.Sin(altitude))
		toward = sun
	}
	if s.summit && altitude < 0 {
		light = maths.RGB(.36, .43, .6)
		toward = moon
	}
	if p.Starlight {
		light = maths.RGB(.025, .033, .05)
		toward = maths.V(-.4, .7, .3).Unit()
	}
	s.world.Lamps[0] = film.Sun{Toward: toward, Strength: light, Size: .53 * maths.Degree}
	extra := 0.0
	if p.Burst {
		extra = smooth((state.At - 67*240.0/66) / 1.2)
		s.world.Lamps[0] = film.Sun{Toward: maths.V(-.3, .6, -.4).Unit(), Strength: maths.RGB(.22, .17, .105).Add(maths.RGB(1.2, 1.5, 2).Scale(extra)), Size: .01}
		for i, index := range s.burstLamps {
			dir := []maths.Vec{maths.V(.7, .4, -.3), maths.V(-.5, .5, .6), maths.V(.2, .8, .5)}[i]
			light := []maths.Light{maths.RGB(1.1, .6, .3), maths.RGB(.4, .7, 1.4), maths.RGB(.8, .8, .65)}[i]
			s.world.Lamps[index] = film.Sun{Toward: dir.Unit(), Strength: light.Scale(extra), Size: .02}
		}
	}
	s.world.Sky = heardSky{sun: sun, moon: moon, day: math.Max(0, math.Sin(altitude)), turn: turn, burst: extra, summit: s.summit}
	if len(s.world.Lamps) > 8 {
		return fmt.Errorf("heard ground exceeds fastlight's eight lamps")
	}
	return nil
}

// beamPose composes the same optical pivot and azimuth used by the hardware.
func (s *heardStage) beamPose(beam stageBeam, p *stagePlan, u float64) maths.Subspace {
	progress := smooth(math.Min(1, u/.62))
	yaw := maths.About(maths.YAxis, mix(p.Aim, progress)*maths.Degree)
	tilt := maths.About(maths.XAxis, mix(p.Elevation, progress)*maths.Degree)
	local := maths.NewSubspace(beam.pivot.Sub(tilt.Out(beam.pivot)), tilt, 1)
	return beam.exit.Within(local.Within(maths.NewSubspace(beam.at, yaw, 1)))
}

// heardMoon shares the stars' authored sidereal clock. Star lookup rotates
// a world ray into the sky, so a body rotates in the inverse direction.
// This is film staging, not an ephemeris; its initial direction is authored.
func heardMoon(turn float64) maths.Vec {
	initial := maths.V(-.65, math.Sin(12*maths.Degree), -.73).Unit()
	return maths.About(heardSkyAxis(), -turn).Out(initial)
}

// heardSkyAxis is the axis that the stars, the Moon and the Milky Way turn on.
func heardSkyAxis() maths.Vec {
	return maths.V(.2, .8, .4).Unit()
}

// heardSky rotates stars, the Moon and a textured Milky Way on one clock.
// The band is an authored sky illustration, not measured survey pixels.
type heardSky struct {
	sun, moon maths.Vec
	day, turn float64
	burst     float64
	summit    bool
}

// Light returns the sky radiance seen along direction.
func (s heardSky) Light(direction maths.Vec) maths.Light {
	d := direction.Unit()
	rotated := maths.About(heardSkyAxis(), s.turn).Out(d)
	stars := sky.Stars{Share: .12, Size: .17, Bright: .7, Seed: 29}.Light(rotated)
	band := math.Exp(-math.Pow(rotated.Dot(maths.V(.3, .7, .6).Unit())/.11, 2))
	dust := .25 + .75*noise.Fractal(81, rotated.X*14, rotated.Y*14, rotated.Z*14, 3)
	// The physical terrain, rather than a world-horizontal sky cutoff,
	// determines visibility from an elevated observer. Keep the atmospheric
	// gradient continuous across horizontal and let real geometry occlude it.
	height := math.Max(0, d.Y)
	base := maths.RGB(.018+.23*s.day, .012+.09*s.day, .012+.04*s.day).Scale(1 - height).Add(maths.RGB(.001, .002, .007).Scale(height))
	if s.summit {
		// Authored blue-hour sky: cool overhead, with the remaining warm
		// twilight concentrated toward the setting Sun, not a brown wash.
		twilight := 1 - maths.Smooth(.08, .24, -s.sun.Y)
		warm := math.Pow(math.Max(0, d.Dot(s.sun)), 12) * math.Exp(-height*9)
		base = maths.RGB(.004, .009, .03).Scale(height).Add(maths.RGB(.025, .035, .075).Scale(1 - height))
		base = base.Add(maths.RGB(.2, .065, .035).Scale(warm * twilight))
	}
	base = base.Add(stars.Scale(1 - math.Min(1, s.day*6))).Add(maths.RGB(.035, .027, .024).Scale(band * dust * (1 - math.Min(1, s.day*6))))
	if s.burst > 0 {
		// 100x as many tiles and .8/.12 occupancy: about667x the star population.
		dense := sky.Stars{Tiling: hexes.Sphere{Splits: 1200}, Share: .8, Size: .14, Bright: 2, Seed: 195}.Light(rotated)
		resolved := sky.Stars{Tiling: hexes.Sphere{Splits: 200}, Share: .25, Size: .22, Bright: 9, Seed: 196}.Light(rotated)
		lane := math.Exp(-math.Pow(rotated.Dot(maths.V(.7, -.15, -.25).Unit())/.2, 2))
		texture := math.Max(0, .5+.5*noise.Fractal(197, rotated.X*28, rotated.Y*28, rotated.Z*28, 3))
		cloud := maths.RGB(.9, .6, .4).Scale(lane * texture)
		base = base.Add(dense.Scale(.08 + .92*lane).Add(resolved).Add(cloud).Scale(s.burst))
	}
	// Opaque celestial disks cover the stars, including the response sky.
	// The warm Sun emits; the neutral Moon reflects that Sun with a visible
	// terminator. Both angular radii retain the existing authored staging.
	if d.Dot(s.sun) > math.Cos(.009) && s.sun.Y > -.02 {
		return maths.RGB(1.5, .65, .15)
	}
	if light, hit := s.moonLight(d); hit {
		return light
	}
	return base
}

// moonLight shades a spherical lunar disk instead of a second glowing sun.
// Its mottling and faint earthshine are illustrative, not survey pixels.
func (s heardSky) moonLight(direction maths.Vec) (maths.Light, bool) {
	const radius = .0046
	cosine := direction.Dot(s.moon)
	if cosine <= math.Cos(radius) {
		return maths.Light{}, false
	}
	impact := direction.Sub(s.moon.Scale(cosine)).Scale(1 / math.Sin(radius))
	depth := math.Sqrt(math.Max(0, 1-impact.Dot(impact)))
	normal := impact.Sub(s.moon.Scale(depth))
	fixed := maths.About(heardSkyAxis(), s.turn).Out(normal)
	mottle := .75 + .25*noise.Fractal(291, fixed.X*13, fixed.Y*13, fixed.Z*13, 3)
	phase := math.Max(0, normal.Dot(s.sun))
	return maths.RGB(.23, .235, .24).Scale(mottle * (.015 + phase)), true
}

// radioRig shares the authored antenna mechanism across all twelve positions.
func (s *heardStage) radioRig(at maths.Vec) error {
	r, err := manufacture.MakeRadioRig("sma-antenna")
	if err != nil {
		return err
	}
	for role, parts := range [][]thing.Thing{r.Fixed, r.Azimuth, r.Optical} {
		obj := stageObject{at: at, pivot: r.Pivot, scale: 1, role: role}
		for _, part := range parts {
			obj.things = append(obj.things, s.world.Add(part))
		}
		s.objects = append(s.objects, obj)
	}
	return nil
}
