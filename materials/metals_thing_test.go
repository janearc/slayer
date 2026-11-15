// These tests are in package materials_test, not materials, because they
// read scene/thing's metals to compare against, and an in-package test
// importing thing would cycle once thing imports materials.
package materials_test

import (
	"testing"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/scene/thing"
)

// TestTheMetalsMatchWhatThingHas compares every committed metal's stored
// n and k, exactly, with what scene/thing/metals.go returns for the same
// metal: the two must carry the same numbers, band for band, until thing
// reads them from here. The anti-test: no two metals carry the same
// numbers, so a lookup that gave one metal's page for every name would
// not pass.
func TestTheMetalsMatchWhatThingHas(t *testing.T) {
	seen := map[[6]float64]string{}
	for _, each := range []struct {
		name  string
		metal *thing.Metal
	}{
		{"gold", thing.Gold()},
		{"silver", thing.Silver()},
		{"aluminium", thing.Aluminium()},
		{"beryllium", thing.Beryllium()},
		{"titanium", thing.Titanium()},
	} {
		made, err := materials.ByName(each.name)
		if err != nil {
			t.Fatalf("%s: %v", each.name, err)
		}
		for band, colour := range map[materials.Band]int{
			materials.Red: 0, materials.Green: 1, materials.Blue: 2,
		} {
			read := made.Optics().At(band)
			if read.N() != each.metal.N[colour] {
				t.Errorf("%s %s: n %v, thing has %v",
					each.name, band, read.N(), each.metal.N[colour])
			}
			if read.K() != each.metal.K[colour] {
				t.Errorf("%s %s: k %v, thing has %v",
					each.name, band, read.K(), each.metal.K[colour])
			}
		}
		numbers := [6]float64{each.metal.N[0], each.metal.N[1], each.metal.N[2],
			each.metal.K[0], each.metal.K[1], each.metal.K[2]}
		if other, twice := seen[numbers]; twice {
			t.Errorf("%s and %s carry the same numbers", each.name, other)
		}
		seen[numbers] = each.name
	}
}
