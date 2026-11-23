package cities

import (
	"strings"
	"testing"
)

// fineGreebleParts are what the cyberpunk layer adds at the demo level.
var fineGreebleParts = []string{"air conditioner", "fan grille", "duct",
	"hung cable", "bolted box", "conduit", "bolt head"}

// cyberPins are a plain curtain tower under the cyberpunk layer.
func cyberPins() Pins {
	pins := plainPins(skinCurtain)
	pins.Cyber = &yes
	return pins
}

func TestTheDemoCyberpunkLayerAddsFineGreeblesAndTheQuickOneNone(
	t *testing.T) {
	quickTower := towerOf(t, 3, 0, cyberPins())
	quick := counts(drawnTower(t, quickTower, "quick", 0))
	demoTower := towerOf(t, 3, 0, cyberPins())
	demo := counts(drawnTower(t, demoTower, "demo", 0))
	for _, name := range fineGreebleParts {
		if quick[name] != 0 || demo[name] == 0 {
			t.Errorf("%d %q at quick, %d at demo", quick[name], name,
				demo[name])
		}
	}
	if demo["bolt head"] != 4*demo["bolted box"] {
		t.Errorf("%d bolt heads on %d bolted boxes", demo["bolt head"],
			demo["bolted box"])
	}
}

func TestATowerWithoutTheCyberpunkLayerHasNoFineGreebles(t *testing.T) {
	demoTower := towerOf(t, 3, 0, plainPins(skinCurtain))
	demo := counts(drawnTower(t, demoTower,
		"demo", 0))
	for _, name := range fineGreebleParts {
		if demo[name] != 0 {
			t.Errorf("%d %q on a tower with no cyberpunk layer", demo[name],
				name)
		}
	}
}

func TestFineGreeblesAreSteelSoTheWorldRustsThem(t *testing.T) {
	demoTower := towerOf(t, 3, 0, cyberPins())
	drawn := drawnTower(t, demoTower, "demo", 0)
	for _, name := range []string{"air conditioner", "duct", "bolted box"} {
		for _, one := range named(drawn, name) {
			if !strings.Contains(one.Material.Name, "steel") {
				t.Errorf("a %s is %q", name, one.Material.Name)
			}
		}
	}
}

func TestEveryFineGreebleStaysOnItsFace(t *testing.T) {
	quickTower := towerOf(t, 3, 0, cyberPins())
	_, outer := boundsOf(drawnTower(t, quickTower,
		"quick", 0))
	demoTower := towerOf(t, 3, 0, cyberPins())
	drawn := drawnTower(t, demoTower, "demo", 0)
	for _, name := range fineGreebleParts {
		for _, one := range named(drawn, name) {
			low, high := boundsOf([]Rendered{one})
			if low.Y < -tolerance || high.Y > outer.Y+tolerance {
				t.Errorf("a %s runs from %v to %v, off its face", name,
					low.Y, high.Y)
			}
		}
	}
}
