package manufacture

import (
	"strings"
	"testing"
)

// nativeKinds are the profiles of the island and coast sets' own plants,
// in the profile data.
var nativeKinds = []string{"pandanus-tectorius", "scaevola-taccada",
	"metrosideros-polymorpha", "barringtonia-asiatica", "quercus-ilex",
	"erica-arborea", "salix-alba"}

// nativeSeeds is how many different seeds a test grows of each kind.
const nativeSeeds = 6

func TestNativeKindsAreGrownAtEveryDetailFromManySeeds(t *testing.T) {
	for _, name := range nativeKinds {
		row, err := testKind(name)
		if err != nil {
			t.Fatalf("%s is not a kind: %v", name, err)
		}
		for number := range uint64(nativeSeeds) {
			for _, detail := range Complexities() {
				made, err := MakeWithComplexity(row, Sow(number+1), detail)
				if err != nil {
					t.Fatalf("%s from seed %d at %s: %v", name, number+1,
						detail, err)
				}
				if made.extent == nil {
					t.Fatalf("%s at %s has no extent", name, detail)
				}
			}
		}
	}
}

func TestEveryNativeProfileSaysWhereItsNumbersWereRead(t *testing.T) {
	for _, name := range nativeKinds {
		profile := plantProfiles[name]
		if !strings.Contains(profile.By, "read from the source") {
			t.Errorf("%s says only %q of where its numbers come from", name,
				profile.By)
		}
	}
}

func TestTheHalaStandsOnItsAerialRootsAndOthersOnNone(t *testing.T) {
	if plantProfiles["pandanus-tectorius"].Roots == nil {
		t.Error("the hala shows no aerial roots, which its source names")
	}
	for _, name := range nativeKinds[1:] {
		if plantProfiles[name].Roots != nil {
			t.Errorf("%s shows roots no source named", name)
		}
	}
}
