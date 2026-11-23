package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"sync"
)

// This final fresh construction uses a sector-defined appendage field, not
// the earlier path, voxel or ring-assembly animal models. Two units are 1 m.
var squidFiveRest = sync.OnceValues(func() (*shape.Mesh, error) {
	field := func(p maths.Vec) float64 {
		x := p.X
		rad := math.Hypot(p.Y, p.Z)
		u := math.Max(0, math.Min(1, (x+1.6)/1.6))
		// A smooth polynomial section, chosen as an illustrative .8 m mantle.
		w := .018 + .255*(u*u*(3-2*u))
		mantle := math.Max(rad-w, math.Max(-1.6-x, x-.01))
		head := (math.Sqrt(math.Pow((x-.12)/.26, 2)+math.Pow(p.Y/.20, 2)+math.Pow(p.Z/.21, 2)) - 1) * .20
		join := func(a, b float64) float64 {
			h := math.Max(.075-math.Abs(a-b), 0) / .075
			return math.Min(a, b) - .01875*h*h
		}
		d := join(mantle, head)
		// Four independent soft fin lobes approximate one continuous rounded fin
		// complex. Their dimensions are design targets, not measured silhouettes.
		for _, side := range []float64{-1, 1} {
			for _, xc := range []float64{-1.36, -1.08} {
				v := math.Sqrt(math.Pow((x-xc)/.23, 2) + math.Pow(p.Y/.035, 2) + math.Pow((p.Z-side*.22)/.25, 2))
				d = join(d, (v-1)*.035)
			}
		}
		// Arm centers are angular sectors of the head cross section. This is a
		// distinct analytic arrangement rather than prior swept Bezier models.
		theta := math.Atan2(p.Y, p.Z)
		nearest := math.Round(theta/(math.Pi/4)) * math.Pi / 4
		armU := math.Max(0, math.Min(1, (x-.16)/1.0))
		armRadius := .10 + .16*armU
		cy, cz := armRadius*math.Sin(nearest), armRadius*math.Cos(nearest)
		armWidth := .065 * (1 - .73*armU)
		arm := math.Max(math.Hypot(p.Y-cy, p.Z-cz)-armWidth, math.Max(.14-x, x-1.16))
		d = join(d, arm)
		// The two longer sectors end in extended clubs. These are authored, not a
		// prediction of the orientation of a particular living squid.
		for _, a := range []float64{math.Pi / 8, math.Pi + math.Pi/8} {
			t := math.Max(0, math.Min(1, (x-.16)/1.48))
			r := .10 + .17*t
			cy, cz := r*math.Sin(a), r*math.Cos(a)
			width := .032 * (1 - .35*t)
			if t > .78 {
				width += .045 * math.Sin(math.Pi*(t-.78)/.22)
			}
			tent := math.Max(math.Hypot(p.Y-cy, p.Z-cz)-width, math.Max(.14-x, x-1.64))
			d = join(d, tent)
		}
		return d
	}
	return bioformFieldMeshFine(field, func(p maths.Vec) maths.Vec { return p })
})

type squidFiveCoat struct{}

// At is the coat's reddish-brown colour at a point, mottled by noise.
func (squidFiveCoat) At(p maths.Vec) maths.Light {
	n := noise.Fractal(1711, p.X*35, p.Y*35, p.Z*35, 4)
	return maths.RGB(.32+.035*n, .09+.018*n, .085+.015*n)
}

// squidFivePose moves a point of the rest body by a displacement cage at a
// phase in radians.
func squidFivePose(p maths.Vec, phase float64) maths.Vec {
	// A new tensor-product Bernstein displacement cage; its authored nodal
	// movements do not represent measured mantle or appendage tissue forces.
	bern := func(t float64) [3]float64 {
		t = math.Max(0, math.Min(1, t))
		return [3]float64{(1 - t) * (1 - t), 2 * t * (1 - t), t * t}
	}
	bx := bern((p.X + 1.7) / 3.5)
	by := bern((p.Y + .55) / 1.1)
	bz := bern((p.Z + .65) / 1.3)
	delta := maths.Vec{}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				w := bx[i] * by[j] * bz[k]
				x, y, z := float64(i-1), float64(j-1), float64(k-1)
				delta = delta.Add(maths.V(.015*x, .11*y*math.Sin(phase+.55*x), .085*z*math.Sin(phase+.45*x)).Scale(w))
			}
		}
	}
	return p.Add(delta).Scale(.5)
}

