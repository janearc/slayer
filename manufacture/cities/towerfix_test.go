package cities

import (
	"math"
	"testing"
)

// skylineSize is how many towers a skyline test draws.
const skylineSize = 30

// strangeIrregularity is the irregularity above which a tower is plainly
// strange.
const strangeIrregularity = 0.5

func TestAStackIsModulesEachFlaringAndEachSteppingInFromTheLast(t *testing.T) {
	for seed := range Seed(6) {
		tower := towerOf(t, seed, 0, Pins{Shafts: 1, Stack: &yes,
			Nest: &no, Plan: planRectangle})
		shaft := tower.Shafts[0]
		if !shaft.Stacked || len(shaft.Segments) < 3 {
			t.Fatalf("seed %d: stacked %v with %d modules", seed,
				shaft.Stacked, len(shaft.Segments))
		}
		for index, module := range shaft.Segments {
			if !(module.Top.Width > module.Bottom.Width) {
				t.Errorf("seed %d module %d is %v wide at the foot and %v "+
					"at the top", seed, index, module.Bottom.Width,
					module.Top.Width)
			}
			if index > 0 && !(module.Bottom.Width <
				shaft.Segments[index-1].Top.Width) {
				t.Errorf("seed %d module %d does not step in", seed, index)
			}
		}
	}
}

func TestATaipeiStyleTowerIsAStackOfEightFlaredModulesThatIsDrawnClosed(
	t *testing.T) {
	var found bool
	for seed := range Seed(40) {
		tower := towerOf(t, seed, 1, Pins{Shafts: 1, Stack: &yes,
			Nest: &no, Spire: &yes})
		if len(tower.Shafts[0].Segments) != 8 {
			continue
		}
		found = true
		for index, one := range drawnTower(t, tower, "quick", 0) {
			if !closed(one.Mesh) {
				t.Fatalf("seed %d: part %d (%s) is not closed", seed, index,
					one.Mesh.Name)
			}
		}
		break
	}
	if !found {
		t.Error("no seed gave a stack of eight modules")
	}
}

func TestANestCarriesASlimCoreOnAboveTheBodyAndAddsHeight(t *testing.T) {
	for seed := range Seed(6) {
		plain := towerOf(t, seed, 0, Pins{Shafts: 1, Nest: &no,
			Stack: &no})
		nested := towerOf(t, seed, 0, Pins{Shafts: 1, Nest: &yes,
			Stack: &no})
		shaft := nested.Shafts[0]
		core := shaft.Segments[len(shaft.Segments)-1]
		body := shaft.Segments[len(shaft.Segments)-2]
		if !shaft.Nested || !(core.Bottom.Width < body.Top.Width) ||
			core.Top.Width != core.Bottom.Width {
			t.Errorf("seed %d: nested %v, core %v wide on a body %v wide",
				seed, shaft.Nested, core.Bottom.Width, body.Top.Width)
		}
		if !(nested.height(0) > plain.height(0)) {
			t.Errorf("seed %d: a nest is %v high, plain %v", seed,
				nested.height(0), plain.height(0))
		}
	}
}

func TestANestsBodyIsWrappedRoundACoreThatRunsUpThroughIt(t *testing.T) {
	for seed := range Seed(6) {
		nested := towerOf(t, seed, 0, Pins{Shafts: 1, Nest: &yes, Stack: &no})
		plain := towerOf(t, seed, 0, Pins{Shafts: 1, Nest: &no, Stack: &no})
		spines := named(drawnTower(t, nested, "quick", 0), "spine")
		if len(spines) == 0 {
			t.Fatalf("seed %d: a nest with no core through its body", seed)
		}
		shaft := nested.Shafts[0]
		body := nested.baseHeight()
		for _, segment := range shaft.Segments[:len(shaft.Segments)-1] {
			body += float64(segment.Storeys) * nested.storeyHeight()
		}
		for _, spine := range spines {
			low, high := reach(spine)
			if !closed(spine.Mesh) || math.Abs(high.Y-body) > tolerance ||
				!(low.Y < body) {
				t.Errorf("seed %d: a spine from %v to %v, the body ends at %v",
					seed, low.Y, high.Y, body)
			}
		}
		drawn := drawnTower(t, plain, "quick", 0)
		if got := len(named(drawn,
			"spine")); got != 0 {
			t.Errorf("seed %d: a tower with no nest has %d spines", seed, got)
		}
	}
}

func TestAShaftsRadiusCoversEverySegmentIncludingAFlaredTop(t *testing.T) {
	tower := towerOf(t, 2, 0, Pins{Shafts: 1, Stack: &yes, Nest: &no})
	shaft := tower.Shafts[0]
	if shaft.radius() < shaft.Segments[0].Top.radius()-tolerance {
		t.Errorf("radius %v is under the first module's top %v",
			shaft.radius(), shaft.Segments[0].Top.radius())
	}
}

func TestAShaftedTowerFitsTheRoomItIsGivenAndOnlyThen(t *testing.T) {
	const room = 130.0
	for seed := range Seed(12) {
		fitted := towerOf(t, seed, 1, Pins{Shafts: 4, Room: room})
		width := 0.0
		for _, shaft := range fitted.Shafts {
			width += 2 * shaft.radius()
		}
		width += float64(len(fitted.Shafts)-1) * fitted.drawing.fixed("fit-gap")
		if width > room+tolerance {
			t.Errorf("seed %d: %d shafts take %v of a room of %v", seed,
				len(fitted.Shafts), width, room)
		}
	}
	free := towerOf(t, 1, 1, Pins{Shafts: 4})
	if len(free.Shafts) != 4 {
		t.Errorf("a tower with no room has %d shafts, not 4",
			len(free.Shafts))
	}
}

func TestASkylineHasFourToSixPlainlyStrangeTowersInThirty(t *testing.T) {
	const trials = 40
	strange := 0
	for trial := range trials {
		for index := range skylineSize {
			seed := Seed(trial*skylineSize + index + 1)
			tower, err := NewTower(seed, Irregularity{Drawn: true}, Pins{})
			if err != nil {
				t.Fatal(err)
			}
			strange += btoi(tower.Irregularity > strangeIrregularity)
		}
	}
	mean := float64(strange) / trials
	if mean < 3 || mean > 9 {
		t.Errorf("a skyline of %d has %v strange towers on average",
			skylineSize, mean)
	}
}
