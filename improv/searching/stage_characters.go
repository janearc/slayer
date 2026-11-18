package searching

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"time"
)

// nativeCharacter uses the original shared panel builders. Wear is zero at
// this proof resolution: panel anatomy and separate rings remain intact.
func (s *heardStage) nativeCharacter(kind int, at maths.Vec, scale float64) error {
	w := thing.NewWorld()
	var err error
	if kind == 0 {
		b := lantern.Zappy()
		b.Radius = .12
		b.Wear = 0
		b.Green = 100
		b.Glare = 8
		b.Orbits *= 2
		b.Spread *= .42
		b.Panels = int(float64(b.Panels) * 1.35)
		clock := new(time.Duration)
		_, err = b.Place(w, clock)
	} else {
		_, err = (lantern.Lantern{Radius: .32, Orbits: 96, Panels: 80, Kelvin: 4800, Green: 38, Wide: .35, Turn: 60 * time.Second, Shells: lantern.Shells{Spread: .018, Fastest: 30 * time.Second}}).Place(w)
	}
	if err != nil {
		return err
	}
	obj := stageObject{at: at, scale: scale, role: 10 + kind}
	for _, part := range w.Things {
		clone := *part
		mat := *part.Material
		clone.Material = &mat
		if mat.Name == "the star" {
			clone.Attrs.Clear = true
		}
		if mat.Glow.Luminance() == 0 {
			mat.Glow = maths.RGB(.008, .012, .025)
			if kind == 1 {
				mat.Glow = maths.RGB(.025, .012, .003)
			}
		}
		obj.locals = append(obj.locals, part.At(0))
		obj.motions = append(obj.motions, part.Motion)
		obj.glows = append(obj.glows, mat.Glow)
		obj.things = append(obj.things, s.world.Add(clone))
	}
	// Local lights are attached to the stars, with shadow-free stellar cores.
	light := maths.RGB(12, 20, 40)
	if kind == 1 {
		light = maths.RGB(12, 7, 2)
	}
	s.world.Lamp(emit.Lit{At: at, Strength: light.Scale(scale * scale)})
	if kind == 0 {
		// Two participating media describe a knotted, expanding flow and its
		// diffuse cocoon. Neither has an opaque surface or a constant-width end.
		for layer := 0; layer < 2; layer++ {
			spread, thick, glow := .12, 1.8, maths.RGB(.6, 3, 14)
			if layer == 1 {
				spread, thick, glow = .28, .35, maths.RGB(.4, .12, 1.8)
			}
			v := s.world.AddVolume(thing.Volume{
				Medium:   medium.Jet{Length: 3.8, Base: .16, Spread: spread, Thick: thick, From: .18, Knots: .85, Seed: uint64(710 + layer), Flow: 3.8},
				Material: &thing.Material{Name: "zappy expanding radiation", Glow: glow}})
			obj.volumes = append(obj.volumes, v)
			obj.volumeGlows = append(obj.volumeGlows, glow)
		}
	}

	s.objects = append(s.objects, obj)
	return nil
}

// characterRod supplies actual thin sleeves, not dotted approximations.
func (s *heardStage) characterRod(a, b maths.Vec, width float64, name string, glow maths.Light) (*thing.Thing, error) {
	mesh, err := flowTube(width / b.Sub(a).Len())
	if err != nil {
		return nil, err
	}
	return s.world.Add(thing.Thing{Shape: mesh, Material: &thing.Material{Name: name, Colour: maths.RGB(.15, .1, .08), Glow: glow, Gloss: .5}, Motion: thing.Still(maths.NewSubspace(a, maths.LookAt(a, b), b.Sub(a).Len()))}), nil
}

