//go:build draw && darwin

package gravitational

import (
	"fmt"
	"math"
	"slices"
	"testing"
)

func openPlaneMetal(t testing.TB, field *Field) *MetalField {
	t.Helper()
	plan, err := field.OpenMetal()
	if err != nil || plan == nil {
		t.Fatalf("OpenMetal: %v, %v", plan, err)
	}
	t.Cleanup(plan.Close)
	return plan
}

// requirePlaneMetal retains a fixed physical amplitude for each channel.
// Its bounds stay proportional to strain and rate, even at 1e-21 strain.
func requirePlaneMetal(t testing.TB, got, want Profiles, count int) {
	t.Helper()
	planeProfiles(t, got, count)
	planeProfiles(t, want, count)
	for component, pair := range [][2][]float64{{got.Plus, want.Plus}, {got.Cross, want.Cross}, {got.PlusRate, want.PlusRate}, {got.CrossRate, want.CrossRate}} {
		scale := 1e-21
		if component >= 2 {
			scale = 1e-20
		}
		for i, value := range pair[0] {
			if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value-pair[1][i]) > 3e-5*scale {
				t.Fatalf("component %d sample %d: got %g, want %g, physical scale %g", component, i, value, pair[1][i], scale)
			}
		}
	}
}

// Both polarizations and rates match the float64 model across FFT parities.
func TestGravityMetalMatchesPhysicalProfiles(t *testing.T) {
	for _, samples := range []int{64, 512, 4096} {
		t.Run(fmt.Sprint(samples), func(t *testing.T) {
			conf := planeConfig()
			conf.Samples = samples
			conf.MaxSamples = samples
			initial := planeInitial(conf)
			field := openPlane(t, conf, initial)
			plan := openPlaneMetal(t, field)
			for _, seconds := range []float64{0, 0.125, -0.375, 0.07} {
				got, err := plan.At(seconds)
				if err != nil {
					t.Fatal(err)
				}
				want, err := field.At(seconds)
				if err != nil {
					t.Fatal(err)
				}
				requirePlaneMetal(t, got, want, samples)
				if seconds == 0 && (!slices.Equal(got.Plus, initial.Plus) || !slices.Equal(got.CrossRate, initial.CrossRate)) {
					t.Fatal("zero time lost exact physical profiles")
				}
			}
		})
	}
}

// anti-test of TestGravityMetalMatchesPhysicalProfiles
// A rejected time or out-of-regime candidate cannot alter later profiles.
func TestGravityMetalWeakFieldRefusalsRemainUsable(t *testing.T) {
	conf := planeConfig()
	initial := planeInitial(conf)
	for i := range initial.Plus {
		initial.Plus[i] = 0
		initial.Cross[i] = 0
		initial.PlusRate[i] = 0.02
		initial.CrossRate[i] = 0
	}
	field := openPlane(t, conf, initial)
	plan := openPlaneMetal(t, field)
	for _, seconds := range []float64{1, math.NaN(), math.Inf(1)} {
		got, err := plan.At(seconds)
		if err == nil || got.Plus != nil || got.Cross != nil || got.PlusRate != nil || got.CrossRate != nil {
			t.Fatal("invalid or out-of-regime Metal candidate accepted")
		}
	}
	got, err := plan.At(0)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, got, conf.Samples)
	if !slices.Equal(got.PlusRate, initial.PlusRate) {
		t.Fatal("rejected query changed accepted initial rates")
	}
}

// Detector and tidal observations consume the actual returned device strain.
func TestGravityMetalProfilesDriveLocalTensorObservations(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	plan := openPlaneMetal(t, field)
	got, err := plan.At(0.125)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, got, 32)
	tensor, err := field.Tensor(got.Plus[0], got.Cross[0])
	if err != nil {
		t.Fatal(err)
	}
	response, err := tensor.DifferentialStrain([3]float64{1, 0, 0}, [3]float64{0, 1, 0})
	if err != nil {
		t.Fatal(err)
	}
	planeClose(t, response, got.Plus[0], 1e-21)
	offset, err := tensor.Offset([3]float64{2, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	planeClose(t, offset[0], got.Plus[0], 1e-21)
	planeClose(t, offset[1], got.Cross[0], 1e-21)
}

// anti-test of TestGravityMetalProfilesDriveLocalTensorObservations
// Closing and missing resources cannot silently select the CPU model.
func TestGravityMetalMissingResourcesRefuse(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	closed := openPlaneMetal(t, field)
	closed.Close()
	closed.Close()
	for _, plan := range []*MetalField{nil, {}, closed} {
		plan.Close()
		for _, seconds := range []float64{0, 1} {
			got, err := plan.At(seconds)
			if err == nil || got.Plus != nil || got.Cross != nil || got.PlusRate != nil || got.CrossRate != nil {
				t.Fatal("non-live device plan accepted a query")
			}
		}
	}
	for _, field := range []*Field{nil, {}} {
		plan, err := field.OpenMetal()
		if plan != nil {
			plan.Close()
		}
		if err == nil || plan != nil {
			t.Fatal("uninitialized physical field opened Metal")
		}
	}
}

// Independent profile copies must remain stable across later device queries.
func TestGravityMetalSnapshotsAreIndependent(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	plan := openPlaneMetal(t, field)
	first, err := plan.At(0.125)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, first, 32)
	before := slices.Clone(first.Plus)
	first.Cross[0] = 99
	later, err := plan.At(-0.25)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, later, 32)
	if !slices.Equal(first.Plus, before) {
		t.Fatal("later query changed an earlier profile")
	}
	again, err := plan.At(0.125)
	if err != nil {
		t.Fatal(err)
	}
	want, err := field.At(0.125)
	if err != nil {
		t.Fatal(err)
	}
	requirePlaneMetal(t, again, want, 32)
}

// anti-test of TestGravityMetalSnapshotsAreIndependent
// Finite rates which CPU can store must still fit the device spectrum range.
func TestGravityMetalRefusesUnrepresentableRates(t *testing.T) {
	conf := planeConfig()
	initial := planeInitial(conf)
	for i := range initial.PlusRate {
		initial.PlusRate[i] = 1e100
	}
	field := openPlane(t, conf, initial)
	plan, err := field.OpenMetal()
	if plan != nil {
		plan.Close()
	}
	if err == nil || plan != nil {
		t.Fatal("Metal accepted unrepresentable gravitational rates")
	}
}

// The comparison includes profile unpacking and physical regime validation.
func BenchmarkGravityMetalComparison(b *testing.B) {
	conf := planeConfig()
	conf.Samples = 32768
	conf.MaxSamples = 32768
	field, err := New(conf, planeInitial(conf))
	if err != nil {
		b.Fatal(err)
	}
	for _, gpu := range []bool{false, true} {
		name := "cpu"
		if gpu {
			name = "metal"
		}
		b.Run(name, func(b *testing.B) {
			at := field.At
			if gpu {
				at = openPlaneMetal(b, field).At
			}
			if _, err := at(0.125); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for range b.N {
				got, err := at(0.125)
				if err != nil || len(got.Plus) != conf.Samples || len(got.CrossRate) != conf.Samples {
					b.Fatalf("propagate: %v", err)
				}
			}
		})
	}
}
