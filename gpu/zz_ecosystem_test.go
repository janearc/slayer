//go:build darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

func TestEcosystemRecipesAgreeOnCPUAndGPU(t *testing.T) {
	for _, name := range []string{"coast-redwood", "giant-sequoia", "optical-observatory", "coast-redwood-shoot", "giant-sequoia-shoot"} {
		for _, detail := range manufacture.Complexities() {
			t.Run(name+"/"+detail, func(t *testing.T) {
				row, err := manufacture.KindNamed(name)
				if err != nil {
					t.Fatal(err)
				}
				made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(5), detail)
				if err != nil {
					t.Fatal(err)
				}
				things, err := made.Things(maths.Subspace{})
				if err != nil {
					t.Fatal(err)
				}
				world := thing.NewWorld()
				world.Sky = sky.Gradient{Horizon: maths.Grey(0.02), Zenith: maths.Grey(0.02)}
				for _, one := range things {
					world.Add(one)
				}
				lamp, err := emit.NewDistant(maths.V(-1, 2, 3), maths.Grey(1))
				if err != nil {
					t.Fatal(err)
				}
				world.Lamp(lamp)
				height := made.Bounds().Max.Y
				from, to := maths.V(height, height*0.8, height*1.8), maths.V(0, height*0.5, 0)
				if name == "coast-redwood-shoot" || name == "giant-sequoia-shoot" {
					box := made.Bounds()
					size := box.Max.Sub(box.Min)
					extent := max(size.X, size.Y, size.Z)
					to = box.Min.Add(box.Max).Scale(0.5)
					from = to.Add(maths.V(extent, extent*0.3, extent*1.8))
				}

				eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, to)}
				picture := drawn(t, world, eye)
				covered := 0
				for _, pixel := range picture.Pix {
					if pixel.Cover > 0 {
						covered++
					}
				}
				if covered < 5 {
					t.Fatalf("only %d surface pixels", covered)
				}
				apart := drawnBoth(t, world, eye)
				t.Logf("mean relative difference %.6g", apart.mean)
				if !(apart.mean <= 2e-3) || apart.cover > 4 {
					t.Errorf("CPU/GPU disagreement: %+v", apart)
				}
			})
		}
	}
}

func TestSharedGroveIsBakedAndDrawnOnGPU(t *testing.T) {
	field, err := medium.NewGrove(medium.GroveWeather{
		Extinction: 0.08, Depth: 8, Coverage: 1, Fade: true},
		[]medium.Canopy{{Radius: 7, Height: 12}}, func(x, z float64) float64 { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Sky = sky.Gradient{Horizon: maths.Grey(1), Zenith: maths.Grey(1), Ground: maths.Grey(1)}
	world.AddVolume(thing.Volume{Medium: field,
		Material: &thing.Material{Name: "prescribed mist"}})
	from := maths.V(0, 4, 20)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 4, 0))}
	picture := drawn(t, world, eye)
	dimmed := 0
	for _, pixel := range picture.Pix {
		if pixel.R < 0.95 {
			dimmed++
		}
	}
	if dimmed < 100 {
		t.Fatalf("only %d pixels passed through mist", dimmed)
	}
	apart := drawnBoth(t, world, eye)
	t.Logf("mean relative difference %.6g", apart.mean)
	if !(apart.mean <= 2e-3) {
		t.Errorf("CPU/GPU disagreement: %+v", apart)
	}
}
