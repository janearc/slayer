//go:build draw && darwin

package gpu

import (
	"errors"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// lampWide and lampHigh are the picture the lamp tests draw, at lampSide
// rays a pixel side, and an odd size so a ray passes through the very
// middle.
const lampWide, lampHigh, lampSide = 97, 65, 1

// lamplit is a floor, a ball standing on it, a ball in the air, and a
// puff of haze beside the low ball: surfaces that face up, sideways and
// down, and a medium. It holds no lamp: the caller adds the only light.
func lamplit(t *testing.T) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	grey := &thing.Material{Name: "floor", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: floor(t, 30), Material: grey})
	red := &thing.Material{Name: "ball", Colour: maths.RGB(0.7, 0.3, 0.2)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.7}, Material: red,
		Motion: at(0.8, 0.7, 0.2)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1.2}, Material: red,
		Motion: at(1.2, 7, -2.4)})
	haze := &thing.Material{Name: "haze", Colour: maths.RGB(0.9, 0.8, 0.7)}
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1.5,
		Thick: 0.8}, Material: haze, Motion: at(-1.6, 1.5, 1)})
	from := maths.V(0, 6, 22)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 3.5, 0))}
}

// a flood's light is its strength weighed by how a surface faces its
// dome, and for a bit of a medium by how the ray's way back faces it.
// the dome here leans off the world's up, so a flood drawn with the
// wrong dome is a different picture.
func TestTheGPUDrawsAFloodAsSlayerDoes(t *testing.T) {
	world, eye := lamplit(t)
	world.Lamp(film.Flood{Up: maths.V(0.3, 1, -0.2),
		Strength: maths.RGB(0.9, 0.8, 1.1)})
	againstAt(t, world, eye, lampWide, lampHigh, lampSide)
}

// a flood with no Up is round the world's own up.
func TestTheGPUDrawsAFloodWithNoUpAsSlayerDoes(t *testing.T) {
	world, eye := lamplit(t)
	world.Lamp(film.Flood{Strength: maths.RGB(0.9, 0.8, 1.1)})
	againstAt(t, world, eye, lampWide, lampHigh, lampSide)
}

// pulsar is a polar emitter of two cones, tilted off its spin so where
// they point depends on the time, hung between the floor and the high ball.
func pulsar() emit.Polar {
	return emit.Polar{At: maths.V(0, 4, 0),
		Light:  bands.Split(swatch.Blackbody(9000)),
		Beyond: 0.5, Width: 16 * maths.Degree, Soft: 9 * maths.Degree,
		Offset: 12 * maths.Degree, Tilt: 25 * maths.Degree,
		Spin: 4 * time.Second, Axis: maths.V(0.1, -1, 0.15),
		Cone: 10 * maths.Degree, Precess: 6 * time.Second}
}

// polarAt is a world under the pulsar's beams read at a time, with the
// strength of the light scaled so the pools are plainly lit.
func polarAt(t *testing.T, clock *time.Duration) (*thing.World,
	maths.Subspace) {
	world, eye := lamplit(t)
	polar := pulsar()
	polar.Light = polar.Light.Scale(40)
	world.Lamp(polar.Beams(clock))
	return world, eye
}

// the beams are the same two cones on the gpu as in slayer's tracer, at
// the time the clock holds when the picture is drawn.
func TestTheGPUDrawsPolarBeamsAsSlayerDoes(t *testing.T) {
	clock := 700 * time.Millisecond
	world, eye := polarAt(t, &clock)
	againstAt(t, world, eye, lampWide, lampHigh, lampSide)
}

// and the clock is in the picture: two times draw two pictures, so a
// beam read at the wrong time would show.
func TestTheBeamsPicturesChangeWithTheClock(t *testing.T) {
	clock := 700 * time.Millisecond
	world, eye := polarAt(t, &clock)
	first, err := tracedAt(world, eye)
	if err != nil {
		t.Fatal(err)
	}
	clock = 2300 * time.Millisecond
	second, err := tracedAt(world, eye)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(first, second, lampWide, lampHigh)
	if found.mean < 0.01 {
		t.Errorf("the beams at two times differ by only %g", found.mean)
	}
}

// streetLamps is how many lamps beside a flood light the nine-lamp
// street: six street lamps, a sun and a flare.
const streetLamps = 8

// a street's nine lamps, a flood among them, are each lit at every
// point, past lampsAsked and within lampsEvery: the gpu draws them,
// where it once refused the flood, and draws them as slayer's tracer.
func TestTheGPUDrawsAFloodAmongNineLampsAsSlayerDoes(t *testing.T) {
	world, eye := lamplit(t)
	world.Lamp(film.Flood{Up: maths.V(0.3, 1, -0.2),
		Strength: maths.RGB(0.3, 0.3, 0.35)})
	for index := range streetLamps - 2 {
		along := float64(index) - 2.5
		world.Lamp(emit.Lit{At: maths.V(along*2, 4, 3-along),
			Strength: maths.RGB(6, 4.5, 2.5)})
	}
	world.Lamp(film.Sun{Toward: maths.V(-0.4, 0.7, 0.3).Unit(),
		Strength: maths.RGB(0.8, 0.75, 0.7)})
	world.Lamp(emit.Lit{At: maths.V(-3, 9, -1),
		Strength: maths.RGB(20, 4, 2)})
	if lamps := len(world.Lamps); lamps <= lampsAsked ||
		lamps > lampsEvery {
		t.Fatalf("the street has %d lamps", lamps)
	}
	againstAt(t, world, eye, lampWide, lampHigh, lampSide)
}

// the flood among many lamps is refused by name: past lampsEvery the
// tracer keeps every ambient lamp and picks among the rest, which the
// kernel does not do.
func TestAFloodAmongManyLampsIsRefused(t *testing.T) {
	world := thing.NewWorld()
	world.Lamp(film.Flood{Strength: maths.Grey(1)})
	for range lampsEvery {
		world.Lamp(emit.Lit{At: maths.V(0, 5, 0), Strength: maths.Grey(1)})
	}
	_, err := Sampled(world, maths.Subspace{}, nil, 0, 4, 4, 1)
	var refused unsupported
	if !errors.As(err, &refused) {
		t.Errorf("a flood among %d lamps was not refused: %v",
			len(world.Lamps), err)
	}
}

// beams with no clock cannot be read at any time, and are refused.
func TestBeamsWithNoClockAreRefused(t *testing.T) {
	world := thing.NewWorld()
	world.Lamp(pulsar().Beams(nil))
	_, err := Sampled(world, maths.Subspace{}, nil, 0, 4, 4, 1)
	var refused unsupported
	if !errors.As(err, &refused) {
		t.Errorf("beams with no clock were not refused: %v", err)
	}
}

// tracedAt is the picture slayer's tracer draws of a world at the lamp
// tests' size.
func tracedAt(world *thing.World, eye maths.Subspace) (*frame.Frame, error) {
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	return slayer.Sampled(world, eye, lens, 0, lampWide, lampHigh, lampSide)
}
