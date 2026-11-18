package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// An identity describes an individual, not an index into a shelf of complete
// characters. Only elementary geometry can be shared. Stable seeds let the
// same individual return without making its neighbours copies of it.
type beingIdentity struct {
	Seed                                                  uint64
	Purpose                                               int
	Rings, Limbs                                          int
	Radius, Width, Speed, Precession, Eccentricity, Phase float64
	Colour, Signal                                        maths.Light
}

const (
	shepherd = iota
	harvester
	exchanger
	eclipse
	radiator
	listener
	lanternBeing
	jetBeing
)

// identityOf returns the seeded identity of the individual with id.
func identityOf(id int) beingIdentity {
	seed := uint64(id+1)*7919 + 4081
	pick := func(n int) float64 { return identityHash(seed, n) }
	purpose := int(pick(1) * 8)
	// Orbitals are a population, with extra shepherds and harvesters.
	if pick(2) < .24 {
		purpose = int(pick(3) * 2)
	}
	g := beingIdentity{Seed: seed, Purpose: purpose,
		Rings: 1 + int(math.Pow(pick(4), 2.8)*49), Limbs: 3 + int(pick(5)*14),
		Radius: .8 + 2.1*pick(6), Width: .006 * math.Pow(180, pick(7)),
		Speed:      (2*pick(8) - 1) * math.Pow(12, pick(9)-.5),
		Precession: .03 + .4*pick(10), Eccentricity: .34 * pick(11), Phase: 2 * math.Pi * pick(12)}
	// Deliberate extremes are spread through the population, not a cycle of
	// whole objects. All their other traits remain independently seeded.
	if id%41 == 0 {
		g.Rings = 50
		g.Purpose = id % 2
	}
	if id%37 == 0 {
		g.Rings = 1
	}
	hue := pick(13) * 2 * math.Pi
	g.Colour = maths.RGB(.12+.24*(1+math.Cos(hue)), .12+.24*(1+math.Cos(hue-2.1)), .12+.24*(1+math.Cos(hue+2.1)))
	g.Signal = maths.RGB(.08+2*(1+math.Cos(hue+.8)), .08+2*(1+math.Cos(hue-1.3)), .08+2*(1+math.Cos(hue+2.9)))
	if pick(14) < .28 {
		g.Colour = g.Colour.Scale(.025)
		g.Signal = g.Signal.Scale(.08)
	}
	return g
}

// ringSkin adds resolvable service seams and settlement bands; unresolved
// seams converge to their mean. Every ring's phase and geography are seeded.
type ringSkin struct {
	seed   uint64
	colour maths.Light
}

// At returns the colour at p at full resolution.
func (t ringSkin) At(p maths.Vec) maths.Light { return t.Within(p, 0) }

// Within returns the colour at p averaged over a footprint of the given size.
func (t ringSkin) Within(p maths.Vec, footprint float64) maths.Light {
	a := math.Atan2(p.Z, p.X)
	macro := .65 + .35*noise.Fractal(t.seed, p.X*3, p.Y*2, p.Z*3, 3)
	if footprint > .035 {
		return t.colour.Scale(macro * .9)
	}
	seam := math.Abs(math.Sin(a * (25 + float64(t.seed%31))))
	longitudinal := math.Abs(math.Sin(p.Y * 90))
	return t.colour.Scale(macro * (.35 + .65*maths.Smooth(.015, .09, math.Min(seam, longitudinal))))
}

