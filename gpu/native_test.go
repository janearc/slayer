//go:build darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

func TestNativeShadowsKeepPrimaryHitsAndAgreeWithSoftware(t *testing.T) {
	world, eye := hall(t)
	compareNativeFrame(t, world, eye)
}

func compareNativeFrame(t *testing.T, world *thing.World, eye maths.Subspace) {
	t.Helper()
	const width, height = 48, 36
	software, _, err := sampledTimed(world, nil, eye, film.Pinhole{}, 0,
		width, height, 2)
	if err != nil {
		t.Fatal(err)
	}
	native, spent, err := sampledBackend(world, nil, eye, film.Pinhole{}, 0,
		width, height, 2, nativeShadowTraversal)
	if err != nil {
		t.Fatal(err)
	}
	if spent.acceleration <= 0 || spent.accelerationGPU <= 0 {
		t.Fatal("native hierarchy construction was not measured")
	}
	for i, pixel := range software.Pix {
		other := native.Pix[i]
		if math.Float32bits(pixel.Cover) != math.Float32bits(other.Cover) ||
			math.Float32bits(pixel.Depth) != math.Float32bits(other.Depth) {
			t.Fatalf("primary hit changed at pixel %d", i)
		}
	}
	apart := compare(software, native, width, height)
	t.Logf("native shadows: %+v; build wall %v, gpu %v; trace %v",
		apart, spent.acceleration, spent.accelerationGPU, spent.trace)
	if apart.mean > 2e-3 {
		t.Fatalf("native shadow disagreement: %+v", apart)
	}
}

func TestNativeShadowsOfSharedFoliageAgreeWithSoftware(t *testing.T) {
	for _, name := range []string{"coast-redwood", "giant-sequoia"} {
		t.Run(name, func(t *testing.T) {
			row, err := manufacture.KindNamed(name)
			if err != nil {
				t.Fatal(err)
			}
			made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(5), "quick")
			if err != nil {
				t.Fatal(err)
			}
			things, err := made.Things(maths.Subspace{})
			if err != nil {
				t.Fatal(err)
			}
			world := thing.NewWorld()
			world.Sky = sky.Gradient{Horizon: maths.Grey(0.02), Zenith: maths.Grey(0.02)}
			groups := 0
			for _, one := range things {
				if _, ok := one.Shape.(*shape.MeshGroup); ok {
					groups++
				}
				world.Add(one)
			}
			if groups == 0 {
				t.Fatal("no shared foliage in the fixture")
			}
			lamp, err := emit.NewDistant(maths.V(-1, 2, 3), maths.Grey(1))
			if err != nil {
				t.Fatal(err)
			}
			world.Lamp(lamp)
			height := made.Bounds().Max.Y
			from := maths.V(height, height*0.8, height*1.8)
			eye := maths.Subspace{Origin: from,
				Orientation: maths.LookAt(from, maths.V(0, height*0.5, 0))}
			compareNativeFrame(t, world, eye)
		})
	}
}
