package trace

import (
	"sync/atomic"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// counting is a shape that nothing ever meets, as wide as the world so
// that every shadow ray asks it, and counts the asking.
type counting struct{ asked *atomic.Int64 }

func (counting) Bounds() maths.Box            { return maths.Everywhere() }
func (counting) Distance(p maths.Vec) float64 { return 1 }
func (c counting) Hit(maths.Ray, float64, float64) (float64, bool) {
	c.asked.Add(1)
	return 0, false
}

// A surface that passes all the light that reaches it keeps none of its
// own, so the sixteen rays that would light it from the sky are not cast:
// a ray down onto a pane asks the world for the nearest thing and for
// nothing else but what the bend and the sky beyond ask, a handful of
// times, where sixteen rays more would be seventeen or more.
func TestAPanePassingAllLightCastsNoSkyRays(t *testing.T) {
	var asked atomic.Int64
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: counting{&asked}})
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(1), Through: 1}})
	world.Sky = even(maths.Grey(1))
	ray := maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(0, -1, 0)}
	lone(world, ray, 0)
	if got := asked.Load(); got > 8 {
		t.Errorf("a ray onto a pane asked the world %d times, where "+
			"the pane's own sky light would ask it 17 or more", got)
	}
}
