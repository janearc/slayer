//go:build darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/metal"
)

// probeLimb asks the kernel for every branch it is given by its whorl and its
// place in the whorl: where it leaves the trunk, which way it points, how long
// and thick it is, and how many branches its whorl has
const probeLimb = `
kernel void probe_limb(device const Whorled *trees [[buffer(0)]],
                       device const int4 *asked [[buffer(1)]],
                       device float *out [[buffer(2)]],
                       constant uint2 &origin [[buffer(3)]],
                       uint2 at [[thread_position_in_grid]]) {
	int4 which = asked[at.x];
	Limb limb = limb_of(trees[0], which.x, which.y);
	device float *one = out + at.x * 12;
	one[0] = limb.base.x; one[1] = limb.base.y; one[2] = limb.base.z;
	one[3] = limb.dir.x; one[4] = limb.dir.y; one[5] = limb.dir.z;
	one[6] = limb.length; one[7] = limb.radius;
	one[8] = float(limb_count(trees[0], which.x));
}
`

// askedStruct is one whorl and place, for the probe
var askedStruct = structure{"Asked", []field{intField("which", "")}}

// whorledValues is a conifer's parameters, standing at the origin and
// unturned, as the kernel reads a Whorled
func whorledValues(p manufacture.ProperConiferParams) values {
	return values{"origin": {0, 0, 0, 1}, "x": {1, 0, 0}, "y": {0, 1, 0},
		"z": {0, 0, 1},
		"shape": {p.Height, p.TrunkRadius, p.CrownStart,
			p.BranchLength},
		"count": wholes(p.Whorls, p.PerWhorl, 0, 1),
		"seed":  wholes(int(uint32(p.Seed)), int(uint32(p.Seed>>32)))}
}

// whorledBytes is a conifer's parameters as the kernel reads a Whorled
func whorledBytes(p manufacture.ProperConiferParams) []byte {
	var trees packer
	trees.row(whorledStruct, whorledValues(p))
	return trees.bytes
}

// askedBytes is every whorl and place a conifer has, as the probe reads
func askedBytes(p manufacture.ProperConiferParams) ([]byte, [][2]int) {
	var asked packer
	var ids [][2]int
	for whorl := range p.Whorls {
		for k := range manufacture.BranchCount(p, whorl) {
			which := values{"which": wholes(whorl, k)}
			asked.row(askedStruct, which)
			ids = append(ids, [2]int{whorl, k})
		}
	}
	return asked.bytes, ids
}

// near is whether the kernel's float is the go's double to two parts in ten
// thousand, or within as much of it where the number is near zero
func near(got float32, want float64) bool {
	return math.Abs(float64(got)-want) <= 2e-4*math.Max(1, math.Abs(want))
}

// numbersOf is what manufacture says of a branch, in the order the probe
// gives it: base, direction, length, radius, and its whorl's count
func numbersOf(p manufacture.ProperConiferParams, whorl, k int) []float64 {
	b := manufacture.BranchOf(p, whorl, k)
	return []float64{b.Base[0], b.Base[1], b.Base[2], b.Dir[0], b.Dir[1],
		b.Dir[2], b.Length, b.Radius,
		float64(manufacture.BranchCount(p, whorl))}
}

// limbsDiffer says where the kernel's branches are not manufacture's
func limbsDiffer(t *testing.T, kernel *metal.Kernel,
	p manufacture.ProperConiferParams) {
	t.Helper()
	asked, ids := askedBytes(p)
	got, err := kernel.Grid(len(ids), 1, 1, 12,
		[][]byte{whorledBytes(p), asked})
	if err != nil {
		t.Fatal(err)
	}
	for index, id := range ids {
		for place, want := range numbersOf(p, id[0], id[1]) {
			if seen := got[index*12+place]; !near(seen, want) {
				t.Errorf("seed %x %v #%d: %v, not %v",
					p.Seed, id, place, seen, want)
			}
		}
	}
}

// the kernel works out every branch of a conifer as manufacture does:
// where it leaves, which way it points, its length and radius, and how
// many branches its whorl has. the seeds include one with high bits, and
// one conifer is tall, its upper whorls turned through a hundred radians.
func TestTheKernelsBranchesAreManufacturesBranches(t *testing.T) {
	kernel, err := metal.Open(source()+probeLimb, "probe_limb")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	for _, seed := range []uint64{0, 1, 7, 0xDEADBEEFCAFEF00D} {
		p := manufacture.DefaultProperConiferParams()
		p.Seed = seed
		limbsDiffer(t, kernel, p)
	}
	tall := manufacture.DefaultProperConiferParams()
	tall.Whorls, tall.PerWhorl, tall.Height, tall.Seed = 48, 9, 24, 5
	limbsDiffer(t, kernel, tall)
}

// wholeCount is the kernel's branch count, in whole numbers only: the
// same as manufacture's BranchCount, which is in floats and rounds
func wholeCount(whorls, per, whorl int) int {
	top := per * (40*whorls - 11*(2*whorl+1))
	return max(3, (2*top+40*whorls)/(80*whorls))
}

// half is whether a whorl's branch count falls exactly on a half, where
// rounding has to choose: the float calculation and the exact one may not
// choose the same way
func half(whorls, per, whorl int) bool {
	top := per * (40*whorls - 11*(2*whorl+1))
	return (2*top)%(80*whorls) == 40*whorls
}

// the whole-number count is manufacture's, for every conifer of up to 64 whorls
// and 16 branches in the lowest, except where the count is exactly a half:
// there go's floats choose by their own rounding, and the two can differ. that
// is the only way they differ.
func TestTheWholeNumberCountIsManufacturesCountButForHalves(t *testing.T) {
	halves, differ := 0, 0
	for whorls := 1; whorls <= 64; whorls++ {
		for per := 1; per <= 16; per++ {
			p := manufacture.DefaultProperConiferParams()
			p.Whorls, p.PerWhorl = whorls, per
			for whorl := range whorls {
				got := wholeCount(whorls, per, whorl)
				want := manufacture.BranchCount(p, whorl)
				tie := half(whorls, per, whorl)
				if tie {
					halves++
				}
				if got != want {
					differ++
				}
				if got != want && !tie {
					at := [3]int{whorls, per, whorl}
					t.Errorf("%v: %d not %d", at, got, want)
				}
			}
		}
	}
	t.Logf("%d counts fall on a half, and %d of them differ", halves,
		differ)
}
