package mie

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
)

// MaxSize is the largest size parameter Sphere takes. Picked: a cloud
// droplet a hundred micrometres across is about 600 at visible light,
// and 20,000 is far beyond any cloud this package is for; past it the
// series is long and the recurrence's rounding has grown.
const MaxSize = 20000.0

// MinSize is the smallest size parameter Sphere takes. Picked: the
// upward recurrence works psi of order one as sin(x)/x less cos(x), the
// difference of two numbers near one, which is x squared over three; at
// 1e-4 that is 3e-9 against the 1e-16 the subtraction can hold, so the
// coefficient is good to about 1e-7, and below 1e-6 it is not good at
// all.
//
// A sphere of 1e-4 is a few picometres across at visible light, and is
// Rayleigh's law to any precision this package is asked for.
const MinSize = 1e-4

// ErrSize is a size parameter that is under MinSize, or over MaxSize, or
// is not a number.
var ErrSize = errors.New("mie: the size parameter is refused")

// ErrIndex is a refractive index that gains light (a negative imaginary
// part), has no real part above nought, is not a number, or is one: the
// medium's own, which is no sphere at all.
var ErrIndex = errors.New("mie: the refractive index is refused")

// The series' stopping rule and its start, as Bohren and Huffman and
// Wiscombe give them.
const (
	cubeRootTerm   = 4.05 // defined: the coefficient of x^(1/3) in the stopping rule, Bohren and Huffman's (Wiscombe's is 4)
	spareTerms     = 2    // defined: the stopping rule's constant
	downwardMargin = 15   // defined: terms above the series' end that the downward recurrence starts from, Bohren and Huffman's
)

// Efficiencies is what one sphere does to light: the shares of the
// light that falls on its cross section that it takes out of the beam
// and that it scatters, and how far forward the scattering leans.
type Efficiencies struct {
	// Extinction is Q_ext, the beam's loss to scattering and
	// absorption, over the sphere's geometric cross section.
	Extinction float64
	// Scattering is Q_sca, the part of that which is scattered.
	Scattering float64
	// Asymmetry is g, the mean cosine of the scattering angle: nought
	// for an even scatter, towards one for the forward.
	Asymmetry float64
}

// Sphere is the efficiencies of a homogeneous sphere whose size
// parameter is x = 2 pi r / lambda, with r its radius and lambda the
// wavelength in the medium, and whose complex refractive index is
// relative to the medium's. It refuses a size under MinSize or over
// MaxSize, or not a number, and an index with a negative imaginary part, a
// real part not above nought, or equal to one.
func Sphere(size float64, index complex128) (Efficiencies, error) {
	if err := check(size, index); err != nil {
		return Efficiencies{}, err
	}
	terms := int(size + cubeRootTerm*math.Cbrt(size) + spareTerms)
	return series(size, index, logDerivatives(complex(size, 0)*index, terms)), nil
}

// check is why a sphere is refused, if it is.
func check(size float64, index complex128) error {
	if !(size >= MinSize && size <= MaxSize) {
		return fmt.Errorf("%w: %g is not from %g to %g", ErrSize, size,
			MinSize, MaxSize)
	}
	if !(imag(index) >= 0) {
		return fmt.Errorf("%w: %v has a negative imaginary part, which "+
			"is a gain", ErrIndex, index)
	}
	if !(real(index) > 0) {
		return fmt.Errorf("%w: %v has a real part not above nought",
			ErrIndex, index)
	}
	if index == 1 {
		return fmt.Errorf("%w: %v is the medium's own, so there is no "+
			"sphere", ErrIndex, index)
	}
	return nil
}

// logDerivatives is D_n(y) for n from one to terms, the logarithmic
// derivative of the Riccati-Bessel function psi, by the downward
// recurrence D(n-1) = n/y - 1/(D(n) + n/y), which is stable where the
// upward is not. It starts a margin above the series' end, and above
// the size of y if that is longer, from nought.
func logDerivatives(argument complex128, terms int) []complex128 {
	start := max(terms, int(cmplx.Abs(argument))) + downwardMargin
	derivatives := make([]complex128, terms+1)
	run := complex128(0)
	for order := start; order >= 1; order-- {
		ratio := complex(float64(order), 0) / argument
		run = ratio - 1/(run+ratio)
		if order-1 <= terms {
			derivatives[order-1] = run
		}
	}
	return derivatives
}

