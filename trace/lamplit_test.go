package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// xrayLine is a kiloelectronvolt, an x-ray line at 1.24 nanometres.
var xrayLine = maths.Band{Nanometres: 1.24}

// above is where the x-ray lamp stands, up and toward the eye from the
// ball's nearest point, (0, 0, 1): seven up and six along.
var above = maths.V(0, 7, 7)

// xrayLamp is a lamp of strength five in the x-ray line alone, at a place.
func xrayLamp(at maths.Vec) emit.Lit {
	return emit.Point{At: at,
		Light: bands.Lines(map[float64]float64{1.24: 5})}.When(0)
}

// scattering is a unit ball at the middle of the world that sends back
// 0.6 of the x-ray that lands on it.
func scattering() thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "ball",
			Radiant: thing.Spectrum{Returns: map[radiation.Range]float64{
				radiation.XRay(): 0.6}}}}
}

// litIn is what comes back in the x-ray line from the ball's nearest
// point, seen from ten back.
func litIn(world *thing.World) float64 {
	lens := film.Pinhole{Wide: 1 * maths.Degree}
	return float64(drawIn(world, tenBack(), lens, 0, 3, 3,
		xrayLine).At(1, 1).R)
}

// matteUnder is the ball's nearest point under the lamp above as a matte
// surface sends it: 0.6 over pi, times five over the square of the way,
// times how squarely it lands.
func matteUnder() float64 {
	way := above.Sub(maths.V(0, 0, 1))
	return 0.6 / math.Pi * 5 / way.Dot(way) * way.Unit().Z
}

// A lamp that lights in a band lights a thing that scatters it as a lamp
// lights a matte surface in the visible: its scatters over pi times the
// band's power landing there, by the inverse square and how squarely.
func TestALampLightsAThingInABand(t *testing.T) {
	world := thing.NewWorld()
	world.Add(scattering())
	world.Lamp(xrayLamp(above))
	if got, want := litIn(world), matteUnder(); !within(got, want,
		1e-6*want) {
		t.Errorf("under the x-ray lamp the ball is %v, want %v", got, want)
	}
}

// A surface in the way shadows a lamp in a band as it does in the
// visible.
func TestALampInABandIsShadowed(t *testing.T) {
	world := thing.NewWorld()
	world.Add(scattering())
	// halfway to the lamp, and clear of the eye's way to the ball
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5},
		Material: &thing.Material{Name: "wall"},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 3.5, 4)})})
	world.Lamp(xrayLamp(above))
	if got := litIn(world); got != 0 {
		t.Errorf("in the wall's shadow the ball is %v, want dark", got)
	}
}

// A volume in the way passes a lamp's band by what it passes of it: one
// that passes all of it changes nothing, and one that passes none of it
// leaves the ball dark.
func TestALampInABandPassesThroughWhatPassesIt(t *testing.T) {
	through := func(passing float64) float64 {
		world := thing.NewWorld()
		world.Add(scattering())
		world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1,
			Thick: 3},
			Material: &thing.Material{Name: "gas",
				Radiant: thing.Spectrum{Passes: map[radiation.Range]float64{
					radiation.XRay(): passing}}},
			Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 3.5, 4)})})
		world.Lamp(xrayLamp(above))
		return litIn(world)
	}
	if got, want := through(1), matteUnder(); !within(got, want,
		1e-6*want) {
		t.Errorf("through gas that passes the band, %v, want %v", got,
			want)
	}
	if got := through(0); got != 0 {
		t.Errorf("through gas that passes none of it, %v, want dark", got)
	}
}

// Only a lamp that lights in the band lights in it, only on the side it
// faces, and only a thing that scatters the band sends any of it back.
func TestALampLightsInABandOnlyWhatScattersItAndFacesIt(t *testing.T) {
	sun := thing.NewWorld()
	sun.Add(scattering())
	sun.Lamp(thing.Sun(maths.ZAxis, maths.Grey(1)))
	if got := litIn(sun); got != 0 {
		t.Errorf("under a sun with no spectrum the ball is %v in x-ray, "+
			"want dark", got)
	}
	behind := thing.NewWorld()
	behind.Add(scattering())
	behind.Lamp(xrayLamp(maths.V(0, 0, -10)))
	if got := litIn(behind); got != 0 {
		t.Errorf("lit from behind the ball is %v, want dark", got)
	}
	plain := thing.NewWorld()
	plain.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "ball",
			Radiant: thing.Spectrum{Passes: map[radiation.Range]float64{
				radiation.XRay(): 0}}}})
	plain.Lamp(xrayLamp(above))
	if got := litIn(plain); got != 0 {
		t.Errorf("a ball that scatters no x-ray is %v, want dark", got)
	}
}
