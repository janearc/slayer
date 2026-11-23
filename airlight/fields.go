package airlight

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/skylight"
)

// Fields is an air's light at its ground, at each wavelength of a plan:
// for each bin of the sun's zenith, the view's cosine from up and the
// azimuth from the sun, the radiance arriving there per unit sunlight at
// the top, per steradian, and the crossings behind it.
//
// With it, at each step of the sun's zenith, the share of the direct
// sun that reaches the ground; the sun's irradiance at the top; the
// band each wavelength stands for; the share of the sunlight on the
// part of the planet walked that its ground absorbed; and the span of
// the sun's zenith walked for, outside which the bins hold only part of
// their light.
type Fields struct {
	wavelengths              []int
	band                     float64
	sun, absorbed            []float64
	zeniths, views, azimuths int
	radiance, counts         [][]float64
	transmittance            [][]float64
	suns                     skylight.Suns
}

// newFields is fields of a plan's wavelengths, grid and suns, holding no
// light yet.
func newFields(plan Plan) *Fields {
	waves := len(plan.wavelengths)
	return &Fields{wavelengths: slices.Clone(plan.wavelengths),
		suns: plan.suns, band: plan.band, sun: slices.Clone(plan.sun),
		absorbed: make([]float64, waves), zeniths: plan.grid.Zeniths,
		views: plan.grid.Views, azimuths: plan.grid.Azimuths,
		radiance: make([][]float64, waves), counts: make([][]float64, waves),
		transmittance: make([][]float64, waves)}
}

// keep is a wavelength's field, read at the ground at each bin's centre,
// where skylight's reading is that bin's own, and its budget's share for
// the ground.
func (fields *Fields) keep(wave int, field *skylight.Field,
	spent skylight.Budget) {
	size := fields.zeniths * fields.views * fields.azimuths
	radiance, counts := make([]float64, none, size), make([]float64, none, size)
	for zenith := range fields.zeniths {
		sunZenith := centre(zenith, fields.zeniths) * math.Pi
		for view := range fields.views {
			cosine := centre(view, fields.views)*two - one
			for round := range fields.azimuths {
				azimuth := centre(round, fields.azimuths) * math.Pi
				radiance = append(radiance, field.Radiance(none, sunZenith,
					cosine, azimuth))
				counts = append(counts, float64(field.Count(none, sunZenith,
					cosine, azimuth)))
			}
		}
	}
	shares := make([]float64, transmittanceSteps+one)
	for step := range shares {
		shares[step] = field.Transmittance(none, stepZenith(step))
	}
	fields.radiance[wave], fields.counts[wave] = radiance, counts
	fields.transmittance[wave] = shares
	fields.absorbed[wave] = spent.Ground / spent.Launched
}

// centre is the middle of a bin of bins equal bins, as a share from
// nought to one.
func centre(bin, bins int) float64 {
	return (float64(bin) + half) / float64(bins)
}

// stepZenith is the sun's zenith, in radians, at a step of the direct
// sun's table.
func stepZenith(step int) float64 {
	return float64(step) / transmittanceSteps * math.Pi
}

// index is a bin's place in a wavelength's radiance: by the sun's
// zenith, then the view, then the azimuth.
func (fields *Fields) index(zenith, view, round int) int {
	return (zenith*fields.views+view)*fields.azimuths + round
}

// slice is a wavelength's radiance for the sun at a zenith, in radians,
// at each view and azimuth bin, view by view: read linearly between the
// centres of the two zenith bins about it, and held past the first and
// last, as skylight reads its own.
func (fields *Fields) slice(wave int, sunZenith float64) []float64 {
	at := axisOf(sunZenith/math.Pi, fields.zeniths)
	out := make([]float64, none, fields.views*fields.azimuths)
	light := fields.radiance[wave]
	for view := range fields.views {
		for round := range fields.azimuths {
			low := light[fields.index(at.low, view, round)]
			high := light[fields.index(at.high, view, round)]
			out = append(out, low*(one-at.share)+high*at.share)
		}
	}
	return out
}

// transmittanceAt is the share of the direct sun at a wavelength that
// reaches the ground with the sun at a zenith, in radians: the table's,
// read linearly between its steps.
func (fields *Fields) transmittanceAt(wave int, sunZenith float64) float64 {
	shares := fields.transmittance[wave]
	last := float64(len(shares) - one)
	place := min(max(sunZenith/math.Pi*last, none), last)
	low := min(int(place), len(shares)-two)
	share := place - float64(low)
	return shares[low]*(one-share) + shares[low+one]*share
}

