package cities

import (
	"math"
	"testing"
)

func TestEveryBalconyKeepsAClearStretchAndSomeHaveAChair(t *testing.T) {
	pins := plainPins(skinConcrete)
	pins.Planted = &yes
	tower := towerOf(t, 5, 0, pins)
	drawn := drawnTower(t, tower, "quick", 0)
	clear := towerNumber(t, "planted-clear")
	if len(named(drawn, "chair seat")) == 0 {
		t.Errorf("no balcony has a chair")
	}
	for _, name := range []string{"hedge"} {
		for _, one := range named(drawn, name) {
			low, high := math.Inf(1), math.Inf(-1)
			for _, face := range one.Mesh.Triangles {
				for _, corner := range face {
					at := one.Place.Out(corner)
					low, high = math.Min(low, at.X), math.Max(high, at.X)
				}
			}
			if low < clear-1e-6 && high > -clear+1e-6 &&
				math.Abs(one.Place.Orientation.X.X) > 0.99 {
				t.Fatalf("a %s runs from %v to %v across the clear stretch",
					name, low, high)
			}
		}
	}
}

// A balcony with no room under it hangs no plant from its edge, so every
// planted tower draws, and every hanging plant has a length. Found in the
// city by the bay on 2026-11-23, where a star tower's balcony at its foot
// drew a hanging plant 0 m long and the whole set was refused.
func TestEveryHangingPlantHasRoomToHang(t *testing.T) {
	const seeds = 60
	for seed := Seed(1); seed <= seeds; seed++ {
		pins := Pins{Plan: planStar, Skin: skinStainless, Shafts: 1,
			Planted: &yes}
		tower, err := NewTower(seed, Irregularity{Value: 0.28}, pins)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		design, err := tower.Design("quick", 0)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		drawn, err := design.Rendered("quick")
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		for _, plant := range named(drawn, "hanging plant") {
			low, high := boundsOf([]Rendered{plant})
			if !(high.Y-low.Y > 0) {
				t.Fatalf("seed %d: a hanging plant %v m long", seed,
					high.Y-low.Y)
			}
		}
	}
}

func TestATowerThatIsNotPlantedHasNoChairs(t *testing.T) {
	tower := towerOf(t, 5, 0, plainPins(skinConcrete))
	drawn := drawnTower(t, tower, "quick",
		0)
	if found := len(named(drawn, "chair seat")); found != 0 {
		t.Errorf("%d chairs on a tower with no balconies", found)
	}
}
