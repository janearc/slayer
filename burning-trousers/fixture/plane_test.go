// Package fixture holds slayer's tracer to answers physics gives in
// closed form. Every scene is made of the harness's own parts and every
// expected value is worked out here, never asked of the tracer.
package fixture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// lampOnPlane is a level plane of albedo, lit by one point lamp of
// intensity above its middle, with no sky, and rays from an eye to the
// ground at each of xs along the x axis.
type lampOnPlane struct {
	albedo, intensity, height float64
	eye                       maths.Vec
	xs                        []float64
}

// plane is the fixture used throughout: albedo a half, intensity ten,
// the lamp two up, the eye one up and three back.
var plane = lampOnPlane{albedo: 0.5, intensity: 10, height: 2,
	eye: maths.V(0, 1, 3), xs: []float64{0, 0.5, 1, 2, 4}}

// world is the plane and its lamp.
func (fixture lampOnPlane) world() *thing.World {
	ground := &thing.Thing{Shape: fake.NewPlane(100, nil),
		Material: &thing.Material{Colour: maths.Grey(fixture.albedo)}}
	lamp := fake.NewLamp(maths.V(0, fixture.height, 0),
		maths.Grey(fixture.intensity), nil)
	return &thing.World{Things: []*thing.Thing{ground},
		Lamps: []thing.Lamp{lamp}}
}

// sensor aims one pixel at each ground point.
func (fixture lampOnPlane) sensor() fake.Aimed {
	var rays []maths.Ray
	for _, x := range fixture.xs {
		toward := maths.V(x, 0, 0).Sub(fixture.eye).Unit()
		rays = append(rays, maths.Ray{At: fixture.eye, Dir: toward})
	}
	return fake.NewAimed(rays...)
}

// irradiance is the light arriving on the ground at x: the lamp's
// intensity times the cosine of its slant over the square of its
// distance.
func (fixture lampOnPlane) irradiance(x float64) float64 {
	square := x*x + fixture.height*fixture.height
	slant := fixture.height / math.Sqrt(square)
	return fixture.intensity * slant / square
}

// drawn is the picture the tracer makes of the fixture, one red value
// a ground point.
func (fixture lampOnPlane) drawn(t *testing.T) []float64 {
	t.Helper()
	picture, err := trace.Render(fixture.world(), maths.Subspace{},
		fixture.sensor(), 0, len(fixture.xs), 1)
	if err != nil {
		t.Fatal(err)
	}
	var red []float64
	for x := range fixture.xs {
		red = append(red, float64(picture.At(x, 0).R))
	}
	return red
}

// The light on the ground falls off as the cosine over the distance
// squared: each point against the one under the lamp.
func TestLitGroundFallsOffAsPhysicsSays(t *testing.T) {
	red := plane.drawn(t)
	for at, x := range plane.xs {
		want := plane.irradiance(x) / plane.irradiance(0)
		got := red[at] / red[0]
		if math.Abs(got/want-1) > 1e-5 {
			t.Errorf("at x %.1f the ground is %.6f of the middle, physics "+
				"says %.6f", x, got, want)
		}
	}
}

// A lamp twice as high lights the ground under it a quarter as much:
// the inverse square alone, with the point and the eye held still and
// no slant to the light.
func TestALampTwiceAsHighLightsAQuarter(t *testing.T) {
	near, far := plane, plane
	far.height = 2 * plane.height
	near.xs, far.xs = []float64{0}, []float64{0}
	got := far.drawn(t)[0] / near.drawn(t)[0]
	if math.Abs(got/0.25-1) > 1e-5 {
		t.Errorf("a lamp twice as high lights %.6f as much; the inverse "+
			"square says 0.25", got)
	}
}

// A matte surface sends back its albedo over pi of the light that
// arrives: radiance, the quantity a camera sees. The tracer's own
// comments say it multiplies colour by light by the cosine, with no pi,
// so this fixture says by how much that differs.
func TestLitGroundIsRadiance(t *testing.T) {
	red := plane.drawn(t)
	want := plane.albedo / math.Pi * plane.irradiance(0)
	if math.Abs(red[0]/want-1) > 1e-5 {
		t.Errorf("under the lamp the ground is %.6f; radiance is %.6f, "+
			"%.4f times that", red[0], want, red[0]/want)
	}
}
