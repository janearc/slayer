package pinnipeds

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestWalrusCreaseIsDeepestOnItsLine(t *testing.T) {
	line := walrusCreaseLine{at: 1, lean: .1, from: .2, to: 1.8, depth: .1,
		width: .03}
	const round = 1.
	on := line.cut(line.at-line.lean*round, round, 1)
	if !(math.Abs(on-line.depth) <= 1e-9) {
		t.Fatalf("on its line the crease cuts %.4f, not %.4f", on, line.depth)
	}
	beside := line.cut(line.at-line.lean*round+line.width, round, 1)
	if !(beside < on) {
		t.Fatalf("a width off its line the crease cuts %.4f", beside)
	}
	if far := line.cut(line.at+.5, round, 1); !(far < 1e-9) {
		t.Fatalf("half a metre off its line the crease cuts %.4f", far)
	}
}

func TestWalrusCreaseStopsAtItsEnds(t *testing.T) {
	line := walrusCreaseLine{at: 1, from: .8, to: 1.6, depth: .1, width: .03}
	for _, round := range []float64{.8 - walrusCreaseFade,
		1.6 + walrusCreaseFade, 3} {
		if got := line.cut(line.at, round, 1); got != 0 {
			t.Fatalf("at %.2f rad round, past its end, the crease cuts %.4f",
				round, got)
		}
	}
}

func TestWalrusCreaseSidesMeetOnTheBack(t *testing.T) {
	for i, line := range walrusCreaseLines {
		for _, x := range []float64{line.at - .02, line.at, line.at + .02} {
			left, right := line.cut(x, 0, -1), line.cut(x, 0, 1)
			if left != right {
				t.Fatalf("crease %d at %.2f: %.5f on the left of the back's "+
					"middle and %.5f on the right", i, x, left, right)
			}
		}
	}
}

func TestWalrusNoCreaseIsARing(t *testing.T) {
	// A crease that ran from the back's middle to the belly's on both
	// sides would close: none comes within a fade of doing so.
	for i, line := range walrusCreaseLines {
		if !(line.to+walrusCreaseFade < math.Pi) {
			t.Fatalf("crease %d runs to %.2f rad, the whole way round", i, line.to)
		}
		if !(line.from >= 0 && line.from < line.to) {
			t.Fatalf("crease %d runs from %.2f to %.2f", i, line.from, line.to)
		}
	}
}

func TestWalrusCreasesKeepToTheNeckAndShoulders(t *testing.T) {
	creases := walrusCreasesOf(testWalrusOutline(t))
	outline := testWalrusOutline(t)
	for _, x := range []float64{-1.2, -.5, 0, walrusCreaseBack - .01,
		walrusCreaseFront + .01, 1.7} {
		section := outline.at(x)
		for _, p := range []maths.Vec{
			maths.V(x, section.centre+section.high, 0),
			maths.V(x, section.centre, section.wide),
			maths.V(x, section.centre, -section.wide),
		} {
			if got := creases(p); got != 0 {
				t.Fatalf("a crease cuts %.4f at %v", got, p)
			}
		}
	}
	deepest := 0.
	for x := walrusCreaseBack; x < walrusCreaseFront; x += .005 {
		section := outline.at(x)
		point := maths.V(x, section.centre,
			section.wide)
		deepest = math.Max(deepest, creases(point))
	}
	if !(deepest > .05) {
		t.Fatalf("no crease cuts the flank of the neck: deepest %.4f", deepest)
	}
}
