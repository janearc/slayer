package mie

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"
)

// Tolerances the tests hold the sphere to, each the brief's own.
const (
	rayleighTolerance = 1e-4  // defined: one part in ten thousand of Rayleigh's law
	absorbTolerance   = 1e-10 // defined: how close extinction and scattering are without absorption
	adaTolerance      = 0.02  // defined: two percent of van de Hulst's approximation
	paradoxTolerance  = 0.01  // defined: one percent of the extinction paradox's two
)

// relative is how far a is from b, as a share of b.
func relative(got, want float64) float64 {
	return math.Abs(got-want) / math.Abs(want)
}

// rayleighScattering is Rayleigh's scattering efficiency of a small
// sphere, (8/3) x^4 |(m^2 - 1) / (m^2 + 2)|^2: an independent formula.
func rayleighScattering(size float64, index complex128) float64 {
	square := index * index
	lorentz := cmplx.Abs((square - 1) / (square + 2))
	return 8.0 / 3.0 * math.Pow(size, 4) * lorentz * lorentz // defined: Rayleigh's law
}

// anomalousDiffraction is van de Hulst's extinction efficiency for a
// sphere whose index is near one: an independent formula, not a series.
func anomalousDiffraction(size, index float64) float64 {
	phase := 2 * size * (index - 1)      // defined: the phase shift through the sphere's middle
	return 2 - 4/phase*math.Sin(phase) + // defined: van de Hulst's formula
		4/(phase*phase)*(1-math.Cos(phase)) // defined: van de Hulst's formula
}

// must is a sphere's efficiencies, the test stopped where it refuses.
func must(t *testing.T, size float64, index complex128) efficiencies {
	t.Helper()
	got, err := sphere(size, index)
	if err != nil {
		t.Fatalf("size %g index %v: %v", size, index, err)
	}
	return got
}

// A very small sphere scatters as Rayleigh's law says, and hardly
// leans forward.
func TestSmallSphereScattersAsRayleighSaid(t *testing.T) {
	const size = 0.01 // picked: far inside Rayleigh's range, as the brief asks
	index := complex(1.44, 0)
	got := must(t, size, index)
	if miss := relative(got.Scattering, rayleighScattering(size, index)); miss > rayleighTolerance {
		t.Errorf("scattering %g is %g off Rayleigh's law", got.Scattering, miss)
	}
	if got.Asymmetry >= 1e-3 { // defined: the brief's bound on a small sphere's g
		t.Errorf("asymmetry %g, a small sphere should not lean", got.Asymmetry)
	}
}

// Rayleigh's law is no good for a sphere that is not small, and such a
// sphere leans forward: the test above would notice if it were not true.
func TestRayleighLawFailsForABiggerSphere(t *testing.T) {
	const size = 0.8 // picked: a size where Rayleigh's law is known to fail
	index := complex(1.44, 0)
	got := must(t, size, index)
	if miss := relative(got.Scattering, rayleighScattering(size, index)); miss < 100*rayleighTolerance {
		t.Errorf("x = %g still follows Rayleigh's law, off by %g", size, miss)
	}
	if got.Asymmetry < 0.1 { // picked: a sphere this size leans forward by far more than this
		t.Errorf("asymmetry %g at x = %g", got.Asymmetry, size)
	}
}

// Without absorption every bit of extinguished light is scattered.
func TestWithoutAbsorptionExtinctionIsScattering(t *testing.T) {
	for _, index := range []float64{1.2, 1.44, 2.0} { // picked: water-ish, acid, and a high index
		for _, size := range []float64{0.1, 1, 5, 20, 100, 200} { // picked: small to large
			got := must(t, size, complex(index, 0))
			if miss := relative(got.Extinction, got.Scattering); miss > absorbTolerance {
				t.Errorf("m %g x %g: extinction %g scattering %g", index, size,
					got.Extinction, got.Scattering)
			}
		}
	}
}

// With absorption the extinction is more than the scattering.
func TestWithAbsorptionExtinctionIsMoreThanScattering(t *testing.T) {
	got := must(t, 5, complex(1.44, 0.01))    // picked: slightly absorbing, mid-size
	if got.Extinction-got.Scattering < 1e-3 { // picked: far over rounding
		t.Errorf("extinction %g scattering %g", got.Extinction, got.Scattering)
	}
}

