package shape

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sort"
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The axes of space, in the order the parts of a vector are kept.
const (
	meshtreeMoreAxisX = 0 // defined: the first axis
	meshtreeMoreAxisY = 1 // defined: the second axis
	meshtreeMoreAxisZ = 2 // defined: the third axis
)

// meshtreeMoreAxes is how many axes space has.
const meshtreeMoreAxes = 3 // defined: space has three axes

// meshtreeMoreRoot is the node a tree begins at.
const meshtreeMoreRoot int32 = nought // defined: the first node is the root

// meshtreeMoreFirst is the first half of a cut node, which comes next
// after it in the tree.
const meshtreeMoreFirst int32 = unity // defined: the node after the cut

// The csv files this file reads, each a table of rows by name.
const (
	meshtreeMoreHalvings = "testdata/meshtree-more-halvings.csv"
	meshtreeMoreNested   = "testdata/meshtree-more-nested.csv"
	meshtreeMoreClusters = "testdata/meshtree-more-clusters.csv"
	meshtreeMoreCuts     = "testdata/meshtree-more-cuts.csv"
	meshtreeMoreSlabs    = "testdata/meshtree-more-slabs.csv"
	meshtreeMoreMisses   = "testdata/meshtree-more-misses.csv"
	meshtreeMoreLongest  = "testdata/meshtree-more-longest.csv"
	meshtreeMoreAway     = "testdata/meshtree-more-away.csv"
	meshtreeMoreWindings = "testdata/meshtree-more-windings.csv"
	meshtreeMorePlaces   = "testdata/meshtree-more-places.csv"
	meshtreeMoreSeeds    = "testdata/meshtree-more-seeds.csv"
)

// meshtreeMoreOrigin is the point where all three parts are nought.
var meshtreeMoreOrigin = maths.V(nought, nought, nought)

// meshtreeMorePicked is picked for a test or a benchmark alike.
func meshtreeMorePicked(tb testing.TB, name string) float64 {
	tb.Helper()
	chosen, err := chosenNumbers()
	if err != nil {
		tb.Fatal(err)
	}
	number, err := chosen.Get(name)
	if err != nil {
		tb.Fatal(err)
	}
	return number.Value
}

// meshtreeMoreVector is the vector in the columns prefix-x, prefix-y and
// prefix-z of a row.
func meshtreeMoreVector(t *testing.T, row map[string]string,
	prefix string) maths.Vec {
	t.Helper()
	return maths.V(cell(t, row, prefix+"-x"), cell(t, row, prefix+"-y"),
		cell(t, row, prefix+"-z"))
}

