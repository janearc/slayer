package shape

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The tables the tests of this file read: points and rays, the radii a
// sphere is given, and the tubes that are no tube.
const (
	sphereRefusalsVectorFile = "testdata/sphere-refusals-vectors.csv"
	sphereRefusalsRadiusFile = "testdata/sphere-refusals-radii.csv"
	sphereRefusalsTubeFile   = "testdata/sphere-refusals-tubes.csv"
)

// sphereRefusalsTube is the tube with a wall and a length the tests of this
// file measure, from the rows of numbers.yaml that say its radius, wall
// and length.
func sphereRefusalsTube(t *testing.T) Tube {
	t.Helper()
	return Tube{Radius: picked(t, "sphere-refusals-tube-radius"),
		Thick:  picked(t, "sphere-refusals-tube-wall"),
		Height: picked(t, "sphere-refusals-tube-length")}
}

// TestASphereMeasuresALongDirInLengthsOfIt says that a ray whose Dir is two
// long meets a sphere at half the distance a unit one does, and a span is
// halved the same. Worked by hand: radius 2, from 10 out, a unit Dir meets
// at 8 and leaves at 12; a Dir of length 2 meets at 4 and leaves at 6.
func TestASphereMeasuresALongDirInLengthsOfIt(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	ball := Sphere{Radius: picked(t, "sphere-refusals-radius")}
	long := vectors.ray("long-dir")
	tolerance := picked(t, "sphere-refusals-tolerance")
	wantHit := picked(t, "sphere-refusals-long-dir-hit")
	wantLeave := picked(t, "sphere-refusals-long-dir-leave")
	got, ok := ball.Hit(long, nought, math.Inf(unity))
	if !ok || math.Abs(got-wantHit) > tolerance {
		t.Errorf("a Dir two long hit at %v, %v; want %v", got, ok, wantHit)
	}
	enter, leave, ok := ball.Span(long, nought, math.Inf(unity))
	if !ok || math.Abs(enter-wantHit) > tolerance ||
		math.Abs(leave-wantLeave) > tolerance {
		t.Errorf("a Dir two long spans %v to %v, %v; want %v to %v", enter,
			leave, ok, wantHit, wantLeave)
	}
}

// TestAUnitDirAndALongDirDoNotAgree is the anti-test of the one above: the
// unit ray's answer is 8, not 4, so a Hit that ignored the length of Dir
// would pass one of the two and fail the other.
func TestAUnitDirAndALongDirDoNotAgree(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	ball := Sphere{Radius: picked(t, "sphere-refusals-radius")}
	want := picked(t, "sphere-refusals-unit-dir-hit")
	got, ok := ball.Hit(vectors.ray("unit-dir"), nought, math.Inf(unity))
	if !ok || math.Abs(got-want) > picked(t, "sphere-refusals-tolerance") {
		t.Errorf("a unit Dir hit at %v, %v; want %v", got, ok, want)
	}
}

// sphereRefusalsFarRay is the ray, straight down, from the range of the far
// small sphere and offset off the line of the sphere's centre.
func sphereRefusalsFarRay(t *testing.T, offset float64) maths.Ray {
	t.Helper()
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	rangeOut := picked(t, "sphere-refusals-far-small-range")
	return maths.Ray{At: maths.V(offset, nought, rangeOut),
		Dir: vectors.point("straight-down")}
}

// TestAFarSmallSphereKeepsItsWidth says that a ball of radius one, a hundred
// million away and half a radius off the line of the ray, is met a
// radius-and-a-bit short of its centre: at the distance to the centre less
// the root of three quarters, not at the centre's distance, as when the
// square of the distance swallowed the radius.
func TestAFarSmallSphereKeepsItsWidth(t *testing.T) {
	ball := Sphere{Radius: unity}
	offset := picked(t, "sphere-refusals-far-small-offset")
	rangeOut := picked(t, "sphere-refusals-far-small-range")
	ray := sphereRefusalsFarRay(t, offset)
	want := rangeOut - math.Sqrt(unity-offset*offset)
	tolerance := picked(t, "sphere-refusals-far-small-tolerance")
	if got, ok := ball.Hit(ray, nought, math.Inf(unity)); !ok ||
		math.Abs(got-want) > tolerance {
		t.Errorf("hit at %v, %v; want %v", got, ok, want)
	}
}

