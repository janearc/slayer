// Package manufactory holds slayer's manufactory to what can be checked
// of a made thing from outside: the same seed makes the same thing,
// every corner is a number, and a closed skin is closed and faces out,
// worked out here from its corners, never asked of the mesh.
package manufactory_test

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// made is one of a kind from a seed, as the skins of the meshes a
// scene would hold, standing nowhere in particular. Shared groups contribute
// each prototype once; their placements are checked without expanding them.
func made(t *testing.T, name string, seed uint64) []skin {
	t.Helper()
	return readSkins(t, name, seed, standing(t, name, seed))
}

// madeKind reads the meshes from a checked profile-backed or registered kind.
func madeKind(t *testing.T, name string, kind manufacture.Kind, seed uint64) []skin {
	t.Helper()
	return readSkins(t, name, seed, standingKind(t, name, kind, seed))
}

func readSkins(t *testing.T, name string, seed uint64, things []thing.Thing) []skin {
	t.Helper()
	var skins []skin
	seen := map[*mesh.Mesh]bool{}
	seenShapes := map[*shape.Mesh]bool{}
	appendMesh := func(held *mesh.Mesh) {
		if !seen[held] {
			skins = append(skins, skin{points: held.Points(),
				faces: held.Faces(), closed: held.Closed()})
			seen[held] = true
		}
	}
	appendShapeMesh := func(held *shape.Mesh) {
		if held == nil {
			t.Fatal("a reference mesh is nil")
		}
		if !seenShapes[held] {
			for _, face := range held.Faces {
				for _, corner := range face {
					if corner < 0 || corner >= len(held.Vertices) {
						t.Fatalf("reference mesh face names missing corner %d", corner)
					}
				}
			}
			skins = append(skins, skin{points: slices.Clone(held.Vertices),
				faces: slices.Clone(held.Faces), closed: held.Closed()})
			seenShapes[held] = true
		}
	}
	for _, one := range things {
		switch held := one.Shape.(type) {
		case *mesh.Mesh:
			appendMesh(held)
		case *shape.Mesh:
			appendShapeMesh(held)
		case shape.Sphere, shape.Tube:
			if err := checkPrimitive(one.Shape); err != nil {
				t.Fatalf("%s from seed %d: %v", name, seed, err)
			}
			if !finitePlacedBounds(one.At(0), one.Shape.Bounds()) {
				t.Fatalf("%s from seed %d has nonfinite placed %T", name, seed, one.Shape)
			}
		case *shape.MeshGroup:
			for _, member := range held.Members() {
				// With a uniform positive scale and proper rotation, winding
				// is preserved. Check the transform independently of the group.
				x := member.Space.Orientation.Out(maths.XAxis)
				y := member.Space.Orientation.Out(maths.YAxis)
				z := member.Space.Orientation.Out(maths.ZAxis)
				if member.Space.Scale < 0 || x.Cross(y).Dot(z) <= 0 {
					t.Fatal("group placement reverses winding")
				}
				if !finitePlacedBounds(member.Space, member.Mesh.Bounds()) {
					t.Fatal("nonfinite placed prototype")
				}
				switch prototype := member.Mesh.(type) {
				case *mesh.Mesh:
					appendMesh(prototype)
				case *shape.Mesh:
					appendShapeMesh(prototype)
				default:
					t.Fatalf("unexpected group prototype %T", member.Mesh)
				}
			}
		default:
			t.Fatalf("%s from seed %d is a %T, not a mesh", name, seed,
				one.Shape)
		}
	}
	return skins
}

func finitePlacedBounds(space maths.Subspace, box maths.Box) bool {
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
		if !finite(space.Out(p)) {
			return false
		}
	}
	return true
}

// checkPrimitive checks the finite measures and independent inside/outside
// witnesses of analytical shapes; they have no triangle skin to read.
func checkPrimitive(solid shape.Shape) error {
	number := func(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
	box := solid.Bounds()
	if !finite(box.Min) || !finite(box.Max) {
		return fmt.Errorf("%T has nonfinite bounds", solid)
	}
	switch held := solid.(type) {
	case shape.Sphere:
		if !number(held.Radius) || !(held.Radius > 0) {
			return fmt.Errorf("sphere has invalid radius %g", held.Radius)
		}
		if got := held.Distance(maths.V(held.Radius/2, 0, 0)); !(got < 0) {
			return fmt.Errorf("sphere interior distance is %g", got)
		}
		if outside := held.Radius * 2; number(outside) {
			if got := held.Distance(maths.V(outside, 0, 0)); !(got > 0) {
				return fmt.Errorf("sphere exterior distance is %g", got)
			}
		}
		if normal := held.Normal(maths.V(held.Radius, 0, 0)); !finite(normal) || normal.Sub(maths.XAxis).Len() > 8*(math.Nextafter(1, 2)-1) {
			return fmt.Errorf("sphere outward normal is %v", normal)
		}
	case shape.Tube:
		if !number(held.Radius) || held.Radius < 0 || !number(held.Thick) ||
			!number(held.Height) || held.Height < 0 {
			return fmt.Errorf("tube has invalid measures %v", held)
		}
		if held.Thick > 0 && held.Height > 0 {
			if got := held.Distance(maths.V(held.Radius, 0, 0)); !(got < 0) {
				return fmt.Errorf("tube interior distance is %g", got)
			}
			outer := held.Radius + held.Thick/2
			outside := outer + math.Max(1, outer)
			if number(outside) {
				if got := held.Distance(maths.V(outside, 0, 0)); !(got > 0) {
					return fmt.Errorf("tube exterior distance is %g", got)
				}
			}
		}
	default:
		return fmt.Errorf("unsupported analytical shape %T", solid)
	}
	return nil
}

// standing is one of a kind from a seed as thing.Things.
func standing(t *testing.T, name string, seed uint64) []thing.Thing {
	t.Helper()
	var kind manufacture.Kind
	var err error
	family := map[string]string{"fern": "fern", "spruce": "conifer"}[name]
	if family != "" {
		file, raw, err := manufacture.EmbeddedPlantBook(family)
		if err != nil {
			t.Fatal(err)
		}
		book, err := manufacture.ReadPlantBook(family, file, raw)
		if err != nil {
			t.Fatal(err)
		}
		kind, err = book.New(manufacture.PlantOptions{Species: name})
	} else {
		kind, err = manufacture.KindNamed(name)
	}
	if err != nil {
		t.Fatal(err)
	}
	return standingKind(t, name, kind, seed)
}

// standingKind constructs the supplied kind without looking it up by name.
func standingKind(t *testing.T, name string, kind manufacture.Kind, seed uint64) []thing.Thing {
	t.Helper()
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
