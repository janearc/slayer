package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// farAway is how far out a marched ball is placed, a hundred kilometres.
const farAway = 1e5

// ballAt is a staged world of one marched ball of radius one whose own
// middle is at its own origin, placed at the world's place.
func ballAt(place maths.Vec) *staged {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: farBall{radius: 1},
		Motion: resting(place)})
	return stage(world, 0)
}

// ballClearance is two touches of the ball's reach, the diagonal of its
// box, two root three, at a point one from its own origin: what a ray
// leaving its surface must start off it, worked by hand.
var ballClearance = 2 * touch * 2 * math.Sqrt(3)

// The marched part of the lift, clear, a marched ball needs is that of
// the ball, wherever it stands. A ball of radius one at its own origin,
// at the world's origin or a hundred kilometres out, is met within a
// touch of its reach, 3.5 millionths of a metre, and its rays start two
// touches off it, 6.9 millionths: the same. The distance of the world's
// origin from it is not what the march measures, and a bound worked
// from that is four tenths of a metre, for a ball met within a few
// millionths. This is clear alone: the lift is lifting's, below.
func TestAMarchedBallAtItsOwnOriginIsClearedByItsOwnTouchAnywhere(
	t *testing.T) {
	for _, place := range []maths.Vec{{}, maths.V(farAway, 0, 0)} {
		scene := ballAt(place)
		above := place.Add(maths.V(0, 0, 1))
		got := scene.marching.clear(above)
		if !within(got, ballClearance, 1e-9*ballClearance) ||
			!(got <= 7e-6) {
			t.Errorf("a ball at %v is cleared by %g, want %g, at most 7e-6",
				place, got, ballClearance)
		}
	}
}

// Anti: the same ball with its own middle a hundred kilometres from its
// own origin is met within a tenth of a metre, and is cleared by two
// of them: the distance that makes the lift large is the march's own.
func TestAMarchedBallFarFromItsOwnOriginIsClearedByTwoOfItsTouches(
	t *testing.T) {
	world := thing.NewWorld()
	origin := resting(maths.V(-farAway, 0, 0))
	world.Add(thing.Thing{Shape: farBall{middle: maths.V(farAway, 0, 0),
		radius: 1}, Motion: origin})
	scene := stage(world, 0)
	got := scene.marching.clear(maths.V(0, 0, 1))
	if want := 2 * touch * farAway; !(got >= want) || !(got <= 1.05*want) {
		t.Errorf("a ball far from its own origin is cleared by %g, want "+
			"%g to a twentieth over", got, want)
	}
}

// The lift a ray gets from a surface of a marched ball a hundred
// kilometres out, at its own origin, is lifting's, and not clear's: the
// larger of clear, 6.9 millionths of a metre, and the scene-wide share, a
// millionth of the larger of the spread and the point's distance from the
// world's origin. The spread is the world's box's reach, a hundred
// kilometres and a metre, so the share is a tenth of a metre, a hundred
// thousand times clear, and that is the lift of every surface there.
func TestTheLiftAtAMarchedBallAHundredKilometresOutIsTheSceneWideShare(
	t *testing.T) {
	scene := ballAt(maths.V(farAway, 0, 0))
	got := scene.lifting(maths.V(farAway, 0, 1))
	const want = 0.1
	if !within(got, want, 1e-4*want) || !(got > 1e4*ballClearance) {
		t.Errorf("the lift is %g, want %g, far over %g", got, want,
			ballClearance)
	}
}

// Anti: with the same ball at the world's origin the lift is clear's, the
// larger of the two there, and no tenth of a metre.
func TestTheLiftAtAMarchedBallAtTheWorldsOriginIsItsOwnClearance(
	t *testing.T) {
	got := ballAt(maths.Vec{}).lifting(maths.V(0, 0, 1))
	if !within(got, ballClearance, 1e-9*ballClearance) {
		t.Errorf("the lift is %g, want %g", got, ballClearance)
	}
}

