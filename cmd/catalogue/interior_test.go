package main

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"os"
	"testing"
)

// TestInteriorComparisonKeepsCameraAndAir checks the actual checked pair
// through scene construction, including the full shell and internal eye.
func TestInteriorComparisonKeepsCameraAndAir(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{"keck-i"})
	if err != nil {
		t.Fatal(err)
	}
	var previous *specimen
	for _, state := range []string{"off", "on"} {
		path := "../../improv/searching/studies/keck-i-interior-" + state + ".yaml"
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := readPlan(path, raw)
		if err != nil {
			t.Fatal(err)
		}
		object, err := plan.construct(entries[0], "quick")
		if err != nil {
			t.Fatal(err)
		}
		if object.cutaway {
			t.Fatal("interior removed enclosure")
		}
		if object.eye.Out(maths.Vec{}) != maths.V(6, 10, -9) {
			t.Fatal("entry framing replaced interior camera", object.eye)
		}
		if _, ok := object.world.Sky.(sky.Stars); !ok {
			t.Fatal("missing stars")
		}
		if len(object.world.Volumes) != 1 {
			t.Fatal("missing shared air")
		}
		beams := 0
		for _, lamp := range object.world.Lamps {
			if _, ok := lamp.(*emit.Beam); ok {
				beams++
			}
		}
		if state == "off" && beams != 0 || state == "on" && beams != 2 {
			t.Fatal("incorrect beam count", beams)
		}
		if previous != nil && (*previous.eye != *object.eye || len(previous.world.Things) != len(object.world.Things)) {
			t.Fatal("comparison changed view or geometry")
		}
		previous = object
	}
}

// TestInteriorBeamRefusesDirection checks a degenerate source before a draw.
func TestInteriorBeamRefusesDirection(t *testing.T) {
	p := plan{Interior: &interiorPlan{Beams: []interiorBeam{{Radius: .15, Reach: 500}}}}
	if p.interiorSky(thing.NewWorld()) == nil {
		t.Fatal("accepted zero direction")
	}
}

// TestInteriorCamerasClearSolidGeometry checks the saved viewpoints against
// every physical mesh rather than a rectangular building envelope.
func TestInteriorCamerasClearSolidGeometry(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"keck-i", "keck-ii", "subaru", "rubin", "elt"} {
		path := "../../improv/searching/studies/" + kind + "-interior-off.yaml"
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err := readPlan(path, raw)
		if err != nil {
			t.Fatal(err)
		}
		entries, err := menu.selectEntries([]string{kind})
		if err != nil {
			t.Fatal(err)
		}
		object, err := p.construct(entries[0], "demo")
		if err != nil {
			t.Fatal(err)
		}
		for _, part := range object.world.Things {
			if distance := part.Shape.Distance(p.Interior.At); distance < 0.1 {
				t.Errorf("%s camera intersects %s: %g", kind, part.Material.Name, distance)
			}
		}
	}
}

// TestInteriorAirDoesNotCountAsTheSpecimen permits attenuated sky misses
// while still requiring finite-depth geometry and refusing invalid hits.
func TestInteriorAirDoesNotCountAsTheSpecimen(t *testing.T) {
	p := defaultPlan(t)
	p.MinCoverage = 0.5
	picture := frame.New(2, 1)
	picture.Pix[0] = frame.Pixel{Cover: 1, Depth: 2}
	picture.Pix[1].Cover = 0.1
	if p.visible(picture) == nil {
		t.Fatal("surface-only plan accepted covered miss")
	}
	p.Interior = &interiorPlan{Extinction: 0.00012}
	if err := p.visible(picture); err != nil {
		t.Fatal(err)
	}
	picture.Pix[0].Depth = -1
	if p.visible(picture) == nil {
		t.Fatal("accepted negative hit depth")
	}
	picture = frame.New(2, 1)
	picture.Pix[0].Cover = 0.1
	picture.Pix[1].Cover = 0.1
	if p.visible(picture) == nil {
		t.Fatal("air alone counted as specimen")
	}
}

// TestInteriorBeamAxesLeaveThroughOpenings prevents a launcher being moved
// behind a mirror, brace or closed shell while the render still succeeds.
func TestInteriorBeamAxesLeaveThroughOpenings(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"keck-i", "keck-ii", "subaru", "rubin", "elt"} {
		path := "../../improv/searching/studies/" + kind + "-interior-on.yaml"
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err := readPlan(path, raw)
		if err != nil {
			t.Fatal(err)
		}
		entries, err := menu.selectEntries([]string{kind})
		if err != nil {
			t.Fatal(err)
		}
		object, err := p.construct(entries[0], "demo")
		if err != nil {
			t.Fatal(err)
		}
		for _, beam := range p.Interior.Beams {
			ray := maths.Ray{At: beam.At, Dir: beam.Toward.Unit()}
			for _, part := range object.world.Things {
				h, ok := part.Shape.(shape.Hitter)
				if !ok {
					t.Fatalf("unprobed geometry %T", part.Shape)
				}
				if distance, hit := h.Hit(ray, 0.001, beam.Reach); hit {
					t.Errorf("%s beam blocked by %s at %gm", kind, part.Material.Name, distance)
				}
			}
		}
	}
}
