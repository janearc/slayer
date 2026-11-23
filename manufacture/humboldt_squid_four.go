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

type squidFourPath struct {
	p        [4]maths.Vec
	tentacle bool
}

// squidFourBezier is the point at u, from nought to one, on a cubic Bezier
// curve through four control points.
func squidFourBezier(p [4]maths.Vec, u float64) maths.Vec {
	v := 1 - u
	return p[0].Scale(v * v * v).Add(p[1].Scale(3 * v * v * u)).Add(p[2].Scale(3 * v * u * u)).Add(p[3].Scale(u * u * u))
}

// squidFourPaths is the Bezier paths of the arms and tentacles, drawn in a
// frame of two units to the metre.
func squidFourPaths() []squidFourPath {
	paths := []squidFourPath{}
	// Independently drawn Bezier trajectories. Dimensions are authored in a
	// two-units-per-metre construction frame, not measured specimen landmarks.
	for k := 0; k < 10; k++ {
		angle := (float64(k%8) + .12) * math.Pi / 4
		if k >= 8 {
			angle = math.Pi*.30 + math.Pi*float64(k-8)
		}
		tent := k >= 8
		end := 1.11
		if tent {
			end = 1.60
		}
		polar := func(x, r, offset float64) maths.Vec {
			return maths.V(x, r*math.Sin(angle+offset), r*math.Cos(angle+offset))
		}
		paths = append(paths, squidFourPath{p: [4]maths.Vec{polar(.22, .09, 0), polar(.50, .13, .08), polar(.78+(end-1.1)*.5, .25, .12), polar(end, .31, .03)}, tentacle: tent})
	}
	return paths
}

var squidFourRest = sync.OnceValues(func() (*shape.Mesh, error) {
	paths := squidFourPaths()
	field := func(p maths.Vec) float64 {
		// A superelliptic axial body silhouette, blended with an independent head.
		u := (p.X + 1.60) / 1.60
		cl := math.Max(0, math.Min(1, u))
		width := .018 + .26*math.Pow(math.Sin(cl*math.Pi/2), .72)
		mantle := math.Max(math.Hypot(p.Y, p.Z)-width, math.Max(-1.60-p.X, p.X))
		h := math.Sqrt(math.Pow((p.X-.10)/.25, 2) + math.Pow(p.Y/.20, 2) + math.Pow(p.Z/.22, 2))
		head := (h - 1) * .20
		blend := func(a, b, k float64) float64 { v := math.Max(k-math.Abs(a-b), 0) / k; return math.Min(a, b) - k*v*v/4 }
		d := blend(mantle, head, .10)
		// Broad, thin terminal fin bounded by an independently authored rounded
		// diamond; its body overlap is unioned with the skin.
		// A rounded superelliptic planform, independently chosen to reach
		// the tail tip and blend across the mantle rather than the prior
		// study's linearly clipped diamond field.
		fx := math.Abs((p.X + 1.22) / .38)
		fz := math.Abs(p.Z / .46)
		outline := (math.Pow(math.Pow(fx, 1.35)+math.Pow(fz, 1.35), 1/1.35) - 1) * .27
		fin := math.Max(math.Abs(p.Y)-.027, outline)
		d = blend(d, fin, .03)
		for _, path := range paths {
			best := math.Inf(1)
			for j := 0; j < 25; j++ {
				t := float64(j) / 24
				q := squidFourBezier(path.p, t)
				r := .070 * (1 - .45*t)
				if path.tentacle && t > .75 {
					r += .028 * math.Sin(math.Pi*(t-.75)/.25)
				}
				best = math.Min(best, p.Sub(q).Len()-r)
			}
			d = blend(d, best, .032)
		}
		return d
	}
	return bioformFieldMeshFine(field, func(p maths.Vec) maths.Vec { return p })
})

type squidFourTexture struct{}

