//go:build draw && darwin

package gpu

import (
	"bytes"
	"math"
	"runtime"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// stripedWidth and stripedHeight are the hall's picture in pixels, and
// stripedSide its samples a side: three, so that a stripe of 64 rows of
// samples ends partway through a row of pixels
const (
	stripedWidth, stripedHeight, stripedSide = 96, 64, 3
)

// oneBand is a stripe of one band of rays across the hall's picture in
// samples: three stripes for its 192 rows
const oneBand = rows * stripedWidth * stripedSide * rayStride

// serialRays is every sample's ray through a lens, made one at a time
// and packed word by word, as the scene made them before workers shared
// the rows
func serialRays(lensed *through) []byte {
	var packed packer
	for y := range lensed.height {
		for x := range lensed.width {
			ray := film.Look(lensed.eye, lensed.lens, x, y, lensed.width,
				lensed.height, maths.Band{})
			for _, part := range [rayWords]float64{ray.At.X, ray.At.Y,
				ray.At.Z, ray.Dir.X, ray.Dir.Y, ray.Dir.Z} {
				packed.word(math.Float32bits(float32(part)))
			}
		}
	}
	return packed.bytes
}

// the rays the workers make between them are the bytes one makes alone,
// with one worker and with many
func TestRaysMadeByManyWorkersAreTheRaysOneMakes(t *testing.T) {
	_, eye := hall(t)
	lensed := &through{eye: eye, lens: thinLens(t),
		width: stripedWidth * stripedSide, height: stripedHeight * stripedSide}
	want := serialRays(lensed)
	for _, workers := range []int{1, 8} {
		was := runtime.GOMAXPROCS(workers)
		got := lensed.rays(0, lensed.height)
		runtime.GOMAXPROCS(was)
		if !bytes.Equal(got, want) {
			t.Fatalf("%d workers made other rays than one at a time", workers)
		}
	}
	part := lensed.rays(rows, 2*rows)
	from := rows * lensed.width * rayStride
	if !bytes.Equal(part, want[from:from+len(part)]) {
		t.Fatal("a stripe's rays are not the picture's rays of its rows")
	}
}

// the hall drawn through a lens a stripe at a time is the hall drawn
// whole, pixel for pixel: in three stripes of one band each, ending
// partway through rows of pixels, and in a stripe of two bands and a
// short last stripe of one
func TestAPictureDrawnInStripesIsThePictureDrawnWhole(t *testing.T) {
	world, eye := hall(t)
	glass := thinLens(t)
	baking := bakeSide
	bakeSide = noBake
	defer restoreBakeSide(baking)
	whole, _, err := sampledBackend(world, nil, eye, glass, noTime,
		stripedWidth, stripedHeight, stripedSide, softwareTraversal)
	if err != nil {
		t.Fatal(err)
	}
	defer restoreStripes(stripeBytes)
	for _, bands := range []int{oneStripeBand, twoStripeBands} {
		stripeBytes = bands * oneBand
		lensed := &through{width: stripedWidth * stripedSide,
			height: stripedHeight * stripedSide}
		if lensed.whole() || lensed.stripe() != bands*rows {
			t.Fatalf("the hall's rays should take stripes of %d bands", bands)
		}
		striped, _, err := sampledBackend(world, nil, eye, glass, noTime,
			stripedWidth, stripedHeight, stripedSide, softwareTraversal)
		if err != nil {
			t.Fatal(err)
		}
		samePictures(t, striped, whole)
	}
}

// noBake turns the bake off, as the lens test does; noTime is the
// picture's moment, the start; oneStripeBand and twoStripeBands are the
// stripes' sizes, in bands
const (
	noBake         = 0
	noTime         = 0
	oneStripeBand  = 1
	twoStripeBands = 2
)

// samePictures fails at the first pixel where two pictures differ
func samePictures(t *testing.T, got, want *frame.Frame) {
	t.Helper()
	for y := range want.H {
		for x := range want.W {
			if seen, wanted := got.At(x, y), want.At(x, y); seen != wanted {
				t.Fatalf("pixel %d, %d: striped %+v, whole %+v", x, y, seen, wanted)
			}
		}
	}
}

// restoreBakeSide puts the bake's side back after a test
func restoreBakeSide(side int) { bakeSide = side }

// restoreStripes puts the stripes' size back after a test
func restoreStripes(size int) { stripeBytes = size }

// the rays are the input the kernel binds as its rays
func TestTheRaysSitWhereTheKernelBindsThem(t *testing.T) {
	encoded := newScene()
	encoded.rays.bytes = []byte("rays")
	if got := encoded.inputs()[raysAt]; !bytes.Equal(got, encoded.rays.bytes) {
		t.Fatalf("input %d is %q, not the rays", raysAt, got)
	}
}
