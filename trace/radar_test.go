package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// magellan is twelve and a half centimetres, magellan's band, which is
// radio as libtheme draws the ranges: its microwave stops at fifteen
// millimetres.
var magellan = maths.Band{Nanometres: 1.26e8}

// radarGround is a ball of radius two at the middle of the world that
// returns 0.8 of radio, as rough as a texture says, or wholly rough
// with none.
func radarGround(rough texture.Texture) thing.Thing {
	returns := map[radiation.Range]float64{radiation.Radio(): 0.8}
	return thing.Thing{Shape: shape.Sphere{Radius: 2},
		Material: &thing.Material{Name: "ground", Rough: rough,
			Radiant: thing.Spectrum{Returns: returns}}}
}

// radarLamp is a radio radar of strength one at a place.
func radarLamp(at maths.Vec) film.Radar {
	return film.Radar{At: at, Range: radiation.Radio(), Strength: 1}
}

// radarWorld is one thing under a radar at a place.
func radarWorld(ground thing.Thing, radar maths.Vec) *thing.World {
	world := thing.NewWorld()
	world.Add(ground)
	world.Lamp(radarLamp(radar))
	return world
}

// echoOf is what comes back in magellan's band from the middle of a
// world, to the eye ten back.
func echoOf(world *thing.World) float64 {
	lens := film.Pinhole{Wide: 10 * maths.Degree}
	picture := drawIn(world, tenBack(), lens, 0, 3, 3, magellan)
	return float64(picture.At(1, 1).R)
}

// Under a radar at the eye, a thing that returns radio shows by how
// squarely the beam lands: all of what it returns at the middle, and
// toward the edge that times the cosine between the ball's normal
// there and the way back to the radar.
func TestARadarsEchoIsByHowSquarelyTheBeamLands(t *testing.T) {
	eye, lens := tenBack(), film.Pinhole{Wide: 30 * maths.Degree}
	world := radarWorld(radarGround(nil), eye.Origin)
	picture := drawIn(world, eye, lens, 0, 41, 41, magellan)
	middle, edge := picture.At(20, 20).R, picture.At(20, 7).R
	if !within(float64(middle), 0.8, 1e-3) {
		t.Errorf("the middle returns %v, want 0.8", middle)
	}
	aim := film.Look(eye, lens, 20, 7, 41, 41, magellan)
	along, met := shape.Sphere{Radius: 2}.Hit(aim, 0, nowhere)
	if !met {
		t.Fatal("the edge pixel's ray misses the ball")
	}
	point := aim.Point(along)
	square := point.Unit().Dot(eye.Origin.Sub(point).Unit())
	if !(square < 0.9) || !within(float64(edge), 0.8*square, 1e-3) {
		t.Errorf("the edge returns %v, want %v", edge, 0.8*square)
	}
}

// A radar lights nothing in visible light.
func TestARadarLightsNothingInVisibleLight(t *testing.T) {
	eye, lens := tenBack(), film.Pinhole{Wide: 30 * maths.Degree}
	world := radarWorld(radarGround(nil), eye.Origin)
	seen := draw(world, eye, lens, 0, 41, 41).At(20, 20)
	if seen.Cover != 1 || seen.Light() != (maths.Light{}) {
		t.Errorf("in visible light the radar's ground is %+v", seen)
	}
}

// What comes back is by how rough the ground is where the beam lands:
// a quarter as rough returns a quarter as much.
func TestASmootherGroundReturnsLess(t *testing.T) {
	quarter := texture.Checker(100, maths.Grey(0.25), maths.Grey(0.25))
	smooth := radarWorld(radarGround(quarter), tenBack().Origin)
	if got := echoOf(smooth); !within(got, 0.8*0.25, 1e-3) {
		t.Errorf("a quarter as rough returns %v, want 0.2", got)
	}
}

// A thing that returns nothing of a band is dark in it, under any
// radar.
func TestAThingThatReturnsNothingIsDarkToARadar(t *testing.T) {
	wall := thing.Thing{Shape: shape.Sphere{Radius: 2},
		Material: &thing.Material{Name: "wall"}}
	if got := echoOf(radarWorld(wall, tenBack().Origin)); got != 0 {
		t.Errorf("a wall returns %v", got)
	}
}

