package specimen_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/materials/specimen"
	"pgregory.net/rapid"
)

func solids() []specimen.Solid {
	return []specimen.Solid{
		{Name: "matrix", MassKG: 2, DensityKGPerM3: 2000, Basis: "analytic fixture"},
		{Name: "inclusion", MassKG: 3, DensityKGPerM3: 3000, Basis: "analytic fixture"},
	}
}

func pores() specimen.Pores {
	return specimen.Pores{VolumeFraction: 0.2, Saturation: 0.25,
		WaterDensityKGPerM3: 1000, Basis: "analytic fixture"}
}

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > 1e-12*math.Max(1, math.Abs(want)) {
		t.Fatalf("got %.15g, want %.15g", got, want)
	}
}

func TestVolumesAndMasses(t *testing.T) {
	made, err := specimen.New(solids(), pores())
	if err != nil {
		t.Fatal(err)
	}
	// Each solid occupies 1 litre; at 20% porosity bulk volume is 2.5 L.
	near(t, made.VolumeM3(), 0.0025)
	near(t, made.SolidMassKG(), 5)
	near(t, made.WaterMassKG(), 0.125)
	near(t, made.DryDensityKGPerM3(), 2000)
	near(t, made.BulkDensityKGPerM3(), 2050)
	near(t, made.WaterVolumeFraction(), 0.05)
	near(t, made.EmptyVolumeFraction(), 0.15)
	got := made.Phases()
	for index, phase := range got {
		near(t, phase.VolumeFraction, 0.4)
		if phase.Name != solids()[index].Name || phase.Basis != "analytic fixture" {
			t.Fatal(phase)
		}
		near(t, phase.MassKG, solids()[index].MassKG)
		near(t, phase.DensityKGPerM3, solids()[index].DensityKGPerM3)
	}
	if made.Pores() != pores() {
		t.Fatal(made.Pores())
	}
	// Equal mass fractions are not equal volume fractions or arithmetic density.
	equal := solids()
	equal[1].MassKG = 2
	other, err := specimen.New(equal, specimen.Pores{
		WaterDensityKGPerM3: 1000, Basis: "analytic fixture"})
	if err != nil {
		t.Fatal(err)
	}
	near(t, other.Phases()[0].VolumeFraction, 0.6)
	near(t, other.DryDensityKGPerM3(), 2400)
}

func TestSaturationChangesWaterWithoutMovingSolids(t *testing.T) {
	for _, saturation := range []float64{0, 0.25, 1} {
		pore := pores()
		pore.Saturation = saturation
		made, err := specimen.New(solids(), pore)
		if err != nil {
			t.Fatal(err)
		}
		near(t, made.VolumeM3(), 0.0025)
		near(t, made.WaterMassKG(), 0.5*saturation)
		near(t, made.BulkDensityKGPerM3(), 2000+200*saturation)
		near(t, made.Phases()[0].VolumeFraction+made.Phases()[1].VolumeFraction+
			made.WaterVolumeFraction()+made.EmptyVolumeFraction(), 1)
	}
}

func TestInputsAndReadbackCannotMutateSpecimen(t *testing.T) {
	input := solids()
	made, err := specimen.New(input, pores())
	if err != nil {
		t.Fatal(err)
	}
	input[0].MassKG = 200
	got := made.Phases()
	got[0].VolumeFraction = 1
	got[0].Name = "changed"
	near(t, made.Phases()[0].VolumeFraction, 0.4)
	if made.Phases()[0].Name != "matrix" {
		t.Fatal(made.Phases())
	}
}

func TestRejectsMissingOrInvalidPhysicalInputs(t *testing.T) {
	for _, trial := range []struct {
		name, message string
		alter         func(*[]specimen.Solid, *specimen.Pores)
	}{
		{"none", "no solids", func(s *[]specimen.Solid, p *specimen.Pores) { *s = nil }},
		{"name", "name", func(s *[]specimen.Solid, p *specimen.Pores) { (*s)[0].Name = " " }},
		{"padded name", "padded name", func(s *[]specimen.Solid, p *specimen.Pores) { (*s)[0].Name = "matrix " }},
		{"basis", "basis", func(s *[]specimen.Solid, p *specimen.Pores) { (*s)[0].Basis = "" }},
		{"duplicate", "duplicate", func(s *[]specimen.Solid, p *specimen.Pores) { (*s)[1].Name = (*s)[0].Name }},
		{"pore basis", "basis", func(s *[]specimen.Solid, p *specimen.Pores) { p.Basis = "" }},
	} {
		t.Run(trial.name, func(t *testing.T) {
			s, p := solids(), pores()
			trial.alter(&s, &p)
			_, err := specimen.New(s, p)
			if err == nil || !strings.Contains(err.Error(), trial.message) {
				t.Fatal(err)
			}
		})
	}
	for _, invalid := range []float64{-1, 0, math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, field := range []string{"mass", "density", "water density"} {
			s, p := solids(), pores()
			want := "invalid solid " + field
			switch field {
			case "mass":
				s[0].MassKG = invalid
			case "density":
				s[0].DensityKGPerM3 = invalid
			case "water density":
				p.WaterDensityKGPerM3 = invalid
				want = "invalid pore water density"
			}
			if _, err := specimen.New(s, p); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s %v: got %v, want %s", field, invalid, err, want)
			}
		}
	}
	for _, invalid := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		p := pores()
		p.VolumeFraction = invalid
		if _, err := specimen.New(solids(), p); err == nil || !strings.Contains(err.Error(), "invalid pore volume fraction") {
			t.Fatalf("porosity %v: %v", invalid, err)
		}
		p = pores()
		p.Saturation = invalid
		if _, err := specimen.New(solids(), p); err == nil || !strings.Contains(err.Error(), "invalid pore saturation") {
			t.Fatalf("saturation %v: %v", invalid, err)
		}
	}
	p := pores()
	p.VolumeFraction = 1
	if _, err := specimen.New(solids(), p); err == nil || !strings.Contains(err.Error(), "invalid pore volume fraction") {
		t.Fatalf("all pores: %v", err)
	}
}

