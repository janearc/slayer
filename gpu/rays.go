package gpu

import (
	"encoding/binary"
	"math"
	"runtime"
	"sync"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// rayWords is how many numbers a ray takes: its start and its way
const rayWords = 6

// wordBytes is a number's bytes, as the kernel reads it: a float32
const wordBytes = 4

// rayStride is a ray's bytes
const rayStride = rayWords * wordBytes

// raysAt is the rays' place among the scene's inputs, as inputs gives
// them and as trace.metal binds them: buffer(13)
const raysAt = 13

// firstRow is the picture's first row of samples
const firstRow = 0

// askOnly asks GOMAXPROCS how many workers are allowed without changing it
const askOnly = 0

// stripeBytes is the most a stripe of the picture's rays may take. Metal
// refuses an input over two gigabytes, and a lens's rays for a picture
// 4096 by 3002 at sixteen a pixel are 4.7; so a picture whose rays do
// not fit is drawn a stripe at a time, each with its own. One that fits
// is drawn whole, as before. A variable, so a test can make it small.
var stripeBytes = 1 << 30

// through is how the cpu makes rays through a lens that is not a
// pinhole: the eye, the lens, and the picture's width and height in
// samples, not pixels.
type through struct {
	eye           maths.Subspace
	lens          film.Sensor
	width, height int
	// side is the samples along a pixel's side where they are spread
	// over their pixels, and ramp the width of the slope of the filter
	// they are spread under; nought where each starts at its own middle
	side int
	ramp float64
	// exact holds a total sample count; width then counts samples across,
	// while pictureWidth and pictureHeight are the finished image size.
	exact, pictureWidth, pictureHeight int
}

// lensRays readies the rays through a lens: every sample's ray now when
// they fit in one stripe, or none until each stripe is drawn.
func (encoded *scene) lensRays(eye maths.Subspace, lens film.Sensor,
	width, height int) {
	encoded.through = &through{eye: eye, lens: lens, width: width,
		height: height}
	encoded.looked = true
	if encoded.through.whole() {
		encoded.rays.bytes = encoded.through.rays(firstRow, height)
	}
}

// whole is whether every sample's ray fits in one stripe.
func (lensed *through) whole() bool {
	return lensed.width*lensed.height*rayStride <= stripeBytes
}

// rays is the rays of the rows of samples from first up to last, not
// last, row by row, each made by film.Look as slayer's tracer makes it:
// so whatever a lens does, its aperture, its focus, its distortion, the
// kernel's rays are the tracer's own.
//
// The rows are shared among the Go workers allowed. Each writes only its
// own rows, in their place, so the bytes are the ones one worker makes.
func (lensed *through) rays(first, last int) []byte {
	made := make([]byte, (last-first)*lensed.width*rayStride)
	jobs := make(chan int, last-first)
	for y := first; y < last; y++ {
		jobs <- y
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(runtime.GOMAXPROCS(askOnly), last-first) {
		workers.Add(1)
		go lensed.work(jobs, made, first, &workers)
	}
	workers.Wait()
	return made
}

// work makes each row it is handed into its place in made, whose first
// row is the picture's row first.
func (lensed *through) work(jobs <-chan int, made []byte, first int,
	workers *sync.WaitGroup) {
	defer workers.Done()
	for y := range jobs {
		at := (y - first) * lensed.width * rayStride
		for x := range lensed.width {
			putRay(made[at+x*rayStride:], lensed.look(x, y))
		}
	}
}

// look is a sample's ray: through its middle, or, where the samples are
// spread, through the place film.Within gives it in its pixel.
func (lensed *through) look(x, y int) maths.Ray {
	if lensed.exact > 0 {
		column, index := x/lensed.exact, x%lensed.exact
		across, down := film.Place(index, lensed.exact, lensed.ramp)
		return film.LookThrough(lensed.eye, lensed.lens, column, y, across, down,
			lensed.pictureWidth, lensed.pictureHeight, maths.Band{})
	}
	if lensed.side == wholeZero {
		return film.Look(lensed.eye, lensed.lens, x, y, lensed.width,
			lensed.height, maths.Band{})
	}
	across, down := film.Within(x, y, lensed.side, lensed.ramp)
	return film.LookThrough(lensed.eye, lensed.lens, x, y, across, down,
		lensed.width, lensed.height, maths.Band{})
}

// putRay writes a ray's start and its way, six float32s, as the scene's
// packer writes words.
func putRay(into []byte, ray maths.Ray) {
	parts := [rayWords]float64{ray.At.X, ray.At.Y, ray.At.Z,
		ray.Dir.X, ray.Dir.Y, ray.Dir.Z}
	for i, part := range parts {
		bits := math.Float32bits(float32(part))
		binary.LittleEndian.PutUint32(into[i*wordBytes:], bits)
	}
}
