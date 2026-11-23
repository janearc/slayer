package manufacture

import (
	"errors"
	"testing"
)

// yurtQuickParts and winnebagoQuickParts are how many parts each kind has
// at the quick level, which the demo level's detail must not change: the
// yurt's wall, roof, crown and door, and the winnebago's body, cab, six
// wheels, seven panes and door.
const (
	yurtQuickParts      = 4
	winnebagoQuickParts = 16
)

// partsOf is how many parts a good is made of, across its materials.
func partsOf(made good) int {
	total := 0
	for _, parts := range made.groupParts {
		total += parts
	}
	return total
}

// detailKinds are the kinds drawn part by part at the demo level, each
// with how it makes a good at a named level.
var detailKinds = []struct {
	name   string
	quick  int
	madeAt func(seed, string) (good, error)
	made   func(seed) (good, error)
}{
	{"yurt", yurtQuickParts, yurtRow{}.madeAt, yurtRow{}.made},
	{"winnebago", winnebagoQuickParts, winnebagoRow{}.madeAt,
		winnebagoRow{}.made},
}

func TestADetailKindAtTheDemoLevelHasMorePartsThanAtQuick(t *testing.T) {
	for _, kind := range detailKinds {
		quick, err := kind.madeAt(sow(1), "quick")
		if err != nil {
			t.Fatal(err)
		}
		demo, err := kind.madeAt(sow(1), "demo")
		if err != nil {
			t.Fatal(err)
		}
		if partsOf(demo) <= partsOf(quick) {
			t.Errorf("%s: %d parts at demo, %d at quick", kind.name,
				partsOf(demo), partsOf(quick))
		}
	}
}

func TestADetailKindAtQuickAndHullIsAsItWas(t *testing.T) {
	for _, kind := range detailKinds {
		made, err := kind.made(sow(1))
		if err != nil {
			t.Fatal(err)
		}
		if partsOf(made) != kind.quick {
			t.Errorf("%s: %d parts at quick, want %d", kind.name,
				partsOf(made), kind.quick)
		}
		hull, err := kind.madeAt(sow(1), "hull")
		if err != nil {
			t.Fatal(err)
		}
		if partsOf(hull) != kind.quick {
			t.Errorf("%s: %d parts at hull, want %d", kind.name,
				partsOf(hull), kind.quick)
		}
	}
}

func TestTheSameSeedMakesTheSamePartsAtTheDemoLevel(t *testing.T) {
	for _, kind := range detailKinds {
		first, err := kind.madeAt(sow(7), "demo")
		if err != nil {
			t.Fatal(err)
		}
		again, err := kind.madeAt(sow(7), "demo")
		if err != nil {
			t.Fatal(err)
		}
		if partsOf(first) != partsOf(again) ||
			first.Bounds() != again.Bounds() ||
			len(first.slots()) != len(again.slots()) {
			t.Errorf("%s: two makes from one seed differ", kind.name)
		}
	}
}

func TestABivouacIsMadeAtEveryLevelAndHasMorePartsAtDemo(t *testing.T) {
	for number := uint64(1); number <= 6; number++ {
		quick, err := bivouacRow{}.madeAt(sow(number), "quick")
		if err != nil {
			t.Fatal(err)
		}
		demo, err := bivouacRow{}.madeAt(sow(number), "demo")
		if err != nil {
			t.Fatal(err)
		}
		again, err := bivouacRow{}.madeAt(sow(number), "demo")
		if err != nil {
			t.Fatal(err)
		}
		if partsOf(demo) <= partsOf(quick) || partsOf(demo) != partsOf(again) ||
			demo.Bounds() != again.Bounds() {
			t.Errorf("seed %d: %d parts at demo, %d at quick, %d again",
				number, partsOf(demo), partsOf(quick), partsOf(again))
		}
	}
	if _, err := (bivouacRow{}).made(seed{}); !errors.Is(err, errGood) {
		t.Errorf("an unsown seed made a bivouac: %v", err)
	}
	if made := mustMake(t, "bivouac", 1); len(made.slots()) == 0 {
		t.Error("the bivouac kind made nothing")
	}
}