// For an index near one, extinction is van de Hulst's, whose formula
// has nothing in common with the series.
func TestExtinctionNearIndexOneIsAnomalousDiffraction(t *testing.T) {
	for _, size := range []float64{200, 500, 1000} { // defined: the brief's three sizes
		got := must(t, size, complex(1.01, 0)) // defined: the brief's index
		want := anomalousDiffraction(size, 1.01)
		if miss := relative(got.Extinction, want); miss > adaTolerance {
			t.Errorf("x %g: %g against van de Hulst's %g", size, got.Extinction, want)
		}
	}
}

// The approximation is not simply two: at these sizes it is far from
// the paradox's limit, so agreeing with it says something.
func TestAnomalousDiffractionIsNotTheParadoxLimit(t *testing.T) {
	for _, size := range []float64{200, 500, 1000} { // defined: the brief's three sizes
		got := must(t, size, complex(1.01, 0))
		if relative(got.Extinction, 2) < adaTolerance {
			t.Errorf("x %g: extinction %g is within %g of two", size,
				got.Extinction, adaTolerance)
		}
	}
}

// A large sphere takes out twice the light it blocks: the extinction
// paradox.
func TestLargeSphereExtinguishesTwiceItsShadow(t *testing.T) {
	got := must(t, 5000, complex(1.44, 0)) // defined: the brief's size and index
	if miss := relative(got.Extinction, 2); miss > paradoxTolerance {
		t.Errorf("extinction %g, off two by %g", got.Extinction, miss)
	}
}

// A sphere about as wide as the wavelength is nowhere near two.
func TestSphereAsWideAsTheWavelengthIsNotTwo(t *testing.T) {
	got := must(t, 1, complex(1.44, 0)) // picked: one radian of size
	if got.Extinction >= 1 {            // picked: well under two
		t.Errorf("extinction %g at x = 1", got.Extinction)
	}
}

// Averaged over a population, the asymmetry rises with size.
func TestAsymmetryRisesWithSizeOverAPopulation(t *testing.T) {
	const wavelength = 550e-9 // picked: the middle of the visible
	last := -1.0
	for _, radius := range []float64{0.02e-6, 0.05e-6, 0.1e-6, 0.2e-6} { // picked: x = 0.2 to 2.3
		population, err := NewPopulation(radius, 1.3) // picked: the width this tree takes by default
		if err != nil {
			t.Fatal(err)
		}
		got, err := population.Average(wavelength, complex(1.44, 0))
		if err != nil {
			t.Fatal(err)
		}
		if got.Asymmetry <= last {
			t.Errorf("radius %g: asymmetry %g, after %g", radius, got.Asymmetry, last)
		}
		last = got.Asymmetry
	}
}

// One sphere's asymmetry is not monotone: it oscillates with size, so
// only the average can be tested above.
func TestOneSpheresAsymmetryOscillates(t *testing.T) {
	last, fell := 0.0, false
	for size := 20.0; size < 40; size += 0.37 { // picked: a scan fine enough to see the ripple
		got := must(t, size, complex(1.44, 0))
		fell = fell || got.Asymmetry < last
		last = got.Asymmetry
	}
	if !fell {
		t.Error("a single sphere's asymmetry rose at every step of 0.37 from 20 to 40")
	}
}

// A population of one width, very narrow, is one sphere.
func TestNarrowPopulationIsOneSphere(t *testing.T) {
	const radius, wavelength = 0.3e-6, 550e-9 // picked: x about 3.4
	index := complex(1.44, 0)
	population, err := NewPopulation(radius, 1.0001) // picked: far narrower than any cloud
	if err != nil {
		t.Fatal(err)
	}
	got, err := population.Average(wavelength, index)
	if err != nil {
		t.Fatal(err)
	}
	one := must(t, 2*math.Pi*radius/wavelength, index)
	area := math.Pi * radius * radius
	for name, pair := range map[string][2]float64{
		"extinction": {got.Extinction, one.Extinction * area},
		"albedo":     {got.Albedo, one.Scattering / one.Extinction},
		"asymmetry":  {got.Asymmetry, one.Asymmetry},
		"geometric":  {got.Geometric, area},
	} {
		if miss := relative(pair[0], pair[1]); miss > 1e-3 { // picked: the narrow width's own rounding
			t.Errorf("%s: %g against a sphere's %g", name, pair[0], pair[1])
		}
	}
}

