package pinnipeds

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Small independent scalar fields exercise the same lazy cache used by the
// full walrus, without paying for several full-resolution meshing runs.
func TestWalrusRestSurfacesAreConcurrentAndIndependent(t *testing.T) {
	var calls atomic.Int64
	field := func(radius float64) func(maths.Vec) float64 {
		return func(point maths.Vec) float64 { calls.Add(1); return point.Len() - radius }
	}
	grid := [3]int{16, 12, 12}
	a, b := &walrusRig{}, &walrusRig{}
	type outcome struct {
		surface *walrusRestSurface
		err     error
	}
	results := make(chan outcome, 8)
	start := make(chan struct{})
	for i := 0; i < cap(results); i++ {
		go func() {
			<-start
			surface, err := a.restSurface(field(.4), true, grid)
			results <- outcome{surface, err}
		}()
	}
	close(start)
	var ground *walrusRestSurface
	for i := 0; i < cap(results); i++ {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		if ground == nil {
			ground = result.surface
		}
		if result.surface != ground {
			t.Fatal("one rig built multiple cached surfaces")
		}
	}
	if len(ground.points) == 0 || len(ground.faces) == 0 {
		t.Fatal("empty fixture surface")
	}
	before := calls.Load()
	again, err := a.restSurface(field(.6), true, grid)
	if err != nil {
		t.Fatal(err)
	}
	if again != ground || calls.Load() != before {
		t.Fatal("cache did not reuse its first surface")
	}
	swimming, err := a.restSurface(field(.6), false, grid)
	if err != nil {
		t.Fatal(err)
	}
	other, err := b.restSurface(field(.6), true, grid)
	if err != nil {
		t.Fatal(err)
	}
	if swimming == ground || other == ground || other == swimming {
		t.Fatal("separate fields shared an entry")
	}
	if reflect.DeepEqual(ground.points, swimming.points) {
		t.Fatal("ground and swimming fields were conflated")
	}
	if !reflect.DeepEqual(swimming.points, other.points) || !reflect.DeepEqual(swimming.faces, other.faces) {
		t.Fatal("identical independent fields differ")
	}
}

func TestWalrusPaintRetainsOnlyItsSmallImmutableState(t *testing.T) {
	rig, err := newWalrusRig(defaultPinniped("walrus"))
	if err != nil {
		t.Fatal(err)
	}
	paint := rig.hideMaterial().Texture.(walrusHideTexture)
	if paint.rig == rig || paint.rig.body != nil {
		t.Fatal("paint retained the geometry recipe")
	}
	points := []maths.Vec{maths.V(1.2, 0, .1), maths.V(.3, .2, .4), maths.V(-.7, -.1, -.2)}
	before := make([]maths.Light, len(points))
	for i, point := range points {
		before[i] = (walrusHideTexture{rig: rig}).At(point)
	}
	rig = nil
	runtime.GC()
	for i, point := range points {
		if got := paint.At(point); got != before[i] {
			t.Fatalf("paint changed after rig release at %v", point)
		}
	}
	runtime.KeepAlive(paint)
}
