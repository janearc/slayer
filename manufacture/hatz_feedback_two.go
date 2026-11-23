package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// HatzFeedbackTwo builds a new faceted-control cage and independent wing rig.
// Membranes and body are separate shells; there is no animal-specific reuse.
func HatzFeedbackTwo(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("hatz 02: nonfinite phase")
	}
	body, err := hatzTwoBody()
	if err != nil {
		return nil, err
	}
	hide := &thing.Material{Name: "hatz 02 skin hypothesis", Colour: maths.RGB(.44, .35, .27), Satin: .43, Gloss: .08}
	membrane := &thing.Material{Name: "hatz 02 membrane hypothesis", Colour: maths.RGB(.32, .25, .20), Satin: .38, Gloss: .05}
	eye := &thing.Material{Name: "hatz 02 eye", Colour: maths.RGB(.015, .013, .012), Gloss: .25}
	pieces := []thing.Thing{{Shape: body, Material: hide, Motion: thing.Still(at)}}
	for _, side := range []float64{-1, 1} {
		j := hatzTwoJoints(phase, side)
		rest := hatzTwoJoints(0, side)
		wing, e := hatzTwoWing(j, rest, side)
		if e != nil {
			return nil, e
		}
		pieces = append(pieces, thing.Thing{Shape: wing, Material: membrane, Motion: thing.Still(at)})
		// Eyeballs are separate simplified shapes and have no socket morphology.
		pieces = append(pieces, thing.Thing{Shape: shape.Sphere{Radius: .045}, Material: eye, Motion: thing.Still(maths.NewSubspace(maths.V(1.85, .12, side*.17), maths.Identity, 1).Within(at))})
	}
	return pieces, nil
}

// hatzTwoBody is the second study's body: a control mesh refined and then
// lightly smoothed.
func hatzTwoBody() (*shape.Mesh, error) {
	v := []maths.Vec{maths.V(-1.28, 0, 0), maths.V(-.55, .46, 0), maths.V(-.55, -.39, 0), maths.V(-.55, 0, .36), maths.V(-.55, 0, -.36), maths.V(.45, .38, 0), maths.V(.45, -.32, 0), maths.V(.45, 0, .31), maths.V(.45, 0, -.31), maths.V(1.40, .18, 0), maths.V(1.40, -.14, 0), maths.V(1.40, 0, .14), maths.V(1.40, 0, -.14), maths.V(1.95, .25, 0), maths.V(1.95, -.18, 0), maths.V(1.95, 0, .18), maths.V(1.95, 0, -.18), maths.V(3.05, -.015, 0)}
	f := [][3]int{{0, 1, 3}, {0, 3, 2}, {0, 2, 4}, {0, 4, 1}, {1, 5, 7}, {1, 7, 3}, {3, 7, 6}, {3, 6, 2}, {2, 6, 8}, {2, 8, 4}, {4, 8, 5}, {4, 5, 1}, {5, 9, 11}, {5, 11, 7}, {7, 11, 10}, {7, 10, 6}, {6, 10, 12}, {6, 12, 8}, {8, 12, 9}, {8, 9, 5}, {9, 13, 15}, {9, 15, 11}, {11, 15, 14}, {11, 14, 10}, {10, 14, 16}, {10, 16, 12}, {12, 16, 13}, {12, 13, 9}, {13, 17, 15}, {15, 17, 14}, {14, 17, 16}, {16, 17, 13}}
	// Direct control mesh is independently refined, then lightly smoothed.
	for pass := 0; pass < 3; pass++ {
		type edge struct{ a, b int }
		mids := map[edge]int{}
		mid := func(a, b int) int {
			if a > b {
				a, b = b, a
			}
			key := edge{a, b}
			if i, ok := mids[key]; ok {
				return i
			}
			i := len(v)
			v = append(v, v[a].Add(v[b]).Scale(.5))
			mids[key] = i
			return i
		}
		nf := make([][3]int, 0, len(f)*4)
		for _, t := range f {
			a, b, c := mid(t[0], t[1]), mid(t[1], t[2]), mid(t[2], t[0])
			nf = append(nf, [3]int{t[0], a, c}, [3]int{a, t[1], b}, [3]int{c, b, t[2]}, [3]int{a, b, c})
		}
		f = nf
	}
	for pass := 0; pass < 2; pass++ {
		sum := make([]maths.Vec, len(v))
		count := make([]float64, len(v))
		for _, t := range f {
			for k := 0; k < 3; k++ {
				a, b := t[k], t[(k+1)%3]
				sum[a] = sum[a].Add(v[b])
				sum[b] = sum[b].Add(v[a])
				count[a]++
				count[b]++
			}
		}
		next := append([]maths.Vec{}, v...)
		for i := range v {
			if i != 0 && i != 17 && count[i] > 0 {
				next[i] = v[i].Scale(.75).Add(sum[i].Scale(.25 / count[i]))
			}
		}
		v = next
	}
	return hatzTwoMesh(v, f)
}

