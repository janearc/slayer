package cities

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// deckSeeds is how many bridges the deck tests draw, each of a seed of
// its own.
const deckSeeds = 6

// lookCosines are the cosines to the normal the deck is looked at by, from
// along its face to straight on.
var lookCosines = []float64{0, 0.05, 0.25, 0.5, 0.75, 1} // picked: a spread

// A surface's gloss is a share of the light it mirrors, and the roughness
// of a surface does not blur it, so any gloss on a slab of rough concrete
// is a perfect faint mirror: the water shows on its underside and a laser
// beam draws a sharp line across its edge. The deck of a bridge, which is
// concrete, mirrors nothing at any look.
func TestABridgesDeckMirrorsNothingAtAnyLook(t *testing.T) {
	for seed := range Seed(deckSeeds) {
		drawn := designOf(t, bridgeOf(t, seed, 0.5, shortSpan, 0))
		decks := named(drawn, "deck")
		if len(decks) == 0 {
			t.Fatalf("seed %d: a bridge with no deck", seed)
		}
		for _, deck := range decks {
			for _, cos := range lookCosines {
				if got := deck.Material.Reflects(cos); got != (maths.Light{}) {
					t.Fatalf("seed %d: the deck mirrors %v at a look of "+
						"cosine %v: gloss on rough concrete is a mirror "+
						"that roughness does not blur", seed, got, cos)
				}
			}
		}
	}
}

// the anti-test of the above: the cables of a bridge are steel and do
// mirror, so the deck's nothing is the deck's and not the check's.
func TestABridgesCablesStillMirror(t *testing.T) {
	drawn := designOf(t, bridgeOf(t, 1, 0.5, shortSpan, 0))
	cables := named(drawn, "cable")
	if len(cables) == 0 {
		t.Fatal("a bridge with no cable")
	}
	for _, cable := range cables {
		if got := cable.Material.Reflects(1); got == (maths.Light{}) {
			t.Fatalf("a cable mirrors nothing straight on: %v", got)
		}
	}
}
