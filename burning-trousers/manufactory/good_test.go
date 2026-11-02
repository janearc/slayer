package manufactory_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
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
// corners.
func TestSeedsMakeDifferentThings(t *testing.T) {
	for _, name := range []string{"spruce", "boulder", "fern"} {
		seen := map[maths.Vec]uint64{}
		for seed := range uint64(seeds) {
			middle := middleOf(made(t, name, seed)[0].points)
			if earlier, twice := seen[middle]; twice {
				t.Errorf("%s from seeds %d and %d are the same", name,
					earlier, seed)
			}
			seen[middle] = seed
		}
	}
}

// middleOf is the average of some corners.
func middleOf(corners []maths.Vec) maths.Vec {
	var sum maths.Vec
	for _, corner := range corners {
		sum = sum.Add(corner)
	}
	return sum.Scale(1 / float64(len(corners)))
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
