package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// HatzCage is an authored reconstruction built around a subdivided skin cage
// and numerical endpoint-constrained control chains, not fossil motion data.
func HatzCage(fold, seconds float64, at maths.Subspace) ([]thing.Thing, error) {
	if fold < 0 || fold > 1 || math.IsNaN(fold) || math.IsInf(fold, 0) || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return nil, fmt.Errorf("hatz cage: invalid pose")
	}
	hide := &thing.Material{Name: "invented hide", Colour: maths.RGB(.28, .26, .19), Satin: .45}
	cloth := &thing.Material{Name: "inferred membrane", Colour: maths.RGB(.30, .19, .13), Satin: .48}
	dark := &thing.Material{Name: "invented claws and eyes", Colour: maths.RGB(.07, .05, .035), Satin: .3}
	body, e := hatzCageBody(seconds)
	if e != nil {
		return nil, e
	}
	pieces := []thing.Thing{{Shape: body, Material: hide, Motion: thing.Still(at)}}
	bone := func(a, b maths.Vec, r float64, mat *thing.Material) error {
		axis := b.Sub(a)
		u := axis.Cross(maths.V(0, 0, 1))
		if u.Len() < 1e-8 {
			u = axis.Cross(maths.Up)
		}
		u = u.Unit().Scale(r)
		v := axis.Cross(u).Unit().Scale(r)
		piece, err := sealPiece([][]maths.Vec{sealRing(a, u, v, func(p maths.Vec) maths.Vec { return p }), sealRing(b, u.Scale(.6), v.Scale(.6), func(p maths.Vec) maths.Vec { return p })}, mat, at)
		if err != nil {
			return err
		}
		pieces = append(pieces, piece)
		return nil
	}
	for _, side := range []float64{-1, 1} {
		wing := hatzCageWingRig(fold, seconds, side)
		for i := 0; i < len(wing)-1; i++ {
			if err := bone(wing[i], wing[i+1], .10-.015*float64(i), hide); err != nil {
				return nil, err
			}
		}
		membrane, err := hatzCageMembrane(wing, side)
		if err != nil {
			return nil, err
		}
		pieces = append(pieces, thing.Thing{Shape: membrane, Material: cloth, Motion: thing.Still(at)})
		leg := hatzCageLegRig(fold, side)
		for i := 0; i < len(leg)-1; i++ {
			if err := bone(leg[i], leg[i+1], .12-.035*float64(i), hide); err != nil {
				return nil, err
			}
		}
		for digit := -1; digit <= 1; digit++ {
			a := leg[len(leg)-1].Add(maths.V(0, 0, .09*float64(digit)))
			b := a.Add(maths.V(.25, 0, .03*float64(digit)))
			if err := bone(a, b, .06, dark); err != nil {
				return nil, err
			}
		}
		if fold > .999 {
			pieces = append(pieces, thing.Thing{Shape: shape.Sphere{Radius: .12}, Material: hide, Motion: thing.Still(maths.NewSubspace(wing[2], maths.Identity, 1).Within(at))})
		}
		eye := hatzCageDeform(maths.V(1.88, 4.0, side*.23), seconds)
		pieces = append(pieces, thing.Thing{Shape: shape.Sphere{Radius: .045}, Material: dark, Motion: thing.Still(maths.NewSubspace(eye, maths.Identity, 1).Within(at))})
	}
	return pieces, nil
}

// A weighted head-joint map applied to the fixed indexed cage.
func hatzCageDeform(p maths.Vec, seconds float64) maths.Vec {
	t := math.Max(0, math.Min(1, (p.X-.32)/1.25))
	t = t * t * (3 - 2*t)
	angle := .10 * math.Sin(seconds*2*math.Pi/3)
	dx, dy := p.X-.32, p.Y-2.72
	rot := maths.V(.32+dx*math.Cos(angle)-dy*math.Sin(angle), 2.72+dx*math.Sin(angle)+dy*math.Cos(angle), p.Z)
	return p.Scale(1 - t).Add(rot.Scale(t))
}

type hatzCageEdge struct{ a, b int }

