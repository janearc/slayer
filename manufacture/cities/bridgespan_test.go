package cities

import "testing"

func TestAWideBridgeOfDepthTwoCanBeDrawnForManySeeds(t *testing.T) {
	for seed := range Seed(6) {
		bridge := bridgeOf(t, seed, 0.5, 2000, 2)
		if _, err := bridge.Design("quick", 0); err != nil {
			t.Errorf("seed %d: %v", seed, err)
		}
	}
}
