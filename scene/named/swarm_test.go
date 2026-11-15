package named_test

import (
	"errors"
	"io"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/scene/named"
)

// mustGet is the scene of that name, or the test stops.
func mustGet(t *testing.T, name string) *named.Scene {
	t.Helper()
	scene, err := named.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return scene
}

// mustBuild is the scene built with the values, or the test stops.
func mustBuild(t *testing.T, name string,
	with map[string]any) *named.Built {
	t.Helper()
	built, err := mustGet(t, name).Build(with, io.Discard)
	if err != nil {
		t.Fatalf("%s with %v: %v", name, with, err)
	}
	return built
}

// a small swarm: few orbits of few panels
var smallSwarm = map[string]any{"orbits": few, "panels": few}

// The swarm's eye is as far out as asked, looking at the star.
func TestTheSwarmsEyeIsAtTheDistanceAsked(t *testing.T) {
	built := mustBuild(t, "swarm", map[string]any{"orbits": once,
		"panels": once, "distance": fraction * few})
	if away := built.Eye().Origin.Len(); math.Abs(away-fraction*few) >
		closeEnough {
		t.Errorf("the eye is %v out, want %v", away, fraction*few)
	}
}

// closeEnough is how near two lengths must be to be the same: picked by
// the author of this package, 2026-11-12, well over a float64's rounding
// at these sizes.
const closeEnough = 1e-9

// anti-tests: a swarm with no panels to an orbit is refused by the
// lantern, and nothing is built.
func TestASwarmTheLanternRefusesIsNotBuilt(t *testing.T) {
	built, err := mustGet(t, "swarm").Build(map[string]any{
		"orbits": once, "panels": none}, nil)
	if err == nil || built != nil {
		t.Errorf("a swarm of no panels: %v, %v", built, err)
	}
}

// The swarm is drawn by the cpu and the gpu.
func TestTheSwarmDrawsOnBoth(t *testing.T) {
	scene := mustGet(t, "swarm")
	if !scene.Draws(named.CPU) || !scene.Draws(named.GPU) {
		t.Errorf("the swarm draws on %v", scene.Renderers())
	}
}

// A whole number is read from a float64 with no fraction, as a JSON
// decoder gives it, and from an int64 or a uint64, as a YAML decoder may.
// A duration is read from its text.
func TestValuesAreReadAsTheirKind(t *testing.T) {
	scene := mustGet(t, "swarm")
	for _, with := range []map[string]any{
		{"orbits": float64(few)}, {"orbits": int64(few)},
		{"orbits": uint64(few)}, {"band-width": few},
		{"band-width": int64(few)}, {"band-width": uint64(few)},
		{"fastest": spelled}, {"fastest": aSpell},
	} {
		if err := scene.Check(with); err != nil {
			t.Errorf("%v: %v", with, err)
		}
	}
}

// anti-tests: a whole number with a fraction, a whole number too large to
// hold, a number that is no number, and a duration with no unit or one
// that is not a duration at all are each refused as the wrong kind.
func TestNearMissesOfAKindAreRefused(t *testing.T) {
	scene := mustGet(t, "swarm")
	for _, with := range []map[string]any{
		{"orbits": fraction}, {"orbits": uint64(math.MaxUint64)},
		{"orbits": math.Inf(once)}, {"band-width": math.NaN()},
		{"fastest": "20"}, {"fastest": int64(time.Second)},
	} {
		if err := scene.Check(with); !errors.Is(err, named.ErrKind) {
			t.Errorf("%v: %v; want ErrKind", with, err)
		}
	}
}
