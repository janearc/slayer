package trace

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// errNoWorld reports a nil world at a public rendering entry point.
var errNoWorld = errors.New("trace: there is no world to draw")

// mostPixels limits total ray samples to 1 << 28 before allocation. A frame of
// that many 20-byte pixels would occupy roughly 5 GiB.
const mostPixels = 1 << 28

// view holds a validated render request: scene, camera, lens, time, dimensions,
// and samples per pixel axis. Construct it through newView.
type view struct {
	world         *thing.World
	eye           maths.Subspace
	lens          film.Sensor
	at            time.Duration
	width, height int
	rays          int
	inside        [mostHeld]held
	holding       int
}

// InitialMedium is one explicitly supplied clear medium the camera starts
// inside. Index corresponds to a clear surface material's Index (or Dense
// default) and must be finite in [1e-9, 1e9], the renderer's supported safe
// envelope. Rank is that material's Rank.
//
// At most MaxInitialMedia entries may be supplied. The slice passed to
// RenderWithInitialMedia or SampledWithInitialMedia is ordered by entry: among
// equal ranks, its last member is active, as it is for media crossed during
// tracing.
//
// The renderer does not infer this state from geometry. The caller is
// responsible for supplying the media that actually contain the camera and
// for matching their Index and Rank to the boundary materials. Two clear
// things with the same Index and Rank are one medium in the tracer's current
// surface model.
type InitialMedium struct {
	// Index is the refractive index of this explicitly declared medium.
	Index float64
	// Rank determines which overlapping clear medium is active.
	Rank int
	// ClearMedium supplies this medium's optional homogeneous RGB
	// extinction and scene-to-metre conversion. Its name and properties
	// must match any boundary material carrying the same named identity.
	ClearMedium *thing.ClearMedium
}

// MaxInitialMedia is the most media an initial camera state may contain. The
// tracer uses the same bound for media accumulated while crossing surfaces.
const MaxInitialMedia = mostHeld

// initialMedia validates and copies a caller's ordered start state into the
// same fixed stack used as a ray crosses clear surfaces.
func initialMedia(media []InitialMedium) ([mostHeld]held, int, error) {
	var inside [mostHeld]held
	if len(media) > MaxInitialMedia {
		return inside, 0, fmt.Errorf("trace: an initial medium stack holds at most %d entries, got %d",
			MaxInitialMedia, len(media))
	}
	for i, each := range media {
		if !finite(each.Index) || each.Index < 1e-9 || each.Index > 1e9 {
			return inside, 0, fmt.Errorf("trace: initial medium %d index must be finite and between 1e-9 and 1e9, got %g",
				i, each.Index)
		}
		current := held{index: each.Index, rank: each.Rank,
			clearMedium: each.ClearMedium}
		for prior := 0; prior < i; prior++ {
			if current.clearMedium.Defined() &&
				inside[prior].clearMedium.Name() == current.clearMedium.Name() {
				return inside, 0, fmt.Errorf("trace: initial clear medium ID %q occurs more than once", current.clearMedium.Name())
			}
			if inside[prior].index == current.index &&
				inside[prior].rank == current.rank {
				if inside[prior].clearMedium.Name() == current.clearMedium.Name() {
					return inside, 0, fmt.Errorf("trace: initial medium %d duplicates medium %d (index %g, rank %d)",
						i, prior, each.Index, each.Rank)
				}
				return inside, 0, fmt.Errorf("trace: initial media at index %g rank %d have ambiguous identities", each.Index, each.Rank)
			}
		}
		inside[i] = current
	}
	return inside, len(media), nil
}

// withInitialMedia adds a validated, caller-owned start state to a view.
// An empty list preserves the view's original all-air zero state.
func (seen view) withInitialMedia(media []InitialMedium) (view, error) {
	if len(media) > 0 {
		inside, holding, err := initialMedia(media)
		if err != nil {
			return view{}, err
		}
		seen.inside, seen.holding = inside, holding
	}
	if err := ValidateClearState(seen.world, media); err != nil {
		return view{}, err
	}
	if err := validateOceanInitialMedia(seen.world, seen.eye, media,
		seen.at); err != nil {
		return view{}, err
	}
	return seen, nil
}

