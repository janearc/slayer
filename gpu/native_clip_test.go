//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
)

// clipProbe runs native_clip on one segment a thread and writes the
// start, or the stop, or minus one when the segment is clipped away.
const clipProbe = `
kernel void clip_start(device const packed_float3 *from [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       constant float4 *given [[buffer(2)]],
                       uint k [[thread_position_in_grid]]) {
	float start, stop;
	bool met = native_clip(given[0], from[k], given[1].xyz, given[2].x,
	                       given[1].w, start, stop);
	out[k] = met ? start : -1;
}
kernel void clip_stop(device const packed_float3 *from [[buffer(0)]],
                      device float *out [[buffer(1)]],
                      constant float4 *given [[buffer(2)]],
                      uint k [[thread_position_in_grid]]) {
	float start, stop;
	bool met = native_clip(given[0], from[k], given[1].xyz, given[2].x,
	                       given[1].w, start, stop);
	out[k] = met ? stop : -1;
}
`

// clipCase is a segment and what the clip owes it, found by hand: the
// sphere's crossings in float64, or that there are none.
type clipCase struct {
	name         string
	from, way    maths.Vec
	near, far    float64
	enters, exit float64
	missed       bool
}

// clipCentre and clipRadius are the sphere the cases are clipped to.
var clipCentre = maths.V(10, -20, 30)

const clipRadius = 3000

// crossings is where the line through from along a unit way enters and
// leaves the sphere, by float64 arithmetic, and whether it misses it.
func crossings(from, way maths.Vec) (float64, float64, bool) {
	to := clipCentre.Sub(from)
	along := to.Dot(way)
	across2 := to.Dot(to) - along*along
	if across2 > clipRadius*clipRadius {
		return 0, 0, true
	}
	half := math.Sqrt(clipRadius*clipRadius - across2)
	return along - half, along + half, false
}

// segment is a case from a place, aimed at a point, with a near and far.
func segment(name string, from, aim maths.Vec, near, far float64) clipCase {
	way := aim.Sub(from).Unit()
	enters, exit, missed := crossings(from, way)
	// a line through the sphere still misses it when the segment ends
	// before the sphere or starts after it
	missed = missed || exit <= math.Max(0, near) || enters >= far
	return clipCase{name: name, from: from, way: way, near: near, far: far,
		enters: enters, exit: exit, missed: missed}
}

// clipCases are segments about the sphere, among them the shape of the one
// that never came back: from 2.5e8 m off along the slant of the primary
// ray, back toward the scene, for 2.564e8 m.
func clipCases() []clipCase {
	slant := maths.V(0.9187, 0.3204, -0.2312).Unit()
	far := slant.Scale(2.5e8)
	beside := clipCentre.Add(maths.V(0, 0, 2*clipRadius))
	return []clipCase{
		segment("the far planet's shadow ray", far, clipCentre, 0, 2.564e8),
		segment("from inside", clipCentre.Add(maths.V(100, 50, -70)),
			clipCentre.Add(maths.V(100, 1e6, -70)), 0, 1e30),
		segment("too short to arrive", far, clipCentre, 0, 1e5),
		segment("pointing away", far, far.Add(slant), 0, 2.564e8),
		segment("beside it", far, beside, 0, 2.564e8),
		segment("starting past it", far, clipCentre, 2.6e8, 2.7e8),
		segment("unbounded length", far, clipCentre, 0, 1e30),
	}
}

// given is the probe's params for a case: the bounds, then the way and
// far, then the near.
func (one clipCase) given() []byte {
	var bytes []byte
	for _, number := range []float64{clipCentre.X, clipCentre.Y, clipCentre.Z,
		clipRadius, one.way.X, one.way.Y, one.way.Z, one.far, one.near, 0, 0, 0} {
		bytes = binary.LittleEndian.AppendUint32(bytes,
			math.Float32bits(float32(number)))
	}
	return bytes
}

// answer is what a probe kernel in a source says of a case.
func (one clipCase) answer(t *testing.T, source, probe string) float64 {
	t.Helper()
	kernel, err := metal.Open("#define SLAYER_NATIVE_SHADOWS 1\n"+source+clipProbe, probe)
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	got, err := kernel.Points([][3]float32{{float32(one.from.X),
		float32(one.from.Y), float32(one.from.Z)}}, one.given())
	if err != nil {
		t.Fatal(err)
	}
	return float64(got[0])
}

