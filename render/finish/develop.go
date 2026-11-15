package finish

import (
	"fmt"
	"image"
	"math"
	"slices"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm"
	ninjacolour "github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/roll"
	"github.com/janearc/slayer/render/carry"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// Developer turns one frame's light into the picture a person watches.
// It is a plain tone curve or a film stock, chosen once from a job.
type Developer struct {
	plain colour.Tone
	stock *stocked
}

// stocked is a roll of a stock and the screen its prints are shown on.
type stocked struct {
	roll   roll.Roll
	screen ninjacolour.Tone
}

// NewDeveloper is the developer a job asks for: its stock if it says
// one, else its print, else the tone the caller gives. A job that says
// both is refused by package job, and here a stock stands first.
func NewDeveloper(made job.Job, fallback colour.Tone) (Developer, error) {
	if film, said := made.Stock(); said {
		seed, _ := film.Seed()
		rated, _ := film.Rated()
		return stockDeveloper(film.Name(), seed, rated)
	}
	if print, said := made.Print(); said {
		return Developer{plain: toneOf(print)}, nil
	}
	return Developer{plain: fallback}, nil
}

// toneOf is a job's print as a tone: its stops doubling the light each.
func toneOf(print job.Print) colour.Tone {
	stops, _ := print.Stops()
	bloom, _ := print.Bloom()
	return colour.Tone{Exposure: math.Pow(stopBase, stops),
		Knee: print.Knee(), Top: print.Top(), Bloom: bloom}
}

// stockDeveloper is a developer through the stock of a name, one roll
// of it, rated at some stops of push or pull. A name not on the shelf,
// or a seed below nought or past what a job holds, is refused.
func stockDeveloper(name string, seed int64, rated float64) (Developer,
	error) {
	shelf, err := ninjafilm.Stocks()
	if err != nil {
		return Developer{}, fmt.Errorf("finish: the shelf cannot be "+
			"listed: %w", err)
	}
	exposure, known := transmission[name]
	if !slices.Contains(shelf, name) || !known {
		return Developer{}, fmt.Errorf("finish: no stock is called %q; "+
			"the shelf holds %v; name one of those", name, shelf)
	}
	if seed < none || seed > seedMost {
		return Developer{}, fmt.Errorf("finish: seed %d is outside "+
			"%d to %d; pick one inside", seed, none, int64(seedMost))
	}
	loaded, err := ninjafilm.Roll(name, uint64(seed))
	if err != nil {
		return Developer{}, err
	}
	screen := screenOf(exposure, loaded.Clearest())
	return Developer{stock: &stocked{loaded.Rated(rated), screen}}, nil
}

// screenOf is the tone a print is shown through: turned up by the
// exposure, with a shoulder that lands the stock's clearest on white.
// The knee is a share of the stops the clearest holds above white, that
// many below it. A print that never passes white has no shoulder.
func screenOf(exposure float64, clearest maths.Light) ninjacolour.Tone {
	top := exposure * clearest.Luminance()
	if !(top > whole) {
		return ninjacolour.NewTone(exposure, none, none, none)
	}
	knee := math.Pow(top, -shoulderShare)
	return ninjacolour.NewTone(exposure, knee, top, none)
}

// Stocked is whether the developer is a film stock and not a tone.
func (made Developer) Stocked() bool { return made.stock != nil }

// Develop is the picture of the frame numbered number in the film,
// from its glimpses. One glimpse is a frame of one instant; several are
// a frame blurred by a shutter. A stock takes them all and a tone takes
// their mean in linear light. The number fixes a stock's grain: the
// same number gives the same grain, the next number new grain.
func (made Developer) Develop(number int, glimpses []*frame.Frame) (
	*image.RGBA, error) {
	if number < none {
		return nil, fmt.Errorf("finish: frame number %d is below "+
			"nought; a film counts its frames from nought", number)
	}
	if made.stock != nil {
		return made.stock.develop(number, glimpses)
	}
	light, err := Mean(glimpses)
	if err != nil {
		return nil, err
	}
	return made.plain.Develop(light), nil
}

// develop is the frame as a stock prints it and the screen shows it.
func (held *stocked) develop(number int, glimpses []*frame.Frame) (
	*image.RGBA, error) {
	printed, err := held.roll.Print(number, carry.NinjaFrames(glimpses))
	if err != nil {
		return nil, err
	}
	return held.screen.Develop(printed), nil
}
