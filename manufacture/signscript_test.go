package manufacture

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/janearc/slayer/sources"
)

// simplified is the script the tests ask for, and herTrades the trades a
// set asks every building to carry.
const simplified = "simplified-chinese"

var herTrades = []string{"womens-shelter", "dog-boutique"}

// signedTower is the kind tower with signs pinned on.
func signedTower(t *testing.T) Kind {
	t.Helper()
	on := true
	row, err := Tower(TowerSpec{Signs: &on})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestAScriptIsAskedOfTheKindsThatDrawSignsAndNoOther(t *testing.T) {
	for _, name := range append([]string{"tower"}, signNames...) {
		row, err := kindNamed(name)
		if name == "tower" {
			row, err = signedTower(t), nil
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := WithScript(row, simplified, nil); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, name := range []string{"frontage", "frontage-ramen", "street"} {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := WithScript(row, simplified, nil); err == nil {
			t.Errorf("%s was given a script, and draws no words", name)
		}
		if _, err := WithFaces(row, map[string]string{"a": "b"}); err == nil {
			t.Errorf("%s was given faces, and draws no signs", name)
		}
	}
	plain, err := Tower(TowerSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithScript(plain, simplified, nil); err == nil {
		t.Error("a tower that pins no signs was given a script")
	}
}

func TestAScriptOrATradeTheCultureDoesNotHaveIsRefused(t *testing.T) {
	row := signedTower(t)
	if _, err := WithScript(row, "klingon", nil); err == nil {
		t.Error("a script there is not was kept")
	}
	if _, err := WithScript(row, simplified, []string{"bank"}); err == nil {
		t.Error("a trade there is not was kept")
	}
	if _, err := WithScript(row, simplified, herTrades); err != nil {
		t.Errorf("her trades: %v", err)
	}
	screen, err := kindNamed("sign-screen")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithScript(screen, simplified,
		herTrades[:1]); err == nil {
		t.Error("a screen was asked to carry a trade")
	}
}

func TestTheFacesOfAScriptAreSourcesThatSourcesYamlPins(t *testing.T) {
	if got, err := SignFaces("latin"); err != nil || len(got) != 0 {
		t.Errorf("the Latin script has faces %v: %v", got, err)
	}
	keys, err := SignFaces(simplified)
	if err != nil || len(keys) == 0 {
		t.Fatalf("the simplified script has faces %v: %v", keys, err)
	}
	list, err := sources.Read(filepath.Join("..", "sources.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		source, found := list.Find(key)
		if !found || !source.Pinned() {
			t.Errorf("sources.yaml has no pinned source %s", key)
		}
	}
	if !slices.Contains(SignScripts(), simplified) {
		t.Errorf("the scripts a set may ask for are %v", SignScripts())
	}
}

func TestATowerOfTheLatinScriptIsTheTowerOfNoScript(t *testing.T) {
	plain := signedTower(t)
	latin, err := WithScript(plain, "latin", nil)
	if err != nil {
		t.Fatal(err)
	}
	for number := uint64(1); number <= signedSeeds; number++ {
		one, err := MakeWithComplexity(plain, sow(number), "quick")
		if err != nil {
			t.Fatal(err)
		}
		other, err := MakeWithComplexity(latin, sow(number), "quick")
		if err != nil {
			t.Fatal(err)
		}
		if len(one.slots()) != len(other.slots()) ||
			len(one.lights) != len(other.lights) ||
			!reflect.DeepEqual(one.Bounds(), other.Bounds()) {
			t.Errorf("seed %d: the Latin script made another tower", number)
		}
	}
}
