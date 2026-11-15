package materials

import (
	"math"
	"testing"
)

// The yard's six substances are what they are made of: each formula
// sums to its elements' weights, and ByName finds each by name. The
// anti-test: they are nothing more, and none carries a density or
// optics, for none is cited.
func TestTheYardsSubstancesAreWhatTheyAreMadeOf(t *testing.T) {
	for _, each := range []struct {
		name  string
		parts map[string]float64
		mass  float64
	}{
		// plain carbon steel, iron with a little carbon
		{"steel", map[string]float64{"Fe": 0.998, "C": 0.002}, 55.757332},
		// ferric oxide, Fe2O3
		{"ochre", map[string]float64{"Fe": 2, "O": 3}, 159.687},
		// titanium dioxide, TiO2
		{"white", map[string]float64{"Ti": 1, "O": 2}, 79.865},
		// cellulose, C6H10O5
		{"timber", map[string]float64{"C": 6, "H": 10, "O": 5}, 162.141},
		// polyisoprene, C5H8
		{"tyre", map[string]float64{"C": 5, "H": 8}, 68.119},
		// silica, SiO2
		{"window", map[string]float64{"Si": 1, "O": 2}, 60.083},
	} {
		kind, err := ByName(each.name)
		if err != nil {
			t.Fatalf("%s: %v", each.name, err)
		}
		if kind.Name() != each.name {
			t.Errorf("%s answers to the name %s", each.name, kind.Name())
		}
		if len(kind.made.parts) != len(each.parts) {
			t.Errorf("%s is made of %d elements, want %d", each.name,
				len(kind.made.parts), len(each.parts))
		}
		for symbol, count := range each.parts {
			if kind.made.parts[symbol] != count {
				t.Errorf("%s counts %s %v times, want %v", each.name,
					symbol, kind.made.parts[symbol], count)
			}
		}
		if mass := kind.made.Mass(); math.Abs(mass-each.mass) > 1e-9 {
			t.Errorf("%s weighs %v, want %v", each.name, mass, each.mass)
		}
		if kind.Density() != 0 {
			t.Errorf("%s has a density of %v with none cited", each.name,
				kind.Density())
		}
		for _, band := range []Band{Red, Green, Blue} {
			if kind.Optics().Has(band) {
				t.Errorf("%s has %s optics with none cited", each.name, band)
			}
		}
	}
}
