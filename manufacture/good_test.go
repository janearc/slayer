package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// what the things of these tests look like
var (
	bark    = &thing.Material{Name: "bark"}
	needle  = &thing.Material{Name: "needle"}
	granite = &thing.Material{Name: "granite"}
)

func mustGrow(t *testing.T, grown growth, length, girth float64) good {
	t.Helper()
	made, err := grow(grown, size{length: measure.Metres(length),
		girth: measure.Metres(girth)}, sow(5))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// facesOf is how many faces the parts of one matter have between them.
func facesOf(made good, of slot) int {
	faces := 0
	for _, one := range made.parts {
		if one.of == of {
			faces += len(one.skin())
		}
	}
	return faces
}

// a grown thing is handed to a scene as one mesh a matter, each of its
// own look, every face of every part in it, where it was asked to
// stand. slayer's mesh built each, so each is a surface a scene can hold.
func TestAGrownThingIsOneMeshAMatter(t *testing.T) {
	spruce, err := spruceGrowth(6, 5, spruceHabit())
	if err != nil {
		t.Fatal(err)
	}
	made := mustGrow(t, spruce, 6, 0.2)
	at := maths.Subspace{Origin: maths.V(3, 0, -4)}
	things, err := made.in(looks{body: bark,
		leaf: needle}).things(at)
	if err != nil {
		t.Fatal(err)
	}
	if len(things) != 2 {
		t.Fatalf("%d things, not one of bark and one of needles",
			len(things))
	}
	want := map[*thing.Material]int{bark: facesOf(made, body),
		needle: facesOf(made, leaf)}
	for _, one := range things {
		skin, ok := one.Shape.(*mesh.Mesh)
		if !ok {
			t.Fatalf("a thing of %s is a %T, not a mesh",
				one.Material.Name, one.Shape)
		}
		got, parts := skin.FaceCount(), want[one.Material]
		if got != parts || got == 0 {
			t.Errorf("%s: %d faces, its parts have %d",
				one.Material.Name, got, parts)
		}
		if one.At(0).Origin != at.Origin {
			t.Errorf("%s stands at %v, not %v", one.Material.Name,
				one.At(0).Origin, at.Origin)
		}
	}
	if things[0].Material != bark || things[1].Material != needle {
		t.Error("the things are not in the order of their matters")
	}
}

// a thing of one form is one thing: a stone, a can.
func TestAThingOfOneFormIsOneThing(t *testing.T) {
	fine := mustCut(t, 12)
	forms := map[string]form{
		"a stone": mustLump(t, 0.5, 0.4, 0.3, 0.2, 8),
		"a can":   mustTube(t, 0.122, 0.033, 0.033),
	}
	for name, shape := range forms {
		made, err := newGood(shape, fine)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		things, err := made.in(looks{body: granite}).things(
			maths.Subspace{})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(things) != 1 || things[0].Material != granite {
			t.Errorf("%s: %d things", name, len(things))
		}
	}
}

func TestAThingThatCannotBeMadeOrDrawnIsRefused(t *testing.T) {
	fine := mustCut(t, 6)
	if _, err := newGood(nil, fine); !errors.Is(err, errGood) {
		t.Errorf("a thing of no form: %v, not %v", err, errGood)
	}
	if _, err := newGood(mustTube(t, 1, 1, 1), cut{}); !errors.Is(err,
		errGood) {
		t.Errorf("a thing with no cut: %v, not %v", err, errGood)
	}
	spruce, err := spruceGrowth(6, 5, spruceHabit())
	if err != nil {
		t.Fatal(err)
	}
	stem := size{length: measure.Metres(6), girth: measure.Metres(0.2)}
	if _, err := grow(growth{}, stem, sow(1)); !errors.Is(err, errGood) {
		t.Errorf("a growth never made: %v, not %v", err, errGood)
	}
	if _, err := grow(spruce, stem, seed{}); !errors.Is(err, errGood) {
		t.Errorf("a seed never sown: %v, not %v", err, errGood)
	}
	if _, err := grow(spruce, size{}, sow(1)); !errors.Is(err,
		errTubeLength) {
		t.Errorf("a spruce of no size: %v, not %v", err, errTubeLength)
	}
	made := mustGrow(t, spruce, 6, 0.2)
	// a spruce has needles, and nobody said what needles look like
	if _, err := made.in(looks{body: bark}).things(
		maths.Subspace{}); !errors.Is(err, errLook) {
		t.Errorf("a matter with no look: %v, not %v", err, errLook)
	}
	if _, err := (good{}).things(maths.Subspace{}); !errors.Is(err,
		errGood) {
		t.Errorf("a thing of no parts: %v, not %v", err, errGood)
	}
}

// a thing of too many faces is refused before it is flattened, by
// name: flattening is what runs a machine out of memory. the bound is
// the most allowed, and one face more is refused.
func TestAThingOfTooManyFacesIsRefused(t *testing.T) {
	one, err := newGood(mustTube(t, 1, 0.1, 0.1), mustCut(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	// five sides round: ten faces of wall and ten of caps
	const each = 20
	if got := one.faces(); got != each {
		t.Fatalf("a tube of five sides has %d faces, not %d", got, each)
	}
	var most good
	for range mostFaces / each {
		most.parts = append(most.parts, one.parts[0])
	}
	if got := most.faces(); got != mostFaces {
		t.Fatalf("the test's own count is off: %d", got)
	}
	// so many of the same tube in one place is no mesh slayer's mesh will
	// take, but it is not too many
	_, err = most.in(looks{body: bark}).things(maths.Subspace{})
	if errors.Is(err, errTooMany) {
		t.Errorf("a thing of exactly %d faces: %v", mostFaces, err)
	}
	blade, err := newBlade(measure.Metres(1), measure.Metres(0.1),
		measure.Metres(0))
	if err != nil {
		t.Fatal(err)
	}
	over := good{parts: append(most.parts, part{shape: blade,
		space: maths.Subspace{Scale: 1}})}
	_, err = over.in(looks{body: bark}).things(maths.Subspace{})
	if !errors.Is(err, errTooMany) {
		t.Errorf("a thing of %d faces: %v, not %v", over.faces(), err,
			errTooMany)
	}
}

// a thing of one form is its form, at its own size and no other, and
// the coarsest cut there is will do.
func TestAThingOfOneFormIsThatFormAtItsSize(t *testing.T) {
	const tall, wide = 0.122, 0.033
	made, err := newGood(mustTube(t, tall, wide, wide),
		mustCut(t, leastAround))
	if err != nil {
		t.Fatal(err)
	}
	things, err := made.in(looks{body: granite}).things(
		maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	bounds := things[0].Shape.(*mesh.Mesh).Bounds()
	if got := bounds.Max.Y - bounds.Min.Y; math.Abs(got-tall) > close {
		t.Errorf("a can %v tall is %v tall as a thing", tall, got)
	}
}
