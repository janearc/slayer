package named

import (
	"errors"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"io"
	"testing"
	"time"
)

func TestAuthoredUsesNormalWorldAndAbsoluteCameraClock(t *testing.T) {
	failed := errors.New("motion failure")
	world := thing.NewWorld()
	s := Scene{name: "test-authored", authored: &Authored{
		Check: func(p map[string]any) error {
			if p["kind"] != "test" {
				return failed
			}
			return nil
		},
		Build: func(map[string]any, io.Writer) (*thing.World, maths.Subspace, func(time.Duration) (maths.Subspace, error), error) {
			return world, maths.Local(), func(at time.Duration) (maths.Subspace, error) {
				if at < 0 {
					return maths.Subspace{}, failed
				}
				return maths.NewSubspace(maths.V(at.Seconds(), 0, 0), maths.Identity, 1), nil
			}, nil
		},
	}}
	if _, err := s.Build(map[string]any{}, nil); !errors.Is(err, failed) {
		t.Fatal("unchecked input")
	}
	b, err := s.Build(map[string]any{"kind": "test"}, nil)
	if err != nil || b.World() != world || !b.Timed() {
		t.Fatal("lost constructed world or clock", err)
	}
	for _, at := range []time.Duration{3 * time.Second, time.Second, 3 * time.Second} {
		if err = b.Advance(at); err != nil || b.Eye().Origin.X != at.Seconds() {
			t.Fatal("camera not absolute", err)
		}
	}
	if !errors.Is(b.Advance(-time.Second), failed) {
		t.Fatal("motion failure swallowed")
	}
}

func TestAuthoredCannotReplaceAnExistingScene(t *testing.T) {
	a := Authored{Name: "orbital", Unit: "m", Width: 1, Height: 1, Rays: 1, Wide: 1,
		Check: func(map[string]any) error { return nil },
		Build: func(map[string]any, io.Writer) (*thing.World, maths.Subspace, func(time.Duration) (maths.Subspace, error), error) {
			return nil, maths.Subspace{}, nil, nil
		}}
	if RegisterAuthored(a) == nil {
		t.Fatal("replaced built-in orbital")
	}
}

func TestAuthoredLightingPreviewRequiresOptIn(t *testing.T) {
	for _, scene := range []*Scene{{}, {authored: &Authored{}}} {
		if scene.AllowsFastLighting() {
			t.Fatal("unrequested lighting preview enabled")
		}
	}
	scene := &Scene{authored: &Authored{FastLighting: true}}
	if !scene.AllowsFastLighting() {
		t.Fatal("authored lighting preview refused")
	}
}

func TestAuthoredShotLensKeepsItsAbsoluteClock(t *testing.T) {
	scene := Scene{name: "authored lens", authored: &Authored{
		Check: func(map[string]any) error { return nil },
		Build: func(map[string]any, io.Writer) (*thing.World, maths.Subspace, func(time.Duration) (maths.Subspace, error), error) {
			return thing.NewWorld(), maths.Local(), nil, nil
		},
		Lens: func(map[string]any) (func(time.Duration) float64, error) {
			return func(at time.Duration) float64 {
				if at < 4*time.Second {
					return 55 * maths.Degree
				}
				return 90 * maths.Degree
			}, nil
		},
	}}
	made, err := scene.Build(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Duration{0, 4 * time.Second, time.Second} {
		wide, ok := made.WideAt(at)
		want := 55 * maths.Degree
		if at >= 4*time.Second {
			want = 90 * maths.Degree
		}
		if !ok || wide != want {
			t.Fatal("authored cut lost its lens", at, wide)
		}
	}
}
