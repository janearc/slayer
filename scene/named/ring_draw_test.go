//go:build draw

package named_test

import (
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// The ring builds at its defaults and draws at a tiny size, on the cpu
// alone.
func TestTheRingBuildsAndDraws(t *testing.T) {
	scene := mustGet(t, "ring")
	if !scene.Draws(named.CPU) || scene.Draws(named.GPU) {
		t.Errorf("the ring draws on %v; want the cpu alone",
			scene.Renderers())
	}
	built := mustBuild(t, "ring", nil)
	if built.Timed() {
		t.Error("the ring keeps time, and has no lamp that does")
	}
	drawsTiny(t, "ring", built)
}
