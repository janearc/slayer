package stage

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// still is the moment a lamp is lit at: no lamp on a set pulses
const still = 0

// metresPerKilometre turns a body's kilometres, which its template is
// written in, into metres
const metresPerKilometre = 1e3

// a builder puts one of the lists a set's world says into the tracer's
// world, its pieces wearing the materials built already.
type builder func(world *thing.World, said set.World,
	worn map[string]*thing.Material) error

// builders are the lists that put something into the world, by the key
// the set file writes each under. the materials and the motions are worn
// and ridden by others, and the unit is a size, so none is here.
var builders = map[string]builder{
	"boxes": boxes, "spheres": spheres, "lamps": lamps, "spots": spots,
	"floods": floods, "sky": hang, "bodies": bodies, "suns": suns,
	"wisps": wisps,
}

// boxes puts each box in, as triangles about its middle.
func boxes(world *thing.World, said set.World,
	worn map[string]*thing.Material) error {
	for _, each := range said.Boxes() {
		size := each.Size()
		shaped, err := box(size.X, size.Y, size.Z)
		if err != nil {
			return fmt.Errorf("box %s: %w", each.Name(), err)
		}
		world.Add(thing.Thing{Shape: shaped, Material: worn[each.Material()],
			Motion: thing.Still(maths.Subspace{Origin: each.Middle()})})
	}
	return nil
}

// box is a box of triangles about its own middle, each face turning
// counterclockwise seen from outside.
func box(wide, tall, deep float64) (*mesh.Mesh, error) {
	x, y, z := wide/2, tall/2, deep/2
	corners := []maths.Vec{
		maths.V(-x, -y, -z), maths.V(x, -y, -z), maths.V(x, y, -z),
		maths.V(-x, y, -z), maths.V(-x, -y, z), maths.V(x, -y, z),
		maths.V(x, y, z), maths.V(-x, y, z)}
	faces := [][3]int{{0, 3, 2}, {0, 2, 1}, {4, 5, 6}, {4, 6, 7},
		{0, 4, 7}, {0, 7, 3}, {1, 2, 6}, {1, 6, 5},
		{0, 1, 5}, {0, 5, 4}, {3, 7, 6}, {3, 6, 2}}
	return mesh.New(corners, faces, nil, false)
}

// spheres puts each sphere in, still at its middle or carried by its
// motion from there.
func spheres(world *thing.World, said set.World,
	worn map[string]*thing.Material) error {
	runs := map[string]set.Motion{}
	for _, each := range said.Motions() {
		runs[each.Name()] = each
	}
	for _, each := range said.Spheres() {
		var moving thing.Motion = thing.Still(
			maths.Subspace{Origin: each.Middle()})
		if name, carried := each.Motion(); carried {
			moving = keyed(runs[name], each.Middle())
		}
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: each.Radius()},
			Material: worn[each.Material()], Motion: moving})
	}
	return nil
}

// keyed is a motion's run, from a thing's place before it.
func keyed(run set.Motion, base maths.Vec) thing.Keyed {
	length, ease, hold := run.Run()
	var frames []thing.Keyframe
	for _, each := range run.Frames() {
		frames = append(frames, thing.Keyframe{Share: each.Share,
			At: each.At})
	}
	return thing.Keyed{Frames: frames, Length: length, Ease: ease,
		Hold: hold, Base: base}
}

// lamps lights each bulb, a black body's light from beyond its radius.
func lamps(world *thing.World, said set.World,
	_ map[string]*thing.Material) error {
	for _, each := range said.Lamps() {
		heat, strength := each.Shine()
		spectrum, _, err := blackbody(heat, strength)
		if err != nil {
			return fmt.Errorf("lamp %s: %w", each.Name(), err)
		}
		bulb := emit.Point{At: each.At(), Light: spectrum,
			Beyond: each.Radius()}
		world.Lamp(bulb.When(still))
	}
	return nil
}

