package job

// Shutter is a film's motion blur: how much of each frame's interval the
// shutter is open, and how many times a frame is drawn while it is.
type Shutter struct {
	wire shutterWire
}

// Shutter is the job's shutter, and whether it says one; without one, a
// frame is one instant.
func (job Job) Shutter() (Shutter, bool) {
	if job.wire.Shutter == nil {
		return Shutter{}, false
	}
	return Shutter{*job.wire.Shutter}, true
}

// Angle is how much of each frame's interval the shutter is open, in
// degrees; 360 is all of it.
func (open Shutter) Angle() float64 { return open.wire.Angle.Value }

// Glimpses is how many times each frame is drawn while the shutter is
// open, the light averaged before development.
func (open Shutter) Glimpses() int { return open.wire.Glimpses }

// Print is plain development: the tone curve from light to picture.
type Print struct {
	wire printWire
}

// Print is the job's print, and whether it says one.
func (job Job) Print() (Print, bool) {
	if job.wire.Print == nil {
		return Print{}, false
	}
	return Print{*job.wire.Print}, true
}

// Knee is where the curve's shoulder starts, as a share of white.
func (curve Print) Knee() float64 { return curve.wire.Knee }

// Top is the brightest light the print holds, in whites.
func (curve Print) Top() float64 { return curve.wire.Top }

// Bloom is how far bright light spreads, in pixels, and whether the job
// said; unsaid, it is none.
func (curve Print) Bloom() (float64, bool) { return pointed(curve.wire.Bloom) }

// Stops is the exposure given at the print, and whether the job said;
// unsaid, it is none.
func (curve Print) Stops() (float64, bool) { return pointed(curve.wire.Stops) }

// pointed is the number a pointer holds, and whether there is one.
func pointed[Number int64 | float64](value *Number) (Number, bool) {
	if value == nil {
		return nothing, false
	}
	return *value, true
}

// Stock is development through a ninjafilm stock.
type Stock struct {
	wire stockWire
}

// Stock is the job's stock, and whether it says one.
func (job Job) Stock() (Stock, bool) {
	if job.wire.Stock == nil {
		return Stock{}, false
	}
	return Stock{*job.wire.Stock}, true
}

// Name is the stock's name, one of the shelf.
func (film Stock) Name() string { return film.wire.Name }

// Seed is the seed that fixes the grain, and whether the job said;
// unsaid, it is zero.
func (film Stock) Seed() (int64, bool) { return pointed(film.wire.Seed) }

// Rated is the push or pull in stops, and whether the job said; unsaid,
// it is none.
func (film Stock) Rated() (float64, bool) { return pointed(film.wire.Rated) }
