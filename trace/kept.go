package trace

// A long draw is drawn a whole picture row at a time, so that it can
// stop between two rays and later pick up where it stopped. Each row is
// its rays by rays fine rows, cast and averaged in the order Sampled
// averages them, so a picture drawn in pieces is Sampled's to the bit.

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// The counts a kept draw is made of.
const (
	oneRay       = 1 // defined: a pixel of one ray is its fine pixel
	fewestWorker = 1 // defined: a draw has at least one worker
)

// Rows is where a draw keeps each picture row it finishes, and finds the
// rows an earlier draw of the same picture finished.
//
// Begin is told how many rows the picture has, before any is asked for.
// Taken is a row an earlier draw kept, and whether there is one; a row
// of another width is not taken. Drawn is called by the draw's workers
// at once, so it must be safe for that, and it must return quickly.
type Rows interface {
	Begin(rows int)
	Taken(row, width int) ([]frame.Pixel, bool)
	Drawn(row int, pixels []frame.Pixel)
}

// Keeping is what a draw stops on and where it keeps its rows. The zero
// Keeping never stops and keeps nothing.
type Keeping struct {
	stop context.Context
	rows Rows
}

// NewKeeping is a keeping that stops when stop is done and keeps its
// rows in rows. Either may be nil: a draw that does not stop, or one
// that keeps nothing.
func NewKeeping(stop context.Context, rows Rows) Keeping {
	return Keeping{stop: stop, rows: rows}
}

// Unkept is the keeping with its stop and no rows: for a draw that is one
// of several pictures of one frame, whose rows the frame's rows are not.
func (keeping Keeping) Unkept() Keeping {
	keeping.rows = nil
	return keeping
}

// SampledKept is SampledWithInitialMedia drawn a picture row at a time:
// the rows the keeping has from an earlier draw are taken up, and the
// rest are drawn and each kept as it is finished. The picture is
// Sampled's to the bit, however many draws it took.
//
// When the keeping's stop is done, every worker leaves its row at the
// next ray, that row is not kept, and the draw gives the stop's error
// with the rows finished; the rows finished stay kept. A row drawn
// while the scene has a refusal is not kept, and the refusal is given.
func SampledKept(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays int,
	media []InitialMedium, keeping Keeping) (*frame.Frame, error) {
	seen, err := newView(world, eye, lens, moment, width, height, rays)
	if err != nil {
		return nil, err
	}
	seen, err = seen.withInitialMedia(media)
	if err != nil {
		return nil, err
	}
	return keeping.render(seen)
}

// painting is one kept draw in flight: the fine view rays are cast in,
// the rays a side of a pixel, the staged scene, the angle a fine pixel
// sees, the picture, the rows left to draw, and how many are finished.
type painting struct {
	keeping               Keeping
	fine                  view
	rays                  int
	scene                 *staged
	wide                  float64
	regionLeft, regionTop int
	picture               *frame.Frame
	left                  chan int
	finished              atomic.Int64
}

// render is the view drawn by rows, or the refusal of a scene that cannot
// be drawn, or the stop's error where the draw was told to stop first.
func (keeping Keeping) render(seen view) (*frame.Frame, error) {
	fine := seen.fine()
	scene, err := fine.staging()
	if err != nil {
		return nil, err
	}
	made := newPainting(keeping, seen, scene)
	made.paint()
	if err := scene.refusal(); err != nil {
		return nil, err
	}
	if done := int(made.finished.Load()); done < seen.height {
		return nil, fmt.Errorf("trace: the draw stopped with %d of %d "+
			"rows finished: %w", done, seen.height, keeping.stop.Err())
	}
	return made.picture, nil
}

// newPainting is a draw of the view by rows through the staged scene. A
// scene that tells its rays' walks takes up nothing, so every ray is
// told.
func newPainting(keeping Keeping, seen view, scene *staged) *painting {
	fine := seen.fine()
	if scene.walked != nil {
		keeping.rows = nil
	}
	return &painting{keeping: keeping, fine: fine, rays: seen.rays,
		scene: scene, wide: fine.wide(),
		picture: frame.New(seen.width, seen.height),
		left:    make(chan int, seen.height)}
}

// paint takes up the rows kept before, then draws the rest across
// GOMAXPROCS workers, as paint does.
func (made *painting) paint() {
	made.takeUp()
	close(made.left)
	var working sync.WaitGroup
	for range max(fewestWorker, runtime.GOMAXPROCS(0)) {
		working.Add(1)
		go made.work(&working)
	}
	working.Wait()
}

// takeUp puts each row kept before into the picture, and every other row
// on the list to draw.
func (made *painting) takeUp() {
	if made.keeping.rows != nil {
		made.keeping.rows.Begin(made.picture.H)
	}
	for row := 0; row < made.picture.H; row++ {
		if !made.taken(row) {
			made.left <- row
		}
	}
}

// taken is whether row was kept before, at this width; it is put into
// the picture where it was.
func (made *painting) taken(row int) bool {
	if made.keeping.rows == nil {
		return false
	}
	pixels, found := made.keeping.rows.Taken(row, made.picture.W)
	if !found || len(pixels) != made.picture.W {
		return false
	}
	made.put(row, pixels)
	return true
}

// work draws rows off the list until it is empty or the draw is told to
// stop, and keeps each row it finishes while the scene has no refusal.
func (made *painting) work(working *sync.WaitGroup) {
	defer working.Done()
	for row := range made.left {
		pixels, whole := made.row(row)
		if !whole {
			return
		}
		made.put(row, pixels)
		if made.keeping.rows != nil && made.scene.refusal() == nil {
			made.keeping.rows.Drawn(row, pixels)
		}
	}
}

// put sets a finished row into the picture and counts it.
func (made *painting) put(row int, pixels []frame.Pixel) {
	for x, pixel := range pixels {
		made.picture.Set(x, row, pixel)
	}
	made.finished.Add(1)
}

// row is picture row y drawn from its fine rows, and whether it was drawn
// whole: a draw told to stop leaves it at the next ray.
func (made *painting) row(y int) ([]frame.Pixel, bool) {
	block := frame.New(made.picture.W*made.rays, made.rays)
	for down := 0; down < made.rays; down++ {
		if !made.across(block, (made.regionTop+y)*made.rays+down, down) {
			return nil, false
		}
	}
	return made.averaged(block), true
}

// across casts fine row at into the block's row down, and says whether
// it got to the row's end.
func (made *painting) across(block *frame.Frame, at, down int) bool {
	for x := 0; x < block.W; x++ {
		if made.keeping.told() {
			return false
		}
		block.Set(x, down, made.fine.through(made.scene, made.wide, made.regionLeft*made.rays+x, at))
	}
	return true
}

// averaged is a block of fine rows as one picture row: each pixel the
// mean of its rays by rays, in averageOf's order, or the fine row itself
// at one ray, as Sampled gives it.
func (made *painting) averaged(block *frame.Frame) []frame.Pixel {
	pixels := make([]frame.Pixel, made.picture.W)
	if made.rays == oneRay {
		copy(pixels, block.Pix)
		return pixels
	}
	for x := range pixels {
		pixels[x] = meanUnder(block, x, 0, made.rays)
	}
	return pixels
}

// told is whether the keeping's stop is done.
func (keeping Keeping) told() bool {
	if keeping.stop == nil {
		return false
	}
	select {
	case <-keeping.stop.Done():
		return true
	default:
		return false
	}
}
