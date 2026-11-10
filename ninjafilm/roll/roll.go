// Package roll loads a photographic stock, integrates light received during a
// shutter opening, and develops the exposure grain by grain. Its output is the
// linear light transmitted by the developed frame under projection.
package roll

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/ninjafilm/lay"
	"github.com/janearc/slayer/ninjafilm/number"
	"github.com/janearc/slayer/ninjafilm/record"
)

// what a roll refuses to print
var (
	errNoRoll = errors.New("ninjafilm: a roll comes from Roll, and is " +
		"rated within what a number holds")
	errNoGlimpses = errors.New("ninjafilm: a frame is printed from one " +
		"glimpse or more")
	errUnsound = errors.New("ninjafilm: a glimpse is a frame of one " +
		"pixel or more, with a light for every pixel")
	errUnlike = errors.New("ninjafilm: the glimpses of a frame are all " +
		"one size")
	errEndless = errors.New("ninjafilm: a glimpse holds light that is " +
		"not a number")
)

// perMillimetre is how many micrometres a millimetre is. It is defined
// as 1000 by the metric prefixes.
const perMillimetre = 1000.0

// ratedGain is the gain of a roll exposed at its rating. It is defined
// as one.
const ratedGain = 1

// none is no count at all. It is defined as zero.
const none = 0

// smallest is the fewest pixels a frame has each way. It is defined as
// one.
const smallest = 1

// whole is the whole of a share. It is defined as one.
const whole = 1

// firstPlace is the place of the first item of a list. It is defined as zero.
const firstPlace = 0

// firstDraw puts the first record of a site one on from the seed of the
// roll, so that none draws with the seed itself. It is defined as one.
const firstDraw = 1

// Stock describes photosensitive records, their layout, and the film gauge.
type Stock struct {
	gauge   film.Gauge
	lay     lay.Lay
	records []record.Record
}

// NewStock assembles records on a layout and film gauge.
func NewStock(gauge film.Gauge, laid lay.Lay,
	records []record.Record) Stock {
	return Stock{gauge, laid, records}
}

// Shelf supplies named stock descriptions for loading a roll.
type Shelf interface {
	// Find is the stock of a name, or why there is none.
	Find(name string) (Stock, error)
}

// Roll is a seeded stock with an exposure gain. Different seeds retain the
// stock's response but produce different grain.
type Roll struct {
	stock Stock
	seed  uint64
	gain  float64
}

// Load obtains a named stock from the shelf and initializes a seeded roll at
// its rated exposure. Unknown names return the shelf's error.
func Load(shelf Shelf, name string, seed uint64) (Roll, error) {
	loaded, err := shelf.Find(name)
	if err != nil {
		return Roll{}, err
	}
	return Roll{stock: loaded, seed: seed, gain: ratedGain}, nil
}

// Rated adjusts exposure by stops relative to the stock's rating.
// Positive values increase exposure, negative values reduce it, and
// each stop doubles the light.
//
//ninja:vow practice slayer rates each roll here; no caller in this tree
func (loaded Roll) Rated(stops float64) Roll {
	loaded.gain = math.Exp2(stops)
	return loaded
}

// Gauge returns the roll's film dimensions for lens projection.
//
//ninja:vow practice slayer takes the gauge here; no caller in this tree
func (loaded Roll) Gauge() film.Gauge { return loaded.stock.gauge }

// Print integrates a frame's glimpses of linear light and develops the
// result. which identifies the frame for reproducible grain.
//
// The output is linear transmission under a white projector lamp, at most
// one per channel. Coverage and depth come from the first glimpse.
//
//ninja:vow practice slayer develops frames here; no caller in this tree
func (loaded Roll) Print(which int, glimpses []*frame.Frame) (*frame.Frame,
	error) {
	if !number.Positive(loaded.gain) {
		return nil, errNoRoll
	}
	caught, err := gathered(glimpses)
	if err != nil {
		return nil, err
	}
	held := loaded.stock
	width := held.gauge.Width() * perMillimetre / float64(caught.W)
	across := held.lay.Across(width)
	through := pass{loaded, int64(which), width, across}
	for row := range caught.H {
		for column := range caught.W {
			through.develop(caught, column, row)
		}
	}
	return caught, nil
}

