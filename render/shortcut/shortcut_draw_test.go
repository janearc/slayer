//go:build draw

package shortcut

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// render is trace.Render for a test: an error fails the test.
func render(t *testing.T, w *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height int) *frame.Frame {
	t.Helper()
	f, err := trace.Render(w, eye, lens, at, width, height)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// smoke is a puff of smoke on the chalk floor under the sun, the set
// of smoke nine.
func smoke() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "chalk", Colour: swatches.Light(swatches.Chalk)}})
	w.AddVolume(thing.Volume{
		Medium:   medium.Wispy{Of: medium.Puff{Radius: 1.5, Thick: 1.2}, Seed: 3, Scale: 0.5, Amount: 0.6},
		Material: &thing.Material{Name: "smoke", Colour: maths.Grey(0.8)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 1.5, 0)}),
	})
	w.Lamp(thing.Sun(maths.V(1, 2, 1), swatches.Light(swatches.Daylight)))
	from := maths.V(0, 2.5, 7)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 1.2, 0))}
}

// A set parses as it prints, both ways, and refuses nonsense; a plan
// records the global set and each thing's own; with no shortcut the
// applied world renders byte for byte the exact one; baked noise at
// the default resolution is within an eye of exact on the smoke.
func TestShortcutsAreDeclaredAndHonest(t *testing.T) {
	for _, text := range []string{"none", "all", "baked", "baked(32) skip", "distance once"} {
		s, err := Parse(text)
		if err != nil {
			t.Fatal(err)
		}
		if s.String() != text {
			t.Errorf("%q parses and prints as %q", text, s)
		}
	}
	if _, err := Parse("baked(1)"); err == nil {
		t.Error("baked(1) passed")
	}
	if _, err := Parse("fast"); err == nil || !strings.Contains(err.Error(), "none of") {
		t.Errorf("fast gave %v", err)
	}
	p := Plan{Global: All()}.With("smoke", None()).With("rings", Set{Baked: true, Resolution: 64})
	if r := p.Record(); len(r) != 3 || r[0] != "global: all" || r[1] != "rings: baked(64)" || r[2] != "smoke: none" {
		t.Errorf("the record is %v", r)
	}
	if p.For("clouds") != All() || p.For("smoke") != None() {
		t.Errorf("the plan gives clouds %v and smoke %v", p.For("clouds"), p.For("smoke"))
	}
	w, eye := smoke()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	exact := render(t, w, eye, lens, 0, 96, 40)
	same := render(t, Apply(w, Plan{Global: None()}), eye, lens, 0, 96, 40)
	for y := 0; y < 40; y++ {
		for x := 0; x < 96; x++ {
			if exact.At(x, y) != same.At(x, y) {
				t.Fatalf("with no shortcut pixel %d,%d differs", x, y)
			}
		}
	}
	baked := Apply(w, Plan{Global: Set{Baked: true}})
	if _, ok := baked.Volumes[0].Medium.(*Baked); !ok {
		t.Fatalf("the smoke was not baked: %T", baked.Volumes[0].Medium)
	}
	if w.Volumes[0].Medium.(medium.Wispy).Of.(medium.Puff).Radius != 1.5 {
		t.Error("the exact world was changed")
	}
	taken := render(t, baked, eye, lens, 0, 96, 40)
	worst, mean := Error(exact, taken, colour.Tone{})
	if !Honest("baked", worst) {
		t.Errorf("baked noise is %.4f from exact at worst (mean %.4f); the budget is %.2f", worst, mean, Budgets["baked"])
	}
	// the bake is a real one: its density at the middle is not the
	// exact one's, though the picture does not show it
	if bk := baked.Volumes[0].Medium.(*Baked); bk.Density(maths.Vec{}) == w.Volumes[0].Medium.Density(maths.Vec{}) {
		t.Errorf("the baked density at the middle is exactly the exact one's: %v", bk.Density(maths.Vec{}))
	}
	_ = mean
	// a coarse bake is further off, and says so
	coarse := render(t, Apply(w, Plan{Global: Set{Baked: true, Resolution: 6}}), eye, lens, 0, 96, 40)
	if far, _ := Error(exact, coarse, colour.Tone{}); far <= worst {
		t.Errorf("a bake six cells a side is %.4f off, the default %.4f: coarser should be worse", far, worst)
	}
	// a baked medium ticks by baking again
	b := Bake(medium.Puff{Radius: 1, Thick: 1}, 8)
	if _, ok := b.Tick(time.Second).(*Baked); !ok {
		t.Error("a baked medium did not tick as one")
	}
}
