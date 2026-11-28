//go:build draw && darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// lockedMean and lockedLargest are how far the gpu may be from the tracer
// looking at a locked sky, away from a patch's edge, 2026-11-26: the
// bounds the whole frame was held to before the round two weather, which
// measured 0.021 to 0.039 mean and 0.19 to 0.22 largest. They are not
// widened. Where a frame has an edge, a patch's or a tower's, float32 and
// float64 put it a fraction of a pixel apart, and a pixel there is wrong by
// as much as the edge is high; that is bounded as lockedStay says.
const lockedMean, lockedLargest = 0.045, 0.30

// lockedEdge is how much brighter, as a share of the frame's brightest,
// the brightest pixel of a pixel's neighbours and itself may be than the
// dimmest for the pixel to be called flat, and no edge. Picked 2026-11-27.
const lockedEdge = 0.05

// lockedBlurMean and lockedBlurLarge are how far, as a share of the
// frame's brightest, the gpu's frame may be from the tracer's once both are
// averaged over lockedBlur pixels either way: structure that is in one and
// not the other, a patch, a tower or a whorl, stays in the average; a
// pixel at the edge of a tower's shadow does not. Measured 2026-11-27 over
// the five views by a per pixel difference of the two draws (the
// tracer's float64 against the gpu's float32): the pixels whose
// neighbours are all within a twentieth of them, a seventh of the frame,
// agree to 4e-7 at worst; the rest, at patch and tower edges and in
// shadow, disagree by up to 0.28 of the brightest, in the thirteen by
// fourteen pixels of one view looked at all where the tracer's pixel is in
// shadow and the gpu's is a little lit. A shadow ray takes ten samples and
// a tower's cliff is hit or missed by each, so one more or less lights or
// darkens a pixel by a tenth of the shadow, which I take to be the cause
// and have not proved; averaged over five pixels the two frames differ by 0.9 per
// cent on the mean and 0.12 at the worst place. With the tower tops billowed
// and the floor gone the worst place is 0.141, measured 2026-11-27, and the
// mean is unchanged. After adding march jitter (an integer shift of each
// ray's samples, merged 2026-11-27), the worst place is 0.042 and the mean
// is 0.005. This world has no lamp, so no shadow march
// runs in it, and the ten shadow samples were not the cause; the view
// march's steps lining up were. The bound is 0.06.
const lockedBlurMean, lockedBlurLarge, lockedBlur = 0.02, 0.06, 2

// split is how two frames differ: over the flat pixels, those whose
// neighbours are all within lockedEdge of them in the tracer's frame, the
// mean and the largest as compare has them, and over all pixels the same
// of the two frames after they are averaged.
type split struct {
	flat, edge          int
	flatMean, flatLarge float64
	blurMean, blurLarge float64
}

// lights is a frame's light, red, green and blue added, as compare has it,
// by row and column.
func lights(from *frame.Frame, width, height int) [][]float64 {
	made := make([][]float64, height)
	for row := range made {
		made[row] = make([]float64, width)
		for column := range made[row] {
			pixel := from.At(column, row)
			made[row][column] = float64(pixel.R + pixel.G + pixel.B)
		}
	}
	return made
}

// boxBlurred is lights averaged over a window lockedBlur pixels either way,
// less at the frame's edge.
func boxBlurred(light [][]float64) [][]float64 {
	made := make([][]float64, len(light))
	for row := range made {
		made[row] = make([]float64, len(light[row]))
		for column := range made[row] {
			sum, count := 0.0, 0
			for y := max(row-lockedBlur, 0); y <= min(row+lockedBlur,
				len(light)-1); y++ {
				for x := max(column-lockedBlur, 0); x <= min(column+
					lockedBlur, len(light[row])-1); x++ {
					sum += light[y][x]
					count++
				}
			}
			made[row][column] = sum / float64(count)
		}
	}
	return made
}

// splitOf is how the gpu's frame differs from the tracer's, flat pixels and
// edge pixels apart, and after averaging.
func splitOf(truth, drawn *frame.Frame, width, height int) split {
	one, other := lights(truth, width, height), lights(drawn, width, height)
	soft, softer := boxBlurred(one), boxBlurred(other)
	brightest := 0.0
	for _, row := range one {
		for _, light := range row {
			brightest = math.Max(brightest, light)
		}
	}
	var found split
	for row := range height {
		for column := range width {
			low, high := math.Inf(1), math.Inf(-1)
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					light := one[min(max(row+dy, 0), height-1)][min(
						max(column+dx, 0), width-1)]
					low, high = math.Min(low, light), math.Max(high, light)
				}
			}
			blurred := math.Abs(soft[row][column]-softer[row][column]) /
				brightest
			found.blurMean += blurred / float64(width*height)
			found.blurLarge = math.Max(found.blurLarge, blurred)
			if high-low > lockedEdge*brightest {
				found.edge++
				continue
			}
			found.flat++
			apart := math.Abs(one[row][column]-other[row][column]) / brightest
			found.flatMean += apart
			found.flatLarge = math.Max(found.flatLarge, apart)
		}
	}
	if found.flat > 0 {
		found.flatMean /= float64(found.flat)
	}
	return found
}

