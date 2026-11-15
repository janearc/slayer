package named_test

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// the values the played tests hold: twice as bright, half again as large,
// picked by the author of this file, 2026-11-12
const (
	brighter = 2.0
	larger   = 1.5
)

// anti-tests: a scene with no channels has none, takes no values, and
// refuses any by name.
func TestAnUnplayedSceneRefusesValues(t *testing.T) {
	built := mustBuild(t, "swarm", smallSwarm)
	if len(built.Channels()) != 0 || built.Play(nil) != nil {
		t.Error("the swarm has channels")
	}
	err := built.Play(map[string]float64{"star-brightness": brighter})
	if !errors.Is(err, named.ErrUnplayed) {
		t.Errorf("the swarm played: %v; want ErrUnplayed", err)
	}
}