func TestRejectsUnrepresentableSpecimens(t *testing.T) {
	for _, s := range [][]specimen.Solid{
		{{Name: "overflow", MassKG: math.MaxFloat64, DensityKGPerM3: 0.01, Basis: "test"}},
		{{Name: "underflow", MassKG: math.SmallestNonzeroFloat64, DensityKGPerM3: math.MaxFloat64, Basis: "test"}},
	} {
		if _, err := specimen.New(s, pores()); err == nil || !strings.Contains(err.Error(), "unrepresentable solid volume") {
			t.Fatalf("solid volume: %v", err)
		}
	}
}

// An unconstructed value is invalid, rather than a silent zero-density solid.
func TestZeroSpecimenDensitiesAreNotPhysical(t *testing.T) {
	var zero specimen.Specimen
	if !math.IsNaN(zero.DryDensityKGPerM3()) || !math.IsNaN(zero.BulkDensityKGPerM3()) {
		t.Fatal("zero specimen reports a physical density")
	}
}

// Each fixture reaches one late refusal while the preceding calculations
// remain representable. The error must identify the failing quantity.
func TestEachBulkCalculationHasItsOwnRefusal(t *testing.T) {
	for _, trial := range []struct {
		name                               string
		masses, densities                  []float64
		porosity, saturation, waterDensity float64
	}{
		{"bulk volume", []float64{math.MaxFloat64}, []float64{1}, 0.5, 0, 1000},
		{"solid mass sum", []float64{math.MaxFloat64, math.MaxFloat64}, []float64{math.MaxFloat64, math.MaxFloat64}, 0, 0, 1000},
		{"dry density", []float64{1e-320}, []float64{1e-320}, math.Nextafter(1, 0), 0.25, 1000},
		{"water mass", []float64{1e200}, []float64{1}, 0.5, 1, 1e200},
		// A convex-density argument holds in real arithmetic, but rounding a
		// subnormal volume can make the computed wet sum overflow alone.
		{"bulk density", []float64{1}, []float64{math.MaxFloat64}, 0.001, 1, math.MaxFloat64},
	} {
		t.Run(trial.name, func(t *testing.T) {
			var input []specimen.Solid
			for index, mass := range trial.masses {
				input = append(input, specimen.Solid{Name: string(rune('a' + index)),
					MassKG: mass, DensityKGPerM3: trial.densities[index], Basis: "numeric limit fixture"})
			}
			p := specimen.Pores{VolumeFraction: trial.porosity, Saturation: trial.saturation,
				WaterDensityKGPerM3: trial.waterDensity, Basis: "numeric limit fixture"}
			_, err := specimen.New(input, p)
			want := "unrepresentable " + trial.name
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("got %v, want %s", err, want)
			}
		})
	}
}

func TestConservationAcrossMassDensityAndSaturation(t *testing.T) {
	rapid.Check(t, func(draw *rapid.T) {
		input := solids()
		for index := range input {
			input[index].MassKG = rapid.Float64Range(0.001, 100).Draw(draw, "mass")
			input[index].DensityKGPerM3 = rapid.Float64Range(100, 10000).Draw(draw, "density")
		}
		p := pores()
		p.VolumeFraction = rapid.Float64Range(0, 0.9).Draw(draw, "porosity")
		p.Saturation = rapid.Float64Range(0, 1).Draw(draw, "saturation")
		made, err := specimen.New(input, p)
		if err != nil {
			draw.Fatal(err)
		}
		// Expectations use only drawn inputs, never specimen readback.
		solidVolume, mass := 0.0, 0.0
		for _, solid := range input {
			solidVolume += solid.MassKG / solid.DensityKGPerM3
			mass += solid.MassKG
		}
		wantVolume := solidVolume / (1 - p.VolumeFraction)
		wantWater := wantVolume * p.VolumeFraction * p.Saturation * p.WaterDensityKGPerM3
		for _, value := range []struct{ got, want float64 }{
			{made.VolumeM3(), wantVolume},
			{made.WaterMassKG(), wantWater},
			{made.WaterVolumeFraction(), p.VolumeFraction * p.Saturation},
			{made.EmptyVolumeFraction(), p.VolumeFraction * (1 - p.Saturation)},
			{made.DryDensityKGPerM3(), mass / wantVolume},
			{made.BulkDensityKGPerM3(), (mass + wantWater) / wantVolume},
		} {
			if !(math.Abs(value.got-value.want) <= 1e-12*math.Max(1, math.Abs(value.want))) {
				draw.Fatalf("got %g, want %g from inputs", value.got, value.want)
			}
		}
		for index, phase := range made.Phases() {
			want := input[index].MassKG / input[index].DensityKGPerM3 / wantVolume
			if !(math.Abs(phase.VolumeFraction-want) <= 1e-12) {
				draw.Fatal("wrong phase fraction")
			}
		}
		for index := range input {
			input[index].MassKG *= 7
		}
		scaled, err := specimen.New(input, p)
		if err != nil {
			draw.Fatal(err)
		}
		if math.Abs(scaled.BulkDensityKGPerM3()-made.BulkDensityKGPerM3()) > 1e-9 {
			draw.Fatal("density depends on specimen size")
		}
	})
}
