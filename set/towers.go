package set

import (
	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// towerMostShafts is the most shafts a set may pin a tower to. The
// tower's own data holds the true limit and refuses a pin past it, when
// the recipe is checked.
const towerMostShafts = 64

// towerMostStoreys is the most storeys, or segments, a set may pin a
// shaft to: a tower of a thousand storeys is past any a set draws, and
// the tower's own data holds the rest of the limit.
const towerMostStoreys = 1000

// irregularity reads how far a tower recipe may stray from a plain block:
// a number from nought to one, or the word drawn, which makes each tower
// of the recipe draw its own from the skyline's distribution.
func (reading reader) irregularity(spec *manufacture.TowerSpec,
	bridge *manufacture.BridgeSpec) step {
	return func(key, value *yaml.Node) error {
		if value.Value == "drawn" {
			spec.Drawn, bridge.Drawn = true, true
			return nil
		}
		err := reading.number(&spec.Irregularity, 0, 1)(key, value)
		bridge.Irregularity = spec.Irregularity
		return err
	}
}

// bridgeMostSpan is the longest span a set may give a bridge, in metres:
// a thousand kilometres, which is past any limit a bridge has.
const bridgeMostSpan = 1e6

// bridgeMostDepth is the deepest a bridge's lots may nest, as a set may
// say: past it the number of bridges grows past any use.
const bridgeMostDepth = 6

// bridgeDepth reads how many bridges deep a deck's lots may nest.
func (reading reader) bridgeDepth(spec *manufacture.BridgeSpec) step {
	var depth float64
	return func(key, value *yaml.Node) error {
		err := reading.number(&depth, 0, bridgeMostDepth)(key, value)
		spec.Depth, spec.DepthSaid = int(depth), true
		return err
	}
}

// yesNo reads a pin that is true or false, and leaves it unpinned if
// the set does not say it.
func (reading reader) yesNo(target **bool) step {
	return func(key, value *yaml.Node) error {
		answer := value.Value == "true"
		if value.Value != "true" && value.Value != "false" {
			return reading.refuse(value, "%s is true or false, not %q",
				key.Value, value.Value)
		}
		*target = &answer
		return nil
	}
}

// towerPins reads the features a set pins for a tower recipe.
func (reading reader) towerPins(spec *manufacture.TowerSpec) step {
	var shafts, storeys, segments float64
	return reading.block(map[string]step{
		"plan": reading.word(&spec.Plan, manufacture.TowerPlans()...),
		"shafts": func(key, value *yaml.Node) error {
			err := reading.number(&shafts, 1, towerMostShafts)(key, value)
			spec.Shafts = int(shafts)
			return err
		},
		"storeys": func(key, value *yaml.Node) error {
			err := reading.number(&storeys, 1, towerMostStoreys)(key, value)
			spec.Storeys = int(storeys)
			return err
		},
		"segments": func(key, value *yaml.Node) error {
			err := reading.number(&segments, 1, towerMostStoreys)(key, value)
			spec.Segments = int(segments)
			return err
		},
		"skin":  reading.word(&spec.Skin, manufacture.TowerSkins()...),
		"use":   reading.word(&spec.Use, manufacture.TowerUses()...),
		"style": reading.word(&spec.Style, "punched", "ribbon"),
		"colour": func(_, value *yaml.Node) error {
			spec.Colour = value.Value
			return nil
		},
		"cyber":     reading.yesNo(&spec.Cyber),
		"signs":     reading.yesNo(&spec.Signs),
		"retail":    reading.yesNo(&spec.Retail),
		"planted":   reading.yesNo(&spec.Planted),
		"spiral":    reading.yesNo(&spec.Spiral),
		"width":     reading.quantity(&spec.Width, "m", 1, bridgeMostSpan),
		"taper":     reading.yesNo(&spec.Taper),
		"twist":     reading.yesNo(&spec.Twist),
		"setbacks":  reading.yesNo(&spec.Setbacks),
		"gargoyles": reading.yesNo(&spec.Gargoyles),
		"spire":     reading.yesNo(&spec.Spire),
		"lit-crown": reading.yesNo(&spec.LitCrown),
		"stack":     reading.yesNo(&spec.Stack),
		"nest":      reading.yesNo(&spec.Nest),
		"room":      reading.quantity(&spec.Room, "m", 1, bridgeMostSpan),
		"span":      reading.word(&spec.Span, manufacture.TowerSpans()...),
	})
}
