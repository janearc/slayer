package gravitational

import (
	"math"
	"pgregory.net/rapid"
	"slices"
	"testing"
)

func planeConfig() Config {
	return Config{Samples: 32, MaxSamples: 32, LengthMetres: LightSpeed, Direction: [3]float64{0, 0, 1}, MaxStrain: 0.01}
}
func planeInitial(conf Config) Profiles {
	state := Profiles{Plus: make([]float64, conf.Samples), Cross: make([]float64, conf.Samples), PlusRate: make([]float64, conf.Samples), CrossRate: make([]float64, conf.Samples)}
	const amplitude = 1e-21
	omega := 2 * math.Pi * LightSpeed / conf.LengthMetres
	for i := range conf.Samples {
		phase := 2 * math.Pi * float64(i) / float64(conf.Samples)
		state.Plus[i] = amplitude * math.Cos(phase)
		state.Cross[i] = 0.5 * amplitude * math.Sin(phase)
		state.PlusRate[i] = amplitude * omega * math.Sin(phase)
		state.CrossRate[i] = -0.5 * amplitude * omega * math.Cos(phase)
	}
	return state
}
func openPlane(t testing.TB, conf Config, initial Profiles) *Field {
	t.Helper()
	field, err := New(conf, initial)
	if err != nil || field == nil {
		t.Fatalf("New: %v, %v", field, err)
	}
	return field
}
func planeClose(t testing.TB, got, want, scale float64) {
	t.Helper()
	if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > 2e-11*scale {
		t.Fatalf("got %g want %g, scale %g", got, want, scale)
	}
}
func planeProfiles(t testing.TB, got Profiles, count int) {
	t.Helper()
	for _, values := range [][]float64{got.Plus, got.Cross, got.PlusRate, got.CrossRate} {
		if len(values) != count {
			t.Fatalf("profile length %d, want %d", len(values), count)
		}
	}
}

// Both Cauchy components travel at the defined speed, at real strain scale.
func TestVacuumPolarizationsTravelAtLightSpeed(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	for _, seconds := range []float64{0, 0.125, -0.25} {
		got, err := field.At(seconds)
		if err != nil {
			t.Fatal(err)
		}
		planeProfiles(t, got, 32)
		for i := range 32 {
			phase := 2 * math.Pi * (float64(i)/32 - seconds)
			planeClose(t, got.Plus[i], 1e-21*math.Cos(phase), 1e-21)
			planeClose(t, got.Cross[i], 0.5e-21*math.Sin(phase), 1e-21)
			planeClose(t, got.PlusRate[i], 1e-21*2*math.Pi*math.Sin(phase), 1e-20)
			planeClose(t, got.CrossRate[i], -0.5e-21*2*math.Pi*math.Cos(phase), 1e-20)
		}
	}
}

// anti-test of TestVacuumPolarizationsTravelAtLightSpeed
// Input and returned arrays cannot alter the initial profiles.
func TestVacuumProfilesAreIndependentCopies(t *testing.T) {
	conf := planeConfig()
	initial := planeInitial(conf)
	before := slices.Clone(initial.Plus)
	field := openPlane(t, conf, initial)
	initial.Plus[0] = 0.005
	first, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, first, 32)
	if !slices.Equal(first.Plus, before) {
		t.Fatal("initial profile aliases caller memory")
	}
	first.Plus[0] = 0.002
	again, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(again.Plus, before) {
		t.Fatal("snapshots alias stored profiles")
	}
}

