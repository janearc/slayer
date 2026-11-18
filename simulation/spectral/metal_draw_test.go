//go:build draw && darwin

package spectral_test

import (
	"fmt"
	"math"
	"math/cmplx"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/simulation/internal/metalfft"
	"github.com/janearc/slayer/simulation/spectral"
	"pgregory.net/rapid"
)

// metalTolerance is the picked float32 acceptance threshold for these
// bounded grids, about 250 float32 epsilons. It is not a float64 claim.
const metalTolerance = 0.00003

// metalShape gives a test exactly the cell budget its dimensions need.
func metalShape(width, height int) spectral.Shape {
	return spectral.Shape{Width: width, Height: height,
		MaxCells: width * height}
}

// openedMetal fails when hardware or compilation is unavailable. These
// tests establish executed Metal behavior and never skip a missing device.
func openedMetal(test testing.TB, shape spectral.Shape) *spectral.MetalPlan {
	test.Helper()
	plan, err := spectral.OpenMetal(shape)
	if err != nil || plan == nil {
		test.Fatalf("OpenMetal(%+v): plan %v, error %v", shape, plan, err)
	}
	test.Cleanup(plan.Close)
	if plan.Device() == "" {
		test.Fatal("a live Metal plan names no device")
	}
	return plan
}

// metalValues fixes the same signed, complex input for each implementation.
func metalValues(cells int) []complex128 {
	values := make([]complex128, cells)
	for index := range values {
		values[index] = complex(float64(index%7)-2.5,
			float64(index%5)/3-0.75)
	}
	return values
}

// metalDFT is the defining finite sum, independent of a butterfly layout.
func metalDFT(shape spectral.Shape, values []complex128,
	inverse bool) []complex128 {
	answer := make([]complex128, len(values))
	sign := -1.0
	if inverse {
		sign = 1
	}
	for ky := range shape.Height {
		for kx := range shape.Width {
			var sum complex128
			for y := range shape.Height {
				for x := range shape.Width {
					turns := float64(kx*x)/float64(shape.Width) +
						float64(ky*y)/float64(shape.Height)
					sum += values[y*shape.Width+x] *
						cmplx.Exp(complex(0, sign*2*math.Pi*turns))
				}
			}
			if inverse {
				sum /= complex(float64(len(values)), 0)
			}
			answer[ky*shape.Width+kx] = sum
		}
	}
	return answer
}

