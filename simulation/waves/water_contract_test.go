package waves

import (
	"math"
	"math/cmplx"
	"slices"
	"testing"
)

// A diagonal mode uses both physical periods in the isotropic dispersion
// relation, and its initial rate contributes independently to both fields.
func TestWaterDiagonalModeUsesCombinedWavenumberAndInitialRate(t *testing.T) {
	config := waterConfig()
	config.WidthMetres = 4 * math.Pi
	config.HeightMetres = 2 * math.Pi
	config.DepthMetres = 0.4
	config.Gravity = 9

	initial := waterMode(config, 1, 1)
	for i, value := range initial.Value {
		initial.Rate[i] = complex(0.3, -0.2) * value
	}
	before := State{Value: slices.Clone(initial.Value), Rate: slices.Clone(initial.Rate)}
	field, err := NewWater(config, initial)
	if err != nil || field == nil {
		t.Fatalf("NewWater: %v", err)
	}

	kx, ky := 2*math.Pi/config.WidthMetres, 2*math.Pi/config.HeightMetres
	k := math.Hypot(kx, ky)
	omega := math.Sqrt(config.Gravity * k * math.Tanh(k*config.DepthMetres))
	const seconds = 0.17
	got, err := field.At(seconds)
	if err != nil {
		t.Fatal(err)
	}
	want := State{Value: make([]complex128, len(initial.Value)), Rate: make([]complex128, len(initial.Rate))}
	cosine, sine := math.Cos(omega*seconds), math.Sin(omega*seconds)
	for i := range want.Value {
		want.Value[i] = before.Value[i]*complex(cosine, 0) + before.Rate[i]*complex(sine/omega, 0)
		want.Rate[i] = -before.Value[i]*complex(omega*sine, 0) + before.Rate[i]*complex(cosine, 0)
	}
	requireWave(t, got, want)
	if !slices.Equal(initial.Value, before.Value) || !slices.Equal(initial.Rate, before.Rate) {
		t.Fatal("NewWater changed caller-owned Cauchy data")
	}
}

// Roundoff-sized mean removal preserves the resolved standing-wave rate and
// caller data; a clearly larger mean still violates the periodic mass rule.
func TestWaterRemovesOnlyRoundoffRateMean(t *testing.T) {
	config := waterConfig()
	initial := waterMode(config, 1, 0)
	for y := range config.Shape.Height {
		for x := range config.Shape.Width {
			index := y*config.Shape.Width + x
			initial.Rate[index] = complex(math.Cos(2*math.Pi*float64(x)/float64(config.Shape.Width))+5e-15, 0)
		}
	}
	before := slices.Clone(initial.Rate)
	field, err := NewWater(config, initial)
	if err != nil || field == nil {
		t.Fatalf("NewWater rejected a roundoff-sized mean: %v", err)
	}
	zero, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	for y := range config.Shape.Height {
		var rateMean float64
		for x := range config.Shape.Width {
			index := y*config.Shape.Width + x
			want := math.Cos(2 * math.Pi * float64(x) / float64(config.Shape.Width))
			if cmplx.Abs(zero.Rate[index]-complex(want, 0)) > 2e-14 {
				t.Fatalf("resolved rate at (%d,%d) is %v, want %g", x, y, zero.Rate[index], want)
			}
			rateMean += real(zero.Rate[index])
		}
		if math.Abs(rateMean/float64(config.Shape.Width)) > 2e-15 {
			t.Fatalf("roundoff mean remains in row %d: %g", y, rateMean/float64(config.Shape.Width))
		}
	}
	if !slices.Equal(initial.Rate, before) {
		t.Fatal("NewWater changed caller-owned rate samples")
	}

	tooMuch := State{Value: slices.Clone(initial.Value), Rate: slices.Clone(before)}
	for i := range tooMuch.Rate {
		tooMuch.Rate[i] += 1e-3
	}
	tooMuchBefore := slices.Clone(tooMuch.Rate)
	refused, err := NewWater(config, tooMuch)
	if err == nil || refused != nil {
		t.Fatalf("NewWater accepted a material mean rate: field=%v err=%v", refused, err)
	}
	if !slices.Equal(tooMuch.Rate, tooMuchBefore) {
		t.Fatal("refused NewWater changed caller-owned rate samples")
	}
}
