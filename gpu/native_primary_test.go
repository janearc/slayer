//go:build darwin

package gpu

import (
	"errors"
	"image"
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The still budget is the 2026-10-14 agreement: at most two
// percent of developed pixels beyond one JND. Report worst error separately.
func perceptualDifference(one, two image.Image) (float64, float64) {
	count, beyond, worst := 0, 0, 0.0
	for y := one.Bounds().Min.Y; y < one.Bounds().Max.Y; y++ {
		for x := one.Bounds().Min.X; x < one.Bounds().Max.X; x++ {
			lab := func(img image.Image) ok.OKLab {
				r, g, b, _ := img.At(x, y).RGBA()
				return ok.FromSwatch(srgb.RGB8(uint8(r>>8), uint8(g>>8), uint8(b>>8)).Swatch())
			}
			apart := ok.Distance(lab(one), lab(two)) / ok.Eye
			if apart > 1 {
				beyond++
			}
			worst = max(worst, apart)
			count++
		}
	}
	return float64(beyond) / float64(count), worst
}

func acceleratedWithinBudget(t *testing.T, world *thing.World, eye maths.Subspace) {
	t.Helper()
	const width, height = 96, 64
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	truth, err := Sampled(world, eye, lens, 0, width, height, 2)
	if err != nil {
		t.Fatal(err)
	}
	drawn, costs, err := SampledAccelerated(world, eye, lens, 0, width, height, 2)
	if err != nil {
		t.Fatal(err)
	}
	if costs.AccelerationWall <= 0 || costs.AccelerationGPU <= 0 {
		t.Fatal("native construction was not measured")
	}
	if costs.Native.Refined == 0 ||
		costs.Native.Refined+costs.Native.Fallbacks != costs.Native.Candidates {
		t.Fatalf("native replies were hidden by fallback: %+v", costs.Native)
	}
	t.Logf("native outcomes: %+v", costs.Native)
	covered := 0
	for i, p := range drawn.Pix {
		if math.IsNaN(float64(p.R+p.G+p.B+p.Cover)) || math.IsInf(float64(p.R+p.G+p.B), 0) {
			t.Fatalf("nonfinite pixel %d", i)
		}
		if p.Cover < 0 || p.Cover > 1 || p.Depth < 0 {
			t.Fatalf("invalid coverage/depth at %d", i)
		}
		if p.Cover > 0 {
			covered++
		}
	}
	if covered < 5 {
		t.Fatalf("fixture drew only %d surface pixels", covered)
	}
	tone := colour.Tone{Knee: 0.7, Top: 8}
	share, worst := perceptualDifference(tone.Develop(truth), tone.Develop(drawn))
	t.Logf("%.6f%% past one JND; worst %.6f JND; GPU %v", share*100, worst, costs.TraceGPU)
	if share > 0.02 {
		t.Fatalf("outside still budget: %.6f%% past one JND", share*100)
	}
}

func TestNativePrimaryPreservesShadingAndTextures(t *testing.T) {
	for name, fixture := range map[string]func(*testing.T) (*thing.World, maths.Subspace){
		"hall": hall, "textures": gallery, "smooth": lowLit, "nested": nativePlacedFixture,
	} {
		t.Run(name, func(t *testing.T) { world, eye := fixture(t); acceleratedWithinBudget(t, world, eye) })
	}
}

// Out-of-order placements force global and local trees to reorder IDs. Each
// world instance has its own material; local members share only geometry.
func nativePlacedFixture(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	group, triangle := groupFixture(t)
	world := thing.NewWorld()
	for i, x := range []float64{8, -8, 4, -4, 0} {
		material := &thing.Material{Colour: maths.RGB(float64(i+1)/6, 0.6, float64(6-i)/7), Shine: banded{}}
		solid := shape.Shape(group)
		if i%2 == 1 {
			solid = triangle
		}
		space := maths.Subspace{Origin: maths.V(x, -0.5, 0), Scale: 0.6 + float64(i)*0.1,
			Orientation: maths.About(maths.YAxis, float64(i)*0.15)}
		world.Add(thing.Thing{Shape: solid, Material: material, Motion: thing.Still(space), Attrs: thing.Attrs{Clear: i == 2}})
	}
	lamp, err := emit.NewDistant(maths.V(-1, 2, 3), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(lamp)
	from := maths.V(0, 5, 24)
	return world, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(1, 0, 0))}
}

func TestNativePrimaryRefusesUnsupportedScenes(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}})
	picture, _, err := SampledAccelerated(world, maths.Subspace{}, film.Pinhole{}, 0, 4, 4, 1)
	var why unsupported
	if picture != nil || !errors.As(err, &why) {
		t.Fatalf("unsupported scene was not refused: %v", err)
	}
}

func TestNativePrimaryFoliageAcrossCameraMovement(t *testing.T) {
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
			for _, one := range things {
				world.Add(one)
			}
			lamp, err := emit.NewDistant(maths.V(-1, 2, 3), maths.Grey(1))
			if err != nil {
				t.Fatal(err)
			}
			world.Lamp(lamp)
			height := made.Bounds().Max.Y
			for i := range 3 {
				from := maths.V(height*(0.85+float64(i)*0.15), height*0.8, height*1.8)
				eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, height*0.5, 0))}
				acceleratedWithinBudget(t, world, eye)
			}
		})
	}
}

// A whole-image error budget can hide a wrong material on a small object.
// Fully covered pixels of distinct emissive instances check that contract
// directly, without lighting or background light masking an ID mistake.
func TestNativePrimaryKeepsPerInstanceMaterials(t *testing.T) {
	group, triangle := groupFixture(t)
	world := thing.NewWorld()
	colors := []maths.Light{maths.RGB(0.9, 0.1, 0.2), maths.RGB(0.1, 0.8, 0.2),
		maths.RGB(0.2, 0.1, 0.9), maths.RGB(0.8, 0.7, 0.1), maths.RGB(0.1, 0.7, 0.8)}
	for i, x := range []float64{8, -8, 4, -4, 0} {
		solid := shape.Shape(group)
		if i%2 == 1 {
			solid = triangle
		}
		world.Add(thing.Thing{Shape: solid, Material: &thing.Material{Glow: colors[i]},
			Motion: at(x, 0, 0)})
	}
	from := maths.V(0, 3, 28)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(2, 0, 0))}
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	truth, err := Sampled(world, eye, lens, 0, 192, 128, 2)
	if err != nil {
		t.Fatal(err)
	}
	picture, profile, err := SampledAccelerated(world, eye, lens, 0, 192, 128, 2)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Native.Refined == 0 || profile.Native.Fallbacks != 0 {
		t.Fatalf("material fixture did not use native replies: %+v", profile.Native)
	}
	counts := make([]int, len(colors))
	for i, p := range truth.Pix {
		if p.Cover != 1 {
			continue
		}
		for c, light := range colors {
			if math.Abs(float64(p.R)-light.R)+math.Abs(float64(p.G)-light.G)+math.Abs(float64(p.B)-light.B) > 1e-5 {
				continue
			}
			counts[c]++
			other := picture.Pix[i]
			difference := math.Abs(float64(other.R-p.R)) + math.Abs(float64(other.G-p.G)) + math.Abs(float64(other.B-p.B))
			if difference > 1e-4 {
				t.Fatalf("instance material changed at pixel %d: difference %g", i, difference)
			}
		}
	}
	for i, count := range counts {
		if count < 3 {
			t.Fatalf("material %d has only %d pure surface pixels", i, count)
		}
	}
}
