package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// shoreKinds are the kinds a shore's set asks for that say how big they
// are from their recipes.
var shoreKinds = []string{"street", "human-scale-2m", "avicennia-marina",
	"aegiceras-corniculatum", "bruguiera-gymnorhiza", "kandelia-obovata",
	"sonneratia-apetala", "pier", "tower", "bridge", "condominium",
	"sampan", "sampan-at-anchor", "ferry", "ferry-at-anchor",
	"fishing-boat", "fishing-boat-at-anchor", VillageKind, "village-house",
	SuperblockKind, LashedStreetKind}

// extentSeeds are the seeds each kind is checked over.
const extentSeeds = 4

// holds is whether a box has all of another inside it, to a hair.
func holds(outer, inner maths.Box) bool {
	const hair = 1e-9
	return outer.Min.X <= inner.Min.X+hair && outer.Min.Y <= inner.Min.Y+hair &&
		outer.Min.Z <= inner.Min.Z+hair && outer.Max.X >= inner.Max.X-hair &&
		outer.Max.Y >= inner.Max.Y-hair && outer.Max.Z >= inner.Max.Z-hair
}

func TestAnExtentHoldsEverythingTheKindMakes(t *testing.T) {
	for _, name := range shoreKinds {
		row, err := testKind(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for number := range uint64(extentSeeds) {
			from := sow(number)
			box, err := extentOf(row, from)
			if err != nil {
				t.Fatalf("%s, seed %d: %v", name, number, err)
			}
			made, err := makeOf(row, from)
			if err != nil {
				t.Fatalf("%s, seed %d: %v", name, number, err)
			}
			if !holds(box, made.Bounds()) {
				t.Errorf("%s, seed %d: the extent %+v is smaller than "+
					"what is made, %+v", name, number, box, made.Bounds())
			}
		}
	}
}

func TestAnExtentIsTheSameForTheSameSeed(t *testing.T) {
	for _, name := range shoreKinds {
		row, err := testKind(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		first, err := extentOf(row, sow(7))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		again, err := extentOf(row, sow(7))
		if err != nil || first != again {
			t.Errorf("%s: %+v then %+v (%v)", name, first, again, err)
		}
	}
}

func TestAKindWhoseSeedWasNeverSownHasNoExtentAndIsNotSeen(t *testing.T) {
	row, err := kindNamed("street")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := extentOf(row, seed{}); err == nil {
		t.Error("an extent was given for a seed never sown")
	}
	if _, err := seenOf(row, seed{}, maths.Local(), lookingDown()); err == nil {
		t.Error("a sight was given for a seed never sown")
	}
}

// lookingDown is an eye at the origin looking along minus z, a quarter turn
// wide, on a film of a thousand by five hundred.
func lookingDown() strategies.Eye {
	return strategies.Eye{Place: maths.Subspace{Origin: maths.Vec{},
		Orientation: maths.LookAtUp(maths.Vec{}, maths.V(0, 0, -1), maths.Up)},
		Wide: math.Pi / 2, Across: 1000, Down: 500}
}

func TestAStreetAheadIsSeenAndOneBehindIsNot(t *testing.T) {
	row, err := kindNamed("street")
	if err != nil {
		t.Fatal(err)
	}
	ahead := maths.Subspace{Origin: maths.V(0, 0, -60),
		Orientation: maths.Identity, Scale: 1}
	behind := maths.Subspace{Origin: maths.V(0, 0, 200),
		Orientation: maths.Identity, Scale: 1}
	got, err := seenOf(row, sow(1), ahead, lookingDown())
	if err != nil || !got.Visible || !(got.Pixels > 0) {
		t.Errorf("a street ahead is %+v (%v)", got, err)
	}
	got, err = seenOf(row, sow(1), behind, lookingDown())
	if err != nil || got.Visible {
		t.Errorf("a street behind is %+v (%v)", got, err)
	}
}

func TestANearerThingSpansMorePixelsThanAFarOne(t *testing.T) {
	row, err := kindNamed("human-scale-2m")
	if err != nil {
		t.Fatal(err)
	}
	near, far := maths.V(0, 0, -5), maths.V(0, 0, -50)
	one, err := seenOf(row, sow(1), maths.Subspace{Origin: near,
		Orientation: maths.Identity, Scale: 1}, lookingDown())
	if err != nil {
		t.Fatal(err)
	}
	other, err := seenOf(row, sow(1), maths.Subspace{Origin: far,
		Orientation: maths.Identity, Scale: 1}, lookingDown())
	if err != nil || !(one.Pixels > other.Pixels) {
		t.Errorf("near %v pixels, far %v (%v)", one.Pixels, other.Pixels, err)
	}
}

func TestAnEyeThatSeesNothingIsRefusedByEveryKindsSeen(t *testing.T) {
	blind := strategies.Eye{}
	for _, name := range []string{"street", "pier", "stone"} {
		if row, err := kindNamed(name); err == nil {
			if _, err := seenOf(row, sow(1), maths.Local(), blind); err == nil {
				t.Errorf("%s was seen by an eye that sees nothing", name)
			}
		}
	}
}

func TestAGreedyExtentIsTheBoundsOfWhatIsMade(t *testing.T) {
	row, err := kindNamed("pebble")
	if err != nil {
		t.Fatal(err)
	}
	box, err := greedyExtent(row.how, sow(3))
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(3))
	if err != nil || box != made.Bounds() {
		t.Errorf("greedy %+v, made %+v (%v)", box, made.Bounds(), err)
	}
}

// Every non-maker object checks the eye before asking for its own extent.
func TestObjectSightKeepsEyeAndExtentRefusals(t *testing.T) {
	for _, value := range []object{Ecosystem{}, Atmosphere{}, Sea{}, Plume{}, Text{}, Laser{}} {
		if _, err := value.seen(sow(1), maths.Local(), strategies.Eye{}); !errors.Is(err, strategies.ErrEye) {
			t.Errorf("%T did not refuse the eye first: %v", value, err)
		}
	}
	if _, err := (Laser{}).seen(sow(1), maths.Local(), lookingDown()); !errors.Is(err, errLaser) {
		t.Errorf("valid eye lost the laser extent refusal: %v", err)
	}
}