// livingBand is a closed habitat belt with inner and outer skin and actual
// rims. Geometry varies in width, eccentricity and warp, not just its pose.
func livingBand(g beingIdentity, ring, segments int) (*mesh.Mesh, error) {
	r := g.Radius * (.42 + .58*float64(ring+1)/float64(g.Rings))
	width := g.Width * (.65 + .7*identityHash(g.Seed, 100+ring))
	thickness := math.Max(.004, width*.035)
	vertices := make([]maths.Vec, 0, segments*4)
	faces := make([][3]int, 0, segments*8)
	for j := 0; j < segments; j++ {
		a := float64(j) * 2 * math.Pi / float64(segments)
		radial := r * (1 + g.Eccentricity*math.Cos(a))
		warp := .08 * r * math.Sin(a*3+g.Phase) * g.Eccentricity
		for _, v := range [][2]float64{{-thickness / 2, -width / 2}, {thickness / 2, -width / 2}, {thickness / 2, width / 2}, {-thickness / 2, width / 2}} {
			point := maths.V((radial+v[0])*math.Cos(a), v[1]+warp, (radial+v[0])*math.Sin(a))
			if g.Purpose == exchanger {
				// Braided transfer loops wind through one another around an open
				// passage; paired individuals can align these passages.
				twist := float64(2+ring%3)*a + g.Phase
				rr := radial + .22*r*math.Cos(twist)
				point = maths.V((rr+v[0])*math.Cos(2*a), v[1]+.35*r*math.Sin(twist), (rr+v[0])*math.Sin(2*a))
			}
			vertices = append(vertices, point)

		}
	}
	for j := 0; j < segments; j++ {
		for k := 0; k < 4; k++ {
			a := j*4 + k
			b := j*4 + (k+1)%4
			c := ((j+1)%segments)*4 + k
			d := ((j+1)%segments)*4 + (k+1)%4
			faces = append(faces, [3]int{a, b, c}, [3]int{b, d, c})
		}
	}
	return mesh.New(vertices, faces, nil, false)
}

// collectorLeaf is a curved scoop aimed at the origin. A family can fold,
// receive or radiate with the same elementary surface vocabulary, without
// copying any complete assembly.
func collectorLeaf(g beingIdentity, index, segments int) (*mesh.Mesh, error) {
	v := []maths.Vec{}
	f := [][3]int{}
	length := g.Radius * (.75 + identityHash(g.Seed, 200+index))
	spread := .15 + .5*identityHash(g.Seed, 300+index)
	for j := 0; j <= segments; j++ {
		u := float64(j) / float64(segments)
		for k := 0; k < 3; k++ {
			side := float64(k - 1)
			width := spread * math.Sin(math.Pi*(.08+.84*u))
			point := maths.V(.3+u*length, side*width, .6*u*u+side*side*.18*math.Sin(math.Pi*u))
			switch g.Purpose {
			case radiator:
				point.Y += .3 * length * math.Sin(2*math.Pi*u+g.Phase) * u
				point.Z += .7 * u * math.Sin(float64(index)*1.3)
			case listener:
				point.X = .25 + length*u
				point.Z = length * u * u * .8
			case lanternBeing:
				point.X = .4 + length*math.Sin(math.Pi*u)
				point.Z = length * (u - .5) * 1.8
			case jetBeing:
				point.X = .3 + .6*u
				point.Y = side * width * .5
				point.Z = (u - .5) * length * 2
			}
			v = append(v, point)

		}
	}
	for j := 0; j < segments; j++ {
		for k := 0; k < 2; k++ {
			n := j*3 + k
			f = append(f, [3]int{n, n + 1, n + 3}, [3]int{n + 1, n + 4, n + 3})
		}
	}
	return mesh.New(v, f, nil, false)
}

