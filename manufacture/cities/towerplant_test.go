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

func TestATowerThatIsNotPlantedHasNoChairs(t *testing.T) {
	tower := towerOf(t, 5, 0, plainPins(skinConcrete))
	drawn := drawnTower(t, tower, "quick",
		0)
	if found := len(named(drawn, "chair seat")); found != 0 {
		t.Errorf("%d chairs on a tower with no balconies", found)
	}
}
