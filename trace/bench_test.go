package trace

import (
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// earthShot is the earth's template alone under a sun, and an eye three
// radii out, which the two earth benchmarks draw.
func earthShot(b *testing.B) (*thing.World, maths.Subspace) {
	b.Helper()
	world := thing.NewWorld()
	world.Lamp(thing.Sun(maths.V(-1, 0.2, 0.4), maths.RGB(1.2, 1.2, 1.2)))
	earth := template.Earth()
	earth.Moons = nil
	err := earth.Place(world, maths.Vec{}, 1/earth.Radius, 8)
	if err != nil {
		b.Fatal(err)
	}
	return world, lookingAt(maths.V(0.1, 0.5, 3.1), maths.Vec{})
}

// drawEarth draws the earth at a time of its day, once a round.
func drawEarth(b *testing.B, at time.Duration) {
	world, eye := earthShot(b)
	lens := film.Pinhole{Wide: 40 * maths.Degree}
	b.ResetTimer()
	for range b.N {
		draw(world, eye, lens, at, 32, 18)
	}
}

// The earth costs about the same to draw at any time of its day. That
// is read by putting these two side by side, with benchstat, and not
// by a test: a test of wall-clock time fails on a busy machine and
// passes when both draws are slow.
func BenchmarkEarthAtNought(b *testing.B) { drawEarth(b, 0) }

// BenchmarkEarthFourHoursOn is the same earth four and a half hours
// into its day.
func BenchmarkEarthFourHoursOn(b *testing.B) {
	drawEarth(b, 4*time.Hour+30*time.Minute)
}

// BenchmarkGasPicture draws earthlike, a ball in gas air, side on to
// its sun: every step through the air asks the gas for its optics.
func BenchmarkGasPicture(b *testing.B) {
	world := earthlike()
	eye := lookingAt(maths.V(0, 0, 3), maths.Vec{})
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	b.ResetTimer()
	for range b.N {
		draw(world, eye, lens, 0, 64, 36)
	}
}

// BenchmarkThroughManyVolumes is one ray's march through six plates
// that overlap in pairs, which is the stepper and the crossings and
// little else.
func BenchmarkThroughManyVolumes(b *testing.B) {
	var plates []thing.Volume
	for i := range 6 {
		near := float64(2 * i)
		plates = append(plates, thing.Volume{
			Medium: plate(0.05, near, near+3)})
	}
	scene := staging(plates...)
	b.ResetTimer()
	for range b.N {
		through(scene, alongZ, 1e9)
	}
}

// BenchmarkFirstManyThings is one ray's search for the nearest of
// sixty-four balls in a row.
func BenchmarkFirstManyThings(b *testing.B) {
	world := thing.NewWorld()
	for i := range 64 {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.4},
			Motion: resting(maths.V(0, 0, float64(i)))})
	}
	crowded := gathered(place(world, 0))
	b.ResetTimer()
	for range b.N {
		crowded.first(alongZ, 0)
	}
}