// foldAnatomy keeps the three familiar hinges but adds pleated surfaces,
// lattice ribs, nested frames and seam light to each independently moving wing.
func (s *heardStage) foldAnatomy(at maths.Vec, scale float64) error {
	for i := 0; i < 3; i++ {
		a := float64(i) * 2 * math.Pi / 3
		hinge := maths.V(math.Cos(a), math.Sin(a), 0)
		obj := stageObject{at: at, pivot: hinge, scale: scale, role: 20 + i}
		first := len(s.world.Things)
		for layer := 0; layer < 5; layer++ {
			vertices := []maths.Vec{}
			faces := [][3]int{}
			for x := 0; x <= 16; x++ {
				u := float64(x) / 16
				for y := 0; y <= 10; y++ {
					v := float64(y) / 10
					edge := (.8 + .55*u) * (1 + .06*math.Sin(u*31))
					vertices = append(vertices, maths.V(-1.2+3.2*u, (2*v-1)*edge, .16*math.Sin(u*math.Pi*10)+.06*math.Cos(v*math.Pi*12)+float64(layer)*.18))
				}
			}
			for x := 0; x < 16; x++ {
				for y := 0; y < 10; y++ {
					j := x*11 + y
					if (x*7+y*3+layer)%13 != 0 {
						faces = append(faces, [3]int{j, j + 1, j + 11}, [3]int{j + 1, j + 12, j + 11})
					}
				}
			}

			mesh, err := shape.NewMesh(vertices, faces, nil, false)
			if err != nil {
				return err
			}
			s.world.Add(thing.Thing{Shape: mesh, Material: &thing.Material{Name: "fold membrane", Colour: maths.RGB(.15, .06, .2), Glow: maths.RGB(.005, .002, .008), Gloss: .7}})
		}
		for j := 0; j <= 16; j++ {
			u := float64(j) / 16
			x := -1.2 + 3.2*u
			y := .8 + .55*u
			if _, err := s.characterRod(maths.V(x, -y, .3), maths.V(x, y, .3), .018, "fold rib", maths.RGB(.15, .045, .2)); err != nil {
				return err
			}
			if j < 16 {
				for _, sign := range []float64{-1, 1} {
					if _, err := s.characterRod(maths.V(x, sign*y, .3), maths.V(x+.2, sign*(y+.55/16), .3), .025, "fold edge", maths.RGB(.4, .12, .25)); err != nil {
						return err
					}
				}
			}
		}
		for y := 0; y < 17; y++ {
			v := float64(y)/16*2 - 1
			if _, err := s.characterRod(maths.V(-1.2, v*.8, .34), maths.V(2, v*1.35, .34), .009, "fold cross weave", maths.RGB(.1, .06, .18)); err != nil {
				return err
			}
		}
		for ring := 0; ring < 5; ring++ {
			for j := 0; j < 24; j++ {
				a := float64(j) * math.Pi / 12
				b := float64(j+1) * math.Pi / 12
				r := .18 + float64(ring)*.09
				if _, err := s.characterRod(maths.V(-.8+r*math.Cos(a), r*math.Sin(a), .5), maths.V(-.8+r*math.Cos(b), r*math.Sin(b), .5), .013, "fold hinge bearing", maths.RGB(.35, .16, .08)); err != nil {
					return err
				}
			}
		}
		for _, part := range s.world.Things[first:] {
			obj.things = append(obj.things, part)
			obj.locals = append(obj.locals, part.At(0))
		}
		s.objects = append(s.objects, obj)
		s.sphere(hinge, .13, maths.Light{}, []maths.Light{maths.RGB(3, .7, .1), maths.RGB(.2, 2, 3), maths.RGB(2, .3, 2)}[i])
	}
	return nil
}

// occulterAnatomy is a deep iris: layered asymmetric dark scales, bright
// edge machinery and a structured corona expose a changing shaped absence.
func (s *heardStage) occulterAnatomy() error {
	for layer := 0; layer < 5; layer++ {
		for i := 0; i < 32; i++ {
			a := float64(i)*2*math.Pi/32 + .13*float64(layer)
			v := []maths.Vec{maths.V(.6, -.12, 0), maths.V(2.8, -.43, .08), maths.V(2.4, .54, .16), maths.V(.95, .3, .02)}
			mesh, err := shape.NewMesh(v, [][3]int{{0, 1, 2}, {0, 2, 3}}, nil, false)
			if err != nil {
				return err
			}
			pose := maths.NewSubspace(maths.V(0, 0, float64(layer)*.22), maths.About(maths.ZAxis, a), 1)
			s.world.Add(thing.Thing{Shape: mesh, Material: &thing.Material{Name: "occulter petal", Colour: maths.RGB(.02, .025, .035), Glow: maths.RGB(.002, .004, .007), Gloss: .8}, Motion: thing.Still(pose)})
			for j := 0; j < 3; j++ {
				p := pose.Out(v[j])
				q := pose.Out(v[(j+1)%4])
				if _, err := s.characterRod(p, q, .012, "occulter edge", maths.RGB(.08, .18, .3)); err != nil {
					return err
				}
			}
		}
	}
	for i := 0; i < 180; i++ {
		a := float64(i) * 2.3999632297
		r := 2.7 + .6*math.Sin(float64(i)*1.7)
		s.sphere(maths.V(r*math.Cos(a), r*math.Sin(a), -.8), .025, maths.Light{}, maths.RGB(.2, .5, 1))
	}
	return nil
}

// polarShaft carries the familiar fast sweep and slower precession.
type polarShaft struct{ local maths.Subspace }

// At returns the shaft pose at the given time.
func (p polarShaft) At(at time.Duration) maths.Subspace {
	t := at.Seconds()
	turn := maths.About(maths.XAxis, .65*math.Sin(t*2*math.Pi/1.5)).Then(maths.About(maths.ZAxis, .35*math.Cos(t*2*math.Pi/5)))
	return p.local.Within(maths.NewSubspace(maths.Vec{}, turn, 1))
}
