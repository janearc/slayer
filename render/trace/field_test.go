package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// compass is a sky whose light is the way a ray looks: what comes back
// from it says where a ray went.
type compass struct{}

// Light is the direction looked toward, as light.
func (compass) Light(toward maths.Vec) maths.Light {
	return maths.Light{R: toward.X, G: toward.Y, B: toward.Z}
}

// hole is an empty world with a hole of unit mass at its middle,
// reaching forty masses, under a compass sky.
func hole(motion thing.Motion) *thing.World {
	w := thing.NewWorld()
	w.Lamps = nil
	w.Sky = compass{}
	w.AddField(thing.Field{Mass: 1, Reach: 40, Motion: motion})
	return w
}

// A ray enters the nearest field it will cross, and how far along: at zero when
// it starts inside, never when it misses or has passed, and a field on a motion
// of scale two reaches twice as far.
func TestEntering(t *testing.T) {
	fs := placeFields(hole(nil), 0)
	at := func(from, dir maths.Vec) (float64, bool) {
		_, d, ok := entering(fs, maths.Ray{At: from, Dir: dir.Unit()})
		return d, ok
	}
	ahead := maths.V(0, 0, 1)
	if d, ok := at(maths.V(0, 0, -100), ahead); !ok || math.Abs(d-60) > 1e-9 {
		t.Errorf("from a hundred out it enters at %v, %v", d, ok)
	}
	if d, ok := at(maths.V(0, 0, -10), ahead); !ok || d != 0 {
		t.Errorf("from inside it enters at %v, %v", d, ok)
	}
	if _, ok := at(maths.V(0, 50, -100), ahead); ok {
		t.Error("a ray passing fifty out entered")
	}
	if _, ok := at(maths.V(0, 0, 100), ahead); ok {
		t.Error("a ray already past the field entered")
	}
	twice := placeFields(hole(thing.Still(maths.Subspace{Scale: 2})), 0)
	_, d, ok := entering(twice, maths.Ray{At: maths.V(0, 0, -200), Dir: ahead})
	if !ok || math.Abs(d-120) > 1e-9 {
		t.Errorf("at scale two it enters at %v, %v, not 120", d, ok)
	}
}

// A ray through the edge of the reach comes out bent toward the hole;
// one aimed at the hole is black and covers its pixel. the degree this
// holds is the cutoff it began with, not the truth: the exact bend at an impact of
// 39.9 masses is 6.2 degrees, and this tracer bends it 0.24 (see loose).
func TestARayLeavesOrFalls(t *testing.T) {
	w := hole(nil)
	edge := Trace(w, maths.Ray{At: maths.V(0, 39.9, -100),
		Dir: maths.V(0, 0, 1)}, 0)
	out := maths.V(float64(edge.R), float64(edge.G), float64(edge.B))
	if out.Len() < 0.5 {
		t.Fatalf("the ray past the edge came back %v", edge)
	}
	turned := math.Acos(math.Min(1, out.Unit().Z)) / maths.Degree
	if turned > 1 || out.Y >= 0 {
		t.Errorf("past the edge the ray turned %.3f degrees, toward %v",
			turned, out.Unit())
	}
	in := Trace(w, maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}, 0)
	if in.Light() != (maths.Light{}) || in.Cover != 1 {
		t.Errorf("the ray into the hole came back %v", in)
	}
}

// The march's first step starts on the reach, and does not leave by it
// even when it starts a hair outside, as rounding may put it; a later
// step that far out does.
func TestTheFirstStepNeverLeaves(t *testing.T) {
	w := hole(nil)
	sc := stage(w, 0)
	f := sc.fs[0]
	r := maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}
	fresh := func() *marcher {
		m := &marcher{sc: sc, f: f, keep: maths.Grey(1)}
		m.start(r, 60-1e-6)
		return m
	}
	m := fresh()
	if rad := f.hole.Radius(m.ray.Space()); !(rad > f.f.Reach) {
		t.Fatalf("the march starts at %v, not outside the reach", rad)
	}
	if _, done := m.march(0); done {
		t.Error("the first step left the field")
	}
	if _, done := fresh().march(1); !done {
		t.Error("a later step outside the reach stayed in")
	}
}
