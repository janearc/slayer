package cities

import (
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/texture"
)

func TestAtTheDemoLevelWallsArePlasterAndShuttersAndDoorsAreTimber(t *testing.T) {
	drawn := terraceAt(t, "portofino", 3, "demo")
	plaster, timber := 0, 0
	for _, one := range drawn {
		switch one.Material.Texture.(type) {
		case texture.Dusted:
			plaster++
			if !slices.Contains([]string{"wall", "cornice", "chimney", "balcony", "spandrel"},
				one.Mesh.Name) {
				t.Errorf("plaster on a %s", one.Mesh.Name)
			}
		case texture.Wood:
			timber++
			if !strings.HasSuffix(one.Material.Name, "timber") {
				t.Errorf("timber named %q", one.Material.Name)
			}
		}
	}
	if plaster == 0 || timber == 0 {
		t.Errorf("%d plaster parts and %d timber parts", plaster, timber)
	}
}

func TestPlasterHasABumpAndTheRevealIsDeeperThanTheFrame(t *testing.T) {
	fine := drawerOf(t, "portofino", 1).finish()
	drawn := terraceAt(t, "portofino", 3, "demo")
	bumped := 0
	for _, one := range drawn {
		if _, plaster := one.Material.Texture.(texture.Dusted); plaster &&
			one.Material.Crinkle != nil && one.Material.Crinkle.Slope > 0 {
			bumped++
		}
	}
	if bumped == 0 {
		t.Error("no plaster has a bump")
	}
	reveal := drawerOf(t, "portofino", 1).fixed("reveal-depth")
	for _, frame := range named(drawn, "frame") {
		low, high := reach(frame)
		if high.Z-low.Z > fine.FrameDepth+tolerance {
			t.Fatalf("a frame bar %v deep, set back %v", high.Z-low.Z,
				fine.FrameDepth)
		}
	}
	_, topFront := reach(named(drawn, "cornice")[0])
	front := topFront.Z - fine.CorniceProud
	for _, light := range named(drawn, "light") {
		if low, _ := reach(light); low.Z > front-reveal+tolerance {
			t.Fatalf("a light at z %v, the face at %v, the reveal %v deep",
				low.Z, front, reveal)
		}
	}
	for _, one := range terraceAt(t, "portofino", 3, "quick") {
		if one.Material.Crinkle != nil {
			t.Fatal("a quick surface has a bump")
		}
	}
}

func TestBelowTheDemoLevelNoSurfaceHasATexture(t *testing.T) {
	for _, level := range []Detail{"hull", "quick"} {
		for _, one := range terraceAt(t, "portofino", 3, level) {
			if one.Material.Texture != nil {
				t.Fatalf("a %s at %s has a texture", one.Mesh.Name, level)
			}
		}
	}
}

func TestPlasterIsTheWallsColourWithADarkerStain(t *testing.T) {
	for _, one := range terraceAt(t, "portofino", 3, "demo") {
		dusted, plastered := one.Material.Texture.(texture.Dusted)
		if !plastered {
			continue
		}
		if dusted.Rock != one.Material.Colour ||
			!(dusted.Dust.R < dusted.Rock.R) {
			t.Fatalf("plaster %+v on a wall of colour %v", dusted,
				one.Material.Colour)
		}
		return
	}
	t.Error("no plaster at the demo level")
}
