//go:build darwin

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
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// floor is a glossy floor of squares, side by side, two faces each:
// enough faces for the tree to branch many times over
func floor(t *testing.T, side int) *shape.Mesh {
	var corners []maths.Vec
	var faces [][3]int
	middle := float64(side) / 2
	for row := range side + 1 {
		for column := range side + 1 {
			corner := maths.V(float64(column), 0, float64(row))
			corners = append(corners, corner.Sub(maths.V(middle, 0,
				middle)))
		}
	}
	for row := range side {
		for column := range side {
			near := row*(side+1) + column
			far := near + side + 1
			faces = append(faces, [3]int{near, far, near + 1},
				[3]int{near + 1, far, far + 1})
		}
	}
	mesh, err := shape.NewMesh(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return mesh
}

// at is a thing standing still at a point
func at(x, y, z float64) thing.Motion {
	return thing.Still(maths.Subspace{Origin: maths.V(x, y, z)})
}

// hall is everything the kernel draws in one picture, each placed so
// it changes the picture: a glossy floor of 1,800 faces mirroring the
// star and a ball, the ball's shadow on the floor, a clear ball and a
// clear awning that cast none, a solid awning that does, a sky, and
// two volumes that overlap, one wispy, lit by a point emitter. the
// star's own clear flag changes nothing, since the lamp starts beyond
// its surface; the clear ball and awning are what test the flag.
func hall(t *testing.T) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(0.01, 0.01, 0.02), Seed: 3}
	clear := thing.Attrs{Clear: true}
	star := &thing.Material{Name: "star", Glow: maths.RGB(2, 1.8, 1.5)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: star,
		Attrs: clear})
	ball := &thing.Material{Name: "ball", Colour: maths.RGB(0.3, 0.5, 0.7)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5}, Material: ball,
		Motion: at(1.2, -0.6, 1.5)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.4}, Material: ball,
		Motion: at(-1.5, -0.7, 1.5), Attrs: clear})
	gloss := &thing.Material{Name: "floor", Colour: maths.Grey(0.4),
		Gloss: 0.8, Index: 2}
	world.Add(thing.Thing{Shape: floor(t, 30), Material: gloss,
		Motion: at(0, -1.5, 0)})
	awning := &thing.Material{Name: "awning", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: floor(t, 4), Material: awning,
		Motion: at(0, -0.8, -3)})
	world.Add(thing.Thing{Shape: floor(t, 2), Material: awning,
		Motion: at(3, -0.9, 0), Attrs: clear})
	world.Lamp(emit.Lit{Beyond: 1.02, Strength: maths.RGB(9, 8, 7)})
	haze := &thing.Material{Name: "haze", Colour: maths.RGB(0.9, 0.8, 0.7)}
	world.AddVolume(thing.Volume{Medium: medium.Wispy{
		Of: medium.Puff{Radius: 5, Thick: 0.1}, Seed: 3, Scale: 1.5,
		Amount: 0.5}, Material: haze})
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 3,
		Thick: 0.2}, Material: haze, Motion: at(-2, 0, 1)})
	from := maths.V(2, 5, 12)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// orrery is spheres alone, so the tree is empty and the kernel must not
// walk it: a star and three balls round it, one clear
func orrery() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	star := &thing.Material{Name: "star", Glow: maths.RGB(2, 1.8, 1.5)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: star,
		Attrs: thing.Attrs{Clear: true}})
	ball := &thing.Material{Name: "ball", Colour: maths.RGB(0.7, 0.5, 0.3),
		Gloss: 0.5}
	places := []maths.Vec{maths.V(3, 0, 0), maths.V(-2, 1, 2),
		maths.V(0, -2, -3)}
	for index, place := range places {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.6},
			Material: ball, Motion: at(place.X, place.Y, place.Z),
			Attrs: thing.Attrs{Clear: index == 1}})
	}
	world.Lamp(emit.Lit{Beyond: 1.02, Strength: maths.RGB(9, 8, 7)})
	from := maths.V(1, 3, 10)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// difference is how far apart two frames are: the mean difference in
// light as a share of the mean light, the largest difference in any one
// pixel as a share of the brightest, and the pixels whose cover or
// depth disagree
type difference struct {
	mean, largest float64
	cover, depth  int
}

// compare measures two frames of the same size against each other
func compare(truth, drawn *frame.Frame, width, height int) difference {
	var found difference
	var total, brightest float64
	for row := range height {
		for column := range width {
			one := truth.At(column, row)
			other := drawn.At(column, row)
			light := float64(one.R + one.G + one.B)
			apart := math.Abs(float64(one.R-other.R)) +
				math.Abs(float64(one.G-other.G)) +
				math.Abs(float64(one.B-other.B))
			total += light
			brightest = math.Max(brightest, light)
			found.mean += apart
			found.largest = math.Max(found.largest, apart)
			if math.Abs(float64(one.Cover-other.Cover)) > 1e-3 {
				found.cover++
			}
			if !sameDepth(one.Depth, other.Depth) {
				found.depth++
			}
		}
	}
	found.mean /= total
	found.largest /= brightest
	return found
}

// sameDepth is two depths within a part in a thousand, or both nothing
func sameDepth(one, other float32) bool {
	if math.IsInf(float64(one), 1) || math.IsInf(float64(other), 1) {
		return one == other
	}
	return math.Abs(float64(one-other)) <= 1e-3*float64(one)
}

// against draws a world both ways at 96 by 64, four rays per pixel
func against(t *testing.T, world *thing.World, eye maths.Subspace) {
	againstAt(t, world, eye, 96, 64, 2)
}

