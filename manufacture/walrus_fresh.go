package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// WalrusFresh1 is the first original construction experiment. Appendages
// are separate closed pieces; the body is a fresh swept skin with caps.
func WalrusFresh1(yaw, phase float64) ([]thing.Thing, error) {
	const n, m = 36, 32
	vertices := make([]maths.Vec, 0, (n+1)*m+2)
	faces := make([][3]int, 0, n*m*2+m*2)
	warp := func(p maths.Vec) maths.Vec {
		w := (1.5 - p.X) / 3
		p.Z += yaw * w * w * .6
		return p
	}
	for i := 0; i <= n; i++ {
		x := -1.5 + 3*float64(i)/n
		t := float64(i) / n
		radius := .48 * math.Pow(math.Max(.002, math.Sin(math.Pi*t)), .47)
		if t > .74 {
			radius *= 1 - .5*(t-.74)/.26
		}
		for j := 0; j < m; j++ {
			a := 2 * math.Pi * float64(j) / m
			vertices = append(vertices, warp(maths.V(x, radius*.92*math.Cos(a), radius*math.Sin(a))))
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			a := i*m + j
			b := i*m + (j+1)%m
			c := (i+1)*m + (j+1)%m
			d := (i+1)*m + j
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	tail := len(vertices)
	vertices = append(vertices, warp(maths.V(-1.5, 0, 0)))
	nose := len(vertices)
	vertices = append(vertices, warp(maths.V(1.5, 0, 0)))
	for j := 0; j < m; j++ {
		k := (j + 1) % m
		faces = append(faces, [3]int{tail, k, j}, [3]int{nose, n*m + j, n*m + k})
	}
	skin, err := shape.NewMesh(vertices, faces, nil, false)
	if err != nil {
		return nil, err
	}
	coat := &thing.Material{Name: "walrus study skin", Colour: maths.RGB(.30, .185, .14), Gloss: .13, Satin: .59}
	ivory := &thing.Material{Name: "tusk", Colour: maths.RGB(.77, .72, .58), Gloss: .15, Satin: .42}
	eye := &thing.Material{Name: "eye", Colour: maths.RGB(.018, .013, .011), Gloss: .4}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))}}
	for _, side := range []float64{-1, 1} {
		at := warp(maths.V(1.08, .16, side*.19))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .036}, Material: eye, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1))})
		pad, e := walrusStudyEllipsoid(warp(maths.V(1.34, -.035, side*.10)), maths.V(.17, .12, .13), 0)
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: pad, Material: coat, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))})
		for row := 0; row < 3; row++ {
			for column := 0; column < 8; column++ {
				root := warp(maths.V(1.46, -.02-float64(row)*.035, side*(.06+float64(column)*.013)))
				end := root.Add(maths.V(.12, .025-float64(row)*.025, side*(.08+float64(column)*.01)))
				axis := end.Sub(root)
				parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .0015, Thick: .003, Height: axis.Len()}, Material: ivory, Motion: thing.Still(maths.NewSubspace(root.Add(end).Scale(.5), maths.Upright(axis), 1))})
			}
		}
		// Straight rigid temporary tusks: their shape and roots need inspection.
		root := warp(maths.V(1.42, -.11, side*.12))
		length := .34
		parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .025, Thick: .05, Height: length}, Material: ivory, Motion: thing.Still(maths.NewSubspace(root.Add(maths.V(0, -length/2, 0)), maths.Identity, 1))})
		// These independent pieces move with the axial control, but are not yet
		// stitched to the trunk. Hind flippers drive the test stroke.
		for _, hind := range []bool{false, true} {
			x, z, ry, rz, roll := .48, side*.42, .065, .30, side*.12
			if hind {
				x, z, ry, rz, roll = -1.17, side*.24, .055, .27, side*(.24+.35*math.Sin(phase))
			}
			fin, e := walrusStudyEllipsoid(warp(maths.V(x, -.22, z)), maths.V(.26, ry, rz), roll)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: fin, Material: coat, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))})
		}
	}
	return parts, nil
}

// walrusStudyEllipsoid produces a closed piece from an explicit surface,
// with every triangle and both caps generated in this study.
func walrusStudyEllipsoid(center maths.Vec, radii maths.Vec, roll float64) (*shape.Mesh, error) {
	const n, m = 16, 24
	vertices := make([]maths.Vec, 0, (n+1)*m+2)
	faces := make([][3]int, 0, n*m*2+2*m)
	point := func(x, y, z float64) maths.Vec {
		c, s := math.Cos(roll), math.Sin(roll)
		return center.Add(maths.V(x, y*c-z*s, y*s+z*c))
	}
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		x := radii.X * (2*t - 1)
		r := math.Sqrt(math.Max(.0001, 1-(2*t-1)*(2*t-1)))
		for j := 0; j < m; j++ {
			a := 2 * math.Pi * float64(j) / m
			vertices = append(vertices, point(x, r*radii.Y*math.Cos(a), r*radii.Z*math.Sin(a)))
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			a := i*m + j
			b := i*m + (j+1)%m
			c := (i+1)*m + (j+1)%m
			d := (i+1)*m + j
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	first := len(vertices)
	vertices = append(vertices, point(-radii.X, 0, 0))
	last := len(vertices)
	vertices = append(vertices, point(radii.X, 0, 0))
	for j := 0; j < m; j++ {
		k := (j + 1) % m
		faces = append(faces, [3]int{first, k, j}, [3]int{last, n*m + j, n*m + k})
	}
	return shape.NewMesh(vertices, faces, nil, false)
}
