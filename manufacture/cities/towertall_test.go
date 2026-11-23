package cities

import "testing"

// tallest is the storeys a tower has to have to be called a supertall one
// in these tests, in storeys.
const tallest = 150

func TestAtFullIrregularityTheGeneratorReachesSupertallTowersWithManySegments(
	t *testing.T) {
	found := 0
	for seed := range Seed(60) {
		tower := towerOf(t, seed, 1, Pins{Plan: planY, Shafts: 1,
			Setbacks: &yes, Taper: &no, Twist: &no, Spire: &yes})
		shaft := tower.Shafts[0]
		if shaft.Storeys >= tallest && len(shaft.Segments) >= 6 {
			found++
			t.Logf("seed %d: %d storeys in %d segments, %.0f m", seed,
				shaft.Storeys, len(shaft.Segments), tower.height(0))
		}
	}
	if found == 0 {
		t.Errorf("no Y of sixty seeds had %d storeys in six segments",
			tallest)
	}
}
