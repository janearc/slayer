package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// clearCloudPhaseOrigin is the middle of space.
func clearCloudPhaseOrigin() maths.Vec {
	return maths.V(nought, nought, nought)
}

// clearCloudPhaseValid is the phase asymmetries a cloud takes: a retreating
// one, none, a forward one and the largest below one.
func clearCloudPhaseValid(t *testing.T) []float64 {
	t.Helper()
	return []float64{picked(t, "clear-cloud-phase-retreating"), nought,
		picked(t, "clear-cloud-phase-forward"), math.Nextafter(unity, nought)}
}

// clearCloudPhaseInvalid is the phase asymmetries a cloud refuses: one either
// way, no number, and an infinity either way.
func clearCloudPhaseInvalid() []float64 {
	return []float64{-unity, unity, math.NaN(), math.Inf(unity),
		math.Inf(-unity)}
}

// TestClearCloudExplicitHGValidatesAndDistinguishesZero holds that a cloud
// takes only a phase asymmetry strictly between minus one and one, and says
// a given zero apart from none given.
func TestClearCloudExplicitHGValidatesAndDistinguishesZero(t *testing.T) {
	field := Puff{Radius: picked(t, "clear-cloud-phase-field-radius"),
		Thick: picked(t, "clear-cloud-phase-field-thick")}
	omitted, err := NewClearCloud(field)
	if err != nil {
		t.Fatal(err)
	}
	phase, explicit := omitted.PhaseAsymmetry()
	if phase != nought || explicit {
		t.Fatal("omitted phase acquired an override")
	}
	for _, phase := range clearCloudPhaseValid(t) {
		cloud, err := NewClearCloudWithHenyeyGreenstein(field, phase)
		if err != nil {
			t.Fatal(err)
		}
		got, explicit := cloud.PhaseAsymmetry()
		if !explicit || got != phase {
			t.Fatalf("phase %g read (%g,%t)", phase, got, explicit)
		}
		origin := clearCloudPhaseOrigin()
		if cloud.Density(origin) != omitted.Density(origin) {
			t.Fatal("phase changed density")
		}
	}
	for _, phase := range clearCloudPhaseInvalid() {
		_, refused := NewClearCloudWithHenyeyGreenstein(field, phase)
		if refused == nil {
			t.Fatalf("invalid phase %g accepted", phase)
		}
	}
	forward := picked(t, "clear-cloud-phase-forward")
	if _, err := NewClearCloudWithHenyeyGreenstein(nil, forward); err == nil {
		t.Fatal("phase bypassed field bounds validation")
	}
}

// phaseClockField is a field whose density is one more than the seconds it
// has been ticked to.
type phaseClockField struct{ seconds float64 }

// Bounds is the box a unit either side of the middle.
func (field phaseClockField) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-unity, -unity, -unity),
		Max: maths.V(unity, unity, unity)}
}

// Density is one more than the seconds, wherever it is asked.
func (field phaseClockField) Density(maths.Vec) float64 {
	return field.seconds + unity
}

// Tick is the field at a time, a copy and not the same field.
func (field phaseClockField) Tick(at time.Duration) Medium {
	field.seconds = at.Seconds()
	return field
}

// TestClearCloudTickRetainsExplicitHGAndTicksUnderlyingField holds that a
// ticked cloud keeps the zero phase it was given, ticks the field under it,
// and leaves the cloud it was ticked from as it was.
func TestClearCloudTickRetainsExplicitHGAndTicksUnderlyingField(
	t *testing.T) {
	cloud, err := NewClearCloudWithHenyeyGreenstein(phaseClockField{}, nought)
	if err != nil {
		t.Fatal(err)
	}
	seconds := time.Duration(pickedCount(t, "clear-cloud-phase-tick-seconds"))
	ticked := cloud.Tick(seconds * time.Second).(ClearCloud)
	if phase, explicit := ticked.PhaseAsymmetry(); phase != nought ||
		!explicit {
		t.Fatal("Tick lost explicit isotropic HG")
	}
	origin := clearCloudPhaseOrigin()
	want := picked(t, "clear-cloud-phase-ticked-density")
	if ticked.Density(origin) != want || cloud.Density(origin) != unity {
		t.Fatal("phase wrapper lost immutable field clock")
	}
}