// meshtreeMoreFlag is the true or false in a column of a row, or a stop.
func meshtreeMoreFlag(t *testing.T, row map[string]string,
	column string) bool {
	t.Helper()
	got, err := strconv.ParseBool(row[column])
	if err != nil {
		t.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// meshtreeMoreGroup is a column of the rows of a csv file whose group
// column is group, in the order of the file.
func meshtreeMoreGroup(t *testing.T, file, group,
	column string) []float64 {
	t.Helper()
	var found []float64
	for _, row := range readSheet(t, file) {
		if row["group"] == group {
			found = append(found, cell(t, row, column))
		}
	}
	return found
}

// meshtreeMorePlacesOf is the places, as vectors, of the rows of a csv file
// whose group column is group, in the order of the file.
func meshtreeMorePlacesOf(t *testing.T, file, group string) []maths.Vec {
	t.Helper()
	var found []maths.Vec
	for _, row := range readSheet(t, file) {
		if row["group"] == group {
			found = append(found, meshtreeMoreVector(t, row, "place"))
		}
	}
	return found
}

// meshtreeMoreRandom is the random stream whose two seeds are in the row of
// the seeds file called name.
func meshtreeMoreRandom(t *testing.T, name string) *rand.Rand {
	t.Helper()
	row := readSheet(t, meshtreeMoreSeeds).row(t, meshtreeMoreSeeds, name)
	first := uint64(cellCount(t, row, "first"))
	second := uint64(cellCount(t, row, "second"))
	return rand.New(rand.NewPCG(first, second))
}

// meshtreeMoreRay is the ray from at, going along direction.
func meshtreeMoreRay(at, direction maths.Vec) maths.Ray {
	return maths.Ray{At: at, Dir: direction}
}

// meshtreeMoreBox is the box from low to high.
func meshtreeMoreBox(low, high maths.Vec) maths.Box {
	return maths.Box{Min: low, Max: high}
}

// meshtreeMoreParts is the parts of a point, one for each axis.
func meshtreeMoreParts(point maths.Vec) [meshtreeMoreAxes]float64 {
	return [meshtreeMoreAxes]float64{point.X, point.Y, point.Z}
}

// meshtreeMoreShifted is a face with each of its corners numbered by more.
func meshtreeMoreShifted(face [cornersOfAFace]int,
	by int) [cornersOfAFace]int {
	var shifted [cornersOfAFace]int
	for place, corner := range face {
		shifted[place] = corner + by
	}
	return shifted
}

// meshtreeMoreCorners is the numbers of the corners of the face that
// begins at first, one after another.
func meshtreeMoreCorners(first int) [cornersOfAFace]int {
	return [cornersOfAFace]int{first, first + secondCorner,
		first + thirdCorner}
}

// meshtreeMoreCollector gathers the faces under a node of a tree.
type meshtreeMoreCollector struct {
	mesh  *Mesh
	found []int
}

// under adds the faces under node to those found.
func (collector *meshtreeMoreCollector) under(node int32) {
	cut := collector.mesh.nodes[node]
	if cut.count > nought {
		for place := cut.next; place < cut.next+cut.count; place++ {
			face := collector.mesh.tris[place].face
			collector.found = append(collector.found, int(face))
		}
		return
	}
	collector.under(node + unity)
	collector.under(cut.next)
}

// held is the faces under tree node i, by their numbers in the mesh,
// smallest first.
func held(mesh *Mesh, i int32) []int {
	collector := &meshtreeMoreCollector{mesh: mesh}
	collector.under(i)
	slices.Sort(collector.found)
	return collector.found
}

// meshtreeMoreWalker goes down a tree, calling visit at every node.
type meshtreeMoreWalker struct {
	mesh  *Mesh
	visit func(i int32, depth int)
}

// down visits node i and then, if it is a cut, the nodes under it.
func (walker meshtreeMoreWalker) down(i int32, depth int) {
	walker.visit(i, depth)
	if cut := walker.mesh.nodes[i]; cut.count == nought {
		walker.down(i+unity, depth+unity)
		walker.down(cut.next, depth+unity)
	}
}

// walk calls visit for every node of the tree with its depth, the root at
// nought.
func walk(mesh *Mesh, visit func(i int32, depth int)) {
	walker := meshtreeMoreWalker{mesh: mesh, visit: visit}
	walker.down(meshtreeMoreRoot, nought)
}

// shapeOf writes the tree as its leaves' sizes: a leaf is its count, and a
// cut is its two halves in brackets.
func shapeOf(mesh *Mesh, i int32) string {
	cut := mesh.nodes[i]
	if cut.count > nought {
		return fmt.Sprint(cut.count)
	}
	return "(" + shapeOf(mesh, i+unity) + " " + shapeOf(mesh, cut.next) + ")"
}

// nest adds a triangle about the point centre for each of the sizes: its
// corners are c+(s,0,0), c+(0,s,0) and c+(-s,-s,0), which turn
// counterclockwise seen from above and add up to 3c. With c and s whole
// numbers each middle is c exactly, and each box is flat at c's height,
// 2s across and 2s up.
func nest(vertices *[]maths.Vec, faces *[][cornersOfAFace]int,
	centre maths.Vec, sizes ...float64) {
	for _, size := range sizes {
		first := len(*vertices)
		right := centre.Add(maths.V(size, nought, nought))
		upward := centre.Add(maths.V(nought, size, nought))
		behind := centre.Add(maths.V(-size, -size, nought))
		*vertices = append(*vertices, right, upward, behind)
		*faces = append(*faces, meshtreeMoreCorners(first))
	}
}

// grown is nest at each of the places in at, with the sizes for each, made
// into one mesh.
func grown(test testing.TB, at []maths.Vec, sizes ...[]float64) *Mesh {
	var vertices []maths.Vec
	var faces [][cornersOfAFace]int
	for i, centre := range at {
		nest(&vertices, &faces, centre, sizes[i]...)
	}
	return made(test, vertices, faces, nil, false)
}

// meshtreeMoreCensus is what a walk of a tree counts: how deep it goes,
// and how many faces its leaves hold, and which leaves hold too many.
type meshtreeMoreCensus struct {
	t        *testing.T
	why      string
	mesh     *Mesh
	leafMost int
	deepest  int
	total    int
}

// visit counts the node i at depth.
func (census *meshtreeMoreCensus) visit(i int32, depth int) {
	census.deepest = max(census.deepest, depth)
	count := census.mesh.nodes[i].count
	if int(count) > census.leafMost {
		census.t.Errorf("%s: a leaf holds %d faces", census.why, count)
		return
	}
	census.total += int(count)
}

// bounded holds a tree to what the code relies on: no leaf past eight
// faces, every face in a leaf, and nothing deeper than the 128 places a
// walk has to keep its way back.
func bounded(t *testing.T, why string, mesh *Mesh) {
	t.Helper()
	census := &meshtreeMoreCensus{t: t, why: why, mesh: mesh,
		leafMost: pickedCount(t, "meshtree-more-leaf-most")}
	walk(mesh, census.visit)
	if census.deepest >= pickedCount(t, "meshtree-more-walk-places") {
		t.Errorf("%s: the tree is %d deep", why, census.deepest)
	}
	if census.total != len(mesh.Faces) {
		t.Errorf("%s: the leaves hold %d of %d faces", why, census.total,
			len(mesh.Faces))
	}
}

// Faces with one middle cannot be told apart by where they are, so they
// are cut by count: up to eight make a leaf, and more are cut in halves,
// the smaller first.
//
// Worked by hand: nine cut at 9/2, four and five; ten, five and five;
// seventeen, eight and nine, and the nine again four and five.
func TestFacesWithOneMiddleAreHalvedByCount(t *testing.T) {
	for _, row := range readSheet(t, meshtreeMoreHalvings) {
		count := cellCount(t, row, "count")
		sizes := make([]float64, count)
		for i := range sizes {
			sizes[i] = float64(i + unity)
		}
		at := []maths.Vec{meshtreeMoreOrigin}
		mesh := grown(t, at, sizes)
		if got := shapeOf(mesh, meshtreeMoreRoot); got != row["want"] {
			t.Errorf("%d faces: %s, want %s", count, got, row["want"])
		}
		bounded(t, fmt.Sprint(count, " faces"), mesh)
	}
}

// Eight faces that lie almost on one another are searched as well by
// testing all eight as by cutting them, so they stay one leaf.
//
// Worked by hand: eight copies of one triangle, 10 across and 15 up, each
// an eighth along from the last, have boxes of area 300 and, all together,
// 326.25. The cheapest cut, four and four, costs 311.25*4 + 311.25*4 =
// 2490, and 1+2490/326.25 is 8.63, past the 8 a leaf costs.
func TestFacesThatOverlapStayOneLeaf(t *testing.T) {
	copies := pickedCount(t, "meshtree-more-overlap-copies")
	divisions := picked(t, "meshtree-more-overlap-divisions")
	base := picked(t, "meshtree-more-overlap-half-base")
	apex := picked(t, "meshtree-more-overlap-apex")
	var vertices []maths.Vec
	var faces [][cornersOfAFace]int
	for i := range copies {
		shift := maths.V(float64(i)/divisions, nought, nought)
		left := maths.V(-base, -base, nought).Add(shift)
		right := maths.V(base, -base, nought).Add(shift)
		top := maths.V(nought, apex, nought).Add(shift)
		vertices = append(vertices, left, right, top)
		faces = append(faces, meshtreeMoreCorners(cornersOfAFace*i))
	}
	mesh := made(t, vertices, faces, nil, false)
	if got := shapeOf(mesh, meshtreeMoreRoot); got != fmt.Sprint(copies) {
		t.Errorf("eight overlapping faces are %s", got)
	}
}

// More than eight faces are cut whatever a leaf would cost, for a leaf
// of more than eight is not allowed.
//
// Worked by hand: eight nested squares, 200 to 214 across, and a ninth 200
// across and one to the side. The eight are in a box of area 91592 at most,
// and the ninth in 80000, so the cut costs 8*91592 + 80000 = 812736 and
// 1+812736/91592 is 9.87, past the 9 a leaf of them would cost. Cut, they
// are eight and one.
func TestMoreThanEightFacesAreCutWhateverALeafCosts(t *testing.T) {
	var vertices []maths.Vec
	var faces [][cornersOfAFace]int
	eight := meshtreeMoreGroup(t, meshtreeMoreNested, "eight", "size")
	ninth := meshtreeMoreGroup(t, meshtreeMoreNested, "ninth", "size")
	aside := maths.V(picked(t, "meshtree-more-ninth-aside"), nought, nought)
	nest(&vertices, &faces, meshtreeMoreOrigin, eight...)
	nest(&vertices, &faces, aside, ninth...)
	mesh := made(t, vertices, faces, nil, false)
	want := fmt.Sprintf("(%d %d)", len(eight), len(ninth))
	if got := shapeOf(mesh, meshtreeMoreRoot); got != want {
		t.Errorf("nine faces, one off to the side: %s, want %s", got, want)
	}
	got := held(mesh, meshtreeMoreFirst)
	wantFaces := upTo(nought, len(eight)-unity)
	if !slices.Equal(got, wantFaces) {
		t.Errorf("the first half holds %v", got)
	}
}

// Faces far enough apart that cutting costs less than testing them are
// cut, even when there are few: two are a leaf, as any two are, and three
// are cut in a one and a two.
//
// Worked by hand: three squares 2 across, at 0, 100 and 200 along, cost
// 8 + 2*408 = 824 for either cut, over 808 for the box. 1+824/808 is 2.02,
// under the 3 a leaf of them costs.
func TestThreeFacesFarApartAreCut(t *testing.T) {
	size := []float64{picked(t, "meshtree-more-far-size")}
	one := maths.V(picked(t, "meshtree-more-far-one-along"), nought, nought)
	two := maths.V(picked(t, "meshtree-more-far-two-along"), nought, nought)
	twice := grown(t, []maths.Vec{meshtreeMoreOrigin, one}, size, size)
	if got := shapeOf(twice, meshtreeMoreRoot); got != "2" {
		t.Errorf("two faces far apart are %s, want 2", got)
	}
	three := grown(t, []maths.Vec{meshtreeMoreOrigin, one, two}, size, size,
		size)
	got := shapeOf(three, meshtreeMoreRoot)
	if got != "(1 2)" && got != "(2 1)" {
		t.Errorf("three faces far apart are %s, want a cut", got)
	}
}

// Two clusters a long way apart are split between them at the root, each
// whole in a leaf of its own, the lower one first.
//
// Worked by hand: the five are squares up to 10 across, area 200 at most,
// and the three up to 6, area 72, so the cut costs 5*200 + 3*72 = 1216,
// over 20160 for the box: 1.06 against the 8 a leaf of eight costs.
func TestTwoFarClustersAreSplitAtTheRoot(t *testing.T) {
	five := meshtreeMoreGroup(t, meshtreeMoreClusters, "five", "size")
	three := meshtreeMoreGroup(t, meshtreeMoreClusters, "three", "size")
	apart := maths.V(picked(t, "meshtree-more-cluster-apart"), nought, nought)
	mesh := grown(t, []maths.Vec{meshtreeMoreOrigin, apart}, five, three)
	want := fmt.Sprintf("(%d %d)", len(five), len(three))
	if got := shapeOf(mesh, meshtreeMoreRoot); got != want {
		t.Fatalf("two clusters are %s, want %s", got, want)
	}
	lower := upTo(nought, len(five)-unity)
	if got := held(mesh, meshtreeMoreFirst); !slices.Equal(got, lower) {
		t.Errorf("the first cluster is %v", got)
	}
	upper := upTo(len(five), len(five)+len(three)-unity)
	second := mesh.nodes[meshtreeMoreRoot].next
	if got := held(mesh, second); !slices.Equal(got, upper) {
		t.Errorf("the second cluster is %v", got)
	}
}

// cutAt is the faces on each side of the root of a mesh of squares 2
// across, one at each of the places, which the root must have cut in two.
func cutAt(t *testing.T, at ...maths.Vec) (low, high []int) {
	t.Helper()
	size := []float64{picked(t, "meshtree-more-cut-size")}
	sizes := make([][]float64, len(at))
	for i := range sizes {
		sizes[i] = size
	}
	mesh := grown(t, at, sizes...)
	root := mesh.nodes[meshtreeMoreRoot]
	if root.count > nought {
		t.Fatalf("the root of %d faces is a leaf", len(at))
	}
	return held(mesh, meshtreeMoreFirst), held(mesh, root.next)
}

// A cut that leaves one face alone is a cut like any other, at whichever
// end the one is.
//
// Worked by hand, with a square of area 8 at each place, and the middles'
// bins by sixteenths of the 100 along x. Faces at x = 0 (50 up), 7 and 100
// are in bins 0, 1 and 15: the first alone costs 8 + 2*380 = 768, and the
// first two, 2*936 + 8 = 1880. Faces at x = 0, 93 and 100 (50 up) are in
// bins 0, 14 and 15: the last alone costs 2*380 + 8 = 768, and the first
// alone, 8 + 2*936 = 1880.
func TestAFaceAloneAtEitherEndIsCutOff(t *testing.T) {
	first := meshtreeMorePlacesOf(t, meshtreeMoreCuts, "first-alone")
	low, high := cutAt(t, first...)
	if !slices.Equal(low, []int{nought}) ||
		!slices.Equal(high, []int{unity, pair}) {
		t.Errorf("the first alone: %v and %v", low, high)
	}
	last := meshtreeMorePlacesOf(t, meshtreeMoreCuts, "last-alone")
	low, high = cutAt(t, last...)
	if !slices.Equal(low, []int{nought, unity}) ||
		!slices.Equal(high, []int{pair}) {
		t.Errorf("the last alone: %v and %v", low, high)
	}
}

// The cut taken is the one where the two halves cost least, after
// whichever bin that is.
//
// Worked by hand: squares 2 across at x = 0, 10, 20 and 100 are in bins
// 0, 1, 3 and 15 of the 100. The first alone costs 8 + 3*368 = 1112; the
// first two, 2*48 + 2*328 = 752; the first three, 3*88 + 8 = 272.
func TestTheCheapestCutIsTheOneTaken(t *testing.T) {
	places := meshtreeMorePlacesOf(t, meshtreeMoreCuts, "four-in-a-row")
	low, high := cutAt(t, places...)
	last := len(places) - unity
	if !slices.Equal(low, upTo(nought, pair)) ||
		!slices.Equal(high, []int{last}) {
		t.Errorf("four in a row: %v and %v", low, high)
	}
}

// chain is count triangles in a row along x, each ten times the size of
// the last and ten times as far out: the one of rank k has corners
// (x-s,-s), (x+s,-s) and (x,2s) with x = sign * 10^k and s = x/10. The
// faces are numbered out of order, and the rank of face j is rank[j].
func chain(t *testing.T, count int, sign float64) (*Mesh, []int) {
	t.Helper()
	stride := pickedCount(t, "meshtree-more-chain-stride")
	tenth := picked(t, "meshtree-more-chain-tenth")
	var vertices []maths.Vec
	var faces [][cornersOfAFace]int
	rank := make([]int, count)
	for face := range count {
		k := (face * stride) % count
		rank[face] = k
		x, size := sign*math.Pow10(k), math.Pow10(k)/tenth
		left := maths.V(x-size, -size, nought)
		right := maths.V(x+size, -size, nought)
		top := maths.V(x, pair*size, nought)
		vertices = append(vertices, left, right, top)
		faces = append(faces, meshtreeMoreCorners(cornersOfAFace*face))
	}
	return made(t, vertices, faces, nil, false), rank
}

// ranks is the ranks of the faces under node i, smallest first.
func ranks(mesh *Mesh, rank []int, i int32) []int {
	var out []int
	for _, face := range held(mesh, i) {
		out = append(out, rank[face])
	}
	slices.Sort(out)
	return out
}

// upTo is the ranks from low to high, both included.
func upTo(low, high int) []int {
	var out []int
	for k := low; k <= high; k++ {
		out = append(out, k)
	}
	return out
}

// meshtreeMoreChained is a chain, with the rank of each of its faces and
// how many faces it has.
type meshtreeMoreChained struct {
	mesh   *Mesh
	rank   []int
	length int
}

// step checks the node at depth of a chain of the given sign: it holds
// every rank but the highest depth ones, and cuts off the highest alone.
// It is the node of the rest that step gives back.
func (chained meshtreeMoreChained) step(t *testing.T, sign float64,
	at int32, depth int) int32 {
	t.Helper()
	present := ranks(chained.mesh, chained.rank, at)
	want := upTo(nought, chained.length-unity-depth)
	if !slices.Equal(present, want) {
		t.Fatalf("chain %v depth %d: ranks %v", sign, depth, present)
	}
	rest, alone := at+unity, chained.mesh.nodes[at].next
	if sign < nought {
		rest, alone = alone, rest
	}
	cut := ranks(chained.mesh, chained.rank, alone)
	if chained.mesh.nodes[alone].count != unity ||
		!slices.Equal(cut, []int{chained.length - unity - depth}) {
		t.Fatalf("chain %v depth %d: the one alone is %v", sign, depth, cut)
	}
	return rest
}

// Faces that each stand ten times as far out as the one before are cut
// one at a time, the outermost alone, down to depth 64. From there a node
// is cut in half by count, at the median of the middles, the lower half
// first.
//
// Worked by hand: of m faces at 10^0 to 10^(m-1), the farthest, at p, is
// in bin 15, the next in bin 1 and the rest in bin 0. Cutting off the
// farthest costs about (0.0066m + 0.12) p squared, and cutting off the
// two about 1.2 p squared, so the farthest goes alone. The node at depth
// d holds ranks 0 to 79-d; at depth 64 it holds 16, and cuts them 8 and 8.
//
// Mirrored, it is the lowest face, which is the farthest out, that goes
// alone, and the halves are the other way about.
func TestACutPerFaceThenHalvesAtTheDepthLimit(t *testing.T) {
	length := pickedCount(t, "meshtree-more-chain-length")
	limit := pickedCount(t, "meshtree-more-depth-limit")
	each := pickedCount(t, "meshtree-more-halved-each")
	for _, sign := range []float64{unity, -unity} {
		mesh, rank := chain(t, length, sign)
		chained := meshtreeMoreChained{mesh: mesh, rank: rank, length: length}
		bounded(t, fmt.Sprint("chain ", sign), mesh)
		at := meshtreeMoreRoot
		for depth := nought; depth < limit; depth++ {
			at = chained.step(t, sign, at, depth)
		}
		first, second := at+unity, mesh.nodes[at].next
		low, high := upTo(nought, each-unity), upTo(each, each+each-unity)
		if sign < nought {
			low, high = high, low
		}
		if int(mesh.nodes[first].count) != each ||
			int(mesh.nodes[second].count) != each ||
			!slices.Equal(ranks(mesh, rank, first), low) ||
			!slices.Equal(ranks(mesh, rank, second), high) {
			t.Errorf("chain %v depth %d: halves %v and %v, want %v and %v",
				sign, limit, ranks(mesh, rank, first),
				ranks(mesh, rank, second), low, high)
		}
	}
}

// meshtreeMoreHairCase is a mesh, and the hair its boxes are expected to
// be let out by: worked by hand, or nought where the formula is left to
// say.
type meshtreeMoreHairCase struct {
	why  string
	mesh *Mesh
	hair float64
}

// meshtreeMoreHairCheck is what a walk of a mesh's tree checks: that the
// box of each node is the box of its faces let out by a hair.
type meshtreeMoreHairCheck struct {
	t    *testing.T
	why  string
	mesh *Mesh
	hair float64
}

// visit checks the box of node i, at depth.
func (check meshtreeMoreHairCheck) visit(i int32, depth int) {
	faces := spanOf(check.mesh, held(check.mesh, i))
	got := check.mesh.nodes[i].box
	hair := check.hair
	want := meshtreeMoreBox(
		maths.V(faces.lo[meshtreeMoreAxisX]-hair,
			faces.lo[meshtreeMoreAxisY]-hair,
			faces.lo[meshtreeMoreAxisZ]-hair),
		maths.V(faces.hi[meshtreeMoreAxisX]+hair,
			faces.hi[meshtreeMoreAxisY]+hair,
			faces.hi[meshtreeMoreAxisZ]+hair))
	if got != want {
		check.t.Errorf("%s: node %d at depth %d is %v, want %v", check.why,
			i, depth, got, want)
	}
}

// meshtreeMoreCheckHair walks the tree of a case, with the hair a mesh's
// longest side leaves as a share, and holds every box to it.
func meshtreeMoreCheckHair(t *testing.T, each meshtreeMoreHairCase,
	share float64) {
	t.Helper()
	mesh := each.mesh
	bounds := mesh.Bounds()
	size := bounds.Max.Sub(bounds.Min)
	hair := share * math.Max(size.X, math.Max(size.Y, size.Z))
	if each.hair != nought && hair != each.hair {
		t.Fatalf("%s: the sides are %v, so the hair is %g, not %g",
			each.why, size, hair, each.hair)
	}
	check := meshtreeMoreHairCheck{t: t, why: each.why, mesh: mesh,
		hair: hair}
	walk(mesh, check.visit)
}

// Every node's box is the box of its faces let out by a hair on every
// side, so that a ray along a face's edge is not turned away by it. The
// hair is a billionth of the longest side of the whole mesh.
//
// Worked by hand for the first mesh, a cube from -1 to 1 stretched four
// times up and twice deep: its sides are 2, 8 and 4, so the hair is 8e-9.
// The faces' boxes are found here from the corners, face by face.
func TestEveryBoxIsItsFacesLetOutByAHair(t *testing.T) {
	rng := meshtreeMoreRandom(t, "hair")
	soupFaces := pickedCount(t, "meshtree-more-hair-soup-faces")
	soupVertices, soupCorners := soup(rng, soupFaces)
	cubeCells := pickedCount(t, "meshtree-more-hair-cube-cells")
	cubeVertices, cubeCorners := cube(cubeCells, false)
	up := picked(t, "meshtree-more-hair-stretch-up")
	deep := picked(t, "meshtree-more-hair-stretch-deep")
	for i := range cubeVertices {
		corner := cubeVertices[i]
		cubeVertices[i] = maths.V(corner.X, up*corner.Y, deep*corner.Z)
	}
	length := pickedCount(t, "meshtree-more-chain-length")
	chained, _ := chain(t, length, unity)
	stretched := made(t, cubeVertices, cubeCorners, nil, false)
	soupy := made(t, soupVertices, soupCorners, nil, false)
	byHand := picked(t, "meshtree-more-hair-stretched")
	share := picked(t, "meshtree-more-hair-share")
	for _, each := range []meshtreeMoreHairCase{
		{"a stretched cube", stretched, byHand},
		{"a soup", soupy, nought},
		{"a chain", chained, nought},
	} {
		meshtreeMoreCheckHair(t, each, share)
	}
}

// boxAt is where a ray from at, going along way, enters the box from low to
// high, and whether it does before far, from near.
func boxAt(at, way, low, high maths.Vec, near, far float64) (float64, bool) {
	query := ready(meshtreeMoreRay(at, way))
	box := meshtreeMoreBox(low, high)
	return query.box(&box, near, far)
}

// The box test says where a ray enters a box, and whether it does.
//
// Worked by hand against the box from 0 to 1 on each axis: along each
// axis the ray is in the slab from (lo-at)/way to (hi-at)/way, and it
// enters the box at the latest of the slabs' starts and no sooner than
// near, if that is no later than the earliest of their ends and far.
func TestARayEntersABoxWhereItsSlabsSay(t *testing.T) {
	one := maths.V(unity, unity, unity)
	for _, row := range readSheet(t, meshtreeMoreSlabs) {
		at := meshtreeMoreVector(t, row, "at")
		way := meshtreeMoreVector(t, row, "way")
		near, far := cell(t, row, "near"), cell(t, row, "far")
		enter, ok := boxAt(at, way, meshtreeMoreOrigin, one, near, far)
		wantEnter := cell(t, row, "enter")
		wantOK := meshtreeMoreFlag(t, row, "enters")
		if ok != wantOK || (ok && enter != wantEnter) {
			t.Errorf("%s: enters at %v, %v; want %v, %v", row["name"],
				enter, ok, wantEnter, wantOK)
		}
	}
}

// A slab's distances are a little uncertain, so a ray whose slabs miss
// each other by no more than a trillionth of the leaving is let through,
// and one that misses by more is not.
//
// Worked by hand: along (1,-1,0) from x = 0, the x slab is left at 1, and
// the y slab is entered at (at.y - 1), which for at.y of 2 plus 2^-41
// (4.5e-13) is 1 plus 4.5e-13, and for 2^-39 (1.8e-12) 1 plus 1.8e-12.
func TestASlabMissedByAHairIsLetThrough(t *testing.T) {
	one := maths.V(unity, unity, unity)
	height := picked(t, "meshtree-more-miss-at-y")
	way := maths.V(unity, -unity, nought)
	for _, row := range readSheet(t, meshtreeMoreMisses) {
		miss := cell(t, row, "miss")
		at := maths.V(nought, height+miss, half)
		enter, ok := boxAt(at, way, meshtreeMoreOrigin, one, nought, forever)
		if want := meshtreeMoreFlag(t, row, "lets"); ok != want {
			t.Errorf("missing by %g: %v at %v, want %v", miss, ok, enter,
				want)
		}
	}
}

// area is a box's whole surface: for 2 by 3 by 5, twice 6+15+10.
func TestTheAreaOfABox(t *testing.T) {
	sides := maths.V(picked(t, "meshtree-more-area-width"),
		picked(t, "meshtree-more-area-length"),
		picked(t, "meshtree-more-area-height"))
	want := picked(t, "meshtree-more-area-surface")
	if got := area(meshtreeMoreBox(meshtreeMoreOrigin, sides)); got != want {
		t.Errorf("a box 2 by 3 by 5 has area %v, want %v", got, want)
	}
}

// longest is the axis along which a box is longest. Boxes are given by
// their sides, each axis the longest in turn, and one where the first is
// longer than the second but not than the third.
func TestTheLongestAxisOfABox(t *testing.T) {
	for _, row := range readSheet(t, meshtreeMoreLongest) {
		sides := meshtreeMoreVector(t, row, "side")
		box := meshtreeMoreBox(meshtreeMoreOrigin, sides)
		want := cellCount(t, row, "axis")
		if got := longest(box); got != want {
			t.Errorf("a box %v across: axis %d, want %d", sides, got, want)
		}
	}
}

// along is a point's part on the axis 0, 1 or 2.
func TestAPointAlongAnAxis(t *testing.T) {
	row := readSheet(t, meshtreeMorePlaces).row(t, meshtreeMorePlaces,
		"along")
	point := meshtreeMoreVector(t, row, "point")
	for axis, want := range meshtreeMoreParts(point) {
		if got := along(point, axis); got != want {
			t.Errorf("axis %d of %v is %v, want %v", axis, point, got, want)
		}
	}
}

// away is how far a point is from a box, squared, on each axis and past
// either end: from the box of 0 to 1, a point 2 past a side is 4 away,
// and one 3 past on one axis and 2 and 1 on the others is 9+4+1.
func TestHowFarAPointIsFromABox(t *testing.T) {
	box := meshtreeMoreBox(meshtreeMoreOrigin, maths.V(unity, unity, unity))
	for _, row := range readSheet(t, meshtreeMoreAway) {
		point := meshtreeMoreVector(t, row, "point")
		want := cell(t, row, "square-x") + cell(t, row, "square-y") +
			cell(t, row, "square-z")
		if got := away(box, point); got != want {
			t.Errorf("%v is %v from the box, want %v", point, got, want)
		}
	}
}

// meshtreeMoreAsker puts the questions to a mesh, over a point at aim
// along x and along y.
type meshtreeMoreAsker struct {
	aim float64
}

// bounds asks for the box.
func (asker meshtreeMoreAsker) bounds(mesh *Mesh) { mesh.Bounds() }

// hit asks for the hit of a ray from above, straight down.
func (asker meshtreeMoreAsker) hit(mesh *Mesh) {
	mesh.Hit(down(asker.aim, asker.aim), nought, forever)
}

// normal asks for the normal at a point.
func (asker meshtreeMoreAsker) normal(mesh *Mesh) {
	mesh.Normal(maths.V(asker.aim, asker.aim, unity))
}

// distance asks for the distance to a point.
func (asker meshtreeMoreAsker) distance(mesh *Mesh) {
	mesh.Distance(maths.V(asker.aim, asker.aim, unity))
}

// meshtreeMoreSayPanic is deferred to say what a question panicked with:
// it is a fault if that is not want.
func meshtreeMoreSayPanic(t *testing.T, name, want string) {
	if got := recover(); got != want {
		t.Errorf("%s: panicked with %v, want %q", name, got, want)
	}
}

// meshtreeMoreAskUnbuilt puts a question to a Mesh that was not built.
func meshtreeMoreAskUnbuilt(t *testing.T, name string, ask func(*Mesh),
	want string) {
	defer meshtreeMoreSayPanic(t, name, want)
	ask(&Mesh{Vertices: []maths.Vec{meshtreeMoreOrigin, maths.XAxis,
		maths.Up}, Faces: [][cornersOfAFace]int{{nought, unity, pair}}})
}

// A Mesh not made by NewMesh says what is wrong with it, in words, from
// whichever question is put to it.
func TestAMeshNotBuiltSaysWhatIsWrong(t *testing.T) {
	const want = "shape: a Mesh is made by NewMesh, which builds its tree"
	asker := meshtreeMoreAsker{aim: picked(t, "meshtree-more-unbuilt-aim")}
	for name, ask := range map[string]func(mesh *Mesh){
		"Bounds":   asker.bounds,
		"Hit":      asker.hit,
		"Normal":   asker.normal,
		"Distance": asker.distance,
	} {
		meshtreeMoreAskUnbuilt(t, name, ask, want)
	}
}

// The face nearest a point is the one face of a mesh of one, wherever the
// point is, even so far out that every distance squared is past what a
// double holds: worked by hand, 1e200 squared is endless.
func TestTheOnlyFaceIsNearestFromAnywhere(t *testing.T) {
	mesh := lone(t)
	out := picked(t, "meshtree-more-far-out")
	want := maths.V(nought, nought, unity)
	if got := mesh.Normal(maths.V(out, nought, nought)); got != want {
		t.Errorf("the normal from 1e200 out is %v, want up", got)
	}
}

// winding is how many more faces a ray leaves through than it enters by.
//
// Worked by hand on the cube from -1 to 1: from inside, out through one
// face; from outside going away, none; through the middle, in and out;
// from inside it and a cube twice the size round it, out through both;
// from between the two, out through the larger alone.
func TestWindingCountsFacesLeftLessFacesEntered(t *testing.T) {
	cells := pickedCount(t, "meshtree-more-winding-cells")
	vertices, faces := cube(cells, false)
	one := made(t, vertices, faces, nil, false)
	count, inner := len(vertices), len(faces)
	for i := range count {
		vertices = append(vertices, vertices[i].Scale(pair))
	}
	for _, face := range faces[:inner] {
		faces = append(faces, meshtreeMoreShifted(face, count))
	}
	meshes := map[string]*Mesh{"one": one,
		"two": made(t, vertices, faces, nil, false)}
	for _, row := range readSheet(t, meshtreeMoreWindings) {
		at := meshtreeMoreVector(t, row, "at")
		direction := meshtreeMoreVector(t, row, "dir")
		ray := meshtreeMoreRay(at, direction)
		want := cellCount(t, row, "want")
		if got := meshes[row["mesh"]].winding(ray); got != want {
			t.Errorf("%s: %d, want %d", row["name"], got, want)
		}
	}
}

// span is the box of some faces, found from their corners.
type span struct{ lo, hi [meshtreeMoreAxes]float64 }

// spanOf is the box of the faces of the mesh numbered in faces.
func spanOf(mesh *Mesh, faces []int) span {
	var lowest, highest [meshtreeMoreAxes]float64
	for k := range lowest {
		lowest[k], highest[k] = forever, -forever
	}
	found := span{lowest, highest}
	for _, face := range faces {
		for _, vertex := range mesh.Faces[face] {
			point := mesh.Vertices[vertex]
			for k, part := range meshtreeMoreParts(point) {
				found.lo[k] = math.Min(found.lo[k], part)
				found.hi[k] = math.Max(found.hi[k], part)
			}
		}
	}
	return found
}

// surface is twice the sum of the three products of the sides.
func (found span) surface() float64 {
	x := found.hi[meshtreeMoreAxisX] - found.lo[meshtreeMoreAxisX]
	y := found.hi[meshtreeMoreAxisY] - found.lo[meshtreeMoreAxisY]
	z := found.hi[meshtreeMoreAxisZ] - found.lo[meshtreeMoreAxisZ]
	return pair * (x*y + y*z + z*x)
}

// meshtreeMorePlanner is what plan works with: the mesh, and the numbers
// its tree is cut by.
type meshtreeMorePlanner struct {
	mesh       *Mesh
	alwaysLeaf int
	leafMost   int
	depthLimit int
	bins       int
	lastBin    int
	tieBelow   float64
	tieAbove   float64
}

// meshtreeMoreNewPlanner is a planner for a mesh.
func meshtreeMoreNewPlanner(t *testing.T, mesh *Mesh) meshtreeMorePlanner {
	t.Helper()
	return meshtreeMorePlanner{
		mesh:       mesh,
		alwaysLeaf: pickedCount(t, "meshtree-more-plan-always-leaf"),
		leafMost:   pickedCount(t, "meshtree-more-leaf-most"),
		depthLimit: pickedCount(t, "meshtree-more-depth-limit"),
		bins:       pickedCount(t, "meshtree-more-plan-bins"),
		lastBin:    pickedCount(t, "meshtree-more-plan-last-bin"),
		tieBelow:   picked(t, "meshtree-more-plan-tie-below"),
		tieAbove:   picked(t, "meshtree-more-plan-tie-above"),
	}
}

// meshtreeMoreSpread is where the middles of some faces lie: each one, and
// the lowest and the highest on each axis.
type meshtreeMoreSpread struct {
	middles   [][meshtreeMoreAxes]float64
	low, high [meshtreeMoreAxes]float64
}

// widest is the axis the middles spread furthest on, the first of equals.
func (spread meshtreeMoreSpread) widest() int {
	axis := meshtreeMoreAxisX
	for k := meshtreeMoreAxisY; k < meshtreeMoreAxes; k++ {
		if spread.high[k]-spread.low[k] >
			spread.high[axis]-spread.low[axis] {
			axis = k
		}
	}
	return axis
}

// middleOf is the middle of a face of the planner's mesh: its corners
// added up and divided by their count.
func (planner meshtreeMorePlanner) middleOf(
	face int) [meshtreeMoreAxes]float64 {
	var sum, middle [meshtreeMoreAxes]float64
	for _, vertex := range planner.mesh.Faces[face] {
		point := planner.mesh.Vertices[vertex]
		for k, part := range meshtreeMoreParts(point) {
			sum[k] += part
		}
	}
	for k := range sum {
		middle[k] = sum[k] / float64(cornersOfAFace)
	}
	return middle
}

// spreadOf is where the middles of the faces lie.
func (planner meshtreeMorePlanner) spreadOf(faces []int) meshtreeMoreSpread {
	spread := meshtreeMoreSpread{
		middles: make([][meshtreeMoreAxes]float64, len(faces))}
	for i, face := range faces {
		middle := planner.middleOf(face)
		spread.middles[i] = middle
		for k := range middle {
			if i == nought {
				spread.low[k], spread.high[k] = middle[k], middle[k]
			}
			spread.low[k] = math.Min(spread.low[k], middle[k])
			spread.high[k] = math.Max(spread.high[k], middle[k])
		}
	}
	return spread
}

// meshtreeMorePrices are the cuts tried between the bins, the cost of each
// by the bin it is before, the faces below it, and the least cost.
type meshtreeMorePrices struct {
	cost  []float64
	below [][]int
	best  float64
}

// divide is the faces below the bin cut, and the faces from it up.
func (planner meshtreeMorePlanner) divide(faces []int,
	spread meshtreeMoreSpread, axis, cut int) (under, over []int) {
	width := spread.high[axis] - spread.low[axis]
	for i, face := range faces {
		place := float64(planner.bins) *
			(spread.middles[i][axis] - spread.low[axis]) / width
		if min(int(place), planner.lastBin) < cut {
			under = append(under, face)
		} else {
			over = append(over, face)
		}
	}
	return under, over
}

// price tries every cut between two bins along the axis.
func (planner meshtreeMorePlanner) price(faces []int,
	spread meshtreeMoreSpread, axis int) meshtreeMorePrices {
	prices := meshtreeMorePrices{cost: make([]float64, planner.bins),
		below: make([][]int, planner.bins), best: forever}
	for k := unity; k < planner.bins; k++ {
		under, over := planner.divide(faces, spread, axis, k)
		if len(under) == nought || len(over) == nought {
			continue
		}
		prices.cost[k] = spanOf(planner.mesh, under).surface()*
			float64(len(under)) +
			spanOf(planner.mesh, over).surface()*float64(len(over))
		prices.below[k] = under
		prices.best = math.Min(prices.best, prices.cost[k])
	}
	return prices
}

// meshtreeMoreHas is whether cut is one of cuts.
func meshtreeMoreHas(cuts [][]int, cut []int) bool {
	for _, each := range cuts {
		if slices.Equal(each, cut) {
			return true
		}
	}
	return false
}

// choices says whether a leaf is allowed, and which cuts are as cheap as
// the cheapest, for faces cut by price.
func (planner meshtreeMorePlanner) choices(faces []int,
	spread meshtreeMoreSpread, axis int) (leaf bool, cuts [][]int) {
	prices := planner.price(faces, spread, axis)
	count := len(faces)
	cheaper := unity + prices.best/spanOf(planner.mesh, faces).surface()
	leaf = count <= planner.leafMost &&
		cheaper >= float64(count)*planner.tieBelow
	if count <= planner.leafMost &&
		cheaper >= float64(count)*planner.tieAbove {
		return leaf, nil
	}
	for k := unity; k < planner.bins; k++ {
		if prices.below[k] != nil &&
			prices.cost[k] <= prices.best*planner.tieAbove {
			slices.Sort(prices.below[k])
			if !meshtreeMoreHas(cuts, prices.below[k]) {
				cuts = append(cuts, prices.below[k])
			}
		}
	}
	return leaf, cuts
}

// meshtreeMoreByAxis orders faces by where their middles lie on an axis.
type meshtreeMoreByAxis struct {
	faces   []int
	order   []int
	middles [][meshtreeMoreAxes]float64
	axis    int
}

// less is whether the face at place first of the order lies before the
// face at place second.
func (by meshtreeMoreByAxis) less(first, second int) bool {
	lower := by.middles[slices.Index(by.faces, by.order[first])]
	higher := by.middles[slices.Index(by.faces, by.order[second])]
	return lower[by.axis] < higher[by.axis]
}

// meshtreeMoreLowerHalf is the lower half of the faces by their order along
// the axis, smallest numbers first.
func meshtreeMoreLowerHalf(faces []int, spread meshtreeMoreSpread,
	axis int) []int {
	order := slices.Clone(faces)
	by := meshtreeMoreByAxis{faces: faces, order: order,
		middles: spread.middles, axis: axis}
	sort.Slice(order, by.less)
	lower := slices.Clone(order[:len(faces)/pair])
	slices.Sort(lower)
	return lower
}

// plan is what the tree may do with the faces of the mesh at a depth, by
// brute force from their corners alone, trying every cut. Along the axis
// the middles spread furthest on, the faces fall into sixteen bins, and
// the cut between two bins is cheap where the faces below times the
// surface of their box, and the faces above times theirs, come to least.
//
// A leaf of eight or fewer is kept instead when a cut, priced as a box
// and the cost over the box's surface, is no cheaper than the faces tested
// one by one. Past depth 64, or without a spread, eight or fewer make a
// leaf and more are cut in half by their order along the axis.
//
// Prices within a billionth of each other are taken as equal, so either
// choice is allowed: leaf says a leaf is, and cuts are the first halves
// that are.
func (planner meshtreeMorePlanner) plan(faces []int,
	depth int) (leaf bool, cuts [][]int) {
	count := len(faces)
	if count <= planner.alwaysLeaf {
		return true, nil
	}
	spread := planner.spreadOf(faces)
	axis := spread.widest()
	if depth < planner.depthLimit && spread.high[axis] > spread.low[axis] {
		return planner.choices(faces, spread, axis)
	}
	if count <= planner.leafMost {
		return true, nil
	}
	return false, [][]int{meshtreeMoreLowerHalf(faces, spread, axis)}
}

// scatter is count triangles of about the given size, strewn through the
// cube from -spread to spread, overlapping as they fall.
func scatter(rng *rand.Rand, count int, spread,
	size float64) ([]maths.Vec, [][cornersOfAFace]int) {
	var vertices []maths.Vec
	var faces [][cornersOfAFace]int
	for range count {
		middle := within(rng, spread)
		scale := size * (half + rng.Float64())
		for range cornersOfAFace {
			direction := anyway(rng).Scale(scale)
			vertices = append(vertices, middle.Add(direction))
		}
		first := len(vertices) - cornersOfAFace
		faces = append(faces, meshtreeMoreCorners(first))
	}
	return vertices, faces
}

// lumps is count lumps of small triangles each, strewn through the cube
// from the reach to minus it.
func lumps(test testing.TB, rng *rand.Rand,
	count int) ([]maths.Vec, [][cornersOfAFace]int) {
	each := int(meshtreeMorePicked(test, "meshtree-more-lump-faces"))
	spread := meshtreeMorePicked(test, "meshtree-more-lump-spread")
	size := meshtreeMorePicked(test, "meshtree-more-lump-size")
	reach := meshtreeMorePicked(test, "meshtree-more-lump-reach")
	var vertices []maths.Vec
	var faces [][cornersOfAFace]int
	for range count {
		lumpVertices, lumpFaces := scatter(rng, each, spread, size)
		centre, base := within(rng, reach), len(vertices)
		for _, point := range lumpVertices {
			vertices = append(vertices, point.Add(centre))
		}
		for _, face := range lumpFaces {
			faces = append(faces, meshtreeMoreShifted(face, base))
		}
	}
	return vertices, faces
}

// meshtreeMoreBrute is what a walk of a tree checks: that each node is a
// leaf or a cut that brute force allows there.
type meshtreeMoreBrute struct {
	t       *testing.T
	name    string
	mesh    *Mesh
	planner meshtreeMorePlanner
	leaves  int
	cuts    int
}

// visit checks node i at depth.
func (brute *meshtreeMoreBrute) visit(i int32, depth int) {
	faces := held(brute.mesh, i)
	leaf, halves := brute.planner.plan(faces, depth)
	if brute.mesh.nodes[i].count > nought {
		brute.leaves++
		if !leaf {
			brute.t.Errorf("%s: node %d at depth %d of %d faces is a "+
				"leaf, and a cut is meant", brute.name, i, depth,
				len(faces))
		}
		return
	}
	brute.cuts++
	got := held(brute.mesh, i+unity)
	if !meshtreeMoreHas(halves, got) {
		brute.t.Errorf("%s: node %d at depth %d of %d faces is cut "+
			"%d and %d, and no cut brute force allows does that",
			brute.name, i, depth, len(faces), len(got), len(faces)-len(got))
	}
}

// meshtreeMoreBruteMeshes is the meshes the brute force is tried on, the
// soup, the thin, the thick, the lumps and the chains, by name.
func meshtreeMoreBruteMeshes(t *testing.T) ([]string, []*Mesh) {
	t.Helper()
	rng := meshtreeMoreRandom(t, "brute")
	names := []string{"a soup", "thin", "thick", "lumps", "chain", "mirror"}
	var meshes []*Mesh
	vertices, faces := soup(rng, pickedCount(t,
		"meshtree-more-brute-soup-faces"))
	meshes = append(meshes, made(t, vertices, faces, nil, false))
	vertices, faces = scatter(rng, pickedCount(t,
		"meshtree-more-brute-thin-faces"),
		picked(t, "meshtree-more-brute-thin-spread"),
		picked(t, "meshtree-more-brute-thin-size"))
	meshes = append(meshes, made(t, vertices, faces, nil, false))
	vertices, faces = scatter(rng, pickedCount(t,
		"meshtree-more-brute-thick-faces"),
		picked(t, "meshtree-more-brute-thick-spread"),
		picked(t, "meshtree-more-brute-thick-size"))
	meshes = append(meshes, made(t, vertices, faces, nil, false))
	vertices, faces = lumps(t, rng, pickedCount(t,
		"meshtree-more-brute-lumps"))
	meshes = append(meshes, made(t, vertices, faces, nil, false))
	length := pickedCount(t, "meshtree-more-chain-length")
	upward, _ := chain(t, length, unity)
	mirrored, _ := chain(t, length, -unity)
	return names, append(meshes, upward, mirrored)
}

// Every node of the tree is a leaf or a cut that brute force allows there,
// found again from the corners, for faces strewn thinly, thickly and in
// lumps, and for the chains.
func TestEveryNodeIsCutAsBruteForceAllows(t *testing.T) {
	names, meshes := meshtreeMoreBruteMeshes(t)
	for which, mesh := range meshes {
		brute := &meshtreeMoreBrute{t: t, name: names[which], mesh: mesh,
			planner: meshtreeMoreNewPlanner(t, mesh)}
		walk(mesh, brute.visit)
		bounded(t, names[which], mesh)
		t.Logf("%s: %d faces, %d leaves, %d cuts", names[which],
			len(mesh.Faces), brute.leaves, brute.cuts)
	}
}
