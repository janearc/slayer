package searching

import (
	_ "embed"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// The exact authored geometry from the selected 84-frame reference.
// Only its score timing and placement in the continuous retreat change.
//
//go:embed reveal-reference.yaml
var revealReference []byte

// makeContinuousField builds the continuous cosmos field from the plan's
// catalogue.
func (s *heardStage) makeContinuousField(p *StagePlan) error {
	data, err := readStageCatalogue(p.Catalogue)
	if err != nil {
		return err
	}
	c := &continuousField{centre: maths.V(-10e6, 1e6, -6e6), fixed: map[*thing.Thing]maths.Subspace{}}
	s.cosmos = c
	s.world.Sky = sky.Stars{Share: .15, Size: .12, Bright: .2, Dark: maths.RGB(.0001, .0001, .0003), Seed: 29}
	add := func(at maths.Vec, radius float64, glow maths.Light, layer int) {
		part := s.sphere(at, radius, maths.Light{}, glow)
		c.members = append(c.members, fieldMember{part: part, pose: part.At(0), glow: glow, layer: layer})
	}
	for _, point := range data.Points {
		level := .5 + 1.5*math.Pow(10, -.08*math.Max(0, point.Magnitude-6))
		add(point.position(), .001, maths.RGB(level, level*.85, level*.65), 0)
	}
	add(maths.Vec{}, .001, maths.RGB(.08, 1.4, 2), 4)
	first := len(s.world.Things)
	s.galaxy(maths.V(-8200, 0, 0), 18000, 1400, 29)
	s.galaxy(maths.V(620000, 140000, -400000), 24000, 450, 31)
	s.galaxy(maths.V(-320000, -80000, 190000), 6000, 160, 32)
	for _, part := range s.world.Things[first:] {
		c.members = append(c.members, fieldMember{part: part, pose: part.At(0), glow: part.Material.Glow, layer: 1})
	}
	c.plan = &FlowPlan{}
	if err := yamlfile.Read("reveal-reference.yaml", revealReference, c.plan); err != nil {
		return err
	}
	c.reference, err = newFlow(c.plan)
	if err != nil {
		return err
	}
	for _, part := range c.reference.world.Things {
		c.fixed[part] = part.At(0)
	}
	for _, v := range c.reference.world.Volumes {
		c.volumePoses = append(c.volumePoses, v.At(0))
	}
	s.world.Things = append(s.world.Things, c.reference.world.Things...)
	s.world.Volumes = append(s.world.Volumes, c.reference.world.Volumes...)
	// Stars lie in front of the black silhouette after recognition. The
	// same aperture and coordinates are retained during the later entry.
	for i := 0; i < 1200; i++ {
		a := float64(i) * 2.3999632297
		r := math.Sqrt((float64(i) + .5) / 1200)
		point := maths.About(maths.ZAxis, .8).Out(maths.V(3.6*r*math.Cos(a), 1.3*r*math.Sin(a), 0))
		point.Z = 7.15 + float64(i%11)*.025
		add(c.centre.Add(point.Scale(1e6)), 5000+float64(i%7)*900, maths.RGB(.35+float64(i%5)*.1, .55, .9), 6)
	}
	if p.Plunge {
		for i := 0; i < 48; i++ {
			b := crowdBeing{identity: identityOf(10000 + i)}
			if err := s.buildIndividual(&b, true); err != nil {
				return err
			}
			a := float64(i) * 2.3999632297
			r := math.Sqrt((float64(i) + .5) / 48)
			point := maths.About(maths.ZAxis, .8).Out(maths.V(3*r*math.Cos(a), 1.1*r*math.Sin(a), 0))
			point.Z = 7.5
			root := maths.NewSubspace(c.centre.Add(point.Scale(1e6)), maths.Identity, 35000)
			for _, part := range b.parts {
				pose := part.local
				if part.framed {
					pose = pose.Within(part.frame)
				}
				pose = pose.Within(root)
				part.part.Motion = thing.Still(pose)
				c.members = append(c.members, fieldMember{part: part.part, pose: pose, glow: part.glow, layer: 6, phase: b.identity.Phase + .0001})
			}
		}

		s.world.Lamp(thing.Sun(maths.V(-1, 2, 3), maths.RGB(.4, .25, .5)))
	}
	return nil
}

// updateContinuousField returns the cosmos pose at time at. The plunge flag
// selects the plunge pose.
func (s *heardStage) updateContinuousField(at float64, plunge bool) maths.Subspace {
	c := s.cosmos
	distance := fieldDistance(at)
	factor := 10 / distance
	eye := maths.V(0, .45, 1).Scale(10)
	look := maths.Vec{}
	reveal := smooth((at - 203.636363636) / 12.363636364)
	target := c.centre.Scale(factor)
	eye = eye.Scale(1 - reveal).Add(target.Add(maths.V(0, 0, 50e6*factor)).Scale(reveal))
	look = target.Scale(reveal)
	c.reference.updateClock(c.plan, at, 203.636363636, 216, 240)
	changed := map[*thing.Thing]bool{c.reference.opening: true}
	for _, segment := range c.reference.segments {
		changed[segment.object] = true
	}
	root := maths.NewSubspace(target, maths.Identity, 1e6*factor)
	for _, part := range c.reference.world.Things {
		pose := c.fixed[part]
		if changed[part] {
			pose = part.At(0)
		}
		part.Motion = thing.Still(pose.Within(root))
		part.Attrs.Hidden = at < 203 || factor*1e6 > 10000
	}
	for i, v := range c.reference.world.Volumes {
		v.Motion = thing.Still(c.volumePoses[i].Within(root))
		v.Attrs.Hidden = at < 203 || factor*1e6 > 10000
	}
	for _, m := range c.members {
		pose := m.pose
		level := 1.
		switch m.layer {
		case 0:
			level = 1 - smooth((math.Log10(distance)-3)/1.6)
			pose.Scale = math.Max(.001, pose.Origin.Sub(eye.Scale(1/factor)).Len()*.0015)
		case 1:
			level = smooth((math.Log10(distance)-2)/2) * (1 - reveal)
			pose.Scale = math.Max(pose.Scale, math.Min(distance*.0009, 50000))
		case 4:
			pose.Scale = distance * .0016
			level = 1 - reveal
		case 6:
			if m.phase != 0 {
				pose.Orientation = maths.About(maths.YAxis, .4*math.Sin(at*.7+m.phase)).Then(pose.Orientation)
			}
			level = smooth((at - 224) / 8)
		}
		m.part.Attrs.Hidden = level < .00001 || pose.Origin.Len()*factor > 1e5
		pose.Origin = pose.Origin.Scale(factor)
		pose.Scale *= factor
		m.part.Motion = thing.Still(pose)
		m.part.Material.Glow = m.glow.Scale(level)
	}
	if plunge {
		u := smooth((at - 74*240.0/66) / (2 * 240.0 / 66))
		eye = target.Add(maths.V(0, 0, (50-42.42*u)*1e6*factor))
		look = target.Add(maths.V(0, 0, 7e6*factor))
	}
	return maths.NewSubspace(eye, maths.LookAt(eye, look), 1)
}