// newView validates a render request. It requires a world, finite camera
// position and scale, positive dimensions and sampling, and a total ray count
// no greater than mostPixels.
func newView(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays int) (view, error) {
	if world == nil {
		return view{}, errNoWorld
	}
	if err := ValidateClearState(world, nil); err != nil {
		return view{}, err
	}
	if err := validateOceanDirectSunWorld(world); err != nil {
		return view{}, err
	}
	from := eye.Origin
	if !finite(from.X, from.Y, from.Z, eye.Scale) {
		return view{}, fmt.Errorf("trace: an eye is at a place, and "+
			"this one is at %v at a scale of %v", from, eye.Scale)
	}
	if width < 1 || height < 1 {
		return view{}, fmt.Errorf("trace: a picture is at least one "+
			"pixel each way, and this is %d by %d", width, height)
	}
	if rays < 1 {
		return view{}, fmt.Errorf("trace: a pixel has at least one "+
			"ray each way, and this asks for %d", rays)
	}
	// every ray of every pixel, counted without multiplying them out
	room := mostPixels / width / height
	if room < 1 || rays > room/rays {
		return view{}, fmt.Errorf("trace: %d by %d pixels at %d rays "+
			"each way is more than %d rays", width, height, rays,
			mostPixels)
	}
	seen := view{world: world, eye: eye, lens: lens, at: moment,
		width: width, height: height, rays: rays}
	return seen, nil
}

// finite reports whether every input is neither NaN nor infinite.
func finite(numbers ...float64) bool {
	for _, number := range numbers {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return false
		}
	}
	return true
}

// fine expands dimensions by the sampling rate and uses one ray per pixel,
// producing the high-resolution image to average afterward.
func (seen view) fine() view {
	seen.width, seen.height = seen.width*seen.rays, seen.height*seen.rays
	seen.rays = 1
	return seen
}

// pixelAngle estimates a pixel's angular width in radians using adjacent rays
// at the image centre. It is the angle between their directions, taken as the
// arctangent of the length of their cross product over their dot product, which
// stays exact at angles too small for a cosine to tell from one.
func pixelAngle(eye maths.Subspace, lens film.Sensor, width,
	height int) float64 {
	middle := film.Look(eye, lens, width/2, height/2, width, height,
		maths.Band{})
	beside := film.Look(eye, lens, width/2+1, height/2, width, height,
		maths.Band{})
	return math.Atan2(middle.Dir.Cross(beside.Dir).Len(),
		middle.Dir.Dot(beside.Dir))
}

// Render traces the world at moment through lens from eye into a width by
// height frame. Independent rows run in parallel; scheduling does not change
// the result for an unchanged world. Invalid requests return an error before
// rendering; see newView for the limits.
func Render(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height int) (*frame.Frame, error) {
	return RenderWithInitialMedia(world, eye, lens, moment, width, height, nil)
}

// RenderWithInitialMedia traces the world from a camera whose starting clear
// media are explicitly given in entry order. An empty slice has exactly the
// all-air behavior of Render. Initial media are values; they are copied and
// validated before rendering starts.
func RenderWithInitialMedia(world *thing.World, eye maths.Subspace,
	lens film.Sensor, moment time.Duration, width, height int,
	media []InitialMedium) (*frame.Frame, error) {
	seen, err := newView(world, eye, lens, moment, width, height, 1)
	if err != nil {
		return nil, err
	}
	seen, err = seen.withInitialMedia(media)
	if err != nil {
		return nil, err
	}
	return render(seen)
}

// paint evaluates pixel once per coordinate, distributing rows across
// GOMAXPROCS workers. The callback runs concurrently and must be independent
// for each coordinate.
func paint(width, height int,
	pixel func(x, y int) frame.Pixel) *frame.Frame {
	painted := frame.New(width, height)
	rows := make(chan int, height)
	for y := 0; y < height; y++ {
		rows <- y
	}
	close(rows)
	var working sync.WaitGroup
	for range max(1, runtime.GOMAXPROCS(0)) {
		working.Add(1)
		go func() {
			defer working.Done()
			for y := range rows {
				for x := 0; x < width; x++ {
					painted.Set(x, y, pixel(x, y))
				}
			}
		}()
	}
	working.Wait()
	return painted
}