// seen is light by wavelength as moonlight takes it, each wavelength's
// power over its whole band: so much a unit of sunlight at the top, as
// the function gives it, times the sun's irradiance and the band, laid
// evenly over every whole nanometre of the band it stands for.
//
// Laid over the band, not put at the wavelength as one line: moonlight's
// Picture weighs a line by the observer's mean over the bin of its own
// grid the line falls in, 24 an octave, which is narrower than 20 nm and
// not centred on it. An even spectrum given as lines came out purple.
func (fields *Fields) seen(power func(wave int) float64) moonlight.Spectrum {
	width := max(one, int(math.Round(fields.band)))
	light := make(moonlight.Spectrum, len(fields.wavelengths)*width)
	for wave, nanometres := range fields.wavelengths {
		each := power(wave) * fields.sun[wave] * fields.band / float64(width)
		for step := range width {
			light[nanometres-width/two+step] += each
		}
	}
	return light
}

// an axis is where a look falls between the centres of one axis's bins:
// the bin at or below it, the one above, and how far toward that.
type axis struct {
	low, high int
	share     float64
}

// axisOf is where a share from nought to one falls between the centres
// of bins equal bins, as skylight's own reading has it. Past the first
// or last centre it is at that one.
func axisOf(share float64, bins int) axis {
	place := min(max(share*float64(bins)-half, none), float64(bins-one))
	low := int(place)
	return axis{low: low, high: min(low+one, bins-one),
		share: place - float64(low)}
}

// folded is an azimuth taken to nought to pi: the sky either side of
// the sun's vertical is the same.
func folded(azimuth float64) float64 {
	return math.Abs(math.Remainder(azimuth, fullTurn))
}

// The places of the numbers that begin the fields' bytes: the counts of
// wavelengths, of zenith, view and azimuth bins and of the direct sun's
// steps, which are its header's counts; the band; and the span of suns.
const (
	wavesAt = iota
	zenithsAt
	viewsAt
	azimuthsAt
	stepsAt
	// headerCounts is how many of the header's numbers are counts, and
	// the place of the band after them.
	headerCounts
	// fromAt and toAt are the places of the span of suns walked for.
	fromAt
	toAt
	// headerNumbers is how many numbers the header holds: its counts,
	// the band and the span.
	headerNumbers
)

// Noise is what is behind the bins above the horizon that a sun along
// a way is read from: the median crossings a bin at the middle
// wavelength, and the colour noise they leave between neighbouring
// points of the sky's table, colourNoiseAtOne over their root.
func (fields *Fields) Noise(way maths.Vec) (float64, float64) {
	held := fields.held(readSpan(zenithOf(way), fields.zeniths))
	if !(held > none) {
		return none, math.Inf(one)
	}
	return held, colourNoiseAtOne / math.Sqrt(held)
}

// held is the median crossings a bin above the horizon, at the middle
// wavelength, over the zenith bins of a span; nought where none is
// in it.
func (fields *Fields) held(span skylight.Suns) float64 {
	counts := fields.counts[len(fields.counts)/two]
	var above []float64
	for zenith := range fields.zeniths {
		middle := centre(zenith, fields.zeniths) * math.Pi
		if middle >= span.From && middle <= span.To {
			above = append(above, counts[fields.index(zenith,
				fields.views/two, none):fields.index(zenith+one, none,
				none)]...)
		}
	}
	if len(above) == none {
		return none
	}
	slices.Sort(above)
	return above[len(above)/two]
}

// Holds is nil when the fields were walked for a sun along a way, and
// otherwise refuses it: the bins it is read between hold only part of
// their light.
func (fields *Fields) Holds(way maths.Vec) error {
	read := readSpan(zenithOf(way), fields.zeniths)
	if read.From >= fields.suns.From && read.To <= fields.suns.To {
		return nil
	}
	return fmt.Errorf("airlight: fields walked for a sun %.1f to %.1f "+
		"degrees from up are read for one %.1f degrees from up",
		fields.suns.From/maths.Degree, fields.suns.To/maths.Degree,
		zenithOf(way)/maths.Degree)
}

