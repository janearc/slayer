package shortcut

import (
	"testing"

	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Apply bakes the volumes whose material's name the plan bakes, by the
// longest thing the name begins with, at that thing's resolution; one
// with no material takes the global, a volume with no medium is left
// alone, and the world given is not touched.
//
// four volumes and a plan with the global off, "smoke" baked at 3, and
// "smoke thin" baked at 5: a volume "smoke dense" takes smoke's 3; one
// "smoke thin haze" takes the 5; one "stone" is not baked; a nameless
// one is the global's, which is off.
func TestApplyBakesTheVolumesThePlanNames(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}})
	names := []string{"smoke dense", "smoke thin haze", "stone"}
	for _, n := range names {
		w.AddVolume(thing.Volume{Medium: ramp{}, Material: &thing.Material{Name: n}})
	}
	w.AddVolume(thing.Volume{Medium: ramp{}})
	plan := Plan{Global: None()}.
		With("smoke", Set{Baked: true, Resolution: 3}).
		With("smoke thin", Set{Baked: true, Resolution: 5})
	out := Apply(w, plan)
	if len(out.Volumes) != 4 || len(out.Things) != 1 {
		t.Fatalf("the world has %d volumes and %d things, want 4 and 1", len(out.Volumes), len(out.Things))
	}
	for i, want := range []int{3, 5, 0, 0} {
		b, baked := out.Volumes[i].Medium.(*Baked)
		switch {
		case want > 0 && (!baked || b.Resolution != want):
			t.Errorf("volume %d is %T, want a bake of %d", i, out.Volumes[i].Medium, want)
		case want == 0 && baked:
			t.Errorf("volume %d was baked and should not be", i)
		case want == 0 && out.Volumes[i] != w.Volumes[i]:
			t.Errorf("volume %d was not baked and was copied", i)
		}
	}
	for i, v := range w.Volumes {
		if v.Medium != (medium.Medium)(ramp{}) {
			t.Errorf("the exact world's volume %d became %T", i, v.Medium)
		}
	}
	if out == w || out.Volumes[0] == w.Volumes[0] {
		t.Error("Apply handed back the world it was given, or its volume")
	}
	if out.Volumes[0].Material != w.Volumes[0].Material {
		t.Error("the baked volume lost its material")
	}
}

// a volume with no material is the global's: with the global baked at 4
// it is baked, and a material of no name is too. a volume with a
// material but no medium is left as it was even when its name is baked.
func TestApplyTakesTheGlobalForANamelessVolume(t *testing.T) {
	w := thing.NewWorld()
	w.AddVolume(thing.Volume{Medium: ramp{}})
	w.AddVolume(thing.Volume{Medium: ramp{}, Material: &thing.Material{}})
	w.AddVolume(thing.Volume{Material: &thing.Material{Name: "smoke"}})
	out := Apply(w, Plan{Global: Set{Baked: true, Resolution: 4}})
	for i := 0; i < 2; i++ {
		if b, ok := out.Volumes[i].Medium.(*Baked); !ok || b.Resolution != 4 {
			t.Errorf("volume %d is %T, want a bake of 4", i, out.Volumes[i].Medium)
		}
	}
	if out.Volumes[2] != w.Volumes[2] || out.Volumes[2].Medium != nil {
		t.Errorf("a volume with no medium was changed: %+v", out.Volumes[2])
	}
}
