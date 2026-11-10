package manufactory_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Every kind is found by its name, and a name that is no kind is
// refused with the names there are.
func TestEveryKindIsFoundByItsName(t *testing.T) {
	names := manufacture.KindNames()
	if len(names) == 0 {
		t.Fatal("the manufactory has no kinds")
	}
	_, err := manufacture.KindNamed("larch")
	if err == nil {
		t.Fatal("a larch is a kind, and there is no larch")
	}
	for _, name := range names {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the refusal of a larch does not name the %s", name)
		}
	}
}

// The same kind from the same seed is the same thing, bit for bit.
func TestTheSameSeedMakesTheSameThing(t *testing.T) {
	for _, name := range manufacture.KindNames() {
		for seed := range uint64(seeds) {
			first, again := standing(t, name, seed), standing(t, name, seed)
			if !reflect.DeepEqual(first, again) {
				t.Errorf("%s from seed %d is made two ways", name, seed)
			}
		}
	}
}

// Different seeds make different spruces, stones and ferns: the seed
// does the work hand-set numbers did. Each is told by the middle of its
// corners where they stand: a shared prototype is the same mesh from
// every seed, so its corners count where each member places them.
func TestSeedsMakeDifferentThings(t *testing.T) {
	for _, name := range []string{"spruce", "boulder", "fern"} {
		seen := map[maths.Vec]uint64{}
		for seed := range uint64(seeds) {
			middle := middleStanding(t, standing(t, name, seed))
			if earlier, twice := seen[middle]; twice {
				t.Errorf("%s from seeds %d and %d are the same", name,
					earlier, seed)
			}
			seen[middle] = seed
		}
	}
}

// middleStanding is the average of every corner of some standing
// things, a group's prototypes taken out through each member's space.
func middleStanding(t *testing.T, things []thing.Thing) maths.Vec {
	t.Helper()
	var sum maths.Vec
	corners := 0
	for _, one := range things {
		switch held := one.Shape.(type) {
		case *mesh.Mesh:
			for _, point := range held.Points() {
				sum = sum.Add(point)
				corners++
			}
		case *shape.MeshGroup:
			for _, member := range held.Members() {
				prototype, ok := member.Mesh.(*mesh.Mesh)
				if !ok {
					t.Fatalf("unexpected group prototype %T", member.Mesh)
				}
				for _, point := range prototype.Points() {
					sum = sum.Add(member.Space.Out(point))
					corners++
				}
			}
		default:
			t.Fatalf("unexpected standing shape %T", one.Shape)
		}
	}
	if corners == 0 {
		t.Fatal("nothing stands")
	}
	return sum.Scale(1 / float64(corners))
}

// Every corner of every kind is a number, from every seed.
func TestEveryCornerIsANumber(t *testing.T) {
	for _, name := range manufacture.KindNames() {
		for seed := range uint64(seeds) {
			for at, read := range made(t, name, seed) {
				for _, corner := range read.points {
					if !finite(corner) {
						t.Errorf("%s from seed %d, part %d, has a corner "+
							"at %v", name, seed, at, corner)
						break
					}
				}
			}
		}
	}
}

// finite is whether a point is somewhere.
func finite(point maths.Vec) bool {
	for _, part := range []float64{point.X, point.Y, point.Z} {
		if math.IsNaN(part) || math.IsInf(part, 0) {
			return false
		}
	}
	return true
}

// A skin the mesh calls closed is closed when its corners are read by where
// they are, wound one way, and holds a space greater than zero, so its faces
// face out.
func TestAClosedSkinIsClosedAndFacesOut(t *testing.T) {
	for _, name := range manufacture.KindNames() {
		for seed := range uint64(seeds) {
			for at, read := range made(t, name, seed) {
				if !read.closed {
					continue
				}
				if err := read.unclosed(); err != nil {
					t.Errorf("%s from seed %d, part %d: %v", name, seed, at,
						err)
				} else if held := read.volume(); held <= 0 {
					t.Errorf("%s from seed %d, part %d, holds %g: it "+
						"faces in", name, seed, at, held)
				}
			}
		}
	}
}

// The reading of a skin can fail: a fern's fronds, open blades, are
// not closed, and a boulder wound the other way faces in.
func TestTheSkinReaderSeesAnOpenOrInsideOutSkin(t *testing.T) {
	fronds := made(t, "fern", 1)[1]
	if fronds.unclosed() == nil {
		t.Error("a fern's fronds read as closed")
	}
	stone := made(t, "boulder", 1)[0]
	if held := stone.reversed().volume(); held >= 0 {
		t.Errorf("a boulder wound the other way holds %g", held)
	}
}