// riccati is the Riccati-Bessel functions psi and chi of one order and
// the one before, by the upward recurrence of Bohren and Huffman. Their
// xi is psi less i chi.
type riccati struct{ psi, psiBefore, chi, chiBefore float64 }

// firstRiccati is the functions at order nought and the one before it.
func firstRiccati(size float64) riccati {
	return riccati{psi: math.Sin(size), psiBefore: math.Cos(size),
		chi: math.Cos(size), chiBefore: -math.Sin(size)}
}

// next is the functions of one order above these, the order given.
func (functions riccati) next(order, size float64) riccati {
	lift := (2*order - 1) / size // defined: the recurrence's (2n - 1) / x
	return riccati{psi: lift*functions.psi - functions.psiBefore,
		psiBefore: functions.psi, chi: lift*functions.chi - functions.chiBefore,
		chiBefore: functions.chi}
}

// xi is psi less i chi, of the order the functions are at.
func (functions riccati) xi() complex128 {
	return complex(functions.psi, -functions.chi)
}

// coefficients are a_n and b_n of one order, from the functions of that
// order and of the one before, and D_n.
func coefficients(order, size float64, index, derivative complex128,
	before, now riccati) (complex128, complex128) {
	over := complex(order/size, 0)
	psi, psiBefore := complex(now.psi, 0), complex(before.psi, 0)
	first := derivative/index + over
	second := index*derivative + over
	xi, xiBefore := now.xi(), before.xi()
	return (first*psi - psiBefore) / (first*xi - xiBefore),
		(second*psi - psiBefore) / (second*xi - xiBefore)
}

// sums are the series' running totals: those of extinction, of
// scattering, and of what the asymmetry is made of, with the last
// order's coefficients, which the next one pairs with.
type sums struct {
	extinction, scattering, product float64
	a, b                            complex128
}

// add takes in the coefficients of one order: Bohren and Huffman's
// series for Q_ext, Q_sca and g, appendix A and 4.62 of the book.
func (total *sums) add(order float64, a, b complex128) {
	weight := 2*order + 1 // defined: the series' (2n + 1)
	total.extinction += weight * real(a+b)
	total.scattering += weight * (real(a*cmplx.Conj(a)) + real(b*cmplx.Conj(b)))
	across := (order - 1) * (order + 1) / order // defined: the series' pairing of order n - 1 with n
	total.product += across*real(total.a*cmplx.Conj(a)+total.b*cmplx.Conj(b)) +
		weight/(order*(order+1))*real(a*cmplx.Conj(b))
	total.a, total.b = a, b
}

// efficiencies are the totals over the size parameter: Q is the sum
// over x squared and g the product's sum over the scattering's, twice.
// Scattering is held to the extinction: it cannot be more, and where
// nothing absorbs the two sums differ by rounding, which would
// otherwise make an albedo of one and a few parts in 1e16.
func (total sums) efficiencies(size float64) Efficiencies {
	scale := 2 / (size * size) // defined: Bohren and Huffman's 2 / x^2
	return Efficiencies{Extinction: scale * total.extinction,
		Scattering: min(scale*total.scattering, scale*total.extinction),
		Asymmetry:  2 * total.product / total.scattering} // defined: g = 4 sum / (x^2 Q_sca), Q_sca = 2 sum / x^2
}

// series works the sphere's efficiencies from its logarithmic
// derivatives, term by term.
func series(size float64, index complex128, derivatives []complex128) Efficiencies {
	var total sums
	before := firstRiccati(size)
	now := before
	for order := 1; order < len(derivatives); order++ {
		before, now = now, now.next(float64(order), size)
		a, b := coefficients(float64(order), size, index, derivatives[order],
			before, now)
		total.add(float64(order), a, b)
	}
	return total.efficiencies(size)
}
