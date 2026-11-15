package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"pgregory.net/rapid"
)

// tumblerGlass and poured are a glass and the water in it, as two clear
// materials: the glass outranks the water where they overlap.
func tumblerGlass() *thing.Material {
	return &thing.Material{Name: "glass", Gloss: 1, Index: 1.5, Through: 1, Rank: 1}
}

func poured() *thing.Material {
	return &thing.Material{Name: "water", Gloss: 1, Index: 1.33, Through: 1}
}

// step is one clear surface a look meets, and what it should be to it.
type step struct {
	of       *thing.Material
	real     bool
	entering bool
	relative float64
}

// takenThrough takes a look through clear surfaces in order, checking
// each is what it should be, and hands back the look past the last.
func takenThrough(t *testing.T, steps []step) look {
	t.Helper()
	var seen look
	for at, each := range steps {
		got := seen.meets(each.of)
		if got.real != each.real || got.entering != each.entering ||
			math.Abs(got.relative-each.relative) > 1e-12 {
			t.Fatalf("surface %d, of %s: %+v, want real %v entering %v relative %v",
				at, each.of.Name, got, each.real, each.entering, each.relative)
		}
		if !seen.follows(got) {
			t.Fatalf("surface %d, of %s: the look cannot go on past it", at, each.of.Name)
		}
		seen = seen.passed(seen.ray, each.of, got)
	}
	return seen
}

// A look through the middle of a glass of water bends four times and no
// more: from the air into the glass, from the glass into the water, from
// the water into the glass, and from the glass into the air. The water
// runs into the glass's wall, and its two faces inside the wall are no
// boundary: the look slips through them.
func TestALookThroughAGlassOfWaterBendsFourTimes(t *testing.T) {
	glass, water := tumblerGlass(), poured()
	// in order: the glass's outside; the water's face, inside the wall;
	// the glass's inside, glass to water; the far wall's inside, water
	// to glass; the water's far face, inside the wall; and the glass's
	// outside, into the air
	seen := takenThrough(t, []step{
		{glass, true, true, 1.5},
		{water, false, true, 1},
		{glass, true, false, 1.33 / 1.5},
		{glass, true, true, 1.5 / 1.33},
		{water, false, false, 1},
		{glass, true, false, 1 / 1.5},
	})
	if seen.crossed != 4 || seen.slipped != 2 || seen.holding != 0 {
		t.Errorf("past the glass of water the look has bent %d times, slipped %d, "+
			"and is inside %d things; want 4, 2 and none", seen.crossed,
			seen.slipped, seen.holding)
	}
}

// the anti-test of the one above: it is the rank that makes the water's
// faces in the wall no boundary. With no rank between them the water's
// face inside the wall is a boundary, glass to water, and a look bends
// there, where there is glass on both sides.
func TestWithoutARankTheWatersFaceInTheWallBends(t *testing.T) {
	glass, water := tumblerGlass(), poured()
	glass.Rank = 0
	var seen look
	seen = seen.passed(seen.ray, glass, seen.meets(glass))
	got := seen.meets(water)
	if !got.real || math.Abs(got.relative-1.33/1.5) > 1e-12 {
		t.Errorf("with equal ranks the water's face in the wall is %+v, "+
			"want a boundary of 1.33 over 1.5", got)
	}
}

// One clear thing in the air is as it always was: its index going in and
// the inverse coming out, a bend each, and nothing left inside.
func TestOneClearThingInTheAirIsAsItAlwaysWas(t *testing.T) {
	glass := tumblerGlass()
	seen := takenThrough(t, []step{{glass, true, true, 1.5}, {glass, true, false, 1 / 1.5}})
	if seen.crossed != 2 || seen.slipped != 0 || seen.holding != 0 {
		t.Errorf("through one glass the look has bent %d times, slipped %d and "+
			"is inside %d things", seen.crossed, seen.slipped, seen.holding)
	}
}

// the anti-test of the one above: two things of one index and one rank
// are one medium. The second sheet of a pane is the way out of the glass
// the first was the way into, though each is a thing of its own.
func TestTwoSheetsOfOneMediumAreItsTwoFaces(t *testing.T) {
	front, back := tumblerGlass(), tumblerGlass()
	if front == back {
		t.Fatal("the two sheets are one material; the test wants two")
	}
	seen := takenThrough(t, []step{{front, true, true, 1.5}, {back, true, false, 1 / 1.5}})
	if seen.holding != 0 {
		t.Errorf("past both faces of a pane the look is still inside %d things",
			seen.holding)
	}
}

// A look that has spent its bends can still slip through a surface that
// is no boundary, and one that has spent its slips can still bend. The
// anti-tests, one for each: spent of bends it does not bend again;
// spent of slips it does not slip again; and inside as many things as
// it can hold it enters no other, though it can still leave one.
func TestBendsAndSlipsAreCountedApart(t *testing.T) {
	bend := boundary{real: true, entering: true, relative: 1.5}
	slip := boundary{entering: true, relative: 1}
	spent := look{crossed: refractions}
	if spent.follows(bend) {
		t.Error("a look with no bends left bent again")
	}
	if !spent.follows(slip) {
		t.Error("a look with no bends left could not slip through no boundary")
	}
	slipped := look{slipped: slips}
	if slipped.follows(slip) {
		t.Error("a look with no slips left slipped again")
	}
	if !slipped.follows(bend) {
		t.Error("a look with no slips left could not bend")
	}
	full := look{holding: mostHeld}
	if full.follows(bend) || full.follows(slip) {
		t.Error("a look inside as many things as it can hold entered another")
	}
	if !full.follows(boundary{real: true, relative: 1 / 1.5}) {
		t.Error("a look inside as many things as it can hold could not leave one")
	}
}

