package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func TestVacantLotIndexRecipeReachesClusteredCityConstruction(t *testing.T) {
	source, err := os.ReadFile("../doc/city-clustered-green-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.Replace(string(source), "  set:\n", "  recipes:\n    green district:\n      kind: city-clustered-green-10000\n      vacant-lot-index: off\n  set:\n", 1)
	raw = strings.Replace(raw, "        kind: city-clustered-green-10000", "        recipe: green district", 1)
	scene, err := set.Read("green-city-index.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(scene.Things()) != 1 || scene.Things()[0].Kind() != "city-clustered-green-10000" || scene.Things()[0].Recipe() != "green district" {
		t.Fatalf("resolved recipe provenance: %+v", scene.Things())
	}
	recipe, found := scene.Recipe("green district")
	if !found || recipe.VacantLotIndex() != manufacture.CityVacantLotIndexOff {
		t.Fatalf("recipe config: %+v, found %t", recipe, found)
	}
	if _, err := recipe.Construction(); err != nil {
		t.Fatalf("configured construction: %v", err)
	}
}

func TestVacantLotIndexRecipeRefusesUnsupportedContexts(t *testing.T) {
	source, err := os.ReadFile("../doc/city-clustered-green-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	makeSource := func(kind, mode string) string {
		raw := strings.Replace(string(source), "  set:\n", "  recipes:\n    configured:\n      kind: "+kind+"\n      vacant-lot-index: "+mode+"\n  set:\n", 1)
		raw = strings.Replace(raw, "        kind: city-clustered-green-10000", "        recipe: configured", 1)
		return raw
	}
	wrongContext := strings.Replace(makeSource("city-clustered-green-10000", "off"),
		"    things:\n      city:\n        recipe: configured\n        seed: 31\n        complexity: quick\n        east: 0\n        north: 0\n        height: {value: 0, unit: m}",
		"    stands:\n      city:\n        recipe: configured\n        count: 1\n        seed: 31", 1)
	for name, raw := range map[string]string{
		"unknown mode":   makeSource("city-clustered-green-10000", "sideways"),
		"grid city":      makeSource("city-grid-10000", "off"),
		"tree-free city": makeSource("city-clustered-10000", "off"),
		"non-city":       makeSource("boulder", "off"),
		"stand":          wrongContext,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := set.Read("unsupported-vacant-index.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted unsupported vacant-lot-index context")
			}
		})
	}
	gridDirect, err := os.ReadFile("../doc/city-grid-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	direct := strings.Replace(string(gridDirect), "        kind: city-grid-10000", "        kind: city-grid-10000\n        vacant-lot-index: off", 1)
	if _, err := set.Read("direct-vacant-index.yaml", []byte(direct)); err == nil {
		t.Fatal("accepted a direct thing option outside recipe schema")
	}
}
