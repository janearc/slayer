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
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// swellSamples and swellExtent are the sheet's samples a side and its side
// in metres: cells of a metre and a quarter, coarse enough that the cell
// normal and the vertex slopes part visibly. Chosen.
const (
	swellSamples = 33
	swellExtent  = 40.0
)

// swellRise, swellAlong and swellAcross are the swell's amplitude in
// metres and its wave numbers a metre along x and z. Chosen.
const (
	swellRise   = 0.08
	swellAlong  = 0.8
	swellAcross = 0.7
)

// swellSatin is the water's highlight width: above nought, so the lamp
// leaves a highlight the shading normal moves. Water with none has no
// highlight, and its picture is the same with slopes or without.
const swellSatin = 0.2

// swellApart is the tolerance the clear sheet is held to: the spectral
// sheet's in ocean_spectral_normal_draw_test.go, since both pass the same
// cell march and the same packed slopes.
var swellApart = tolerance{mean: 2e-4, largest: 5e-2}

// swellColumns, swellRows and swellRays are the picture's width and height
// in pixels and its rays along a side of a pixel.
const (
	swellColumns = 96
	swellRows    = 64
	swellRays    = 2
)

// swellLamp is where the lamp stands: across the look point from the eye,
// so its mirror image on the water, about 12 degrees off the middle of the
// view, is in the picture. Chosen.
var swellLamp = maths.V(-4, 10, -6)

// swellHeights is the swell sampled onto the sheet's grid, row by row.
func swellHeights() []float64 {
	heights := make([]float64, swellSamples*swellSamples)
	step := swellExtent / float64(swellSamples-1)
	for row := range swellSamples {
		for column := range swellSamples {
			x := float64(column)*step - swellExtent/2
			z := float64(row)*step - swellExtent/2
			heights[row*swellSamples+column] = swellRise *
				math.Sin(x*swellAlong) * math.Cos(z*swellAcross)
		}
	}
	return heights
}

// swell is a clear sheet of water under a lamp over a lit floor, with the
// vertex-slope shading normals a set names `shading-normal:
// bilinear-vertex-slopes` when smooth, and the cell's own when not.
func swell(t *testing.T, smooth bool) (*thing.World, maths.Subspace) {
	t.Helper()
	field, err := shape.NewHeightField(swellHeights(), swellSamples,
		swellExtent)
	if err != nil {
		t.Fatal(err)
	}
	if smooth {
		if field, err = field.WithApproximateSmoothNormals(); err != nil {
			t.Fatal(err)
		}
	}
	world := thing.NewWorld()
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1, Satin: swellSatin}
	world.Add(thing.Thing{Shape: field, Material: water})
	floor := &thing.Material{Name: "floor", Glow: maths.RGB(0.6, 0.55, 0.4)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 25}, Material: floor,
		Motion: at(0, -27, 0)})
	world.Lamp(emit.Lit{At: swellLamp, Strength: maths.RGB(400, 380, 350)})
	from := maths.V(1.5, 12, 2)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -0.5, 0))}
}

// bothWays draws a world on the gpu and on slayer's tracer at the swell's
// size.
func bothWays(t *testing.T, world *thing.World,
	eye maths.Subspace) (drawn, truth *frame.Frame) {
	t.Helper()
	return bothWaysAt(t, world, eye, swellColumns, swellRows, swellRays)
}

// bothWaysAt draws a world on the gpu and on slayer's tracer at a size and
// rays along a side of a pixel, through the lens `within` uses, or skips
// the test where there is no device.
func bothWaysAt(t *testing.T, world *thing.World, eye maths.Subspace,
	columns, rows, rays int) (drawn, truth *frame.Frame) {
	t.Helper()
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	drawn, err := Sampled(world, eye, lens, 0, columns, rows, rays)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err = slayer.Sampled(world, eye, lens, 0, columns, rows, rays)
	if err != nil {
		t.Fatal(err)
	}
	return drawn, truth
}

// A set's clear sea shaded by vertex slopes is this: a clear height field
// shaded by vertex slopes. The gpu draws it as slayer's tracer does, mirror, bend and
// highlight.
func TestTheGPUDrawsAClearSheetWithVertexSlopesAsSlayerDoes(t *testing.T) {
	world, eye := swell(t, true)
	within(t, world, eye, swellColumns, swellRows, swellRays, swellApart)
}

// Anti-test: the slopes reach the picture on both tracers. With them taken
// away the highlight moves, so each tracer's two pictures are further apart
// than the two tracers may be; a gpu that dropped the slopes would fail the
// test above.
func TestVertexSlopesMoveTheClearSheetsHighlightOnBothTracers(t *testing.T) {
	smoothWorld, eye := swell(t, true)
	cellWorld, _ := swell(t, false)
	smoothDrawn, smoothTruth := bothWays(t, smoothWorld, eye)
	cellDrawn, cellTruth := bothWays(t, cellWorld, eye)
	tracer := compare(smoothTruth, cellTruth, swellColumns, swellRows)
	device := compare(smoothDrawn, cellDrawn, swellColumns, swellRows)
	if len(findings(tracer, swellApart)) == 0 {
		t.Errorf("slayer's tracer drew the same with slopes as without: %+v",
			tracer)
	}
	if len(findings(device, swellApart)) == 0 {
		t.Errorf("the gpu drew the same with slopes as without: %+v", device)
	}
}

// Anti-test: with no highlight the slopes change nothing on either tracer,
// since a clear sheet's mirror and bend take the cell's normal. This is why
// a set's clear sea with no glint draws the same with `shading-normal` as
// without, on either tracer.
func TestVertexSlopesLeaveAClearSheetWithNoHighlightUnchanged(t *testing.T) {
	smoothWorld, eye := swell(t, true)
	cellWorld, _ := swell(t, false)
	smoothWorld.Things[0].Material.Satin = 0
	cellWorld.Things[0].Material.Satin = 0
	smoothDrawn, smoothTruth := bothWays(t, smoothWorld, eye)
	cellDrawn, cellTruth := bothWays(t, cellWorld, eye)
	for _, problem := range findings(compare(cellTruth, smoothTruth,
		swellColumns, swellRows), exact) {
		t.Errorf("slayer's tracer: %s", problem)
	}
	for _, problem := range findings(compare(cellDrawn, smoothDrawn,
		swellColumns, swellRows), exact) {
		t.Errorf("the gpu: %s", problem)
	}
}
