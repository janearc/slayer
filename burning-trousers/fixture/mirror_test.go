package fixture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// ballInMirror is a level mirror with a glowing ball above it and an
// eye looking down at the ball's image. The law of reflection puts the
// ball, seen in the mirror, where its image is: as far under the
// mirror as the ball is over it, and as wide as the image would look
// from the eye.
type ballInMirror struct {
	ball   maths.Vec
	radius float64
	eye    maths.Vec
}

// mirrored is the fixture: the ball one up and two ahead, a quarter
// across, the eye at the ball's height and five back from it.
var mirrored = ballInMirror{ball: maths.V(0, 1, -2), radius: 0.25,
	eye: maths.V(0, 1, 3)}

// world is the mirror and the ball. The mirror has no colour of its
// own and an index so high that fresnel gives back all but a few parts
// in a billion at any look. There are no lamps and no sky, so every
// light in the picture is the ball's glow.
func (fixture ballInMirror) world() *thing.World {
	mirror := &thing.Thing{Shape: fake.NewPlane(100, nil),
		Material: &thing.Material{Gloss: 1, Index: 1e9}}
	ball := &thing.Thing{
		Shape:    fake.NewSphere(fixture.ball, fixture.radius, nil),
		Material: &thing.Material{Glow: maths.Grey(1)}}
	return &thing.World{Things: []*thing.Thing{mirror, ball}}
}

// image is where the ball is seen in the mirror.
func (fixture ballInMirror) image() maths.Vec {
	return maths.V(fixture.ball.X, -fixture.ball.Y, fixture.ball.Z)
}

// edge is the angle, from the eye, between the image's middle and its
// edge.
func (fixture ballInMirror) edge() float64 {
	far := fixture.image().Sub(fixture.eye).Len()
	return math.Asin(fixture.radius / far)
}

// turned is the look turned by an angle toward a way, keeping only the
// part of the way square to the look.
func turned(look, toward maths.Vec, angle float64) maths.Vec {
	square := toward.Sub(look.Scale(toward.Dot(look))).Unit()
	return look.Scale(math.Cos(angle)).Add(square.Scale(math.Sin(angle)))
}

// drawn is the red the tracer draws along each look from the eye: the
// look at the image's middle, turned toward a way by each angle.
func (fixture ballInMirror) drawn(t *testing.T, toward maths.Vec,
	angles []float64) []float64 {
	t.Helper()
	look := fixture.image().Sub(fixture.eye).Unit()
	var rays []maths.Ray
	for _, angle := range angles {
		rays = append(rays, maths.Ray{At: fixture.eye,
			Dir: turned(look, toward, angle)})
	}
	picture, err := trace.Render(fixture.world(), maths.Subspace{},
		fake.NewAimed(rays...), 0, len(rays), 1)
	if err != nil {
		t.Fatal(err)
	}
	var red []float64
	for x := range rays {
		red = append(red, float64(picture.At(x, 0).R))
	}
	return red
}

// Just inside the image's edge the mirror shows the ball's whole glow,
// and just outside it shows nothing: up and down the mirror, where the
// angle of reflection decides it, and across. Inside and outside are a
// hundredth of the edge either side of it, about a fortieth of a
// degree.
func TestAMirrorShowsTheBallWhereItsImageIs(t *testing.T) {
	edge := mirrored.edge()
	ways := map[string]maths.Vec{"up the mirror": maths.V(0, 1, 0),
		"across it": maths.V(1, 0, 0)}
	for name, toward := range ways {
		for _, side := range []float64{-1, 1} {
			angles := []float64{0, side * 0.99 * edge, side * 1.01 * edge}
			want := []float64{1, 1, 0}
			red := mirrored.drawn(t, toward, angles)
			for at, angle := range angles {
				if math.Abs(red[at]-want[at]) > 1e-6 {
					t.Errorf("%s, %.3f deg from the image's middle, the "+
						"mirror shows %.6f; the image is %.0f there", name,
						angle*180/math.Pi, red[at], want[at])
				}
			}
		}
	}
}