// spots lights each spot, the colour libtheme gives a black body at its
// temperature, so many times over.
func spots(world *thing.World, said set.World,
	_ map[string]*thing.Material) error {
	for _, each := range said.Spots() {
		heat, strength := each.Shine()
		cone, soft := each.Cone()
		world.Lamp(film.Spot{At: each.At(), Aim: each.Aim(),
			Strength: swatches.Light(swatch.Planckian(heat)).Scale(strength),
			Cone:     cone * maths.Degree, Soft: soft * maths.Degree})
	}
	return nil
}

// floods lights each flood, its colour so many times over.
func floods(world *thing.World, said set.World,
	_ map[string]*thing.Material) error {
	for _, each := range said.Floods() {
		world.Lamp(film.Flood{
			Strength: light(each.Colour()).Scale(each.Strength())})
	}
	return nil
}

// hang hangs the sky: a gradient, or the stars.
func hang(world *thing.World, said set.World,
	_ map[string]*thing.Material) error {
	hung, _ := said.Sky()
	if stars, starry := hung.Starry(); starry {
		share, size, bright := stars.Look()
		world.Sky = sky.Stars{Seed: stars.Seed(), Share: share, Size: size,
			Bright: bright}
		return nil
	}
	horizon, zenith, ground := hung.Light()
	world.Sky = sky.Gradient{Horizon: light(horizon), Zenith: light(zenith),
		Ground: light(ground)}
	return nil
}

// bodies places each body from its template: its middle at a place, at
// the radius the set says or its own true size in the world's unit, its
// moons' ground tiled so finely, and its axis and aurora as the set says
// or its template has them.
func bodies(world *thing.World, said set.World,
	_ map[string]*thing.Material) error {
	for _, each := range said.Bodies() {
		body, _ := template.Named(each.Template())
		if tilt, tilted := each.Tilt(); tilted {
			body.Tilt = tilt * maths.Degree
		}
		if shown, says := each.Aurora(); says && !shown {
			body.Aurora = nil
		}
		scale := metresPerKilometre / said.Unit()
		if radius, sized := each.Radius(); sized {
			scale = radius / body.Radius
		}
		if err := body.Place(world, each.Middle(), scale,
			each.Detail()); err != nil {
			return fmt.Errorf("body %s: %w", each.Name(), err)
		}
	}
	return nil
}

// suns lights each sun, one way off.
func suns(world *thing.World, said set.World,
	_ map[string]*thing.Material) error {
	for _, each := range said.Suns() {
		toward, err := along(each.Toward())
		if err != nil {
			return fmt.Errorf("sun %s: %w", each.Name(), err)
		}
		world.Lamp(thing.Sun(toward, light(each.Colour())))
	}
	return nil
}

// along is a way as the world's vector, of length one. It refuses a way
// too short to say which way it points.
func along(way maths.Vec) (maths.Vec, error) {
	pointed, err := measure.NewDirection(way.X, way.Y, way.Z)
	if err != nil {
		return maths.Vec{}, err
	}
	unit := pointed.Times(measure.Metres(1))
	return maths.V(unit.AlongX().InMetres(), unit.AlongY().InMetres(),
		unit.AlongZ().InMetres()), nil
}

// wisps lets each wisp of gas in, still at its middle: a ball made
// ragged by noise, which gives back its colour of the light that
// crosses it.
func wisps(world *thing.World, said set.World,
	_ map[string]*thing.Material) error {
	for _, each := range said.Wisps() {
		radius, thick := each.Ball()
		seed, scale, amount := each.Ragged()
		world.AddVolume(thing.Volume{
			Medium: medium.Wispy{Of: medium.Puff{Radius: radius,
				Thick: thick}, Seed: seed, Scale: scale, Amount: amount},
			Material: &thing.Material{Name: each.Name(),
				Colour: light(each.Colour())},
			Motion: thing.Still(maths.Subspace{Origin: each.Middle()}),
		})
	}
	return nil
}
