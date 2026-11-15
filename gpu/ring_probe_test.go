//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/planet"
)

// probeRing asks the kernel's own ring_density, one point a thread, of the
// card the third buffer holds; probeGlow asks ring_shine's red.
const probeRing = `
kernel void probe_ring(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = ring_density(card, float3(points[id]));
}

kernel void probe_flat(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = 0;
	if (id % 2 == 0) {
		out[id] = ring_flat(card, float3(points[id]),
		                    float3(points[id + 1])) ? 1 : 0;
	}
}

kernel void probe_glow(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = ring_shine(card, float3(points[id])).x;
}
`

// cardBytes is a card as the bytes the probe reads it from.
func cardBytes(words []uint32) []byte {
	var bytes []byte
	for _, word := range words {
		bytes = binary.LittleEndian.AppendUint32(bytes, word)
	}
	return bytes
}

// ringPoints is count points across a ring's plane, from its inner radius to
// its outer, by a hash, as float32 as the kernel is given them: all the same
// every time.
func ringPoints(count int, inner, outer float64) [][3]float32 {
	points := make([][3]float32, count)
	for at := range points {
		reach := inner + (outer-inner)*noise.Hash(21, int64(at))
		turn := 2 * math.Pi * noise.Hash(22, int64(at))
		points[at] = [3]float32{float32(reach * math.Cos(turn)), 0,
			float32(reach * math.Sin(turn))}
	}
	return points
}

// probed asks a probe kernel at every point of a ring's card, or skips the
// test when there is no gpu.
func probed(t *testing.T, kernelName string, words []uint32,
	points [][3]float32) []float32 {
	t.Helper()
	kernel, err := metal.Open(source()+probeRing, kernelName)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	got, err := kernel.Points(points, cardBytes(words))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// ringApart is how many of the points of a ring's plane the kernel's density
// is further than a bound from the ring slab's own, and the furthest it is
// at the rest.
func ringApart(t *testing.T, ring planet.Ring, bound float64) (strays int,
	largest float64) {
	t.Helper()
	slab, err := planet.NewRingSlab(ring, ringThickness)
	if err != nil {
		t.Fatal(err)
	}
	inner, outer := ring.Span()
	points := ringPoints(ringProbed, inner, outer)
	words, err := ringCardOf(ring, slab.Half())
	if err != nil {
		t.Fatal(err)
	}
	got := probed(t, "probe_ring", words, points)
	for at, point := range points {
		want := slab.Density(maths.V(float64(point[0]), 0,
			float64(point[2])))
		apart := math.Abs(float64(got[at]) - want)
		if !(apart <= bound) {
			strays++
		} else if apart > largest {
			largest = apart
		}
	}
	return strays, largest
}

// ringProbed is how many points of a ring's plane the probes ask.
const ringProbed = 20000

// ringCase is a ring to probe, the bound its share is held to, and how many
// of the points may be further than that: none, but for a speckle's seams.
type ringCase struct {
	name   string
	ring   planet.Ring
	strays int
}

// the terms of a ring, each alone, on their own: the kernel's density is the
// ring's at every point of the plane within a bound of 1e-3, measured at
// 6e-5 at most for every term but speckle's. there a seam, a radius at
// which its cells' count steps by one, is a whole other hash, and a point
// within float32's reach of one takes the other side: 33 points in 20000
// measured on 2026-11-10, and no more than 100 allowed. the rest of
// speckle's points are within 3.6e-4.
func TestTheKernelsRingShareIsPlanets(t *testing.T) {
	eye := ringEyeAt(30)
	cases := []ringCase{
		{"plain", plainRing(t), 0},
		{"ringlets", ringleted(t, plainRing(t), 0.85, 3e6), 0},
		{"measured", ringSeen(t, measuredRing(t, plainRing(t), 1.2e7, 3.4e7),
			eye), 0},
		{"moonlets", moonleted(t, plainRing(t)), 0},
	}
	for _, pattern := range planet.Patterns() {
		strays := 0
		if pattern == "speckle" {
			strays = ringProbed / 200
		}
		cases = append(cases, ringCase{"grains " + pattern,
			ringSeen(t, grained(t, plainRing(t), pattern), eye), strays})
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			strays, largest := ringApart(t, each.ring, ringShareBound)
			t.Logf("%d strays, the rest within %g", strays, largest)
			if strays > each.strays {
				t.Errorf("%d points are over %g from the ring's share, "+
					"want no more than %d", strays, ringShareBound,
					each.strays)
			}
		})
	}
}

// ringShareBound is how far the kernel's share may be from the ring's: float32
// against float64 through five layers of noise, six times what was measured.
const ringShareBound = 1e-3

