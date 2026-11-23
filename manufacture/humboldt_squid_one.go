package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// HumboldtSquidOne is a first original, illustrative muscular-hydrostat study.
// Separate intersecting pieces do not constitute a welded anatomical skin.
func HumboldtSquidOne(phase float64) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("squid 01: invalid phase")
	}
	skin := &thing.Material{Name: "squid mantle", Colour: maths.RGB(.34, .105, .10), Satin: .48, Gloss: .12}
	finmat := &thing.Material{Name: "squid fins", Colour: maths.RGB(.30, .07, .07), Satin: .43, Gloss: .10}
	eye := &thing.Material{Name: "squid eye", Colour: maths.RGB(.012, .016, .018), Gloss: .55}
	sucker := &thing.Material{Name: "illustrative suckers", Colour: maths.RGB(.57, .36, .30), Satin: .5}
	parts := []thing.Thing{}
	// Build closed indexed rings, with stable coordinates on the unposed mesh.
	tube := func(rings, sides int, at func(float64, float64, bool) maths.Vec) (*mesh.Mesh, error) {
		rest, posed := []maths.Vec{}, []maths.Vec{}
		faces := [][3]int{}
		for i := 0; i <= rings; i++ {
			u := float64(i) / float64(rings)
			for j := 0; j < sides; j++ {
				a := 2 * math.Pi * float64(j) / float64(sides)
				rest = append(rest, at(u, a, false))
				posed = append(posed, at(u, a, true))
			}
		}
		for i := 0; i < rings; i++ {
			for j := 0; j < sides; j++ {
				n := (j + 1) % sides
				a, b := i*sides+j, i*sides+n
				c, d := (i+1)*sides+n, (i+1)*sides+j
				faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
			}
		}
		// Centers close the tiny end rings; coordinate averages avoid assumptions
		// about which axis an appendage occupies.
		for _, end := range []int{0, rings} {
			r, p := maths.Vec{}, maths.Vec{}
			for j := 0; j < sides; j++ {
				r = r.Add(rest[end*sides+j])
				p = p.Add(posed[end*sides+j])
			}
			r = r.Scale(1 / float64(sides))
			p = p.Scale(1 / float64(sides))
			c := len(rest)
			rest = append(rest, r)
			posed = append(posed, p)
			for j := 0; j < sides; j++ {
				n := (j + 1) % sides
				if end == 0 {
					faces = append(faces, [3]int{c, n, j})
				} else {
					faces = append(faces, [3]int{c, end*sides + j, end*sides + n})
				}
			}
		}
		if err := orientBioformFaces(rest, faces); err != nil {
			return nil, err
		}
		return mesh.NewWithRestPoints(posed, rest, faces, sealSmoothNormals(posed, faces), true)
	}
	mantle, err := tube(40, 48, func(u, a float64, moving bool) maths.Vec {
		x := -.80 * u
		r := .14 * math.Sqrt(math.Max(.002, 1-u*u))
		if moving {
			r *= 1 + .055*math.Sin(phase)*math.Sin(math.Pi*u)
		}
		return maths.V(x, r*math.Sin(a), r*math.Cos(a))
	})
	if err != nil {
		return nil, err
	}
	parts = append(parts, thing.Thing{Shape: mantle, Material: skin, Motion: thing.Still(maths.Local())})
	// The fin planform is a diamond. This closed thin prism overlaps the mantle.
	for _, s := range []float64{-1, 1} {
		rest := []maths.Vec{}
		posed := []maths.Vec{}
		for _, y := range []float64{-.008, .008} {
			for _, v := range [][2]float64{{-.75, 0}, {-.57, .225}, {-.39, 0}, {-.57, .055}} {
				p := maths.V(v[0], y, s*v[1])
				q := p
				q.Y += .10 * math.Sin(phase+.3) * v[1] / .225
				rest = append(rest, p)
				posed = append(posed, q)
			}
		}
		faces := [][3]int{{0, 2, 1}, {0, 3, 2}, {4, 5, 6}, {4, 6, 7}, {0, 1, 5}, {0, 5, 4}, {1, 2, 6}, {1, 6, 5}, {2, 3, 7}, {2, 7, 6}, {3, 0, 4}, {3, 4, 7}}
		if err := orientBioformFaces(rest, faces); err != nil {
			return nil, err
		}
		f, e := mesh.NewWithRestPoints(posed, rest, faces, sealSmoothNormals(posed, faces), true)
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: f, Material: finmat, Motion: thing.Still(maths.Local())})
	}
	parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .125}, Material: skin, Motion: thing.Still(maths.NewSubspace(maths.V(.08, 0, 0), maths.Identity, 1))})
	for _, s := range []float64{-1, 1} {
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .052}, Material: eye, Motion: thing.Still(maths.NewSubspace(maths.V(.10, .015, s*.105), maths.Identity, 1))})
	}
	// A small ventral funnel is a separate visible approximation.
	parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .047}, Material: finmat, Motion: thing.Still(maths.NewSubspace(maths.V(.035, -.105, 0), maths.Identity, 1))})
	for k := 0; k < 10; k++ {
		tentacle := k >= 8
		angle := 2 * math.Pi * float64(k%8) / 8
		if tentacle {
			angle = math.Pi/4 + math.Pi*float64(k-8)
		}
		length := .43
		if tentacle {
			length = .66
		}
		center := func(u float64, moving bool) maths.Vec {
			radial := .065 + .12*u
			a := angle + .10*u
			if moving {
				a += .17 * u * math.Sin(phase+float64(k)*.55)
			}
			return maths.V(.16+length*u, radial*math.Sin(a), radial*math.Cos(a))
		}
		arm, e := tube(24, 12, func(u, a float64, moving bool) maths.Vec {
			p := center(u, moving)
			r := .025 * (1 - .78*u)
			if tentacle && u > .80 {
				r += .018 * math.Sin(math.Pi*(u-.8)/.2)
			}
			return p.Add(maths.V(0, r*math.Sin(a), r*math.Cos(a)))
		})
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: arm, Material: skin, Motion: thing.Still(maths.Local())})
		for j := 1; j <= 8; j++ {
			u := .15 + .075*float64(j)
			if tentacle {
				u = .82 + .019*float64(j)
			}
			p := center(u, true)
			p.Y -= .014 * math.Sin(angle)
			p.Z -= .014 * math.Cos(angle)
			parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .006}, Material: sucker, Motion: thing.Still(maths.NewSubspace(p, maths.Identity, 1))})
		}
	}
	return parts, nil
}
