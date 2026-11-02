// Package sensor defines light collectors: areas with one or more channels,
// each recording a fraction of incoming spectral power.
//
// Camera and telescope models are planned above this primitive. A camera’s
// image follows lens projection; a telescope’s resolution also depends on
// diffraction, wavelength, and aperture. Nonvisible telescope bands require a
// separate mapping to display colours. Neither extension is implemented.
package sensor

import (
	"errors"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/slayer/ninjafilm/maths"
)

var errCollector = errors.New("ninjafilm: a collector has at least one " +
	"channel, and every channel has a name")

// Collector gathers the light arriving over its area. Each of its
// channels counts its own share of every band. One channel is a
// detector behind one filter. Three are an eye's cones, or a colour
// film's layers.
//
//ninja:vow practice slayer's camera holds one; tests only so far
type Collector struct {
	area     maths.Area
	channels map[string]Response
}

// Collect is a collector of an area, with its channels by name. It
// refuses one with no channels, or a channel with no name.
//
//ninja:vow practice slayer's camera makes its collector here; tests only
func Collect(area maths.Area, channels map[string]Response) (made Collector,
	err error) {
	if len(channels) == none {
		return made, errCollector
	}
	kept := make(map[string]Response, len(channels))
	for name, response := range channels {
		if name == "" {
			return made, errCollector
		}
		kept[name] = response
	}
	return Collector{area, kept}, nil
}

// Gather is what each channel counts of the light arriving. For every
// band it takes the power arriving on each square metre times the area
// times the fraction the channel counts. Those are summed over the
// bands.
//
// arriving is power per square metre in each band. bands.Light carries
// no unit of its own, so that is said here and not by a type. Whether
// ninjafilm wants its own type for light per square metre is open.
//
//ninja:vow practice slayer's camera reads its collector here; tests only
func (collector Collector) Gather(arriving bands.Light) (Reading, error) {
	first, last := arriving.Span()
	counted := make(map[string]maths.Power, len(collector.channels))
	for name, response := range collector.channels {
		var sum float64
		for offset := range int(last-first) + inclusive {
			band := first + bands.Band(offset)
			sum += arriving.Power(band) * response.of(band)
		}
		power, err := maths.Watts(sum * collector.area.SquareMetres())
		if err != nil {
			return Reading{}, err
		}
		counted[name] = power
	}
	return Reading{counted}, nil
}

// Reading is what each channel of a collector counted, by name.
//
//ninja:vow practice slayer's camera holds one; tests only so far
type Reading struct {
	counted map[string]maths.Power
}

// Channel is what the named channel counted, and whether the collector
// has a channel by that name.
//
//ninja:vow practice slayer's camera reads a channel here; tests only so far
func (reading Reading) Channel(name string) (maths.Power, bool) {
	power, ok := reading.counted[name]
	return power, ok
}