// the glow's shine, as the kernel reads it, is the ring's own at every point
// of the plane, to a part in ten thousand of the light: the x-rays on the
// ring, the share they stop, and what its phosphors give back.
func TestTheKernelsRingGlowIsPlanets(t *testing.T) {
	ring := glowing(t, ringleted(t, plainRing(t), 0.85, 3e6))
	volume := glowSlab(t, ring)
	slab, ok := volume.Medium.(planet.RingGlow)
	if !ok {
		t.Fatal("the glow volume's medium is not a ring's glow")
	}
	inner, outer := ring.Span()
	points := ringPoints(ringProbed, inner, outer)
	words, err := glowCardOf(slab)
	if err != nil {
		t.Fatal(err)
	}
	got := probed(t, "probe_glow", words, points)
	worst, brightest := 0.0, 0.0
	for at, point := range points {
		want := slab.Shine().At(maths.V(float64(point[0]), 0,
			float64(point[2]))).R
		brightest = math.Max(brightest, want)
		worst = math.Max(worst, math.Abs(float64(got[at])-want))
	}
	t.Logf("the largest difference %g of a brightest %g", worst, brightest)
	if !(brightest > 0) || !(worst <= 1e-4*brightest) {
		t.Errorf("the kernel's glow is %g from the ring's, of a brightest "+
			"%g, want within a part in ten thousand", worst, brightest)
	}
}

// slabCrossings is the stretches inside the ring's slab of rays from an eye
// to a grid of points across the ring's plane, start and end alternating,
// in the ring's own space, for the rays that cross the slab at all.
func slabCrossings(eye maths.Vec, half float64) [][3]float32 {
	var stretches [][3]float32
	const across = 64
	outer := float64(ringOuter)
	for row := range across {
		for column := range across {
			target := maths.V(outer*(2*(float64(column)+0.5)/across-1), 0,
				outer*(2*(float64(row)+0.5)/across-1))
			way := target.Sub(eye).Unit()
			enter, leave := (-half-eye.Y)/way.Y, (half-eye.Y)/way.Y
			if enter > leave {
				enter, leave = leave, enter
			}
			start, end := eye.Add(way.Scale(enter)), eye.Add(way.Scale(leave))
			if start.Len() > outer*2 {
				continue
			}
			stretches = append(stretches,
				[3]float32{float32(start.X), float32(start.Y), float32(start.Z)},
				[3]float32{float32(end.X), float32(end.Y), float32(end.Z)})
		}
	}
	return stretches
}

// flatShare is how many of the rays that cross a ring's slab from an eye so
// many degrees over its plane, and land on its annulus, the kernel takes as
// one density, of those that land on it.
func flatShare(t *testing.T, ring planet.Ring, degrees float64) float64 {
	t.Helper()
	words, err := ringCardOf(ring, ringThickness/2)
	if err != nil {
		t.Fatal(err)
	}
	stretches := slabCrossings(ringEyeAt(degrees), ringThickness/2)
	got := probed(t, "probe_flat", words, stretches)
	inner, outer := ring.Span()
	flat, landed := 0, 0
	for at := 0; at < len(stretches); at += 2 {
		middle := maths.V(float64(stretches[at][0]+stretches[at+1][0])/2, 0,
			float64(stretches[at][2]+stretches[at+1][2])/2)
		if reach := middle.Len(); reach < inner || reach >= outer {
			continue
		}
		landed++
		if got[at] == 1 {
			flat++
		}
	}
	t.Logf("%d of %d rays that land on the ring are taken as one density",
		flat, landed)
	return float64(flat) / float64(landed)
}

// a plain ring seen steeply has one density along nearly every ray's crossing
// of its slab, and so the kernel reads it once; seen nearly edge on, a ray
// crosses the slab over a stretch as long as the ring is wide, and most
// cross a rim, where the density steps, and are marched.
func TestTheKernelTakesASteepCrossingAsOneDensity(t *testing.T) {
	if got := flatShare(t, plainRing(t), 60); !(got >= 0.95) {
		t.Errorf("seen from 60 degrees, %g of the rays are flat, want 0.95",
			got)
	}
	if got := flatShare(t, plainRing(t), 1); !(got <= 0.5) {
		t.Errorf("seen from 1 degree, %g of the rays are flat, want at "+
			"most half", got)
	}
}

// a ring with ringlets, a thousandth of whose finest layer is 11 metres, has
// no steep crossing of a slab 1000 km thick that stays inside a thousandth of
// it; a ring with speckle's seams has none at all.
func TestAFineRingIsNeverTakenAsOneDensity(t *testing.T) {
	fine := ringleted(t, plainRing(t), 0.85, 3e6)
	if got := flatShare(t, fine, 89); got != 0 {
		t.Errorf("a ring with ringlets is flat for %g of its rays", got)
	}
	speckled := ringSeen(t, grained(t, plainRing(t), "speckle"), ringEyeAt(60))
	if got := flatShare(t, speckled, 89); got != 0 {
		t.Errorf("a ring with speckle is flat for %g of its rays", got)
	}
}
