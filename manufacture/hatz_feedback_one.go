package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// HatzFeedbackOne is an illustrative, independently authored study. Its
// separate closed shells are not a unified animal skin or tissue mechanics.
func HatzFeedbackOne(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("hatz 01: nonfinite phase")
	}
	body, err := hatzOneBody()
	if err != nil {
		return nil, err
	}
	hide := &thing.Material{Name: "hatz illustrative body", Colour: maths.RGB(.43, .35, .28), Satin: .45, Gloss: .08}
	wing := &thing.Material{Name: "hatz illustrative membrane", Colour: maths.RGB(.34, .27, .21), Satin: .42, Gloss: .06}
	dark := &thing.Material{Name: "hatz eyes", Colour: maths.RGB(.015, .013, .011), Gloss: .25}
	parts := []thing.Thing{{Shape: body, Material: hide, Motion: thing.Still(at)}}
	for _, s := range []float64{-1, 1} {
		joints := hatzOneWingJoints(phase, s)
		membrane, e := hatzOneMembrane(joints, s)
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: membrane, Material: wing, Motion: thing.Still(at)})
		for j := 0; j < len(joints)-1; j++ {
			bone, e := hatzOneBeam(joints[j], joints[j+1], .055*(1-.13*float64(j)))
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: bone, Material: hide, Motion: thing.Still(at)})
		}
		hip := maths.V(-.62, -.22, s*.20)
		knee := maths.V(-.78, -.87, s*.28)
		foot := maths.V(-.60, -1.32, s*.29)
		for _, pair := range [][2]maths.Vec{{hip, knee}, {knee, foot}} {
			leg, e := hatzOneBeam(pair[0], pair[1], .075)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: leg, Material: hide, Motion: thing.Still(at)})
		}
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .052}, Material: dark, Motion: thing.Still(maths.NewSubspace(maths.V(1.82, .13, s*.12), maths.Identity, 1).Within(at))})
	}
	return parts, nil
}

// hatzOneBody is the first study's body: a mesh built on a hand-placed side
// outline with two thickness points.
func hatzOneBody() (*shape.Mesh, error) {
	// A hand-placed side-outline with two thickness control points, not an
	// animal profile copied from a prior build. Early rough silhouette study.
	outline := []maths.Vec{maths.V(3.15, .01, 0), maths.V(2.10, .25, 0), maths.V(1.73, .23, 0), maths.V(.78, .20, 0), maths.V(.48, .48, 0), maths.V(-.55, .43, 0), maths.V(-1.15, .16, 0), maths.V(-1.10, -.24, 0), maths.V(-.48, -.42, 0), maths.V(.42, -.34, 0), maths.V(.76, -.13, 0), maths.V(1.70, -.10, 0), maths.V(2.13, -.12, 0)}
	verts := append([]maths.Vec{}, outline...)
	verts = append(verts, maths.V(-.15, 0, .35), maths.V(-.15, 0, -.35))
	n := len(outline)
	var faces [][3]int
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		faces = append(faces, [3]int{n, i, j}, [3]int{n + 1, j, i})
	}
	return hatzOneMesh(verts, faces)
}

// hatzOneWingJoints is the joints of one wing from the shoulder to the
// tip, each swung a little later in the phase than the one before.
func hatzOneWingJoints(phase, side float64) []maths.Vec {
	p := maths.V(.25, .22, side*.28)
	out := []maths.Vec{p}
	lengths := []float64{.85, 1.05, 1.15, 1.45}
	for i, l := range lengths {
		a := .65 * math.Sin(phase-.12*float64(i))
		sweep := []float64{.22, .06, -.08, -.16}[i]
		d := maths.V(sweep, math.Sin(a), side*math.Cos(a)).Unit()
		p = p.Add(d.Scale(l))
		out = append(out, p)
	}
	return out
}

