package gpu

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// brokenTube is a tube with no length, which the GPU path cannot mesh.
var brokenTube = shape.Tube{Radius: .02, Thick: .004, Height: 0}

// shiningThing is a thing whose material the GPU path refuses for what
// it is, not for its shape.
func shiningThing(name string) thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: name, IndependentEmission: true}}
}

// A world with several things the GPU path refuses has each of them
// listed, by place, kind and material, not only the first.
func TestRefusalsListEveryThingNotTheFirst(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	world.Add(shiningThing("neon"))
	world.Add(thing.Thing{Shape: brokenTube})
	world.Add(shiningThing("sign"))
	found := Refusals(world, 0)
	if len(found) != 3 {
		t.Fatalf("%d refusals, want 3: %+v", len(found), found)
	}
	const shining = "independent surface emission is CPU-only"
	for position, want := range []Refusal{
		{partThing, 1, "shape.Sphere", "neon", shining},
		{partThing, 2, "shape.Tube", "", found[1].Reason},
		{partThing, 3, "shape.Sphere", "sign", shining},
	} {
		if found[position] != want {
			t.Errorf("refusal %d is %+v, want %+v", position, found[position], want)
		}
	}
	if !strings.HasPrefix(found[1].Reason, "a tube that cannot be meshed") {
		t.Errorf("the tube's reason is %q", found[1].Reason)
	}
}

// The world's own refusals come first, as the encoder meets them first,
// and the first of them is the one the drawing path stops at.
func TestRefusalsBeginWithWhatTheEncoderStopsAt(t *testing.T) {
	world := thing.NewWorld()
	world.Haze.Density = .01
	world.Add(shiningThing("neon"))
	found := Refusals(world, 0)
	if len(found) != 2 || found[0].Part != partWorld || found[1].Part != partThing {
		t.Fatalf("refusals %+v, want the world's and then the thing's", found)
	}
	_, err := encode(world, maths.Subspace{}, nil, 0, 8, 8)
	var refused unsupported
	if !errors.As(err, &refused) || refused.what != found[0].Reason {
		t.Errorf("the encoder stopped at %v, the first refusal is %q",
			err, found[0].Reason)
	}
}

// A world the encoder takes has no refusals, and a world it refuses has
// at least one: the list and the encoder do not disagree.
func TestRefusalsAreEmptyExactlyWhenTheEncoderTakesTheWorld(t *testing.T) {
	clean := thing.NewWorld()
	clean.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	clean.Add(thing.Thing{Shape: shape.Tube{Radius: .02, Thick: .004, Height: 1}})
	if _, err := encode(clean, maths.Subspace{}, nil, 0, 8, 8); err != nil {
		t.Fatal(err)
	}
	if found := Refusals(clean, 0); len(found) != 0 {
		t.Errorf("a world the encoder takes has refusals: %+v", found)
	}
	refused := thing.NewWorld()
	refused.Add(shiningThing("neon"))
	if _, err := encode(refused, maths.Subspace{}, nil, 0, 8, 8); err == nil {
		t.Fatal("the encoder took a thing it refuses")
	}
	if found := Refusals(refused, 0); len(found) == 0 {
		t.Error("a world the encoder refuses has no refusals")
	}
}

// A hidden thing is not placed, so it is not refused; listing does not
// change the world, so listing twice says the same.
func TestRefusalsSkipTheHiddenAndChangeNothing(t *testing.T) {
	world := thing.NewWorld()
	hidden := shiningThing("neon")
	hidden.Attrs.Hidden = true
	world.Add(hidden)
	world.Add(thing.Thing{Shape: brokenTube})
	first, second := Refusals(world, 0), Refusals(world, 0)
	if len(first) != 1 || first[0].Index != 1 {
		t.Fatalf("refusals %+v, want the tube at place 1 alone", first)
	}
	if len(second) != 1 || second[0] != first[0] {
		t.Errorf("a second listing is %+v, the first was %+v", second, first)
	}
}

// reasonOf is what Refusals groups on; an unsupported error gives its
// words and no other error is trimmed.
func TestReasonOfUnsupportedAndOtherErrors(t *testing.T) {
	if got := reasonOf(unsupported{"a thing"}); got != "a thing" {
		t.Errorf("reason %q", got)
	}
	if got := reasonOf(errors.New("gpu: something else")); got != "gpu: something else" {
		t.Errorf("reason %q", got)
	}
}
