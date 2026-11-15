package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The eye looks down at the floor's middle at 45 degrees, and mirrored
// is the way a lamp must lie for the floor to mirror it to the eye: the
// way half between the two is then the floor's own normal.
var (
	downAtFloor = maths.Ray{At: maths.V(-1, 1, 0), Dir: maths.V(1, -1, 0).Unit()}
	mirrored    = maths.V(1, 1, 0).Unit()
)

// lamplit is how bright a lamp of three shines in these tests.
const lamplit = 3.0

// lacquer is a black gloss finish as rough as satin says: black, so the
// lamps leave nothing on it but their highlights, and under no sky, so
// it mirrors nothing.
func lacquer(satin float64) *thing.Material {
	return &thing.Material{Name: "lacquer", Gloss: 1, Index: 1.5, Satin: satin}
}

// floored is a floor of a look under one sun lying a way.
func floored(look *thing.Material, toward maths.Vec) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: look})
	world.Lamp(thing.Sun(toward, maths.Grey(lamplit)))
	return world
}

// A highlight is the lamp's light, times what the surface mirrors at
// the way half between the lamp and the eye, times the lobe of a
// surface as rough as its Satin. With the lamp where the floor mirrors
// it to an eye at 45 degrees, the half way is the normal, so the lobe
// is asked at a facing of one and cosines of root a half to the lamp
// and to the eye. A Satin above one is one. The anti-test is a floor
// with no Satin, which has no highlight at all.
func TestAHighlightIsTheLampTimesWhatIsMirroredTimesTheLobe(t *testing.T) {
	slant := math.Sqrt(0.5)
	for _, satin := range []float64{0.2, 0.5, 1, 3} {
		look := lacquer(satin)
		rough := math.Min(satin, 1)
		want := lamplit * look.Reflects(slant).R *
			thing.Microfacet(1, slant, slant, rough*rough)
		got := lone(floored(look, mirrored), downAtFloor, 0).Light()
		// a pixel holds its light in 32 bits: seven figures
		near := want * 1e-6
		if !(want > 0) || !within(got.R, want, near) ||
			!within(got.G, want, near) || !within(got.B, want, near) {
			t.Errorf("at a Satin of %v the highlight is %v, want %v in "+
				"each colour", satin, got, want)
		}
	}
	if got := lone(floored(lacquer(0), mirrored), downAtFloor, 0).Light(); got != (maths.Light{}) {
		t.Errorf("a floor with no Satin shows %v", got)
	}
}

// A highlight is brightest where the surface mirrors the lamp to the
// eye and falls away from there. The anti-tests: a lamp under the floor
// leaves none, and neither does one a ball stands in front of.
func TestAHighlightFallsAwayFromTheMirroredWayAndNeedsItsLamp(t *testing.T) {
	look := lacquer(0.2)
	at := bright(lone(floored(look, mirrored), downAtFloor, 0))
	beside := bright(lone(floored(look, maths.V(1, 0.5, 0)), downAtFloor, 0))
	if !(at > beside) || !(beside >= 0) {
		t.Errorf("the highlight is %v where the lamp is mirrored and %v "+
			"beside it", at, beside)
	}
	if under := bright(lone(floored(look, maths.V(1, -1, 0)), downAtFloor, 0)); under != 0 {
		t.Errorf("a lamp under the floor leaves a highlight of %v", under)
	}
	hidden := floored(look, mirrored)
	hidden.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5},
		Material: &thing.Material{}, Motion: resting(mirrored.Scale(2))})
	if got := bright(lone(hidden, downAtFloor, 0)); got != 0 {
		t.Errorf("a lamp behind a ball leaves a highlight of %v", got)
	}
}

