package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"strings"
	"time"
)

// makeSpace builds small bounded proof sets. Fictional respondents and
// large-scale flow geometry are authored, not presented as survey data.
func (s *heardStage) makeSpace(p *stagePlan) error {
	s.world.Sky = sky.Stars{Share: .14, Size: .16, Bright: .8, Dark: maths.RGB(.0006, .0008, .002), Seed: 29}
	if p.Kind != "space" {
		s.world.Lamp(thing.Sun(maths.V(-1, 2, 3), maths.RGB(.35, .2, .4)))
		s.world.Lamp(thing.Sun(maths.V(1, -1, 2), maths.RGB(.1, .16, .25)))
	}
	switch p.Kind {
	case "individual":
		b := crowdBeing{identity: identityOf(p.Identity), scale: 1, members: 1}
		var err error
		if p.Anatomy == nil {
			err = s.buildIndividual(&b, true)
		} else {
			err = s.buildAuthoredIndividual(&b, p.Anatomy, anatomyCacheFor(p))
		}
		if err != nil {
			return err
		}
		s.crowd = &multitude{beings: []crowdBeing{b}}
		return nil
	case "authored-multitude":
		return s.makeAuthoredMultitude(p)
	case "zappy":
		return s.respondent(0, maths.Vec{}, 1)
	case "space":
		s.world.Lamp(thing.Sun(maths.V(.8, 1.4, -2), maths.RGB(1.8, 1.6, 1.3)))
		s.world.Lamp(thing.Sun(maths.V(-1, 0, 1), maths.RGB(.16, .23, .35)))
		if p.Craft != "" {
			return s.factory(p.Craft, maths.Vec{}, 1, 1)
		}
		for i, name := range []string{"hubble", "webb", "spitzer"} {
			if err := s.factory(name, maths.V(float64(i-1)*18, 0, float64(i%2)*5), 1, 1); err != nil {
				return err
			}
		}
		s.sphere(maths.V(0, -35, -90), 25, maths.RGB(.008, .05, .12), maths.Light{})
	case "milky-way":
		s.galaxy(maths.Vec{}, 22, 1600, 29)
		s.sphere(maths.V(13, 0, 3), .25, maths.Light{}, maths.RGB(.1, 2, 3))
	case "galaxies":
		for i, at := range []maths.Vec{maths.V(0, 0, 0), maths.V(30, 10, -35), maths.V(-20, -4, 18), maths.V(60, -8, -75)} {
			s.galaxy(at, 6+float64(i)*1.5, 250, uint64(30+i))
		}
		for i := range 450 {
			a := float64(i) * 2.399963229728653
			r := 25 + float64(i)*.55
			at := maths.V(r*math.Cos(a), 20*math.Sin(float64(i)*4.2), r*math.Sin(a))
			s.sphere(at, .12, maths.Light{}, maths.RGB(.9, .6, .35))
		}
	case "laniakea", "huge-fold", "answer", "consequence":
		s.cosmicWeb()
		if err := s.respondent(2, maths.Vec{}, 11); err != nil {
			return err
		}
		changed := s.sphere(maths.V(-.5, .2, -18), 4, maths.Light{}, maths.RGB(.08, .3, .8))
		pulse := s.sphere(maths.Vec{}, .12, maths.Light{}, maths.RGB(4, 1.5, .5))
		s.marks = append(s.marks, pulse)
		s.marks = append(s.marks, changed)
	case "fold":
		return s.respondent(2, maths.Vec{}, 1)
	case "lantern":
		return s.respondent(1, maths.Vec{}, 1)
	case "occulter":
		return s.respondent(3, maths.Vec{}, 1)
	case "conversation", "accumulate", "together", "withdraw":
		for i, at := range []maths.Vec{maths.V(-4, 2, 0), maths.V(4, 2, 0), maths.V(-4, -2, 0), maths.V(4, -2, 0)} {
			if err := s.respondent(i, at, .65); err != nil {
				return err
			}
		}
	}
	return nil
}

// sphere adds a sphere of the given radius at a position, with its colour and glow.
func (s *heardStage) sphere(at maths.Vec, radius float64, colour, glow maths.Light) *thing.Thing {
	return s.world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Name: "authored proof surface", Colour: colour, Glow: glow}, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, radius))})
}

