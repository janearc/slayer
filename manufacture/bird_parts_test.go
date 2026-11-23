package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// Helpers for the tests of the parts a bird gains: its lesser coverts,
// scapulars, claws, tarsal scutes, eye rings and the carry of a long
// folded wing.

// birdFloor is how near two worked lengths must lie to be called the same,
// in metres: float rounding only.
const birdFloor = 1e-9

// birdTiniest is the least extent a part may have along each axis to be a
// part and not a speck, in metres: a tenth of a millimetre.
const birdTiniest = 1e-4

// birdEachBody calls check with the body of every species at the test
// seed.
func birdEachBody(t *testing.T, check func(key string, body *birdBody)) {
	t.Helper()
	for _, key := range birdRead.keys() {
		check(key, birdTestBody(t, key))
	}
}

// birdEnds are the head and the tail of the bone named name, at rest.
func birdEnds(t *testing.T, body *birdBody, name string) (maths.Vec,
	maths.Vec) {
	t.Helper()
	at, ok := body.bones.Find(name)
	if !ok {
		t.Fatalf("no bone named %s", name)
	}
	bone := body.bones.Bone(at)
	return bone.Head(), bone.Tail()
}

// birdParentOf is the name of the bone that carries the bone named name.
func birdParentOf(t *testing.T, body *birdBody, name string) string {
	t.Helper()
	at, ok := body.bones.Find(name)
	if !ok {
		t.Fatalf("no bone named %s", name)
	}
	return body.bones.Bone(body.bones.Bone(at).Parent()).Name()
}

// birdBonesWith is how many bones have a name that is prefix followed by
// a digit, or, when plain, that begin with prefix.
func birdBonesWith(body *birdBody, prefix string, plain bool) int {
	count := 0
	for i := 0; i < body.bones.Len(); i++ {
		name := body.bones.Bone(i).Name()
		if plain && strings.HasPrefix(name, prefix) ||
			!plain && birdNumbered(name, prefix) {
			count++
		}
	}
	return count
}

// birdCorners are the corners of piece, carried by frames.
func birdCorners(body *birdBody, piece birdPiece,
	frames []skeleton.Frame) []maths.Vec {
	at, _ := body.bones.Find(piece.bone)
	corners, _ := piece.form.Build([]maths.Vec{}, [][3]int{},
		frames[at].Apply)
	return corners
}

// birdLowestOf is the lowest height of any corner of pieces carried by
// frames, and the name of the bone of the piece that has it.
func birdLowestOf(body *birdBody, pieces []birdPiece,
	frames []skeleton.Frame) (float64, string) {
	low, name := math.Inf(1), ""
	for _, piece := range pieces {
		for _, corner := range birdCorners(body, piece, frames) {
			if corner.Y < low {
				low, name = corner.Y, piece.bone
			}
		}
	}
	return low, name
}

// birdSolid is whether corners reach further than birdTiniest along each
// axis, so that the part they make is not a line or a speck.
func birdSolid(corners []maths.Vec) bool {
	box := birdGrown(maths.Box{Min: maths.V(math.Inf(1), math.Inf(1),
		math.Inf(1)), Max: maths.V(math.Inf(-1), math.Inf(-1),
		math.Inf(-1))}, corners)
	size := box.Max.Sub(box.Min)
	return size.X > birdTiniest && size.Y > birdTiniest &&
		size.Z > birdTiniest
}

// TestAFlatThingIsNotASolidBirdPart is the anti-test of birdSolid: two
// points and a flat square are not parts, a small cube is.
func TestAFlatThingIsNotASolidBirdPart(t *testing.T) {
	line := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0)}
	square := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 1, 0)}
	cube := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 1, 1)}
	if birdSolid(line) || birdSolid(square) || !birdSolid(cube) {
		t.Error("birdSolid does not tell a part from a line and a plane")
	}
}

// TestABirdsPartsAreTheSumOfWhatItIsMadeOf works the count of a bird's
// parts by hand from its row, and checks it against the maker's. Each side
// has three feathers to a primary (it, its greater and its median
// covert), four to a secondary or a tertial (those and the lesser
// covert), the alula's three and the six scapulars; the tail has three to
// a rectrix, the nape the row's plumes. The bare parts are the face, two
// eyes with their rings and two bill sheaths, seven; on each leg the
// femur, shank and tarsus and every toe bone; twelve scutes; eight claws;
// two webs for each pair of toes of a webbed bird; and the three small
// coverts on each wing's bones. One more is the outline.
func TestABirdsPartsAreTheSumOfWhatItIsMadeOf(t *testing.T) {
	const head, legBones, wingCoverts = 7, 3, 6
	toeBones := 0
	for _, toe := range birdToes {
		toeBones += toe.phalanges
	}
	birdEachBody(t, func(key string, body *birdBody) {
		m := body.measures
		side := 3*m.primaries + 4*(m.secondaries+birdTertials) +
			birdAlulaFeathers + birdScapulars
		feathers := 2*side + 3*m.rectrices + m.plumes
		webs := 0
		if m.web > 0 {
			webs = 2 * len(birdWebPairs)
		}
		bare := head + 2*(legBones+toeBones) + 2*birdScutes +
			2*len(birdToes) + webs + wingCoverts
		want, got := feathers+bare+1, len(body.skinPieces())+1
		t.Logf("%s: %d parts", key, got)
		if got != want {
			t.Errorf("%s: %d parts, worked %d", key, got, want)
		}
	})
}

// birdWorkedUp is a species' body with the figure named figure set to
// count, grown from the test seed.
func birdWorkedUp(t *testing.T, key, figure string, count float64) *birdBody {
	t.Helper()
	species := birdRead.Species[key]
	figures := map[string]birdFigure{}
	for name, each := range species.Figures {
		figures[name] = each
	}
	figures[figure] = birdFigure{Least: count, Most: count,
		Basis: birdPickedBasis}
	species.Figures = figures
	measures, err := measured(species, 0)
	if err != nil {
		t.Fatal(err)
	}
	body, err := newBirdBody(species, measures, sow(birdTestSeed))
	if err != nil {
		t.Fatal(err)
	}
	return body
}
