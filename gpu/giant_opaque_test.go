//go:build draw && darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the numbers of the opaque giant's draw test: the frame's size, how
// many columns and rows round its middle are read (all within the giant's
// disc), how many times white the ball behind is, the giant's ramp as a
// share of its radius and its density at depth, and the eye's distance.
const (
	opaqueWidth  = 65
	opaqueHeight = 49
	opaqueAcross = 15
	opaqueDown   = 11
	opaqueBright = 1e4
	opaqueRamp   = 0.08
	opaqueThick  = 500
	opaqueEye    = 4
	opaqueBehind = 50
	opaqueBall   = 10
)

// A giant over a very bright ball hides it in the native draw as it does
// in the software tracer: every pixel over the giant's disc is dark. The
// kernel's march kept the thousandth it stopped at, and a ball ten
// thousand times white made the planet bright.
func TestNativeGiantHidesAVeryBrightBallBehindIt(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{
		Medium: medium.Giant{Radius: 1, Depth: opaqueRamp,
			Thick: opaqueThick},
		Material: &thing.Material{Name: "black"},
	})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: opaqueBall},
		Material: &thing.Material{Glow: maths.Grey(opaqueBright)},
		Motion: thing.Still(maths.Subspace{
			Origin:      maths.V(0, 0, -opaqueBehind),
			Orientation: maths.Rotation(maths.XAxis, 0)})})
	eye := maths.Subspace{Origin: maths.V(0, 0, opaqueEye)}
	renderer, err := NewRenderer(32)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := renderer.Sampled(world, eye,
		film.Pinhole{Wide: 40 * maths.Degree}, 0, opaqueWidth,
		opaqueHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	var brightest float32
	for row := opaqueHeight/2 - opaqueDown; row <= opaqueHeight/2+opaqueDown; row++ {
		for column := opaqueWidth/2 - opaqueAcross; column <= opaqueWidth/2+opaqueAcross; column++ {
			light := drawn.At(column, row)
			brightest = max(brightest, light.R, light.G, light.B)
		}
	}
	if brightest != 0 {
		t.Errorf("the ball shows through the giant: brightest %v", brightest)
	}
}
