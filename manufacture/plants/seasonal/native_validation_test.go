package seasonal

import "testing"

var nativeKinds = []string{"pandanus-tectorius", "scaevola-taccada", "metrosideros-polymorpha", "barringtonia-asiatica", "quercus-ilex", "erica-arborea", "salix-alba"}

func TestANativeProfileWithoutItsLeavesIsRefused(t *testing.T) {
	for _, name := range nativeKinds {
		profile := plantProfiles[name]
		profile.LeafLength = 0
		if profile.valid() == nil {
			t.Errorf("%s with no leaf length was taken", name)
		}
	}
}

func TestATreeHeathWithMoreBladesThanACellHoldsIsRefused(t *testing.T) {
	profile := plantProfiles["erica-arborea"]
	profile.TwigCount *= 2
	if profile.valid() == nil {
		t.Error("a tree heath of twice its twigs, past a cell's blades, " +
			"was taken")
	}
}
