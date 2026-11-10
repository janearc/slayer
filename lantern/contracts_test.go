package lantern

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

func TestLanternPlacesLayeredWornPanelsAndIndependentBeams(t *testing.T) {
	clock := time.Second
	polar := emit.Polar{Axis: maths.V(0, 1, 0), Width: .2, Soft: .1}
	lamp := Lantern{At: maths.V(2, 3, 4), Radius: 1, Orbits: 2, Panels: 3, Kelvin: 4800, Green: 38, Turn: time.Minute, Wide: .4, Wear: .8, Skin: BlackGlass(1.5), Lining: Matte(1.5), Beams: &polar, Clock: &clock, Glare: 4, Shells: Shells{Spread: 2, Narrowest: .2, Fastest: 15 * time.Second, Tumble: time.Minute}}
	world := thing.NewWorld()
	count, err := lamp.Place(world)
	if err != nil {
		t.Fatal(err)
	}
	if count != 6 || len(world.Things) != 5 || len(world.Lamps) != 2 {
		t.Fatalf("%d %d %d", count, len(world.Things), len(world.Lamps))
	}
	if polar.At != (maths.Vec{}) || polar.Beyond != 0 {
		t.Fatal("caller beam recipe changed")
	}
	for _, part := range world.Things[1:] {
		panels, ok := part.Shape.(*mesh.Mesh)
		if !ok || panels.FaceCount() <= 6 {
			t.Fatal("wear did not cut panels")
		}
		if part.Motion.At(time.Second).Origin != lamp.At {
			t.Fatal("ring lost star origin")
		}
	}
	light := lamp.light().When(0)
	if math.Abs(light.Strength.G-38) > 1e-9 || light.Beyond != 1.02 {
		t.Fatalf("star light %+v", light)
	}
	lamp.Orbits = 1
	out, wide, turn := lamp.shelled(0)
	if out != 6 || wide != .4 || turn != time.Minute || lamp.quickened(5*time.Second, 0) != 5*time.Second {
		t.Fatal("single ring changed")
	}
	lamp.Orbits = 3
	out, wide, turn = lamp.shelled(2)
	if out != 10 || math.Abs(wide-.2) > 1e-12 || turn != 15*time.Second || lamp.quickened(time.Minute, 2) != 15*time.Second {
		t.Fatal("outer shell endpoints")
	}
}

func TestLanternDefaultsAndRefusesAnEmptyPanelMesh(t *testing.T) {
	lamp := Lantern{Radius: 1, Orbits: 1, Panels: 3, Kelvin: 4800, Green: 1}
	world := thing.NewWorld()
	if _, err := lamp.Place(world); err != nil {
		t.Fatal(err)
	}
	if world.Things[1].Material != mirrored {
		t.Fatal("default panel material")
	}
	if _, width, _ := lamp.shelled(0); width != .4 {
		t.Fatal("default width")
	}
	lamp.Panels = 0
	if _, err := lamp.Place(thing.NewWorld()); err == nil {
		t.Fatal("empty mesh accepted")
	}
	if BlackGlass(0) != nil || Matte(0) != nil || BlackGlass(1.5).Index != 1.5 || Matte(1.5).Gloss != 0 {
		t.Fatal("glass or lining defaults")
	}
}

func TestWornPanelHasSharedVerticesAndLessMetal(t *testing.T) {
	square := [4]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(1, 1, 0), maths.V(0, 1, 0)}
	points, faces := worn(square, maths.V(0, 0, 1), 17, 1)
	again, cut := worn(square, maths.V(0, 0, 1), 17, 1)
	if !reflect.DeepEqual(points, again) || !reflect.DeepEqual(faces, cut) {
		t.Fatal("wear is not repeatable")
	}
	area := 0.0
	for _, f := range faces {
		area += points[f[1]].Sub(points[f[0]]).Cross(points[f[2]].Sub(points[f[0]])).Len() / 2
	}
	if !(area > .1 && area < .99) {
		t.Fatalf("worn area %g", area)
	}
	plate := plate{square: square, out: maths.V(0, 0, 1), seed: 17, wear: 1}
	a, b := plate.point(0, .5), plate.point(.5, .5)
	one, two := plate.crossing(a, b), plate.crossing(b, a)
	if one != two || plate.point(0, .5) != a {
		t.Fatal("shared edge has duplicate vertices")
	}
	inward := facingInAll(points, faces, maths.V(0, 0, 1))
	for _, f := range inward {
		normal := points[f[1]].Sub(points[f[0]]).Cross(points[f[2]].Sub(points[f[0]]))
		if normal.Z > 0 {
			t.Fatal("worn face points away from star")
		}
	}
}
