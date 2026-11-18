package waves

import (
	"math"
	"math/cmplx"
	"slices"
	"testing"
)

func waterConfig() WaterConfig {
	return WaterConfig{Shape: waveShape(), WidthMetres: 2 * math.Pi,
		HeightMetres: 2 * math.Pi, DepthMetres: math.Atanh(0.5), Gravity: 8}
}

func waterMode(config WaterConfig, xMode, yMode int) State {
	shape := config.Shape
	state := State{Value: make([]complex128, shape.Width*shape.Height), Rate: make([]complex128, shape.Width*shape.Height)}
	for y := range shape.Height {
		for x := range shape.Width {
			phase := 2 * math.Pi * (float64(xMode*x)/float64(shape.Width) + float64(yMode*y)/float64(shape.Height))
			state.Value[y*shape.Width+x] = cmplx.Exp(complex(0, phase))
		}
	}
	return state
}

func waterAnalytic(t *testing.T, config WaterConfig, x, y int, omega, seconds float64) {
	t.Helper()
	initial := waterMode(config, x, y)
	field, err := NewWater(config, initial)
	if err != nil || field == nil {
		t.Fatalf("NewWater: %v", err)
	}
	got, err := field.At(seconds)
	if err != nil {
		t.Fatal(err)
	}
	want := State{Value: make([]complex128, len(initial.Value)), Rate: make([]complex128, len(initial.Rate))}
	for i, value := range initial.Value {
		want.Value[i] = value * complex(math.Cos(omega*seconds), 0)
		want.Rate[i] = value * complex(-omega*math.Sin(omega*seconds), 0)
	}
	requireWave(t, waterScaleRate(got, omega), waterScaleRate(want, omega))
}

// tanh(depth)=1/2, g=8 and unit wavenumber give omega=2.
func TestWaterIntermediateDepthHasResolvedAnalyticFrequency(t *testing.T) {
	waterAnalytic(t, waterConfig(), 1, 0, 2, 0.375)
}

// anti-test of TestWaterIntermediateDepthHasResolvedAnalyticFrequency
// Reversing the spatial mode cannot change an isotropic gravity frequency.
func TestWaterOppositeModeHasSameGravityFrequency(t *testing.T) {
	waterAnalytic(t, waterConfig(), -1, 0, 2, 0.375)
}

// In deep water a unit wavenumber and g=9 give omega=3.
func TestWaterDeepLimitMatchesIndependentFrequency(t *testing.T) {
	conf := waterConfig()
	conf.DepthMetres = 1e6
	conf.Gravity = 9
	waterAnalytic(t, conf, 1, 0, 3, 0.25)
}

// anti-test of TestWaterDeepLimitMatchesIndependentFrequency
// A long wave travels at sqrt(g*depth), rather than the deep-water speed.
func TestWaterShallowLimitMatchesIndependentFrequency(t *testing.T) {
	conf := waterConfig()
	conf.DepthMetres = 1e-6
	conf.Gravity = 9
	waterAnalytic(t, conf, 1, 0, 0.003, 100)
}

// Rectangular physical periods determine each axis frequency independently.
func TestWaterUsesPhysicalAxisLengths(t *testing.T) {
	conf := waterConfig()
	conf.HeightMetres = math.Pi
	conf.DepthMetres = 1e6
	conf.Gravity = 2
	waterAnalytic(t, conf, 0, 1, 2, 0.25)
}

// anti-test of TestWaterUsesPhysicalAxisLengths
// A steady mean level is conserved without inventing a zero-mode frequency.
func TestWaterMeanSurfaceHeightRemainsFixed(t *testing.T) {
	conf := waterConfig()
	initial := waterMode(conf, 0, 0)
	for i := range initial.Value {
		initial.Value[i] = 0.125
	}
	field, err := NewWater(conf, initial)
	if err != nil || field == nil {
		t.Fatalf("NewWater: %v", err)
	}
	got, err := field.At(100)
	if err != nil {
		t.Fatal(err)
	}
	requireWave(t, got, initial)
}