// render traces a validated view, or returns the error of a scene that
// cannot be drawn.
func render(seen view) (*frame.Frame, error) {
	scene := stage(seen.world, seen.at)
	scene.enableClearInscatterFault(seen)
	if err := scene.refusal(); err != nil {
		return nil, err
	}
	wide := seen.wide()
	drawn := paint(seen.width, seen.height, func(x, y int) frame.Pixel {
		return seen.through(scene, wide, x, y)
	})
	if err := scene.refusal(); err != nil {
		return nil, err
	}
	return drawn, nil
}

// wide returns the world's explicit angular pixel width when supplied,
// otherwise estimating it from neighbouring camera rays.
func (seen view) wide() float64 {
	if seen.world.Pixel > 0 {
		return seen.world.Pixel
	}
	return pixelAngle(seen.eye, seen.lens, seen.width, seen.height)
}

// through traces the view's ray for pixel x, y through the staged scene.
func (seen view) through(scene *staged, wide float64, x, y int) frame.Pixel {
	ray := film.Look(seen.eye, seen.lens, x, y, seen.width, seen.height,
		maths.Band{})
	if told := scene.walked; told != nil {
		of := fmt.Sprintf("%dx%d", seen.width, seen.height)
		told.tell("a ray from the lens", "x", x, "y", y, "of", of,
			"ray", ray)
	}
	asked := look{ray: ray, pixel: wide}
	if seen.holding > 0 {
		asked.inside, asked.holding = seen.inside, seen.holding
	}
	return shade(scene, asked)
}

// Sampled traces rays by rays samples per pixel and averages linear light and
// coverage, retaining the nearest depth. Supersampling reduces aliasing of
// details smaller than a pixel. With rays equal to one, it matches Render.
// Invalid sampling or a total count above mostPixels returns an error.
func Sampled(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays int) (*frame.Frame, error) {
	return SampledWithInitialMedia(world, eye, lens, moment, width, height,
		rays, nil)
}

// SampledWithInitialMedia is Sampled with an explicit ordered stack of clear
// media containing the camera. An empty slice has exactly the all-air
// behavior of Sampled. See InitialMedium for the values' meaning and limits.
func SampledWithInitialMedia(world *thing.World, eye maths.Subspace,
	lens film.Sensor, moment time.Duration, width, height, rays int,
	media []InitialMedium) (*frame.Frame, error) {
	seen, err := newView(world, eye, lens, moment, width, height, rays)
	if err != nil {
		return nil, err
	}
	seen, err = seen.withInitialMedia(media)
	if err != nil {
		return nil, err
	}
	fine, err := render(seen.fine())
	if err != nil {
		return nil, err
	}
	if rays == 1 {
		return fine, nil
	}
	return averaged(fine, width, height, rays), nil
}

// averaged reduces the fine frame to the requested dimensions, averaging light
// and coverage over each sample block and retaining minimum depth.
func averaged(fine *frame.Frame, width, height, rays int) *frame.Frame {
	coarse := frame.New(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			coarse.Set(x, y, meanUnder(fine, x, y, rays))
		}
	}
	return coarse
}

// meanUnder combines the rays by rays fine pixels belonging to one output
// pixel.
func meanUnder(fine *frame.Frame, x, y, rays int) frame.Pixel {
	return averageOf(rays, func(over, down int) frame.Pixel {
		return fine.At(x*rays+over, y*rays+down)
	})
}

// averageOf combines a rays by rays block in row-major order: mean RGB and
// coverage, minimum depth. Sampled and Replay share this order so their
// floating-point results match exactly.
func averageOf(rays int, under func(over, down int) frame.Pixel) frame.Pixel {
	mean := frame.Pixel{Depth: float32(math.Inf(1))}
	for down := 0; down < rays; down++ {
		for over := 0; over < rays; over++ {
			each := under(over, down)
			mean.R += each.R
			mean.G += each.G
			mean.B += each.B
			mean.Cover += each.Cover
			mean.Depth = min(mean.Depth, each.Depth)
		}
	}
	share := 1 / float32(rays*rays)
	mean.R, mean.G, mean.B = mean.R*share, mean.G*share, mean.B*share
	mean.Cover *= share
	return mean
}