// TestAFarSmallSphereIsMissedByARayBesideIt is the anti-test of the one
// above: a ray a radius and a half off the line meets nothing, so the
// answer above is the sphere's width and not an answer every ray gets.
func TestAFarSmallSphereIsMissedByARayBesideIt(t *testing.T) {
	ball := Sphere{Radius: unity}
	ray := sphereRefusalsFarRay(t, picked(t, "sphere-refusals-beside-offset"))
	if got, ok := ball.Hit(ray, nought, math.Inf(unity)); ok {
		t.Errorf("a ray beside the sphere hit it at %v", got)
	}
}

// TestARayWithNoDirectionMeetsNothing says that a ray that goes nowhere
// hits no sphere, and spans none.
func TestARayWithNoDirectionMeetsNothing(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	ball := Sphere{Radius: unity}
	still := vectors.ray("still")
	if _, ok := ball.Hit(still, nought, math.Inf(unity)); ok {
		t.Error("a ray with no direction hit a sphere")
	}
	if _, _, ok := ball.Span(still, nought, math.Inf(unity)); ok {
		t.Error("a ray with no direction spanned a sphere")
	}
}

// TestASphereWithNoRadiusIsNowhere says that a radius that is zero, below
// zero, not a number or infinite makes no ball, and that its Bounds,
// Distance, Hit and Span all say so.
func TestASphereWithNoRadiusIsNowhere(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	through := vectors.ray("through")
	asked := vectors.point("asked")
	for _, each := range readSheet(t, sphereRefusalsRadiusFile) {
		radius := cell(t, each, "radius")
		ball := Sphere{Radius: radius}
		if ball.Bounds() != nowhere() {
			t.Errorf("radius %v: bounds %v, want the one point", radius,
				ball.Bounds())
		}
		if got := ball.Distance(asked); !math.IsInf(got, unity) {
			t.Errorf("radius %v: distance %v, want infinite", radius, got)
		}
		if got, ok := ball.Hit(through, nought, math.Inf(unity)); ok {
			t.Errorf("radius %v: hit at %v", radius, got)
		}
		if _, _, ok := ball.Span(through, nought, math.Inf(unity)); ok {
			t.Errorf("radius %v: spanned", radius)
		}
	}
}

// TestASphereWithARadiusIsSomewhere is the anti-test of the one above: a
// radius of one has bounds that are not the one point, a distance that is
// minus one at its centre, and a hit.
func TestASphereWithARadiusIsSomewhere(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	ball := Sphere{Radius: unity}
	through := vectors.ray("through")
	if ball.Bounds() == nowhere() {
		t.Error("a sphere of radius one is the one point")
	}
	centre := maths.V(nought, nought, nought)
	if got := ball.Distance(centre); got != -unity {
		t.Errorf("distance at the centre %v, want %v", got, -unity)
	}
	if _, ok := ball.Hit(through, nought, math.Inf(unity)); !ok {
		t.Error("a ray at a sphere of radius one missed")
	}
}

// brokenTubes are tubes that are no tube, by the reason, read from the
// rows of a table.
func brokenTubes(t *testing.T) map[string]Tube {
	t.Helper()
	tubes := map[string]Tube{}
	for _, each := range readSheet(t, sphereRefusalsTubeFile) {
		tubes[each["name"]] = Tube{Radius: cell(t, each, "radius"),
			Thick:  cell(t, each, "wall"),
			Height: cell(t, each, "length")}
	}
	return tubes
}

// TestATubeThatIsNoTubeIsNowhere says that a tube with a measure below zero
// or not a finite number has bounds that are the one point and a distance
// that is infinite, and that Mesh refuses it in words.
func TestATubeThatIsNoTubeIsNowhere(t *testing.T) {
	asked := shapeVectorsIn(t, sphereRefusalsVectorFile).point("asked")
	for why, tube := range brokenTubes(t) {
		if tube.Bounds() != nowhere() {
			t.Errorf("%s: bounds %v, want the one point", why, tube.Bounds())
		}
		if got := tube.Distance(asked); !math.IsInf(got, unity) {
			t.Errorf("%s: distance %v, want infinite", why, got)
		}
		if _, err := tube.Mesh(TubeChordShare()); err == nil ||
			!strings.Contains(err.Error(), "needs a wall and a length") {
			t.Errorf("%s: mesh error %v, want one about the wall", why, err)
		}
	}
}

