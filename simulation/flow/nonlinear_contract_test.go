package flow

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/janearc/slayer/simulation/spectral"
)

// Two streamfunction modes interact into a resolved, nonzero acceleration.
// The closed form includes the pressure projection, independently of flowRHS.
func TestFlowRHSMatchesAnalyticTwoModeInteraction(t *testing.T) {
	const (
		width  = 8
		height = 16
		length = 2 * math.Pi
		a      = 0.1
		b      = 0.2
	)
	config := Config{
		Shape:          spectral.Shape{Width: width, Height: height, MaxCells: width * height},
		WidthMetres:    length,
		HeightMetres:   length,
		Viscosity:      0,
		MaxStepSeconds: 0.01,
	}
	initial := Velocity{X: make([]float64, width*height), Y: make([]float64, width*height)}
	want := Velocity{X: make([]float64, width*height), Y: make([]float64, width*height)}
	var expectedSquare float64
	for y := range height {
		for x := range width {
			phaseX := 2 * math.Pi * float64(x) / float64(width)
			phaseY := 2 * math.Pi * float64(y) / float64(height)
			index := y*width + x
			initial.X[index] = -2 * b * math.Sin(2*phaseY)
			initial.Y[index] = a * math.Sin(phaseX)
			want.X[index] = (12.0 / 5) * a * b * math.Sin(phaseX) * math.Cos(2*phaseY)
			want.Y[index] = -(6.0 / 5) * a * b * math.Cos(phaseX) * math.Sin(2*phaseY)
			expectedSquare += want.X[index]*want.X[index] + want.Y[index]*want.Y[index]
		}
	}
	expectedRMS := math.Sqrt(expectedSquare / float64(width*height))
	if expectedRMS == 0 || math.IsNaN(expectedRMS) || math.IsInf(expectedRMS, 0) {
		t.Fatalf("analytic interaction norm is not finite and nonzero: %g", expectedRMS)
	}

	got, err := flowRHS(config, initial)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.X) != width*height || len(got.Y) != width*height {
		t.Fatalf("RHS component lengths %d/%d, want %d", len(got.X), len(got.Y), width*height)
	}
	var actualSquare float64
	for index := range got.X {
		checkClose(t, got.X[index], want.X[index], 1e-12)
		checkClose(t, got.Y[index], want.Y[index], 1e-12)
		actualSquare += got.X[index]*got.X[index] + got.Y[index]*got.Y[index]
	}
	actualRMS := math.Sqrt(actualSquare / float64(width*height))
	if actualRMS == 0 || math.IsNaN(actualRMS) || math.IsInf(actualRMS, 0) {
		t.Fatalf("RHS interaction norm is not finite and nonzero: %g", actualRMS)
	}
	checkClose(t, mean(got.X), 0, 1e-12)
	checkClose(t, mean(got.Y), 0, 1e-12)

	// The unmixed self products have wavevectors (2,0) and (0,4). They
	// are grid-resolved, but their accelerations are purely longitudinal.
	uHat, err := spectral.Forward(config.Shape, complexSamples(got.X))
	if err != nil {
		t.Fatal(err)
	}
	vHat, err := spectral.Forward(config.Shape, complexSamples(got.Y))
	if err != nil {
		t.Fatal(err)
	}
	for label, coefficient := range map[string]complex128{
		"x self mode in x acceleration": uHat[2],
		"x self mode in y acceleration": vHat[2],
		"y self mode in x acceleration": uHat[4*width],
		"y self mode in y acceleration": vHat[4*width],
	} {
		if magnitude := cmplx.Abs(coefficient); magnitude > 1e-12 {
			t.Errorf("%s coefficient is %g, want projected away", label, magnitude)
		}
	}
}

// The inviscid Galerkin RHS does no instantaneous work on a resolved,
// interacting divergence-free field.
func TestInviscidFlowRHSDoesNoInstantaneousEnergyWork(t *testing.T) {
	const (
		width  = 8
		height = 16
		length = 2 * math.Pi
		a      = 0.1
		b      = 0.2
		c      = 0.05
		d      = 0.03
	)
	config := Config{
		Shape:          spectral.Shape{Width: width, Height: height, MaxCells: width * height},
		WidthMetres:    length,
		HeightMetres:   length,
		Viscosity:      0,
		MaxStepSeconds: 0.01,
	}
	initial := Velocity{X: make([]float64, width*height), Y: make([]float64, width*height)}
	var inputSquare float64
	for y := range height {
		for x := range width {
			phaseX := 2 * math.Pi * float64(x) / float64(width)
			phaseY := 2 * math.Pi * float64(y) / float64(height)
			index := y*width + x
			initial.X[index] = -2*b*math.Sin(2*phaseY) -
				c*math.Sin(phaseX+phaseY) - d*math.Cos(2*phaseX-phaseY)
			initial.Y[index] = a*math.Sin(phaseX) + c*math.Sin(phaseX+phaseY) -
				2*d*math.Cos(2*phaseX-phaseY)
			inputSquare += initial.X[index]*initial.X[index] + initial.Y[index]*initial.Y[index]
		}
	}
	inputRMS := math.Sqrt(inputSquare / float64(width*height))
	if inputRMS == 0 || math.IsNaN(inputRMS) || math.IsInf(inputRMS, 0) {
		t.Fatalf("input velocity norm is not finite and nonzero: %g", inputRMS)
	}

	derivative, err := flowRHS(config, initial)
	if err != nil {
		t.Fatal(err)
	}
	var work, derivativeSquare float64
	for index := range derivative.X {
		work += initial.X[index]*derivative.X[index] + initial.Y[index]*derivative.Y[index]
		derivativeSquare += derivative.X[index]*derivative.X[index] +
			derivative.Y[index]*derivative.Y[index]
	}
	work /= float64(width * height)
	derivativeRMS := math.Sqrt(derivativeSquare / float64(width*height))
	if derivativeRMS == 0 || math.IsNaN(derivativeRMS) || math.IsInf(derivativeRMS, 0) {
		t.Fatalf("inviscid RHS norm is not finite and nonzero: %g", derivativeRMS)
	}
	checkClose(t, work, 0, 1e-12)
	checkClose(t, mean(derivative.X), 0, 1e-12)
	checkClose(t, mean(derivative.Y), 0, 1e-12)
}

func complexSamples(values []float64) []complex128 {
	result := make([]complex128, len(values))
	for i, value := range values {
		result[i] = complex(value, 0)
	}
	return result
}
