package oceantex

import (
	"errors"
	"math"
)

var (
	stateBounds  = nums("seastate.bounds")
	stateTarget  = num("seastate.target")
	stateWindTop = num("seastate.wind.top")
	beaufortUnit = num("beaufort.scale")
	beaufortRise = num("beaufort.power")
)

// stateNames are the descriptive terms of WMO code table 3700.
var stateNames = [...]string{
	"calm (glassy)", "calm (rippled)", "smooth (wavelets)", "slight",
	"moderate", "rough", "very rough", "high", "very high", "phenomenal",
}

// SeaState is a state of the sea in the World Meteorological
// Organization's code, table 3700: 0, calm and glassy, to 9, phenomenal.
// It is the scale mariners and coast guards report, and the wind-sea half
// of the Douglas scale.
type SeaState int

// String is the code's descriptive term.
func (code SeaState) String() string {
	if code < 0 || int(code) >= len(stateNames) {
		return "not a sea state"
	}
	return stateNames[code]
}

// Heights is the range of wave heights a code covers, in metres. A height
// exactly on a bound belongs to the lower code, as the table's note says:
// four metres is 5, not 6. Code 9 has no upper bound, and reports
// infinity.
func (code SeaState) Heights() (from, to float64, err error) {
	switch {
	case code < 0 || int(code) >= len(stateNames):
		return 0, 0, errors.New("oceantex: sea states run from 0 to 9")
	case code == 0:
		return 0, 0, nil
	case int(code) == len(stateBounds):
		return stateBounds[code-1], math.Inf(1), nil
	}
	return stateBounds[code-1], stateBounds[code], nil
}

// StateOf is the code for a significant wave height in metres.
func StateOf(height float64) SeaState {
	if height <= 0 {
		return 0
	}
	code := SeaState(1)
	for int(code) < len(stateBounds) && height > stateBounds[code] {
		code++
	}
	return code
}

// Raise is a sea in a given state: the wind that, blowing over this fetch
// and depth, raises waves of the height the code describes. The wind is
// found by trying winds until the spectrum's significant height matches.
//
// GLUE: the height aimed for is the middle of the code's range, and for
// code 9, which has no top, its lower bound. If no wind up to 80 metres a
// second can raise the state over this fetch and depth, Raise says so.
func (code SeaState) Raise(fetch, depth, toward float64) (Sea, error) {
	from, to, err := code.Heights()
	if err != nil {
		return Sea{}, err
	}
	sea := Sea{Fetch: fetch, Depth: depth, Toward: toward}
	if code == 0 {
		return sea, nil
	}
	aim := from
	if !math.IsInf(to, 1) {
		aim = from + stateTarget*(to-from)
	}
	calm, gale := 0.0, stateWindTop
	sea.Wind = gale
	if sea.Height() < aim {
		return Sea{}, errors.New("oceantex: no wind raises a " +
			code.String() + " sea over this fetch and depth")
	}
	for range 60 {
		sea.Wind = (calm + gale) / 2
		if sea.Height() < aim {
			calm = sea.Wind
		} else {
			gale = sea.Wind
		}
	}
	return sea, nil
}

// Beaufort is the wind speed, in metres a second ten metres up, of a
// Beaufort force: 0.836 B^(3/2), the relation behind the 1946 scale.
// Forces between whole numbers are allowed; 9.5 is the bottom of force 10.
func Beaufort(force float64) float64 {
	return beaufortUnit * math.Pow(math.Max(force, 0), beaufortRise)
}
