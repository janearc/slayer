package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// OrcaBioform constructs an original procedural study. The body, dorsal and
// pectoral surfaces share one closed mesh; the two flukes are separate meshes.
// The pose is prescribed, not a physical locomotion simulation.
func OrcaBioform(stroke float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(stroke) || math.IsInf(stroke, 0) || math.Abs(stroke) > 1 {
		return nil, fmt.Errorf("orca bioform: stroke must be in [-1,1]")
	}
	body, err := orcaBioSkin(stroke)
	if err != nil {
		return nil, err
	}
	coat := &thing.Material{Name: "orca coat in rest coordinates", Texture: orcaBioCoat{}, Gloss: .18, Satin: .55}
	tailMat := &thing.Material{Name: "separate dark flukes", Colour: maths.RGB(.025, .032, .038), Gloss: .18, Satin: .55}
	pieces := []thing.Thing{{Shape: body, Material: coat, Motion: thing.Still(at)}}
	for _, side := range []float64{-1, 1} {
		var rings [][]maths.Vec
		for _, q := range [][4]float64{{-3.10, .03, .19, .06}, {-3.15, .4, .32, .055}, {-3.23, .85, .27, .04}, {-3.29, 1.20, .10, .02}, {-3.30, 1.24, .01, .005}} {
			rings = append(rings, sealRing(maths.V(q[0], 0, side*q[1]), maths.V(q[2], 0, 0), maths.V(0, q[3], 0), func(p maths.Vec) maths.Vec { return orcaBioPose(p, stroke) }))
		}
		part, e := sealPiece(rings, tailMat, at)
		if e != nil {
			return nil, e
		}
		pieces = append(pieces, part)
	}
	return pieces, nil
}

// An authored section table, not measurements of an individual.
var orcaBioStations = [][3]float64{
	{-3.25, .055, .07}, {-3.03, .085, .10}, {-2.70, .15, .19}, {-2.15, .32, .39},
	{-1.50, .53, .60}, {-.80, .69, .76}, {0, .79, .86}, {.80, .81, .85},
	{1.50, .73, .76}, {2.15, .60, .65}, {2.70, .46, .50}, {3.04, .32, .35},
	{3.20, .18, .22}, {3.25, .06, .08},
}

// Monotone piecewise Hermite interpolation avoids flattening at every station.
func orcaBioRadius(x float64, axis int) float64 {
	n := len(orcaBioStations)
	i := 0
	for i < n-2 && x > orcaBioStations[i+1][0] {
		i++
	}
	gradient := func(j int) float64 {
		return (orcaBioStations[j+1][axis] - orcaBioStations[j][axis]) / (orcaBioStations[j+1][0] - orcaBioStations[j][0])
	}
	tangent := func(j int) float64 {
		if j == 0 {
			return gradient(0)
		}
		if j == n-1 {
			return gradient(n - 2)
		}
		a, b := gradient(j-1), gradient(j)
		if a*b <= 0 {
			return 0
		}
		return 2 * a * b / (a + b)
	}
	h := orcaBioStations[i+1][0] - orcaBioStations[i][0]
	t := (x - orcaBioStations[i][0]) / h
	t2 := t * t
	t3 := t2 * t
	return (2*t3-3*t2+1)*orcaBioStations[i][axis] + (t3-2*t2+t)*h*tangent(i) + (-2*t3+3*t2)*orcaBioStations[i+1][axis] + (t3-t2)*h*tangent(i+1)
}

// orcaBioPose bends a point of the rest body by the stroke: the tail half
// sweeps sideways, more the further aft.
func orcaBioPose(p maths.Vec, stroke float64) maths.Vec {
	t := math.Max(0, math.Min(1, (.6-p.X)/3.85))
	a := math.Atan(-.95 * stroke * t / 3.85)
	return maths.V(p.X-p.Y*math.Sin(a), p.Y*math.Cos(a)+.475*stroke*t*t, p.Z)
}

// OrcaBioformEye places the tiny dark eye on the unbent head surface.
func OrcaBioformEye(side float64) maths.Vec {
	x, y := 2.15, -.08
	ry, rz := orcaBioRadius(x, 1), orcaBioRadius(x, 2)
	return maths.V(x, y, side*(rz*math.Sqrt(1-y*y/(ry*ry))+.012))
}

