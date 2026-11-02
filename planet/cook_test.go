package planet

import (
	"math"
	"testing"
)

// cookingFor is a face that cooks its air over so many seconds.
func cookingFor(seconds float64) *Face {
	return &Face{cooking: seconds}
}

// fresh air has not cooked at all, nor has air of no age, nor air on a
// face that does not cook.
func TestFreshAirIsUncooked(t *testing.T) {
	for _, x := range []struct{ cooking, age float64 }{
		{1e6, 0}, {1e6, -5e5}, {0, 1e9},
	} {
		rung, toward := cookingFor(x.cooking).cookedBy(x.age)
		if rung != 0 || toward != 0 {
			t.Errorf("cooking %g, age %g: rung %d toward %g, want nought",
				x.cooking, x.age, rung, toward)
		}
	}
}

// older air is never less cooked, and each reading sits on a rung that
// has one above it, some way toward it.
func TestCookingOnlyDeepens(t *testing.T) {
	face := cookingFor(1e6)
	last := -1.0
	for age := 0.0; age <= 1e7; age += 1e4 {
		rung, toward := face.cookedBy(age)
		if rung < 0 || rung > rungs-2 || toward < 0 || toward > 1 {
			t.Fatalf("age %g: rung %d toward %g, off the ladder", age, rung,
				toward)
		}
		along := float64(rung) + toward
		if along < last {
			t.Fatalf("age %g: %g along, less than %g before", age, along, last)
		}
		last = along
	}
}

// air cooked a thousand times its cooking is all but full: at the top
// rung's foot, all the way toward the top.
func TestLongCookedAirIsFull(t *testing.T) {
	rung, toward := cookingFor(1e6).cookedBy(1e9)
	if rung != rungs-2 || math.Abs(toward-1) > 1e-9 {
		t.Errorf("rung %d toward %g, want %d toward 1", rung, toward, rungs-2)
	}
}

// air of one cooking's age has cooked 1 - 1/e of the way up the ladder.
func TestOneCookingIsOneMinusOneOverE(t *testing.T) {
	rung, toward := cookingFor(1e6).cookedBy(1e6)
	along := (float64(rung) + toward) / (rungs - 1)
	if want := 1 - math.Exp(-1); math.Abs(along-want) > 1e-12 {
		t.Errorf("%g of the way, want %g", along, want)
	}
}

// the ladder's rungs bear the cooked haze in the column's top layer,
// none at the foot and all of it at the top, evenly between, and leave
// every other layer, and the column it was made from, as they were.
func TestTheLadderRunsFromNoneToAll(t *testing.T) {
	gas := mustGas(t, "gas", flat(0), 0.02)
	cooked, err := NewHaze(1.5, 0.95, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	below, err := NewHaze(8, 0.99, 0.1)
	if err != nil {
		t.Fatal(err)
	}
	column := []Layer{layerOf(t, gas, 0, 1e4, Haze{}),
		layerOf(t, gas, 1e4, 1e5, below)}
	ladder := Body{cooked: cooked}.ladder(column)
	if len(ladder) != rungs {
		t.Fatalf("%d rungs, want %d", len(ladder), rungs)
	}
	for k, made := range ladder {
		want := cooked.depth * float64(k) / (rungs - 1)
		if got := made[0].haze.depth; math.Abs(got-want) > 1e-12 {
			t.Errorf("rung %d: top haze depth %g, want %g", k, got, want)
		}
		if made[0].haze.albedo != cooked.albedo ||
			made[0].haze.lean != cooked.lean {
			t.Errorf("rung %d: top haze is not the cooked haze", k)
		}
		if made[1].haze != below {
			t.Errorf("rung %d: the layer below changed", k)
		}
	}
	if column[0].haze != (Haze{}) {
		t.Errorf("the column the ladder was made from changed")
	}
}

// cooking needs a hazed layer to thicken, and a time to cook over.
func TestCookingNeedsAHaze(t *testing.T) {
	gas := mustGas(t, "gas", flat(0), 0.02)
	hazed, err := NewHaze(1, 0.9, 0)
	if err != nil {
		t.Fatal(err)
	}
	bare := []Layer{layerOf(t, gas, 0, 1e5, Haze{})}
	if _, err := (Body{}).Cooked(bare, 1e6); err == nil {
		t.Errorf("cooked a column with no haze")
	}
	held := []Layer{layerOf(t, gas, 0, 1e5, hazed)}
	if _, err := (Body{}).Cooked(held, 0); err == nil {
		t.Errorf("cooked over no time")
	}
	body, err := (Body{}).Cooked(held, 1e6)
	if err != nil {
		t.Fatal(err)
	}
	if body.cooked != hazed || body.cooking != 1e6 {
		t.Errorf("cooked %+v over %g s, want the column's haze over 1e6",
			body.cooked, body.cooking)
	}
}
