// Package kernel holds slayer's gpu kernel to the same physics as the
// tracer, side by side: each fixture is drawn both ways and each is held
// to the answer worked out here. The kernel draws only scene/thing's own parts,
// so these scenes are made of them: scene/shape's spheres and meshes,
// emit's lamps and film.Pinhole.
package kernel_test

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// drawer is one way of drawing a world, one ray a pixel.
type drawer struct {
	name string
	draw func(world *thing.World, eye maths.Subspace, lens film.Sensor,
		wide, high int) (*frame.Frame, error)
}

// drawers are the tracer on the cpu and the kernel on the gpu.
var drawers = []drawer{
	{name: "the tracer", draw: func(world *thing.World, eye maths.Subspace,
		lens film.Sensor, wide, high int) (*frame.Frame, error) {
		return trace.Sampled(world, eye, lens, 0, wide, high, 1)
	}},
	{name: "the kernel", draw: func(world *thing.World, eye maths.Subspace,
		lens film.Sensor, wide, high int) (*frame.Frame, error) {
		return gpu.Sampled(world, eye, lens, 0, wide, high, 1)
	}},
}

// view is an eye, a pinhole and a picture's size.
type view struct {
	eye        maths.Subspace
	lens       film.Pinhole
	wide, high int
}

// looking is a view from a place toward another, fifty degrees across.
func looking(from, to maths.Vec, wide, high int) view {
	return view{eye: maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, to)},
		lens: film.Pinhole{Wide: 50 * maths.Degree}, wide: wide, high: high}
}

// ray is the ray the view's pixel sees, in the world.
func (seen view) ray(x, y int) maths.Ray {
	return film.Look(seen.eye, seen.lens, x, y, seen.wide, seen.high,
		maths.Band{})
}

// drawn is a world drawn one way from the view.
func (seen view) drawn(t *testing.T, way drawer,
	world *thing.World) *frame.Frame {
	t.Helper()
	picture, err := way.draw(world, seen.eye, seen.lens, seen.wide,
		seen.high)
	if err != nil {
		t.Fatalf("%s: %v", way.name, err)
	}
	return picture
}

// ground is a level square of two faces about the origin, facing up.
func ground(t *testing.T, half float64) *shape.Mesh {
	t.Helper()
	corners := []maths.Vec{maths.V(-half, 0, -half), maths.V(half, 0, -half),
		maths.V(half, 0, half), maths.V(-half, 0, half)}
	mesh, err := shape.NewMesh(corners, [][3]int{{0, 2, 1}, {0, 3, 2}},
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return mesh
}
