package trace

import (
	"math"
	"testing"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// placedField is a field as the tracer places it.
func placedField(f thing.Field) field {
	w := thing.NewWorld()
	w.AddField(f)
	return placeFields(w, 0)[0]
}

// same4 says whether two four-vectors agree to tol, part by part. a part
// that is no number agrees with nothing.
func same4(a, b kerr.Vec4, tol float64) bool {
	ours, theirs := parts4(a), parts4(b)
	for i := range ours {
		if !near(ours[i], theirs[i], tol) {
			return false
		}
	}
	return true
}

// nan4 says whether the time and the turning parts of a four-vector are no
// number; the last is zero for a still or a circling emitter.
func nan4(a kerr.Vec4) bool {
	return math.IsNaN(a.Time()) && math.IsNaN(a.AlongX()) &&
		math.IsNaN(a.AlongY())
}

// a field with no reach or no mass is no field, and one of mass and reach
// is, whatever the size of either: the reach of 1 is placed.
func TestOnlyAFieldWithMassAndReachIsPlaced(t *testing.T) {
	for _, c := range []struct {
		name   string
		f      thing.Field
		placed bool
	}{
		{"a hole", thing.Field{Mass: 1, Reach: 40}, true},
		{"a hole that reaches 1", thing.Field{Mass: 1, Reach: 1}, true},
		{"a hole of a half", thing.Field{Mass: 0.5, Reach: 40}, true},
		{"a hole of no reach", thing.Field{Mass: 1, Reach: 0}, false},
		{"a hole of less than no reach", thing.Field{Mass: 1, Reach: -1}, false},
		{"a hole of no mass", thing.Field{Mass: 0, Reach: 40}, false},
		{"a hole of less than no mass", thing.Field{Mass: -1, Reach: 40}, false},
	} {
		w := thing.NewWorld()
		w.AddField(c.f)
		if got := len(placeFields(w, 0)) == 1; got != c.placed {
			t.Errorf("%s: placed %v, want %v", c.name, got, c.placed)
		}
	}
}

// a ray enters the reach where it first crosses it, and the edge is as
// follows. on the reach itself, heading in, it is not entered: it is
// where it would enter, and the zero distance is not taken. a hair
// outside it enters at the hair, and half a unit outside at half a unit.
//
// a ray that only touches the reach, and one that clips it by a sliver,
// enter it. a field of no reach, a point, 1e-9 ahead is not entered. of
// two fields entered together, the first is the one entered.
func TestEnteringKnowsTheEdgeOfTheReach(t *testing.T) {
	fs := placeFields(hole(nil), 0)
	ahead := maths.V(0, 0, 1)
	for _, c := range []struct {
		name  string
		from  maths.Vec
		enter float64 // negative for not at all
	}{
		{"on the reach, heading in", maths.V(0, 0, -40), -1},
		{"a hair outside, heading in", maths.V(0, 0, -40.001), 0.001},
		{"half a unit outside, heading in", maths.V(0, 0, -40.5), 0.5},
		{"touching the reach", maths.V(0, 40, -100), 100},
		{"clipping the reach", maths.V(0, 39.9999, -100), 100 - math.Sqrt(1600-39.9999*39.9999)},
		{"missing the reach by a hair", maths.V(0, 40.0001, -100), -1},
	} {
		_, d, ok := entering(fs, maths.Ray{At: c.from, Dir: ahead})
		if c.enter < 0 {
			if ok {
				t.Errorf("%s: entered at %v", c.name, d)
			}
			continue
		}
		if !ok || !near(d, c.enter, 1e-9) {
			t.Errorf("%s: enters at %v, %v; want %v", c.name, d, ok, c.enter)
		}
	}

	// a field of no reach at all is a point, and a ray 1e-9 from it is on
	// it
	point := []field{{f: &thing.Field{Mass: 1}, scale: 1}}
	if _, d, ok := entering(point, maths.Ray{At: maths.V(0, 0, -1e-9), Dir: ahead}); ok {
		t.Errorf("a point 1e-9 ahead is entered at %v", d)
	}

	// two fields entered together: the first listed
	w := thing.NewWorld()
	w.AddField(thing.Field{Mass: 1, Reach: 40})
	w.AddField(thing.Field{Mass: 2, Reach: 40})
	if f, _, ok := entering(placeFields(w, 0), maths.Ray{At: maths.V(0, 0, -100), Dir: ahead}); !ok || f.f.Mass != 1 {
		t.Errorf("two fields entered together gave the one of mass %v", f.f.Mass)
	}
}

// the eye inside a field is held still where it is, or is what it is told. a
// still eye has the time part of a still observer's four-velocity, one over the
// root of one less 2 M over r, and nothing else, which at 10 M out is 1 over
// the root of 0.8.
//
// a field told its eye's time and direction gives them as the hole's maths has
// them, the direction turned from the field's axes, (0.3, 0.4, 0.5) to (0.3,
// -0.5, 0.4). a time of zero, or less, is not told.
func TestTheEyeIsHeldStillOrTold(t *testing.T) {
	x := kerr.NewVec3(0, 10, 0)
	still := kerr.NewVec4(1/math.Sqrt(0.8), 0, 0, 0)
	told := func(time float64) field {
		return placedField(thing.Field{Mass: 1, Reach: 40, EyeTime: time, Eye: maths.V(0.3, 0.4, 0.5)})
	}
	for _, c := range []struct {
		name string
		f    field
		want kerr.Vec4
	}{
		{"an eye of no time", told(0), still},
		{"an eye of less than no time", told(-1), still},
		{"an eye of a time of one", told(1), kerr.NewVec4(1, 0.3, -0.5, 0.4)},
		{"an eye of a time of one and a half", told(1.5), kerr.NewVec4(1.5, 0.3, -0.5, 0.4)},
		{"no eye told", placedField(thing.Field{Mass: 1, Reach: 40}), still},
	} {
		if got := c.f.eye(x); !same4(got, c.want, 1e-12) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// a march starts from the eye it is given. an eye outside the reach is
// an observer far away, whose four-velocity is the time axis alone. the
// ray is as an eye held still at the reach sees it, with an energy of
// one, so the conserved energy is the root of one less 2 M over 40 M.
//
// an eye inside the reach is the eye the field is told. and the march
// starts on the reach, in the field's place.
func TestAMarchStartsFromTheEyeItIsGiven(t *testing.T) {
	w := hole(nil)
	sc := stage(w, 0)
	f := sc.fs[0]
	far := maths.Ray{At: maths.V(0, 6, -100), Dir: maths.V(0, 0, 1)}
	_, enter, _ := entering(sc.fs, far)
	m := &marcher{sc: sc, f: f, keep: maths.Grey(1)}
	m.start(far, enter)
	if m.shift.obs != kerr.NewVec4(1, 0, 0, 0) {
		t.Errorf("an eye far away is %v", m.shift.obs)
	}
	if want := math.Sqrt(1 - 2.0/40); !near(m.shift.p0.Time(), want, 1e-12) || m.shift.p0 != m.ray.Momentum() {
		t.Errorf("its ray has energy %v, want %v; p0 %v, ray %v", m.shift.p0.Time(), want, m.shift.p0, m.ray.Momentum())
	}
	if got := m.from; !got.Near(far.Point(enter), 1e-9) {
		t.Errorf("the march starts at %v, want %v", got, far.Point(enter))
	}

	inside := maths.Ray{At: maths.V(0, 0, -10), Dir: maths.V(0, 0, 1)}
	m = &marcher{sc: sc, f: f, keep: maths.Grey(1)}
	m.start(inside, 0)
	if want := kerr.NewVec4(1/math.Sqrt(0.8), 0, 0, 0); !same4(m.shift.obs, want, 1e-12) {
		t.Errorf("an eye held still at 10 M is %v, want %v", m.shift.obs, want)
	}
	// its energy for the ray is one, as that eye measures it
	if e := kerr.Dot(m.ray.Momentum(), m.shift.obs); !near(e, 1, 1e-12) {
		t.Errorf("the eye measures the ray at %v, want 1", e)
	}

	told := placedField(thing.Field{Mass: 1, Reach: 40, EyeTime: 1.5, Eye: maths.V(0.3, 0.4, 0.5)})
	m = &marcher{sc: sc, f: told, keep: maths.Grey(1)}
	m.start(inside, 0)
	if want := kerr.NewVec4(1.5, 0.3, -0.5, 0.4); !same4(m.shift.obs, want, 1e-12) {
		t.Errorf("an eye that is told is %v, want %v", m.shift.obs, want)
	}
}

// an emitter, a volume's four-velocity in the hole's maths, is still where it
// has no beam, or a beam that does not flow at a speed between zero and one. it
// flows at a share of light's speed, along its axis away from its middle, in
// the frame of an observer held still there.
//
// or it goes round the hole, prograde, at a circular orbit's rate. worked by
// hand at 7 M out, a still emitter has a time part of one over the root of 1 -
// 2/7.
//
// round at 10 M out, a circular orbit has the time part 1 over the root of 1 -
// 3/10 and the turn 10 times the rate, the root of a tenth, times that, the
// root of a seventh. inside the photon orbit, at 2.5 M, there is none.
//
// a beam along (3, 4, 12) over 13 at 0.6 of light has a time part, as a still
// observer measures it, of 1.25, and a direction in the observer's own space of
// (3, -12, 4) over 13, turned into the hole's maths, at 0.6 of the rate.
func TestAnEmitterIsStillOrFlowsOrGoesRound(t *testing.T) {
	f := placedField(thing.Field{Mass: 1, Reach: 40})
	emit := func(b *thing.Beam, world maths.Vec) kerr.Vec4 {
		return f.emitter(volume{v: &thing.Volume{Beam: b}, scale: 1}, world)
	}
	at7 := maths.V(6, 3, 2)
	still := kerr.NewVec4(1/math.Sqrt(1-2.0/7), 0, 0, 0)
	for _, c := range []struct {
		name string
		beam *thing.Beam
	}{
		{"no beam", nil},
		{"a beam of no speed", &thing.Beam{Speed: 0}},
		{"a round beam of no speed", &thing.Beam{Speed: 0, Round: true, At: 5}},
		{"a beam backward", &thing.Beam{Speed: -0.5}},
		{"a beam at the speed of light", &thing.Beam{Speed: 1}},
		{"a beam past it", &thing.Beam{Speed: 1.5}},
		{"a beam far past it", &thing.Beam{Speed: 2}},
	} {
		if got := emit(c.beam, at7); !same4(got, still, 1e-12) {
			t.Errorf("%s: %v, want a still emitter's %v", c.name, got, still)
		}
	}

	round := emit(&thing.Beam{Speed: 0.5, Round: true, At: 5}, maths.V(10, 0, 0))
	if want := kerr.NewVec4(1/math.Sqrt(0.7), 0, math.Sqrt(1.0/7), 0); !same4(round, want, 1e-12) {
		t.Errorf("round at 10 M: %v, want %v", round, want)
	}
	if got := emit(&thing.Beam{Speed: 0.5, Round: true, At: 5}, maths.V(2.5, 0, 0)); !nan4(got) {
		t.Errorf("round inside the photon orbit: %v, want no number", got)
	}

	// flowing: the invariants of a four-velocity of speed 0.6 along a
	// direction, as a still observer sees it
	beam := &thing.Beam{Axis: maths.V(3, 4, 12), Speed: 0.6}
	u := emit(beam, at7)
	x := kerr.NewVec3(6, -2, 3)
	frame := f.hole.FrameOf(x, f.hole.Static(x))
	lowered := f.hole.Lower(x, u)
	if norm := kerr.Dot(lowered, u); !near(norm, -1, 1e-12) {
		t.Errorf("the flowing emitter has a length of %v, want -1", norm)
	}
	if e := -kerr.Dot(lowered, frame.Observer()); !near(e, 1.25, 1e-12) {
		t.Errorf("the still observer measures it at %v, want 1.25", e)
	}
	spatial := []kerr.Vec4{frame.XAxis(), frame.YAxis(), frame.ZAxis()}
	for i, d := range []float64{3.0 / 13, -12.0 / 13, 4.0 / 13} {
		if got, want := kerr.Dot(lowered, spatial[i]), 1.25*0.6*d; !near(got, want, 1e-12) {
			t.Errorf("along the observer's axis %d it goes at %v, want %v", i+1, got, want)
		}
	}
}

// the shift of the light of an emitter is its gain, the fourth power of the
// ratio of energies, as the tracer has it. for a still emitter 4 M out and an
// eye far away of the same energy, the ratio is the root of one less 2 M over
// r, a half to the root, and its fourth power a quarter.
//
// a ratio of a half gives a sixteenth; of 3 over the root of 2, 2.12, gives
// 20.25. past 50 it is 50, so that one pixel does not bloom a frame, a ratio of
// 4 over the root of 2 being 64.
//
// a ratio under zero, or one that is no number, as from a still emitter inside
// the horizon, gives nothing. round a hole spinning at 0.9, nothing is still
// inside the ergosphere, which is 2 M out on the equator and at the horizon on
// the axis.
func TestTheGainIsTheFourthPowerOfTheShift(t *testing.T) {
	f := placedField(thing.Field{Mass: 1, Reach: 40})
	c := crossing{v: volume{v: &thing.Volume{}, scale: 1}}
	energy := func(e float64) *shifter {
		return &shifter{f: f, p0: kerr.NewVec4(e, 0, 0, 0), obs: kerr.NewVec4(1, 0, 0, 0),
			p: kerr.NewVec4(1, 0, 0, 0)}
	}
	four := maths.V(4, 0, 0)
	for _, tc := range []struct {
		name  string
		e     float64
		world maths.Vec
		want  float64
	}{
		{"a still emitter at 4 M", 1, four, 0.25},
		{"a ratio of a half", math.Sqrt(2) / 2, four, 1.0 / 16},
		{"a ratio of 2.12", 3, four, 20.25},
		{"a ratio past 50", 4, four, 50},
		{"a ratio of nought", 0, four, 0},
		{"a ratio under nought", -1, four, 0},
		{"a still emitter inside the horizon", 1, maths.V(1, 0, 0), 0},
	} {
		if got := energy(tc.e).gain(c, tc.world); !near(got, tc.want, 1e-9) {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}

	spinning := placedField(thing.Field{Mass: 1, Spin: 0.9, Reach: 40})
	s := &shifter{f: spinning, p0: kerr.NewVec4(1, 0, 0, 0), obs: kerr.NewVec4(1, 0, 0, 0),
		p: kerr.NewVec4(1, 0, 0, 0)}
	for _, tc := range []struct {
		name  string
		world maths.Vec
		still bool
	}{
		// the equator is the hole's x and y: the field's x and minus z
		{"inside the ergosphere on the equator", maths.V(2.0, 0, 0), false},
		{"outside it on the equator", maths.V(2.4, 0, 0), true},
		// the axis is the field's up
		{"inside the horizon on the axis", maths.V(0, 1.3, 0), false},
		{"outside it on the axis", maths.V(0, 1.6, 0), true},
	} {
		got := s.gain(c, tc.world)
		if tc.still != (got > 0) {
			t.Errorf("a still emitter %s gives %v", tc.name, got)
		}
	}
}

// the gain on a crossing is taken at the middle of the crossing, where
// the emitter is.
//
// a glowing block of tau 1 at 6 M from a hole is a still emitter whose
// ratio to an eye far away is the root of one less 2 over 6. it gives
// off its glow times the gain at 6, 4 over 9, and takes the light behind
// it by exp(-1).
func TestTheGainIsTakenAtTheMiddleOfACrossing(t *testing.T) {
	w := hole(nil)
	w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(maths.Grey(1)),
		Motion: at(maths.V(6, 0, 0), 1)})
	sc := stage(w, 0)
	s := &shifter{f: sc.fs[0], p0: kerr.NewVec4(1, 0, 0, 0), obs: kerr.NewVec4(1, 0, 0, 0),
		p: kerr.NewVec4(1, 0, 0, 0)}
	keep, added := throughGain(sc, maths.Ray{At: maths.V(6, 0, 20), Dir: maths.V(0, 0, -1)},
		math.Inf(1), s, 48)
	if want := maths.Grey(math.Exp(-1)); !near(keep.R, want.R, 1e-12) {
		t.Errorf("the block lets %v through, want %v", keep, want)
	}
	if want := (1 - math.Exp(-1)) * 4 / 9; !near(added.R, want, 1e-12) || !near(added.B, want, 1e-12) {
		t.Errorf("the block gives off %v, want %v", added, want)
	}
}
