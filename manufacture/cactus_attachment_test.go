package manufacture

import (
	"math"
	"testing"
)

// The whole base ring fits inside even the deepest pleat. Checking the minimum
// trunk radius is stronger than testing only a few radial directions.
func TestSaguaroArmCapsRemainBuriedInsideThePleatedTrunk(test *testing.T) {
	root := cactusValue("root-radius-share")
	arm := cactusValue("arm-radius-share")
	trough := 1 - 2*cactusValue("pleat-depth")
	if !(math.Hypot(root, arm) < trough) {
		test.Fatal("an arm cap can protrude through a pleat trough", root, arm, trough)
	}
	if err := validCactusShape(); err != nil {
		test.Fatal(err)
	}
}
