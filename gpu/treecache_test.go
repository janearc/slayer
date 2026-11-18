//go:build draw

package gpu

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/janearc/slayer/maths"
)

// cachedFaces is enough faces for a mesh's tree to be worth the file
const cachedFaces = fewestCached + fewestCached/2

// changedFace and otherFace are the faces a test changes, or swaps
const changedFace, otherFace = cachedFaces / 2, cachedFaces / 3

// lowestBit is a float's last bit, the least change there is to it
const lowestBit = 1

// shortWait is how long a test's shelf waits for a held file
const shortWait = 50 * time.Millisecond

// damagedNumber is the number a test damages in a kept tree: one of its
// first face's corners
const damagedNumber = 4

// someFaces is faces, every number and every smooth one drawn from a
// seed, so the same seed gives the same faces
func someFaces(count int, seed uint64) []triangle {
	draw := rand.New(rand.NewPCG(seed, seed))
	faces := make([]triangle, count)
	for i := range faces {
		for corner := range faces[i].corners {
			faces[i].corners[corner] = maths.V(draw.NormFloat64(),
				draw.NormFloat64(), draw.NormFloat64())
			faces[i].normals[corner] = maths.V(draw.Float64(),
				draw.Float64(), draw.Float64())
		}
		faces[i].smooth = draw.IntN(len(faces[i].corners)) == firstPlace
	}
	return faces
}

// grownFresh is grow's tree over a copy of the faces, since grow sorts
// what it is given
func grownFresh(faces []triangle) tree {
	return grow(slices.Clone(faces))
}

// shelfAt is a cache in a file, as one program
func shelfAt(path string, program byte) *treeShelf {
	return &treeShelf{path: path, program: programKey{program}, wait: lockWait}
}

// shelfIn is a cache in a fresh file, as one program
func shelfIn(t *testing.T, program byte) *treeShelf {
	t.Helper()
	return shelfAt(filepath.Join(t.TempDir(), "trees.bolt"), program)
}

// a tree packed and unpacked is the tree grown: its faces in leaf order,
// every bit of every number, and its nodes
func TestAKeptTreeIsTheTreeGrown(t *testing.T) {
	faces := someFaces(cachedFaces, 1)
	want := grownFresh(faces)
	got, err := unpackTree(packTree(grownFresh(faces)), len(faces))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatal("the unpacked tree is not the tree grown")
	}
}

// the first look grows the tree and keeps it; the second takes it from
// the file, and both are the tree grow makes
func TestTheCacheGrowsATreeOnceAndKeepsItAfter(t *testing.T) {
	faces := someFaces(cachedFaces, 2)
	want := grownFresh(faces)
	kept := shelfIn(t, 1)
	for look := range 2 {
		if got := kept.grown(slices.Clone(faces)); !reflect.DeepEqual(got, want) {
			t.Fatalf("look %d: the tree is not the tree grow makes", look)
		}
	}
	if tally := kept.tally; tally.grown != 1 || tally.kept != 1 || tally.failed != 0 {
		t.Fatalf("grown once and kept once, not %+v", tally)
	}
}

// changeCorner moves the changed face's corners by their floats' last bit
func changeCorner(faces []triangle) {
	for i, corner := range faces[changedFace].corners {
		bits := math.Float64bits(corner.X) ^ lowestBit
		faces[changedFace].corners[i].X = math.Float64frombits(bits)
	}
}

// changeNormal moves the changed face's normals by their last bit
func changeNormal(faces []triangle) {
	for i, normal := range faces[changedFace].normals {
		bits := math.Float64bits(normal.Z) ^ lowestBit
		faces[changedFace].normals[i].Z = math.Float64frombits(bits)
	}
}

// changeSmooth turns the changed face from smooth to flat, or back
func changeSmooth(faces []triangle) {
	faces[changedFace].smooth = !faces[changedFace].smooth
}

