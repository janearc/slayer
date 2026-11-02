package trace

import (
	"math"
	"testing"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// stateAt is a march set at a place in the hole's maths, looking a way as
// an eye held still there sees it, with the shift of an eye held still
// there too.
func stateAt(sc staged, f field, x, dir [3]float64) *marcher {
	at := toHole(x)
	still := f.hole.Static(at)
	m := &marcher{sc: sc, f: f, keep: maths.Grey(1)}
	m.ray = f.hole.Arriving(at, f.hole.FrameOf(at, still), toHole(dir))
	m.shift = shifter{f: f, p0: m.ray.Momentum(), obs: still}
	m.from = f.at.Out(f.f.FromHole(x))
	return m
}

// emptyState is a march with nothing but the hole in the world.
func emptyState(mass float64, x, dir [3]float64) *marcher {
	w := thing.NewWorld()
	w.Sky = compass{}
	w.AddField(thing.Field{Mass: mass, Reach: 40 * mass})
	sc := stage(w, 0)
	return stateAt(sc, sc.fs[0], x, dir)
}

// a march is let go where it is going out, past 12 masses, and not
// before or at the 12th; for a hole of mass 2, past 24; and a ray that is
// going in is never let go for that. worked at 20 out and 10 out, at
// exactly 12, and for a hole of mass 2, which has its 12 masses at 24.
//
// that it is going out is its coordinate velocity along its place. the
// positions with parts of both signs and different sizes are to tell the
// sum of the three products from any three of them, and the ways go
// straight out and straight in.
func TestAMarchIsLetGoPastTwelveMassesGoingOut(t *testing.T) {
	for _, c := range []struct {
		name string
		mass float64
		x    [3]float64
		out  bool
		done bool
	}{
		{"going out at 20 M", 1, [3]float64{20, 0, 0}, true, true},
		{"going out at 10 M", 1, [3]float64{10, 0, 0}, true, false},
		{"going out at exactly 12 M", 1, [3]float64{12, 0, 0}, true, false},
		{"going in at 20 M", 1, [3]float64{20, 0, 0}, false, false},
		{"going out at 20 M, 2 masses", 2, [3]float64{20, 0, 0}, true, false},
		{"going out at 30 M, 2 masses", 2, [3]float64{30, 0, 0}, true, true},
		{"going out, both signs", 1, [3]float64{15, -10, 8}, true, true},
		{"going in, both signs", 1, [3]float64{15, -10, 8}, false, false},
		{"going out, other signs", 1, [3]float64{-9, 14, 7}, true, true},
		{"going in, other signs", 1, [3]float64{-9, 14, 7}, false, false},
		{"going out, 3 and 5 and 17", 1, [3]float64{3, -5, 17}, true, true},
		{"going in, 3 and 5 and 17", 1, [3]float64{3, -5, 17}, false, false},
	} {
		r := math.Sqrt(c.x[0]*c.x[0] + c.x[1]*c.x[1] + c.x[2]*c.x[2])
		way := [3]float64{c.x[0] / r, c.x[1] / r, c.x[2] / r}
		if !c.out {
			way = [3]float64{-way[0], -way[1], -way[2]}
		}
		m := emptyState(c.mass, c.x, way)
		if _, done := m.march(0); done != c.done {
			t.Errorf("%s: let go %v, want %v", c.name, done, c.done)
		}
	}
}

// a ray beyond the reach is let go though it is going in, but not on the
// reach, and not on the first step: a ray 45 out and going in is let go
// by the second step and not by the first; one exactly 40 out going in is
// not let go by either.
func TestAMarchIsLetGoPastTheReachOnItsSecondStep(t *testing.T) {
	for _, c := range []struct {
		name string
		x    [3]float64
		step int
		done bool
	}{
		{"45 out, first step", [3]float64{45, 0, 0}, 0, false},
		{"45 out, second step", [3]float64{45, 0, 0}, 1, true},
		{"40 out, second step", [3]float64{40, 0, 0}, 1, false},
		{"39 out, second step", [3]float64{39, 0, 0}, 1, false},
	} {
		m := emptyState(1, c.x, [3]float64{-1, 0, 0})
		if _, done := m.march(c.step); done != c.done {
			t.Errorf("%s: let go %v, want %v", c.name, done, c.done)
		}
	}
}

// black is what comes from the hole's edge and from its middle. a ray
// that hovers, come from the edge, is black at any radius; one deep
// inside, nearer than a fifth of a mass, is black whatever else; one
// between that and the horizon is marched on.
//
// worked for a hole of mass 1, and for mass 2 where a fifth of a mass is
// 0.4: so a ray at 0.3 is black for it and not for mass 1.
func TestARayFromTheEdgeOrTheMiddleIsBlack(t *testing.T) {
	for _, c := range []struct {
		name  string
		mass  float64
		x     [3]float64
		p     [4]float64
		black bool
	}{
		{"deep inside", 1, [3]float64{0.1, 0, 0}, [4]float64{1, 0, 0, 0}, true},
		{"just inside the fifth", 1, [3]float64{0.19, 0, 0}, [4]float64{1, 0, 0, 0}, true},
		{"exactly a fifth", 1, [3]float64{0.2, 0, 0}, [4]float64{1, 0, 0, 0}, false},
		{"a little further in", 1, [3]float64{0.3, 0, 0}, [4]float64{1, 0, 0, 0}, false},
		{"a mass of 2", 2, [3]float64{0.3, 0, 0}, [4]float64{1, 0, 0, 0}, true},
		{"a mass of 2, further", 2, [3]float64{0.5, 0, 0}, [4]float64{1, 0, 0, 0}, false},
		{"hovering", 1, [3]float64{10, 0, 0}, [4]float64{1, -5000, 0, 0}, true},
		{"hovering, a mass of 2", 2, [3]float64{10, 0, 0}, [4]float64{1, -5000, 0, 0}, true},
		{"not hovering, far in", 1, [3]float64{10, 0, 0}, [4]float64{1, -1, 0, 0}, false},
	} {
		m := emptyState(c.mass, c.x, [3]float64{0, 1, 0})
		m.ray = kerr.NewRay(kerr.NewVec4(0, c.x[0], c.x[1], c.x[2]), vec4(c.p))
		if hover := m.f.hole.Hovering(m.ray); hover != (c.name == "hovering" || c.name == "hovering, a mass of 2") {
			t.Fatalf("%s: the ray hovers: %v", c.name, hover)
		}
		px, done := m.march(1)
		if done != c.black {
			t.Errorf("%s: done %v, want black %v", c.name, done, c.black)
		}
		if done && (px.Light() != (maths.Light{}) || px.Cover != 1) {
			t.Errorf("%s: came back %+v, want the hole's black", c.name, px)
		}
	}
}

// ahead is where the segment of a march's next step runs from, which way
// it runs, and how long it is, the step taken as the march takes it.
func ahead(m *marcher) (from maths.Vec, along maths.Vec, length float64) {
	next := m.f.hole.Step(m.ray, m.f.hole.PaceIn(m.ray))
	to := m.f.at.Out(m.f.f.FromHole(parts3(next.Space())))
	seg := to.Sub(m.from)
	length = seg.Len()
	return m.from, seg.Scale(1 / length), length
}

// a ray stops at a surface its step crosses, with the surface's glow, and
// at no surface its step does not reach. a glowing ball with its near
// face half a step ahead, or 0.3 ahead, which is under a unit, is the
// pixel, covering it, at the depth the march has come.
//
// a ball a step and a half ahead is not reached by this step, which goes
// on, and has come that far. a ball between a step's length and its
// square is reached if the step is the longer of the two, and not if
// not, which holds that the direction of the segment is a unit one.
//
// worked for a march at 30 M, going round.
func TestAMarchStopsAtASurfaceItsStepCrosses(t *testing.T) {
	x, way := [3]float64{30, 0, 0}, [3]float64{0, 1, 0}
	probe := emptyState(1, x, way)
	from, dir, length := ahead(probe)
	if length < 0.5 || length > 1.5 {
		t.Fatalf("the step is %v long; the cases here want about a unit", length)
	}
	between := (length + length*length) / 2
	for _, c := range []struct {
		name string
		near float64 // how far ahead the ball's near face is
		hit  bool
	}{
		{"half a step ahead", length / 2, true},
		{"3 tenths ahead", 0.3, true},
		{"a step and a half ahead", 1.5 * length, false},
		{"between the step and its square", between, between <= length},
		{"a hair short of the step", length * (1 - 1e-6), true},
	} {
		w := thing.NewWorld()
		w.Sky = compass{}
		w.AddField(thing.Field{Mass: 1, Reach: 40})
		w.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.05},
			Material: &thing.Material{Glow: maths.Grey(0.7)},
			Motion:   at(from.Add(dir.Scale(c.near+0.05)), 1)})
		sc := stage(w, 0)
		m := stateAt(sc, sc.fs[0], x, way)
		px, done := m.march(1)
		if done != c.hit {
			t.Errorf("%s: stopped %v, want %v", c.name, done, c.hit)
			continue
		}
		if done {
			colours(t, c.name, px, maths.Grey(0.7), 1e-6)
			if px.Cover != 1 || px.Depth != 0 {
				t.Errorf("%s: covers %v at %v, want 1 at 0", c.name, px.Cover, px.Depth)
			}
		} else if want := float32(length); m.depth != want {
			t.Errorf("%s: the march has come %v, want %v", c.name, m.depth, want)
		}
	}
}