// At is the rest-space colour at a point: a dark eye at the side of the
// head, and mottling elsewhere.
func (squidFourTexture) At(p maths.Vec) maths.Light {
	// Rest-space illustration: dark ellipse at the side of the head, not a
	// scanned eye; subtle procedural mottling is not a chromatophore simulation.
	eye := math.Pow((p.X-.055)/.023, 2) + math.Pow((p.Y-.005)/.020, 2)
	if eye < 1 && math.Abs(p.Z) > .07 {
		return maths.RGB(.012, .015, .019)
	}
	n := noise.Fractal(837, p.X*36, p.Y*36, p.Z*36, 3)
	return maths.RGB(.34+.026*n, .105+.012*n, .10+.011*n)
}

// squidFourPose moves a point of the rest body at a phase in radians:
// rotations about the long axis, with their own windows for the fins and
// the ends of the appendages.
func squidFourPose(p maths.Vec, phase float64) maths.Vec {
	q := p
	// Newly chosen rotations around the animal's long axis, with independent
	// support windows for fins and the distal appendages. Not a tissue solver.
	fin := math.Max(0, 1-math.Abs((p.X+1.18)/.42)) * math.Max(0, math.Min(1, (math.Abs(p.Z)-.13)/.20))
	arm := math.Max(0, math.Min(1, (p.X-.25)/.9))
	// Smooth directional weighting avoids an angular branch cut at ±π.
	direction := (math.Sin(phase)*p.Z + math.Cos(phase)*p.Y) / (.35 + math.Hypot(p.Y, p.Z))
	theta := .22*fin*math.Sin(phase+.3) + .18*arm*direction
	q.Y = p.Y*math.Cos(theta) - p.Z*math.Sin(theta)
	q.Z = p.Y*math.Sin(theta) + p.Z*math.Cos(theta)
	// An authored pulse changes surface radii; no displaced water is calculated.
	body := math.Max(0, 1-math.Abs((p.X+.70)/.85))
	q.Y *= 1 + .04*body*math.Sin(phase)
	q.Z *= 1 + .04*body*math.Sin(phase)
	return q.Scale(.5)
}

// HumboldtSquidFour is an independent continuous-field reconstruction.
func HumboldtSquidFour(phase float64) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("squid 04 invalid phase")
	}
	raw, err := squidFourRest()
	if err != nil {
		return nil, err
	}
	rest, posed := make([]maths.Vec, len(raw.Vertices)), make([]maths.Vec, len(raw.Vertices))
	for i, p := range raw.Vertices {
		rest[i] = p.Scale(.5)
		posed[i] = squidFourPose(p, phase)
	}
	shell, err := mesh.NewWithRestPoints(posed, rest, raw.Faces, sealSmoothNormals(posed, raw.Faces), true)
	if err != nil {
		return nil, err
	}
	hide := &thing.Material{Name: "squid 04 skin", Texture: squidFourTexture{}, Colour: maths.RGB(.34, .105, .10), Satin: .46, Gloss: .1}
	ivory := &thing.Material{Name: "squid 04 illustrative suckers", Colour: maths.RGB(.56, .37, .31), Satin: .5}
	out := []thing.Thing{{Shape: shell, Material: hide, Motion: thing.Still(maths.Local())}}
	// Surface-nearest decorative points track a real indexed skin vertex.
	// These spheres remain a coarse suggestion, not detailed biological suckers.
	nearest := func(q maths.Vec) int {
		best := math.Inf(1)
		index := 0
		for i, p := range raw.Vertices {
			d := p.Sub(q).Len()
			if d < best {
				best = d
				index = i
			}
		}
		return index
	}
	for _, path := range squidFourPaths() {
		for j := 0; j < 8; j++ {
			u := .16 + .078*float64(j)
			if path.tentacle {
				u = .78 + .027*float64(j)
			}
			center := squidFourBezier(path.p, u)
			for _, off := range []float64{-.014, .014} {
				q := center
				q.Y -= .035
				q.Z += off
				v := nearest(q)
				out = append(out, thing.Thing{Shape: shape.Sphere{Radius: .003}, Material: ivory, Motion: thing.Still(maths.NewSubspace(posed[v], maths.Identity, 1))})
			}
		}
	}
	return out, nil
}
