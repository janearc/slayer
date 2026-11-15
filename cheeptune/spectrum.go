package cheeptune

import (
	"math"
	"math/cmplx"
)

// The spectrum of a stretch of sound: how strong each frequency is. The
// studio's tests are written against it, since nobody who writes an
// instrument here can hear it, and the instrument sheet is drawn from it.

// spectrum is the strength of each frequency in a stretch of samples,
// from nought to half the rate.
type spectrum struct {
	strength []float64 // one a bin; a full-scale sine reads about one
	rate     float64
	size     int
}

// newSpectrum measures a stretch through a Hann window. The stretch is
// cut or padded with silence to the power of two at or above its length.
func newSpectrum(x []float32, rate int) *spectrum {
	size := 1
	for size < len(x) {
		size *= 2
	}
	bins := make([]complex128, size)
	window := 0.0
	for i, v := range x {
		w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(x)))
		bins[i] = complex(float64(v)*w, 0)
		window += w
	}
	transform(bins)
	out := &spectrum{rate: float64(rate), size: size,
		strength: make([]float64, size/2)}
	for i := range out.strength {
		out.strength[i] = 2 * cmplx.Abs(bins[i]) / window
	}
	return out
}

// transform is the fast Fourier transform, in place, of a length that is
// a power of two: Cooley and Tukey's, bit-reversed then butterflied.
func transform(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for span := 2; span <= n; span *= 2 {
		turn := cmplx.Rect(1, -2*math.Pi/float64(span))
		for start := 0; start < n; start += span {
			w := complex(1, 0)
			for k := range span / 2 {
				a, b := x[start+k], x[start+k+span/2]*w
				x[start+k], x[start+k+span/2] = a+b, a-b
				w *= turn
			}
		}
	}
}

// hertz is the frequency a bin sits at.
func (s *spectrum) hertz(bin int) float64 {
	return float64(bin) * s.rate / float64(s.size)
}

// bin is the bin nearest a frequency, held inside the spectrum.
func (s *spectrum) bin(hertz float64) int {
	at := int(math.Round(hertz * float64(s.size) / s.rate))
	return max(0, min(len(s.strength)-1, at))
}

// near is the strongest bin within a width either side of a frequency:
// its strength and where exactly it is.
func (s *spectrum) near(hertz, width float64) (strength, at float64) {
	best := s.bin(hertz - width)
	for i := best; i <= s.bin(hertz+width); i++ {
		if s.strength[i] > s.strength[best] {
			best = i
		}
	}
	return s.strength[best], s.peak(best)
}

// peak is where a peak at a bin really lies, in hertz, from a parabola
// through the logs of the bin and its two neighbours.
func (s *spectrum) peak(bin int) float64 {
	if bin <= 0 || bin >= len(s.strength)-1 {
		return s.hertz(bin)
	}
	a := math.Log(s.strength[bin-1] + spectrumFloor)
	b := math.Log(s.strength[bin] + spectrumFloor)
	c := math.Log(s.strength[bin+1] + spectrumFloor)
	if a-2*b+c == 0 {
		return s.hertz(bin)
	}
	shift := 0.5 * (a - c) / (a - 2*b + c)
	return (float64(bin) + shift) * s.rate / float64(s.size)
}

// spectrumFloor keeps a log off nought. Picked: far under any sample a
// sixteen-bit file can hold.
const spectrumFloor = 1e-12

// most is the greatest strength between two frequencies.
func (s *spectrum) most(from, to float64) float64 {
	best := 0.0
	for i := s.bin(from); i <= s.bin(to); i++ {
		best = math.Max(best, s.strength[i])
	}
	return best
}

// decibels is a ratio of strengths in decibels.
func decibels(ratio float64) float64 {
	return 20 * math.Log10(ratio+spectrumFloor)
}