// the clip keeps a segment's whole crossing of the bounds, less no more
// than its margin at each end, ends the segment where it is inside, and
// clips away a segment that does not reach the bounds.
func TestNativeClipKeepsTheCrossingAndDropsWhatMisses(t *testing.T) {
	source := sourceWith(1, 1)
	for _, one := range clipCases() {
		t.Run(one.name, func(t *testing.T) {
			start := one.answer(t, source, "clip_start")
			stop := one.answer(t, source, "clip_stop")
			if one.missed {
				if start != -1 {
					t.Fatalf("a segment that misses the bounds starts at %v; want it clipped away", start)
				}
				return
			}
			reach := clipCentre.Sub(one.from).Len()
			margin := clipRadius*clipShare + reach*clipRounding
			// a float32 holds a distance of 2.5e8 to within 16 m
			slack := 64.0
			if one.enters < 0 {
				if start != 0 {
					t.Errorf("a segment from inside starts at %v; want zero", start)
				}
			} else if drop := one.enters - start; !(drop >= 0 && drop <= 2*margin+slack) {
				t.Errorf("start %v is %v m before the entry %v; want from zero to %v",
					start, drop, one.enters, 2*margin+slack)
			}
			want := math.Min(one.far, one.exit+margin)
			if !(math.Abs(stop-want) <= margin+slack) {
				t.Errorf("stop %v; want %v within %v", stop, want, margin+slack)
			}
		})
	}
}

// a clip that starts at the bounds' middle, after the first surface of
// anything inside, is told from the right one by the same measure.
func TestNativeClipMeasureCatchesAWrongStart(t *testing.T) {
	good := sourceWith(1, 1)
	wrong := strings.Replace(good,
		"start = max(max(beam_zero, near), along - half_chord);",
		"start = max(max(beam_zero, near), along);", 1)
	if wrong == good {
		t.Fatal("the clip's start line is not in the kernel")
	}
	one := clipCases()[0]
	right := one.enters - one.answer(t, good, "clip_start")
	late := one.enters - one.answer(t, wrong, "clip_start")
	if !(right >= 0 && right < clipRadius/2) || !(late < 0) || !(-late > clipRadius/2) {
		t.Fatalf("the right start is %v m before the entry and the wrong one %v; want the measure to tell them apart",
			right, late)
	}
}

// headerBounds is the sphere the encoder wrote in the header, as the
// kernel reads it: the centre and then the radius.
func headerBounds(encoded *scene) (maths.Vec, float64) {
	at := fieldOffset(headerStruct, "bounds")
	var number [4]float64
	for k := range number {
		bits := binary.LittleEndian.Uint32(encoded.header.bytes[at+4*k:])
		number[k] = float64(math.Float32frombits(bits))
	}
	return maths.V(number[0], number[1], number[2]), number[3]
}

// the header's bounds are a sphere that holds every corner of every
// face of a standing mesh, and not a much larger one: the corners are
// found here from the members' own placements.
func TestHeaderBoundsHoldEveryStandingCorner(t *testing.T) {
	group, _ := groupFixture(t)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group, Motion: at(5, 2, -3),
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	from := maths.V(0, 4, 16)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
	encoded, err := encodeGrown(world, nil, eye, film.Pinhole{}, 0, 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	centre, radius := headerBounds(encoded)
	corners := []maths.Vec{maths.V(-1, -1, 0), maths.V(1, -1, 0), maths.V(0, 1, 0)}
	round := empty()
	for i := range 8 {
		space := maths.Subspace{
			Origin:      maths.V(float64(i%4)*2, 0, float64(i/4)*-2).Add(maths.V(5, 2, -3)),
			Orientation: maths.About(maths.YAxis, float64(i)*0.2),
			Scale:       0.5 + float64(i%2)*0.3}
		for _, corner := range corners {
			world := space.Out(corner)
			round = round.fit(world)
			if far := world.Sub(centre).Len(); !(far <= radius*1.0001) {
				t.Errorf("corner %v of member %d is %v m from the bounds' centre %v; want within the radius %v",
					world, i, far, centre, radius)
			}
		}
	}
	half := round.high.Sub(round.low).Len() / 2
	if !(radius <= half*1.05) {
		t.Errorf("the bounds' radius %v is more than 5%% over the half diagonal %v of the corners' box", radius, half)
	}
}
