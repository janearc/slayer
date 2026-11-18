//go:build draw && darwin

package optics

import (
	"fmt"
	"math"
	"math/cmplx"
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/simulation/spectral"
)

// openedFieldMetal requires the actual backend; an absent device fails.
func openedFieldMetal(t testing.TB, field *Field) *MetalField {
	t.Helper()
	plan, err := field.OpenMetal()
	if err != nil || plan == nil {
		t.Fatalf("OpenMetal: %v, %v", plan, err)
	}
	t.Cleanup(plan.Close)
	return plan
}

// TestMetalFieldsMatchScalarReference exercises propagation beyond the
// aperture plane, including evanescent relative phase at a longer wavelength.
func TestMetalFieldsMatchScalarReference(t *testing.T) {
	for _, wavelength := range []float64{0.05, 4} {
		for _, size := range []int{8, 64, 256} {
			t.Run(fmt.Sprintf("%g-%d", wavelength, size), func(t *testing.T) {
				conf := Config{Shape: spectral.Shape{Width: size, Height: size, MaxCells: size * size},
					WidthMetres: 8, HeightMetres: 8, WavelengthMetres: wavelength}
				initial := make([]complex128, size*size)
				for y := range size {
					for x := range size {
						// Two apertures with an asymmetric complex illumination.
						if (x > size/8 && x < size*3/8 || x > size*5/8 && x < size*7/8) && y >= size/4 && y < size*3/4 {
							initial[y*size+x] = complex(0.75, float64(y%7)/10-0.2)
						}
					}
				}
				field, err := New(conf, initial)
				if err != nil {
					t.Fatal(err)
				}
				before := slices.Clone(initial)
				plan := openedFieldMetal(t, field)
				for _, distance := range []float64{0, 0.125, 2, 0.125} {
					want, err := field.At(distance)
					if err != nil {
						t.Fatal(err)
					}
					got, err := plan.At(distance)
					if err != nil {
						t.Fatal(err)
					}
					requireMetalField(t, got, want)
					if distance == 0 && !slices.Equal(got, initial) {
						t.Fatal("zero distance lost exact samples")
					}
					got[0] = 123
				}
				if !slices.Equal(initial, before) {
					t.Fatal("Metal propagation changed caller input")
				}
			})
		}
	}
}

// requireMetalField compares finite complex fields with a fixed relative L2
// bound and absolute floor for an extinguished evanescent field.
func requireMetalField(t testing.TB, got, want []complex128) {
	t.Helper()
	if len(got) != len(want) || len(got) == 0 {
		t.Fatalf("field lengths %d/%d", len(got), len(want))
	}
	var delta, norm float64
	for i := range want {
		if !fieldFiniteComplex(got[i]) {
			t.Fatalf("non-finite Metal field at %d", i)
		}
		delta = math.Hypot(delta, cmplx.Abs(got[i]-want[i]))
		norm = math.Hypot(norm, cmplx.Abs(want[i]))
	}
	if delta > 3e-5*math.Max(norm, 1e-8) {
		t.Fatalf("field L2 error %g, reference norm %g", delta, norm)
	}
}

