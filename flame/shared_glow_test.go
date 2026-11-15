package flame

import (
	"math"
	"slices"
	"testing"
)

// The temperatures and exposure the shared table's tests ask for: a
// crown fire's edge and hottest, at a picture's exposure. Picked, from
// hillside's crown fire, by the disk caches slice's author, 2026-11-12.
const (
	sharedEdge     = 1100.0
	sharedHottest  = 1400.0
	sharedExposure = 2.0
)

func TestAGlowTableAskedTwiceIsMadeOnceAndIsTheSame(t *testing.T) {
	first := newGlowing(sharedEdge, sharedHottest, sharedExposure)
	again := newGlowing(sharedEdge, sharedHottest, sharedExposure)
	if &first.glows[0] != &again.glows[0] {
		t.Fatal("the same table was made twice")
	}
	worked := workedGlowing(sharedEdge, sharedHottest, sharedExposure)
	if !slices.Equal(first.glows, worked.glows) ||
		first.coolest != worked.coolest || first.apart != worked.apart {
		t.Fatal("the shared table is not the table worked out")
	}
}

func TestAGlowTableOfOtherNumbersIsItsOwn(t *testing.T) {
	first := newGlowing(sharedEdge, sharedHottest, sharedExposure)
	for name, other := range map[string]glowing{
		"coolest": newGlowing(math.Nextafter(sharedEdge, 0), sharedHottest,
			sharedExposure),
		"hottest":  newGlowing(sharedEdge, sharedHottest+1, sharedExposure),
		"exposure": newGlowing(sharedEdge, sharedHottest, sharedExposure*2),
	} {
		if &other.glows[0] == &first.glows[0] {
			t.Errorf("another %s was given the same table", name)
		}
	}
}
