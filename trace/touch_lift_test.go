package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// farBall is a ball that says only how far a point is from it, so it is
// marched to, and whose middle is somewhere in its own space.
type farBall struct {
	middle maths.Vec
	radius float64
}

// Bounds is the box round the ball.
func (ball farBall) Bounds() maths.Box {
	reach := maths.V(ball.radius, ball.radius, ball.radius)
	return maths.Box{Min: ball.middle.Sub(reach),
		Max: ball.middle.Add(reach)}
}

// Distance is how far a point is from the ball's surface, under nought
// inside it.
func (ball farBall) Distance(point maths.Vec) float64 {
	return point.Sub(ball.middle).Len() - ball.radius
}

// sunwardSide is the light of looks down z onto the sunward side of a
// unit ball at the world's middle, under a sun of one along z, at the
// middle of the side and four places a half out; the ball's own middle
// is so far along x in its own space, and it is placed back by as much.
func sunwardSide(away float64) []float64 {
	world := thing.NewWorld()
	ballAt := resting(maths.V(-away, 0, 0))
	world.Add(thing.Thing{
		Shape:  farBall{middle: maths.V(away, 0, 0), radius: 1},
		Motion: ballAt})
	along := maths.V(0, 0, 1)
	light := maths.Grey(1)
	world.Lamp(thing.Sun(along, light))
	var lights []float64
	for _, at := range [][2]float64{{0, 0}, {0.5, 0}, {-0.5, 0}, {0, 0.5},
		{0, -0.5}} {
		look := maths.Ray{At: maths.V(at[0], at[1], 5),
			Dir: maths.V(0, 0, -1)}
		shine := bright(lone(world, look, 0))
		lights = append(lights, shine)
	}
	return lights
}

// sunwardWant is what those looks should read: a thing with no material
// is grey of 0.8, and a matte surface sends its colour over pi times the
// sun by the cosine it lands at, one at the middle and the square root
// of 0.75 a half out.
var sunwardWant = []float64{0.8 / math.Pi, 0.8 / math.Pi * math.Sqrt(0.75),
	0.8 / math.Pi * math.Sqrt(0.75), 0.8 / math.Pi * math.Sqrt(0.75),
	0.8 / math.Pi * math.Sqrt(0.75)}

// sunwardClose is how near a marched ball's light is held to the exact
// one's: a march stops within a touch of the surface, which moves the
// normal's cosine by far less than a hundredth.
const sunwardClose = 1e-2

// A shape drawn the same wherever its maker put its own origin is lit
// the same. A ball marched to, whose middle is a thousand units from
// the origin of its own space and which is placed back at the world's
// middle, is the ball at the world's middle: every look at its sunward
// side is lit, by the matte surface's own number.
//
// A march takes a hit within touch, a millionth, of the larger of the
// shape's reach and the point's distance from the shape's own origin: a
// thousandth here. The ray to the sun then starts lift, a millionth, of
// the world's lengths off the point, a few millionths here, inside that
// thousandth, and meets the ball it left: the side is black.
func TestAMarchedBallFarFromItsOwnOriginIsLitOnItsSunwardSide(
	t *testing.T) {
	for place, got := range sunwardSide(1000) {
		if !within(got, sunwardWant[place], sunwardClose) {
			t.Errorf("look %d at the sunward side reads %v, want %v",
				place, got, sunwardWant[place])
		}
	}
}

// Anti: the same ball with its middle at its own origin is lit as
// worked, at every one of the looks.
func TestAMarchedBallAtItsOwnOriginIsLitOnItsSunwardSide(t *testing.T) {
	for place, got := range sunwardSide(0) {
		if !within(got, sunwardWant[place], sunwardClose) {
			t.Errorf("look %d at the sunward side reads %v, want %v",
				place, got, sunwardWant[place])
		}
	}
}