// changeOrder swaps two faces, so the same faces come in another order
func changeOrder(faces []triangle) {
	faces[changedFace], faces[otherFace] = faces[otherFace], faces[changedFace]
}

// any change to the faces, the least bit of a corner or a normal, a
// smooth face made flat, or two faces swapped, is another tree: a miss,
// grown as grow grows it, and never the first one's
func TestAChangedFaceIsAnotherTree(t *testing.T) {
	faces := someFaces(cachedFaces, 9)
	changes := map[string]func([]triangle){"a corner": changeCorner,
		"a normal": changeNormal, "a smooth face": changeSmooth,
		"the order": changeOrder}
	for name, change := range changes {
		kept := shelfIn(t, 1)
		kept.grown(slices.Clone(faces))
		changed := slices.Clone(faces)
		change(changed)
		want := grownFresh(changed)
		if got := kept.grown(slices.Clone(changed)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s changed: the tree is not the tree grow makes", name)
		}
		if tally := kept.tally; tally.grown != 2 || tally.kept != 0 {
			t.Errorf("%s changed, yet the first tree was taken: %+v", name, tally)
		}
	}
}

// another program's trees are not taken: a rebuilt program grows its own
func TestAnotherProgramGrowsItsOwnTrees(t *testing.T) {
	faces := someFaces(cachedFaces, 3)
	first := shelfIn(t, 1)
	first.grown(slices.Clone(faces))
	second := shelfAt(first.path, 2)
	second.grown(slices.Clone(faces))
	if second.tally.grown != 1 || second.tally.kept != 0 {
		t.Fatalf("another program took a tree it did not grow: %+v", second.tally)
	}
}

// a kept tree damaged in one bit of one number, whole in every size and
// index, is caught by its sha256, grown again, counted, and stored over
func TestADamagedTreeIsCaughtAndGrownAgain(t *testing.T) {
	faces := someFaces(cachedFaces, 10)
	want := grownFresh(faces)
	kept := shelfIn(t, 1)
	damaged := packTree(grownFresh(faces))
	damaged[sumBytes+headBytes+damagedNumber*numberBytes] ^= lowestBit
	if err := kept.store(kept.key(faces), damaged); err != nil {
		t.Fatal(err)
	}
	for look := range 2 {
		if got := kept.grown(slices.Clone(faces)); !reflect.DeepEqual(got, want) {
			t.Fatalf("look %d: the tree is not the tree grow makes", look)
		}
	}
	tally := kept.tally
	if tally.failed != 1 || tally.kept != 1 || tally.first == nil {
		t.Fatalf("one failure, then the tree stored over it, not %+v", tally)
	}
	if !strings.Contains(tally.String(), "damaged") {
		t.Fatalf("the tally does not say the file was damaged: %s", tally)
	}
}

// a broken tree under the key, too short to be one, is grown again and
// stored over
func TestABrokenTreeIsGrownAgainAndStoredOver(t *testing.T) {
	faces := someFaces(cachedFaces, 4)
	want := grownFresh(faces)
	kept := shelfIn(t, 1)
	if err := kept.store(kept.key(faces), []byte("not a tree")); err != nil {
		t.Fatal(err)
	}
	for look := range 2 {
		if got := kept.grown(slices.Clone(faces)); !reflect.DeepEqual(got, want) {
			t.Fatalf("look %d: the tree is not the tree grow makes", look)
		}
	}
	if tally := kept.tally; tally.failed != 1 || tally.kept != 1 {
		t.Fatalf("one failure, then the tree stored over it, not %+v", tally)
	}
}

