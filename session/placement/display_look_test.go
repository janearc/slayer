package placement

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// lookScene is a display scene with its eye at the origin looking along
// minus z through a square picture with a ninety degree field, so a point at
// distance d and offset o lands o/d from the centre in half pictures.
func lookScene() *DisplayScene {
	return &DisplayScene{Eye: maths.Local(), Lens: film.Pinhole{Wide: 90 * maths.Degree, Pixel: 1}, Width: 100, Height: 100}
}

// near is how closely two measurements of the same thing must agree.
const near = 1e-9

func TestPickAtAgreesWithPickOverTheWholePicture(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	hits := 0
	for column := 0; column < 20; column++ {
		for row := 0; row < 20; row++ {
			x, y := (float64(column)+.5)/20, (float64(row)+.5)/20
			hit, found := scene.PickAt(x, y)
			want := scene.Pick(x, y)
			if hit.ID == "scene.world" {
				want = ""
			}
			if want != hit.ID && !(want == "" && !found) {
				t.Fatalf("at %v,%v Pick said %q and PickAt %q", x, y, want, hit.ID)
			}
			if found {
				hits++
			}
		}
	}
	if hits == 0 {
		t.Fatal("no ray in a twenty by twenty grid hit anything")
	}
}

func TestPickAtGivesAPointAtItsDistanceFromTheEye(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	for column := 0; column < 20; column++ {
		hit, found := scene.PickAt((float64(column)+.5)/20, .5)
		if !found {
			continue
		}
		if math.Abs(hit.Point.Sub(scene.Eye.Origin).Len()-hit.Distance) > 1e-6 || hit.Distance <= 0 {
			t.Fatalf("%s: the point is %v from the eye, the hit says %v", hit.Name, hit.Point.Sub(scene.Eye.Origin).Len(), hit.Distance)
		}
		if !strings.HasSuffix(hit.ID, "/"+hit.Name) && hit.ID != "scene.world" {
			t.Fatalf("the id %q does not end in the name %q", hit.ID, hit.Name)
		}
	}
}

func TestPickAtRefusesAPointOffThePicture(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range [][2]float64{{-.1, .5}, {.5, 1.1}, {math.NaN(), .5}, {.5, math.Inf(1)}} {
		if hit, found := scene.PickAt(at[0], at[1]); found {
			t.Fatalf("%v is off the picture and hit %q", at, hit.ID)
		}
	}
	var none *DisplayScene
	if _, found := none.PickAt(.5, .5); found {
		t.Fatal("a missing scene was hit")
	}
}

func TestPartAtNamesTheMaterialOfThePartHit(t *testing.T) {
	mesh := &DisplayMesh{hitter: displayComponents{
		{hitter: shape.Sphere{Radius: 1}, at: maths.Local(), material: "glass"},
	}}
	ray := maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}
	if got := mesh.partAt(ray); got != "glass" {
		t.Fatalf("the part hit was %q", got)
	}
	away := maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, 1)}
	if got := mesh.partAt(away); got != "" {
		t.Fatalf("a ray that misses named the part %q", got)
	}
	if got := (&DisplayMesh{}).partAt(ray); got != "" {
		t.Fatalf("a mesh with no hitter named the part %q", got)
	}
}

func TestSetPointIsEastNorthUp(t *testing.T) {
	if got := SetPoint(maths.V(1, 2, -3)); got != [3]float64{1, 3, 2} {
		t.Fatalf("got %v", got)
	}
	if got := SetPoint(maths.V(0, 0, 0)); got != [3]float64{0, 0, 0} {
		t.Fatalf("the origin moved to %v", got)
	}
}

func TestBoundsHoldEveryVertexAndAnEmptyMeshHasNone(t *testing.T) {
	mesh := &DisplayMesh{Vertices: []DisplayVertex{
		{Position: maths.V(1, 5, -2)}, {Position: maths.V(-3, 6, 4)}, {Position: maths.V(0, 4, 0)},
	}}
	low, high, ok := mesh.Bounds()
	if !ok || low != maths.V(-3, 4, -2) || high != maths.V(1, 6, 4) {
		t.Fatalf("bounds were %v %v %v", low, high, ok)
	}
	if _, _, ok := (&DisplayMesh{}).Bounds(); ok {
		t.Fatal("an empty mesh had bounds")
	}
	var missing *DisplayMesh
	if _, _, ok := missing.Bounds(); ok {
		t.Fatal("a missing mesh had bounds")
	}
}

func TestProjectIsTheInverseOfTheEyeRay(t *testing.T) {
	scene := lookScene()
	for _, at := range [][2]int{{10, 10}, {50, 50}, {90, 25}, {3, 97}} {
		x, y := (float64(at[0])+.5)/100, (float64(at[1])+.5)/100
		ray, err := scene.displayRay(x, y)
		if err != nil {
			t.Fatal(err)
		}
		along := ray.Point(37)
		gotX, gotY := scene.project(scene.Eye.In(along))
		if math.Abs(gotX-x) > near || math.Abs(gotY-y) > near {
			t.Fatalf("pixel %v came back as %v,%v, not %v,%v", at, gotX, gotY, x, y)
		}
	}
}

