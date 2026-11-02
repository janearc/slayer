package planet

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// mustRing is a ring, or the test stops.
func mustRing(t *testing.T, inner, outer, depth, albedo, lean float64) Ring {
	t.Helper()
	ring, err := NewRing(inner, outer, depth, albedo, lean)
	if err != nil {
		t.Fatal(err)
	}
	return ring
}

// a ring keeps what its depth leaves along the slant light crosses it
// at, which way it crosses.
func TestARingKeepsByItsSlant(t *testing.T) {
	ring := mustRing(t, 2, 3, 0.4, 0.8, -0.3)
	for _, slant := range []float64{1, 0.5, -0.5} {
		want := math.Exp(-0.4 / math.Abs(slant))
		if got := ring.kept(slant); !near(got, want, 1e-12) {
			t.Errorf("slant %g: %g, want %g", slant, got, want)
		}
	}
}

// a point whose way to the star crosses a ring is in its shadow; one
// whose way crosses the gap, or passes outside, or never meets the
// ring's plane, is not; and a star in the plane casts none.
func TestTheRingsShadowTheDisc(t *testing.T) {
	ring := mustRing(t, 2, 3, 0.4, 0.8, 0)
	rings := []Ring{ring}
	pole := maths.V(0, 1, 0)
	star := maths.V(1, 1, 0).Unit()
	slant := star.Dot(pole)
	for _, look := range []struct {
		at   maths.Vec
		want float64
	}{
		{maths.V(1.5, -1, 0), ring.kept(slant)},
		{maths.V(0.5, -1, 0), 1},
		{maths.V(2.5, -1, 0), 1},
		{maths.V(1.5, 1, 0), 1},
	} {
		got := shadowed(rings, look.at, star, pole)
		if !near(got, look.want, 1e-12) {
			t.Errorf("at %v: %g, want %g", look.at, got, look.want)
		}
	}
	if got := shadowed(rings, maths.V(0, -1, 0), maths.V(1, 0, 0),
		pole); got != 1 {
		t.Errorf("a star in the plane: %g", got)
	}
	// a step of one up the slant from x is a step of one out, so a point
	// at x meets the plane at x plus one
	if got := shadowed(rings, maths.V(1, -1, 0), star, pole); !near(got,
		ring.kept(slant), 1e-12) {
		t.Errorf("at the ring's inner edge: %g", got)
	}
	if got := shadowed(rings, maths.V(2, -1, 0), star, pole); got != 1 {
		t.Errorf("at the ring's outer edge: %g", got)
	}
}

// rings must stand clear of the body and of each other, inner to outer;
// and what a ring cannot be is refused.
func TestRingsStandClear(t *testing.T) {
	body := bareBody(t, 1, nil)
	clear := mustRing(t, 7e7, 8e7, 1, 0.5, 0)
	next := mustRing(t, 8e7, 9e7, 1, 0.5, 0)
	if _, err := body.Ringed([]Ring{clear, next}); err != nil {
		t.Errorf("touching rings refused: %v", err)
	}
	inside := mustRing(t, 5e7, 7e7, 1, 0.5, 0)
	for _, rings := range [][]Ring{{inside}, {next, clear}} {
		if _, err := body.Ringed(rings); err == nil {
			t.Errorf("rings %+v taken", rings)
		}
	}
	for _, wrong := range [][5]float64{{0, 1, 1, 0.5, 0}, {2, 2, 1, 0.5, 0},
		{1, 2, -1, 0.5, 0}, {1, 2, 1, 1.5, 0}, {1, 2, 1, -0.1, 0},
		{1, 2, 1, 0.5, 1}, {1, 2, 1, 0.5, -1}} {
		if _, err := NewRing(wrong[0], wrong[1], wrong[2], wrong[3],
			wrong[4]); err == nil {
			t.Errorf("a ring of %v", wrong)
		}
	}
	if _, err := NewRing(1, 2, 0, 0, 0); err != nil {
		t.Errorf("a clear, black ring refused: %v", err)
	}
	if _, err := NewRing(1, 2, 1, 1, 0.5); err != nil {
		t.Errorf("a white ring refused: %v", err)
	}
}