// a file held by another past the wait stops the shelf using it for the
// rest of the run: the tree is grown, the failure counted, and the next
// tree does not wait
func TestAHeldFileStopsTheCacheForTheRun(t *testing.T) {
	faces := someFaces(cachedFaces, 11)
	want := grownFresh(faces)
	kept := shelfIn(t, 1)
	kept.wait = shortWait
	var said strings.Builder
	kept.said = &said
	holder, err := bolt.Open(kept.path, fileMode, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if got := kept.grown(slices.Clone(faces)); !reflect.DeepEqual(got, want) {
		t.Fatal("the tree is not the tree grow makes")
	}
	if kept.stopped() == nil || kept.tally.failed != 1 {
		t.Fatalf("a held file did not stop the shelf: %+v", kept.tally)
	}
	started := time.Now()
	kept.grown(slices.Clone(faces))
	if waited := time.Since(started); waited >= shortWait || kept.tally.failed != 2 {
		t.Fatalf("the stopped shelf waited %v, tally %+v", waited, kept.tally)
	}
	if lines := strings.Count(said.String(), "gave up on "+kept.path); lines != 1 {
		t.Fatalf("giving up was said %d times, want once: %q", lines,
			said.String())
	}
}

// a shelf that never gives up on its file says nothing of it
func TestAShelfThatKeepsItsFileSaysNothing(t *testing.T) {
	faces := someFaces(cachedFaces, 3)
	kept := shelfIn(t, 1)
	var said strings.Builder
	kept.said = &said
	kept.grown(slices.Clone(faces))
	kept.grown(slices.Clone(faces))
	if said.Len() != 0 || kept.tally.kept != 1 {
		t.Fatalf("said %q, tally %+v", said.String(), kept.tally)
	}
}

// a file that cannot be opened grows every tree, and the pictures do not
// know: each is the tree grow makes, and the failure is counted
func TestATreeIsGrownWhenItsFileCannotBeOpened(t *testing.T) {
	faces := someFaces(cachedFaces, 5)
	want := grownFresh(faces)
	kept := shelfAt(t.TempDir(), 1)
	if got := kept.grown(slices.Clone(faces)); !reflect.DeepEqual(got, want) {
		t.Fatal("the tree is not the tree grow makes")
	}
	if kept.tally.failed != 1 {
		t.Fatalf("a directory is not a cache, yet nothing failed: %+v", kept.tally)
	}
}

// what does not fit a tree is refused: too short, kept for other faces,
// a size that is not its counts', and a node pointing outside it
func TestPackedTreesThatDoNotFitAreRefused(t *testing.T) {
	faces := someFaces(cachedFaces, 6)
	grown := grownFresh(faces)
	outside := grown
	outside.nodes = slices.Clone(grown.nodes)
	outside.nodes[firstPlace].first = len(outside.nodes)
	packed := packTree(grown)
	cases := map[string][]byte{
		"a stub":              packed[:sumBytes+headBytes-lowestBit],
		"one byte short":      packed[:len(packed)-lowestBit],
		"a node outside":      packTree(outside),
		"kept for more faces": packed,
	}
	for name, bad := range cases {
		count := len(faces)
		if name == "kept for more faces" {
			count++
		}
		if _, err := unpackTree(bad, count); err == nil {
			t.Errorf("%s was taken as a tree", name)
		}
	}
}

// CacheTrees turns the cache on for meshes big enough and off again, and
// TreesCached says what it did
func TestCacheTreesTurnsTheCacheOnAndOff(t *testing.T) {
	defer stopCaching()
	if err := CacheTrees(filepath.Join(t.TempDir(), "trees.bolt")); err != nil {
		t.Fatal(err)
	}
	faces := someFaces(cachedFaces, 7)
	for range 2 {
		grownOver(slices.Clone(faces))
	}
	grownOver(someFaces(fewestCached-1, 8))
	if tally := TreesCached(); tally.grown != 1 || tally.kept != 1 {
		t.Fatalf("grown once and kept once, the small mesh not at all: %+v", tally)
	}
	if !TreesCached().Used() {
		t.Error("a tally of trees kept and grown is not used")
	}
	stopCaching()
	if tally := TreesCached(); tally != (TreeTally{}) || tally.Used() {
		t.Fatalf("off, yet it says %+v", tally)
	}
}

// stopCaching turns the tree cache off after a test
func stopCaching() { _ = CacheTrees("") }