// ballScaled is the staged world of a ball of radius one in its own
// space, whose middle is its origin, placed at the world's middle and
// scaled by scale.
func ballScaled(scale float64) *staged {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: farBall{radius: 1},
		Motion: thing.Still(maths.Subspace{Scale: scale})})
	return stage(world, 0)
}

// A thing scaled by ten is met within ten times the touch: its reach and
// its distance from its own origin are the world's lengths, ten times
// those of its own space, and so is the lift off it.
func TestAMarchedBallScaledByTenIsClearedByTenTimesTheTouch(t *testing.T) {
	const scale = 10
	got := ballScaled(scale).marching.clear(maths.V(0, 0, scale))
	if want := scale * ballClearance; !within(got, want, 1e-9*want) {
		t.Errorf("a ball scaled by ten is cleared by %g, want %g", got,
			want)
	}
}

// Anti: the same ball unscaled is cleared by one tenth of that, so the
// scale is what the ten came from.
func TestAMarchedBallUnscaledIsClearedByItsOwnTouch(t *testing.T) {
	got := ballScaled(1).marching.clear(maths.V(0, 0, 1))
	if !within(got, ballClearance, 1e-9*ballClearance) {
		t.Errorf("an unscaled ball is cleared by %g, want %g", got,
			ballClearance)
	}
}

// ballTurned is the staged world of a ball of radius one whose own
// middle is a thousand along x, its space turned a quarter about z and
// placed so that its middle is at the world's middle, and its origin
// where that puts it.
func ballTurned() *staged {
	turned := maths.Subspace{Orientation: maths.Rotation(
		maths.V(0, 0, 1), math.Pi/2)}
	turned.Origin = turned.Out(maths.V(1000, 0, 0)).Neg()
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: farBall{middle: maths.V(1000, 0, 0),
		radius: 1}, Motion: thing.Still(turned)})
	return stage(world, 0)
}

// A thing turned is measured as it is not: its box is in its own space,
// and the world's point is taken there. The ball is the one at the
// world's middle in the test of its own origin a thousand off, turned a
// quarter, and its lift at the top of it is that of the same ball not
// turned, since the distance from its origin is the same turned or not.
func TestAMarchedBallTurnedIsClearedAsTheSameBallNotTurned(t *testing.T) {
	point := maths.V(0, 0, 1)
	got := ballTurned().marching.clear(point)
	want := stagedBall(1000).marching.clear(point)
	if !(want > 1e-3) || got != want {
		t.Errorf("a turned ball is cleared by %g, and the same not "+
			"turned by %g, over a thousandth", got, want)
	}
}

// Anti: a point outside the turned ball's box, five off its middle, is
// cleared of it by nothing, as for the ball not turned.
func TestAMarchedBallTurnedClearsNothingOutsideItsBox(t *testing.T) {
	point := maths.V(5, 0, 0)
	if got := ballTurned().marching.clear(point); got != 0 {
		t.Errorf("a point outside the turned ball is cleared by %g", got)
	}
	if got := stagedBall(1000).marching.clear(point); got != 0 {
		t.Errorf("a point outside the ball is cleared by %g", got)
	}
}

// litTop is the light of a look straight down at the top of a ball of
// radius one, in its own space where its middle is middle, placed by at
// so that the middle is the world's, under a sun of one along z; the
// ball is wide metres across in the world.
func litTop(at maths.Subspace, middle maths.Vec, wide float64) float64 {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: farBall{middle: middle, radius: 1},
		Motion: thing.Still(at)})
	along := maths.V(0, 0, 1)
	light := maths.Grey(1)
	world.Lamp(thing.Sun(along, light))
	look := maths.Ray{At: maths.V(0, 0, wide+5), Dir: maths.V(0, 0, -1)}
	return bright(lone(world, look, 0))
}

