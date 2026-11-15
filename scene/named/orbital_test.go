package named_test

import (
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// The orbital is drawn by the cpu and the gpu: cmd/orbital's gpu test
// holds the gpu's pictures of both views to the tracer's.
func TestTheOrbitalDrawsOnBoth(t *testing.T) {
	scene := mustGet(t, "orbital")
	if !scene.Draws(named.CPU) || !scene.Draws(named.GPU) {
		t.Errorf("the orbital draws on %v", scene.Renderers())
	}
}
