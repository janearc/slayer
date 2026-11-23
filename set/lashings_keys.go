package set

import (
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	yaml "gopkg.in/yaml.v3"
)

// lashingsMostStreet is the widest street a set may give a lashed street,
// in metres: past it a cable is no longer a thing a street holds up.
const lashingsMostStreet = 100

// lashingsMostLength is the longest a set may make a lashed street, in
// metres: past it the lanterns of a thick street number past any use.
const lashingsMostLength = 500

// lashingsKeys are the keys only a lashed street takes.
var lashingsKeys = []string{"density", "street", "length", "sign", "style"}

// lashingDensity reads how thickly a street is hung, from nought to one,
// and notes that the set said it, so that nought is not read as unsaid.
func (reading reader) lashingDensity(spec *manufacture.LashingsSpec) step {
	return func(key, value *yaml.Node) error {
		err := reading.number(&spec.Density, 0, 1)(key, value)
		spec.DensitySaid = err == nil
		return err
	}
}

// lashingKeys are the steps of the keys a lashed street takes.
func (reading reader) lashingKeys(spec *manufacture.LashingsSpec) map[string]step {
	return map[string]step{
		"density": reading.lashingDensity(spec),
		"street":  reading.quantity(&spec.Street, "m", 1, lashingsMostStreet),
		"length":  reading.quantity(&spec.Length, "m", 1, lashingsMostLength),
		"sign":    reading.word(&spec.Sign, manufacture.KindNames()...),
		"style":   reading.word(&spec.Style, cities.Styles()...),
	}
}

// lashingsRefused is the refusal of a recipe that is not a lashed street
// and says a key only one takes, or the refusal of a lashed street the
// manufactory cannot keep, or nothing.
func (reading reader) lashingsRefused(recipe Recipe, name, block *yaml.Node) error {
	if recipe.kind != manufacture.LashedStreetKind {
		for _, key := range lashingsKeys {
			// a street has a length too, and says it by the same word
			if key == "length" && recipe.kind == "street" {
				continue
			}
			if says(block, key) {
				return reading.refuse(block, "recipe %s is not a lashed street",
					name.Value)
			}
		}
		return nil
	}
	if _, err := recipe.Construction(); err != nil {
		return reading.refuse(block, "%v", err)
	}
	return nil
}
