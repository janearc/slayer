package typography

import (
	"math"
	"testing"

	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

// on is the point a share t of the way along a piece, by de casteljau.
func on(p piece, t float64) point {
	run := append(piece(nil), p...)
	for len(run) > 1 {
		for i := 0; i+1 < len(run); i++ {
			run[i] = run[i].scale(1 - t).add(run[i+1].scale(t))
		}
		run = run[:len(run)-1]
	}
	return run[0]
}

// strays is the furthest that any of so many points along a piece is
// from the steps that stand for it, starting at the piece's first.
func strays(p piece, steps ring, samples int) float64 {
	path := append(ring{p[0]}, steps...)
	worst := 0.0
	for s := 0; s <= samples; s++ {
		q, near := on(p, float64(s)/float64(samples)), math.Inf(1)
		for i := 0; i+1 < len(path); i++ {
			near = math.Min(near, apart(path[i], path[i+1], q))
		}
		worst = math.Max(worst, near)
	}
	return worst
}

// mustFont is one of go's own fonts, which every machine has alike.
func mustFont(t *testing.T, raw []byte) *Font {
	t.Helper()
	made, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// curves is every curve in the outlines of so many letters of a font.
func curves(t *testing.T, face *Font, letters string) []piece {
	t.Helper()
	var made []piece
	for _, letter := range letters {
		glyph, err := face.Index(letter)
		if err != nil {
			t.Fatal(err)
		}
		outline, err := face.outline(glyph)
		if err != nil {
			t.Fatal(err)
		}
		for _, contour := range outline {
			for _, each := range contour {
				if len(each) > 2 {
					made = append(made, each)
				}
			}
		}
	}
	return made
}

// a cubic that bends hard one way and back: the italic's curves are all
// quadratics, as truetype's are, so the cubics are written here.
var cubics = []piece{
	{{0, 0}, {300, 900}, {700, -900}, {1000, 0}},
	{{0, 0}, {0, 500}, {500, 500}, {500, 0}},
	{{0, 0}, {1000, 1000}, {0, 1000}, {1000, 0}},
}

// every curve of go's regular and italic letters, and cubics that bend
// both ways, flattened at four tolerances, stays within the tolerance of
// the true curve at every one of five hundred points along it.
func TestFlatteningStaysWithinItsTolerance(t *testing.T) {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ@&8"
	all := append(curves(t, mustFont(t, goregular.TTF), letters),
		curves(t, mustFont(t, goitalic.TTF), letters)...)
	all = append(all, cubics...)
	for _, tolerance := range []float64{0.25, 1, 4, 16} {
		for i, each := range all {
			steps := flatten(each, tolerance, deepest, nil)
			if got := strays(each, steps, 500); got > tolerance {
				t.Errorf("curve %d at %v strays %v",
					i, tolerance, got)
			}
		}
	}
}

// the check above can fail: a curve stood for by the one step between
// its ends strays further than its tolerance, and strays says so.
func TestAStraightStepForACurveStraysPastItsTolerance(t *testing.T) {
	for i, each := range cubics {
		chord := ring{each[len(each)-1]}
		if got := strays(each, chord, 500); got <= 16 {
			t.Errorf("cubic %d by its chord strays only %v", i, got)
		}
	}
}

// a curve that bends harder is cut into more steps, a finer tolerance
// cuts any curve into more, and a curve that does not bend at all, its
// points on one line, is one step.
func TestATighterCurveGetsMoreSteps(t *testing.T) {
	gentle := piece{{0, 0}, {500, 40}, {1000, 0}}
	tight := piece{{0, 0}, {500, 800}, {1000, 0}}
	flat := piece{{0, 0}, {300, 0}, {700, 0}, {1000, 0}}
	count := func(p piece, tolerance float64) int {
		return len(flatten(p, tolerance, deepest, nil))
	}
	if g, k := count(gentle, 1), count(tight, 1); g >= k {
		t.Errorf("gentle in %d steps, tight in %d", g, k)
	}
	if c, f := count(tight, 4), count(tight, 1); c >= f {
		t.Errorf("at tolerance 4, %d steps; at 1, %d", c, f)
	}
	if got := count(flat, 0.001); got != 1 {
		t.Errorf("a straight cubic in %d steps", got)
	}
	if got := len(flattened([]piece{gentle, {{1000, 0}, {0, 0}}}, 1)); got <
		3 {
		t.Errorf("a contour of a curve and a line in %d points", got)
	}
}

// the two halves of a curve are the curve: the front half at t is the
// whole at t over two, the back at t is the whole at a half and t over
// two, for quadratics and cubics.
func TestTheHalvesOfACurveAreTheCurve(t *testing.T) {
	quadratic := piece{{0, 0}, {5, 9}, {10, 0}}
	for _, each := range append([]piece{quadratic}, cubics...) {
		front, back := halves(each)
		for s := 0; s <= 10; s++ {
			u := float64(s) / 10
			early := on(front, u).sub(on(each, u/2)).length()
			late := on(back, u).sub(on(each, 0.5+u/2)).length()
			if early > 1e-9 || late > 1e-9 {
				t.Errorf("%v halved strays at %v", each, u)
			}
		}
	}
}

// bend bounds how far a curve strays from the step between its ends, for every
// curve of go's regular letters and the cubics, sampled; a quadratic pulled
// straight up from the middle of its step strays half as far as its pull, and
// bend is the pull.
func TestBendIsNeverLessThanHowFarACurveStrays(t *testing.T) {
	letters := "abcdefghijklmnopqrstuvwxyzOSQ@"
	for i, each := range append(curves(t, mustFont(t, goregular.TTF),
		letters), cubics...) {
		chord := ring{each[len(each)-1]}
		got, most := strays(each, chord, 500), bend(each)
		if got > most {
			t.Errorf("curve %d strays %v, past its bend %v", i, got,
				most)
		}
	}
	p := piece{{0, 0}, {50, 80}, {100, 0}}
	if got := bend(p); got != 80 {
		t.Errorf("bend %v, want the pull of 80", got)
	}
	if got := strays(p, ring{p[2]}, 1000); math.Abs(got-40) > 1e-9 {
		t.Errorf("the quadratic strays %v, want 40", got)
	}
}