// HumboldtSquidFive is an authored single-field skin and separate details.
func HumboldtSquidFive(phase float64) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("squid 05 invalid phase")
	}
	raw, err := squidFiveRest()
	if err != nil {
		return nil, err
	}
	rest, posed := make([]maths.Vec, len(raw.Vertices)), make([]maths.Vec, len(raw.Vertices))
	for i, p := range raw.Vertices {
		rest[i] = p.Scale(.5)
		posed[i] = squidFivePose(p, phase)
	}
	skin, err := mesh.NewWithRestPoints(posed, rest, raw.Faces, sealSmoothNormals(posed, raw.Faces), true)
	if err != nil {
		return nil, err
	}
	hide := &thing.Material{Name: "squid 05 illustrated skin", Texture: squidFiveCoat{}, Colour: maths.RGB(.32, .09, .085), Satin: .47, Gloss: .12}
	dark := &thing.Material{Name: "squid 05 eye", Colour: maths.RGB(.012, .015, .019), Gloss: .45}
	pale := &thing.Material{Name: "squid 05 illustrative sucker details", Colour: maths.RGB(.56, .37, .31), Satin: .45}
	out := []thing.Thing{{Shape: skin, Material: hide, Motion: thing.Still(maths.Local())}}
	// Choose nearest skin vertices for visible eyes; roots follow actual indexed
	// points, but round eye geometry and surface contact remain approximations.
	closest := func(q maths.Vec) int {
		best := math.Inf(1)
		at := 0
		for i, p := range raw.Vertices {
			v := p.Sub(q).Len()
			if v < best {
				best = v
				at = i
			}
		}
		return at
	}
	for _, side := range []float64{-1, 1} {
		i := closest(maths.V(.12, .02, side*.20))
		out = append(out, thing.Thing{Shape: shape.Sphere{Radius: .018}, Material: dark, Motion: thing.Still(maths.NewSubspace(posed[i], maths.Identity, 1))})
	}
	// A visible hollow tube suggests a funnel opening; it is separate from the
	// surface and is not an anatomical canal or a jet simulation.
	out = append(out, thing.Thing{Shape: shape.Tube{Radius: .020, Thick: .008, Height: .05}, Material: hide, Motion: thing.Still(maths.NewSubspace(maths.V(.035, -.105, 0), maths.Identity, 1))})
	// Decorative two-row samples on each arm and four on club sectors. These
	// coarse beads do not reproduce true counts, toothed rings or suction.
	for k := 0; k < 10; k++ {
		angle := float64(k%8) * math.Pi / 4
		rows := 2
		if k >= 8 {
			angle = math.Pi/8 + math.Pi*float64(k-8)
			rows = 4
		}
		for j := 0; j < 12; j++ {
			u := .18 + .052*float64(j)
			x := .16 + u
			if k >= 8 {
				x = 1.28 + .025*float64(j)
			}
			radius := .10 + .16*math.Min(1, u)
			for row := 0; row < rows; row++ {
				offset := (float64(row) - float64(rows-1)/2) * .012
				q := maths.V(x, (radius-.035)*math.Sin(angle)+offset*math.Cos(angle), (radius-.035)*math.Cos(angle)-offset*math.Sin(angle))
				i := closest(q)
				out = append(out, thing.Thing{Shape: shape.Sphere{Radius: .003}, Material: pale, Motion: thing.Still(maths.NewSubspace(posed[i], maths.Identity, 1))})
			}
		}
	}
	return out, nil
}
