package main

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

func TestMoonlightCanOmitItsOffscreenDiscWithoutLosingTheLight(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "  light: none",
		"  light: moon\n  moon: {azimuth: 115, altitude: 28, disc: false}", 1)
	if err = os.WriteFile(given.set, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if world.Sky == nil {
		t.Fatal("moonlit sky was lost")
	}
	distant := 0
	for _, lamp := range world.Lamps {
		if _, ok := lamp.(emit.Distant); ok {
			distant++
		}
	}
	if distant != 1 {
		t.Fatal("physical moonlight was lost")
	}
	for _, surface := range world.Things {
		if surface.Material != nil && surface.Material.Name == "moon" {
			t.Fatal("hidden lunar disc still affects scene geometry")
		}
	}
	if slayer.Spread(world, 0) > 1e4 {
		t.Fatal("lunar distance corrupted terrain ray offsets")
	}
}

func moonDiscLightFixture(t *testing.T) (*thing.World, maths.Subspace, film.Pinhole) {
	t.Helper()
	given := mustParse(t)
	given.set = onFixture(t, "  terrain:", "  moon: {azimuth: 115, altitude: 28}\n  terrain:")
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	isolated := thing.NewWorld()
	for _, surface := range world.Things {
		if surface.Material != nil && surface.Material.Name == "moon" {
			isolated.Add(*surface)
		}
	}
	if len(isolated.Things) != 1 {
		t.Fatal("fixture lost lunar geometry")
	}
	for _, lamp := range world.Lamps {
		if _, ok := lamp.(emit.Distant); ok {
			isolated.Lamp(lamp)
		}
	}
	if len(isolated.Lamps) != 1 {
		t.Fatal("fixture lost lunar light")
	}
	isolated.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Colour: maths.Grey(1)}})
	from := maths.V(0, 3, 0)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAtUp(from, maths.Vec{}, maths.ZAxis)}
	lens := film.Pinhole{Wide: 40 * maths.Degree}
	return isolated, eye, lens
}

func TestVisibleMoonDoesNotBlockItsOwnDistantLight(t *testing.T) {
	isolated, eye, lens := moonDiscLightFixture(t)
	first, err := slayer.Render(isolated, eye, lens, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	lit := first.Pix[0]
	if lit.R <= 0 || lit.G <= 0 || lit.B <= 0 {
		t.Fatal("Moon blocks its own illumination", lit)
	}
	isolated.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}, Material: &thing.Material{}, Motion: thing.Still(maths.Subspace{Origin: skyward(115, 28).Scale(10)})})
	second, err := slayer.Render(isolated, eye, lens, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	shaded := second.Pix[0]
	if shaded.R != 0 || shaded.G != 0 || shaded.B != 0 {
		t.Fatal("ordinary occluder no longer shades moonlight", shaded)
	}
}
