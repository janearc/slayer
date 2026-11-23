package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

type squidThreeCoat struct{}

// At is the coat's red colour at a point, mottled by noise.
func (squidThreeCoat) At(p maths.Vec) maths.Light {
	n := noise.Fractal(731, p.X*31, p.Y*31, p.Z*31, 3)
	return maths.RGB(.33+.028*n, .095+.012*n, .09+.011*n)
}

// HumboldtSquidThree independently authors a shaped body, thin fins and
// curved muscular appendages. Separate overlapping parts are not welded skin.
func HumboldtSquidThree(phase float64) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("squid 03 invalid phase")
	}
	hide := &thing.Material{Name: "squid 03 mantle", Texture: squidThreeCoat{}, Colour: maths.RGB(.33, .095, .09), Satin: .45, Gloss: .13}
	finmat := &thing.Material{Name: "squid 03 fins", Colour: maths.RGB(.30, .07, .075), Satin: .45}
	ivory := &thing.Material{Name: "squid 03 illustrative suckers", Colour: maths.RGB(.60, .39, .32), Satin: .5}
	pupil := &thing.Material{Name: "squid 03 eyes", Colour: maths.RGB(.013, .016, .018), Gloss: .5}
	out := []thing.Thing{}
	// This independently built ring sweep produces stable point correspondence;
	// its body uses shaped sections and its appendages use new curved axes.
	build := func(n, m int, point func(float64, float64, bool) maths.Vec) (*mesh.Mesh, error) {
		rest, posed := make([]maths.Vec, 0, (n+1)*m+2), make([]maths.Vec, 0, (n+1)*m+2)
		faces := [][3]int{}
		for i := 0; i <= n; i++ {
			u := float64(i) / float64(n)
			for j := 0; j < m; j++ {
				a := 2 * math.Pi * float64(j) / float64(m)
				rest = append(rest, point(u, a, false))
				posed = append(posed, point(u, a, true))
			}
		}
		for i := 0; i < n; i++ {
			for j := 0; j < m; j++ {
				a, b := i*m+j, i*m+(j+1)%m
				c, d := (i+1)*m+(j+1)%m, (i+1)*m+j
				faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
			}
		}
		for _, i := range []int{0, n} {
			r, p := maths.Vec{}, maths.Vec{}
			for j := 0; j < m; j++ {
				r = r.Add(rest[i*m+j])
				p = p.Add(posed[i*m+j])
			}
			idx := len(rest)
			rest = append(rest, r.Scale(1/float64(m)))
			posed = append(posed, p.Scale(1/float64(m)))
			for j := 0; j < m; j++ {
				a, b := i*m+j, i*m+(j+1)%m
				if i == 0 {
					a, b = b, a
				}
				faces = append(faces, [3]int{idx, a, b})
			}
		}
		if err := orientBioformFaces(rest, faces); err != nil {
			return nil, err
		}
		return mesh.NewWithRestPoints(posed, rest, faces, sealSmoothNormals(posed, faces), true)
	}
	body, err := build(72, 72, func(u, a float64, moving bool) maths.Vec {
		x := -.80 + .98*u
		var r float64
		if x < 0 {
			v := math.Max(0, math.Min(1, (x+.80)/.80))
			r = .015 + .128*math.Pow(math.Sin(v*math.Pi/2), .8)
		} else {
			v := x / .18
			r = .11 * (1 - .38*v)
		}
		yy, zz := r*math.Sin(a), r*math.Cos(a)
		if moving {
			pulse := 1 + .045*math.Sin(phase)*math.Sin(math.Pi*math.Max(0, math.Min(1, (x+.8)/.8)))
			yy *= pulse
			zz *= pulse
		}
		return maths.V(x, yy, zz)
	})
	if err != nil {
		return nil, err
	}
	out = append(out, thing.Thing{Shape: body, Material: hide, Motion: thing.Still(maths.Local())})
	// Thin separate terminal fin sheets; diamond outline extends from tip -.80
	// to -.44 m with max span .225 m. Shell overlaps body and is not welded.
	for _, s := range []float64{-1, 1} {
		rest, posed := []maths.Vec{}, []maths.Vec{}
		for _, y := range []float64{-.006, .006} {
			for _, v := range [][2]float64{{-.80, .01}, {-.63, .225}, {-.44, .01}, {-.62, .08}} {
				p := maths.V(v[0], y, s*v[1])
				q := p
				q.Y += .075 * (v[1] / .225) * math.Sin(phase+.4)
				rest = append(rest, p)
				posed = append(posed, q)
			}
		}
		faces := [][3]int{{0, 2, 1}, {0, 3, 2}, {4, 5, 6}, {4, 6, 7}, {0, 1, 5}, {0, 5, 4}, {1, 2, 6}, {1, 6, 5}, {2, 3, 7}, {2, 7, 6}, {3, 0, 4}, {3, 4, 7}}
		if err := orientBioformFaces(rest, faces); err != nil {
			return nil, err
		}
		fin, e := mesh.NewWithRestPoints(posed, rest, faces, sealSmoothNormals(posed, faces), true)
		if e != nil {
			return nil, e
		}
		out = append(out, thing.Thing{Shape: fin, Material: finmat, Motion: thing.Still(maths.Local())})
	}
	for _, s := range []float64{-1, 1} {
		out = append(out, thing.Thing{Shape: shape.Sphere{Radius: .033}, Material: pupil, Motion: thing.Still(maths.NewSubspace(maths.V(.075, .008, s*.095), maths.Identity, 1))})
	}
	// A hollow cylinder is a visual opening, not a conical anatomical funnel.
	out = append(out, thing.Thing{Shape: shape.Tube{Radius: .025, Thick: .01, Height: .075}, Material: finmat, Motion: thing.Still(maths.NewSubspace(maths.V(.045, -.11, 0), maths.Identity, 1))})
	for k := 0; k < 10; k++ {
		a := math.Pi * float64(k%8) / 4
		isTent := k >= 8
		if isTent {
			a = math.Pi*.25 + math.Pi*float64(k-8)
		}
		length := .44
		if isTent {
			length = .68
		}
		axis := func(u float64, moving bool) maths.Vec {
			turn := .10 * math.Sin(u*math.Pi)
			if moving {
				turn += .27 * u * math.Sin(phase+.7*float64(k))
			}
			radius := .07 + .14*u
			return maths.V(.17+length*u, radius*math.Sin(a+turn), radius*math.Cos(a+turn))
		}
		arm, e := build(40, 16, func(u, t float64, moving bool) maths.Vec {
			p := axis(u, moving)
			radius := .038 * math.Pow(1-.95*u, 1.05)
			if isTent && u > .83 {
				radius += .020 * math.Sin(math.Pi*(u-.83)/.17)
			}
			return p.Add(maths.V(0, radius*math.Sin(t), radius*math.Cos(t)))
		})
		if e != nil {
			return nil, e
		}
		out = append(out, thing.Thing{Shape: arm, Material: hide, Motion: thing.Still(maths.Local())})
		for j := 0; j < 12; j++ {
			u := .12 + .065*float64(j)
			if isTent {
				u = .84 + .013*float64(j)
			}
			p := axis(u, true)
			r := .038 * math.Pow(1-.95*u, 1.05)
			for _, row := range []float64{-.20, .20} {
				dir := a + math.Pi + row
				pos := p.Add(maths.V(0, r*math.Sin(dir), r*math.Cos(dir)))
				out = append(out, thing.Thing{Shape: shape.Sphere{Radius: .0035}, Material: ivory, Motion: thing.Still(maths.NewSubspace(pos, maths.Identity, 1))})
			}
		}
	}
	return out, nil
}
