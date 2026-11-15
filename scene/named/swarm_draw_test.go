//go:build draw

package named_test

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/trace"
)

// drawsTiny draws a built scene at the tiny size through its own lens at
// its own time, and fails the test unless something in it is lit.
func drawsTiny(t *testing.T, name string, built *named.Built) {
	t.Helper()
	made := mustGet(t, name).Defaults()
	lens := film.Pinhole{Wide: made.Wide()}
	drawn, err := trace.Sampled(built.World(), built.Eye(), lens,
		made.At(), tinyWide, tinyHigh, once)
	if err != nil {
		t.Fatal(err)
	}
	for _, pixel := range drawn.Pix {
		if pixel.R+pixel.G+pixel.B > none {
			return
		}
	}
	t.Errorf("%s drawn tiny is black", name)
}

// The swarm builds small and draws at a tiny size: a star, its panels and
// the haze, with nothing that keeps its own time.
func TestTheSwarmBuildsAndDraws(t *testing.T) {
	built := mustBuild(t, "swarm", smallSwarm)
	if len(built.World().Volumes) != once || built.Timed() {
		t.Errorf("the swarm has %d volumes and timed %v; want the "+
			"haze alone, untimed", len(built.World().Volumes),
			built.Timed())
	}
	built.Set(aSpell)
	drawsTiny(t, "swarm", built)
}