func TestProjectKeepsTheShapeOfAWidePicture(t *testing.T) {
	scene := lookScene()
	scene.Width = 200
	x, y := scene.project(maths.V(1, 1, -10))
	// one across and one up, ten ahead, is a tenth of the half width each
	// way through a lens ninety degrees wide. the picture is 200 by 100, so
	// that is ten pixels across and ten up: the same count, and the shape
	// is kept.
	across := (x - .5) * float64(scene.Width)
	up := (.5 - y) * float64(scene.Height)
	if math.Abs(across-10) > near || math.Abs(up-10) > near {
		t.Fatalf("a point one across and one up came out %v pixels across and %v up, not ten and ten", across, up)
	}
}

// its anti-test: on a square picture the same point is five pixels across
// and five up, so the wide picture's ten across is the width's doing.
func TestProjectOnASquarePictureIsEvenBothWays(t *testing.T) {
	scene := lookScene()
	x, y := scene.project(maths.V(1, 1, -10))
	across := (x - .5) * float64(scene.Width)
	up := (.5 - y) * float64(scene.Height)
	if math.Abs(across-5) > near || math.Abs(up-5) > near {
		t.Fatalf("a point one across and one up came out %v pixels across and %v up, not five and five", across, up)
	}
}

func TestBoxShareOfABoxAheadIsItsOutlineOnThePicture(t *testing.T) {
	scene := lookScene()
	share := scene.boxShare(maths.Local(), maths.V(-1, -1, -12), maths.V(1, 1, -10))
	if math.Abs(share-.01) > near {
		t.Fatalf("share was %v, not .01", share)
	}
}

func TestBoxShareIsZeroBehindTheCameraAndBesideThePicture(t *testing.T) {
	scene := lookScene()
	if share := scene.boxShare(maths.Local(), maths.V(-1, -1, 10), maths.V(1, 1, 12)); share != 0 {
		t.Fatalf("a box behind the camera had share %v", share)
	}
	if share := scene.boxShare(maths.Local(), maths.V(100, -1, -12), maths.V(102, 1, -10)); share != 0 {
		t.Fatalf("a box far off to one side had share %v", share)
	}
}

func TestBoxShareOfABoxTheCameraStandsInIsWholeAndNeverMore(t *testing.T) {
	scene := lookScene()
	share := scene.boxShare(maths.Local(), maths.V(-5, -5, -5), maths.V(5, 5, 5))
	if share <= 0 || share > 1 {
		t.Fatalf("a box around the camera had share %v", share)
	}
}

func TestBoxDistanceIsToTheNearestPointOfTheBox(t *testing.T) {
	scene := lookScene()
	if got := scene.boxDistance(maths.Local(), maths.V(-1, -1, -12), maths.V(1, 1, -10)); math.Abs(got-10) > near {
		t.Fatalf("distance was %v, not 10", got)
	}
	if got := scene.boxDistance(maths.Local(), maths.V(-1, -1, -1), maths.V(1, 1, 1)); got != 0 {
		t.Fatalf("a box holding the eye was %v away", got)
	}
}

func TestClipNearCutsAtTheNearPlane(t *testing.T) {
	if _, _, in := clipNear(maths.V(0, 0, 1), maths.V(0, 0, 2)); in {
		t.Fatal("a segment behind the eye was kept")
	}
	first, second, in := clipNear(maths.V(0, 0, 4), maths.V(0, 0, -10))
	if !in || math.Abs(-first.Z-DisplayNear) > near || second.Z != -10 {
		t.Fatalf("the cut ends were %v and %v (%v)", first, second, in)
	}
	first, second, in = clipNear(maths.V(0, 0, -10), maths.V(0, 0, 4))
	if !in || first.Z != -10 || math.Abs(-second.Z-DisplayNear) > near {
		t.Fatalf("the cut ends were %v and %v (%v)", first, second, in)
	}
}

func TestInViewIsNearestFirstAndLeavesOutWhatIsNotOnThePicture(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	seen := scene.InView()
	if len(seen) == 0 {
		t.Fatal("the scene's own camera sees nothing")
	}
	for index, each := range seen {
		if each.Share <= 0 || each.Share > 1 || each.Distance < 0 {
			t.Fatalf("%s: share %v distance %v", each.Name, each.Share, each.Distance)
		}
		if index > 0 && seen[index-1].Distance > each.Distance {
			t.Fatalf("%s is nearer than %s but listed after it", each.Name, seen[index-1].Name)
		}
	}
	var none *DisplayScene
	if none.InView() != nil {
		t.Fatal("a missing scene saw something")
	}
}

func TestInViewSeesTheFocusOfTheCameraAtTheCentre(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	x, y := scene.project(scene.Eye.In(scene.Focus))
	if math.Abs(x-.5) > 1e-6 || math.Abs(y-.5) > 1e-6 {
		t.Fatalf("the camera's focus is at %v,%v on its own picture", x, y)
	}
}
