package trace

import (
	"runtime"
	"sync"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// knocking is a sky that does one thing the first time a ray asks it
// for light, which is when a picture is under way and its world has
// been staged.
type knocking struct {
	once *sync.Once
	do   func()
}

// A picture is drawn from the world as it was when the picture began.
// The first ray to reach the sky changes the world's lamp where it
// lies and adds two hundred more; every pixel after it is still the
// first lamp's, so the picture is the first lamp's picture.
//
// The change is made from inside the drawing, by one of its own
// workers, so it lands part way through every time, whatever the
// threads. Only the race detector sees the read itself; this test sees
// what the read does to the picture.
func TestALampChangedWhileDrawingIsNotRead(t *testing.T) {
	lens := film.Pinhole{}
	quiet := smoking()
	quiet.Sky = knocking{once: &sync.Once{}, do: func() {}}
	want := draw(quiet, overTheScene, lens, 0, 64, 64)

	world := smoking()
	world.Sky = knocking{once: &sync.Once{}, do: func() {
		world.Lamps[0] = otherSun()
		for range 200 {
			world.Lamp(otherSun())
		}
	}}
	got := draw(world, overTheScene, lens, 0, 64, 64)
	samePixels(t, got, want, "with its lamp changed part way")
}

// paint asks for every pixel of a frame exactly once, whatever the frame's
// shape, and the frame is the same on one thread as on four: which is Render's
// claim that a picture is the same however its rows are shared out.
func TestPaintAsksForEveryPixelOnce(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {3, 7}, {64, 1}} {
		width, height := size[0], size[1]
		var counting sync.Mutex
		asked := map[[2]int]int{}
		paint(width, height, func(x, y int) frame.Pixel {
			counting.Lock()
			defer counting.Unlock()
			asked[[2]int{x, y}]++
			return frame.Pixel{}
		})
		if len(asked) != width*height {
			t.Errorf("%d by %d: %d pixels asked for", width, height,
				len(asked))
		}
		for pixel, times := range asked {
			if times == 1 {
				continue
			}
			t.Errorf("%d by %d: pixel %v asked for %d times", width,
				height, pixel, times)
		}
	}
	lens := film.Pinhole{}
	was := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(was)
	alone := draw(smoking(), overTheScene, lens, 0, 16, 16)
	runtime.GOMAXPROCS(4)
	shared := draw(smoking(), overTheScene, lens, 0, 16, 16)
	samePixels(t, shared, alone, "on four threads against one,")
}

// Light is a dim grey, and the first call does what the sky was given
// to do.
func (sky knocking) Light(maths.Vec) maths.Light {
	sky.once.Do(sky.do)
	return maths.Grey(0.1)
}