// hatzOneMembrane is the wing's membrane as a triangle mesh stretched
// between the joint chain, the torso and the trailing edge.
func hatzOneMembrane(joints []maths.Vec, side float64) (*shape.Mesh, error) {
	const n = 18
	// Triangular barycentric topology: one edge follows the full joint chain,
	// one follows the torso, and the third is the wing's free trailing edge.
	idx := map[[2]int]int{}
	var pts []maths.Vec
	var ij [][2]int
	lead := func(t float64) maths.Vec {
		u := math.Min(float64(len(joints)-1)-1e-9, t*float64(len(joints)-1))
		i := int(u)
		return joints[i].Scale(1 - (u - float64(i))).Add(joints[i+1].Scale(u - float64(i)))
	}
	hip := maths.V(-1.05, -.20, side*.22)
	for i := 0; i <= n; i++ {
		for j := 0; j <= n-i; j++ {
			r := float64(j) / n
			t := 0.
			if j < n {
				t = float64(i) / float64(n-j)
			}
			p := lead(t).Scale(1 - r).Add(hip.Scale(r))
			idx[[2]int{i, j}] = len(pts)
			pts = append(pts, p)
			ij = append(ij, [2]int{i, j})
		}
	}
	// Directional relaxation is a visual analogy, not a stress solver.
	for pass := 0; pass < 24; pass++ {
		next := append([]maths.Vec{}, pts...)
		for k, q := range ij {
			i, j := q[0], q[1]
			if i == 0 || j == 0 || i+j == n {
				continue
			}
			sum := maths.Vec{}
			weight := 0.
			for _, d := range [][3]int{{1, 0, 3}, {-1, 0, 3}, {0, 1, 1}, {0, -1, 1}, {1, -1, 2}, {-1, 1, 2}} {
				if v, ok := idx[[2]int{i + d[0], j + d[1]}]; ok {
					w := float64(d[2])
					sum = sum.Add(pts[v].Scale(w))
					weight += w
				}
			}
			next[k] = pts[k].Scale(.4).Add(sum.Scale(.6 / weight))
		}
		pts = next
	}
	var triangles [][3]int
	for i := 0; i < n; i++ {
		for j := 0; j < n-i; j++ {
			a, b, c := idx[[2]int{i, j}], idx[[2]int{i + 1, j}], idx[[2]int{i, j + 1}]
			triangles = append(triangles, [3]int{a, b, c})
			if i+j < n-1 {
				d := idx[[2]int{i + 1, j + 1}]
				triangles = append(triangles, [3]int{b, d, c})
			}
		}
	}
	normal := joints[len(joints)-1].Sub(joints[0]).Cross(hip.Sub(joints[0])).Unit()
	if normal.Len() < .5 {
		normal = maths.Up
	}
	off := normal.Scale(.009)
	count := len(pts)
	var verts []maths.Vec
	for _, p := range pts {
		verts = append(verts, p.Add(off))
	}
	for _, p := range pts {
		verts = append(verts, p.Sub(off))
	}
	var faces [][3]int
	type edge struct{ a, b int }
	edgeCount := map[edge]int{}
	for _, f := range triangles {
		faces = append(faces, f, [3]int{f[0] + count, f[2] + count, f[1] + count})
		for k := 0; k < 3; k++ {
			a, b := f[k], f[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			edgeCount[edge{a, b}]++
		}
	}
	for e, c := range edgeCount {
		if c == 1 {
			faces = append(faces, [3]int{e.a, e.b, e.b + count}, [3]int{e.a, e.b + count, e.a + count})
		}
	}
	return hatzOneMesh(verts, faces)
}

// hatzOneBeam is a beam mesh of radius r from one point to another.
func hatzOneBeam(a, b maths.Vec, r float64) (*shape.Mesh, error) {
	axis := b.Sub(a).Unit()
	u := axis.Cross(maths.Up)
	if u.Len() < .1 {
		u = axis.Cross(maths.V(1, 0, 0))
	}
	u = u.Unit().Scale(r)
	v := axis.Cross(u).Unit().Scale(r)
	pts := []maths.Vec{a.Add(u), a.Add(v), a.Sub(u), a.Sub(v), b.Add(u.Scale(.7)), b.Add(v.Scale(.7)), b.Sub(u.Scale(.7)), b.Sub(v.Scale(.7))}
	var faces [][3]int
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		faces = append(faces, [3]int{i, j, 4 + j}, [3]int{i, 4 + j, 4 + i})
	}
	faces = append(faces, [3]int{0, 2, 1}, [3]int{0, 3, 2}, [3]int{4, 5, 6}, [3]int{4, 6, 7})
	return hatzOneMesh(pts, faces)
}

// hatzOneMesh is a mesh of vertices and faces, the faces turned to face
// out and the normals smoothed.
func hatzOneMesh(verts []maths.Vec, faces [][3]int) (*shape.Mesh, error) {
	if err := orientBioformFaces(verts, faces); err != nil {
		return nil, err
	}
	normals := make([]maths.Vec, len(verts))
	for _, f := range faces {
		a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
		norm := b.Sub(a).Cross(c.Sub(a))
		for _, i := range f {
			normals[i] = normals[i].Add(norm)
		}
	}
	return shape.NewMesh(verts, faces, normals, true)
}
