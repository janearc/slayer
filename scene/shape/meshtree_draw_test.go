//go:build draw

package shape

import (
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The tree finds what asking every face finds: the same nearest face at
// the same distance, to the last bit, on a soup of ten thousand faces,
// and the same nearest point to anywhere.
func TestTheTreeFindsWhatEveryFaceFinds(t *testing.T) {
	rng := rand.New(rand.NewPCG(10, 11))
	vs, fs := soup(rng, 10000)
	m := made(t, vs, fs, nil, false)
	hits := 0
	for i := range 10000 {
		r := aimed(anyway(rng).Scale(3), within(rng, 1.2))
		if i%3 == 2 {
			r = maths.Ray{At: within(rng, 1), Dir: anyway(rng)}
		}
		near, far := 0.0, forever
		if i%7 == 6 {
			near, far = rng.Float64(), 1+2*rng.Float64()
		}
		got, at, ok := m.first(r, near, far)
		want, wat, wok := every(m, r, near, far)
		if ok != wok || got != want || at != wat {
			t.Fatalf("%v: the tree met face %d at %v, %v; every "+
				"face, %d at %v, %v", r, at, got, ok, wat, want,
				wok)
		}
		if ok {
			hits++
		}
	}
	t.Logf("%d of 10000 rays met the soup", hits)
	for range 2000 {
		p := within(rng, 1.5)
		_, _, got := m.nearest(p)
		if want := nearestOfAll(m, p); got != want {
			t.Fatalf("nearest %v: the tree %v, every face %v",
				p, got, want)
		}
	}
}
