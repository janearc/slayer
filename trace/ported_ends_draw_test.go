//go:build draw

package trace

import (
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// finishes runs call and fails the test when it has not returned in five
// seconds: a loop that counts the wrong way fails the run, and does not
// hold it up.
func finishes(t *testing.T, what string, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		call()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not finish in five seconds", what)
	}
}

// A picture of a few pixels is drawn in no time, whatever way the rows,
// the steps of a march and the shadow samples count, in every kind of
// world: a ball under a sun, smoke over a floor, air round a planet, and a
// hole. A ray that gets out of a field comes back lit, which is asked
// first: a march that goes on after it is done does not come back at all.
func TestPortedEveryPictureIsDrawnInNoTime(t *testing.T) {
	from := maths.V(0, 3, 6)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0.7, 0))}
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	for _, c := range []struct {
		name  string
		world *thing.World
	}{
		{"a ball under a sun", smoking()},
		{"smoke over a glowing floor", puffed(2)},
		{"the air round a planet", earthlike()},
	} {
		world := c.world
		finishes(t, c.name+", drawn", func() {
			must(Render(world, eye, lens, 0, 6, 5))
		})
		finishes(t, c.name+", sampled twice a pixel", func() {
			must(Sampled(world, eye, lens, 0, 3, 3, 2))
		})
		finishes(t, c.name+", in a band", func() {
			must(RenderIn(world, eye, lens, 0, 6, 5,
				maths.Band{Nanometres: 1}))
		})
		finishes(t, c.name+", in a band with a sun to light it", func() {
			must(RenderIn(world, eye, lens, 0, 6, 5,
				maths.Band{Nanometres: 656}))
		})
	}
	hole := holeWorld(nil)
	var escaped maths.Light
	finishes(t, "a ray bent by a field", func() {
		escaped = lone(hole, maths.Ray{At: maths.V(0, 8, -100),
			Dir: maths.V(0, 0, 1)}, 0).Light()
	})
	if escaped == (maths.Light{}) {
		t.Fatal("a ray that gets out of a field came back dark")
	}
	finishes(t, "rays in a field", func() {
		lone(hole, maths.Ray{At: maths.V(0, 0, -20), Dir: maths.V(0, 0, -1)}, 0)
		lone(hole, maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}, 0)
	})
}
