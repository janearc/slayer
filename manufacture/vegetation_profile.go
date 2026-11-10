package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// valid refuses malformed private profile edits before scaffold/prototype
// loops.
func (profile plantProfile) valid() error {
	if err := profile.finite(); err != nil {
		return err
	}
	if profile.Form != "twig" && profile.Form != "rosette" {
		return fmt.Errorf("manufacture: unknown plant form %q", profile.Form)
	}
	if !positive(profile.Scale.Minimum) || profile.Scale.Maximum < profile.Scale.Minimum ||
		!positive(profile.Scale.Maximum) || !positive(profile.Slenderness) ||
		!positive(profile.Trunk) || !positive(profile.Reach) || !positive(profile.RadiusPower) {
		return fmt.Errorf("manufacture: invalid plant dimensions for %s", profile.Kind)
	}
	if !positive(profile.Contraction) || profile.Contraction >= spaceOne ||
		!positive(profile.Taper) || profile.Taper >= spaceOne ||
		profile.Upward < spaceZero || profile.Upward > spaceOne ||
		profile.Fork < spaceZero || profile.Fork >= plantValue("maximum-fork-deg") {
		return fmt.Errorf("manufacture: invalid plant branching for %s", profile.Kind)
	}
	if profile.Orders < spaceZero || profile.Orders > int(plantValue("maximum-orders")) ||
		profile.Children < spaceOne || profile.Children > int(plantValue("maximum-children")) ||
		profile.Stems < spaceOne || profile.Stems > int(plantValue("maximum-stems")) {
		return fmt.Errorf("manufacture: excessive plant recursion for %s", profile.Kind)
	}
	return profile.validLeaves()
}

// finite checks every scalar, including optional controls on other forms.
func (profile plantProfile) finite() error {
	values := []float64{profile.Scale.Minimum, profile.Scale.Maximum,
		profile.Slenderness, profile.Trunk, profile.Reach, profile.Contraction,
		profile.Taper, profile.RadiusPower, profile.Fork, profile.Twist,
		profile.Upward, profile.LeaderGain, profile.StemSpread,
		profile.LeafLength, profile.LeafWidth, profile.LeafThickness,
		profile.LobeDepth, profile.CellLength, profile.CellSpacing,
		profile.TwigLength, profile.TwigRadius, profile.RosetteAngle}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, spaceZero) {
			return fmt.Errorf("manufacture: non-finite plant profile %s", profile.Kind)
		}
	}
	for _, colour := range []maths.Light{profile.Bark, profile.Green, profile.Autumn, profile.Dry} {
		for _, channel := range []float64{colour.R, colour.G, colour.B} {
			if math.IsNaN(channel) || channel < spaceZero || channel > spaceOne {
				return fmt.Errorf("manufacture: invalid plant colour for %s", profile.Kind)
			}
		}
	}
	if profile.StemSpread < spaceZero || (profile.Leader && !positive(profile.LeaderGain)) ||
		profile.RosetteAngle < spaceZero || profile.RosetteAngle > math.Pi/maths.Degree {
		return fmt.Errorf("manufacture: invalid plant pose for %s", profile.Kind)
	}
	return nil
}

// validLeaves keeps microgeometry finite and budgets its explicit leaf count.
func (profile plantProfile) validLeaves() error {
	if !positive(profile.LeafLength) || !positive(profile.LeafWidth) ||
		!positive(profile.LeafThickness) || !positive(profile.CellLength) ||
		!positive(profile.CellSpacing) || !positive(profile.TwigRadius) {
		return fmt.Errorf("manufacture: invalid physical foliage for %s", profile.Kind)
	}
	switch profile.LeafShape {
	case "oval", "lobed", "lance", "deltoid":
	default:
		return fmt.Errorf("manufacture: unknown leaf outline %q", profile.LeafShape)
	}
	if profile.LobeDepth < spaceZero || profile.LobeDepth >= spaceHalf ||
		profile.Lobes < spaceZero {
		return fmt.Errorf("manufacture: invalid leaf lobes for %s", profile.Kind)
	}
	if profile.Form == "rosette" {
		if profile.RosetteLeaves < spaceOne ||
			profile.RosetteLeaves > int(plantValue("maximum-cell-blades"))/intPairCount {
			return fmt.Errorf("manufacture: excessive rosette for %s", profile.Kind)
		}
		return nil
	}
	if profile.TwigCount < spaceOne || profile.TwigCount > int(plantValue("maximum-cell-twigs")) ||
		profile.LeafPairs < spaceOne || profile.LeafPairs > int(plantValue("maximum-twig-pairs")) ||
		profile.TwigCount*profile.LeafPairs*intPairCount > int(plantValue("maximum-cell-blades")) ||
		!positive(profile.TwigLength) {
		return fmt.Errorf("manufacture: excessive foliage cell for %s", profile.Kind)
	}
	return nil
}

// intPairCount is defined as the two leaves in a paired attachment.
const intPairCount = 2
