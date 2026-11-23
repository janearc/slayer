package session

import (
	"errors"
	"io"
	"reflect"
	"strconv"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
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

// meshesOf is the distinct meshes of a world's spruce, each with the
// faces it keeps, and the faces drawn of them, counting each time one
// stands. A shared mesh group keeps its prototypes and draws every
// standing, as the stock accounts them.
func meshesOf(world *thing.World) (map[shape.Shape]int, int) {
	meshes, drawn := map[shape.Shape]int{}, 0
	for _, each := range world.Things {
		if each.Material == nil {
			continue
		}
		if name := each.Material.Name; name != "bark" && name != "needles" {
			continue
		}
		switch skin := each.Shape.(type) {
		case *mesh.Mesh:
			meshes[skin] = skin.FaceCount()
			drawn += skin.FaceCount()
		case *shape.MeshGroup:
			meshes[skin] = skin.StoredFaces()
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
	given := mustParse(t, append(arguments, more...)...)
	return hillside(given)
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
	for _, stored := range meshes {
		kept += stored
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
// and every spruce's stored faces are kept, once each. A spruce's own
// needle prototypes still repeat within it, so the faces drawn exceed
// those kept even unshared.
func TestNoVarietyIsEveryThingItsOwn(t *testing.T) {
	world, made, err := thirty(t, "0")
	if err != nil {
		t.Fatal(err)
	}
	meshes, drawn := meshesOf(world)
	kept := 0
	for _, stored := range meshes {
		kept += stored
	}
	if len(meshes) != 2*30 || made.kept != kept || made.faces != drawn {
		t.Errorf("thirty spruce of their own are %d meshes keeping %d "+
			"faces of %d drawn; the tally says %d kept of %d",
			len(meshes), kept, drawn, made.kept, made.faces)
	}
}

// pastTheOldCount is a count of kept faces, 3,846,153, that the tests
// below set a scene against: no scene is refused for how many faces it
// keeps, past it or under it.
const pastTheOldCount = 3_846_153

// manySpruce is how many spruce of their own the scene past the old count
// is made of. Thirty keep 2,925,924 faces now that a spruce's needles are
// physical shoots, measured, under the old count; sixty keep twice that.
const manySpruce = 60

// A scene that keeps more than three and a half million faces is
// staged, not refused: sixty needled spruce of their own keep more.
func TestASceneOfManyKeptFacesIsStaged(t *testing.T) {
	spruces := strconv.Itoa(manySpruce)
	arguments := append(only(map[string]string{
		"spruce": spruces}), "-variety", "0")
	world, made, err := hillside(mustParse(t, arguments...))
	if err != nil {
		t.Fatalf("%d spruce of their own were refused: %v", manySpruce, err)
	}
	if made.kept <= pastTheOldCount || made.things != manySpruce ||
		len(world.Things) == 0 {
		t.Errorf("%d things keeping %d faces, in %d parts; want %d "+
			"things keeping more than %d", made.things, made.kept,
			len(world.Things), manySpruce, pastTheOldCount)
	}
}

// The count is of faces kept, which is what costs memory: thirty spruce
// of two keep far fewer than thirty of their own, and are staged too.
func TestASceneOfFewKeptFacesIsStagedToo(t *testing.T) {
	_, shared, err := thirty(t, "2")
	if err != nil {
		t.Fatal(err)
	}
	if shared.kept == 0 || shared.kept >= pastTheOldCount {
		t.Errorf("thirty spruce of two keep %d faces; want some, and "+
			"fewer than %d", shared.kept, pastTheOldCount)
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
