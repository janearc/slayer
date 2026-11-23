package named_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/named"
)

// the steps round the ring the tests ask for: the fewest that close it,
// one fewer, and four times the fewest, defined from cmd/ring's floor of
// eight; and the triangles to each step round, five rows across each of
// the faces and one across each rim, two triangles a row, both rims and
// both faces: (4 + 1 + 4 + 1) * 2
const (
	fewestRound   = 8
	tooFewRound   = fewestRound - once
	moreRound     = 4 * fewestRound
	trianglesStep = 20
)

// solid is the built ring's mirror, as a mesh, or the test stops.
func solid(t *testing.T, built *named.Built) *mesh.Mesh {
	t.Helper()
	for _, each := range built.World().Things {
		if each.Material != nil && each.Material.Name == "ring" {
			if shape, made := each.Shape.(*mesh.Mesh); made {
				return shape
			}
		}
	}
	t.Fatal("the ring has no mirror")
	return nil
}

// The ring is a closed solid at the fewest steps round and at more, each
// step two triangles to every row of its four strips.
func TestTheRingIsAClosedSolid(t *testing.T) {
	for _, around := range []int{fewestRound, moreRound} {
		ring := solid(t, mustBuild(t, "ring",
			map[string]any{"around": around}))
		if !ring.Closed() || ring.FaceCount() != around*trianglesStep {
			t.Errorf("around %d: closed %v, %d faces", around,
				ring.Closed(), ring.FaceCount())
		}
	}
}

// anti-tests: fewer steps round than close the ring are refused by name,
// not drawn as more, as cmd/ring once quietly did.
func TestTooFewStepsRoundAreRefused(t *testing.T) {
	err := mustGet(t, "ring").Check(map[string]any{"around": tooFewRound})
	if !errors.Is(err, named.ErrBounds) ||
		!strings.Contains(err.Error(), "around") {
		t.Errorf("around %d: %v; want ErrBounds", tooFewRound, err)
	}
}

// Dressed, by day and by night, the ring keeps its geometry and its
// light, and says what its floor holds; its open inner face is the
// dressed floor's, so the mirror is not closed.
func TestTheDressedRingKeepsItsGeometryAndLight(t *testing.T) {
	for _, night := range []bool{false, true} {
		var said bytes.Buffer
		built, err := mustGet(t, "ring").Build(map[string]any{
			"around": fewestRound, "dress": true, "night": night},
			&said)
		if err != nil {
			t.Fatal(err)
		}
		world := built.World()
		if len(world.Things) < few || len(world.Lamps) == none ||
			!containsAll(said.String(), "the world:", "the floor:") {
			t.Errorf("night %v: %d things, %d lamps, said %q", night,
				len(world.Things), len(world.Lamps), said.String())
		}
	}
}

// anti-tests: with no sun and no dress the ring has no lamp at all, and
// says nothing.
func TestTheBareRingWithNoSunIsUnlit(t *testing.T) {
	var said bytes.Buffer
	built, err := mustGet(t, "ring").Build(map[string]any{
		"around": fewestRound, "sun": false}, &said)
	if err != nil {
		t.Fatal(err)
	}
	if len(built.World().Lamps) != none || said.Len() != none {
		lamps := len(built.World().Lamps)
		t.Errorf("a ring with no sun has %d lamps and said %q",
			lamps, said.String())
	}
}
