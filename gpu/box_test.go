//go:build darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/metal"
)

// probe asks the kernel's own box_hit, one case a pixel: a case is the
// ray's start, its way (and how far), and the box's two corners
const probe = `
kernel void probe(device const float4 *cases [[buffer(0)]],
                  device float *out [[buffer(1)]],
                  constant uint2 &origin [[buffer(2)]],
                  uint2 at [[thread_position_in_grid]]) {
	device const float4 *one = cases + at.x * 4;
	Node box = {one[2], one[3], int4(0)};
	out[at.x] = box_hit(one[0].xyz, one[1].xyz, box, one[1].w) ? 1 : 0;
}
`

// boxCase is a ray and a box, every number whole, many of the ray's parts zero
// and many of its starts on a box's face: the cases the box test once missed
type boxCase struct {
	start, way maths.Vec
	box        maths.Box
}

// whole is a whole number from low to high by a hash
func whole(seed uint64, id int64, low, high int) float64 {
	span := high - low + 1
	return float64(low + int(noise.Hash(seed, id)*float64(span))%span)
}

// boxCases is count cases, the same every time
func boxCases(count int) []boxCase {
	cases := make([]boxCase, 0, count)
	for index := range int64(count) {
		low := maths.V(whole(1, index, -3, 2), whole(2, index, -3, 2),
			whole(3, index, -3, 2))
		size := maths.V(whole(4, index, 1, 3), whole(5, index, 1, 3),
			whole(6, index, 1, 3))
		start := maths.V(whole(7, index, -4, 4), whole(8, index, -4, 4),
			whole(9, index, -4, 4))
		way := maths.V(whole(10, index, -1, 1), whole(11, index, -1, 1),
			whole(12, index, -1, 1))
		if way == (maths.Vec{}) {
			way = maths.V(0, 0, 1)
		}
		cases = append(cases, boxCase{start: start, way: way,
			box: maths.Box{Min: low, Max: low.Add(size)}})
	}
	return cases
}

// packed is the cases as the probe reads them
func packed(cases []boxCase) []byte {
	var bytes []byte
	put := func(values ...float64) {
		for _, value := range values {
			bytes = binary.LittleEndian.AppendUint32(bytes,
				math.Float32bits(float32(value)))
		}
	}
	for _, one := range cases {
		put(one.start.X, one.start.Y, one.start.Z, 0)
		put(one.way.X, one.way.Y, one.way.Z, 100)
		put(one.box.Min.X, one.box.Min.Y, one.box.Min.Z, 0)
		put(one.box.Max.X, one.box.Max.Y, one.box.Max.Z, 0)
	}
	return bytes
}

// the kernel's box test is maths.Box's, on whole numbers where a ray runs
// along a face or starts on one: without the slab's guard, 864 of these
// twenty thousand differ
func TestTheBoxTestIsMathsBoxs(t *testing.T) {
	kernel, err := metal.Open(source()+probe, "probe")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	cases := boxCases(20000)
	got, err := kernel.Grid(len(cases), 1, 1, 1, [][]byte{packed(cases)})
	if err != nil {
		t.Fatal(err)
	}
	missed := 0
	for index, one := range cases {
		ray := maths.Ray{At: one.start, Dir: one.way}
		_, _, want := one.box.Hit(ray, 0, 100)
		if (got[index] == 1) != want {
			missed++
		}
	}
	if missed > 0 {
		t.Errorf("%d of %d boxes differ from maths.Box's", missed,
			len(cases))
	}
}