// Canonical plus stretches x and contracts y; cross occupies xy and yx.
func TestCanonicalPolarizationTensorAndDetectorResponse(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	tensor, err := field.Tensor(0.003, 0.004)
	if err != nil {
		t.Fatal(err)
	}
	want := Tensor{{0.003, 0.004, 0}, {0.004, -0.003, 0}, {0, 0, 0}}
	for row := range 3 {
		for column := range 3 {
			planeClose(t, tensor[row][column], want[row][column], 1)
		}
	}
	response, err := tensor.DifferentialStrain([3]float64{2, 0, 0}, [3]float64{0, 3, 0})
	if err != nil {
		t.Fatal(err)
	}
	planeClose(t, response, 0.003, 1)
	diagonal, err := tensor.DifferentialStrain([3]float64{1, 1, 0}, [3]float64{-1, 1, 0})
	if err != nil {
		t.Fatal(err)
	}
	planeClose(t, diagonal, 0.004, 1)
	offset, err := tensor.Offset([3]float64{2, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range [3]float64{0.003, 0.004, 0} {
		planeClose(t, offset[i], want, 1)
	}
}

// anti-test of TestCanonicalPolarizationTensorAndDetectorResponse
// Arbitrary finite directions retain transversality and zero trace.
func TestRotatedPolarizationsRemainTransverseTraceless(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conf := planeConfig()
		for i := range 3 {
			conf.Direction[i] = rapid.Float64Range(-10, 10).Draw(t, "direction")
		}
		norm := math.Hypot(math.Hypot(conf.Direction[0], conf.Direction[1]), conf.Direction[2])
		if norm == 0 {
			conf.Direction[2] = 1
			norm = 1
		}
		field, err := New(conf, planeInitial(conf))
		if err != nil || field == nil {
			t.Fatalf("New: %v", err)
		}
		tensor, err := field.Tensor(0.003, -0.004)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(tensor[0][0]+tensor[1][1]+tensor[2][2]) > 2e-14 {
			t.Fatal("nonzero tensor trace")
		}
		for row := range 3 {
			var longitudinal float64
			for column := range 3 {
				value := tensor[row][column]
				if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value-tensor[column][row]) > 2e-14 {
					t.Fatal("non-finite or asymmetric tensor")
				}
				longitudinal += value * conf.Direction[column] / norm
			}
			if math.Abs(longitudinal) > 2e-14 {
				t.Fatal("longitudinal metric perturbation")
			}
		}
	})
}

// Weak-field and finite-time refusals leave the immutable profile usable.
func TestVacuumQueriesRefuseOutOfRegimeWithoutMutation(t *testing.T) {
	conf := planeConfig()
	initial := planeInitial(conf)
	for i := range initial.Plus {
		initial.Plus[i] = 0
		initial.PlusRate[i] = 0.02
		initial.Cross[i] = 0
		initial.CrossRate[i] = 0
	}
	field := openPlane(t, conf, initial)
	for _, seconds := range []float64{1, math.NaN(), math.Inf(1), math.MaxFloat64} {
		got, err := field.At(seconds)
		if err == nil || got.Plus != nil || got.Cross != nil || got.PlusRate != nil || got.CrossRate != nil {
			t.Fatalf("accepted invalid/out-of-regime query %g", seconds)
		}
	}
	got, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, got, 32)
	if !slices.Equal(got.PlusRate, initial.PlusRate) {
		t.Fatal("refusal changed initial rate")
	}
	for _, empty := range []*Field{nil, {}} {
		if _, err := empty.At(0); err == nil {
			t.Fatal("empty field accepted time query")
		}
		if _, err := empty.Tensor(0, 0); err == nil {
			t.Fatal("empty field accepted tensor query")
		}
	}
}

// anti-test of TestVacuumQueriesRefuseOutOfRegimeWithoutMutation
// Finite direction components can have an overflowing raw magnitude.
func TestVacuumDirectionNormalizationAvoidsUnnecessaryOverflow(t *testing.T) {
	conf := planeConfig()
	conf.Direction = [3]float64{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}
	field := openPlane(t, conf, planeInitial(conf))
	tensor, err := field.Tensor(0.003, 0.004)
	if err != nil {
		t.Fatal(err)
	}
	for row := range 3 {
		planeClose(t, tensor[row][0]+tensor[row][1]+tensor[row][2], 0, 1)
	}
}

// Malformed data and unrepresentable derived frequencies are refused.
func TestVacuumNewRefusesInvalidInputs(t *testing.T) {
	for _, which := range []string{"size", "budget", "length", "direction", "nan direction", "cap", "large cap", "short", "nan data", "infinite rate", "strain", "frequency", "frequency product", "spectral rate range"} {
		t.Run(which, func(t *testing.T) {
			conf := planeConfig()
			initial := planeInitial(conf)
			switch which {
			case "size":
				conf.Samples = 3
			case "budget":
				conf.MaxSamples = 1
			case "length":
				conf.LengthMetres = 0
			case "direction":
				conf.Direction = [3]float64{}
			case "nan direction":
				conf.Direction[0] = math.NaN()
			case "cap":
				conf.MaxStrain = 0
			case "large cap":
				conf.MaxStrain = 0.1
			case "short":
				initial.CrossRate = initial.CrossRate[:3]
			case "nan data":
				initial.Plus[0] = math.NaN()
			case "infinite rate":
				initial.CrossRate[0] = math.Inf(1)
			case "strain":
				initial.Plus[0] = 0.02
			case "frequency":
				conf.LengthMetres = math.SmallestNonzeroFloat64
			case "frequency product":
				conf.LengthMetres = 1e-300
			case "spectral rate range":
				for i := range initial.PlusRate {
					initial.PlusRate[i] = math.MaxFloat64 / 2
				}
			}
			field, err := New(conf, initial)
			if err == nil || field != nil {
				t.Fatal("invalid vacuum field accepted")
			}
		})
	}
}