type hatzTwoMatrix [3][3]float64

// hatzTwoMult is the product of two three-by-three matrices.
func hatzTwoMult(a, b hatzTwoMatrix) (c hatzTwoMatrix) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return
}

// hatzTwoVec is a point turned by a matrix.
func hatzTwoVec(m hatzTwoMatrix, p maths.Vec) maths.Vec {
	return maths.V(m[0][0]*p.X+m[0][1]*p.Y+m[0][2]*p.Z, m[1][0]*p.X+m[1][1]*p.Y+m[1][2]*p.Z, m[2][0]*p.X+m[2][1]*p.Y+m[2][2]*p.Z)
}

// hatzTwoJoints is the joints of one wing at a phase in radians, found
// between four cyclic key poses.
func hatzTwoJoints(phase, side float64) []maths.Vec {
	// Cyclic key poses; these are not measurements of pterosaur flight.
	keys := [4][5]float64{{0, .60, 0, -.61, 0}, {0, -.17, 0, .20, 0}, {0, .13, 0, -.13, 0}, {0, .08, 0, -.10, 0}}
	u := math.Mod(phase/(2*math.Pi), 1)
	if u < 0 {
		u++
	}
	u *= 4
	i := int(u)
	t := u - float64(i)
	t = t * t * (3 - 2*t)
	frame := hatzTwoMatrix{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	p := maths.V(.26, .18, side*.30)
	out := []maths.Vec{p}
	for bone, length := range []float64{.86, 1.04, 1.13, 1.48} {
		a := side * (keys[bone][i]*(1-t) + keys[bone][i+1]*t)
		sweep := side * []float64{-.20, .08, .11, .08}[bone]
		rx := hatzTwoMatrix{{1, 0, 0}, {0, math.Cos(a), -math.Sin(a)}, {0, math.Sin(a), math.Cos(a)}}
		ry := hatzTwoMatrix{{math.Cos(sweep), 0, math.Sin(sweep)}, {0, 1, 0}, {-math.Sin(sweep), 0, math.Cos(sweep)}}
		frame = hatzTwoMult(frame, hatzTwoMult(ry, rx))
		p = p.Add(hatzTwoVec(frame, maths.V(0, 0, side*length)))
		out = append(out, p)
	}
	return out
}

// hatzTwoWing is the wing's mesh: a fan from the skeleton's joints and the
// hip, refined toward its centroid, posed from the rest joints.
func hatzTwoWing(joints, rest []maths.Vec, side float64) (*shape.Mesh, error) {
	// Explicit skeleton vertices lie on the boundary; the fan is independently
	// centroid-refined, unlike the prior regular rectangular/triangular grids.
	hip := maths.V(-1.0, -.20, side*.17)
	verts := append([]maths.Vec{}, joints...)
	verts = append(verts, hip)
	boundary := len(verts)
	center := maths.Vec{}
	for _, v := range verts {
		center = center.Add(v)
	}
	center = center.Scale(1 / float64(boundary))
	verts = append(verts, center)
	var tris [][3]int
	for i := 0; i < boundary; i++ {
		tris = append(tris, [3]int{i, (i + 1) % boundary, boundary})
	}
	for pass := 0; pass < 3; pass++ {
		next := make([][3]int, 0, len(tris)*3)
		for _, t := range tris {
			c := verts[t[0]].Add(verts[t[1]]).Add(verts[t[2]]).Scale(1. / 3)
			idx := len(verts)
			verts = append(verts, c)
			next = append(next, [3]int{t[0], t[1], idx}, [3]int{t[1], t[2], idx}, [3]int{t[2], t[0], idx})
		}
		tris = next
	}
	// Rest-length constraints use a separately evaluated reference pose with
	// identical topology; anisotropy is a numerical illustration, not measured.
	ref := append([]maths.Vec{}, rest...)
	ref = append(ref, hip)
	c := maths.Vec{}
	for _, v := range ref {
		c = c.Add(v)
	}
	ref = append(ref, c.Scale(1/float64(boundary)))
	reftris := make([][3]int, 0, boundary)
	for i := 0; i < boundary; i++ {
		reftris = append(reftris, [3]int{i, (i + 1) % boundary, boundary})
	}
	for pass := 0; pass < 3; pass++ {
		next := make([][3]int, 0, len(reftris)*3)
		for _, t := range reftris {
			c := ref[t[0]].Add(ref[t[1]]).Add(ref[t[2]]).Scale(1. / 3)
			idx := len(ref)
			ref = append(ref, c)
			next = append(next, [3]int{t[0], t[1], idx}, [3]int{t[1], t[2], idx}, [3]int{t[2], t[0], idx})
		}
		reftris = next
	}
	type edge struct{ a, b int }
	edges := map[edge]bool{}
	for _, t := range tris {
		for k := 0; k < 3; k++ {
			a, b := t[k], t[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			edges[edge{a, b}] = true
		}
	}
	// Stable edge ordering avoids randomized map iteration affecting the proof.
	var ordered []edge
	for e := range edges {
		ordered = append(ordered, e)
	}
	for i := 1; i < len(ordered); i++ {
		for j := i; j > 0 && (ordered[j].a < ordered[j-1].a || ordered[j].a == ordered[j-1].a && ordered[j].b < ordered[j-1].b); j-- {
			ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
		}
	}
	for pass := 0; pass < 18; pass++ {
		for _, e := range ordered {
			d := verts[e.b].Sub(verts[e.a])
			length := d.Len()
			if length < 1e-9 {
				continue
			}
			r := ref[e.b].Sub(ref[e.a]).Len()
			strength := .18
			if math.Abs(ref[e.a].Z-ref[e.b].Z) > math.Abs(ref[e.a].X-ref[e.b].X) {
				strength = .30
			}
			change := d.Scale(strength * (length - r) / length)
			if e.a >= boundary {
				verts[e.a] = verts[e.a].Add(change.Scale(.5))
			}
			if e.b >= boundary {
				verts[e.b] = verts[e.b].Sub(change.Scale(.5))
			}
		}
	}
	normal := joints[4].Sub(joints[0]).Cross(hip.Sub(joints[0])).Unit()
	if normal.Len() < .5 {
		normal = maths.Up
	}
	off := normal.Scale(.008)
	base := len(verts)
	shell := make([]maths.Vec, 0, 2*base)
	for _, p := range verts {
		shell = append(shell, p.Add(off))
	}
	for _, p := range verts {
		shell = append(shell, p.Sub(off))
	}
	var faces [][3]int
	for _, t := range tris {
		faces = append(faces, t, [3]int{t[0] + base, t[2] + base, t[1] + base})
	}
	for i := 0; i < boundary; i++ {
		j := (i + 1) % boundary
		faces = append(faces, [3]int{i, j, j + base}, [3]int{i, j + base, i + base})
	}
	return hatzTwoMesh(shell, faces)
}

// hatzTwoMesh is a mesh of vertices and faces, the faces turned to face
// out and the normals smoothed.
func hatzTwoMesh(v []maths.Vec, f [][3]int) (*shape.Mesh, error) {
	if err := orientBioformFaces(v, f); err != nil {
		return nil, err
	}
	normal := make([]maths.Vec, len(v))
	for _, t := range f {
		n := v[t[1]].Sub(v[t[0]]).Cross(v[t[2]].Sub(v[t[0]]))
		for _, i := range t {
			normal[i] = normal[i].Add(n)
		}
	}
	return shape.NewMesh(v, f, normal, true)
}
