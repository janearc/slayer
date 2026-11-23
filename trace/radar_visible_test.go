package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// A radar lights nothing in visible light.
func TestARadarLightsNothingInVisibleLight(t *testing.T) {
	eye, lens := tenBack(), film.Pinhole{Wide: 30 * maths.Degree}
	world := radarWorld(radarGround(nil), eye.Origin)
	seen := draw(world, eye, lens, 0, 41, 41).At(20, 20)
	if seen.Cover != 1 || seen.Light() != (maths.Light{}) {
		t.Errorf("in visible light the radar's ground is %+v", seen)
	}
}
