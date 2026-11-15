package template

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
)

// each template hands back its own copy: the body's fields are private
// behind its door, and Aimed copies the tails it writes, so aiming one
// call's tails must leave a fresh call unaimed - of the same template
// and of every other.
func TestEachBodyHandsBackItsOwnCopy(t *testing.T) {
	pristine := map[string]bodies.Body{}
	for name, make := range named {
		pristine[name] = its(t, make)
	}
	for name, make := range named {
		mine := its(t, make)
		mine = mine.Aimed(maths.V(9, 9, 9), maths.V(1, 0, 0))
		if len(mine.Tails()) > 0 &&
			len(apartWithColours(mine, pristine[name])) == 0 {
			t.Errorf("%s: aiming changed nothing, so the check "+
				"below proves nothing", name)
		}
		for other, again := range named {
			for _, a := range apartWithColours(its(t, again), pristine[other]) {
				t.Errorf("after aiming a %s, a fresh %s differs: %s", name, other, a)
			}
		}
	}
}

// a call twice is the same body twice, before anything is changed.
func TestTwoCallsOfATemplateAreTheSameBody(t *testing.T) {
	for name, make := range named {
		for _, a := range apartWithColours(its(t, make), its(t, make)) {
			t.Errorf("%s: two calls differ: %s", name, a)
		}
	}
}
