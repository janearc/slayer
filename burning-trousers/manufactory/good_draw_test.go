//go:build draw

package manufactory_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/janearc/slayer/burning-trousers/specimens"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// allSpecimens includes both registered kinds and checked profile recipes.
func allSpecimens(t *testing.T) []specimens.Specimen {
	t.Helper()
	profiled, err := specimens.Profiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range manufacture.KindNames() {
		kind, err := manufacture.KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		profiled = append(profiled, specimens.Specimen{Name: name, Kind: kind})
	}
	return profiled
}

// The same kind from the same seed is the same thing, bit for bit.
func TestTheSameSeedMakesTheSameThing(t *testing.T) {
	for _, specimen := range allSpecimens(t) {
		t.Run(specimen.Name, func(t *testing.T) {
			for seed := range uint64(seeds) {
				t.Run(strconv.FormatUint(seed, 10), func(t *testing.T) {
					first := standingKind(t, specimen.Name, specimen.Kind, seed)
					again := standingKind(t, specimen.Name, specimen.Kind, seed)
					if !reflect.DeepEqual(first, again) {
						t.Errorf("%s from seed %d is made two ways", specimen.Name, seed)
					}
				})
			}
		})
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
	for _, specimen := range allSpecimens(t) {
		t.Run(specimen.Name, func(t *testing.T) {
			for seed := range uint64(seeds) {
				t.Run(strconv.FormatUint(seed, 10), func(t *testing.T) {
					for at, read := range madeKind(t, specimen.Name, specimen.Kind, seed) {
						for _, corner := range read.points {
							if !finite(corner) {
								t.Errorf("%s from seed %d, part %d, has a corner "+
									"at %v", specimen.Name, seed, at, corner)
								break
							}
						}
					}
				})
			}
		})
	}
}

// A skin the mesh calls closed is closed when its corners are read by where
// they are, wound one way, and holds a space greater than zero, so its faces
// face out.
func TestAClosedSkinIsClosedAndFacesOut(t *testing.T) {
	for _, specimen := range allSpecimens(t) {
		t.Run(specimen.Name, func(t *testing.T) {
			for seed := range uint64(seeds) {
				t.Run(strconv.FormatUint(seed, 10), func(t *testing.T) {
					for at, read := range madeKind(t, specimen.Name, specimen.Kind, seed) {
						if !read.closed {
							continue
						}
						if err := read.unclosed(); err != nil {
							t.Errorf("%s from seed %d, part %d: %v", specimen.Name, seed, at,
								err)
						} else if held := read.volume(); held <= 0 {
							t.Errorf("%s from seed %d, part %d, holds %g: it "+
								"faces in", specimen.Name, seed, at, held)
						}
					}
				})
			}
		})
	}
}
