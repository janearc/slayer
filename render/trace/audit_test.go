package trace

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// refusal is what f panics with, as a string, or "" for nothing.
func refusal(f func()) (got string) {
	defer func() { got, _ = recover().(string) }()
	f()
	return ""
}

// a hole spinning at its mass or past, or by no number, has no horizon,
// and is refused by name when the world is placed, on the caller's own
// goroutine, before any worker.
func TestAHoleSpinsUnderItsMass(t *testing.T) {
	for _, spin := range []float64{1, 2, math.NaN()} {
		w := thing.NewWorld()
		w.AddField(thing.Field{Mass: 1, Spin: spin, Reach: 40})
		if got := refusal(func() { stage(w, 0) }); !strings.Contains(got, "spins") {
			t.Errorf("a hole spinning at %v: %q", spin, got)
		}
	}
	w := thing.NewWorld()
	w.AddField(thing.Field{Mass: 1, Spin: 0.9, Reach: 40})
	if got := refusal(func() { stage(w, 0) }); got != "" {
		t.Errorf("a hole spinning at 0.9: %q", got)
	}
}

// a gas whose asymmetry is no lobe is refused when the world is placed,
// on the caller's goroutine, where the panic can be caught, not inside a
// worker, where it takes the process down.
func TestAGasThatCannotScatterIsRefusedWhenPlaced(t *testing.T) {
	bad := gas.Haze()
	bad.Phase = 1.5
	w := thing.NewWorld()
	w.AddVolume(thing.Volume{Medium: gas.Of{Shape: medium.Puff{Radius: 1, Thick: 1}, In: bad, Unit: 1}})
	if got := refusal(func() { stage(w, 0) }); !strings.Contains(got, "no lobe") {
		t.Errorf("a gas of asymmetry 1.5: %q", got)
	}
}
