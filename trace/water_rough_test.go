package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The small faces' slopes about the surface's own have the variance
// asked, taken over every place the faces can fall.
func TestTiltedFacesHaveTheSlopeVarianceAsked(t *testing.T) {
	const variance, steps = 0.02, 64
	normal := maths.V(0, 1, 0)
	var sum float64
	for index := range roughLooks {
		for first := range steps {
			for second := range steps {
				face := tilt(normal, variance, index,
					(float64(first)+0.5)/steps, (float64(second)+0.5)/steps)
				sum += (1 - face.Y*face.Y) / (face.Y * face.Y)
			}
		}
	}
	mean := sum / (roughLooks * steps * steps)
	if math.Abs(mean-variance) > 0.02*variance {
		t.Errorf("mean square slope %g, want %g", mean, variance)
	}
}

// Anti: with no variance every face is the surface's own normal.
func TestTiltedFacesOfNoVarianceAreTheNormal(t *testing.T) {
	normal := maths.V(0.6, 0.8, 0)
	for index := range roughLooks {
		if face := tilt(normal, 0, index, 0.3, 0.7); !face.Near(normal,
			1e-12) {
			t.Errorf("face %d is %v", index, face)
		}
	}
}

// The faces' shares are of one surface: given over the area all of them
// show, they sum to fresnel's share and to no more than all the light.
// Anti: faces that show the eye nothing are no surface at all.
func TestFacetSharesAreOfOneSurface(t *testing.T) {
	var faces [roughLooks]facet
	for index := range faces {
		faces[index].share = 0.5 * float64(index+1)
	}
	divided, any := shared(faces, 0.5*(1+2+3+4))
	if !any || math.Abs(facetShare(divided)-1) > 1e-12 {
		t.Errorf("shares %v sum to %g", divided, facetShare(divided))
	}
	if _, any := shared(faces, 0); any {
		t.Error("faces that show nothing were taken as a surface")
	}
}

// The numbers that place a look's faces are the same for the same look,
// unlike for a look a hair away, and from nought up to one.
func TestDitheredIsTheSameForTheSameLookAndUnlikeForAnother(t *testing.T) {
	point, way := maths.V(10, 0.3, -7), maths.V(0.6, -0.8, 0)
	first, second := dithered(point, way)
	again, also := dithered(point, way)
	if first != again || second != also {
		t.Error("the same look fell in two places")
	}
	nudged := point.Add(maths.V(1e-9, 0, 0))
	other, _ := dithered(nudged, way)
	if other == first {
		t.Error("a look a hair away fell in the same place")
	}
	for _, number := range []float64{first, second, other} {
		if number < 0 || number >= 1 {
			t.Errorf("%g is not from nought up to one", number)
		}
	}
}

// Anti: a shape that is no sea mirrors by its one normal, whatever slope
// a pixel has lost.
func TestFacetsAreOnlyForASea(t *testing.T) {
	glass := &thing.Material{Through: 1, Gloss: 1}
	met := hit{thing: placedThing{thing: &thing.Thing{
		Shape: shape.Sphere{Radius: 1}, Material: glass}}}
	ray := maths.Ray{At: maths.V(0, 5, 0), Dir: maths.V(0, -1, 0)}
	at := boundary{relative: 1.333, real: true, entering: true}
	if _, rough := met.facets(glass, ray, maths.V(0, 1, 0),
		maths.V(0, 1, 0), 0.02, at); rough {
		t.Error("a ball of glass was mirrored as rough water")
	}
}

// chop is a flat sea that says the slope variance of the waves it does
// not hold: rough water, for facets.
type chop struct {
	shape.Plane
	unseen float64
}

// Unseen is the slope variance the sea does not hold.
func (sea chop) Unseen() float64 { return sea.unseen }

// choppy is a hit on a chop of water, and the water it is made of.
func choppy(unseen float64) (hit, *thing.Material) {
	water := &thing.Material{Through: 1, Gloss: 1, Index: waterIndex}
	return hit{thing: placedThing{thing: &thing.Thing{
		Shape: chop{unseen: unseen}, Material: water}}}, water
}

// intoWater is the boundary a look from the air meets water at.
var intoWater = boundary{relative: waterIndex, real: true, entering: true}

// facetWeights is, for each face, its share over Fresnel's share at its
// own tilt, which is how much of what the eye sees of the surface the
// face is; and that weight over the area a face of its tilt shows the
// eye, its cosine to the look over its cosine to the surface, which is
// the same number for every face of one surface.
func facetWeights(faces [roughLooks]facet, way,
	normal maths.Vec) (weights, perArea [roughLooks]float64) {
	for index, face := range faces {
		facing := -way.Dot(face.normal)
		weights[index] = face.share / mirrorShare(facing, waterIndex)
		perArea[index] = weights[index] * face.normal.Dot(normal) / facing
	}
	return weights, perArea
}

