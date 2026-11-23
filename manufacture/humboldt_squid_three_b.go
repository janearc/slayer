package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"sync"
)

// A sculpted occupancy grid in units of two per metre. Each brush is an
// independently placed ellipsoidal stamp; its distribution is illustrative.
var squidThreeBRest = sync.OnceValues(func() (*shape.Mesh, error) {
	const nx, ny, nz = 104, 42, 62
	count := (nx + 1) * (ny + 1) * (nz + 1)
	grid := make([]float64, count)
	idx := func(i, j, k int) int { return (i*(ny+1)+j)*(nz+1) + k }
	type dab struct{ x, y, z, rx, ry, rz float64 }
	dabs := []dab{}
	// Mantle volume grows from the posterior point toward the opening.
	for i := 0; i < 32; i++ {
		u := float64(i) / 31
		x := -1.56 + 1.54*u
		r := .035 + .245*math.Pow(math.Sin(math.Pi*u/2), .8)
		dabs = append(dabs, dab{x, 0, 0, .095, r, r})
	}
	dabs = append(dabs, dab{.13, 0, 0, .28, .20, .22})
	// Brush strokes trace tapered curved arms; one pair extends into clubs.
	for k := 0; k < 10; k++ {
		a := math.Pi * float64(k%8) / 4
		if k >= 8 {
			a = math.Pi*.25 + math.Pi*float64(k-8)
		}
		l := .90
		if k >= 8 {
			l = 1.28
		}
		for j := 0; j < 22; j++ {
			u := float64(j) / 21
			theta := a + .12*math.Sin(math.Pi*u)
			r := .09 + .19*u
			thick := .072 * (1 - .60*u)
			if k >= 8 && u > .83 {
				thick += .035 * math.Sin(math.Pi*(u-.83)/.17)
			}
			dabs = append(dabs, dab{.22 + l*u, r * math.Sin(theta), r * math.Cos(theta), .065, thick, thick})
		}
	}
	for i := 0; i <= nx; i++ {
		x := -1.9 + 3.8*float64(i)/nx
		for j := 0; j <= ny; j++ {
			y := -.75 + 1.5*float64(j)/ny
			for k := 0; k <= nz; k++ {
				z := -1 + 2*float64(k)/nz
				v := 0.
				for _, d := range dabs {
					if math.Abs(x-d.x) > d.rx || math.Abs(y-d.y) > d.ry || math.Abs(z-d.z) > d.rz {
						continue
					}
					if math.Pow((x-d.x)/d.rx, 2)+math.Pow((y-d.y)/d.ry, 2)+math.Pow((z-d.z)/d.rz, 2) < 1 {
						v = 1
						break
					}
				}
				// Wedge is stamped directly into the same volume as the mantle.
				finWidth := .46 * math.Max(0, 1-math.Abs((x+1.21)/.36))
				if math.Abs(z) < finWidth && math.Abs(y) < .027 {
					v = 1
				}
				grid[idx(i, j, k)] = v
			}
		}
	}
	// Two bounded six-neighbour relaxations smooth voxel corners.
	for step := 0; step < 2; step++ {
		next := make([]float64, count)
		for i := 1; i < nx; i++ {
			for j := 1; j < ny; j++ {
				for k := 1; k < nz; k++ {
					q := idx(i, j, k)
					next[q] = (2*grid[q] + grid[idx(i-1, j, k)] + grid[idx(i+1, j, k)] + grid[idx(i, j-1, k)] + grid[idx(i, j+1, k)] + grid[idx(i, j, k-1)] + grid[idx(i, j, k+1)]) / 8
				}
			}
		}
		grid = next
	}
	field := func(p maths.Vec) float64 {
		a, b, c := (p.X+1.9)*nx/3.8, (p.Y+.75)*ny/1.5, (p.Z+1)*nz/2
		if a < 0 || a >= nx || b < 0 || b >= ny || c < 0 || c >= nz {
			return .22
		}
		i, j, k := int(a), int(b), int(c)
		u, v, w := a-float64(i), b-float64(j), c-float64(k)
		sum := 0.
		for di := 0; di < 2; di++ {
			for dj := 0; dj < 2; dj++ {
				for dk := 0; dk < 2; dk++ {
					wx, wy, wz := 1-u, 1-v, 1-w
					if di == 1 {
						wx = u
					}
					if dj == 1 {
						wy = v
					}
					if dk == 1 {
						wz = w
					}
					sum += grid[idx(i+di, j+dj, k+dk)] * wx * wy * wz
				}
			}
		}
		return .22 - sum
	}
	return bioformFieldMeshFine(field, func(p maths.Vec) maths.Vec { return p })
})