// Only a radar that beams the band adds to its picture, and only where
// its beam lands on the side it comes from. A sun beside the radar
// adds nothing; a radar of another range adds nothing; and a radar
// behind the ball leaves the side the eye sees dark.
func TestWhatAddsNothingToAnEcho(t *testing.T) {
	alone := echoOf(radarWorld(radarGround(nil), tenBack().Origin))
	if !within(alone, 0.8, 1e-3) {
		t.Fatalf("the radar alone returns %v, want 0.8", alone)
	}
	sunlit := radarWorld(radarGround(nil), tenBack().Origin)
	sunlit.Lamp(thing.Sun(maths.V(0, 0, 1), maths.Grey(9)))
	if got := echoOf(sunlit); got != alone {
		t.Errorf("with a sun beside the radar the echo is %v, not %v",
			got, alone)
	}
	gamma := thing.NewWorld()
	gamma.Add(radarGround(nil))
	gamma.Lamp(film.Radar{At: tenBack().Origin, Range: radiation.Gamma(),
		Strength: 1})
	if got := echoOf(gamma); got != 0 {
		t.Errorf("a gamma beam comes back as radio: %v", got)
	}
	behind := radarWorld(radarGround(nil), maths.V(0, 0, -10))
	if got := echoOf(behind); got != 0 {
		t.Errorf("a radar behind the ball lights its front: %v", got)
	}
}

// hollow is a ball that says a ray does not cross it, while naming
// the places a ray down z from ten back would enter and leave a ball
// of radius one, so that a caller who ignores the answer shows.
type hollow struct{ ball shape.Sphere }

// Bounds is the ball's.
func (shell hollow) Bounds() maths.Box { return shell.ball.Bounds() }

// Distance is the ball's.
func (shell hollow) Distance(point maths.Vec) float64 {
	return shell.ball.Distance(point)
}

// Span is nine and eleven, and no.
func (hollow) Span(maths.Ray, float64, float64) (float64, float64, bool) {
	return 9, 11, false
}

// A thing that passes a band but whose shape finds no way through it
// for the ray is dark: nothing behind it shows.
func TestAThingWithNoWayThroughIsDarkInItsBand(t *testing.T) {
	passing := map[radiation.Range]float64{radiation.Gamma(): 0.5}
	world := thing.NewWorld()
	world.Add(source())
	world.Add(thing.Thing{Shape: hollow{ball: shape.Sphere{Radius: 1}},
		Material: &thing.Material{Name: "shell",
			Radiant: thing.Spectrum{Passes: passing}}})
	lens := film.Pinhole{Wide: 2 * maths.Degree}
	seen := drawIn(world, tenBack(), lens, 0, 3, 3, gammaBand).At(1, 1)
	if seen.R != 0 || seen.Cover != 1 {
		t.Errorf("behind a shell with no way through the source is %+v",
			seen)
	}
}

// Smoke between a radar and its ground thins the beam on its way in
// and the echo on its way back, each by what the smoke lets through.
func TestSmokeInARadarsWayThinsTheEchoTwice(t *testing.T) {
	const density, deep = 0.5, 1.0
	world := radarWorld(radarGround(nil), tenBack().Origin)
	world.AddVolume(thing.Volume{Medium: plate(density, 4, 4+deep)})
	once := math.Exp(-density * deep)
	if got := echoOf(world); !within(got, 0.8*once*once, 1e-3) {
		t.Errorf("through smoke the echo is %v, want %v; thinned "+
			"once it would be %v", got, 0.8*once*once, 0.8*once)
	}
}

// A volume that gives off a band adds, at each step, what it gives
// times what a volume that thick would absorb of light; and one that
// passes all of the band glows the same, with nothing taken from what
// is behind it.
func TestAVolumeGlowsInABandByWhatItWouldAbsorb(t *testing.T) {
	const density, deep, gives = 0.7, 2.0, 3.0
	glowing := &thing.Material{Name: "glow", Radiant: thing.Spectrum{
		Gives:  map[radiation.Range]float64{radiation.Gamma(): gives},
		Passes: map[radiation.Range]float64{radiation.Gamma(): 1}}}
	scene := staging(thing.Volume{Medium: plate(density, -1, 1),
		Material: glowing})
	got := throughIn(scene, alongZ, math.Inf(1), gammaBand)
	step := density * deep / strides
	want := strides * (1 - math.Exp(-step)) * gives
	if got.keep != 1 || !within(got.added, want, 1e-9) {
		t.Errorf("a glowing slab keeps %v and adds %v, want all and %v",
			got.keep, got.added, want)
	}
}