// anti-test of TestVacuumNewRefusesInvalidInputs
// A strain may be tiny while its finite rate is physically resolved.
func TestVacuumZeroInitialStrainCanCarryIndependentRate(t *testing.T) {
	conf := planeConfig()
	initial := planeInitial(conf)
	for i := range initial.Plus {
		initial.Plus[i] = 0
		initial.Cross[i] = 0
		initial.CrossRate[i] = 0
		initial.PlusRate[i] = 1e-21 * math.Cos(2*math.Pi*float64(i)/32)
	}
	field := openPlane(t, conf, initial)
	got, err := field.At(0.25)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, got, 32)
	for i := range got.Plus {
		turn := float64(i)
		planeClose(t, got.Plus[i], 1e-21*math.Cos(2*math.Pi*turn/32)/(2*math.Pi), 1e-21)
	}
}

// Invalid tensor queries and observation geometry must return no result.
func TestVacuumObservationsRefuseInvalidAndOverflowingInputs(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	for _, pair := range [][2]float64{{math.NaN(), 0}, {0, math.Inf(1)}, {0.009, 0.009}} {
		tensor, err := field.Tensor(pair[0], pair[1])
		if err == nil || tensor != (Tensor{}) {
			t.Fatal("invalid tensor query accepted")
		}
	}
	tensor := Tensor{{0.003, 0, 0}, {0, -0.003, 0}, {0, 0, 0}}
	for _, direction := range [][3]float64{{}, {math.NaN(), 0, 1}, {0, math.Inf(1), 0}} {
		response, err := tensor.DifferentialStrain(direction, [3]float64{0, 1, 0})
		if err == nil || response != 0 {
			t.Fatal("invalid detector arm accepted")
		}
	}
	for _, position := range [][3]float64{{math.NaN(), 0, 0}, {0, 0, math.Inf(1)}} {
		offset, err := tensor.Offset(position)
		if err == nil || offset != ([3]float64{}) {
			t.Fatal("non-finite position accepted")
		}
	}
	bad := Tensor{{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}, {0, 0, 0}, {0, 0, 0}}
	if value, err := bad.Offset([3]float64{2, 2, 2}); err == nil || value != ([3]float64{}) {
		t.Fatal("overflowing offset accepted")
	}
}

// anti-test of TestVacuumObservationsRefuseInvalidAndOverflowingInputs
// Identical arm directions cancel, and a transverse wave leaves an axial
// separation unchanged to first order.
func TestVacuumDetectorAndOffsetKeepTheirNullDirections(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	tensor, err := field.Tensor(0.003, 0.004)
	if err != nil {
		t.Fatal(err)
	}
	response, err := tensor.DifferentialStrain([3]float64{1, 2, 3}, [3]float64{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	planeClose(t, response, 0, 1)
	offset, err := tensor.Offset([3]float64{0, 0, 2})
	if err != nil || offset != ([3]float64{}) {
		t.Fatal("axial offset must vanish")
	}
}

// Distinct modes and opposite travel directions must keep their own
// frequencies and initial rates, including the cross polarization.
func TestVacuumWavePacketKeepsModesAndTravelDirections(t *testing.T) {
	conf := planeConfig()
	initial := planeInitial(conf)
	const amplitude = 1e-21
	for i := range conf.Samples {
		phase := 2 * math.Pi * float64(i) / float64(conf.Samples)
		initial.Plus[i] = amplitude * (0.7*math.Cos(3*phase) + 0.3*math.Sin(2*phase))
		initial.PlusRate[i] = amplitude * 2 * math.Pi * (2.1*math.Sin(3*phase) + 0.6*math.Cos(2*phase))
		initial.Cross[i] = -0.5 * amplitude * math.Cos(5*phase)
		initial.CrossRate[i] = -2.5 * amplitude * 2 * math.Pi * math.Sin(5*phase)
	}
	field := openPlane(t, conf, initial)
	seconds := 0.07
	got, err := field.At(seconds)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, got, conf.Samples)
	for i := range conf.Samples {
		phase := 2 * math.Pi * float64(i) / float64(conf.Samples)
		right, left := phase-2*math.Pi*seconds, phase+2*math.Pi*seconds
		planeClose(t, got.Plus[i], amplitude*(0.7*math.Cos(3*right)+0.3*math.Sin(2*left)), amplitude)
		planeClose(t, got.Cross[i], -0.5*amplitude*math.Cos(5*right), amplitude)
		planeClose(t, got.PlusRate[i], amplitude*2*math.Pi*(2.1*math.Sin(3*right)+0.6*math.Cos(2*left)), 1e-19)
		planeClose(t, got.CrossRate[i], -2.5*amplitude*2*math.Pi*math.Sin(5*right), 1e-19)
	}
}

// anti-test of TestVacuumWavePacketKeepsModesAndTravelDirections
// Evolving one polarization cannot create an orthogonal polarization.
func TestVacuumWavePacketDoesNotCreateCrossPolarization(t *testing.T) {
	conf := planeConfig()
	initial := planeInitial(conf)
	for i := range conf.Samples {
		initial.Cross[i] = 0
		initial.CrossRate[i] = 0
	}
	field := openPlane(t, conf, initial)
	got, err := field.At(0.07)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, got, conf.Samples)
	for i := range conf.Samples {
		planeClose(t, got.Cross[i], 0, 1e-21)
		planeClose(t, got.CrossRate[i], 0, 1e-20)
	}
}

