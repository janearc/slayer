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
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// seeds is how many seeds each kind is made from.
const seeds = 8

// made is one of a kind from a seed, as the skins of the meshes a
// scene would hold, standing nowhere in particular. Shared groups contribute
// each prototype once; their placements are checked without expanding them.
func made(t *testing.T, name string, seed uint64) []skin {
	t.Helper()
	things := standing(t, name, seed)
	var skins []skin
	seen := map[*mesh.Mesh]bool{}
	appendMesh := func(held *mesh.Mesh) {
		if !seen[held] {
			skins = append(skins, skin{points: held.Points(),
				faces: held.Faces(), closed: held.Closed()})
			seen[held] = true
		}
	}
	for _, one := range things {
		switch held := one.Shape.(type) {
		case *mesh.Mesh:
			appendMesh(held)
		case *shape.MeshGroup:
			for _, member := range held.Members() {
				prototype, ok := member.Mesh.(*mesh.Mesh)
				if !ok {
					t.Fatalf("unexpected group prototype %T", member.Mesh)
				}
				// With a uniform positive scale and proper rotation, winding
				// is preserved. Check the transform independently of the group.
				x := member.Space.Orientation.Out(maths.XAxis)
				y := member.Space.Orientation.Out(maths.YAxis)
				z := member.Space.Orientation.Out(maths.ZAxis)
				if member.Space.Scale < 0 || x.Cross(y).Dot(z) <= 0 {
					t.Fatal("group placement reverses winding")
				}
				box := prototype.Bounds()
				for corner := range 8 {
					p := box.Min
					if corner&1 != 0 {
						p.X = box.Max.X
					}
					if corner&2 != 0 {
						p.Y = box.Max.Y
					}
					if corner&4 != 0 {
						p.Z = box.Max.Z
					}
					if !finite(member.Space.Out(p)) {
						t.Fatal("nonfinite placed prototype")
					}
				}
				appendMesh(prototype)
			}
		default:
			t.Fatalf("%s from seed %d is a %T, not a mesh", name, seed,
				one.Shape)
		}
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