// tolerance is how far apart the gpu may draw from slayer's tracer: on
// average, and at most, as shares of the light
type tolerance struct{ mean, largest float64 }

// exact is the tolerance with nothing baked: float where the tracer has
// double, and a triangle test that is not watertight. measured on
// 2026-10-15, the hall is 2.2 millionths apart on average and 69 at
// most; these are about five and fifteen times those.
var exact = tolerance{mean: 1e-5, largest: 1e-3}

// againstAt draws a world both ways at a size, side by side rays a pixel, with
// nothing baked, and fails when they differ by more than the exact tolerance.
// one ray a pixel at an odd size puts a ray through the picture's very middle,
// which more rays per pixel never do.
func againstAt(t *testing.T, world *thing.World, eye maths.Subspace,
	width, height, side int) {
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	within(t, world, eye, width, height, side, exact)
}

// within draws a world both ways and fails when they differ by more
// than a tolerance
func within(t *testing.T, world *thing.World, eye maths.Subspace,
	width, height, side int, allowed tolerance) {
	t.Helper()
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	withinThrough(t, world, eye, lens, width, height, side, allowed)
}

// withinThrough is within through a lens
func withinThrough(t *testing.T, world *thing.World, eye maths.Subspace,
	lens film.Sensor, width, height, side int, allowed tolerance) {
	t.Helper()
	drawn, err := Sampled(world, eye, lens, 0, width, height, side)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, width, height, side)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(truth, drawn, width, height)
	t.Logf("apart: %+v", found)
	if found.mean > allowed.mean || found.largest > allowed.largest {
		t.Errorf("the gpu's light is %.6f off slayer's on average, and "+
			"%.4f at most", found.mean, found.largest)
	}
	if found.cover > 0 || found.depth > 0 {
		t.Errorf("%d pixels' cover and %d pixels' depth differ",
			found.cover, found.depth)
	}
}

// the gpu draws what slayer's tracer draws, near enough
func TestTheGPUDrawsTheHallAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	against(t, world, eye)
}

// baked, the haze is read from a grid of cells blended between, which
// the tracer never does: measured on 2026-10-15, the hall is then 46
// millionths apart on average and 124 at most, and is held to about
// six and eight times those
func TestTheBakedHallIsCloseToSlayers(t *testing.T) {
	world, eye := hall(t)
	within(t, world, eye, 96, 64, 2,
		tolerance{mean: 3e-4, largest: 1e-3})
}

// with no faces at all the tree is an empty root, which the kernel must
// not walk; at an odd size the middle pixel's ray runs through the
// origin, where an empty root of one point would have been entered
func TestTheGPUDrawsSpheresAloneAsSlayerDoes(t *testing.T) {
	world, eye := orrery()
	againstAt(t, world, eye, 97, 65, 1)
}

// a regular floor seen head on: the middle row's rays run straight down
// an edge two faces share, and without slack at the edges 18 of them
// slipped between the faces into the sky
func TestAFloorSeenHeadOnHasNoGaps(t *testing.T) {
	world := thing.NewWorld()
	grey := &thing.Material{Name: "floor", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: floor(t, 30), Material: grey})
	world.Lamp(emit.Lit{At: maths.V(0, 6, 4),
		Strength: maths.RGB(20, 20, 20)})
	from := maths.V(0, 3, 12)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	againstAt(t, world, eye, 97, 65, 1)
}

// air is a gas, which the kernel refuses
type air struct{ medium.Puff }

func (air) Gas() gas.Gas { return gas.Gas{Name: "air"} }

// shine is a texture of one light
type shine struct{}

func (shine) At(maths.Vec) maths.Light { return maths.Grey(1) }

// every way a world can hold what the kernel does not draw yet, each
// refused by name rather than drawn without it
func TestWhatItCannotDrawIsRefused(t *testing.T) {
	smooth, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0),
		maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}},
		[]maths.Vec{maths.V(0, 0, 1), maths.V(0, 0, 1),
			maths.V(0, 0, 1)}, true)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*thing.World) film.Sensor{
		"a sun": func(world *thing.World) film.Sensor {
			world.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
			return nil
		},
		"a plane": func(world *thing.World) film.Sensor {
			world.Add(thing.Thing{Shape: shape.Plane{}})
			return nil
		},
		"a smooth mesh": func(world *thing.World) film.Sensor {
			world.Add(thing.Thing{Shape: smooth})
			return nil
		},
		"a gas": func(world *thing.World) film.Sensor {
			world.AddVolume(thing.Volume{Medium: air{}})
			return nil
		},
		"a field": func(world *thing.World) film.Sensor {
			hole := &thing.Field{Mass: 1, Reach: 1}
			world.Fields = append(world.Fields, hole)
			return nil
		},
		"the world's haze": func(world *thing.World) film.Sensor {
			world.Haze.Density = 0.1
			return nil
		},
		"a beam": func(world *thing.World) film.Sensor {
			ball := medium.Puff{Radius: 1}
			world.AddVolume(thing.Volume{Medium: ball,
				Beam: &thing.Beam{}})
			return nil
		},
		"too many volumes": func(world *thing.World) film.Sensor {
			for range mostVolumes + 1 {
				world.AddVolume(thing.Volume{
					Medium: medium.Puff{Radius: 1}})
			}
			return nil
		},
	}
	for name, add := range cases {
		world := thing.NewWorld()
		lens := add(world)
		_, err := Sampled(world, maths.Subspace{}, lens, 0, 4, 4, 1)
		var refused unsupported
		if !errors.As(err, &refused) {
			t.Errorf("%s was not refused: %v", name, err)
		}
	}
}
