package trace

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// the fixtures of the tests worked by hand. each is a world, or a part of
// one, whose answer can be had on paper: a medium as thick as one number
// all through a cube, so that the thickness a ray adds up is that number
// times its way through.

// block is a medium of thickness k all through a cube of half-width half
// about its middle, and clear outside it. it says where a ray crosses it
// by its box alone.
type block struct{ k, half float64 }

func (b block) Density(p maths.Vec) float64 {
	h := b.half
	if math.Abs(p.X) <= h && math.Abs(p.Y) <= h && math.Abs(p.Z) <= h {
		return b.k
	}
	return 0
}

func (b block) Bounds() maths.Box {
	h := b.half
	return maths.Box{Min: maths.V(-h, -h, -h), Max: maths.V(h, h, h)}
}

// ball is a medium of thickness k all through a ball, clear outside it,
// that says where a ray crosses it by its ball and not by its box.
type ball struct{ k, radius float64 }

func (b ball) Density(p maths.Vec) float64 {
	if p.Len() <= b.radius {
		return b.k
	}
	return 0
}

func (b ball) Bounds() maths.Box { return block{half: b.radius}.Bounds() }

func (b ball) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	mid := r.At.Dot(r.Dir)
	disc := mid*mid - (r.At.Dot(r.At) - b.radius*b.radius)
	if disc < 0 {
		return 0, 0, false
	}
	root := math.Sqrt(disc)
	enter, leave := math.Max(-mid-root, near), math.Min(-mid+root, far)
	return enter, leave, leave > enter
}

// told is a block that says where a ray crosses it as it is told to, and
// not as the block lies: the enter and leave it is given, and whether it
// was crossed. a medium may give any enter and leave when it says it was
// not crossed.
type told struct {
	block
	enter, leave float64
	crossed      bool
}

func (t told) Span(maths.Ray, float64, float64) (float64, float64, bool) {
	return t.enter, t.leave, t.crossed
}

// endlessly is a medium of endless thickness that says it is crossed
// from one distance to another, whatever ray is asked.
type endlessly struct{ enter, leave float64 }

func (endlessly) Density(maths.Vec) float64 { return math.Inf(1) }

func (endlessly) Bounds() maths.Box { return block{half: 100}.Bounds() }

func (e endlessly) Span(maths.Ray, float64, float64) (float64, float64, bool) {
	return e.enter, e.leave, true
}

// cornered is a ball of thickness k in a box that holds it and its
// corners: a ray through a corner of the box is in the box and out of the
// ball, and finds the medium clear all the way. it says where a ray is
// crossed by its box.
type cornered struct{ k, radius float64 }

func (c cornered) Density(p maths.Vec) float64 {
	if p.Len() <= c.radius {
		return c.k
	}
	return 0
}

func (c cornered) Bounds() maths.Box { return block{half: c.radius}.Bounds() }

// leaky is a block whose thickness does not stop at its box: k all
// through space, with the box only where a ray is told to look.
type leaky struct{ block }

func (l leaky) Density(maths.Vec) float64 { return l.k }

// at is a motion that stays at a place, at a scale.
func at(origin maths.Vec, scale float64) thing.Motion {
	return thing.Still(maths.Subspace{Origin: origin, Scale: scale})
}

// glowOf is a material that gives off a colour and takes none.
func glowOf(l maths.Light) *thing.Material { return &thing.Material{Glow: l} }

// along is a ray from the front, 20 out along z, going away down minus z
// from a place across: x and y.
func along(x, y float64) maths.Ray {
	return maths.Ray{At: maths.V(x, y, 20), Dir: maths.V(0, 0, -1)}
}

// within says whether a is within tol of b, in float32 light. a number
// that is no number is within nothing.
func within(a float32, b, tol float64) bool {
	return math.Abs(float64(a)-b) <= tol
}

// near says whether a is within tol of b. a number that is no number is
// near nothing, where a test written as a > tol would let it by.
func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// colours checks the three parts of a pixel against three numbers.
func colours(t *testing.T, what string, got interface {
	Light() maths.Light
}, want maths.Light, tol float64) {
	t.Helper()
	g := got.Light()
	if !near(g.R, want.R, tol) || !near(g.G, want.G, tol) ||
		!near(g.B, want.B, tol) {
		t.Errorf("%s: got %v, want %v", what, g, want)
	}
}

// ends runs call and fails the test if it has not come back in a few
// seconds. a loop that counts the wrong way never ends, and would hold
// up the whole run instead of failing in it.
func ends(t *testing.T, what string, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		call()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not finish in five seconds", what)
	}
}

// steady is a lamp that gives one answer wherever it is asked.
type steady struct {
	toward maths.Vec
	far    float64
	light  maths.Light
}

func (s *steady) Light(_, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	return s.toward, s.far, s.light
}

// tinted is a gas that has a number for each of the eye's three bands and
// none for any other wavelength: at 610, 550 and 465 nanometres the
// absorption and scattering given, and at any other a gas that takes
// everything, so a band asked at a wrong wavelength shows.
func tinted(red, green, blue gas.Optics) gas.Gas {
	return gas.Gas{Name: "tinted", Visible: func(nm float64) (float64, float64) {
		switch nm {
		case 610:
			return red.Absorb, red.Scatter
		case 550:
			return green.Absorb, green.Scatter
		case 465:
			return blue.Absorb, blue.Scatter
		}
		return 1e6, 0
	}}
}

// mediumOf is a volume of a medium and nothing else.
func mediumOf(m medium.Medium) thing.Volume { return thing.Volume{Medium: m} }
