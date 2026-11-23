package cities

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// drawerOf is a drawer of a culture for a seed, or the test's end.
func drawerOf(t *testing.T, culture string, seed Seed) *drawer {
	t.Helper()
	drawing, err := newDrawer(culture, seed)
	if err != nil {
		t.Fatal(err)
	}
	return drawing
}

func TestEveryCultureLoadsAndEveryElementIsPickedOrRecalled(t *testing.T) {
	names := cultureNames()
	for _, want := range []string{"groningen", "portofino", "midwest",
		"condominium"} {
		if !slices.Contains(names, want) {
			t.Errorf("no culture %s among %v", want, names)
		}
	}
	for _, name := range names {
		culture, err := loadCulture(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		for _, each := range culture.elements {
			chosen := each.Kind == pickedKind || each.Kind == recalledKind
			if !chosen || each.By == "" || each.Unit == "" {
				t.Errorf("%s: %s is not a picked or recalled number with "+
					"a unit and who chose it", name, each.Name)
			}
		}
	}
}

func TestACultureThatIsNotOneIsRefused(t *testing.T) {
	for _, name := range []string{commonCulture, "atlantis", ""} {
		if _, err := loadCulture(name); err == nil {
			t.Errorf("%q was loaded as a culture", name)
		}
	}
}

func TestAnElementWithoutItsProvenanceIsRefused(t *testing.T) {
	good := element{Name: "x", Says: "x", Scope: scopeRow, Draw: drawShare,
		Unit: "chance", Share: 0.5, Kind: pickedKind, By: "test"}
	if err := good.check(); err != nil {
		t.Fatal(err)
	}
	flawed := map[string]func(*element){
		"measured":      func(each *element) { each.Kind = "measured" },
		"no unit":       func(each *element) { each.Unit = "" },
		"no by":         func(each *element) { each.By = "" },
		"unknown scope": func(each *element) { each.Scope = "street" },
		"unknown draw":  func(each *element) { each.Draw = "dice" },
		"chance over 1": func(each *element) { each.Share = 1.5 },
		"backwards": func(each *element) {
			each.Draw, each.Least, each.Most = drawRange, 2, 1
		},
		"no options": func(each *element) { each.Draw = drawOptions },
		"bad colour": func(each *element) {
			each.Draw = drawOptions
			each.Options = []option{{Name: "a", Weight: 1,
				Colour: []float64{2, 0, 0}}}
		},
	}
	for name, spoil := range flawed {
		each := good
		spoil(&each)
		if err := each.check(); err == nil {
			t.Errorf("%s: the element was accepted", name)
		}
	}
}

func TestADrawIsTheSameForTheSameSeedAndPlace(t *testing.T) {
	first := drawerOf(t, "portofino", 7)
	second := drawerOf(t, "portofino", 7)
	for bay := range 10 {
		at := place{Bay: bay}
		if first.count("storeys", at) != second.count("storeys", at) {
			t.Errorf("bay %d has different storeys on the same seed", bay)
		}
	}
	if first.trouble != nil || second.trouble != nil {
		t.Fatal(first.trouble, second.trouble)
	}
}

func TestADrawAtACoarseScopeIgnoresTheFinerPlace(t *testing.T) {
	drawing := drawerOf(t, "portofino", 3)
	row := drawing.number("storey-height", place{Bay: 1, Storey: 4})
	if row != drawing.number("storey-height", place{Bay: 9}) {
		t.Error("a row's element changed from bay to bay")
	}
	bay := drawing.count("storeys", place{Bay: 2, Storey: 0})
	if bay != drawing.count("storeys", place{Bay: 2, Storey: 5}) {
		t.Error("a bay's element changed from storey to storey")
	}
	varied := map[int]bool{}
	for index := range 30 {
		varied[drawing.count("storeys", place{Bay: index})] = true
	}
	if len(varied) < 2 {
		t.Errorf("a bay's storeys never varied across bays: %v", varied)
	}
}

func TestADistrictIsSharedByTheSeedsThatAgreeButInTheLastDigits(t *testing.T) {
	names := func(seed Seed) string {
		drawing := drawerOf(t, "midwest", seed)
		subset := drawing.subset("palette", 1, place{})
		return fmt.Sprint(namesOf(subset))
	}
	if names(256) != names(257) || names(256) != names(511) {
		t.Error("seeds in one district drew different palettes")
	}
	seen := map[string]bool{}
	for district := range Seed(40) {
		seen[names(district*256)] = true
	}
	if len(seen) < 2 {
		t.Errorf("forty districts all drew the palette %v", seen)
	}
}

func TestAnElementTheCultureLacksIsRefusedWhenAsked(t *testing.T) {
	drawing := drawerOf(t, "midwest", 1)
	drawing.count("flying-buttresses", place{})
	if drawing.trouble == nil ||
		!strings.Contains(drawing.trouble.Error(), "flying-buttresses") {
		t.Errorf("an unknown element was drawn: %v", drawing.trouble)
	}
	if drawing.number("bay-width", place{}) != 0 {
		t.Error("a drawer that had failed went on drawing")
	}
}

func TestAnElementAskedForTheWrongWayIsRefused(t *testing.T) {
	drawing := drawerOf(t, "midwest", 1)
	drawing.count("bay-width", place{})
	if drawing.trouble == nil {
		t.Error("a range of metres was drawn as a count")
	}
	other := drawerOf(t, "midwest", 1)
	other.chance("bay-width", place{})
	if other.trouble == nil {
		t.Error("a range was drawn as a chance")
	}
}

func TestAWeightedPickSkipsWhatItIsToldToAndOnlyThat(t *testing.T) {
	drawing := drawerOf(t, "groningen", 5)
	options := drawing.options("palette")
	for bay := range 40 {
		if drawing.pick("palette", options, place{Bay: bay},
			"blue", "red", "grey").Name != "orange" {
			t.Fatal("a skipped option was picked")
		}
	}
	all := namesOf(options)
	if drawing.pick("palette", options, place{}, all...).Name == "" {
		t.Error("with every option skipped, nothing was picked")
	}
}
