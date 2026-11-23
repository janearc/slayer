package spectral

import (
	"math"
	"math/cmplx"
	"strconv"
	"testing"

	"pgregory.net/rapid"
)

const cellBudget = 256

func shape(width, height int) Shape {
	return Shape{Width: width, Height: height, MaxCells: cellBudget}
}

func direct(values []complex128, grid Shape, inverse bool) []complex128 {
	result := make([]complex128, len(values))
	sign := -1.0
	if inverse {
		sign = 1
	}
	for ky := range grid.Height {
		for kx := range grid.Width {
			var sum complex128
			for y := range grid.Height {
				for x := range grid.Width {
					turns := float64(kx*x)/float64(grid.Width) +
						float64(ky*y)/float64(grid.Height)
					angle := sign * 2 * math.Pi * turns
					sum += values[y*grid.Width+x] * cmplx.Exp(complex(0, angle))
				}
			}
			if inverse {
				sum /= complex(float64(grid.Width*grid.Height), 0)
			}
			result[ky*grid.Width+kx] = sum
		}
	}
	return result
}

func checkComplexes(test *testing.T, got, want []complex128, tolerance float64) {
	test.Helper()
	if len(got) != len(want) {
		test.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for index := range want {
		if !finite(real(got[index])) || !finite(imag(got[index])) {
			test.Errorf("value %d is non-finite: %v", index, got[index])
			continue
		}
		limit := tolerance * math.Max(1, cmplx.Abs(want[index]))
		if difference := cmplx.Abs(got[index] - want[index]); difference > limit {
			test.Errorf("value %d is %v, want %v (difference %g, limit %g)",
				index, got[index], want[index], difference, limit)
		}
	}
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func checkUnchanged(test *testing.T, before, after []complex128) {
	test.Helper()
	if len(before) != len(after) {
		test.Fatalf("input length changed from %d to %d", len(before), len(after))
	}
	for index := range before {
		if after[index] != before[index] {
			test.Errorf("input value %d changed from %v to %v",
				index, before[index], after[index])
		}
	}
}

func TestForwardMatchesDirectDFTForRectangles(test *testing.T) {
	for _, grid := range []Shape{shape(2, 2), shape(4, 2), shape(2, 8), shape(8, 4)} {
		test.Run(strconv.Itoa(grid.Width)+"x"+strconv.Itoa(grid.Height),
			func(test *testing.T) {
				values := make([]complex128, grid.Width*grid.Height)
				for index := range values {
					sine := math.Sin(float64(index*3) + 0.25)
					cosine := math.Cos(float64(index*index) + 0.75)
					values[index] = complex(
						sine,
						cosine)
				}
				before := append([]complex128(nil), values...)
				got, err := Forward(grid, values)
				if err != nil {
					test.Fatal(err)
				}
				checkComplexes(test, got, direct(values, grid, false), 1e-11)
				checkUnchanged(test, before, values)
			})
	}
}

func TestInverseMatchesDirectDFTForRectangles(test *testing.T) {
	for _, grid := range []Shape{shape(2, 4), shape(8, 2), shape(4, 8)} {
		test.Run(strconv.Itoa(grid.Width)+"x"+strconv.Itoa(grid.Height),
			func(test *testing.T) {
				values := make([]complex128, grid.Width*grid.Height)
				for index := range values {
					cosine := math.Cos(float64(index) + 0.4)
					sine := math.Sin(float64(index*index) + 0.2)
					values[index] = complex(
						cosine,
						sine)
				}
				before := append([]complex128(nil), values...)
				got, err := Inverse(grid, values)
				if err != nil {
					test.Fatal(err)
				}
				checkComplexes(test, got, direct(values, grid, true), 1e-11)
				checkUnchanged(test, before, values)
			})
	}
}

func TestImpulseAndConstantHaveExpectedForwardTerms(test *testing.T) {
	grid := shape(4, 2)
	impulse := make([]complex128, grid.Width*grid.Height)
	impulse[0] = 1
	got, err := Forward(grid, impulse)
	if err != nil {
		test.Fatal(err)
	}
	wantImpulse := make([]complex128, len(impulse))
	for index := range wantImpulse {
		wantImpulse[index] = 1
	}
	checkComplexes(test, got, wantImpulse, 1e-12)

	constant := make([]complex128, grid.Width*grid.Height)
	for index := range constant {
		constant[index] = complex(2, -3)
	}
	got, err = Forward(grid, constant)
	if err != nil {
		test.Fatal(err)
	}
	wantConstant := make([]complex128, len(constant))
	length := float64(len(constant))
	wantConstant[0] = complex(length*2,
		-length*3)
	checkComplexes(test, got, wantConstant, 1e-11)
}

func TestForwardFindsSingleFourierModeWithNegativeExponent(test *testing.T) {
	grid := shape(8, 4)
	const modeX, modeY = 3, 1
	values := make([]complex128, grid.Width*grid.Height)
	for y := range grid.Height {
		for x := range grid.Width {
			turns := float64(modeX*x)/float64(grid.Width) +
				float64(modeY*y)/float64(grid.Height)
			values[y*grid.Width+x] = cmplx.Exp(complex(0, 2*math.Pi*turns))
		}
	}
	got, err := Forward(grid, values)
	if err != nil {
		test.Fatal(err)
	}
	want := make([]complex128, len(values))
	length := float64(len(values))
	want[modeY*grid.Width+modeX] = complex(length, 0)
	checkComplexes(test, got, want, 1e-11)
}

func TestForwardInverseRoundTripAndParseval(test *testing.T) {
	grid := shape(8, 4)
	values := make([]complex128, grid.Width*grid.Height)
	for index := range values {
		sine := math.Sin(float64(index) * 0.7)
		cosine := math.Cos(float64(index) * 0.31)
		values[index] = complex(sine,
			cosine)
	}
	forward, err := Forward(grid, values)
	if err != nil {
		test.Fatal(err)
	}
	back, err := Inverse(grid, forward)
	if err != nil {
		test.Fatal(err)
	}
	checkComplexes(test, back, values, 1e-11)

	var inputPower, spectrumPower float64
	for _, value := range values {
		inputPower += cmplx.Abs(value) * cmplx.Abs(value)
	}
	for _, value := range forward {
		spectrumPower += cmplx.Abs(value) * cmplx.Abs(value)
	}
	spectrumPower /= float64(len(values))
	if difference := math.Abs(inputPower - spectrumPower); difference > 1e-10*inputPower {
		test.Errorf("Parseval powers are %g and %g", inputPower, spectrumPower)
	}
}

func TestTransformsRefuseInvalidShapesAndReturnNoResult(test *testing.T) {
	transformations := map[string]func(Shape, []complex128) ([]complex128, error){
		"forward": Forward,
		"inverse": Inverse,
	}
	invalid := []struct {
		name   string
		grid   Shape
		length int
	}{
		{"negative width", Shape{Width: -2, Height: 2, MaxCells: 8}, 0},
		{"zero width", Shape{Width: 0, Height: 2, MaxCells: 8}, 0},
		{"unit width", Shape{Width: 1, Height: 2, MaxCells: 8}, 2},
		{"non power width", Shape{Width: 3, Height: 2, MaxCells: 8}, 6},
		{"negative height", Shape{Width: 2, Height: -2, MaxCells: 8}, 0},
		{"zero height", Shape{Width: 2, Height: 0, MaxCells: 8}, 0},
		{"unit height", Shape{Width: 2, Height: 1, MaxCells: 8}, 2},
		{"non power height", Shape{Width: 2, Height: 3, MaxCells: 8}, 6},
		{"zero budget", Shape{Width: 2, Height: 2, MaxCells: 0}, 4},
		{"negative budget", Shape{Width: 2, Height: 2, MaxCells: -1}, 4},
		{"over budget", Shape{Width: 4, Height: 4, MaxCells: 15}, 16},
		{"short input", Shape{Width: 4, Height: 2, MaxCells: 8}, 7},
		{"long input", Shape{Width: 2, Height: 2, MaxCells: 8}, 5},
		{"cell count overflows int", Shape{
			Width: int(uint(1) << (strconv.IntSize - 2)), Height: 2,
			MaxCells: int(^uint(0) >> 1),
		}, 0},
	}
	for name, transform := range transformations {
		for _, each := range invalid {
			test.Run(name+"/"+each.name, func(test *testing.T) {
				values := make([]complex128, each.length)
				got, err := transform(each.grid, values)
				if err == nil || got != nil {
					test.Fatalf("got (%v, %v), want an error and nil result", got, err)
				}
			})
		}
	}
}

func TestTransformsRefuseNonFiniteComplexInputs(test *testing.T) {
	transformations := map[string]func(Shape, []complex128) ([]complex128, error){
		"forward": Forward,
		"inverse": Inverse,
	}
	invalid := map[string]complex128{
		"nan real":               complex(math.NaN(), 0),
		"nan imaginary":          complex(0, math.NaN()),
		"positive inf real":      complex(math.Inf(1), 0),
		"negative inf imaginary": complex(0, math.Inf(-1)),
	}
	for name, transform := range transformations {
		for label, value := range invalid {
			test.Run(name+"/"+label, func(test *testing.T) {
				values := []complex128{0, 0, 0, value}
				got, err := transform(shape(2, 2), values)
				if err == nil || got != nil {
					test.Fatalf("got (%v, %v), want an error and nil result", got, err)
				}
			})
		}
	}
}

func TestForwardRefusesFiniteInputsThatOverflowTheResult(test *testing.T) {
	grid := shape(2, 2)
	halfMaximum := math.MaxFloat64 / 2
	values := []complex128{
		complex(halfMaximum, 0),
		complex(halfMaximum, 0),
		complex(halfMaximum, 0),
		complex(halfMaximum, 0),
	}
	before := append([]complex128(nil), values...)
	got, err := Forward(grid, values)
	if err == nil || got != nil {
		test.Fatalf("got (%v, %v), want an error and nil result", got, err)
	}
	checkUnchanged(test, before, values)
}

func TestTransformsLeaveInputsUntouched(test *testing.T) {
	grid := shape(4, 2)
	values := []complex128{1 + 2i, 3 - 4i, -2 + 5i, 7, 8i, -9, 2 - 3i, 4 + 6i}
	for name, transform := range map[string]func(Shape, []complex128) ([]complex128, error){
		"forward": Forward,
		"inverse": Inverse,
	} {
		test.Run(name, func(test *testing.T) {
			before := append([]complex128(nil), values...)
			if _, err := transform(grid, values); err != nil {
				test.Fatal(err)
			}
			checkUnchanged(test, before, values)
		})
	}
}

// anti-test of TestTransformsRefuseInvalidShapesAndReturnNoResult: a valid
// grid returns an independent result whose edits cannot alter its input.
func TestTransformResultsDoNotAliasTheirInputs(test *testing.T) {
	grid := shape(2, 2)
	for name, transform := range map[string]func(Shape, []complex128) ([]complex128, error){
		"forward": Forward,
		"inverse": Inverse,
	} {
		test.Run(name, func(test *testing.T) {
			input := []complex128{1, 2 + 3i, -4i, 5 - 6i}
			before := append([]complex128(nil), input...)
			result, err := transform(grid, input)
			if err != nil {
				test.Fatal(err)
			}
			if len(result) != len(input) || result == nil {
				test.Fatalf("got result %v; want %d values", result, len(input))
			}
			result[0] += 1
			checkUnchanged(test, before, input)
		})
	}
}

// anti-test of TestForwardFindsSingleFourierModeWithNegativeExponent: a
// negative-exponent input mode peaks at its wrapped negative-frequency bin.
func TestForwardPutsTheOppositeModeAtTheWrappedBin(test *testing.T) {
	grid := shape(8, 4)
	const modeX, modeY = 3, 1
	values := make([]complex128, grid.Width*grid.Height)
	for y := range grid.Height {
		for x := range grid.Width {
			turns := float64(modeX*x)/float64(grid.Width) +
				float64(modeY*y)/float64(grid.Height)
			values[y*grid.Width+x] = cmplx.Exp(complex(0, -2*math.Pi*turns))
		}
	}
	got, err := Forward(grid, values)
	if err != nil {
		test.Fatal(err)
	}
	want := make([]complex128, len(values))
	length := float64(len(values))
	want[(grid.Height-modeY)*grid.Width+(grid.Width-modeX)] =
		complex(length, 0)
	checkComplexes(test, got, want, 1e-11)
}

func TestTransformsAgreeWithDirectOracleForSmallRectangles(test *testing.T) {
	rapid.Check(test, func(test *rapid.T) {
		width := 1 << rapid.IntRange(1, 3).Draw(test, "width power")
		height := 1 << rapid.IntRange(1, 3).Draw(test, "height power")
		grid := shape(width, height)
		values := make([]complex128, width*height)
		for index := range values {
			values[index] = complex(
				rapid.Float64Range(-4, 4).Draw(test, "real"),
				rapid.Float64Range(-4, 4).Draw(test, "imaginary"))
		}
		got, err := Forward(grid, values)
		if err != nil {
			test.Fatalf("Forward: %v", err)
		}
		if len(got) != len(values) {
			test.Fatalf("Forward returned %d values, want %d", len(got), len(values))
		}
		want := direct(values, grid, false)
		for index := range want {
			if !finite(real(got[index])) || !finite(imag(got[index])) {
				test.Fatalf("forward value %d is non-finite: %v", index, got[index])
			}
			limit := 1e-10 * math.Max(1, cmplx.Abs(want[index]))
			if difference := cmplx.Abs(got[index] - want[index]); difference > limit {
				test.Fatalf("forward value %d is %v, want %v", index,
					got[index], want[index])
			}
		}
		back, err := Inverse(grid, got)
		if err != nil {
			test.Fatalf("Inverse: %v", err)
		}
		if len(back) != len(values) {
			test.Fatalf("Inverse returned %d values, want %d", len(back), len(values))
		}
		for index := range values {
			if !finite(real(back[index])) || !finite(imag(back[index])) {
				test.Fatalf("inverse value %d is non-finite: %v", index, back[index])
			}
			limit := 1e-10 * math.Max(1, cmplx.Abs(values[index]))
			if difference := cmplx.Abs(back[index] - values[index]); difference > limit {
				test.Fatalf("inverse value %d is %v, want %v", index,
					back[index], values[index])
			}
		}
	})
}
