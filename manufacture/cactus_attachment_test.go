package manufacture

import (
	"math"
	"testing"
)

// The whole base ring fits inside even the deepest pleat. Checking the minimum
// trunk radius is stronger than testing only a few radial directions.
func TestSaguaroArmCapsRemainBuriedInsideThePleatedTrunk(test *testing.T) {
	plant := defaultSaguaro()
	root := plant.value("root-radius-share")
	arm := plant.value("arm-radius-share")
	trough := 1 - 2*plant.value("pleat-depth")
	if !(math.Hypot(root, arm) < trough) {
		test.Fatal("an arm cap can protrude through a pleat trough", root, arm, trough)
	}
	if err := plant.recipe.Validate(); err != nil {
		test.Fatal(err)
	}
}
