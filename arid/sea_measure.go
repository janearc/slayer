// This file measures a sea: what its bands hold, and what a ray meets and
// is shaded by once they are read, so that both can be set beside what
// the spectrum asks for.

package arid

import (
	"math"

	"github.com/janearc/slayer/oceantex"
)

const (
	// significantTimes is how many standard deviations of the height a
	// sea's significant height is: four, the definition wave buoys
	// report by (oceantex's waveheight.factor).
	significantTimes = 4
	// measureSamples is how many places a side the met surface is read
	// at, over the longest band's side. Picked: a power of two, four
	// to each of the longest band's samples.
	measureSamples = 1024
	// measurePlaces is how many places the slopes are read at. Picked:
	// enough that the mean square is steady to its third figure.
	measurePlaces = 1 << 16
	// measureStep is how many steps to a sample of the finest band the
	// met surface's slope is taken over. Picked: well inside one cell.
	measureStep = 8
	// peakRings is how many neighbouring rings are taken together when
	// the peak is looked for, so that one sea's chance does not pick
	// it. Picked.
	peakRings = 5
)

// goldenEast and goldenSouth place the slope readings: the fractions of
// the plastic number's lattice, which fills a square evenly whatever the
// count (Roberts 2018, The unreasonable effectiveness of quasirandom
// sequences: 1/p and 1/p^2 for p = 1.32471795724).
const (
	goldenEast  = 0.7548776662466927
	goldenSouth = 0.5698402909980532
)

// BandHeld is what one band of waves holds in its samples.
type BandHeld struct {
	SideMetres    float64 `json:"side_metres"`
	SpacingMetres float64 `json:"spacing_metres"`
	// HeightVariance is the mean square of its heights, in square
	// metres.
	HeightVariance float64 `json:"height_variance"`
	// SlopeVariance is the mean square of its sampled slopes, both
	// ways together; SlopeExpected is what its spectrum expects of
	// every sea of this wind.
	SlopeVariance float64 `json:"slope_variance"`
	SlopeExpected float64 `json:"slope_expected"`
}

// SeaMeasure is a sea as held and as drawn. Held is what the bands'
// samples carry. Met is the surface a ray's search meets, the bands'
// heights as Hit reads them between samples. Shaded is the slope a hit is
// mirrored and bent by where a pixel is finer than every band.
//
// Heights are significant heights in metres, four times the standard
// deviation; peaks are the length of the wave with the most variance, in
// metres; slopes are mean squares, both ways together.
type SeaMeasure struct {
	Bands            []BandHeld `json:"bands"`
	HeldHeightMetres float64    `json:"held_height_metres"`
	HeldPeakMetres   float64    `json:"held_peak_metres"`
	HeldSlope        float64    `json:"held_slope"`
	MetHeightMetres  float64    `json:"met_height_metres"`
	MetPeakMetres    float64    `json:"met_peak_metres"`
	MetSlope         float64    `json:"met_slope"`
	ShadedSlope      float64    `json:"shaded_slope"`
	// MetSkewness is the skewness of the met surface's height: nought
	// for round waves, above it where crests are peaked and troughs flat.
	MetSkewness float64 `json:"met_skewness"`
	// Breaking is the share of the open sea that is breaking (whitecap).
	Breaking float64 `json:"breaking"`
}

// Measure is the open sea as its bands hold it and as a ray meets it. A
// coast is left out: it is the open sea's waves that are measured.
func (sea *Sea) Measure() SeaMeasure {
	var measure SeaMeasure
	var variance float64
	longest := sea.bands[0]
	for index, band := range sea.bands {
		held := bandHeld(band, sea.slopes[index])
		measure.Bands = append(measure.Bands, held)
		variance += held.HeightVariance
		measure.HeldSlope += held.SlopeVariance
		if band.Extent > longest.Extent {
			longest = band
		}
	}
	measure.HeldHeightMetres = significantTimes * math.Sqrt(variance)
	measure.HeldPeakMetres = peakOf(longest.Height, longest.N,
		longest.Extent)
	met := sea.metHeights()
	measure.MetHeightMetres = significantTimes * math.Sqrt(spread(met))
	measure.MetPeakMetres = peakOf(met, measureSamples, sea.period)
	measure.MetSkewness = skewness(met)
	measure.MetSlope, measure.ShadedSlope, measure.Breaking = sea.metSlopes()
	return measure
}

