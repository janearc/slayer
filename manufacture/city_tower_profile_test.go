package manufacture

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

func checkedTreeBook(t *testing.T, raw []byte) *PlantProfiles {
	t.Helper()
	book, err := ReadPlantProfiles("trees.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	return book
}

func oneTreeBook(t *testing.T) *PlantProfiles {
	t.Helper()
	first := bytes.Index(plantProfilesRaw, []byte("\n- status:"))
	if first < 0 {
		t.Fatal("missing first tree")
	}
	next := bytes.Index(plantProfilesRaw[first+1:], []byte("\n- status:"))
	if next < 0 {
		t.Fatal("missing second tree")
	}
	raw := append([]byte("profiles:"), plantProfilesRaw[first:first+1+next]...)
	return checkedTreeBook(t, raw)
}

func TestCityTreeProfilesKeepGeometryAndSeededPrototypes(t *testing.T) {
	base := DefaultCityTrees()
	book := checkedTreeBook(t, plantProfilesRaw)
	withBook := base
	withBook.TreeProfiles = book
	if err := withBook.check(); err != nil {
		t.Fatal(err)
	}
	from := Sow(31)
	standard, err := treePrototypesAt(base, from, "quick")
	if err != nil {
		t.Fatal(err)
	}
	external, err := treePrototypesAt(withBook, from, "quick")
	if err != nil {
		t.Fatal(err)
	}
	if len(standard) != len(external) {
		t.Fatal("prototype count changed")
	}
	independent := checkedTreeBook(t, plantProfilesRaw)
	legacy, err := independent.NewPlant(base.Kind)
	if err != nil {
		t.Fatal(err)
	}
	for i := range standard {
		wantSeed := citySeed(from.number ^ uint64(i) ^ 0x54524545)
		if standard[i].seed != wantSeed || external[i].seed != wantSeed || standard[i].kind != external[i].kind {
			t.Fatalf("prototype %d changed species or seed", i)
		}
		old, err := MakeWithComplexity(legacy, Sow(wantSeed), "quick")
		if err != nil {
			t.Fatal(err)
		}
		if referenceDigest(t, standard[i].made) != referenceDigest(t, old) || referenceDigest(t, external[i].made) != referenceDigest(t, old) {
			t.Fatalf("prototype %d changed geometry", i)
		}
	}
	changed := bytes.Replace(plantProfilesRaw, []byte("leaf-width: 0.065"), []byte("leaf-width: 0.12"), 1)
	if bytes.Equal(changed, plantProfilesRaw) {
		t.Fatal("missing leaf fixture")
	}
	withBook.TreeProfiles = checkedTreeBook(t, changed)
	newTrees, err := treePrototypesAt(withBook, from, "quick")
	if err != nil {
		t.Fatal(err)
	}
	if referenceDigest(t, newTrees[0].made) == referenceDigest(t, standard[0].made) {
		t.Fatal("authored tree book did not change foliage")
	}
}

func TestTreeProfilesRefuseMissingSpeciesWithoutFallback(t *testing.T) {
	book := oneTreeBook(t) // The first profile is valley oak only.
	spec := DefaultCityTrees()
	spec.TreeProfiles = book
	if err := spec.check(); err != nil {
		t.Fatal(err)
	}
	spec.ParkEnrichment = true
	if err := spec.check(); err == nil {
		t.Fatal("city silently filled missing enrichment species")
	}
	if _, err := Tower(TowerSpec{TreeProfiles: book}); err == nil {
		t.Fatal("tower silently filled missing culture species")
	}
}

func TestCityFoliageSeparatesSpeciesWithSamePrototypeSeed(t *testing.T) {
	book := checkedTreeBook(t, plantProfilesRaw)
	var prototypes []cityTreePrototype
	for _, name := range []string{"valley-oak", "alder"} {
		row, err := book.NewPlant(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := MakeWithComplexity(row, Sow(5), "hull")
		if err != nil {
			t.Fatal(err)
		}
		prototypes = append(prototypes, cityTreePrototype{seed: 5, kind: name, made: made})
	}
	prepared, err := prepareCityFoliage(prototypes, looks{})
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared) != 2 || len(prepared[cityTreeKey{"valley-oak", 5}]) == 0 || len(prepared[cityTreeKey{"alder", 5}]) == 0 {
		t.Fatal("distinct species shared a prototype entry")
	}
}

func TestTowerTreeProfilesKeepSeededChoicesAndGeometry(t *testing.T) {
	book := checkedTreeBook(t, plantProfilesRaw)
	yes := true
	base := TowerSpec{Skin: "concrete", Planted: &yes, Storeys: 30}
	withBook := base
	withBook.TreeProfiles = book
	for _, seed := range []uint64{3, 11} {
		var choices [][]cities.Tree
		for _, spec := range []TowerSpec{base, withBook} {
			row := towerRow{spec: spec}
			tower, err := row.drawn(Sow(seed))
			if err != nil {
				t.Fatal(err)
			}
			design, err := tower.Design("quick", 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := design.Rendered(cities.Detail("quick")); err != nil {
				t.Fatal(err)
			}
			choices = append(choices, tower.Trees())
		}
		if len(choices[0]) == 0 || !reflect.DeepEqual(choices[0], choices[1]) {
			t.Fatal("tree choices or seeds changed")
		}
	}
	can, err := KindNamed("can")
	if err != nil {
		t.Fatal(err)
	}
	baseGood, err := Make(can, Sow(1))
	if err != nil {
		t.Fatal(err)
	}
	trees := []cities.Tree{{Kind: "valley-oak", Seed: 5, At: maths.V(2, 0, 0), Height: 3}}
	detail := func(cities.Tree) string { return "quick" }
	old, err := treed(baseGood, trees, nil, detail)
	if err != nil {
		t.Fatal(err)
	}
	newMade, err := treed(baseGood, trees, book, detail)
	if err != nil {
		t.Fatal(err)
	}
	if referenceDigest(t, old) != referenceDigest(t, newMade) {
		t.Fatal("profile book changed planted tower geometry")
	}
}
