package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// Anti-test: at a coarse world, where tiles are far apart and the
// icosahedron's faces show, the smoothed height and wet have no step at
// a tile's edge or at a vertex where three meet.
func TestSmoothedIsSteadyInACoarseWorld(t *testing.T) {
	w := New(Terrain{Tiling: hexes.Sphere{Splits: 5}, Seed: 7, Ocean: 0.71,
		Plates: 12, Axis: maths.V(0, 1, 0)})
	hair := 1e-6
	worst := 0.0
	for index := range w.tiles {
		for _, other := range w.near[index] {
			a, b := w.middle[index], w.middle[other]
			edge := a.Add(b).Unit()
			along := b.Sub(a).Unit().Scale(hair)
			s, u := w.Smoothed(edge.Sub(along)), w.Smoothed(edge.Add(along))
			worst = math.Max(worst, math.Abs(s.Height-u.Height))
			worst = math.Max(worst, math.Abs(s.Wet-u.Wet))
			for _, third := range w.near[other] {
				sum := a.Add(b).Add(w.middle[third])
				c := sum.Unit()
				d := maths.V(hair, 0, 0)
				s, u = w.Smoothed(c.Add(d)), w.Smoothed(c.Sub(d))
				worst = math.Max(worst, math.Abs(s.Wet-u.Wet))
			}
		}
	}
	if worst > 1e-3 {
		t.Errorf("a step of %v at a tile's edge or vertex", worst)
	}
}