// buildIndividual adds the parts of one individual to the world.
func (s *heardStage) buildIndividual(b *crowdBeing, close bool) error {
	g := b.identity
	segments := 48
	if close {
		segments = 128
	}
	add := func(solid shape.Shape, mat *thing.Material, pose maths.Subspace, axis maths.Vec, rate float64) {
		part := s.world.Add(thing.Thing{Shape: solid, Material: mat})
		b.parts = append(b.parts, crowdPart{part: part, local: pose, glow: mat.Glow, axis: axis, rate: rate})
	}
	// A planet, an energy reservoir, or a truly opaque absence supplies the
	// reason for the surrounding equipment. Black surfaces transmit no stars.
	core := g.Radius * .13
	mat := &thing.Material{Name: "individual reservoir", Colour: g.Colour.Scale(.3), Glow: g.Signal.Scale(.2)}
	if g.Purpose == shepherd {
		core = g.Radius * .27
		mat = &thing.Material{Name: "shepherded planet", Texture: planetSkin{seed: g.Seed}, Gloss: .04}
	}
	if g.Purpose == eclipse {
		core = g.Radius * .37
		mat = &thing.Material{Name: "opaque absence", Colour: maths.Light{}, Gloss: 0}
	}
	add(shape.Sphere{Radius: core}, mat, maths.Local(), maths.YAxis, .05*g.Speed)
	if g.Purpose <= eclipse {
		for j := 0; j < g.Rings; j++ {
			mesh, err := livingBand(g, j, segments)
			if err != nil {
				return err
			}
			tilt := (identityHash(g.Seed, 400+j) - .5) * 2.8
			if g.Purpose == shepherd {
				tilt *= .18
			}
			turn := maths.About(maths.ZAxis, tilt).Then(maths.About(maths.XAxis, g.Phase+float64(j)*.13))
			pose := maths.NewSubspace(maths.Vec{}, turn, 1)
			glow := g.Signal.Scale(.015 + .02*identityHash(g.Seed, 500+j))
			add(mesh, &thing.Material{Name: "individual orbital habitat", Texture: ringSkin{g.Seed + uint64(j), g.Colour}, Glow: glow, Gloss: .22}, pose, maths.YAxis, g.Speed*(.4+identityHash(g.Seed, 600+j)))
			// Local hubs travel with this ring, giving its spin visible landmarks.
			if close || j < 2 {
				for hub := 0; hub < 3; hub++ {
					a := float64(hub)*2*math.Pi/3 + g.Phase
					r := g.Radius * (.42 + .58*float64(j+1)/float64(g.Rings))
					r *= 1 + g.Eccentricity*math.Cos(a)
					local := maths.NewSubspace(maths.V(r*math.Cos(a), g.Width*.6, r*math.Sin(a)), maths.Identity, 1)
					// Spin in the ring plane, then orient the entire ring.
					part := s.world.Add(thing.Thing{Shape: shape.Sphere{Radius: .016 + .03*identityHash(g.Seed, 700+j*3+hub)}, Material: &thing.Material{Name: "orbital moving settlement", Glow: g.Signal}})
					b.parts = append(b.parts, crowdPart{part: part, local: local, frame: pose, glow: g.Signal, axis: maths.YAxis, rate: g.Speed * (.4 + identityHash(g.Seed, 600+j)), framed: true})
				}
			}
		}
	} else {
		for j := 0; j < g.Limbs; j++ {
			mesh, err := collectorLeaf(g, j, 12)
			if err != nil {
				return err
			}
			a := float64(j) * 2 * math.Pi / float64(g.Limbs)
			tilt := g.Phase*.15 + .18*float64(j%3)
			if g.Purpose == lanternBeing {
				tilt = 1.0 + .25*identityHash(g.Seed, 800+j)
			}
			pose := maths.NewSubspace(maths.Vec{}, maths.About(maths.YAxis, tilt).Then(maths.About(maths.ZAxis, a)), 1)
			add(mesh, &thing.Material{Name: "individual receiving surface", Texture: ringSkin{g.Seed + uint64(j), g.Colour}, Glow: g.Signal.Scale(.02), Gloss: .35}, pose, maths.YAxis, 0)
			b.parts[len(b.parts)-1].flex = .12 + .35*identityHash(g.Seed, 900+j)
		}
	}
	return nil
}

type planetSkin struct{ seed uint64 }

// At returns the surface colour of the planet at p.
func (t planetSkin) At(p maths.Vec) maths.Light {
	q := p.Unit()
	land := noise.Fractal(t.seed, q.X*4, q.Y*4, q.Z*4, 4)
	clouds := noise.Fractal(t.seed+1, q.X*9, q.Y*9, q.Z*9, 3)
	c := maths.RGB(.015, .065, .12)
	if land > .05 {
		c = maths.RGB(.09, .16, .065)
	}
	cloud := maths.Smooth(.15, .45, clouds)
	return c.Scale(1 - cloud).Add(maths.Grey(.7).Scale(cloud))
}

// identityHash returns the seeded noise value for index.
func identityHash(seed uint64, index int) float64 { return noise.Hash(seed, int64(index)) }