// orcaBioSkin is the body's mesh, lofted in rest space and posed by the
// stroke, with the rest points kept.
func orcaBioSkin(stroke float64) (*mesh.Mesh, error) {
	const longitudinal, circumference = 128, 80
	rest := make([]maths.Vec, 0, (longitudinal+1)*circumference+2)
	posed := make([]maths.Vec, 0, cap(rest))
	faces := make([][3]int, 0, longitudinal*circumference*2+2*circumference)
	for i := 0; i <= longitudinal; i++ {
		x := -3.25 + 6.5*float64(i)/longitudinal
		ry, rz := orcaBioRadius(x, 1), orcaBioRadius(x, 2)
		for j := 0; j < circumference; j++ {
			a := 2 * math.Pi * float64(j) / circumference
			c, s := math.Cos(a), math.Sin(a)
			dorsalAngle := math.Abs(2 * math.Sin(a/2))
			dorsalCenter := -.56 + .28*math.Min(1, dorsalAngle/.18)
			dorsal := .68 * math.Exp(-math.Pow((x-dorsalCenter)/.36, 2)) * math.Exp(-math.Pow(dorsalAngle/.13, 2))
			angleDistance := math.Abs(c + .42)
			paddleCenter := .75 - .62*math.Max(0, 1-angleDistance/.27)
			paddle := .91 * math.Exp(-math.Pow((x-paddleCenter)/.44, 2)) * math.Exp(-math.Pow(angleDistance/.14, 2))
			p := maths.V(x, (ry+dorsal+paddle)*c, (rz+paddle)*s)
			rest = append(rest, p)
			posed = append(posed, orcaBioPose(p, stroke))
		}
	}
	for i := 0; i < longitudinal; i++ {
		for j := 0; j < circumference; j++ {
			a := i*circumference + j
			b := i*circumference + (j+1)%circumference
			c := b + circumference
			d := a + circumference
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	for _, i := range []int{0, longitudinal} {
		x := -3.25
		if i == longitudinal {
			x = 3.25
		}
		p := maths.V(x, 0, 0)
		n := len(rest)
		rest = append(rest, p)
		posed = append(posed, orcaBioPose(p, stroke))
		for j := 0; j < circumference; j++ {
			k := (j + 1) % circumference
			if i == 0 {
				faces = append(faces, [3]int{n, k, j})
			} else {
				faces = append(faces, [3]int{n, i*circumference + j, i*circumference + k})
			}
		}
	}
	var volume float64
	for _, f := range faces {
		volume += posed[f[0]].Dot(posed[f[1]].Cross(posed[f[2]]))
	}
	if volume < 0 {
		for i := range faces {
			faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
		}
	}
	return mesh.NewWithRestPoints(posed, rest, faces, sealSmoothNormals(posed, faces), true)
}

type orcaBioCoat struct{}

// At is the coat's colour at a point: black above, white on the belly and
// under the jaw, in the body's own frame.
func (orcaBioCoat) At(p maths.Vec) maths.Light {
	black, white := maths.RGB(.018, .025, .031), maths.RGB(.76, .79, .78)
	ry, rz := orcaBioRadius(p.X, 1), orcaBioRadius(p.X, 2)
	limit := -.64*ry + .25*(1-maths.Smooth(-2.4, -.65, p.X))
	belly := (1 - maths.Smooth(limit, limit+.07, p.Y)) * maths.Smooth(-2.95, -2.55, p.X) * (1 - maths.Smooth(2.85, 3.22, p.X)) * (1 - maths.Smooth(rz*.96, rz*1.05, math.Abs(p.Z)))
	patch := (1 - maths.Smooth(.78, 1, math.Hypot((p.X-1.85)/.48, (p.Y-.16)/.14))) * maths.Smooth(.28, .49, math.Abs(p.Z))
	saddle := (1 - maths.Smooth(.72, 1, math.Hypot((p.X+.83)/.28, (p.Y-.63)/.18))) * maths.Smooth(.5, .62, p.Y)
	c := black.Scale(1 - belly).Add(white.Scale(belly))
	c = c.Scale(1 - patch).Add(white.Scale(patch))
	return c.Scale(1 - saddle).Add(maths.RGB(.11, .14, .15).Scale(saddle))
}
