package sensor

import (
	"errors"

	"github.com/janearc/celestial/bands"
)

// none is no fraction and no bands at all. It is defined as zero.
const none = 0

// inclusive is one band on, so that both the first band and the last of
// a run are in it. It is defined as one.
const inclusive = 1

// whole is all of a band's light. It is defined as one.
const whole = 1

var errResponse = errors.New("ninjafilm: a response counts a fraction " +
	"between nought and one, of at least one band")

// Response is the fraction of each band a channel counts. It is how much
// of the light in that band an eye's cone, a film's dye layer or a
// detector behind a filter turns into its reading.
//
// That is quantum efficiency in the trade's word. Bands it does not
// list it does not count.
//
//ninja:vow practice slayer's camera holds one; tests only so far
type Response struct {
	fractions map[bands.Band]float64
}

// Counts is a response from the fraction counted in each band. It
// refuses an empty one. It refuses any fraction less than zero or above
// one or not a number.
//
//ninja:vow practice slayer's camera makes responses here; tests only so far
func Counts(fractions map[bands.Band]float64) (made Response, err error) {
	if len(fractions) == none {
		return made, errResponse
	}
	kept := make(map[bands.Band]float64, len(fractions))
	for band, fraction := range fractions {
		if !(fraction >= none && fraction <= whole) {
			return made, errResponse
		}
		kept[band] = fraction
	}
	return Response{kept}, nil
}

// of is the fraction of a band the response counts.
func (response Response) of(band bands.Band) float64 {
	return response.fractions[band]
}
