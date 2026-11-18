package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// distantGround closes the finite terrain window below the site. The
// former film showed the sky's lower hemisphere beyond the 800-metre mesh.
// This authored far ground is not an extension of the measured DEM.
func (s *heardStage) distantGround(surveyed bool) error {
	s.world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Name: "authored distant ground below survey window", Colour: maths.RGB(.075, .055, .04)}, Motion: thing.Still(maths.NewSubspace(maths.V(0, -100, 0), maths.Identity, 1))})
	// A low irregular ridge gives the authored far ground a real silhouette.
	// It lies outside the local surveyed window and carries no survey claim.
	vertices := []maths.Vec{}
	faces := [][3]int{}
	const count = 256
	centre, half := maths.Vec{}, 400.
	if surveyed {
		centre, half = maths.V(400, 0, 362.5), 800
	}
	for ring := 0; ring < 4; ring++ {
		for i := 0; i < count; i++ {
			a := float64(i) * 2 * math.Pi / count
			r := []float64{half, 2*half + 400, 3500, 6000}[ring]
			if ring == 0 {
				r = (half - .01) / math.Max(math.Abs(math.Cos(a)), math.Abs(math.Sin(a)))
			}
			x, z := centre.X+r*math.Cos(a), centre.Z+r*math.Sin(a)
			y := -90.
			if ring == 0 {
				// Match the square survey window's edge, with only a tiny
				// overlap. A circular apron leaves a shadowed drop at its corners.
				y = s.ground(x, z) + .03
			}
			if ring == 1 {
				y = -30
			}
			if ring == 2 {
				y = 55 + 32*math.Sin(a*7) + 18*math.Cos(a*19) + 11*math.Sin(a*37)
			}
			vertices = append(vertices, maths.V(x, y, z))
			if ring < 3 {
				j := ring*count + i
				k := ring*count + (i+1)%count
				faces = append(faces, [3]int{j, k, k + count}, [3]int{j, k + count, j + count})
			}
		}
	}
	mesh, err := shape.NewMesh(vertices, faces, nil, false)
	if err != nil {
		return err
	}
	groundColour := s.world.Things[0].Material.Colour.Scale(.85)
	s.world.Add(thing.Thing{Shape: mesh, Material: &thing.Material{Name: "authored ground apron and distant ridge beyond the DEM", Colour: groundColour}})
	return nil
}

// enrichELT gives the generic shell an exterior service structure: vertical
// bays, cross braces, access galleries and a meridian rail system. These
// are authored scenery; the optical model remains the native five-mirror rig.
func (s *heardStage) enrichELT(at maths.Vec) error {
	obj := stageObject{at: at, scale: 1, role: 3}
	rod := func(a, b maths.Vec, r float64) error {
		part, err := s.characterRod(a, b, r, "ELT enclosure service structure", maths.RGB(.002, .003, .004))
		if err != nil {
			return err
		}
		part.Material.Colour = maths.RGB(.24, .27, .3)
		obj.things = append(obj.things, part)
		obj.locals = append(obj.locals, part.At(0))
		return nil
	}
	point := func(a, y, r float64) maths.Vec { return maths.V(r*math.Sin(a), y, -r*math.Cos(a)) }
	for i := 0; i < 48; i++ {
		a := float64(i) * 2 * math.Pi / 48
		b := float64(i+1) * 2 * math.Pi / 48
		if a < .45 || a > 2*math.Pi-.45 {
			continue
		}
		for _, y := range []float64{6, 18, 30, 41} {
			if err := rod(point(a, y, 48.5), point(b, y, 48.5), .22); err != nil {
				return err
			}
			if y < 41 {
				if err := rod(point(a, y, 48.5), point(b, y+10, 48.5), .14); err != nil {
					return err
				}
			}
		}
		if err := rod(point(a, 2, 48.6), point(a, 42, 48.6), .28); err != nil {
			return err
		}
		if i%2 == 0 {
			for j := 0; j < 14; j++ {
				u := float64(j) * math.Pi / 28
				v := float64(j+1) * math.Pi / 28
				if err := rod(point(a, 42+48.4*math.Sin(u), 48.4*math.Cos(u)), point(a, 42+48.4*math.Sin(v), 48.4*math.Cos(v)), .24); err != nil {
					return err
				}
			}
		}
	}
	s.objects = append(s.objects, obj)
	for _, row := range []struct {
		name string
		at   maths.Vec
	}{{"access-stairs", maths.V(-55, 0, -20)}, {"mirror-gantry", maths.V(58, 0, 18)}, {"work-platform", maths.V(-52, 0, 22)}} {
		if err := s.factory(row.name, row.at.Add(at), 1, 0); err != nil {
			return err
		}
	}
	return nil
}
