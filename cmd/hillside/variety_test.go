package main

import (
	"errors"
	"io"
	"reflect"
	"strconv"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// A thing made once and stood at a place is the thing made there, to
// the bit: the mesh does not depend on where it stands.
func TestAKeptThingStandsAsOneMadeThere(t *testing.T) {
	row, err := manufacture.KindNamed("spruce")
	if err != nil {
		t.Fatal(err)
	}
	one, err := manufacture.Make(row, manufacture.Sow(7))
	if err != nil {
		t.Fatal(err)
	}
	at := maths.Subspace{Origin: maths.V(3, 0, -2),
		Orientation: maths.About(maths.YAxis, 1), Scale: 1.3}
	want, errThere := one.Things(at)
	kept, errHere := one.Things(maths.Subspace{})
	if errThere != nil || errHere != nil {
		t.Fatal(errThere, errHere)
	}
	if got := stood(kept, at); !reflect.DeepEqual(got, want) {
		t.Error("a kept spruce stood at a place is not one made there")
	}
}

// meshesOf is the distinct meshes of a world's spruce and the faces
// drawn of them, counting each time one stands.
func meshesOf(world *thing.World) (map[*mesh.Mesh]bool, int) {
	meshes, drawn := map[*mesh.Mesh]bool{}, 0
	for _, each := range world.Things {
		skin, isMesh := each.Shape.(*mesh.Mesh)
		if !isMesh || each.Material == nil {
			continue
		}
		if name := each.Material.Name; name == "bark" || name == "needles" {
			meshes[skin] = true
			drawn += skin.FaceCount()
		}
	}
	return meshes, drawn
}

// thirty is a hillside of thirty spruce and nothing else, of a variety.
func thirty(t *testing.T, variety string,
	more ...string) (*thing.World, tally, error) {
	t.Helper()
	arguments := append(only(map[string]string{"spruce": "30"}),
		"-variety", variety)
	return hillside(mustParse(t, append(arguments, more...)...))
}

// A hillside of a variety makes that many of a kind and stands them
// at every place: thirty spruce of three are three spruce's meshes,
// whose faces are kept once and drawn thirty times.
func TestAVarietyIsMadeOnceAndStoodOften(t *testing.T) {
	world, made, err := thirty(t, "3")
	if err != nil {
		t.Fatal(err)
	}
	meshes, drawn := meshesOf(world)
	kept := 0
	for skin := range meshes {
		kept += skin.FaceCount()
	}
	if len(meshes) > 2*3 || made.things != 30 {
		t.Errorf("thirty spruce of three are %d things of %d meshes",
			made.things, len(meshes))
	}
	if made.faces != drawn || made.kept != kept || kept >= drawn {
		t.Errorf("%d faces drawn and %d kept; the meshes are %d drawn "+
			"and %d kept", made.faces, made.kept, drawn, kept)
	}
}

// With no variety every thing is its own: no two spruce share a mesh,
// and every face drawn is kept.
func TestNoVarietyIsEveryThingItsOwn(t *testing.T) {
	world, made, err := thirty(t, "0")
	if err != nil {
		t.Fatal(err)
	}
	if meshes, _ := meshesOf(world); len(meshes) != 2*30 ||
		made.kept != made.faces {
		t.Errorf("thirty spruce of their own are %d meshes, %d faces "+
			"kept of %d", len(meshes), made.kept, made.faces)
	}
}

// The limit on faces is on those kept, which is what costs memory:
// thirty spruce of two fit under a limit of two spruce's faces, and
// thirty of their own do not.
func TestTheLimitIsOnFacesKept(t *testing.T) {
	_, made, err := thirty(t, "2")
	if err != nil {
		t.Fatal(err)
	}
	limit := []string{"-faces", strconv.Itoa(made.kept)}
	if _, _, err := thirty(t, "2", limit...); err != nil {
		t.Errorf("two spruce's faces would not hold thirty of two: %v",
			err)
	}
	if _, _, err := thirty(t, "0", limit...); !errors.Is(err,
		errTooMany) {
		t.Errorf("two spruce's faces held thirty of their own: %v", err)
	}
}

// A variety less than zero is refused at the flag.
func TestAVarietyBelowNoughtIsRefused(t *testing.T) {
	_, err := parse([]string{"-variety", "-1"}, io.Discard)
	if !errors.Is(err, errAsked) {
		t.Errorf("a variety of -1 gave %v", err)
	}
}

// A tally of shared things says how many faces it keeps beside how many
// it draws.
func TestTheTallySaysWhatItKeeps(t *testing.T) {
	said := tally{things: 30, faces: 300, kept: 20, lamps: 2}.String()
	if said != "30 things of 300 faces (20 kept), lit by 2 lamps" {
		t.Errorf("it says %q", said)
	}
}
