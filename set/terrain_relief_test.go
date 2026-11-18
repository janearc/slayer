package set_test

import "testing"

// placedWith is a terrain placed on a sphere whose detail says more.
func placedWith(more string) string {
	return "    lat: -7.5\n    lon: 303\n    detail: {seed: 13" + more + "}\n"
}

// A detail that asks a relief says it: its slope and its longest edge.
func TestTerrainDetailSaysItsRelief(t *testing.T) {
	made, err := legacyLaser(t, byMetres,
		placedWith(", relief: {slope: 0.05, longest: 60}"))
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	slope, longest, asked := ground.Relief()
	if !asked || slope != 0.05 || longest != 60 { // picked: as said
		t.Errorf("relief %v over %v, asked %v", slope, longest, asked)
	}
}

// Anti: a detail with no relief asks none.
func TestTerrainDetailWithoutReliefAsksNone(t *testing.T) {
	made, err := legacyLaser(t, byMetres, placedBy)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	if slope, longest, asked := ground.Relief(); asked || slope != 0 ||
		longest != 0 {
		t.Errorf("relief %v over %v, asked %v", slope, longest, asked)
	}
}

// A relief with half its numbers, or numbers out of range, is refused.
func TestTerrainReliefRefusals(t *testing.T) {
	for name, more := range map[string]string{
		"no longest":   ", relief: {slope: 0.05}",
		"no slope":     ", relief: {longest: 60}",
		"downhill":     ", relief: {slope: -0.1, longest: 60}",
		"cliff":        ", relief: {slope: 11, longest: 60}",
		"no length":    ", relief: {slope: 0.05, longest: 0}",
		"another word": ", relief: {slope: 0.05, longest: 60, grain: 1}",
	} {
		_, err := legacyLaser(t, byMetres, placedWith(more))
		mustHold(t, name, err, "")
	}
}
