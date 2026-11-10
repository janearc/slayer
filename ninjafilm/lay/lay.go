// Package lay arranges a stock's records as stacked layers or side-by-side
// mosaic sites.
package lay

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/hash"
	"github.com/janearc/slayer/ninjafilm/number"
	"github.com/janearc/slayer/ninjafilm/record"
)

// inch is an inch in micrometres. It is defined as 25.4 millimetres by
// the international agreement of 1959.
const inch = 25400.0

// none is no elements and no dimness at all. It is defined as zero.
const none = 0

// whole is the whole of the light and the fewest sites across a pixel.
// It is defined as one.
const whole = 1

// what a lay refuses
var (
	errStacked = errors.New("ninjafilm: a stacked stock has no " +
		"elements and no dimness of its own")
	errMosaic = errors.New("ninjafilm: a mosaic has elements to the " +
		"square inch, and passes some of the light but not more " +
		"than all")
)

// Lay is how a stock's records lie on its base.
type Lay interface {
	// Across is how many sites lie across a pixel that is pitch
	// micrometres wide.
	Across(pitch float64) int
	// Site is what the light meets at one site of a frame. That is the
	// records there, and the filter the light passes on the way in and
	// again on the way out.
	Site(records []record.Record, seed uint64, x, y, frame int64) (
		[]record.Record, maths.Light)
}

// stacked is records one under another, each as wide as the film. Every
// record meets the light at every site, through no filter.
type stacked struct{}

// newStacked is a stacked lay, which takes nothing from the table.
func newStacked(perInch, passes float64) (Lay, error) {
	if perInch != none || passes != none {
		return nil, errStacked
	}
	return stacked{}, nil
}

// Across is one, for a pixel of a stacked stock is one site.
func (stacked) Across(float64) int { return whole }

// Site is every record, under a filter that passes everything.
func (stacked) Site(records []record.Record, _ uint64, _, _, _ int64) (
	[]record.Record, maths.Light) {
	return records, maths.Grey(whole)
}

// mosaic is records side by side. The film is a scatter of elements of
// one width. Each holds one record. The light passes that record's own
// window going in and coming out. The passes are the share of light the
// finished plate lets through at its clearest.
type mosaic struct{ pitch, passes float64 }

// newMosaic is a mosaic of so many elements to the square inch passing
// a share of the light, or why not. The width of an element is worked
// out from the count.
func newMosaic(perInch, passes float64) (Lay, error) {
	if !number.Positive(perInch) || !number.Positive(passes) ||
		passes > whole {
		return nil, errMosaic
	}
	return mosaic{inch / math.Sqrt(perInch), passes}, nil
}

// Across is how many elements fit across a pixel. It is never fewer
// than one. Pixels finer than an element see one element.
func (scatter mosaic) Across(pitch float64) int {
	fitted := math.Ceil(pitch / scatter.pitch)
	return int(math.Max(whole, fitted))
}

// Site is the one record the element at a site holds, picked by the
// emulsion's hash. It stands under that record's window. The window is
// dimmed so that the windows together pass what the plate passes.
func (scatter mosaic) Site(records []record.Record, seed uint64, x, y,
	frame int64) ([]record.Record, maths.Light) {
	kinds := float64(len(records))
	share := hash.Three(seed, x, y, frame)
	picked := records[int(share*kinds)]
	dimmed := picked.Window().Scale(scatter.passes * kinds)
	return []record.Record{picked}, dimmed
}

// lays is every lay by its name in the table. The table chooses and no
// caller does.
var lays = map[string]func(perInch, passes float64) (Lay, error){
	"stacked": newStacked,
	"mosaic":  newMosaic,
}

// New is the lay a table row names, made from the numbers of the row.
func New(name string, perInch, passes float64) (Lay, error) {
	build, known := lays[name]
	if !known {
		return nil, fmt.Errorf("ninjafilm: no lay is called %q", name)
	}
	return build(perInch, passes)
}
