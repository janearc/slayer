//go:build draw

package session

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// wireSide is the test picture's pixels each way.
const wireSide = 8

// wireBall is a world of one sphere before the eye, lit by nothing.
func wireBall() *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -4),
			Orientation: maths.Identity})})
	return world
}

// framesAlike is whether two frames hold the same pixels.
func framesAlike(one, other *frame.Frame) bool {
	if one.W != other.W || one.H != other.H {
		return false
	}
	for i := range one.Pix {
		if one.Pix[i] != other.Pix[i] {
			return false
		}
	}
	return true
}

// wireAndLight is a cpu draw of the ball as asked, and trace.Wire's of
// it from the same eye and lens.
func wireAndLight(t *testing.T, given asked) (*frame.Frame, *frame.Frame) {
	t.Helper()
	eye := maths.Subspace{Orientation: maths.Identity}
	lens := film.Pinhole{}
	drawn, err := sampled(wireBall(), given, eye, lens, 0, wireSide,
		wireSide, 1)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := trace.Wire(wireBall(), eye, lens, 0, wireSide, wireSide, 1)
	if err != nil {
		t.Fatal(err)
	}
	return drawn, wire
}

// TestWireIsDrawnByTraceWire holds that a wire asked on the cpu is
// trace.Wire's picture, pixel for pixel.
func TestWireIsDrawnByTraceWire(t *testing.T) {
	if drawn, wire := wireAndLight(t, asked{wire: true}); !framesAlike(drawn,
		wire) {
		t.Error("the cpu's wire is not trace.Wire's")
	}
}

// anti-test of TestWireIsDrawnByTraceWire
// TestLightIsNotDrawnByTraceWire holds that a draw of light is not the
// wire: the check above can tell them apart.
func TestLightIsNotDrawnByTraceWire(t *testing.T) {
	if drawn, wire := wireAndLight(t, asked{}); framesAlike(drawn, wire) {
		t.Error("the light picture is the wire's")
	}
}