// The coarse cage is authored in space; one generic midpoint subdivision
// smooths each triangle while preserving connectivity and its rest vertices.
func hatzCageBody(seconds float64) (*mesh.Mesh, error) {
	type section struct{ x, y, up, side float64 }
	sections := []section{{-1.30, 2.54, .04, .04}, {-.92, 2.58, .29, .25}, {-.28, 2.68, .43, .34}, {.30, 2.73, .35, .29}, {.65, 2.98, .22, .19}, {1.05, 3.49, .17, .15}, {1.52, 3.89, .20, .18}, {1.85, 3.95, .29, .24}, {2.48, 3.82, .22, .18}, {3.47, 3.59, .07, .06}, {3.82, 3.48, .015, .015}}
	const n = 8
	var cage []maths.Vec
	var faces [][3]int
	for _, s := range sections {
		for j := 0; j < n; j++ {
			a := 2 * math.Pi * float64(j) / n
			cage = append(cage, maths.V(s.x, s.y+s.up*math.Cos(a), s.side*math.Sin(a)))
		}
	}
	for i := 0; i < len(sections)-1; i++ {
		for j := 0; j < n; j++ {
			a, b := i*n+j, i*n+(j+1)%n
			c, d := b+n, a+n
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	for _, row := range []int{0, len(sections) - 1} {
		s := sections[row]
		center := len(cage)
		cage = append(cage, maths.V(s.x, s.y, 0))
		for j := 0; j < n; j++ {
			k := (j + 1) % n
			if row == 0 {
				faces = append(faces, [3]int{center, k, j})
			} else {
				faces = append(faces, [3]int{center, row*n + j, row*n + k})
			}
		}
	}
	// Shared edge midpoints give a single connected subdivided surface.
	points := append([]maths.Vec(nil), cage...)
	cache := map[hatzCageEdge]int{}
	middle := func(a, b int) int {
		if a > b {
			a, b = b, a
		}
		key := hatzCageEdge{a, b}
		if i, ok := cache[key]; ok {
			return i
		}
		p := cage[a].Add(cage[b]).Scale(.5)
		i := len(points)
		points = append(points, p)
		cache[key] = i
		return i
	}
	var refined [][3]int
	for _, f := range faces {
		a, b, c := f[0], f[1], f[2]
		ab, bc, ca := middle(a, b), middle(b, c), middle(c, a)
		refined = append(refined, [3]int{a, ab, ca}, [3]int{b, bc, ab}, [3]int{c, ca, bc}, [3]int{ab, bc, ca})
	}
	posed := make([]maths.Vec, len(points))
	for i, p := range points {
		posed[i] = hatzCageDeform(p, seconds)
	}
	volume := 0.0
	for _, f := range refined {
		volume += posed[f[0]].Dot(posed[f[1]].Cross(posed[f[2]]))
	}
	if volume < 0 {
		for i := range refined {
			refined[i][1], refined[i][2] = refined[i][2], refined[i][1]
		}
	}
	return mesh.NewWithRestPoints(posed, points, refined, sealSmoothNormals(posed, refined), true)
}

// FABRIK projects an authored chain toward its endpoint by alternating
// backwards and forwards segment constraints; convergence is tested later.
func hatzSolveChain(root, target, bend maths.Vec, length []float64) []maths.Vec {
	p := make([]maths.Vec, len(length)+1)
	p[0] = root
	for i := 1; i < len(p); i++ {
		fraction := float64(i) / float64(len(length))
		p[i] = root.Scale(1 - fraction).Add(target.Scale(fraction)).Add(bend.Scale(math.Sin(math.Pi * fraction)))
	}
	for pass := 0; pass < 80; pass++ {
		p[len(p)-1] = target
		for i := len(p) - 2; i >= 0; i-- {
			delta := p[i].Sub(p[i+1])
			if delta.Len() < 1e-9 {
				delta = bend
			}
			p[i] = p[i+1].Add(delta.Unit().Scale(length[i]))
		}
		p[0] = root
		for i := 0; i < len(length); i++ {
			delta := p[i+1].Sub(p[i])
			if delta.Len() < 1e-9 {
				delta = bend
			}
			p[i+1] = p[i].Add(delta.Unit().Scale(length[i]))
		}
	}
	return p
}

// hatzCageWingRig is the points of one wing's arm, solved to reach a target
// between the open wing and the wing folded to the ground, by fold from
// nought to one.
func hatzCageWingRig(fold, seconds, side float64) []maths.Vec {
	root := maths.V(.16, 2.77, side*.34)
	openW := maths.V(.10, 2.75, side*2.64)
	groundW := maths.V(.55, .12, side*.90)
	target := openW.Scale(1 - fold).Add(groundW.Scale(fold))
	arm := hatzSolveChain(root, target, maths.V(.4, .2, side*.15), []float64{1.22, 1.63})
	openTip := arm[2].Add(maths.V(-.35, .12*math.Sin(seconds*2*math.Pi/3), side*2.18))
	foldTip := arm[2].Add(maths.V(-.60, 2.05, side*.20))
	tipGoal := openTip.Scale(1 - fold).Add(foldTip.Scale(fold))
	finger := hatzSolveChain(arm[2], tipGoal, maths.V(-.1, .25, side*.10), []float64{1.25, 1.0})
	return []maths.Vec{arm[0], arm[1], arm[2], finger[1], finger[2]}
}

// hatzCageLegRig is the points of one leg, solved to reach a target between
// the leg in flight and the leg on the ground, by fold from nought to one.
func hatzCageLegRig(fold, side float64) []maths.Vec {
	root := maths.V(-.75, 2.55, side*.30)
	ground := maths.V(-.62, .06, side*.45)
	flight := maths.V(-1.0, 1.55, side*.60)
	target := flight.Scale(1 - fold).Add(ground.Scale(fold))
	return hatzSolveChain(root, target, maths.V(-.45, .05, side*.1), []float64{1.18, 1.40})
}

// Each side is a bounded interpolated sheet. The fixed boundary is a
// geometric transfer from Webb, not tension, material fibres or aerodynamics.
func hatzCageMembrane(j []maths.Vec, side float64) (*mesh.Mesh, error) {
	const span, chord = 20, 10
	root := maths.V(-1.05, 2.45, side*.32)
	tip := j[len(j)-1]
	mix := func(a, b maths.Vec, t float64) maths.Vec { return a.Scale(1 - t).Add(b.Scale(t)) }
	var surface []maths.Vec
	for i := 0; i <= span; i++ {
		t := float64(i) / span
		u := t * float64(len(j)-1)
		k := min(len(j)-2, int(u))
		lead := mix(j[k], j[k+1], u-float64(k))
		trail := mix(root, tip, t).Add(maths.V(-.03*t-.08*math.Sin(math.Pi*t), 0, 0))
		for k := 0; k <= chord; k++ {
			v := float64(k) / chord
			surface = append(surface, mix(lead, trail, v))
		}
	}
	normal := j[1].Sub(j[0]).Cross(root.Sub(j[0]))
	if normal.Len() < 1e-9 {
		normal = maths.Up
	}
	normal = normal.Unit().Scale(.008)
	n := len(surface)
	var points []maths.Vec
	for _, p := range surface {
		points = append(points, p.Add(normal))
	}
	for _, p := range surface {
		points = append(points, p.Sub(normal))
	}
	idx := func(i, k int) int { return i*(chord+1) + k }
	var faces [][3]int
	for i := 0; i < span; i++ {
		for k := 0; k < chord; k++ {
			a, b, c, d := idx(i, k), idx(i+1, k), idx(i+1, k+1), idx(i, k+1)
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d}, [3]int{a + n, c + n, b + n}, [3]int{a + n, d + n, c + n})
		}
	}
	edge := func(a, b int) { faces = append(faces, [3]int{b, a, a + n}, [3]int{b, a + n, b + n}) }
	for k := 0; k < chord; k++ {
		edge(idx(0, k+1), idx(0, k))
		edge(idx(span, k), idx(span, k+1))
	}
	for i := 0; i < span; i++ {
		edge(idx(i, 0), idx(i+1, 0))
		edge(idx(i+1, chord), idx(i, chord))
	}
	v := 0.0
	for _, f := range faces {
		v += points[f[0]].Dot(points[f[1]].Cross(points[f[2]]))
	}
	if v < 0 {
		for i := range faces {
			faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
		}
	}
	return mesh.New(points, faces, sealSmoothNormals(points, faces), true)
}
