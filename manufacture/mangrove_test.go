package manufacture

import "testing"

var mangroveKinds = []string{"kandelia-obovata", "aegiceras-corniculatum", "avicennia-marina", "bruguiera-gymnorhiza", "sonneratia-apetala"}

func TestMangroveKindsAreGrownFromTheirProfilesAtEveryDetail(t *testing.T) {
	for _, name := range mangroveKinds {
		row, err := testKind(name)
		if err != nil {
			t.Fatalf("%s is not a kind: %v", name, err)
		}
		for _, detail := range Complexities() {
			made, err := MakeWithComplexity(row, Sow(7), detail)
			if err != nil {
				t.Fatalf("%s at %s: %v", name, detail, err)
			}
			if made.extent == nil {
				t.Fatalf("%s at %s has no extent", name, detail)
			}
		}
	}
}

func TestMangroveEveryDetailShowsTheSameRoots(t *testing.T) {
	plan, err := plantProfiles["sonneratia-apetala"].Plan(sow(9))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[int]bool{}
	for _, detail := range Complexities() {
		bark, err := plantBark(plan, detail)
		if err != nil {
			t.Fatal(err)
		}
		counts[len(bark)] = true
	}
	if len(counts) != 1 {
		t.Fatalf("details differ in how many woody parts they carry: %v", counts)
	}
}