// bandHeld is what a band's samples hold, beside the slope its spectrum
// expects.
func bandHeld(band *oceantex.Field, expected float64) BandHeld {
	var slope float64
	for index := range band.SlopeX {
		slope += band.SlopeX[index]*band.SlopeX[index] +
			band.SlopeY[index]*band.SlopeY[index]
	}
	return BandHeld{SideMetres: band.Extent,
		SpacingMetres:  band.Extent / float64(band.N),
		HeightVariance: spread(band.Height),
		SlopeVariance:  slope / float64(len(band.SlopeX)),
		SlopeExpected:  expected}
}

// spread is the variance of values about their mean.
func spread(values []float64) float64 {
	var sum, squares float64
	for _, value := range values {
		sum += value
		squares += value * value
	}
	count := float64(len(values))
	return squares/count - sum/count*(sum/count)
}

// peakOf is the length, in metres, of the wave that holds the most
// variance in a periodic grid of heights count a side over side metres:
// the side over the ring of whole wave numbers with the most in it,
// peakRings neighbours taken together. A flat grid has no peak and gives
// nought.
func peakOf(heights []float64, count int, side float64) float64 {
	rings := oceantex.Ringed(heights, count)
	var most float64
	var peak int
	for ring := 1; ring+peakRings <= len(rings); ring++ {
		var sum float64
		for _, held := range rings[ring : ring+peakRings] {
			sum += held
		}
		if sum > most {
			most, peak = sum, ring
		}
	}
	if !(most > 0) {
		return 0
	}
	return side / (float64(peak) + float64(peakRings-1)/2)
}

// metHeights is the open sea's height as a ray's search reads it, at
// measureSamples places a side over the longest band's side, row by row.
func (sea *Sea) metHeights() []float64 {
	spacing := sea.period / measureSamples
	heights := make([]float64, measureSamples*measureSamples)
	for row := range measureSamples {
		for column := range measureSamples {
			heights[row*measureSamples+column] = sea.waves(
				float64(column)*spacing, float64(row)*spacing)
		}
	}
	return heights
}

// metSlopes is the mean square slope, both ways together, of the surface
// a search meets, taken across measureStep-ths of the finest sample, and
// of the slope a hit is shaded by at a pixel of no width; and the share
// of the places that are breaking. Each is over measurePlaces places
// spread evenly over the longest band's side.
func (sea *Sea) metSlopes() (met, shaded, breaking float64) {
	step := sea.finest / measureStep
	for place := range measurePlaces {
		east := sea.period * fraction(float64(place)*goldenEast)
		south := sea.period * fraction(float64(place)*goldenSouth)
		eastward := (sea.waves(east+step, south) -
			sea.waves(east-step, south)) / (2 * step)
		southward := (sea.waves(east, south+step) -
			sea.waves(east, south-step)) / (2 * step)
		met += eastward*eastward + southward*southward
		along, across, _ := sea.slopeWithin(east, south, 0)
		shaded += along*along + across*across
		breaking += sea.whitecap(east, south)
	}
	return met / measurePlaces, shaded / measurePlaces,
		breaking / measurePlaces
}

// skewness is the third moment of values about their mean over the cube
// of their standard deviation, or nought for values all alike.
func skewness(values []float64) float64 {
	var sum float64
	for _, value := range values {
		sum += value
	}
	mean := sum / float64(len(values))
	var squares, cubes float64
	for _, value := range values {
		squares += (value - mean) * (value - mean)
		cubes += (value - mean) * (value - mean) * (value - mean)
	}
	if !(squares > 0) {
		return 0
	}
	count := float64(len(values))
	deviation := math.Sqrt(squares / count)
	return cubes / count / (deviation * deviation * deviation)
}

// fraction is the part of value after its whole number.
func fraction(value float64) float64 { return value - math.Floor(value) }
