//go:build draw && darwin

package session

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// TestAFilmOnTheCPUKeepsNoDrawer gives a gpu film a drawer and a cpu
// film none, and closing either, or no keeper, is harmless.
func TestAFilmOnTheCPUKeepsNoDrawer(t *testing.T) {
	cpu, err := filmKeeper(asked{})
	if err != nil || cpu.frames != nil {
		t.Fatalf("a cpu film kept a drawer (%v)", err)
	}
	onGPU, err := filmKeeper(asked{onGPU: true, nativeMeshes: true})
	if err != nil || onGPU.frames == nil {
		t.Fatalf("a gpu film kept no drawer (%v)", err)
	}
	for _, keep := range []*keeper{cpu, onGPU, nil} {
		keep.close()
		keep.close()
	}
}

// the film these checks draw: a few pixels, one ray each, and the frame
// times of a film at 24 a second, picked 2026-11-12
const (
	keptFilmWidth, keptFilmHeight, keptFilmRays = 24, 16, 1
	keptFilmStep                                = time.Second / 24
	// keptFilmWide is the pinhole's view across, in degrees
	keptFilmWide = 50
)

// rolling is a ball rolling past a lamp over a floor, so a film's frames
// differ, seen from the given place.
func rolling(t *testing.T) (*thing.World, asked) {
	t.Helper()
	world := thing.NewWorld()
	ball := &thing.Material{Name: "ball", Colour: maths.RGB(0.7, 0.4, 0.2)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: ball,
		Motion: rollingBall{}})
	world.Lamp(emit.Lit{At: maths.V(3, 4, 4), Beyond: 0.1,
		Strength: maths.RGB(40, 40, 40)})
	from := maths.V(0, 2, 8)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	return world, asked{onGPU: true, width: keptFilmWidth,
		height: keptFilmHeight, samples: keptFilmRays, wide: keptFilmWide,
		eye: &eye}
}

// rollingBall rolls along x at a metre a second.
type rollingBall struct{}

// TestAKeeperDrawsAFilmAsDrawMeasuredDoes draws a film's frames through
// the keeper's drawer: each is drawMeasured's picture to the bit, each a
// new generation, and the drawer lives through the keeper's frames.
func TestAKeeperDrawsAFilmAsDrawMeasuredDoes(t *testing.T) {
	world, given := rolling(t)
	keep, err := filmKeeper(given)
	if err != nil {
		t.Fatal(err)
	}
	defer keep.close()
	frames := keep.frames
	for index := range keptFilmFrames {
		given.at = time.Duration(index) * keptFilmStep
		got, profile, err := keep.picture(world, given, 0)
		mustFilmDevice(t, err)
		want, _, err := drawMeasured(world, given, 0)
		mustFilmDevice(t, err)
		if !sameFrameBits(want, got) {
			t.Fatalf("frame %d is not drawMeasured's", index)
		}
		if keep.frames != frames || keep.generation != uint64(index+1) ||
			profile.Retained.Held == 0 {
			t.Fatalf("frame %d: drawer %p, generation %d, %+v", index,
				keep.frames, keep.generation, profile.Retained)
		}
	}
}

// keptFilmFrames is how many frames of the film are drawn.
const keptFilmFrames = 3

// TestAKeeperDrawsACPUFrameWithoutItsDrawer draws a frame not asked of
// the gpu: by drawMeasured, with the drawer and its generation untouched.
func TestAKeeperDrawsACPUFrameWithoutItsDrawer(t *testing.T) {
	world, given := rolling(t)
	keep, err := filmKeeper(given)
	if err != nil {
		t.Fatal(err)
	}
	defer keep.close()
	given.onGPU = false
	got, profile, err := keep.measured(world, given, 0)
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := drawMeasured(world, given, 0)
	if err != nil || !sameFrameBits(want, got) || keep.generation != 0 ||
		profile.Retained != (gpu.FrameInputs{}) {
		t.Fatalf("a cpu frame went through the drawer (%v)", err)
	}
}

// mustFilmDevice fails, and does not skip, where there is no Metal
// device: a check that cannot run fails.
func mustFilmDevice(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, metal.ErrNoDevice) {
		t.Fatal("this check needs a Metal device and there is none:", err)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// sameFrameBits is whether two frames are one, every pixel compared as
// bits.
func sameFrameBits(one, other *frame.Frame) bool {
	if one == nil || other == nil || one.W != other.W || one.H != other.H {
		return false
	}
	bits := math.Float32bits
	for index, pixel := range one.Pix {
		next := other.Pix[index]
		if bits(pixel.R) != bits(next.R) || bits(pixel.G) != bits(next.G) ||
			bits(pixel.B) != bits(next.B) ||
			bits(pixel.Cover) != bits(next.Cover) ||
			bits(pixel.Depth) != bits(next.Depth) {
			return false
		}
	}
	return true
}

// At is where the ball is after a time.
func (rollingBall) At(at time.Duration) maths.Subspace {
	return maths.Subspace{Origin: maths.V(at.Seconds(), 0, 0)}
}