// respondent keeps four readable silhouettes: crossed star/swarm, warm
// lantern, three hinged membranes, and a dark irregular occulter.
func (s *heardStage) respondent(kind int, at maths.Vec, scale float64) error {
	if kind == 0 || kind == 1 {
		return s.nativeCharacter(kind, at, scale)
	}
	object := stageObject{at: at, scale: scale, role: 10 + kind}
	first := len(s.world.Things)
	switch kind {
	case 2:
		if err := s.foldAnatomy(at, scale); err != nil {
			return err
		}
	case 3:
		if err := s.occulterAnatomy(); err != nil {
			return err
		}
	}
	for _, part := range s.world.Things[first:] {
		if kind == 2 && strings.HasPrefix(part.Material.Name, "fold ") {
			continue
		}
		// Preserve each local member transform before putting the assembly
		// into the shot. Its own pose is retained in the nested motion.
		local := part.At(0)
		part.Motion = thing.Still(local.Within(maths.NewSubspace(at, maths.Identity, scale)))
		object.things = append(object.things, part)
		object.glows = append(object.glows, part.Material.Glow)
		object.locals = append(object.locals, local)
	}
	s.objects = append(s.objects, object)
	return nil
}

// galaxy is an authored logarithmic spiral illustration. The separate
// preceding catalogue shot carries actual Gaia identities and distances.
func (s *heardStage) galaxy(centre maths.Vec, size float64, count int, seed uint64) {
	for i := range count {
		u := float64(i+1) / float64(count)
		r := size * math.Sqrt(u)
		arm := float64(i%4) * math.Pi / 2
		a := arm + 3*math.Log(.08+u) + .22*noise.Fractal(seed, float64(i)*.91, 0, 0, 1)
		thickness := .4 * noise.Fractal(seed+1, float64(i), 0, 0, 1)
		at := centre.Add(maths.V(r*math.Cos(a), thickness, r*math.Sin(a)))
		s.sphere(at, .025+.055*(1-u), maths.Light{}, maths.RGB(1, .75, .5).Scale(.8+1.5*(1-u)))
	}
}

// cosmicWeb avoids the old regular radial spokes: clustered meanders,
// intermittent knots and diffuse glow leave an irregular central void.
func (s *heardStage) cosmicWeb() {
	for strand := range 18 {
		a := float64(strand) * 2.399963229728653
		for j := range 80 {
			t := float64(j) / 79
			r := 9 + 29*t
			angle := a + .5*math.Sin(t*4+float64(strand))
			at := maths.V(r*math.Cos(angle), r*math.Sin(angle)*.55+4*math.Sin(t*9+a), -8+6*math.Cos(a+t*3))
			scatter := noise.Fractal(uint64(strand+27), t*18, 0, 0, 2)
			s.sphere(at, .035+.13*math.Abs(scatter), maths.Light{}, maths.RGB(.75, .31, .12).Scale(.4+math.Abs(scatter)))
		}
	}
	for i := range 7 {
		a := float64(i) * 2.399963229728653
		at := maths.V(18*math.Cos(a), 11*math.Sin(a), -12)
		s.world.AddVolume(thing.Volume{Medium: medium.Wispy{Of: medium.Puff{Radius: 6, Thick: .025}, Scale: .7, Amount: 1, Seed: uint64(60 + i)}, Material: &thing.Material{Name: "authored diffuse intergalactic proof field", Glow: maths.RGB(.12, .035, .009)}, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1))})
	}
}

