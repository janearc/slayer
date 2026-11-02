// Package manufactory holds slayer's manufactory to what can be checked
// of a made thing from outside: the same seed makes the same thing,
// every corner is a number, and a closed skin is closed and faces out,
// worked out here from its corners, never asked of the mesh.
package manufactory_test

import (
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// seeds is how many seeds each kind is made from.
const seeds = 8

// made is one of a kind from a seed, as the skins of the meshes a
// scene would hold, standing nowhere in particular.
func made(t *testing.T, name string, seed uint64) []skin {
	t.Helper()
	things := standing(t, name, seed)
	var skins []skin
	for _, one := range things {
		held, isMesh := one.Shape.(*mesh.Mesh)
		if !isMesh {
			t.Fatalf("%s from seed %d is a %T, not a mesh", name, seed,
				one.Shape)
		}
		skins = append(skins, skin{points: held.Points(),
			faces: held.Faces(), closed: held.Closed()})
	}
	return skins
}

// standing is one of a kind from a seed as thing.Things.
func standing(t *testing.T, name string, seed uint64) []thing.Thing {
	t.Helper()
	kind, err := manufacture.KindNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	good, err := manufacture.Make(kind, manufacture.Sow(seed))
	if err != nil {
		t.Fatalf("%s from seed %d: %v", name, seed, err)
	}
	things, err := good.Things(maths.Subspace{})
	if err != nil {
		t.Fatalf("%s from seed %d: %v", name, seed, err)
	}
	return things
}