// anti-test of TestMetalFieldsMatchScalarReference
// Rejected distances cannot poison the retained spectrum or an earlier result.
func TestMetalFieldRefusalsPreserveState(t *testing.T) {
	field, err := New(metalOpticsConfig(), mode(metalOpticsConfig().Shape, 1, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	plan := openedFieldMetal(t, field)
	first, err := plan.At(0.2)
	if err != nil {
		t.Fatal(err)
	}
	saved := slices.Clone(first)
	for _, distance := range []float64{-1, math.NaN(), math.Inf(1)} {
		got, err := plan.At(distance)
		if err == nil || got != nil {
			t.Fatalf("accepted invalid distance %g", distance)
		}
	}
	next, err := plan.At(0.2)
	if err != nil {
		t.Fatal(err)
	}
	requireMetalField(t, next, saved)
	if !slices.Equal(first, saved) {
		t.Fatal("later calls changed an earlier result")
	}
}

// A closed, nil or zero Metal field must refuse every distance.
func TestMetalFieldLifecycleRefusesMissingResources(t *testing.T) {
	field, err := New(metalOpticsConfig(), mode(metalOpticsConfig().Shape, 0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	closed := openedFieldMetal(t, field)
	closed.Close()
	closed.Close()
	for _, plan := range []*MetalField{nil, {}, closed} {
		plan.Close()
		for _, distance := range []float64{0, 1} {
			got, err := plan.At(distance)
			if err == nil || got != nil {
				t.Fatal("a non-live plan accepted propagation")
			}
		}
	}
	for _, field := range []*Field{nil, {}} {
		plan, err := field.OpenMetal()
		if err == nil || plan != nil {
			t.Fatal("an uninitialized field opened Metal")
		}
	}
}

// anti-test of TestMetalFieldLifecycleRefusesMissingResources
// Valid resources can propagate a small-angle mode whose phase would
// disappear under direct subtraction or float32 absolute optical phase.
func TestMetalFieldKeepsSmallAnglePhase(t *testing.T) {
	conf := metalOpticsConfig()
	conf.WidthMetres = 1e9
	conf.HeightMetres = 1e9
	conf.WavelengthMetres = 0.5
	field, err := New(conf, mode(conf.Shape, 1, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	plan := openedFieldMetal(t, field)
	want, err := field.At(1e15)
	if err != nil {
		t.Fatal(err)
	}
	got, err := plan.At(1e15)
	if err != nil {
		t.Fatal(err)
	}
	requireMetalField(t, got, want)
	initial, _ := field.At(0)
	if cmplx.Abs(want[0]-initial[0]) < 1e-4 {
		t.Fatal("fixture has no resolved relative phase")
	}
}

// BenchmarkFieldMetalComparison includes transfer construction, upload,
// one batch and final readback; setup is outside the measured loop.
func BenchmarkFieldMetalComparison(b *testing.B) {
	size := 256
	conf := Config{Shape: spectral.Shape{Width: size, Height: size, MaxCells: size * size},
		WidthMetres: 8, HeightMetres: 8, WavelengthMetres: 0.05}
	values := make([]complex128, size*size)
	for i := range values {
		values[i] = complex(float64(i%17)/17, float64(i%13)/13)
	}
	field, err := New(conf, values)
	if err != nil {
		b.Fatal(err)
	}
	for _, gpu := range []bool{false, true} {
		name, propagate := "cpu", field.At
		var setup time.Duration
		if gpu {
			start := time.Now()
			plan := openedFieldMetal(b, field)
			setup = time.Since(start)
			name, propagate = "metal", plan.At
		}
		b.Run(name, func(b *testing.B) {
			if _, err := propagate(2); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for range b.N {
				got, err := propagate(2)
				if err != nil || len(got) != len(values) {
					b.Fatalf("propagate: %v", err)
				}
			}
			b.StopTimer()
			if gpu {
				b.ReportMetric(float64(setup), "setup-ns")
			}
		})
	}
}

// metalOpticsConfig is a small resolved grid with propagating low modes.
func metalOpticsConfig() Config { return config(8, 4, 8, 4, 0.5) }

// Metal refuses finite CPU fields outside its conservative float32 range.
func TestMetalFieldRefusesUnrepresentableSpectrum(t *testing.T) {
	conf := metalOpticsConfig()
	values := mode(conf.Shape, 0, 0, 1)
	for i := range values {
		values[i] *= 1e40
	}
	field, err := New(conf, values)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := field.OpenMetal()
	if plan != nil {
		plan.Close()
	}
	if err == nil || plan != nil {
		t.Fatal("Metal accepted a spectrum outside float32 range")
	}
}

// anti-test of TestMetalFieldRefusesUnrepresentableSpectrum
// A coherently cancelled field remains zero after propagation.
func TestMetalFieldPreservesCoherentCancellation(t *testing.T) {
	conf := metalOpticsConfig()
	values := mode(conf.Shape, 1, 0, 1)
	for i := range values {
		values[i] += -values[i]
	}
	field, err := New(conf, values)
	if err != nil {
		t.Fatal(err)
	}
	plan := openedFieldMetal(t, field)
	got, err := plan.At(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(values) {
		t.Fatal("missing cancelled samples")
	}
	for _, value := range got {
		if value != 0 {
			t.Fatalf("cancelled field became %v", value)
		}
	}
}

// anti-test of TestMetalFieldKeepsSmallAnglePhase
// Accurate host reduction must also survive a large relative optical phase.
func TestMetalFieldKeepsLargeRelativePhase(t *testing.T) {
	conf := config(8, 4, 2*math.Pi, 2*math.Pi, 2*math.Pi)
	initial := mode(conf.Shape, 1, 0, 1)
	field, err := New(conf, initial)
	if err != nil {
		t.Fatal(err)
	}
	plan := openedFieldMetal(t, field)
	got, err := plan.At(1e16)
	if err != nil {
		t.Fatal(err)
	}
	sine, cosine := math.Sincos(-1e16)
	want := make([]complex128, len(initial))
	for i := range want {
		want[i] = initial[i] * complex(cosine, sine)
	}
	requireMetalField(t, got, want)
}
