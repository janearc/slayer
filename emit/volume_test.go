package emit

import (
	"github.com/janearc/celestial/bands"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func TestFiniteVolumeLightHasNoDarkSourceCell(t *testing.T) {
	source, err := NewVolume(Lit{At: maths.Vec{}, Strength: maths.Grey(1)}, measure.Metres(2))
	if err != nil {
		t.Fatal(err)
	}
	for _, distance := range []float64{0, 0.1, 0.5, 1, 2, 4} {
		way, far, light := source.Light(maths.V(0, -distance, 0), maths.YAxis)
		if !(light.R > 0) || math.IsInf(light.R, 0) || math.IsNaN(light.R) ||
			math.IsNaN(way.Len()) || far < 0 {
			t.Fatalf("invalid light at %g", distance)
		}
	}
	if source.Radius() != measure.Metres(2) {
		t.Fatal("source radius lost")
	}
}

func TestFiniteVolumeFalloffHasItsAnalyticLimits(t *testing.T) {
	if volumeFalloff(0, 2) != 0.75 || volumeFalloff(2, 2) != 0.375 {
		t.Fatal("wrong centre or surface fluence")
	}
	near := volumeFalloff(2*(1-1e-8), 2)
	far := volumeFalloff(2*(1+1e-8), 2)
	if math.Abs(near-far) > 1e-6 {
		t.Fatal("fluence jumps at source surface")
	}
	for _, distance := range []float64{20, 200, 2000} {
		value := volumeFalloff(distance, 2) * distance * distance
		if value < 1 || value > 1.003 {
			t.Fatal("source lost its far-field strength")
		}
	}
}

func TestFiniteVolumeFalloffAgreesWithDirectVolumeQuadrature(t *testing.T) {
	// Outside the sphere the integrand is smooth. Integrate polar shells
	// directly, independently of the closed form and its numerical series.
	const rings, angles = 200, 200
	for _, distance := range []float64{1.5, 2, 4} {
		integral := 0.0
		for i := range rings {
			rho := (float64(i) + 0.5) / rings
			for j := range angles {
				mu := -1 + 2*(float64(j)+0.5)/angles
				integral += 3 * rho * rho / (distance*distance + rho*rho - 2*distance*rho*mu) / (rings * angles)
			}
		}
		if math.Abs(volumeFalloff(distance, 1)-integral) > 2e-4*integral {
			t.Fatal("finite-source integral changed")
		}
	}
}

func TestFiniteVolumeLightRejectsInvalidInputs(t *testing.T) {
	for _, radius := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := NewVolume(Lit{At: maths.Vec{}, Strength: maths.Grey(1)}, measure.Metres(radius)); err == nil {
			t.Fatal("invalid emitter radius accepted")
		}
	}
	for _, light := range []maths.Light{maths.RGB(-1, 0, 0), maths.Grey(math.NaN()), maths.Grey(math.Inf(1))} {
		if _, err := NewVolume(Lit{At: maths.Vec{}, Strength: light}, measure.Metres(1)); err == nil {
			t.Fatal("invalid strength accepted")
		}
	}
	if _, err := NewVolume(Lit{At: maths.V(math.Inf(1), 0, 0), Strength: maths.Grey(1)}, measure.Metres(1)); err == nil {
		t.Fatal("invalid location accepted")
	}
}

func TestFiniteVolumeSpectrumUsesTheSameFalloffAndKeepsOtherBandsDark(t *testing.T) {
	source, err := NewVolume(Lit{At: maths.V(1, 2, 3), Strength: maths.Grey(5),
		Spectrum: bands.Lines(map[float64]float64{550: 5})}, measure.Metres(2))
	if err != nil {
		t.Fatal(err)
	}
	for _, distance := range []float64{0, .1, 1, 2, 10} {
		at := source.Point().At.Sub(maths.YAxis.Scale(distance))
		_, _, light := source.Light(at, maths.YAxis)
		landed := source.Lands(at, maths.Band{Nanometres: 550})
		if math.Abs(landed-light.R) > 1e-12*max(1, light.R) || landed <= 0 {
			t.Fatal("band and visible falloff separated")
		}
		if source.Lands(at, maths.Band{Nanometres: 1.24}) != 0 {
			t.Fatal("volume source lit an absent band")
		}
	}
}

func TestFiniteVolumeSeriesMeetTheClosedFormContinuously(t *testing.T) {
	for _, ratio := range []float64{.25, 4} {
		left, right := volumeFalloff(ratio*(1-1e-8), 1), volumeFalloff(ratio*(1+1e-8), 1)
		if math.Abs(left-right) > 1e-6*left || left <= 0 || right <= 0 {
			t.Fatal("numerical series has a visible seam")
		}
	}
}

// The near-series remainder is bounded by 1.1e-8 relative at q=1/4.
// Check the integral, not the individual coefficients of its approximation.
func TestFiniteVolumeFalloffAgreesWithTheIntegralNearItsCentre(t *testing.T) {
	for _, ratio := range []float64{.1, .2, .24, .25} {
		exact := 1.5 * (1 + (1-ratio*ratio)/(2*ratio)*math.Log((1+ratio)/(1-ratio)))
		if math.Abs(volumeFalloff(ratio, 1)-exact) > 1.1e-8*exact {
			t.Fatal("near-source approximation exceeds its remainder bound")
		}
	}
}

func TestFiniteVolumeFalloffScalesAsAnInverseArea(t *testing.T) {
	for _, radius := range []float64{.5, 2, 10} {
		for _, ratio := range []float64{0, .1, .5, 1, 2, 4, 20} {
			got := volumeFalloff(radius*ratio, radius) * radius * radius
			want := volumeFalloff(ratio, 1)
			if math.Abs(got-want) > 1e-12*max(1, want) {
				t.Fatal("finite source lost its radius scaling")
			}
		}
	}
}

func TestFiniteVolumeFalloffKeepsItsFarLimitWithoutCancellation(t *testing.T) {
	for _, distance := range []float64{1e6, 1e9, 1e12} {
		got := volumeFalloff(distance, 1) * distance * distance
		if math.Abs(got-1) > 1e-10 {
			t.Fatal("far-source cancellation lost the light")
		}
	}
}

func TestFiniteVolumeRefusesAnExcludedInteriorOrBeam(t *testing.T) {
	for _, point := range []Lit{{Beyond: -1}, {Beyond: .1}, {Beyond: math.NaN()}, {Width: .1}, {Soft: .1}, {Width: math.NaN()}} {
		_, err := NewVolume(point, measure.Metres(1))
		if err == nil {
			t.Fatal("volume source accepted a dark interior")
		}
	}
}