// a ring's slab is dense only in the ring and inside its thickness, its
// box is the slab's, and its depth through is the ring's, split between
// what it scatters and what it absorbs by its albedo.
func TestARingSlabHoldsTheRing(t *testing.T) {
	ring := mustRing(t, 2e7, 3e7, 0.6, 0.75, -0.3)
	slab, err := NewRingSlab(ring, 2e5)
	if err != nil {
		t.Fatal(err)
	}
	for _, look := range []struct {
		at   maths.Vec
		want float64
	}{
		{maths.V(2.5e7, 0, 0), 1}, {maths.V(0, 9e4, -2.5e7), 1},
		{maths.V(2e7, 0, 0), 1}, {maths.V(3e7, 0, 0), 0},
		{maths.V(1.9e7, 0, 0), 0}, {maths.V(2.5e7, 1.1e5, 0), 0},
		{maths.V(2.5e7, -1.1e5, 0), 0},
	} {
		if got := slab.Density(look.at); got != look.want {
			t.Errorf("at %v: %g", look.at, got)
		}
	}
	box := slab.Bounds()
	if box.Min != maths.V(-3e7, -1e5, -3e7) || box.Max != maths.V(3e7,
		1e5, 3e7) {
		t.Errorf("box %+v", box)
	}
	absorb, scatter := slab.Gas().Visible(550)
	if !near(scatter*2e5, 0.6*0.75, 1e-12) ||
		!near(absorb*2e5, 0.6*0.25, 1e-12) || slab.Gas().Phase != -0.3 {
		t.Errorf("absorb %g, scatter %g, phase %g", absorb, scatter,
			slab.Gas().Phase)
	}
	if _, err := NewRingSlab(ring, 0); err == nil {
		t.Error("a slab of no thickness")
	}
}

// a point of the disc in a ring's shadow is dimmed by the ring, in the
// face as in the shadow.
func TestTheFaceIsDimmedInTheRingsShadow(t *testing.T) {
	body := bareBody(t, 1, nil)
	ring := mustRing(t, 7e7, 9e7, 0.5, 0.5, 0)
	ringed, err := body.Ringed([]Ring{ring})
	if err != nil {
		t.Fatal(err)
	}
	pole, star := maths.V(0, 1, 0), maths.V(1, 1, 0).Unit()
	eye := maths.V(1e9, 0, 0)
	p := maths.V(math.Cos(-0.3), math.Sin(-0.3), 0).Scale(6e7)
	bare := NewFace(body, pole, star, eye, even(), 1).At(p)
	shaded := NewFace(ringed, pole, star, eye, even(), 1).At(p)
	want := shadowed([]Ring{ring}, p, star, pole)
	if !(want < 1) || !near(shaded.G/bare.G, want, 1e-6) {
		t.Errorf("shaded over bare %g, want %g", shaded.G/bare.G, want)
	}
}

// a star straight up the pole shadows the point straight under a ring;
// a point above the ring's plane, whose way to the star never crosses
// it, is not shadowed by the ring behind it; and a point just under the
// plane is shadowed by the ring just above it.
func TestTheShadowsEdges(t *testing.T) {
	ring := mustRing(t, 2, 3, 0.4, 0.8, 0)
	rings, pole := []Ring{ring}, maths.V(0, 1, 0)
	if got := shadowed(rings, maths.V(2.5, -1, 0), pole, pole); !near(got,
		ring.kept(1), 1e-12) {
		t.Errorf("a star overhead: %g", got)
	}
	star := maths.V(1, 1, 0).Unit()
	if got := shadowed(rings, maths.V(3.5, 1, 0), star, pole); got != 1 {
		t.Errorf("above the plane: %g", got)
	}
	if got := shadowed(rings, maths.V(2.5, -0.35, 0), star,
		pole); !near(got, ring.kept(star.Y), 1e-12) {
		t.Errorf("just under the plane: %g", got)
	}
}

// a slab half a metre thick is taken, and a point on its face is in it.
func TestASlabsThinEdges(t *testing.T) {
	ring := mustRing(t, 2, 3, 0.4, 0.8, 0)
	if _, err := NewRingSlab(ring, 0.5); err != nil {
		t.Errorf("half a metre refused: %v", err)
	}
	slab, _ := NewRingSlab(ring, 2)
	if got := slab.Density(maths.V(2.5, 1, 0)); got != 1 {
		t.Errorf("on the slab's face: %g", got)
	}
}

// a point in the ring's own plane does not cross it on its way to the
// star, so the ring casts no shadow on it.
func TestAPointInThePlaneIsNotShadowed(t *testing.T) {
	ring := mustRing(t, 2, 3, 0.4, 0.8, 0)
	star := maths.V(1, 1, 0).Unit()
	if got := shadowed([]Ring{ring}, maths.V(2.5, 0, 0), star,
		maths.V(0, 1, 0)); got != 1 {
		t.Errorf("%g", got)
	}
}
