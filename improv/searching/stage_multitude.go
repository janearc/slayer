package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

type crowdPart struct {
	part         *thing.Thing
	local, frame maths.Subspace
	glow         maths.Light
	axis         maths.Vec
	rate, flex   float64
	framed       bool
}
type crowdBeing struct {
	at                              maths.Vec
	scale, phase                    float64
	family, system, member, members int
	identity                        beingIdentity
	parts                           []crowdPart
	volumes                         []*thing.Volume
}
type multitude struct{ beings []crowdBeing }

// Groups have one to seven members, including binaries and hierarchical
// triples. No finished character is reused. Seeds govern construction,
// temperament, independent component motion, and the group's relationship.
func (s *heardStage) makeMultitude() error {
	s.crowd = &multitude{}
	s.world.Sky = sky.Stars{Share: .18, Size: .10, Bright: .35, Dark: maths.RGB(.0003, .0006, .002), Seed: 81}
	s.world.Lamp(thing.Sun(maths.V(-1, 2, 3), maths.RGB(.8, .6, .4)))
	s.world.Lamp(thing.Sun(maths.V(2, -1, 1), maths.RGB(.2, .35, .7)))
	id := 0
	for system := 0; system < 768; system++ {
		a := float64(system) * 2.3999632297
		radius := math.Sqrt(math.Max(0, float64(system-31)) / 736)
		centre := maths.V(350+360*radius*math.Cos(a), 150*radius*math.Sin(a), -35-float64((system*67)%350))
		if system < 32 {
			centre = maths.V(float64(system)*23, 2*math.Sin(a), -float64(system%3)*4)
		}
		count := 1 + system%7
		for member := 0; member < count; member++ {
			g := identityOf(id)
			size := .35 + 1.4*identityHash(g.Seed, 30)
			if count > 3 && member == 0 {
				size *= 1.7
			}
			b := crowdBeing{at: centre, scale: size, phase: a, identity: g, family: g.Purpose, system: system, member: member, members: count}
			if err := s.buildIndividual(&b, system < 32); err != nil {
				return err
			}
			// Resolved jets are participating volumes with knots and radial falloff.
			// Distant members keep their unique receiving/radiating anatomy; their
			// subpixel gas is omitted explicitly, not replaced with opaque rods.
			if g.Purpose == jetBeing && system < 40 {
				b.volumes = append(b.volumes, s.world.AddVolume(thing.Volume{
					Medium:   medium.Jet{Length: g.Radius * 3, Base: .12, Spread: .16, Thick: 1.8, From: g.Radius * .15, Knots: .8, Seed: g.Seed, Flow: 4},
					Material: &thing.Material{Name: "individual bipolar radiation", Glow: g.Signal.Scale(5)}}))
			}
			s.crowd.beings = append(s.crowd.beings, b)
			id++
		}
		// These clouds are environments shared by related beings, not spherical
		// ornaments attached to every character.
		if system < 96 && system%4 == 1 {
			s.world.AddVolume(thing.Volume{
				Medium:   medium.Wispy{Of: medium.Puff{Radius: 10 + 3*identityHash(uint64(system+1), 1), Thick: .035}, Scale: .8, Amount: 1, Seed: uint64(system + 217)},
				Material: &thing.Material{Name: "inhabited nebula", Glow: identityOf(system).Signal.Scale(.16)},
				Motion:   thing.Still(maths.NewSubspace(centre, maths.Identity, 1))})
		}
	}
	return nil
}

// crowdTravel accelerates the right-to-left passage and then stops for the
// pullback. The absolute clock makes seeking independent of frame order.
func crowdTravel(at float64) float64 {
	t := math.Max(0, math.Min(32, at-76*240.0/66))
	return 2*t + .02*t*t*t
}

// crowdIndex returns the crowd step that is current at the clock at.
func crowdIndex(at float64) int {
	return int(math.Max(0, math.Min(95, math.Floor((at-76*240.0/66)/(10*240.0/66)*96))))
}

// beingOffset returns where a member sits relative to its system centre.
func beingOffset(b crowdBeing, at float64) maths.Vec {
	if b.members == 1 {
		return maths.Vec{}
	}
	phase := b.phase + at*(.06+.28*identityHash(uint64(b.system+1), 45))
	// True binary: both members orbit a common centre. Triples put a binary
	// inside a more slowly orbiting tertiary. Larger hosts carry satellites.
	if b.members == 2 || b.members == 3 {
		if b.member < 2 {
			a := phase + math.Pi*float64(b.member)
			v := maths.V(4.5*math.Cos(a), 2.8*math.Sin(a), 2*math.Sin(a*.7))
			if b.members == 3 {
				v = v.Add(maths.V(2*math.Cos(phase*.3), 0, 2*math.Sin(phase*.3)))
			}
			return v
		}
		return maths.V(12*math.Cos(phase*.3+math.Pi), 4*math.Sin(phase*.3), 9*math.Sin(phase*.3+math.Pi))
	}
	if b.member == 0 {
		return maths.Vec{}
	}
	a := phase*(.5+.22*float64(b.member)) + 2*math.Pi*float64(b.member)/float64(b.members-1)
	radius := 5 + 2.2*float64(b.member)
	return maths.V(radius*math.Cos(a), radius*.42*math.Sin(a+b.phase), radius*.6*math.Sin(a))
}

// updateMultitude moves every being to the clock at and returns the eye.
func (s *heardStage) updateMultitude(at float64) maths.Subspace {
	for _, b := range s.crowd.beings {
		g := b.identity
		turn := maths.About(maths.YAxis, g.Phase+at*g.Speed*.13).Then(maths.About(maths.ZAxis, g.Precession*math.Sin(at*.31+g.Phase))).Then(maths.About(maths.XAxis, g.Precession*math.Sin(at*.23)))
		root := maths.NewSubspace(b.at.Add(beingOffset(b, at)), turn, b.scale)
		// Each temperament has its own attack, waiting interval and colour.
		burst := math.Pow(math.Max(0, math.Sin(at*(.3+identityHash(g.Seed, 40))+g.Phase)), 12)
		for i, part := range b.parts {
			angle := at*part.rate + part.flex*math.Sin(at*(.2+math.Abs(g.Speed))+g.Phase+float64(i)*.7)
			rotation := maths.NewSubspace(maths.Vec{}, maths.About(part.axis, angle), 1)
			pose := rotation.Within(part.local)
			if part.framed {
				pose = part.local.Within(rotation).Within(part.frame)
			}
			part.part.Motion = thing.Still(pose.Within(root))
			part.part.Material.Glow = part.glow.Scale(.65 + 2.5*burst)
		}
		for _, v := range b.volumes {
			v.Motion = thing.Still(root)
			v.Material.Glow = g.Signal.Scale(3 + 9*burst)
		}
	}
	travel := crowdTravel(at)
	eye := maths.V(travel, 1.5, 22)
	look := maths.V(travel+2, 0, -6)
	u := smooth((at - (76*240.0/66 + 32)) / 11.5)
	eye = eye.Scale(1 - u).Add(maths.V(350, 35, 550).Scale(u))
	look = look.Scale(1 - u).Add(maths.V(350, 0, -120).Scale(u))
	return maths.NewSubspace(eye, maths.LookAt(eye, look), 1)
}
