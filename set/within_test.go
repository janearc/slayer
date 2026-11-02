package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// withinOf is the kīlauea iki set with its trees on the rim grown within
// a wedge, written as it is given.
func withinOf(t *testing.T, wedge string) string {
	t.Helper()
	return strings.Replace(kilaueaIki(t), "        alight: 4\n",
		"        alight: 4\n        within: "+wedge+"\n", 1)
}

// a stand's wedge reads as written: the thing it is seen from, its
// bearings, and how near and far. a stand without one says so.
func TestAStandGrowsWithinAWedge(t *testing.T) {
	made, err := set.Read("kilauea-iki.yaml", []byte(withinOf(t,
		"{of: the first tree, from: 350, to: 20, near: 40, far: 250}")))
	if err != nil {
		t.Fatal(err)
	}
	stands := made.Stands()
	of, from, to, near, far, has := stands[1].Within()
	if !has || of != "the first tree" || from != 350 || to != 20 ||
		near != 40 || far != 250 {
		t.Errorf("within %q %v %v %v %v %v", of, from, to, near, far, has)
	}
	if _, _, _, _, _, has := stands[0].Within(); has {
		t.Errorf("a stand with no wedge has one")
	}
}

// a wedge missing a part, reaching nowhere, off the compass, or seen
// from a thing the set has not, is refused.
func TestAWedgesMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ wedge, said string }{
		{"{of: the first tree, from: 80, to: 140, near: 40}",
			"within says of, from, to, near and far"},
		{"{of: the first tree, from: 80, to: 140, near: 90, far: 90}",
			"which is nowhere"},
		{"{of: the first tree, from: 80, to: 400, near: 1, far: 90}",
			"to "},
		{"{of: the far hill, from: 80, to: 140, near: 1, far: 90}",
			`within sight of "the far hill"`},
	} {
		_, err := set.Read("kilauea-iki.yaml", []byte(withinOf(t,
			mistake.wedge)))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%s: refused with %v, wanted %q", mistake.wedge, err,
				mistake.said)
		}
	}
}