// A highlight is tinted by what the surface mirrors and by nothing
// else: gold's has most red and least blue, as gold mirrors. The
// anti-test is lacquer's, which is the lamp's own white and no colour.
func TestAHighlightOnGoldIsGoldAndOnLacquerWhite(t *testing.T) {
	gilt := &thing.Material{Name: "gilt", Gloss: 1, Metal: thing.Gold(),
		Satin: 0.3}
	got := lone(floored(gilt, mirrored), downAtFloor, 0).Light()
	if !(got.R > got.G && got.G > got.B && got.B > 0) {
		t.Errorf("gold's highlight is %v: not gold", got)
	}
	white := lone(floored(lacquer(0.3), mirrored), downAtFloor, 0).Light()
	if white.R != white.G || white.G != white.B || !(white.R > 0) {
		t.Errorf("lacquer's highlight is %v: not white", white)
	}
}

// A surface seen from in front has its highlight. The anti-test: seen
// from behind it has none, for the lamps' light on its far side is not
// the eye's to see.
func TestASurfaceSeenFromBehindHasNoHighlight(t *testing.T) {
	scene := stage(floored(lacquer(0.3), mirrored), 0)
	front := scene.glint(maths.Vec{}, maths.YAxis, downAtFloor.Dir.Neg(),
		lacquer(0.3))
	behind := scene.glint(maths.Vec{}, maths.YAxis, maths.YAxis.Neg(),
		lacquer(0.3))
	if !(front.Luminance() > 0) || behind != (maths.Light{}) {
		t.Errorf("the highlight is %v from in front and %v from behind",
			front, behind)
	}
}

// shaded is a white floor with a ball half a metre in radius two
// metres over its middle, under a sun straight overhead whose light on
// a floor facing it is one.
func shaded(sun thing.Lamp) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5},
		Material: &thing.Material{}, Motion: resting(maths.V(0, 2, 0))})
	world.Lamp(sun)
	return world
}

// floorAt is how bright the floor is at a place along x, seen from
// straight overhead and off to one side, so the ball is not in the way.
func floorAt(world *thing.World, x float64) float64 {
	from := maths.V(x+3, 3, 0)
	return bright(lone(world, maths.Ray{At: from,
		Dir: maths.V(x, 0, 0).Sub(from).Unit()}, 0))
}

// A sun drawn as a disc softens its shadows by its width. The anti-test
// is a point sun, hidden whole under the ball's middle, where the floor
// is black; a sun 0.8 radians wide is sought along the eight ways across its
// disc, and the floor there is lit by the share of them that pass the
// ball, which this works for itself from where the ball is: some, and
// not all. Where nothing hides it the wide sun lights the floor as the
// point one does.
func TestAWideSunSoftensTheShadowItCasts(t *testing.T) {
	overhead := maths.V(0, 1, 0)
	point := film.Sun{Toward: overhead, Strength: maths.Grey(math.Pi)}
	wide := film.Sun{Toward: overhead, Strength: maths.Grey(math.Pi),
		Size: 0.8, Disc: true}
	ball, radius := maths.V(0, 2, 0), 0.5
	var open int
	ways := wide.Spread()
	for _, way := range ways {
		if ball.Sub(way.Scale(ball.Dot(way))).Len() > radius {
			open++
		}
	}
	if open == 0 || open == len(ways) {
		t.Fatalf("%d of %d ways pass the ball: the test holds nothing",
			open, len(ways))
	}
	if got := floorAt(shaded(point), 0); got != 0 {
		t.Errorf("under the ball a point sun lights the floor by %v", got)
	}
	want := float64(open) / float64(len(ways))
	if got := floorAt(shaded(wide), 0); !within(got, want, 1e-9) {
		t.Errorf("under the ball the wide sun lights the floor by %v, "+
			"want %v: %d of its %d ways pass", got, want, open, len(ways))
	}
	clear, soft := floorAt(shaded(point), 5), floorAt(shaded(wide), 5)
	if !within(clear, 1, 1e-9) || !within(soft, clear, 1e-9) {
		t.Errorf("clear of the ball the point sun gives %v and the wide "+
			"one %v, want one from each", clear, soft)
	}
}