// oneSurface is whether weights add up to one and perArea is one number
// throughout: the faces are shares of one surface, each by the area it
// shows the eye.
func oneSurface(weights, perArea [roughLooks]float64) bool {
	var sum float64
	for index := range weights {
		sum += weights[index]
		if !within(perArea[index], perArea[0], 1e-9*perArea[0]) {
			return false
		}
	}
	return within(sum, 1, 1e-9)
}

// A look that enters rough water from above is mirrored by four small
// faces. Each is a unit normal on the surface's own side that faces the
// eye, and mirrors Fresnel's share at its own tilt of the part of the
// surface it is: the parts are by the area each shows the eye, and add
// up to the whole surface.
//
// The slope variance is the sea's own and the pixel's together, 0.02
// here; no face of it can tilt past a slope of 0.87, the square root of
// 0.02 times the 38 a float's finest step below one allows, and the eye
// at a cosine of 0.8 sees every face that tilts under 1.33.
func TestRoughWaterIsMirroredByFacesSharedByTheAreaTheyShowTheEye(
	t *testing.T) {
	met, water := choppy(0.01)
	ray := maths.Ray{At: maths.V(-3, 4, 0), Dir: maths.V(0.6, -0.8, 0)}
	faces, rough := met.facets(water, ray, maths.Vec{}, maths.Up, 0.01,
		intoWater)
	if !rough {
		t.Fatal("a chop of water was not taken for rough water")
	}
	for index, face := range faces {
		if !within(face.normal.Len(), 1, 1e-12) || !(face.normal.Y > 0) ||
			!(face.share > 0) || face.normal == maths.Up {
			t.Errorf("face %d is %+v", index, face)
		}
	}
	if weights, perArea := facetWeights(faces, ray.Dir,
		maths.Up); !oneSurface(weights, perArea) {
		t.Errorf("the faces are %v of the surface, %v by area", weights,
			perArea)
	}
}

// Anti: shares that are not of one surface are seen for what they are.
// The same faces with one share doubled add up to more than the
// surface, and with every share taken as equal are not by area.
func TestFacesThatAreNotSharesOfOneSurfaceAreTold(t *testing.T) {
	met, water := choppy(0.01)
	ray := maths.Ray{At: maths.V(-3, 4, 0), Dir: maths.V(0.6, -0.8, 0)}
	faces, _ := met.facets(water, ray, maths.Vec{}, maths.Up, 0.01,
		intoWater)
	doubled := faces
	doubled[0].share *= 2
	if oneSurface(facetWeights(doubled, ray.Dir, maths.Up)) {
		t.Error("a share counted twice passed for one surface")
	}
	level := faces
	for index := range level {
		level[index].share = mirrorShare(
			-ray.Dir.Dot(level[index].normal), waterIndex) / roughLooks
	}
	if oneSurface(facetWeights(level, ray.Dir, maths.Up)) {
		t.Error("faces weighed alike passed for faces weighed by area")
	}
}

// Seen from straight above, water only just rough mirrors Fresnel's
// share head on, ((n - 1) / (n + 1))^2: for water of 1.333,
// (0.333 / 2.333)^2 = 0.020373. A slope variance of two millionths
// tilts no face past a slope of 0.0088, half a degree, and that far from
// head on the share has moved by less than a part in ten thousand.
func TestWaterOnlyJustRoughSeenFromAboveMirrorsFresnelHeadOn(t *testing.T) {
	met, water := choppy(1e-6)
	ray := maths.Ray{At: maths.V(0, 5, 0), Dir: maths.V(0, -1, 0)}
	faces, rough := met.facets(water, ray, maths.Vec{}, maths.Up, 1e-6,
		intoWater)
	want := (waterIndex - 1) * (waterIndex - 1) /
		((waterIndex + 1) * (waterIndex + 1))
	if got := facetShare(faces); !rough || !within(got, want, 1e-4*want) {
		t.Errorf("rough %v, and the faces mirror %v, want %v", rough, got,
			want)
	}
}

