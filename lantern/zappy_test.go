package lantern_test

import (
	"testing"
	"time"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// zappy as the orbital's far view had it in code before it was named:
// the numbers zappy.yaml must give back, copied from scene/named's
// farView at 71cc0446
var farView = lantern.Beamed{Radius: 9, Kelvin: 9000, Green: 6000,
	Orbits: 12, Panels: 4, Turn: time.Minute,
	Fastest: 15 * time.Second, Tumble: 40 * time.Second, Wide: 9,
	Narrowest: 2, Spread: 1.2, Wear: 1, Glass: 2.2, Glare: 30,
	Axis: maths.V(0.3, 1, 0.2),
	Beams: lantern.Beams{Width: 6, Soft: 4, Offset: 15, Tilt: 50,
		Cone: 35, Spin: 1500 * time.Millisecond,
		Precess: 5 * time.Second}}

// Zappy as defined is the far view's zappy, value for value.
func TestZappyIsTheFarViewsZappy(t *testing.T) {
	if got := lantern.Zappy(); got != farView {
		t.Errorf("zappy is %+v, want %+v", got, farView)
	}
}

// anti-tests: each call is a copy of its own; changing one changes no
// other.
func TestZappyIsACopy(t *testing.T) {
	changed := lantern.Zappy()
	changed.Orbits = once
	if lantern.Zappy().Orbits == once {
		t.Error("changing one zappy changed the definition")
	}
}

// the angles the beams are given as, in degrees, and the radians Go
// makes of each written as a constant: what the beams must be built at
const (
	offsetDegrees = 15
	offsetRadians = offsetDegrees * maths.Degree
	once          = 1
)

// The beams are built at the radians Go makes of their degrees as a
// constant: 15 degrees multiplied at run time lands a bit away, and the
// far view's picture would move.
func TestTheBeamsAreAtTheConstantsRadians(t *testing.T) {
	made, err := lantern.Zappy().Lantern(new(time.Duration))
	if err != nil {
		t.Fatal(err)
	}
	if made.Beams.Offset != offsetRadians {
		t.Errorf("the offset is %v radians, want %v", made.Beams.Offset,
			float64(offsetRadians))
	}
}

// anti-tests: zappy with no clock is refused: its beams could not be told
// the time, and nothing is placed.
func TestZappyWithNoClockIsRefused(t *testing.T) {
	world := thing.NewWorld()
	if _, err := lantern.Zappy().Place(world, nil); err == nil ||
		len(world.Things) != 0 {
		t.Errorf("zappy with no clock: %v, %d things", err,
			len(world.Things))
	}
}
