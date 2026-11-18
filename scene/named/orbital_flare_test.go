package named

import (
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestOrbitalNightFlareKeepsPreviousCallback(t *testing.T) {
	world := thing.NewWorld()
	var called time.Duration
	view := &Built{world: world, tick: func(at time.Duration) { called = at }}
	addOrbitalFlares(view, 6800*time.Millisecond)
	view.Set(1400 * time.Millisecond) // The 8.2 second peak on the continuous flare clock.
	peak := world.Lamps[0].(emit.Lit)
	if called != 1400*time.Millisecond || peak.Strength.Luminance() <= 0 {
		t.Fatal("night flare failed to preserve the preceding city callback or reach its peak")
	}
	view.Set(5 * time.Second)
	quiet := world.Lamps[0].(emit.Lit)
	if called != 5*time.Second || quiet.Strength.Luminance() >= peak.Strength.Luminance()*.001 {
		t.Fatal("night flare did not fade or stopped the preceding callback")
	}
}

func TestOrbitalFlaresIlluminateFromMagnetarAndFade(t *testing.T) {
	world := thing.NewWorld()
	view := &Built{world: world}
	addOrbitalFlares(view, 0)
	view.Set(2500 * time.Millisecond)
	lamp, ok := world.Lamps[0].(emit.Lit)
	if !ok || lamp.At != (maths.Vec{}) {
		t.Fatalf("flare does not originate at magnetar: %#v", world.Lamps[0])
	}
	toward, _, lit := lamp.Light(maths.V(orbitalRadius, 0, 0), maths.V(-1, 0, 0))
	if toward.X > -.99 || lit.B < 3 || lit.B <= lit.R {
		t.Fatalf("unexpected burst direction or colour: %v %v", toward, lit)
	}
	view.Set(5 * time.Second)
	_, _, quiet := world.Lamps[0].(emit.Lit).Light(maths.V(orbitalRadius, 0, 0), maths.V(-1, 0, 0))
	if quiet.Luminance() > .001 {
		t.Fatalf("flare never quiets: %v", quiet)
	}
	view.Set(2500 * time.Millisecond)
	_, _, again := world.Lamps[0].(emit.Lit).Light(maths.V(orbitalRadius, 0, 0), maths.V(-1, 0, 0))
	if again != lit {
		t.Fatal("seeking changes burst history")
	}
}