// updateSpace moves the space objects for the stage at progress u.
func (s *heardStage) updateSpace(p *stagePlan, state State, u float64) {
	// Four stages align with the same four musical bars at both scales.
	phase := math.Min(3, u*4)
	fold := (math.Floor(phase) + smooth(phase-math.Floor(phase))) / 3
	if p.Kind == "laniakea" {
		fold = 0
	}
	if p.Kind == "answer" || p.Kind == "consequence" {
		fold = 1
	}
	for _, obj := range s.objects {
		if obj.role == 1 && p.Kind == "space" {
			// Each spacecraft has its own absolute musical entrance, so a
			// cut never restarts its already completed slew.
			entry := 174.54545454545453 + (obj.at.X/18+1)*240/66
			if p.Craft != "" {
				entry = 174.54545454545453 + float64(map[string]int{"hubble": 0, "webb": 1, "spitzer": 2}[p.Craft])*240/66
			}
			progress := smooth((state.At - entry) / 2.4)
			place := maths.NewSubspace(obj.at, maths.About(maths.YAxis, (-.35+.7*progress)), obj.scale)
			if p.Craft != "" {
				// Factory optical axes are +Y: tilt, then yaw. Different roll keeps
				// silhouettes distinct while the final optical direction agrees.
				roll := map[string]float64{"hubble": .2, "webb": -1.0, "spitzer": 1.1}[p.Craft]
				start := map[string]float64{"hubble": -.7, "webb": .6, "spitzer": -.4}[p.Craft]
				place = maths.NewSubspace(maths.Vec{}, maths.About(maths.YAxis, roll), 1).Within(maths.NewSubspace(maths.Vec{}, maths.About(maths.XAxis, -1.15+start*(1-progress)), 1))
			}
			place.Origin = obj.at.Sub(place.Orientation.Out(obj.pivot).Scale(obj.scale))
			for _, part := range obj.things {
				part.Motion = thing.Still(place)
			}
		}
		if obj.role >= 20 && obj.role < 23 {
			i := obj.role - 20
			a := float64(i) * 2 * math.Pi / 3
			turn := maths.About(maths.ZAxis, a)
			tilt := maths.About(maths.YAxis, (.05+1.45*fold)*float64(1-2*(i%2)))
			place := maths.NewSubspace(obj.at.Add(obj.pivot.Scale(obj.scale)), turn, obj.scale)
			place = maths.NewSubspace(maths.Vec{}, tilt, 1).Within(place)
			for j, part := range obj.things {
				pose := place
				if len(obj.locals) > j {
					pose = obj.locals[j].Within(place)
				}
				part.Motion = thing.Still(pose)
				part.Attrs.Hidden = p.Kind == "accumulate" && u < .5
			}
		}
		if obj.role >= 10 && obj.role <= 13 {
			active := true
			level := 1.0
			if p.Kind == "accumulate" {
				active = obj.role-10 <= int(math.Floor(u*4))
			}
			if p.Kind == "conversation" {
				level = .25
				if int(math.Floor(u*4)) == obj.role-10 {
					level = 1.8
				}
			}
			if p.Kind == "withdraw" {
				level = 1 - smooth(u)
			}
			if obj.role == 11 && p.Kind == "lantern" {
				level = .65 + .35*math.Sin(math.Pi*u)
			}
			turn := 0.0
			if obj.role == 10 {
				turn = .7 * u
				level *= .6 + .8*math.Pow(math.Max(0, math.Cos(state.Beat*math.Pi)), 8)
			}
			if p.Kind == "together" {
				turn = .4 * smooth(u)
			}
			if p.Kind == "conversation" && int(math.Floor(u*4)) != obj.role-10 {
				turn = 0
			}
			if p.Kind == "withdraw" {
				turn = .4
			}
			for i, v := range obj.volumes {
				v.Attrs.Hidden = !active
				local := (polarShaft{local: maths.Local()}).At(time.Duration(state.At * float64(time.Second)))
				root := maths.NewSubspace(obj.at, maths.About(maths.YAxis, turn), obj.scale)
				v.Motion = thing.Still(local.Within(root))
				v.Material.Glow = obj.volumeGlows[i].Scale(level)
			}

			for i, part := range obj.things {
				part.Attrs.Hidden = !active
				place := maths.NewSubspace(obj.at, maths.About(maths.YAxis, turn), obj.scale)
				if obj.role == 13 && part.Material.Name == "occulter shaped absence" {
					place.Origin.X += obj.scale * (-.5 + smooth(u))
				}
				local := obj.locals[i]
				if len(obj.motions) > i && obj.motions[i] != nil {
					clock := state.At
					if obj.role == 11 && p.Kind == "lantern" && i > 0 {
						group := (i - 1) * 8 / (len(obj.things) - 1)
						clock = lanternShellClock(u, group)
					}
					local = obj.motions[i].At(time.Duration(clock * float64(time.Second)))
				}
				if part.Material.Name == "occulter petal" {
					local = local.Within(maths.NewSubspace(maths.Vec{}, maths.About(maths.ZAxis, .28*math.Sin(state.At*.6+float64(i)*.1)), 1))
				}
				part.Motion = thing.Still(local.Within(place))
				// Pulse only luminous surfaces; the base glow is recovered from a
				// stable function instead of multiplying yesterday's material.
				if obj.glows[i].Luminance() > 0 {
					part.Material.Glow = obj.glows[i].Scale(level)
				}
			}
		}
	}
	if len(s.marks) > 0 {
		pulse := s.marks[0]
		pulse.Attrs.Hidden = p.Kind != "answer"
		at := maths.V(0, 0, 1+u*45)
		pulse.Motion = thing.Still(maths.NewSubspace(at, maths.Identity, .12+u*.5))
		s.marks[1].Attrs.Hidden = fold < 1
	}
}