// A manually supplied non-finite perturbation cannot pass by hiding in
// a detector null direction or zero displacement position.
func TestTensorRefusesNonFiniteEvenForNullObservations(t *testing.T) {
	tensor := Tensor{}
	tensor[2][2] = math.NaN()
	if value, err := tensor.DifferentialStrain([3]float64{1, 0, 0}, [3]float64{1, 0, 0}); err == nil || value != 0 {
		t.Fatal("non-finite tensor accepted for null detector")
	}
	if value, err := tensor.Offset([3]float64{}); err == nil || value != ([3]float64{}) {
		t.Fatal("non-finite tensor accepted at origin")
	}
	valid := Tensor{}
	if value, err := valid.DifferentialStrain([3]float64{1, 0, 0}, [3]float64{}); err == nil || value != 0 {
		t.Fatal("invalid second arm accepted")
	}
	huge := Tensor{{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}, {math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}, {math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}}
	if value, err := huge.DifferentialStrain([3]float64{1, 1, 1}, [3]float64{1, -1, 0}); err == nil || value != 0 {
		t.Fatal("overflowing detector response accepted")
	}
}

// anti-test of TestTensorRefusesNonFiniteEvenForNullObservations
// A finite zero perturbation produces exact zero observations even for
// extreme finite geometric scales and direction magnitudes.
func TestZeroTensorKeepsExactNullObservations(t *testing.T) {
	tensor := Tensor{}
	offset, err := tensor.Offset([3]float64{math.MaxFloat64, -math.MaxFloat64, 0})
	if err != nil || offset != ([3]float64{}) {
		t.Fatal("zero tensor lost zero displacement")
	}
	response, err := tensor.DifferentialStrain([3]float64{math.MaxFloat64, 0, 0}, [3]float64{0, math.SmallestNonzeroFloat64, 0})
	if err != nil || response != 0 {
		t.Fatal("zero tensor lost zero detector response")
	}
}

// anti-test of TestVacuumProfilesAreIndependentCopies
// A malformed shared-propagator snapshot cannot leak a partial profile or
// alter a later valid query. The CPU and Metal adapters share this boundary.
func TestVacuumUnpackRefusesMalformedSharedState(t *testing.T) {
	conf := planeConfig()
	field := openPlane(t, conf, planeInitial(conf))
	valid, err := field.wave.At(0)
	if err != nil {
		t.Fatal(err)
	}
	for _, which := range []string{"value", "rate"} {
		bad := valid
		if which == "value" {
			bad.Value = bad.Value[:conf.Samples]
		} else {
			bad.Rate = bad.Rate[:conf.Samples]
		}
		got, err := field.unpack(bad)
		if err == nil || got.Plus != nil || got.Cross != nil || got.PlusRate != nil || got.CrossRate != nil {
			t.Fatal("malformed wave state produced a partial profile")
		}
	}
	got, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	planeProfiles(t, got, conf.Samples)
}
