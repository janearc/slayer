//go:build draw && darwin

package gpu

import (
	"bytes"
	"errors"
	"fmt"
	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"sync"
	"testing"
)

type identifiedFieldLens struct {
	countedFieldLens
	identity string
}

func TestGravityCompleteSeedCacheConcurrentReaders(t *testing.T) {
	renderer, err := NewRenderer(32)
	if err != nil {
		t.Fatal(err)
	}
	black, err := kerr.NewHole(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	field := &thing.Field{Mass: 1, Reach: 40}
	var workers sync.WaitGroup
	errors := make(chan error, 4)
	for worker := range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			input := fieldSeedInput{source: field, black: black, scale: 1,
				eye:  maths.Subspace{Origin: maths.V(float64(worker), 0, 60)},
				lens: film.Pinhole{}, width: 2, height: 2}
			reference := newScene()
			fieldSeeds(reference, input)
			for range 32 {
				encoded := newScene()
				renderer.cachedFieldSeeds(encoded, input)
				if !bytes.Equal(encoded.rays.bytes, reference.rays.bytes) {
					errors <- fmt.Errorf("concurrent seed tape changed for worker %d", worker)
					return
				}
			}
		}()
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
}

func (lens *identifiedFieldLens) RayKey() string { return lens.identity }

func TestGravityCompleteSeedCacheTracksCameraState(t *testing.T) {
	renderer, err := NewRenderer(32)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	field := &thing.Field{Mass: 1, Reach: 40}
	world.Fields = append(world.Fields, field)
	lens := &identifiedFieldLens{identity: "one"}
	failure := errors.New("stop before dispatch")
	ops := fieldDefaults()
	ops.seeds = renderer.cachedFieldSeeds
	ops.open = func(string, string) (fieldKernel, error) { return &failedFieldKernel{err: failure}, nil }
	eye := maths.Subspace{Origin: maths.V(0, 0, 60)}
	w, h := 2, 2
	run := func(sensor film.Sensor, want bool) {
		t.Helper()
		_, spent, err := fieldSampledUsing(world, eye, sensor, 0, w, h, 1, ops)
		if !errors.Is(err, failure) || spent.seedCached != want {
			t.Fatal("seed cache state", spent.seedCached, err)
		}
	}
	run(lens, false)
	saved := append([]byte(nil), renderer.seeds.rays...)
	originalTape := renderer.seeds.rays
	run(lens, true)
	if lens.calls != 4 {
		t.Fatal("unchanged camera was recomputed", lens.calls)
	}
	w = 3
	run(lens, false)
	h = 3
	run(lens, false)
	lens.identity = "two"
	run(lens, false)
	eye.Origin.X = 1
	run(lens, false)
	field.Mass = 2
	run(lens, false)
	field.Motion = thing.Still(maths.Subspace{Origin: maths.V(.1, 0, 0)})
	run(lens, false)
	field.EyeTime = 1.2
	run(lens, false)
	if lens.calls != 64 {
		t.Fatal("changed camera reused stale seeds", lens.calls)
	}
	if len(saved) != 256 {
		t.Fatal("seed tape has wrong size", len(saved))
	}
	if !bytes.Equal(saved, originalTape) {
		t.Fatal("cache replacement mutated an older tape")
	}
	w, h = 2, 2
	unknown := &countedFieldLens{}
	run(unknown, false)
	run(unknown, false)
	if unknown.calls != 8 {
		t.Fatal("unknown lens was cached", unknown.calls)
	}
	lens.identity = ""
	run(lens, false)
	run(lens, false)
	run(nil, false)
	run(nil, true)
	run(film.Pinhole{Wide: .5}, false)
	run(film.Pinhole{Wide: .5}, true)
}