// TestNativeLockedCloudsMatchTheTracer draws a layer of clouds over a
// world locked to its sun, looking straight at its substellar deck, at
// its terminator's broken cloud and at its clear night side, on the gpu
// and on slayer's tracer, and holds the two to each other: the deck, the
// belt, the broken noise and the night's clearing, and the whorls placed
// round the terminator, are in the kernel as they are in the medium. The
// bounds are lockedMean and lockedLargest.
func TestNativeLockedCloudsMatchTheTracer(t *testing.T) {
	toward := maths.V(0.6, 0.5, 0.62).Unit()
	clouds := medium.Clouds{Radius: 1, Low: .1, High: .4, Cover: .24,
		Thick: 5, Seed: 5, Locked: true, Toward: toward}
	first, second := toward.Cross(maths.YAxis).Unit(), maths.Vec{}
	second = toward.Cross(first)
	for name, from := range map[string]maths.Vec{
		"the deck":       toward.Scale(4),
		"the terminator": first.Scale(4),
		"the night":      toward.Scale(-4),
		"a whorl":        clouds.Weather().Storms[2].Centre.Scale(4),
		"between":        first.Add(second).Unit().Scale(4),
	} {
		eye := maths.Subspace{Origin: from,
			Orientation: maths.LookAtUp(from, maths.V(0, 0, 0), maths.Up)}
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: clouds,
			Material: &thing.Material{Glow: maths.Grey(1)}})
		lens := film.Pinhole{Wide: 40 * maths.Degree}
		renderer, err := NewRenderer(32)
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := renderer.Sampled(world, eye, lens, 0, 65, 49, 1)
		if err != nil {
			t.Fatal(err)
		}
		truth, err := slayer.Sampled(world, eye, lens, 0, 65, 49, 1)
		if err != nil {
			t.Fatal(err)
		}
		found := splitOf(truth, drawn, 65, 49)
		t.Logf("%s against the tracer: %+v; over everything %+v", name,
			found, compare(truth, drawn, 65, 49))
		if found.flatMean > lockedMean || found.flatLarge > lockedLargest ||
			found.blurMean > lockedBlurMean ||
			found.blurLarge > lockedBlurLarge {
			t.Errorf("%s drew apart from the tracer: %+v", name, found)
		}
	}
}

// farLeast is how bright the middle of a solid deck, seen from a million
// kilometres off, may be at least: the deck is opaque to a few parts in a
// hundred whatever the slant, so every pixel of it is near one. Measured,
// 2026-11-26: the dimmest pixel is 0.951 now, and was 0 with the old chord.
const farLeast = 0.9

// TestNativeFarCloudsAreAnEvenDeck draws a solid cloud shell seven
// kilometres thick on a world of 6400 kilometres, from a million kilometres
// off, and holds every pixel in the middle of its disc to be bright: the
// shell is opaque, so no pixel of it may be dark. The kernel found the
// shell's chord from the difference of two numbers of 1e18, which a float
// holds to ten kilometres, so some rays found a chord of nothing and some of
// fourteen kilometres, and the deck drew as concentric bands (2026-11-26);
// it now finds the chord from the ray's miss distance. The tracer is no
// reference here: at this distance it draws nothing at all.
func TestNativeFarCloudsAreAnEvenDeck(t *testing.T) {
	const radius, away = 6.4e6, 1.05e9
	clouds := medium.Clouds{Radius: radius, Low: 2e3, High: 9e3, Cover: 1.2,
		Thick: 5e-4, Seed: 5}
	from := maths.V(0, 0, away)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(0, 0, 0), maths.Up)}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: clouds,
		Material: &thing.Material{Glow: maths.Grey(1)}})
	lens := film.Pinhole{Wide: 1.6 * maths.Degree}
	renderer, err := NewRenderer(32)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := renderer.Sampled(world, eye, lens, 0, 65, 65, 1)
	if err != nil {
		t.Fatal(err)
	}
	const middle, reach = 32, 11
	dimmest := 1.0
	for row := middle - reach; row <= middle+reach; row++ {
		for column := middle - reach; column <= middle+reach; column++ {
			if (row-middle)*(row-middle)+(column-middle)*(column-middle) >
				reach*reach {
				continue
			}
			dimmest = min(dimmest, drawn.At(column, row).Light().G)
		}
	}
	t.Logf("the dimmest pixel of the deck's middle is %v", dimmest)
	if dimmest < farLeast {
		t.Errorf("a pixel of the deck is %v, under %v: the shell has bands",
			dimmest, farLeast)
	}
}