// perWavelength is how many single numbers each wavelength has besides
// its bins and steps: its nanometres, its sun and its ground's share.
const perWavelength = 3 // defined: the three named above

// Bytes is the fields as bytes, for the disc cache: the header, then the
// wavelengths, the sun and the ground's shares, then each wavelength's
// radiance, counts and direct sun, each number a float64's bits,
// little-endian. FieldsFromBytes reads them back the same to the bit.
func (fields *Fields) Bytes() []byte {
	numbers := make([]float64, headerNumbers)
	numbers[wavesAt] = float64(len(fields.wavelengths))
	numbers[zenithsAt], numbers[viewsAt] = float64(fields.zeniths),
		float64(fields.views)
	numbers[azimuthsAt] = float64(fields.azimuths)
	numbers[stepsAt] = float64(transmittanceSteps + one)
	numbers[headerCounts] = fields.band
	numbers[fromAt], numbers[toAt] = fields.suns.From, fields.suns.To
	for _, nanometres := range fields.wavelengths {
		numbers = append(numbers, float64(nanometres))
	}
	numbers = append(numbers, fields.sun...)
	numbers = append(numbers, fields.absorbed...)
	for wave := range fields.wavelengths {
		numbers = append(numbers, fields.radiance[wave]...)
		numbers = append(numbers, fields.counts[wave]...)
		numbers = append(numbers, fields.transmittance[wave]...)
	}
	return numberBytes(numbers)
}

// FieldsFromBytes is the fields Bytes gave, or a refusal of bytes that
// are not fields': by their length, by a header whose counts are not
// whole numbers from one to mostBins, and by a span of suns that does
// not run up from nought to pi.
func FieldsFromBytes(raw []byte) (*Fields, error) {
	if len(raw)%float64Bytes != none || len(raw) < headerNumbers*float64Bytes {
		return nil, fmt.Errorf("airlight: %d bytes are not fields", len(raw))
	}
	read := &numbers{raw: raw}
	sizes := make([]int, headerCounts)
	for index := range sizes {
		count := read.next()
		if !(count == math.Trunc(count) && count >= one && count <= mostBins) {
			return nil, fmt.Errorf("airlight: a count of %g in fields' "+
				"header", count)
		}
		sizes[index] = int(count)
	}
	waves, steps := sizes[wavesAt], sizes[stepsAt]
	bins := sizes[zenithsAt] * sizes[viewsAt] * sizes[azimuthsAt]
	want := headerNumbers + waves*(perWavelength+two*bins+steps)
	if len(raw) != want*float64Bytes {
		return nil, fmt.Errorf("airlight: fields of these counts are %d "+
			"bytes, not %d", want*float64Bytes, len(raw))
	}
	made := read.fields(sizes, bins)
	if !(made.suns.From >= none && made.suns.From <= made.suns.To &&
		made.suns.To <= math.Pi) {
		return nil, fmt.Errorf("airlight: a span of suns from %g to %g "+
			"in fields' header", made.suns.From, made.suns.To)
	}
	return made, nil
}

// numbers is bytes read as float64s, one after another.
type numbers struct{ raw []byte }

// next is the next number.
func (read *numbers) next() float64 {
	number := math.Float64frombits(binary.LittleEndian.Uint64(read.raw))
	read.raw = read.raw[float64Bytes:]
	return number
}

// take is the next so many numbers.
func (read *numbers) take(count int) []float64 {
	made := make([]float64, count)
	for index := range made {
		made[index] = read.next()
	}
	return made
}

// fields is the rest of the bytes, after the header's counts, as fields
// of those counts, each wavelength's radiance and counts of so many
// bins.
func (read *numbers) fields(sizes []int, bins int) *Fields {
	waves, steps := sizes[wavesAt], sizes[stepsAt]
	made := &Fields{band: read.next(), zeniths: sizes[zenithsAt],
		views: sizes[viewsAt], azimuths: sizes[azimuthsAt]}
	made.suns = skylight.Suns{From: read.next(), To: read.next()}
	for _, nanometres := range read.take(waves) {
		made.wavelengths = append(made.wavelengths, int(nanometres))
	}
	made.sun, made.absorbed = read.take(waves), read.take(waves)
	for range waves {
		made.radiance = append(made.radiance, read.take(bins))
		made.counts = append(made.counts, read.take(bins))
		made.transmittance = append(made.transmittance, read.take(steps))
	}
	return made
}
