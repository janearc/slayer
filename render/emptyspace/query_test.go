package emptyspace_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/emptyspace"
)

// The fixture is a normalized unit cube with sixteen samples on each axis.
// Two samples per brick give eight cubed bits, or sixty-four bytes.
const (
	samples      = 16
	brickSide    = 2
	brickCount   = samples / brickSide
	brickTotal   = brickCount * brickCount * brickCount
	payloadBytes = brickTotal / 8
	centreSample = samples / 2
)

var cube = maths.Box{Min: maths.Vec{}, Max: maths.V(1, 1, 1)}

func options() emptyspace.Options {
	return emptyspace.Options{BrickSide: brickSide, Bytes: payloadBytes,
		Rays: 2, QueryBricks: brickTotal}
}

func prepared(test *testing.T, density, emission float32,
	policy emptyspace.Options) *emptyspace.Mask {
	test.Helper()
	values := make([]float32, samples*samples*samples)
	light := make([]float32, len(values))
	at := (centreSample*samples+centreSample)*samples + centreSample
	values[at], light[at] = density, emission
	mask, err := emptyspace.Build(cube, [3]int{samples, samples, samples},
		[]emptyspace.Field{{Values: values, Channels: 1},
			{Values: light, Channels: 1}}, policy)
	if err != nil {
		test.Fatal(err)
	}
	return mask
}

func expectBox(test *testing.T, mask *emptyspace.Mask,
	box maths.Box, want bool) {
	test.Helper()
	got, err := mask.EmptyBox(box)
	if err != nil || got != want {
		test.Fatalf("EmptyBox(%v) = %v, %v; want %v", box, got, err, want)
	}
}

func TestExteriorIsUnknownEvenForZeroStoredSamples(test *testing.T) {
	for _, density := range []float32{0, 1} {
		mask := prepared(test, density, 0, options())
		for _, box := range []maths.Box{
			{Min: maths.V(1.5, 0, 0), Max: maths.V(2, 1, 1)},
			{Min: maths.V(-1, 0, 0), Max: maths.V(0, 1, 1)},
			{Min: maths.V(0, 0, 0), Max: maths.V(2, 1, 1)},
		} {
			expectBox(test, mask, box, false)
		}
	}
	expectBox(test, prepared(test, 0, 0, options()), cube, true)
}

func TestTinySamplesAndIndependentEmissionKeepInterpolationSupport(test *testing.T) {
	// The centre is between samples seven and eight on each axis;
	// sample eight contributes one eighth to the trilinear value.
	point := maths.Box{Min: maths.V(.5, .5, .5), Max: maths.V(.5, .5, .5)}
	for _, fields := range [][2]float32{
		{math.SmallestNonzeroFloat32, 0}, {0, math.SmallestNonzeroFloat32},
	} {
		expectBox(test, prepared(test, fields[0], fields[1], options()), point, false)
	}
}

func TestProofBudgetExhaustionKeepsMarching(test *testing.T) {
	policy := options()
	policy.QueryBricks = 1
	mask := prepared(test, 1, 0, policy)
	expectBox(test, mask, maths.Box{}, false)
	// The same origin is certifiably empty when its halo can be inspected.
	expectBox(test, prepared(test, 1, 0, options()), maths.Box{}, true)
}

func TestPacketProofIncludesTheRayThatEntersOccupiedSupport(test *testing.T) {
	mask := prepared(test, 1, 0, options())
	pilot := maths.Ray{At: maths.Vec{}, Dir: maths.XAxis}
	other := maths.Ray{At: maths.V(.5, .5, .5), Dir: maths.XAxis}
	empty, err := mask.PacketEmpty([]maths.Ray{pilot}, 0, .25)
	if err != nil || !empty {
		test.Fatal("fixture pilot should miss occupied support", empty, err)
	}
	empty, err = mask.PacketEmpty([]maths.Ray{pilot, other}, 0, .25)
	if err != nil || empty {
		test.Fatal("pilot miss was shared with an occupied ray", empty, err)
	}
}

func TestPrefixesKeepMemberOrdinalsAndRefuseExteriorProof(test *testing.T) {
	mask := prepared(test, 0, 0, options())
	runs := []emptyspace.Run{
		{Ray: maths.Ray{Dir: maths.XAxis}, First: 0, Stride: .125, Count: 4},
		{Ray: maths.Ray{Dir: maths.YAxis}, First: .25, Stride: .25, Count: 2},
	}
	got, err := mask.SafePrefix(runs, 4)
	if err != nil || got != 2 {
		test.Fatal("prefix exceeded a member's existing samples", got, err)
	}
	runs[0].Ray.At = maths.V(1.5, 0, 0)
	got, err = mask.SafePrefix(runs, 4)
	if err != nil || got != 0 {
		test.Fatal("an exterior sample acquired an emptiness proof", got, err)
	}
}

func TestQueriesExactlyOnEveryFieldFaceKeepBoundarySupport(test *testing.T) {
	for axis := range 3 {
		for _, face := range []float64{0, 1} {
			low, high := [3]float64{}, [3]float64{1, 1, 1}
			low[axis], high[axis] = face, face
			query := maths.Box{Min: maths.V(low[0], low[1], low[2]),
				Max: maths.V(high[0], high[1], high[2])}
			for _, value := range []float32{0, 1} {
				mask := faceSample(test, axis, face, value)
				expectBox(test, mask, query, value == 0)
				// One representable step beyond the same face is unknown.
				direction := math.Copysign(math.Inf(1), face-.5)
				outside := math.Nextafter(face, direction)
				low[axis], high[axis] = outside, outside
				exterior := maths.Box{Min: maths.V(low[0], low[1], low[2]),
					Max: maths.V(high[0], high[1], high[2])}
				expectBox(test, mask, exterior, false)
				low[axis], high[axis] = face, face
			}
		}
	}
}

// faceSample places a value on the requested outer sample plane and near
// the other two axes' middles, independently of the queried brick address.
func faceSample(test *testing.T, axis int, face float64,
	value float32,
) *emptyspace.Mask {
	test.Helper()
	at := [3]int{centreSample, centreSample, centreSample}
	at[axis] = int(face) * (samples - 1)
	values := make([]float32, samples*samples*samples)
	values[(at[2]*samples+at[1])*samples+at[0]] = value
	mask, err := emptyspace.Build(cube, [3]int{samples, samples, samples},
		[]emptyspace.Field{{Values: values, Channels: 1}}, options())
	if err != nil {
		test.Fatal(err)
	}
	return mask
}
