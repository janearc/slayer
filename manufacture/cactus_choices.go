package manufacture

import (
	"fmt"
	"math"
)

// validCactusChoices bounds private data edits before integer loops and sweeps.
func validCactusChoices() error {
	for _, name := range []string{"height", "radius", "arm-run", "bend", "attachment"} {
		low, high := cactusValue(name+"-minimum"), cactusValue(name+"-maximum")
		if !positive(low) || !positive(high) || high < low ||
			high > plantValue("maximum-cactus-metres") {
			return fmt.Errorf("manufacture: invalid cactus range %s", name)
		}
	}
	for _, name := range []string{"ribs", "maximum-spines", "arms-minimum", "arms-maximum"} {
		value := cactusValue(name)
		if !cactusBetween(value, spaceZero, plantValue("maximum-cactus-count")) ||
			value != math.Trunc(value) {
			return fmt.Errorf("manufacture: invalid cactus count %s", name)
		}
	}
	if err := validCactusShape(); err != nil {
		return err
	}
	return validCactusDetail()
}

// cactusBetween explicitly refuses NaN as well as infinities outside a range.
func cactusBetween(value, low, high float64) bool { return value >= low && value <= high }

// validCactusShape prevents an arm's inner bend from folding over itself.
func validCactusShape() error {
	if cactusValue("arms-minimum") > cactusValue("arms-maximum") ||
		cactusValue("arms-maximum") > plantValue("maximum-stems") ||
		cactusValue("ribs") < leastAround || cactusValue("ribs") > mostAround ||
		cactusValue("maximum-spines") < spaceOne {
		return fmt.Errorf("manufacture: excessive cactus construction")
	}
	armRadius := cactusValue("radius-maximum") * cactusValue("arm-radius-share")
	rootEnvelope := math.Hypot(cactusValue("root-radius-share"), cactusValue("arm-radius-share"))
	innerSkin := spaceOne - 2*cactusValue("pleat-depth")
	if !positive(armRadius) || armRadius >= cactusValue("bend-minimum") ||
		!(rootEnvelope < innerSkin) ||
		cactusValue("attachment-maximum") >= spaceOne ||
		!cactusBetween(cactusValue("pleat-depth"), spaceZero, plantValue("maximum-pleat-depth")) ||
		!cactusBetween(cactusValue("root-radius-share"), spaceZero, spaceOne) ||
		!cactusBetween(cactusValue("arm-turn-variation"), spaceZero, spaceHalf) {
		return fmt.Errorf("manufacture: invalid cactus arm or pleat")
	}
	return validCactusSpines()
}

// validCactusSpines checks fixed physical microgeometry and its painted
// colours.
func validCactusSpines() error {
	for _, name := range []string{"spine-length", "spine-radius", "spine-spacing"} {
		if !positive(cactusValue(name)) || cactusValue(name) > plantValue("maximum-cactus-metres") {
			return fmt.Errorf("manufacture: invalid cactus %s", name)
		}
	}
	if !cactusBetween(cactusValue("spine-root-share"), spaceZero, spaceOne) {
		return fmt.Errorf("manufacture: invalid cactus spine root")
	}
	for _, name := range []string{"green-r", "green-g", "green-b", "spine-r", "spine-g", "spine-b"} {
		if !cactusBetween(cactusValue(name), spaceZero, spaceOne) {
			return fmt.Errorf("manufacture: invalid cactus colour %s", name)
		}
	}
	return nil
}

// validCactusDetail bounds all path and ring subdivisions before allocation.
func validCactusDetail() error {
	for _, detail := range Complexities() {
		for _, field := range []string{"curve-", "cap-", "sides-"} {
			value := cactusValue(field + detail)
			if !cactusBetween(value, spaceOne, mostAround) || value != math.Trunc(value) {
				return fmt.Errorf("manufacture: invalid cactus detail %s%s", field, detail)
			}
		}
		if cactusValue("sides-"+detail) < leastAround {
			return fmt.Errorf("manufacture: cactus ring cannot close")
		}
	}
	for _, detail := range []string{"quick", "demo"} {
		value := cactusValue("spine-sides-" + detail)
		if !cactusBetween(value, leastAround, mostAround) || value != math.Trunc(value) {
			return fmt.Errorf("manufacture: invalid cactus spine detail %s", detail)
		}
	}
	return nil
}
