//go:build draw && darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// crowdMeanShare is how far the gpu's mean light over a picture lit by a
// crowd may be from the tracer's, as a share of the tracer's. Both pick
// their lamps by a tree and neither picks the other's lamps, so the two
// agree in the mean and not pixel for pixel. Picked, not measured: the
// first draw of this test says what it is, and it is tightened to about
// five times that.
const crowdMeanShare = 0.05

// crowdSide is the rays a pixel side this test draws at, enough that a
// picture's mean is not one pick's noise.
const crowdSide = 4

// meanLight is a picture's light, red, green and blue, averaged over its
// pixels.
func meanLight(picture *frame.Frame) float64 {
	sum := 0.0
	for _, pixel := range picture.Pix {
		sum += float64(pixel.R) + float64(pixel.G) + float64(pixel.B)
	}
	return sum / float64(3*len(picture.Pix))
}

// manyLamps is a world with n lamps in a row over the floor, dim enough
// that together they are a sensible exposure, and a sun.
func manyLamps(t *testing.T, n int) (*thing.World, maths.Subspace) {
	world, eye := lamplit(t)
	for each := range n {
		along := float64(each) - float64(n)/2
		world.Lamp(emit.Lit{At: maths.V(along*0.5, 3+float64(each%4),
			2-float64(each%7)*0.5),
			Strength: maths.RGB(0.6, 0.5, 0.4)})
	}
	world.Lamp(film.Sun{Toward: maths.V(-0.4, 0.7, 0.3).Unit(),
		Strength: maths.RGB(0.3, 0.3, 0.3)})
	return world, eye
}

// A world of many lamps is lit by the tree as the tracer lights it: the
// two pictures have the same light on average, surfaces and the haze's
// steps together, though not the same pick at any point.
func TestTheGPUDrawsManyLampsAsSlayerDoesOnAverage(t *testing.T) {
	world, eye := manyLamps(t, 5*lampsEvery)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	drawn, err := Sampled(world, eye, lens, 0, lampWide, lampHigh, crowdSide)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, lampWide, lampHigh,
		crowdSide)
	if err != nil {
		t.Fatal(err)
	}
	got, want := meanLight(drawn), meanLight(truth)
	t.Logf("mean light: gpu %.6f, tracer %.6f", got, want)
	if !(want > 0) || math.Abs(got-want) > crowdMeanShare*want {
		t.Errorf("the gpu's mean light is %.6f, the tracer's %.6f, more "+
			"than %g apart", got, want, crowdMeanShare)
	}
}

// Anti-test: lampsEvery lamps are each lit exactly, to the tolerance of
// every other exact picture; the tree changes nothing at the line.
func TestTheGPUKeepsLampsEveryExact(t *testing.T) {
	world, eye := manyLamps(t, lampsEvery-1)
	if len(world.Lamps) != lampsEvery {
		t.Fatalf("the world has %d lamps, want %d", len(world.Lamps),
			lampsEvery)
	}
	againstAt(t, world, eye, lampWide, lampHigh, lampSide)
}