// reaches is a shape that any ray meets at one distance, whatever it is
// asked, so that a test can put a surface exactly at the end of a step.
type reaches struct{ at float64 }

func (reaches) Bounds() maths.Box                                 { return maths.Everywhere() }
func (reaches) Distance(p maths.Vec) float64                      { return p.Y }
func (r reaches) Hit(maths.Ray, float64, float64) (float64, bool) { return r.at, true }

// a surface exactly at the end of a step is on the step: the march stops
// at it with its glow.
func TestAMarchStopsAtASurfaceAtTheEndOfItsStep(t *testing.T) {
	x, way := [3]float64{30, 0, 0}, [3]float64{0, 1, 0}
	_, _, length := ahead(emptyState(1, x, way))
	w := thing.NewWorld()
	w.Sky = compass{}
	w.AddField(thing.Field{Mass: 1, Reach: 40})
	w.Add(thing.Thing{Shape: reaches{length}, Material: &thing.Material{Glow: maths.Grey(0.7)}})
	sc := stage(w, 0)
	m := stateAt(sc, sc.fs[0], x, way)
	px, done := m.march(1)
	if !done {
		t.Fatal("a surface exactly at the end of the step was not met")
	}
	colours(t, "a surface at the end of the step", px, maths.Grey(0.7), 1e-6)
}

