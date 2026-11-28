package workingview

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/gogpu/gputypes"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

func TestSplitWorldPacksNativeResidualAndEyeWithoutMovingMesh(t *testing.T) {
	eye := maths.V(400000000, -800000000, 500000000)
	near := maths.V(.125, -1.5, -31.75)
	position := eye.Add(near)
	vertex := placement.DisplayVertex{Position: position, Normal: maths.YAxis, Colour: maths.RGB(.2, .4, .6)}
	normal := vertexData([]placement.DisplayVertex{vertex}, false)
	wide := vertexData([]placement.DisplayVertex{vertex}, true)
	if len(normal) != 36 || len(wide) != 48 || string(normal) != string(wide[:36]) {
		t.Fatal("split position changed the normal, colour or narrow layout")
	}
	camera := uniformData(matrix{}, eye, [4]float32{})
	value := func(buffer []byte, at int) float32 {
		return math.Float32frombits(binary.LittleEndian.Uint32(buffer[at:]))
	}
	for axis, want := range [3]float64{near.X, near.Y, near.Z} {
		high := value(wide, axis*4) - value(camera, 64+axis*4)
		low := value(wide, 36+axis*4) - value(camera, 80+axis*4)
		if float64(high+low) != want {
			t.Fatalf("axis %d recovered %g, want %g", axis, high+low, want)
		}
	}
}

func TestSplitWorldRequiresAttributeStrideAndBufferLimits(t *testing.T) {
	limits := gputypes.Limits{MaxVertexAttributes: 4, MaxVertexBufferArrayStride: 48, MaxBufferSize: 240}
	if !splitWorldFits(limits, 5) || splitWorldFits(limits, 6) {
		t.Fatal("split buffer limit was not applied before packing")
	}
	limits.MaxVertexAttributes = 3
	if splitWorldFits(limits, 1) {
		t.Fatal("split layout exceeded the attribute limit")
	}
	limits.MaxVertexAttributes = 4
	limits.MaxVertexBufferArrayStride = 47
	if splitWorldFits(limits, 1) {
		t.Fatal("split layout exceeded the stride limit")
	}
}
