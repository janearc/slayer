package cells_test

import (
	"errors"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/cells"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"testing"
	"time"
)

func TestNativeSamplerUsesSharedSceneTimingAndDevelopment(t *testing.T) {
	pace, err := cells.NewPace(0, 6)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	sample := func(world *thing.World, eye maths.Subspace, lens film.Sensor,
		at time.Duration, w, h, side int) (*frame.Frame, error) {
		called = true
		if world == nil || at != time.Second || w != 2 || h != 3 || side != 1 {
			t.Fatal("sampler lost scene inputs", at, w, h, side)
		}
		return frame.New(w, h), nil
	}
	scene, err := cells.New(film.Pinhole{}, colour.Tone{}, 1, pace, nil, sample)
	if err != nil {
		t.Fatal(err)
	}
	scene.Place(thing.NewWorld(), maths.Subspace{})
	image, err := scene.Frame(6, 2, 3)
	if err != nil || !called || image.Bounds().Dy() != 3 {
		t.Fatal("chosen sampler not used", err)
	}
	for _, samplers := range [][]cells.Sampler{{nil}, {sample, sample}} {
		if _, err = cells.New(film.Pinhole{}, colour.Tone{}, 1, pace, nil, samplers...); err == nil {
			t.Fatal("invalid sampler accepted")
		}
	}
	failure := func(*thing.World, maths.Subspace, film.Sensor, time.Duration, int, int, int) (*frame.Frame, error) {
		return nil, errors.New("GPU failure")
	}
	scene, err = cells.New(film.Pinhole{}, colour.Tone{}, 1, pace, nil, failure)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = scene.Frame(0, 1, 1); err == nil {
		t.Fatal("sampler failure hidden")
	}
}
