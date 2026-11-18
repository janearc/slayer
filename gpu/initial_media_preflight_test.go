//go:build darwin

package gpu

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

func TestGPUInitialMediaPreflightRefusesBeforeOpeningKernel(t *testing.T) {
	world := thing.NewWorld()
	frames, err := NewFrames(Software)
	if err != nil {
		t.Fatal(err)
	}
	defer frames.Close()
	_, _, err = frames.DrawWithInitialMedia(world, 1, maths.Subspace{},
		film.Pinhole{}, 0, 1, 1, 1,
		[]trace.InitialMedium{{Index: math.NaN()}})
	if err == nil || !strings.Contains(err.Error(), "initial medium 0 index") {
		t.Fatalf("invalid initial stack was not refused in preflight: %v", err)
	}
}

func TestGPUInitialMediaRefusesMovingCameraAndCompositeBeforeDevice(t *testing.T) {
	world := thing.NewWorld()
	media := []trace.InitialMedium{{Index: 1.33}}
	frames, err := NewFrames(Software)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = frames.DrawOpenWithInitialMedia(world, 1, maths.Subspace{},
		maths.Subspace{Origin: maths.V(0, 1, 0)}, film.Pinhole{}, 0,
		1, 1, 2, media)
	if err == nil || !strings.Contains(err.Error(), "moving shutter") {
		t.Fatalf("moving camera stack was not refused in preflight: %v", err)
	}
	frames.Close()

	composite, err := NewCompositeFrames(Software, SurfacePlate)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = composite.DrawWithInitialMedia(world, 1, maths.Subspace{},
		film.Pinhole{}, 0, 1, 1, 1, media)
	if err == nil || !strings.Contains(err.Error(), "composite pass") {
		t.Fatalf("composite camera stack was not refused in preflight: %v", err)
	}
	composite.Close()
}

func TestGPUFrameIdentityIncludesInitialOpticalValues(t *testing.T) {
	one, err := thing.NewClearMedium("one", maths.RGB(0.1, 0.2, 0.3), 1)
	if err != nil {
		t.Fatal(err)
	}
	two, err := thing.NewClearMedium("two", maths.RGB(0.1, 0.2, 0.3), 1)
	if err != nil {
		t.Fatal(err)
	}
	base := shot{world: thing.NewWorld(), generation: 1,
		at: time.Second, width: 2, height: 2, side: 1}
	first := base
	first.initialMedia = []trace.InitialMedium{{Index: 1.33,
		ClearMedium: one}}
	second := base
	second.initialMedia = []trace.InitialMedium{{Index: 1.33,
		ClearMedium: two}}
	if keyOf(first, softwareTraversal) == keyOf(second, softwareTraversal) {
		t.Fatal("different initial medium identities shared a retained frame key")
	}
	copyOfOne, err := thing.NewClearMedium("one", maths.RGB(0.1, 0.2, 0.3), 1)
	if err != nil {
		t.Fatal(err)
	}
	second.initialMedia = []trace.InitialMedium{{Index: 1.33,
		ClearMedium: copyOfOne}}
	if keyOf(first, softwareTraversal) != keyOf(second, softwareTraversal) {
		t.Fatal("equivalent initial medium values changed retained frame identity")
	}
}
