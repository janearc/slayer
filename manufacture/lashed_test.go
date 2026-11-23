package manufacture

import (
	"strings"
	"testing"
)

// signKind is a kind that is small and quick to make, to stand in a sign's
// slot.
const signKind = "human-scale-2m"

// lashedStreetOf is the lashed street of a spec, or the test's end.
func lashedStreetOf(t *testing.T, spec LashingsSpec) Kind {
	t.Helper()
	row, err := LashedStreet(spec)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestALashedStreetIsOneGoodOfCablesAndLanterns(t *testing.T) {
	made, err := MakeWithComplexity(lashedStreetOf(t, LashingsSpec{Density: 1,
		DensitySaid: true}), sow(1), "hull")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	steel := false
	for _, look := range made.dressed {
		names[look.Name] = true
		// a look is named for its maker, its role and its colour, so the
		// steel of a cable is named for the colour it was drawn in
		steel = steel || strings.HasPrefix(look.Name, "cities lashing steel ")
	}
	if !steel {
		t.Errorf("no steel of a cable among %v", names)
	}
}

func TestTheKindIsKnownByItsName(t *testing.T) {
	row, err := KindNamed(LashedStreetKind)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MakeWithComplexity(row, sow(1), "hull"); err != nil {
		t.Error(err)
	}
}

func TestASignKindStandsInTheSlots(t *testing.T) {
	plain := lashedStreetOf(t, LashingsSpec{Density: 1, DensitySaid: true})
	signed := lashedStreetOf(t, LashingsSpec{Density: 1, DensitySaid: true,
		Sign: signKind})
	one, err := MakeWithComplexity(plain, sow(1), "hull")
	if err != nil {
		t.Fatal(err)
	}
	other, err := MakeWithComplexity(signed, sow(1), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if len(one.parts) == len(other.parts) &&
		len(one.native) == len(other.native) &&
		len(one.dressed) == len(other.dressed) {
		t.Error("a sign kind in the slots changed nothing")
	}
}

// Anti: a sign that is a lashed street, a cluster, a superblock or no kind
// at all is refused.
func TestASignThatIsNoPartIsRefused(t *testing.T) {
	for _, sign := range []string{LashedStreetKind, ClusterKind,
		SuperblockKind, "no such kind"} {
		if _, err := LashedStreet(LashingsSpec{Sign: sign}); err == nil {
			t.Errorf("%q was taken as a sign", sign)
		}
	}
}

// Anti: a density that is said and is not from nought to one is refused,
// and one that is not said is taken from the culture, whatever it holds.
func TestADensityOutOfRangeIsRefusedWhenSaid(t *testing.T) {
	for _, density := range []float64{-0.5, 1.5} {
		if _, err := LashedStreet(LashingsSpec{Density: density,
			DensitySaid: true}); err == nil {
			t.Errorf("a density of %v was taken", density)
		}
	}
	if _, err := LashedStreet(LashingsSpec{Density: 9}); err != nil {
		t.Errorf("a density not said was read all the same: %v", err)
	}
}

// Anti: a street of a width below nought is refused.
func TestAStreetBelowNoughtIsRefused(t *testing.T) {
	if _, err := LashedStreet(LashingsSpec{Street: -1}); err == nil {
		t.Error("a street of a width below nought was taken")
	}
}