// a ray that goes out by the smallest margin, with its place and way
// making a dot of under one, is out all the same: it is let go past 12
// masses, and not held for being slow. the way is a thousandth outward of
// a way that runs round.
func TestAMarchGoingOutBarelyIsLetGo(t *testing.T) {
	x := [3]float64{20, 0, 0}
	m := emptyState(1, x, [3]float64{0.001, math.Sqrt(1 - 1e-6), 0})
	v := parts4(m.f.hole.Velocity(m.ray))
	if dot := x[0]*v[1] + x[1]*v[2] + x[2]*v[3]; !(dot > 0 && dot <= 1) {
		t.Fatalf("the ray's place and way make %v, not something over nought and at most one", dot)
	}
	if _, done := m.march(0); !done {
		t.Error("a ray going out by a hair was held")
	}
}

// slice is a medium of thickness k through the part of a cube that lies
// below a plane across a direction, and clear beyond it.
type slice struct {
	block
	across maths.Vec
	below  float64
}

func (s slice) Density(p maths.Vec) float64 {
	if p.Dot(s.across) < s.below {
		return s.block.Density(p)
	}
	return 0
}

// a short step of the march is cut into four pieces, and gathers the
// light of what lies in it, shifted. a glowing slice along the first two
// fifths of the step, 0.4 of its length, has the middles of the first
// two quarters inside it, 0.125 and 0.375, and not the third, 0.625.
//
// so it is two quarters of the step thick, tau being k times a half of
// the step's length. a uniform block that holds the whole step is its
// whole tau.
//
// the glow is shifted by the gain at the middle of the step, the square
// of the ratio of a still eye's root of one less 2 M over r to the
// emitter's, its r the middle's. it is a ray that goes round at 30 M,
// which is not let go.
func TestAStepGathersTheGlowOfWhatLiesInIt(t *testing.T) {
	x, way := [3]float64{30, 0, 0}, [3]float64{0, 1, 0}
	probe := emptyState(1, x, way)
	from, dir, length := ahead(probe)
	mid := from.Add(dir.Scale(length / 2))
	hx := probe.f.f.InHole(mid)
	fMid := 1 - 2/math.Sqrt(hx[0]*hx[0]+hx[1]*hx[1]+hx[2]*hx[2])
	gain := math.Pow(fMid/(1-2.0/30), 2)

	const k = 0.7
	for _, c := range []struct {
		name string
		m    thing.Volume
		tau  float64
	}{
		{"a slice along 2/5 of the step", thing.Volume{Medium: slice{block{k: k, half: 100}, dir,
			from.Dot(dir) + 0.4*length}}, k * length / 2},
		{"a block that holds the step", thing.Volume{Medium: block{k: k, half: 100},
			Motion: at(from, 1)}, k * length},
	} {
		w := thing.NewWorld()
		w.Sky = compass{}
		w.AddField(thing.Field{Mass: 1, Reach: 40})
		v := c.m
		v.Material = glowOf(maths.RGB(1, 0.5, 0.25))
		w.AddVolume(v)
		sc := stage(w, 0)
		m := stateAt(sc, sc.fs[0], x, way)
		if _, done := m.march(1); done {
			t.Fatalf("%s: let go", c.name)
		}
		seen := 1 - math.Exp(-c.tau)
		want := maths.RGB(1, 0.5, 0.25).Scale(seen * gain)
		colours(t, c.name, fixed(m.inside), want, 1e-9)
		if !near(m.keep.R, math.Exp(-c.tau), 1e-12) {
			t.Errorf("%s: keeps %v, want %v", c.name, m.keep.R, math.Exp(-c.tau))
		}
		if m.depth != float32(length) {
			t.Errorf("%s: has come %v, want %v", c.name, m.depth, float32(length))
		}
	}
}
