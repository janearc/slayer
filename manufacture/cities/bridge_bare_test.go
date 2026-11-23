package cities

import "testing"

// bareSeed is the seed the bare bridge tests draw from. picked.
const bareSeed = 5

// A bare bridge has the deck of a road and nothing built on it: its design
// has fewer parts than a bridge of the same seed that is built on, and
// reaches less far across the deck.
func TestABareBridgeHasNothingBuiltOnItsDeck(t *testing.T) {
	bare, err := NewBareBridge(bareSeed, Irregularity{Value: 0.05}, shortSpan)
	if err != nil {
		t.Fatal(err)
	}
	built := bridgeOf(t, bareSeed, 0.05, shortSpan, 0)
	bareDesign, err := bare.Design(hullLevel, 0)
	if err != nil {
		t.Fatal(err)
	}
	builtDesign, err := built.Design(hullLevel, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(bareDesign.Elements) >= len(builtDesign.Elements) {
		t.Errorf("bare has %d parts, built %d", len(bareDesign.Elements),
			len(builtDesign.Elements))
	}
	low, high, err := bareDesign.extent(hullLevel)
	if err != nil {
		t.Fatal(err)
	}
	builtLow, builtHigh, err := builtDesign.extent(hullLevel)
	if err != nil {
		t.Fatal(err)
	}
	if high.Z-low.Z >= builtHigh.Z-builtLow.Z {
		t.Errorf("bare is %v m across, built %v m", high.Z-low.Z,
			builtHigh.Z-builtLow.Z)
	}
}

// Anti: a bridge made by NewBridge is not bare, and keeps its deck's width.
func TestABridgeBuiltOnIsNotBare(t *testing.T) {
	built := bridgeOf(t, bareSeed, 0.05, shortSpan, 0)
	if built.Bare ||
		built.deckWidth() != built.drawing.fixed("deck-width") {
		t.Errorf("bare %v, deck %v m", built.Bare, built.deckWidth())
	}
}

// A bare bridge has the culture's road width, and the same pylons as a
// built one of its seed: they stand a half deck out, so they are nearer.
func TestABareBridgeHasTheRoadWidth(t *testing.T) {
	bare, err := NewBareBridge(bareSeed, Irregularity{Value: 0.05}, shortSpan)
	if err != nil {
		t.Fatal(err)
	}
	if bare.deckWidth() != bare.drawing.fixed("bare-deck-width") ||
		len(bare.pylons) != len(bridgeOf(t, bareSeed, 0.05, shortSpan,
			0).pylons) {
		t.Errorf("deck %v m, %d pylons", bare.deckWidth(), len(bare.pylons))
	}
}