// TestAWhiskerOfNoLengthIsStillMeasured says that a tube of no length, as a
// whisker or a strand is built, has the distance and the bounds the formula
// gives and is not nowhere, though Mesh refuses it. Worked by hand: radius
// .001, wall .002, from .003 out along x: .002 from the ring's middle, .001
// from the outside of the wall, and level with the ring's edge.
func TestAWhiskerOfNoLengthIsStillMeasured(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	whisker := Tube{Radius: picked(t, "sphere-refusals-whisker-radius"),
		Thick: picked(t, "sphere-refusals-whisker-wall"), Height: nought}
	got := whisker.Distance(vectors.point("whisker-query"))
	want := picked(t, "sphere-refusals-whisker-distance")
	if math.Abs(got-want) > picked(t, "sphere-refusals-whisker-tolerance") {
		t.Errorf("distance %v, want a thousandth", got)
	}
	if whisker.Bounds() == nowhere() || whisker.Bounds().Max.Y != nought {
		t.Errorf("bounds %v, want the flat box of the ring", whisker.Bounds())
	}
	if _, err := whisker.Mesh(TubeChordShare()); err == nil {
		t.Error("a tube of no length was meshed")
	}
}

// TestAPlainRayKeepsItsPlainArithmetic says that a ray one long, from
// not far, meets a sphere at exactly the roots of the plain quadratic: minus
// b less and plus the root of b squared less c, computed here as they have
// always been, so that no existing ray moves in its last bit.
func TestAPlainRayKeepsItsPlainArithmetic(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	ball := Sphere{Radius: picked(t, "sphere-refusals-plain-radius")}
	diagonal := vectors.ray("plain-diagonal")
	diagonal.Dir = diagonal.Dir.Unit()
	for _, ray := range []maths.Ray{vectors.ray("plain-near"), diagonal} {
		along := ray.At.Dot(ray.Dir)
		rest := ray.At.Dot(ray.At) - ball.Radius*ball.Radius
		root := math.Sqrt(along*along - rest)
		first, last, ok := ball.crossings(ray)
		if !ok || first != -along-root || last != -along+root {
			t.Errorf("%+v: %v %v %v, want %v %v", ray, first, last, ok,
				-along-root, -along+root)
		}
	}
}

// TestALongRayAndAFarBallAreToldFromAPlainOne is the anti-test of the one
// above: a ray two long, and a ball a hundred million radii away, are not
// solved in the plain form, and a ray one long from nearby is.
func TestALongRayAndAFarBallAreToldFromAPlainOne(t *testing.T) {
	vectors := shapeVectorsIn(t, sphereRefusalsVectorFile)
	ball := Sphere{Radius: unity}
	type told struct {
		ray   maths.Ray
		plain bool
	}
	rows := map[string]told{"near": {vectors.ray("near"), true},
		"long": {vectors.ray("long"), false},
		"far":  {vectors.ray("far"), false}}
	for why, row := range rows {
		lengthSquared := row.ray.Dir.Dot(row.ray.Dir)
		startSquared := row.ray.At.Dot(row.ray.At)
		plain := math.Abs(lengthSquared-unity) <= unitSlack &&
			startSquared <= cancellationLimit*ball.Radius*ball.Radius
		if plain != row.plain {
			t.Errorf("%s: plain %v, want %v", why, plain, row.plain)
		}
	}
}

// TestATubeWithAWallIsSomewhere is the anti-test of the one above: a tube
// with a wall and a length has bounds that are not the one point and a
// finite distance.
func TestATubeWithAWallIsSomewhere(t *testing.T) {
	asked := shapeVectorsIn(t, sphereRefusalsVectorFile).point("asked")
	tube := sphereRefusalsTube(t)
	if tube.Bounds() == nowhere() {
		t.Error("a tube with a wall is the one point")
	}
	if got := tube.Distance(asked); math.IsInf(got, nought) || math.IsNaN(got) {
		t.Errorf("distance %v, want a number", got)
	}
}