// Anti: facets are for a look entering water from above with some slope
// unseen. Water as smooth as a mirror, a look leaving the water, a
// surface that is no boundary, a metal and a surface nothing passes
// each mirror by their one normal.
func TestFacetsRefuseSmoothWaterAndLooksThatDoNotEnterIt(t *testing.T) {
	ray := maths.Ray{At: maths.V(-3, 4, 0), Dir: maths.V(0.6, -0.8, 0)}
	rough := func(unseen, lost float64, at boundary,
		change func(*thing.Material)) bool {
		met, water := choppy(unseen)
		change(water)
		_, is := met.facets(water, ray, maths.Vec{}, maths.Up, lost, at)
		return is
	}
	plain := func(*thing.Material) {}
	leaving, unreal := intoWater, intoWater
	leaving.entering, unreal.real = false, false
	if !rough(0.01, 0.01, intoWater, plain) {
		t.Fatal("the test's own rough water is not rough")
	}
	for name, is := range map[string]bool{
		"smooth water":   rough(0, roughLeast, intoWater, plain),
		"a look leaving": rough(0.01, 0.01, leaving, plain),
		"no boundary":    rough(0.01, 0.01, unreal, plain),
		"a metal": rough(0.01, 0.01, intoWater, func(m *thing.Material) {
			m.Metal = &thing.Metal{}
		}),
		"an opaque surface": rough(0.01, 0.01, intoWater,
			func(m *thing.Material) { m.Through = 0 }),
	} {
		if is {
			t.Errorf("%s was mirrored as rough water", name)
		}
	}
}

// mirrorFaces are three small faces for a look down along 0.6, -0.8, 0.
// The first is level and mirrors a quarter: the look leaves it along
// 0.6, 0.8, 0. The second faces 0, 0.8, 0.6 and mirrors a half: the look
// meets it at a cosine of 0.64 and leaves along the look plus 1.28 of
// the normal, 0.6, 0.224, 0.768. The third mirrors nothing.
var mirrorFaces = [roughLooks]facet{
	{normal: maths.V(0, 1, 0), share: 0.25},
	{normal: maths.V(0, 0.8, 0.6), share: 0.5},
	{normal: maths.V(1, 0, 0)},
}

// mirrorLook is the look mirrorFaces are worked for.
var mirrorLook = maths.Ray{At: maths.V(-3, 4, 0), Dir: maths.V(0.6, -0.8, 0)}

// Under wayLit the first face's sky is 0.8, 0.9, 0.5 and the second's
// 0.8, 0.612, 0.884. A quarter of the one and a half of the other is
// 0.6, 0.531 and 0.567.
var mirroredSky = maths.RGB(0.6, 0.531, 0.567)

// The sky rough water mirrors when its look has no bounce left is each
// face's sky along its own mirrored way, by the face's share.
func TestRoughSkyIsEachFacesMirroredSkyByItsShare(t *testing.T) {
	got := skyOver(wayLit{}).roughSky(mirrorLook.Dir, mirrorFaces)
	lightIs(t, "the rough sky", got, mirroredSky, 1e-12)
}

// Anti: faces that mirror nothing mirror no sky, whichever way they
// face.
func TestRoughSkyOfFacesWithNoShareIsDark(t *testing.T) {
	faces := mirrorFaces
	for index := range faces {
		faces[index].share = 0
	}
	if got := skyOver(wayLit{}).roughSky(mirrorLook.Dir, faces); got !=
		(maths.Light{}) {
		t.Errorf("faces with no share mirror %v", got)
	}
}

// roughMirrored is what roughMirror brings back for mirrorFaces on a
// level chop at the world's origin, in a world under wayLit.
func roughMirrored(world *thing.World) maths.Light {
	world.Sky = wayLit{}
	met, _ := choppy(0.01)
	return stage(world, 0).roughMirror(look{ray: mirrorLook}, met,
		mirrorLook, maths.Vec{}, maths.Up, mirrorFaces)
}

// Rough water's mirror follows each face's own look through the scene
// and adds what it brings by the face's share. In a world of sky alone
// every look meets the sky, so the mirror is the rough sky's 0.6, 0.531
// and 0.567, as near as a pixel's single precision holds a look's light.
func TestRoughMirrorFollowsEachFacesLookByItsShare(t *testing.T) {
	rough := roughMirrored(thing.NewWorld())
	lightIs(t, "the rough mirror under a sky", rough, mirroredSky, 1e-6)
}

// Anti: the looks are followed and not taken for sky. A black ball five
// along the second face's way hides that face's sky, and the mirror is
// the first face's alone, a quarter of 0.8, 0.9 and 0.5.
func TestRoughMirrorSeesWhatStandsInAFacesWay(t *testing.T) {
	world := thing.NewWorld()
	sphereAt := resting(maths.V(0.6, 0.224, 0.768).Scale(5))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{},
		Motion:   sphereAt})
	lightIs(t, "the rough mirror past a ball", roughMirrored(world),
		maths.RGB(0.2, 0.225, 0.125), 1e-6)
}
