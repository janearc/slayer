package manufacture

import "testing"

// Face on is the normal-incidence case used to distinguish picked conductor
// analogues; none of these finishes asserts PRIMA's actual coating stack.
const primaStudyFaceOn = 1.0

func TestPrimaStudyFinishesUsePickedConductorAnalogues(t *testing.T) {
	study := spacecraftLooks("prima")
	for _, where := range []slot{spaceMirror, spaceSecondary, spacePanel,
		spaceStructure, spaceGold} {
		if study[where].Metal == nil || study[where].Gloss <= 0 {
			t.Fatalf("PRIMA visual analogue at slot %d has no conductor finish", where)
		}
	}

	for _, where := range []slot{spaceMirror, spaceSecondary, spacePanel} {
		reflectance := study[where].Reflects(primaStudyFaceOn)
		if reflectance.R < 0.85 || reflectance.B < 0.85 ||
			reflectance.R-reflectance.B > 0.03 {
			t.Errorf("silver visual analogue at slot %d reflects %+v", where,
				reflectance)
		}
	}
	truss := study[spaceStructure].Reflects(primaStudyFaceOn)
	if truss.R < 0.6 || truss.B < 0.6 ||
		truss.R-truss.B > 0.03 {
		t.Errorf("aluminium visual analogue reflects %+v", truss)
	}
	gold := study[spaceGold].Reflects(primaStudyFaceOn)
	if gold.R < 0.8 || gold.B > 0.5 || gold.R-gold.B < 0.4 {
		t.Errorf("gold visual analogue lost its warm spectral distinction: %+v",
			gold)
	}

	// Anti-test: a generic polished dielectric and the unmodified Hubble
	// profile must not accidentally inherit PRIMA's picked metal finishes.
	if spacecraftLooks("prima")[spaceFoil].Metal != nil {
		t.Error("PRIMA's unspecified shield coating became a claimed metal")
	}
	if spacecraftLooks("hubble")[spacePanel].Metal != nil {
		t.Error("Hubble's panel inherited PRIMA's visual metal analogue")
	}
}
