package main

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"pgregory.net/rapid"
)

//go:embed testdata/framing.yaml
var framingRaw []byte

type framingContract struct {
	From, To                       maths.Vec
	Shift, Yaw, Padding, Tolerance float64
	Elevation                      float64
}

func TestGroundObservatoryCameraUsesTheDeclaredElevation(test *testing.T) {
	var data framingContract
	if err := yamlfile.Read("framing.yaml", framingRaw, &data); err != nil {
		test.Fatal(err)
	}
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{
		"palomar", "keck-i", "keck-ii", "subaru", "rubin", "elt"})
	if err != nil {
		test.Fatal(err)
	}
	plan := defaultPlan(test)
	for _, entry := range entries {
		object, err := plan.construct(entry, "quick")
		if err != nil {
			test.Fatal(err)
		}
		centre := object.bounds.Min.Add(object.bounds.Max).Scale(halfExtent)
		direction := object.eye.Origin.Sub(centre)
		pitch := math.Atan2(direction.Y, math.Hypot(direction.X, direction.Z))
		if math.Abs(pitch/maths.Degree-data.Elevation) > data.Tolerance {
			test.Fatal("camera ignored its declared elevation", entry.Kind, pitch)
		}
	}
}

type framingProperty struct {
	framingContract
	plan *plan
}

func TestMovingBoundsMovesTheCameraWithoutChangingItsHeading(test *testing.T) {
	var data framingContract
	if err := yamlfile.Read("framing.yaml", framingRaw, &data); err != nil {
		test.Fatal(err)
	}
	property := framingProperty{framingContract: data,
		plan: defaultPlan(test)}
	rapid.Check(test, property.translated)
}

func (data framingProperty) translated(test *rapid.T) {
	shift := maths.V(
		rapid.Float64Range(-data.Shift, data.Shift).Draw(test, "east"),
		rapid.Float64Range(-data.Shift, data.Shift).Draw(test, "up"),
		rapid.Float64Range(-data.Shift, data.Shift).Draw(test, "south"))
	yaw := rapid.Float64Range(-data.Yaw, data.Yaw).Draw(test, "yaw")
	world := thing.NewWorld()
	bounds := maths.SegmentBox(data.From, data.To, data.Padding)
	first, err := data.plan.view(world, bounds, noEntries, yaw, maths.YAxis)
	if err != nil {
		test.Fatal(err)
	}
	moved := maths.SegmentBox(data.From.Add(shift),
		data.To.Add(shift), data.Padding)
	second, err := data.plan.view(world, moved, noEntries, yaw, maths.YAxis)
	if err != nil {
		test.Fatal(err)
	}
	displacement := second.eye.Origin.Sub(first.eye.Origin)
	if displacement.Sub(shift).Len() > data.Tolerance {
		test.Fatal("moving bounds changed framing", shift, displacement)
	}
	for _, axis := range []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis} {
		before := first.eye.Orientation.Out(axis)
		after := second.eye.Orientation.Out(axis)
		if before.Sub(after).Len() > data.Tolerance {
			test.Fatal("translation turned the camera", before, after)
		}
	}
}

func TestLooseShootStemsReadUprightInTheCatalogue(test *testing.T) {
	var data framingContract
	if err := yamlfile.Read("framing.yaml", framingRaw, &data); err != nil {
		test.Fatal(err)
	}
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{
		"coast-redwood-shoot", "giant-sequoia-shoot"})
	if err != nil {
		test.Fatal(err)
	}
	plan := defaultPlan(test)
	for _, entry := range entries {
		object, err := plan.construct(entry, "quick")
		if err != nil {
			test.Fatal(err)
		}
		// The loose-shoot maker turns its main stem onto positive X.
		stem := object.eye.Orientation.In(maths.XAxis)
		if math.Abs(stem.X) > data.Tolerance || stem.Y <= zeroLength {
			test.Fatal("shoot stem reads sideways", entry.Kind, stem)
		}
	}
}