// Through flat faces all square to one line, air, glass, water, glass and
// air again, a ray leaves the way it came: every bend is undone by the
// last. And in the water it runs at the angle it would if the glass were
// not there, for Snell's law across flat layers keeps the index times
// the sine the same in each. Any angle short of grazing, as a property.
func TestThroughFlatLayersARayLeavesAsItCame(t *testing.T) {
	face := maths.V(0, 0, 1)
	rapid.Check(t, func(draw *rapid.T) {
		angle := rapid.Float64Range(0, 85).Draw(draw, "degrees") * maths.Degree
		came := maths.V(math.Sin(angle), 0, -math.Cos(angle))
		way := came
		inWater := maths.Vec{}
		for at, relative := range []float64{1.5, 1.33 / 1.5, 1.5 / 1.33, 1 / 1.5} {
			bent, total := refracted(way, face, relative)
			if total {
				draw.Fatalf("at %v degrees the ray turned back at layer %d",
					angle/maths.Degree, at)
			}
			way = bent
			if at == 1 {
				inWater = bent
			}
		}
		if way.Sub(came).Len() > 1e-9 {
			draw.Fatalf("the ray came %v and left %v", came, way)
		}
		if math.Abs(1.33*inWater.X-math.Sin(angle)) > 1e-9 {
			draw.Fatalf("in the water the sine is %v; without the glass it is %v",
				inWater.X, math.Sin(angle)/1.33)
		}
	})
}

// A glass bead under water: a look goes from the air into the water,
// into the glass, out of the glass and out of the water. Leaving the
// glass leaves the look in the water it was in before, so the water's
// far face is a boundary of water against air. It is the anti-test of a
// leaving that forgets everything the look was inside.
func TestLeavingAGlassBeadLeavesTheLookInTheWater(t *testing.T) {
	glass, water := tumblerGlass(), poured()
	seen := takenThrough(t, []step{
		{water, true, true, 1.33},
		{glass, true, true, 1.5 / 1.33},
		{glass, true, false, 1.33 / 1.5},
		{water, true, false, 1 / 1.33},
	})
	if seen.crossed != 4 || seen.holding != 0 {
		t.Errorf("past the bead and the water the look has bent %d times and "+
			"is inside %d things; want 4 and none", seen.crossed, seen.holding)
	}
}

// Among clear media of one rank a look is in the one it entered last.
// The anti-test: the one it entered first is not forgotten, and when
// the look leaves the oil it is in the water again and not in the air.
func TestAmongEqualsTheMediumIsTheOneEnteredLast(t *testing.T) {
	water := poured()
	oil := &thing.Material{Name: "oil", Gloss: 1, Index: 1.47, Through: 1}
	var seen look
	seen = seen.passed(seen.ray, water, seen.meets(water))
	seen = seen.passed(seen.ray, oil, seen.meets(oil))
	in, some := seen.medium()
	if !some || in.index != 1.47 {
		t.Errorf("in water and then oil of one rank the look is in %+v, "+
			"want the oil it entered last", in)
	}
	seen = seen.passed(seen.ray, oil, seen.meets(oil))
	if in, some := seen.medium(); !some || in.index != 1.33 {
		t.Errorf("out of the oil the look is in %+v, %v; want the water "+
			"it entered first", in, some)
	}
}

// A look inside nothing is in the air, which has no rank to outrank
// anything: a clear thing of a rank below nought, met from the air, is
// still a boundary. The anti-test: the air is no medium a look is
// inside, and a look inside nothing does not say it is inside one.
func TestAThingOfLowRankInTheAirIsStillABoundary(t *testing.T) {
	low := &thing.Material{Name: "low", Gloss: 1, Index: 1.33, Through: 1, Rank: -1}
	var seen look
	if _, some := seen.medium(); some {
		t.Fatal("a look inside nothing says it is inside something")
	}
	got := seen.meets(low)
	if !got.real || math.Abs(got.relative-1.33) > 1e-12 {
		t.Errorf("a thing of rank -1 met from the air is %+v, want a boundary "+
			"of 1.33", got)
	}
}

// A look inside as many media as it can hold leaves one and is inside
// one fewer, without reaching past the end of its list.
func TestAFullLookLeavesOne(t *testing.T) {
	glass := tumblerGlass()
	full := look{holding: mostHeld}
	for at := range full.inside {
		full.inside[at] = of(glass)
	}
	left := full.passed(full.ray, glass, boundary{real: true, relative: 1 / 1.5})
	if left.holding != mostHeld-1 {
		t.Errorf("a full look that left one is inside %d, want %d",
			left.holding, mostHeld-1)
	}
}

// A look past a clear surface, bent through it or turned back by it, is
// a fresh leg: it has come no way yet, it is no longer the eye's own
// look, and being turned back costs a bend as passing does. The
// anti-test: a look turned back has not gone in, and is inside nothing.
func TestALegPastAClearSurfaceStartsFresh(t *testing.T) {
	glass := tumblerGlass()
	worn := look{come: 3, crossed: 1}
	through := worn.passed(worn.ray, glass, boundary{real: true, entering: true,
		relative: 1.5})
	if through.come != 0 || !through.onward || through.crossed != 2 {
		t.Errorf("past the surface the look has come %v, onward %v, bent %d; "+
			"want 0, true, 2", through.come, through.onward, through.crossed)
	}
	back := worn.turnedBack(worn.ray)
	if back.come != 0 || !back.onward || back.crossed != 2 || back.holding != 0 {
		t.Errorf("turned back the look has come %v, onward %v, bent %d, inside "+
			"%d; want 0, true, 2, 0", back.come, back.onward, back.crossed,
			back.holding)
	}
}
