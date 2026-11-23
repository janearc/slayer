package manufacture

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// everyKindAhead is how far ahead of the eye, in metres, a kind is stood
// to be asked whether it is seen.
const everyKindAhead = 60.0

// askedBoth asks a kind its extent and its sight from one seed, and says
// what is wrong with the answer: a refusal, a box that is not a box, or a
// span that is not a number of pixels. It says nothing where all is well.
func askedBoth(row kind) error {
	ahead := maths.Subspace{Origin: maths.V(0, 0, -everyKindAhead),
		Orientation: maths.Identity, Scale: 1}
	box, err := extentOf(row, sow(1))
	if err != nil {
		return fmt.Errorf("no extent: %w", err)
	}
	if !(box.Min.X <= box.Max.X && box.Min.Y <= box.Max.Y &&
		box.Min.Z <= box.Max.Z) {
		return fmt.Errorf("the extent %+v is not a box", box)
	}
	got, err := seenOf(row, sow(1), ahead, lookingDown())
	if err != nil {
		return fmt.Errorf("not seen or unseen: %w", err)
	}
	if math.IsNaN(got.Pixels) || got.Pixels < 0 {
		return fmt.Errorf("spans %v pixels", got.Pixels)
	}
	return nil
}

// TestEveryKindIsAnObjectAndTheHonestOnesSayTheirSize walks every kind the
// manufactory lists and checks each is held as a maker, which is the base
// type with extent and seen. Of them it asks the extent and the sight only
// of the shore kinds, which say their size from their recipe: asking each
// of the rest makes it whole. A kind added without the two methods does
// not build.
func TestEveryKindIsAnObjectAndTheHonestOnesSayTheirSize(t *testing.T) {
	for _, name := range kindNames() {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if row.how == nil {
			t.Errorf("%s is not held as the base type", name)
		}
	}
	for _, name := range shoreKinds {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := askedBoth(row); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// crooked is a kind whose extent is a box turned inside out and which
// makes nothing, to be asked what is wrong with.
type crooked struct{}

// extent of a crooked kind has its far corner short of its near.
func (crooked) extent(seed) (maths.Box, error) {
	return maths.Box{Min: maths.V(1, 1, 1), Max: maths.V(0, 0, 0)}, nil
}

// seen of a crooked kind is as any maker's is, from its extent.
func (kindOf crooked) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(kindOf, from, at, by)
}

// made of a crooked kind is nothing.
func (crooked) made(seed) (good, error) { return good{}, nil }

// TestAKindWithABadBoxOrNoMakerIsReported is the anti-test of the walk: a
// kind whose box is inside out, and one with no maker, are each reported
// by the check the walk makes.
func TestAKindWithABadBoxOrNoMakerIsReported(t *testing.T) {
	if err := askedBoth(kind{name: "crooked", how: crooked{}}); err == nil {
		t.Error("a box turned inside out was passed")
	}
	if err := askedBoth(kind{name: "none"}); !errors.Is(err, errKind) {
		t.Errorf("a kind with no maker gave %v, want %v", err, errKind)
	}
}
