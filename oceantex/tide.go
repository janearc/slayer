package oceantex

import (
	"fmt"
	"math"
)

// tideSpeeds are how fast each tidal constituent turns, in degrees an
// hour, as NOAA lists them. They are the same at every port.
var tideSpeeds = func() map[string]float64 {
	speeds := nums("tide.speeds")
	names := []string{"M2", "S2", "N2", "K1", "M4", "O1"}
	out := map[string]float64{}
	for i, name := range names {
		out[name] = speeds[i]
	}
	return out
}()

// Constituent is one of the regular rises and falls a tide is the sum of.
type Constituent struct {
	// Name is M2, S2, N2, K1, O1 or M4.
	Name string
	// Amplitude is half its range, in metres.
	Amplitude float64
	// Phase is how far through its cycle it is, in degrees, behind its
	// own high water at hour zero.
	Phase float64
}

// Tide is the tide at one place, as the constituents measured there. A
// port's are published; NOAA lists them for every station it keeps.
type Tide []Constituent

// Level is the height of the tide above its mean, in metres, at a number
// of hours after hour zero: each constituent's amplitude times the cosine
// of its speed times the hours, less its phase, all added up.
//
// This is the harmonic method with the slow parts left out. A published
// phase is counted from an astronomical zero that moves with the moon's
// orbit, and each amplitude swells and shrinks over 18.6 years. Neither
// correction is here, so Level gives a tide of the right size and rhythm
// for a place, and will not match its tide table hour for hour unless the
// phases are given for the hour zero in use.
func (tide Tide) Level(hours float64) (float64, error) {
	var level float64
	for _, part := range tide {
		speed, known := tideSpeeds[part.Name]
		if !known {
			return 0, fmt.Errorf("oceantex: no tidal constituent "+
				"called %q", part.Name)
		}
		level += part.Amplitude *
			math.Cos((speed*hours-part.Phase)*degrees)
	}
	return level, nil
}

// Flood is a sea with the tide in: its depth is the charted depth, the
// depth at mean water, plus the tide's level at an hour. A depth the tide
// has taken to nothing is an error, since a dry sea has no waves.
func (sea Sea) Flood(charted float64, tide Tide, hours float64) (Sea, error) {
	level, err := tide.Level(hours)
	if err != nil {
		return sea, err
	}
	if charted+level <= 0 {
		return sea, fmt.Errorf("oceantex: the tide is out; %.2f m of "+
			"water over a bottom charted at %.2f m", level, charted)
	}
	sea.Depth = charted + level
	return sea, nil
}