// develop writes the developed transmission at one pixel.
func (through pass) develop(caught *frame.Frame, column, row int) {
	pixel := caught.At(column, row)
	light := through.pixel(pixel.Light(), column, row)
	pixel.R, pixel.G = float32(light.R), float32(light.G)
	pixel.B = float32(light.B)
	caught.Set(column, row, pixel)
}

// sound validates that a glimpse is non-nil, has positive dimensions, and
// supplies one light value per pixel.
func sound(glimpse *frame.Frame) bool {
	return glimpse != nil && glimpse.W >= smallest &&
		glimpse.H >= smallest && len(glimpse.Pix) == glimpse.W*glimpse.H
}

// gathered averages RGB over equally weighted glimpses while retaining the
// first frame's coverage and depth. The mean represents integrated light
// normalized by the shutter-open duration.
func gathered(glimpses []*frame.Frame) (*frame.Frame, error) {
	if len(glimpses) == none {
		return nil, errNoGlimpses
	}
	first := glimpses[firstPlace]
	if !sound(first) {
		return nil, errUnsound
	}
	base := frame.New(first.W, first.H)
	copy(base.Pix, first.Pix)
	lights := make([]maths.Light, len(base.Pix))
	for _, glimpse := range glimpses {
		if err := added(lights, base, glimpse); err != nil {
			return nil, err
		}
	}
	return averaged(base, lights, len(glimpses))
}

// added accumulates a glimpse's RGB after checking its dimensions and channel
// values.
func added(lights []maths.Light, base, glimpse *frame.Frame) error {
	if !sound(glimpse) {
		return errUnsound
	}
	if glimpse.W != base.W || glimpse.H != base.H {
		return errUnlike
	}
	for index, pixel := range glimpse.Pix {
		lights[index] = lights[index].Add(pixel.Light())
	}
	return nil
}

// averaged divides accumulated RGB by count and rejects nonfinite results.
func averaged(base *frame.Frame, totals []maths.Light,
	count int) (*frame.Frame, error) {
	for index, total := range totals {
		if !number.Finite(total) {
			return nil, errEndless
		}
		mean := total.Scale(whole / float64(count))
		pixel := &base.Pix[index]
		pixel.R, pixel.G = float32(mean.R), float32(mean.G)
		pixel.B = float32(mean.B)
	}
	return base, nil
}

// pass stores frame number, pixel width on film in micrometres, and mosaic
// sites per pixel.
type pass struct {
	roll   Roll
	number int64
	width  float64
	across int
}

// pixel averages developed transmission across the sites covered by a pixel.
func (through pass) pixel(light maths.Light, column, row int) maths.Light {
	var total maths.Light
	for down := range through.across {
		for along := range through.across {
			x := int64(column*through.across + along)
			y := int64(row*through.across + down)
			total = total.Add(through.site(light, x, y))
		}
	}
	sites := through.across * through.across
	return total.Scale(whole / float64(sites))
}

// site applies the site filter to the transmission of its developed records.
func (through pass) site(light maths.Light, x, y int64) maths.Light {
	held, seed := through.roll.stock, through.roll.seed
	fitted, filter := held.lay.Site(held.records, seed, x, y,
		through.number)
	sites := float64(through.across * through.across)
	area := through.width * through.width / sites
	exposing := light.Scale(through.roll.gain)
	var blocked maths.Light
	for index, one := range fitted {
		draw := through.draw(index, x, y)
		dense := one.Developed(exposing, area, draw)
		blocked = blocked.Add(one.Image().Scale(dense))
	}
	return filter.Times(cleared(blocked))
}

// draw selects reproducible grain variation for one record at a site.
func (through pass) draw(index int, x, y int64) float64 {
	seed := through.roll.seed + uint64(index) + firstDraw
	return crystal.Chance(seed, x, y, through.number)
}

// cleared converts channel densities to transmission using 10^-density.
func cleared(density maths.Light) maths.Light {
	return maths.RGB(passed(density.R), passed(density.G),
		passed(density.B))
}

// passed returns transmission for one channel density.
func passed(density float64) float64 {
	return math.Exp(-density * math.Ln10)
}