// A wide population is not one sphere: its mean cross section is
// larger than that of the sphere of the mean radius.
func TestWidePopulationIsNotOneSphere(t *testing.T) {
	const radius = 0.3e-6                         // picked: as above
	population, err := NewPopulation(radius, 1.5) // picked: wider than any here
	if err != nil {
		t.Fatal(err)
	}
	got, err := population.Average(550e-9, complex(1.44, 0))
	if err != nil {
		t.Fatal(err)
	}
	logged := math.Log(1.5)
	want := math.Pi * radius * radius * math.Exp(2*math.Pow(logged, 2)) // defined: the log-normal's mean of r squared
	if miss := relative(got.Geometric, want); miss > 1e-3 {             // picked: the quadrature's own error
		t.Errorf("geometric %g against the log-normal's %g", got.Geometric, want)
	}
	if relative(got.Geometric, math.Pi*radius*radius) < 0.1 { // picked: a visible difference
		t.Errorf("geometric %g is the mean radius's own", got.Geometric)
	}
}

// Every refusal, by name.
func TestRefusals(t *testing.T) {
	for name, each := range map[string]struct {
		size  float64
		index complex128
		why   error
	}{
		"nought":          {0, 1.44, errSize},
		"negative":        {-1, 1.44, errSize},
		"under the limit": {minSize / 2, 1.44, errSize},
		"not a number":    {math.NaN(), 1.44, errSize},
		"over the limit":  {maxSize * 1.01, 1.44, errSize},
		"gain":            {1, complex(1.44, -0.1), errIndex},
		"no real part":    {1, complex(0, 0.1), errIndex},
		"negative real":   {1, complex(-1.44, 0), errIndex},
		"the medium":      {1, 1, errIndex},
		"index not a num": {1, complex(math.NaN(), 0), errIndex},
	} {
		if _, err := sphere(each.size, each.index); !errors.Is(err, each.why) {
			t.Errorf("%s: %v, not %v", name, err, each.why)
		}
	}
	for name, each := range map[string]struct{ radius, spread float64 }{
		"radius nought":   {0, 1.3},
		"radius negative": {-1e-6, 1.3},
		"spread one":      {1e-6, 1},
		"spread under":    {1e-6, 0.9},
		"spread nan":      {1e-6, math.NaN()},
	} {
		if _, err := NewPopulation(each.radius, each.spread); !errors.Is(err, errPopulation) {
			t.Errorf("%s: %v", name, err)
		}
	}
	population, err := NewPopulation(1e-6, 1.3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := population.Average(0, 1.44); !errors.Is(err, errPopulation) {
		t.Errorf("wavelength nought: %v", err)
	}
	if _, err := population.Average(550e-9, complex(1.44, -1)); !errors.Is(err, errIndex) {
		t.Errorf("a population with gain: %v", err)
	}
	huge, err := NewPopulation(1e-2, 1.3) // picked: a centimetre, a raindrop's hundredfold
	if err != nil {
		t.Fatal(err)
	}
	if _, err := huge.Average(550e-9, complex(1.44, 0)); !errors.Is(err, errSize) {
		t.Errorf("a population of centimetres: %v", err)
	}
}

// What is allowed is allowed: the edges of what sphere takes, a sphere
// at the smallest size still keeps to Rayleigh's law.
func TestEdgesOfWhatIsAllowed(t *testing.T) {
	for name, each := range map[string]struct {
		size  float64
		index complex128
	}{
		"the largest":      {maxSize, 1.44},
		"the smallest":     {minSize, 1.44},
		"barely absorbing": {1, complex(1.44, 1e-12)},
		"index near one":   {1, 1.0000001},
		"high absorption":  {1, complex(1.44, 10)},
	} {
		got, err := sphere(each.size, each.index)
		if err != nil || math.IsNaN(got.Extinction) || math.IsNaN(got.Asymmetry) {
			t.Errorf("%s: %+v, %v", name, got, err)
		}
	}
	smallest := must(t, minSize, 1.44)
	if miss := relative(smallest.Scattering, rayleighScattering(minSize, 1.44)); miss > 1e-3 { // picked: the cancellation's few parts in 1e7, with room
		t.Errorf("at the smallest size scattering is %g off Rayleigh's law", miss)
	}
}