// closeMetalValues requires finite values and a bounded complex error.
func closeMetalValues(test testing.TB, got, want []complex128) {
	test.Helper()
	if len(got) != len(want) {
		test.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for index, expected := range want {
		error := cmplx.Abs(got[index] - expected)
		limit := metalTolerance * math.Max(1, cmplx.Abs(expected))
		if math.IsNaN(error) || math.IsInf(error, 0) || error > limit {
			test.Fatalf("cell %d: got %v, want %v, error %g, limit %g",
				index, got[index], expected, error, limit)
		}
	}
}

// Both signs and rectangular strides match the direct sum and CPU oracle.
func TestMetalTransformsMatchDirectSumsAndCPU(test *testing.T) {
	for _, shape := range []spectral.Shape{metalShape(2, 2),
		metalShape(4, 2), metalShape(2, 8), metalShape(8, 4)} {
		test.Run(fmt.Sprintf("%dx%d", shape.Width, shape.Height),
			func(test *testing.T) { metalOracles(test, shape) })
	}
}

// metalOracles checks each direction independently, retaining the input.
func metalOracles(test *testing.T, shape spectral.Shape) {
	test.Helper()
	plan := openedMetal(test, shape)
	values := metalValues(shape.Width * shape.Height)
	before := slices.Clone(values)
	for _, inverse := range []bool{false, true} {
		transform, cpu := plan.Forward, spectral.Forward
		if inverse {
			transform, cpu = plan.Inverse, spectral.Inverse
		}
		got, err := transform(values)
		if err != nil {
			test.Fatal(err)
		}
		oracle, err := cpu(shape, values)
		if err != nil {
			test.Fatal(err)
		}
		closeMetalValues(test, got, metalDFT(shape, values, inverse))
		closeMetalValues(test, got, oracle)
		if !slices.Equal(values, before) {
			test.Fatal("the transform changed its input")
		}
	}
}

// anti-test of TestMetalTransformsMatchDirectSumsAndCPU
// wrong shapes
// and exhausted budgets are refused before opening a device.
func TestMetalRefusesInvalidShapes(test *testing.T) {
	for _, shape := range []spectral.Shape{
		{Width: 0, Height: 2, MaxCells: 4},
		{Width: 2, Height: 1, MaxCells: 4},
		{Width: 3, Height: 4, MaxCells: 12},
		{Width: 2, Height: 4, MaxCells: 7},
		{Width: 2, Height: 2, MaxCells: 0},
		{Width: -2, Height: 4, MaxCells: 8},
		{Width: 1 << 62, Height: 4, MaxCells: int(^uint(0) >> 1)},
	} {
		plan, err := spectral.OpenMetal(shape)
		if err == nil || plan != nil {
			if plan != nil {
				plan.Close()
			}
			test.Fatalf("accepted shape %+v: %v", shape, err)
		}
	}
}

// anti-test of TestMetalPlanReusesBuffersWithoutLeakingResults:
// refusals yield no partial result, preserve input bits including NaNs,
// and leave a live plan usable for the next valid transform.
func TestMetalRefusesInvalidValuesWithoutPoisoningThePlan(test *testing.T) {
	plan := openedMetal(test, metalShape(2, 2))
	for _, values := range [][]complex128{nil, make([]complex128, 3),
		{complex(math.NaN(), 0), 0, 0, 0},
		{complex(0, math.Inf(1)), 0, 0, 0},
		{complex(math.MaxFloat64, 0), 0, 0, 0},
		{math.MaxFloat32, math.MaxFloat32, 0, 0}} {
		before := slices.Clone(values)
		for _, transform := range []func([]complex128) ([]complex128,
			error){plan.Forward, plan.Inverse} {
			got, err := transform(values)
			if err == nil || got != nil {
				test.Fatalf("accepted invalid values %v: %v, %v",
					values, got, err)
			}
			unchangedMetalBits(test, values, before)
		}
	}
	got, err := plan.Forward([]complex128{1, 0, 0, 0})
	if err != nil {
		test.Fatal(err)
	}
	closeMetalValues(test, got, []complex128{1, 1, 1, 1})
}

// unchangedMetalBits compares the exact input representation after refusals.
func unchangedMetalBits(test *testing.T, got, want []complex128) {
	test.Helper()
	for index := range want {
		if math.Float64bits(real(got[index])) !=
			math.Float64bits(real(want[index])) ||
			math.Float64bits(imag(got[index])) !=
				math.Float64bits(imag(want[index])) {
			test.Fatalf("input cell %d changed", index)
		}
	}
}

// anti-test of TestMetalTransformsMatchDirectSumsAndCPU
// closing twice
// is harmless, and neither closed nor zero plans draw.
func TestMetalClosedAndZeroPlansRefuse(test *testing.T) {
	plan := openedMetal(test, metalShape(2, 2))
	plan.Close()
	plan.Close()
	for _, closed := range []*spectral.MetalPlan{plan, {}, nil} {
		closed.Close()
		for _, transform := range []func([]complex128) ([]complex128,
			error){closed.Forward, closed.Inverse} {
			got, err := transform([]complex128{1, 0, 0, 0})
			if err == nil || got != nil {
				test.Fatalf("closed plan drew: %v, %v", got, err)
			}
		}
	}
}

// Distinct inputs through one retained plan must not leak the last result.
func TestMetalPlanReusesBuffersWithoutLeakingResults(test *testing.T) {
	plan := openedMetal(test, metalShape(4, 2))
	first, err := plan.Forward([]complex128{1, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		test.Fatal(err)
	}
	kept := slices.Clone(first)
	second, err := plan.Forward(make([]complex128, 8))
	if err != nil {
		test.Fatal(err)
	}
	closeMetalValues(test, second, make([]complex128, 8))
	closeMetalValues(test, first, kept)
}

// The inverse scale restores the input and the forward transform obeys
// Parseval's identity on a retained rectangular plan.
func TestMetalRoundTripAndParsevalProperty(test *testing.T) {
	shape := metalShape(8, 4)
	plan := openedMetal(test, shape)
	rapid.Check(test, func(draw *rapid.T) {
		values := make([]complex128, shape.MaxCells)
		for index := range values {
			values[index] = complex(
				rapid.Float64Range(-1, 1).Draw(draw, "real"),
				rapid.Float64Range(-1, 1).Draw(draw, "imaginary"))
		}
		forward, err := plan.Forward(values)
		if err != nil {
			draw.Fatal(err)
		}
		inverse, err := plan.Inverse(forward)
		if err != nil {
			draw.Fatal(err)
		}
		metalProperty(draw, values, forward, inverse)
	})
}

// anti-test of TestMetalRoundTripAndParsevalProperty
// a round trip alone
// can hide reversed signs or paired scale errors. One asymmetric mode
// must land at its own forward bin and its mirrored inverse bin.
func TestMetalRoundTripCannotHideSignOrScaleErrors(test *testing.T) {
	shape := metalShape(4, 8)
	plan := openedMetal(test, shape)
	values := make([]complex128, shape.MaxCells)
	for y := range shape.Height {
		for x := range shape.Width {
			turns := float64(x)/float64(shape.Width) +
				3*float64(y)/float64(shape.Height)
			values[y*shape.Width+x] = cmplx.Exp(complex(0,
				2*math.Pi*turns))
		}
	}
	forward, err := plan.Forward(values)
	if err != nil {
		test.Fatal(err)
	}
	want := make([]complex128, len(values))
	want[3*shape.Width+1] = complex(float64(len(values)), 0)
	closeMetalValues(test, forward, want)
	inverse, err := plan.Inverse(values)
	if err != nil {
		test.Fatal(err)
	}
	clear(want)
	want[(shape.Height-3)*shape.Width+shape.Width-1] = 1
	closeMetalValues(test, inverse, want)
}

// metalProperty reports complex reconstruction and independent energy sums.
func metalProperty(draw *rapid.T, values, forward, inverse []complex128) {
	if len(inverse) != len(values) || len(forward) != len(values) {
		draw.Fatal("the transform changed the grid size")
	}
	var before, after float64
	for index, value := range values {
		if difference := cmplx.Abs(inverse[index] - value); math.IsNaN(difference) || difference > metalTolerance {
			draw.Fatalf("round trip cell %d: %v != %v", index,
				inverse[index], value)
		}
		before += real(value)*real(value) + imag(value)*imag(value)
		after += real(forward[index])*real(forward[index]) +
			imag(forward[index])*imag(forward[index])
	}
	after /= float64(len(values))
	if math.IsNaN(after) || math.Abs(after-before) >
		metalTolerance*math.Max(1, before) {
		draw.Fatalf("Parseval energy: %g != %g", after, before)
	}
}

// A forward transform, nonlinear pointwise kernel and inverse can be
// appended to one session and executed by a single final commit.
func TestMetalEncoderComposesWithoutIntermediateReadback(test *testing.T) {
	const doubling = `
kernel void fft_test_square(device float2 *values [[buffer(0)]],
                            uint index [[thread_position_in_grid]]) {
    float2 value = values[index];
    values[index] = float2(value.x * value.x - value.y * value.y,
                           2.0f * value.x * value.y);
}
`
	names := append(metalfft.Kernels(), "fft_test_square")
	session, err := metal.OpenSession(metalfft.Source()+doubling, names...)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(session.Close)
	encoder, err := metalfft.New(session,
		metalfft.Shape{Width: 4, Height: 2, MaxCells: 8})
	if err != nil || encoder == nil {
		test.Fatalf("encoder %v, error %v", encoder, err)
	}
	metalComposition(test, session, encoder)
}

// metalComposition reads only after both transforms and the middle kernel.
func metalComposition(test *testing.T, session *metal.Session,
	encoder *metalfft.Encoder) {
	test.Helper()
	buffers := make([]metal.Buffer, 3)
	for index := range buffers {
		var err error
		buffers[index], err = session.Buffer(16)
		if err != nil {
			test.Fatal(err)
		}
	}
	values := metalValues(8)
	for index, value := range values {
		buffers[0].Numbers()[2*index] = float32(real(value))
		buffers[0].Numbers()[2*index+1] = float32(imag(value))
	}
	forward, err := encoder.Append(buffers[0], buffers[1], buffers[2], false)
	if err != nil {
		test.Fatal(err)
	}
	if err := session.Dispatch("fft_test_square", 8, 1, nil, forward); err != nil {
		test.Fatal(err)
	}
	// Forward's result is one of its two work buffers. Pick the other
	// for scratch, preserving the transformed input to the inverse.
	scratch := buffers[2]
	if len(forward.Numbers()) == 0 {
		test.Fatal("the encoder returned no result buffer")
	}
	if &forward.Numbers()[0] == &scratch.Numbers()[0] {
		scratch = buffers[1]
	}
	inverse, err := encoder.Append(forward, buffers[0], scratch, true)
	if err != nil {
		test.Fatal(err)
	}
	if err := session.Commit(); err != nil {
		test.Fatal(err)
	}
	got := make([]complex128, len(values))
	for index := range got {
		got[index] = complex(float64(inverse.Numbers()[2*index]),
			float64(inverse.Numbers()[2*index+1]))
	}
	want := metalDFT(metalShape(4, 2), values, false)
	for index := range want {
		want[index] *= want[index]
	}
	closeMetalValues(test, got, metalDFT(metalShape(4, 2), want, true))
}

// anti-test of TestMetalEncoderComposesWithoutIntermediateReadback:
// invalid work buffers are refused rather than silently queued over an
// alias or a shorter allocation. No commit follows a rejected append.
func TestMetalEncoderRefusesShortAndAliasedScratch(test *testing.T) {
	session, err := metal.OpenSession(metalfft.Source(),
		metalfft.Kernels()...)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(session.Close)
	encoder, err := metalfft.New(session,
		metalfft.Shape{Width: 2, Height: 2, MaxCells: 4})
	if err != nil || encoder == nil {
		test.Fatalf("encoder %v, error %v", encoder, err)
	}
	first, err := session.Buffer(8)
	if err != nil {
		test.Fatal(err)
	}
	second, err := session.Buffer(8)
	if err != nil {
		test.Fatal(err)
	}
	short, err := session.Buffer(7)
	if err != nil {
		test.Fatal(err)
	}
	for _, buffers := range [][3]metal.Buffer{
		{first, second, second}, {first, second, first},
		{short, first, second}, {first, short, second},
		{first, second, short}, {{}, first, second},
	} {
		got, err := encoder.Append(buffers[0], buffers[1],
			buffers[2], false)
		if err == nil || len(got.Numbers()) != 0 {
			test.Fatalf("accepted invalid buffers: %v, %v", got, err)
		}
	}
}

// The exported source supplies every function it advertises, checked
// before a Metal compiler or device is asked to do any work.
func TestMetalSourceDeclaresEveryNamedKernel(test *testing.T) {
	source, names := metalfft.Source(), metalfft.Kernels()
	if strings.TrimSpace(source) == "" {
		test.Fatal("Source returned no shader source")
	}
	if len(names) == 0 {
		test.Fatal("Kernels returned no function names")
	}
	for _, name := range names {
		if name == "" || !strings.Contains(source,
			"kernel void "+name+"(") {
			test.Fatalf("source does not declare its kernel %q", name)
		}
	}
}

// anti-test of TestMetalSourceDeclaresEveryNamedKernel
// a caller may
// change or extend its kernel list without changing another caller's.
func TestMetalKernelNamesAreIndependent(test *testing.T) {
	first, second := metalfft.Kernels(), metalfft.Kernels()
	if len(first) == 0 || !slices.Equal(first, second) {
		test.Fatal("the kernel list is empty or unstable")
	}
	first[0] = "changed"
	if second[0] == "changed" || metalfft.Kernels()[0] == "changed" {
		test.Fatal("the caller changed the shared kernel list")
	}
}

// largeMetalTolerance is a fixed acceptance limit, not a per-bin floor.
// These grids have 12 to 17 radix-two stages. A limit near 250 float32
// epsilons allows conversion, rounded twiddles and accumulated arithmetic;
// measured agreement is established here, not inferred from that allowance.
const largeMetalTolerance = 0.00003

// Both directions on benchmark-sized rectangles match the float64 CPU
// oracle independently, with finite output and untouched input.
func TestMetalLargeGridsMatchCPU64(test *testing.T) {
	for _, shape := range []spectral.Shape{metalShape(64, 64),
		metalShape(256, 256), metalShape(512, 256)} {
		test.Run(fmt.Sprintf("%dx%d", shape.Width, shape.Height),
			func(test *testing.T) { metalLargeOracles(test, shape) })
	}
}

// metalLargeValues is a signed, asymmetric complex fixture with a positive
// frequency mode. Its two components have different non-power-of-two cycles.
func metalLargeValues(shape spectral.Shape) []complex128 {
	values := make([]complex128, shape.MaxCells)
	for y := range shape.Height {
		for x := range shape.Width {
			turns := 3*float64(x)/float64(shape.Width) +
				5*float64(y)/float64(shape.Height)
			background := complex(float64((3*x+5*y)%17-8)/8,
				float64((7*x+11*y)%23-11)/11)
			values[y*shape.Width+x] = background +
				cmplx.Rect(0.5, 2*math.Pi*turns)
		}
	}
	return values
}

// metalLargeOracles compares each sign directly, so paired forward/inverse
// mistakes cannot cancel in a round trip.
func metalLargeOracles(test *testing.T, shape spectral.Shape) {
	test.Helper()
	plan := openedMetal(test, shape)
	values := metalLargeValues(shape)
	before := slices.Clone(values)
	for _, inverse := range []bool{false, true} {
		transform, cpu, direction := plan.Forward, spectral.Forward, "forward"
		if inverse {
			transform, cpu, direction = plan.Inverse, spectral.Inverse, "inverse"
		}
		want, err := cpu(shape, values)
		if err != nil {
			test.Fatal(err)
		}
		got, err := transform(values)
		if err != nil {
			test.Fatal(err)
		}
		requireLargeMetalError(test, direction, values, got, want)
		if !slices.Equal(values, before) {
			test.Fatal("a large transform changed its input")
		}
	}
}

// largeMetalError separates aggregate complex error from the largest bin's
// absolute error, normalized by the input's complex L1 norm.
type largeMetalError struct {
	relativeL2, maxBinOverInputL1 float64
}

// largeMetalErrorOf rejects non-finite components before forming norms.
// A zero reference or input permits only an exactly zero corresponding error.
func largeMetalErrorOf(input, got, want []complex128) (largeMetalError, error) {
	if len(input) == 0 || len(got) != len(input) || len(want) != len(input) {
		return largeMetalError{}, fmt.Errorf("inconsistent large grid sizes")
	}
	var inputL1, referenceL2, errorL2, maximum float64
	for index, value := range input {
		for _, candidate := range []complex128{value, got[index], want[index]} {
			if !metalFinite(candidate) {
				return largeMetalError{}, fmt.Errorf(
					"non-finite large-grid value at cell %d", index)
			}
		}
		difference := cmplx.Abs(got[index] - want[index])
		inputL1 += cmplx.Abs(value)
		referenceL2 = math.Hypot(referenceL2, cmplx.Abs(want[index]))
		errorL2 = math.Hypot(errorL2, difference)
		maximum = math.Max(maximum, difference)
	}
	return largeMetalError{relativeL2: largeMetalRatio(errorL2, referenceL2),
		maxBinOverInputL1: largeMetalRatio(maximum, inputL1)}, nil
}

// metalFinite checks both components; a NaN cannot pass a comparison by
// making an ordered error test false.
func metalFinite(value complex128) bool {
	return !math.IsNaN(real(value)) && !math.IsInf(real(value), 0) &&
		!math.IsNaN(imag(value)) && !math.IsInf(imag(value), 0)
}

// largeMetalRatio defines the zero-norm cases without a hidden absolute floor.
func largeMetalRatio(error, norm float64) float64 {
	if norm == 0 {
		if error == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return error / norm
}

// requireLargeMetalError uses the two fixed aggregate thresholds, leaving
// closeMetalValues and its stricter tiny-grid per-bin checks unchanged.
func requireLargeMetalError(test *testing.T, direction string,
	input, got, want []complex128) {
	test.Helper()
	measured, err := largeMetalErrorOf(input, got, want)
	if err != nil {
		test.Fatal(err)
	}
	test.Logf("%s: relative L2 %.9g, max-bin/input-L1 %.9g", direction,
		measured.relativeL2, measured.maxBinOverInputL1)
	if math.IsNaN(measured.relativeL2) ||
		math.IsNaN(measured.maxBinOverInputL1) ||
		measured.relativeL2 > largeMetalTolerance ||
		measured.maxBinOverInputL1 > largeMetalTolerance {
		test.Fatalf("%s large-grid error %+v exceeds %g", direction,
			measured, largeMetalTolerance)
	}
}

// anti-test of TestMetalLargeGridsMatchCPU64
// A purely positive complex mode has different forward and inverse bins.
// Its analytic amplitudes expose sign, stride and normalization errors even
// when a paired round trip could hide them.
func TestMetalLargeGridsDoNotReverseSignOrLoseScale(test *testing.T) {
	shape := metalShape(512, 256)
	plan := openedMetal(test, shape)
	values := make([]complex128, shape.MaxCells)
	for y := range shape.Height {
		for x := range shape.Width {
			turns := 3*float64(x)/float64(shape.Width) +
				5*float64(y)/float64(shape.Height)
			values[y*shape.Width+x] = complex(0.75, -0.25) *
				cmplx.Exp(complex(0, 2*math.Pi*turns))
		}
	}
	before := slices.Clone(values)
	forward, err := plan.Forward(values)
	if err != nil {
		test.Fatal(err)
	}
	want := make([]complex128, len(values))
	want[5*shape.Width+3] = complex(0.75, -0.25) *
		complex(float64(len(values)), 0)
	requireLargeMetalError(test, "forward analytic mode", values,
		forward, want)
	inverse, err := plan.Inverse(values)
	if err != nil {
		test.Fatal(err)
	}
	clear(want)
	want[(shape.Height-5)*shape.Width+shape.Width-3] = complex(0.75, -0.25)
	requireLargeMetalError(test, "inverse analytic mode", values,
		inverse, want)
	if !slices.Equal(values, before) {
		test.Fatal("the analytic large-grid transform changed its input")
	}
}

// anti-test of TestMetalLargeGridsMatchCPU64
// Invalid output values or sizes must fail the metric itself, rather than
// disappearing into NaN comparisons or empty norms.
func TestMetalLargeMetricsRefuseNonFiniteAndWrongSizes(test *testing.T) {
	input, want := []complex128{1, 2, 3, 4}, []complex128{4, 3, 2, 1}
	for _, got := range [][]complex128{nil, {1, 2, 3},
		{complex(math.NaN(), 0), 3, 2, 1},
		{4, complex(0, math.Inf(1)), 2, 1},
		{4, 3, complex(math.Inf(-1), 0), 1}} {
		if _, err := largeMetalErrorOf(input, got, want); err == nil {
			test.Fatalf("the large-grid metric accepted %v", got)
		}
	}
}

// The comparison includes boundary packing and final readback. Setup and
// the first call are reported separately from the warm median.
func BenchmarkMetalFFTComparison(bench *testing.B) {
	previous := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(previous)
	for _, shape := range []spectral.Shape{metalShape(64, 64),
		metalShape(256, 256), metalShape(512, 256)} {
		bench.Run(fmt.Sprintf("%dx%d", shape.Width, shape.Height),
			func(bench *testing.B) { metalBenchmarkPair(bench, shape) })
	}
}

// metalBenchmarkPair uses the same input for CPU and retained Metal calls.
func metalBenchmarkPair(bench *testing.B, shape spectral.Shape) {
	values := metalValues(shape.MaxCells)
	bench.Run("cpu-GOMAXPROCS-two", func(bench *testing.B) {
		metalBenchmarkCalls(bench, func() ([]complex128, error) {
			return spectral.Forward(shape, values)
		})
	})
	bench.Run("metal-including-readback", func(bench *testing.B) {
		bench.StopTimer()
		start := time.Now()
		plan := openedMetal(bench, shape)
		setup := time.Since(start)
		metalBenchmarkCalls(bench, func() ([]complex128, error) {
			return plan.Forward(values)
		})
		bench.ReportMetric(float64(setup), "setup-ns")
	})
}

// metalBenchmarkCalls retains per-call samples to report an observed median.
func metalBenchmarkCalls(bench *testing.B,
	transform func() ([]complex128, error)) {
	bench.StopTimer()
	start := time.Now()
	first, err := transform()
	if err != nil || len(first) == 0 {
		bench.Fatalf("first transform: %d values, %v", len(first), err)
	}
	cold := time.Since(start)
	times := make([]time.Duration, bench.N)
	bench.ResetTimer()
	bench.StartTimer()
	for index := range bench.N {
		start := time.Now()
		result, err := transform()
		times[index] = time.Since(start)
		if err != nil || len(result) != len(first) {
			bench.Fatalf("transform: %d values, %v", len(result), err)
		}
	}
	bench.StopTimer()
	slices.Sort(times)
	middle := len(times) / 2
	median := float64(times[middle])
	if len(times)%2 == 0 {
		median = (float64(times[middle-1]) + median) / 2
	}
	bench.ReportMetric(float64(cold), "first-call-ns")
	bench.ReportMetric(median, "warm-median-ns")
}