// A bounded zero-mean initial rate contributes sine evolution independently
// of the initial height, which may vanish.
func TestWaterRateExcitesAStandingWave(t *testing.T) {
	conf := waterConfig()
	initial := waterMode(conf, 1, 0)
	initial.Value, initial.Rate = initial.Rate, initial.Value
	before := slices.Clone(initial.Rate)
	field, err := NewWater(conf, initial)
	if err != nil || field == nil {
		t.Fatalf("NewWater: %v", err)
	}
	got, err := field.At(0.375)
	if err != nil {
		t.Fatal(err)
	}
	want := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	for i, value := range before {
		want.Value[i] = value * complex(math.Sin(0.75)/2, 0)
		want.Rate[i] = value * complex(math.Cos(0.75), 0)
	}
	requireWave(t, got, want)
	if !slices.Equal(initial.Rate, before) {
		t.Fatal("water construction mutated caller rate")
	}
}

// anti-test of TestWaterRateExcitesAStandingWave
// Uniform surface rise requires a mass source, absent from this model.
func TestWaterRefusesNonzeroMeanSurfaceRate(t *testing.T) {
	conf := waterConfig()
	initial := waterMode(conf, 0, 0)
	for i := range initial.Rate {
		initial.Rate[i] = 0.25
	}
	field, err := NewWater(conf, initial)
	if err == nil || field != nil {
		t.Fatal("unforced periodic water accepted a mean surface rate")
	}
}

// Extreme finite depth and gravity do not justify overflowing an
// intermediate g*k or underflowing k*depth before their bounded result.
func TestWaterDispersionAvoidsIntermediateOverflowAndUnderflow(t *testing.T) {
	shallow := waterConfig()
	shallow.WidthMetres = 1e308
	shallow.HeightMetres = 1e308
	shallow.DepthMetres = 1e-308
	shallow.Gravity = 1e308
	t.Run("shallow", func(t *testing.T) { waterAnalytic(t, shallow, 1, 0, 2*math.Pi/1e308, 1e307) })
	deep := waterConfig()
	deep.WidthMetres = 2 * math.Pi * 1e-200
	deep.HeightMetres = deep.WidthMetres
	deep.DepthMetres = 1e200
	deep.Gravity = 1e308
	t.Run("deep", func(t *testing.T) { waterAnalytic(t, deep, 1, 0, 1e254, 1e-254) })
}

// anti-test of TestWaterDispersionAvoidsIntermediateOverflowAndUnderflow
// Invalid units, configuration and derived modes are errors, not clamping.
func TestWaterRefusesInvalidUnitsAndGrid(t *testing.T) {
	for _, which := range []string{"width", "height", "depth", "gravity", "nan", "inf", "shape", "budget", "mode overflow"} {
		t.Run(which, func(t *testing.T) {
			conf := waterConfig()
			initial := waterMode(conf, 1, 0)
			switch which {
			case "width":
				conf.WidthMetres = 0
			case "height":
				conf.HeightMetres = -1
			case "depth":
				conf.DepthMetres = 0
			case "gravity":
				conf.Gravity = -1
			case "nan":
				conf.DepthMetres = math.NaN()
			case "inf":
				conf.WidthMetres = math.Inf(1)
			case "shape":
				conf.Shape.Width = 3
			case "budget":
				conf.Shape.MaxCells = 1
			case "mode overflow":
				conf.WidthMetres = math.SmallestNonzeroFloat64
			}
			field, err := NewWater(conf, initial)
			if err == nil || field != nil {
				t.Fatal("invalid water configuration accepted")
			}
		})
	}
}

// waterScaleRate uses the analytic frequency to compare rate in height
// units. The mean mode keeps its absolute rate check; requireWave still
// rejects all non-finite components and keeps its fixed tolerance.
func waterScaleRate(state State, analyticOmega float64) State {
	if analyticOmega == 0 {
		return state
	}
	scaled := State{Value: state.Value, Rate: make([]complex128, len(state.Rate))}
	for i, value := range state.Rate {
		scaled.Rate[i] = complex(real(value)/analyticOmega, imag(value)/analyticOmega)
	}
	return scaled
}