// A ball scaled by ten, and a ball turned, each with its own middle far
// from its own origin, is lit at the top by the matte surface's own
// number, as the ball not scaled and not turned is.
func TestAScaledOrTurnedMarchedBallFarFromItsOriginIsLitOnTop(
	t *testing.T) {
	want := 0.8 / math.Pi
	scaled := maths.Subspace{Origin: maths.V(-10000, 0, 0), Scale: 10}
	turned := maths.Subspace{Orientation: maths.Rotation(
		maths.V(0, 0, 1), math.Pi/2)}
	turned.Origin = turned.Out(maths.V(1000, 0, 0)).Neg()
	for name, got := range map[string]float64{
		"scaled": litTop(scaled, maths.V(1000, 0, 0), 20),
		"turned": litTop(turned, maths.V(1000, 0, 0), 2)} {
		if !within(got, want, sunwardClose) {
			t.Errorf("the %s ball reads %v at the top, want %v", name, got,
				want)
		}
	}
}

// Anti: the same lit ball scaled by ten and turned about its own origin
// is the same as it is not: a ball at its own origin is lit at the top.
func TestAScaledMarchedBallAtItsOwnOriginIsLitOnTop(t *testing.T) {
	scaled := maths.Subspace{Scale: 10}
	if got := litTop(scaled, maths.Vec{}, 20); !within(got, 0.8/math.Pi,
		sunwardClose) {
		t.Errorf("the scaled ball reads %v at the top, want %v", got,
			0.8/math.Pi)
	}
}

// A world of shapes that answer a ray exactly marches nothing, and is
// lifted by the lift's share of the larger of the spread and the point's
// distance from the origin, as it was.
func TestAWorldOfExactShapesIsLiftedAsItWas(t *testing.T) {
	world := thing.NewWorld()
	sphereAt := resting(maths.V(farAway, 0, 0))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Motion: sphereAt})
	scene := stage(world, 0)
	point := maths.V(farAway, 0, 1)
	if len(scene.marching) != 0 {
		t.Errorf("a sphere is marched")
	}
	want := lift * max(scene.spread, point.Len())
	if got := scene.lifting(point); got != want {
		t.Errorf("the lift is %g, want %g", got, want)
	}
}

// Anti: the same world with a marched ball at its own origin in it is
// lifted off the ball by the ball's two touches, which are the more.
func TestAWorldWithAMarchedBallIsLiftedByItsTouches(t *testing.T) {
	scene := ballAt(maths.Vec{})
	point := maths.V(0, 0, 1)
	if len(scene.marching) != 1 {
		t.Fatalf("%d things are marched, want one", len(scene.marching))
	}
	old := lift * max(scene.spread, point.Len())
	got := scene.lifting(point)
	if !(got > old) || !within(got, ballClearance, 1e-9*ballClearance) {
		t.Errorf("the lift is %g, want %g, over %g", got, ballClearance,
			old)
	}
}

// endlessFloor is a marched shape with no edge: the world below a level
// of y, its own origin on the level.
type endlessFloor struct{}

// Bounds is the box of everything.
func (endlessFloor) Bounds() maths.Box {
	return maths.Box{Min: maths.V(math.Inf(-1), math.Inf(-1),
		math.Inf(-1)), Max: maths.V(math.Inf(1), math.Inf(1),
		math.Inf(1))}
}

// Distance is how far a point is above the level.
func (endlessFloor) Distance(point maths.Vec) float64 { return point.Y }

// A shape with no edge has no reach, and is measured where the march is:
// at 3, 4, 0, five from its own origin, a touch of five is met, and two
// touches of five clear it.
func TestAnEdgelessMarchedShapeIsClearedWhereTheMarchIs(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: endlessFloor{}})
	scene := stage(world, 0)
	got := scene.marching.clear(maths.V(3, 4, 0))
	if want := 2 * touch * 5; !within(got, want, 1e-9*want) {
		t.Errorf("an edgeless shape is cleared by %g, want %g", got, want)
	}
}

// Anti: a shape with an edge is cleared at a point outside its box of
// nothing, however far out the point is from its own origin.
func TestAnEdgedMarchedShapeClearsNothingOutsideItsBox(t *testing.T) {
	if got := ballAt(maths.Vec{}).marching.clear(maths.V(3, 4, 0)); got != 0 {
		t.Errorf("a point outside a ball is cleared by %g", got)
	}
}