type squidThreeBPaint struct{}

// At is the colour at a point: a dark eye at the side of the head, and a
// red body elsewhere.
func (squidThreeBPaint) At(p maths.Vec) maths.Light {
	if math.Pow((p.X-.065)/.025, 2)+math.Pow((p.Y-.006)/.025, 2) < 1 && math.Abs(p.Z) > .07 {
		return maths.RGB(.012, .016, .02)
	}
	return maths.RGB(.33, .10, .10)
}

// HumboldtSquidThreeB applies an independently specified spatial displacement
// field to the indexed sculpture; it does not model hydrodynamics.
func HumboldtSquidThreeB(phase float64) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("squid 03b: invalid phase")
	}
	raw, err := squidThreeBRest()
	if err != nil {
		return nil, err
	}
	rest, posed := make([]maths.Vec, len(raw.Vertices)), make([]maths.Vec, len(raw.Vertices))
	// Independently specified lattice nodes. These controls are a visual
	// deformation experiment, not a model of mantle or fin tissue mechanics.
	type control struct {
		at, size, delta maths.Vec
		lag             float64
	}
	controls := []control{
		{maths.V(-.65, .19, 0), maths.V(.70, .24, .42), maths.V(0, .045, 0), 0},
		{maths.V(-.65, -.19, 0), maths.V(.70, .24, .42), maths.V(0, -.045, 0), 0},
		{maths.V(-.65, 0, .20), maths.V(.70, .40, .22), maths.V(0, 0, .045), 0},
		{maths.V(-.65, 0, -.20), maths.V(.70, .40, .22), maths.V(0, 0, -.045), 0},
		{maths.V(-1.23, 0, .39), maths.V(.42, .30, .27), maths.V(0, .19, 0), .4},
		{maths.V(-1.23, 0, -.39), maths.V(.42, .30, .27), maths.V(0, .19, 0), .4},
		{maths.V(.92, .18, .18), maths.V(.80, .24, .24), maths.V(0, .18, .06), .3},
		{maths.V(.92, -.18, -.18), maths.V(.80, .24, .24), maths.V(0, -.18, -.06), .9},
		{maths.V(.92, .18, -.18), maths.V(.80, .24, .24), maths.V(0, .13, -.08), 1.5},
		{maths.V(.92, -.18, .18), maths.V(.80, .24, .24), maths.V(0, -.13, .08), 2.1},
	}
	weight := func(x float64) float64 { t := math.Max(0, 1-math.Abs(x)); return t * t * (3 - 2*t) }
	for i, p := range raw.Vertices {
		rest[i] = p.Scale(.5)
		q := p
		for _, c := range controls {
			w := weight((p.X-c.at.X)/c.size.X) * weight((p.Y-c.at.Y)/c.size.Y) * weight((p.Z-c.at.Z)/c.size.Z)
			q = q.Add(c.delta.Scale(.35 * w * math.Sin(phase+c.lag)))
		}
		posed[i] = q.Scale(.5)
	}
	skin, err := mesh.NewWithRestPoints(posed, rest, raw.Faces, sealSmoothNormals(posed, raw.Faces), true)
	if err != nil {
		return nil, err
	}
	material := &thing.Material{Name: "squid voxel sculpture", Texture: squidThreeBPaint{}, Colour: maths.RGB(.33, .1, .1), Satin: .48, Gloss: .1}
	return []thing.Thing{{Shape: skin, Material: material, Motion: thing.Still(maths.Local())}}, nil
}
