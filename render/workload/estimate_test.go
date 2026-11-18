package workload

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestEstimateUsesWholeProcessEvidenceAndCountsOnlyIndependentCopies(t *testing.T) {
	identity := testIdentity()
	physical, lifetime := uint64(100), uint64(120)
	groupRSS, native, scratch := uint64(90), uint64(80), uint64(70)
	profile := Profile{Identity: identity, Observations: []Observation{estimateObservation(
		&physical, &lifetime, &groupRSS, &native, &scratch)}}
	capacity, current, headroom := uint64(790), uint64(500), uint64(50)
	got := Estimate(profile, EstimateInputs{Identity: identity,
		HostCapacityBytes: &capacity, CurrentUseBytes: &current,
		AdditionalProcesses: 2, FramesPerProcess: 8, HeadroomBytes: &headroom})
	if got.Status != EstimateWithinCapacity || got.PerProcessBytes == nil || *got.PerProcessBytes != lifetime ||
		got.AdditionalBytes == nil || *got.AdditionalBytes != 240 ||
		got.RequiredBytes == nil || *got.RequiredBytes != capacity {
		t.Fatalf("estimate = %+v", got)
	}
	if got.FramesPerProcess != 8 || got.AdditionalProcesses != 2 {
		t.Fatalf("concurrency assumptions = processes %d, frames/process %d", got.AdditionalProcesses, got.FramesPerProcess)
	}
	if got.Peaks.GroupRSS == nil || *got.Peaks.GroupRSS != groupRSS || got.Peaks.Native == nil || *got.Peaks.Native != native || got.Peaks.Scratch == nil || *got.Peaks.Scratch != scratch {
		t.Fatalf("independent observed peaks not preserved: %+v", got.Peaks)
	}
}

func TestEstimateDoesNotSumSubsetMeasurementsAndDoesNotMultiplyFrames(t *testing.T) {
	identity := testIdentity()
	groupRSS, native, scratch := uint64(100), uint64(80), uint64(70)
	profile := Profile{Identity: identity, Observations: []Observation{estimateObservation(
		nil, nil, &groupRSS, &native, &scratch)}}
	capacity, current, headroom := uint64(1_000), uint64(200), uint64(0)
	got := Estimate(profile, EstimateInputs{Identity: identity,
		HostCapacityBytes: &capacity, CurrentUseBytes: &current,
		AdditionalProcesses: 1, FramesPerProcess: 64, HeadroomBytes: &headroom})
	if got.Status != EstimateUnknown || got.PerProcessBytes != nil || got.RequiredBytes != nil {
		t.Fatalf("subset metrics were promoted to total process demand: %+v", got)
	}
	if len(got.Unknown) == 0 {
		t.Fatal("missing whole-process physical evidence was not reported")
	}

	physical := uint64(300)
	profile.Observations[0] = estimateObservation(&physical, nil, &groupRSS, &native, &scratch)
	got = Estimate(profile, EstimateInputs{Identity: identity,
		HostCapacityBytes: &capacity, CurrentUseBytes: &current,
		AdditionalProcesses: 1, FramesPerProcess: 64, HeadroomBytes: &headroom})
	if got.Status != EstimateUnknown || got.RequiredBytes != nil || !hasUnknown(got, "no lifetime high-water mark") {
		t.Fatalf("current-only footprint gave reassuring status: %+v", got)
	}
	lifetime := uint64(310)
	profile.Observations[0] = estimateObservation(&physical, &lifetime, &groupRSS, &native, &scratch)
	got = Estimate(profile, EstimateInputs{Identity: identity,
		HostCapacityBytes: &capacity, CurrentUseBytes: &current,
		AdditionalProcesses: 1, FramesPerProcess: 64, HeadroomBytes: &headroom})
	if got.Status != EstimateWithinCapacity || got.PerProcessBytes == nil || *got.PerProcessBytes != lifetime || got.RequiredBytes == nil || *got.RequiredBytes != 510 {
		t.Fatalf("shared-process frames changed physical demand: %+v", got)
	}
}

func TestEstimateReportsMissingOrMismatchedInputs(t *testing.T) {
	identity := testIdentity()
	physical, peak := uint64(100), uint64(100)
	profile := Profile{Identity: identity, Observations: []Observation{estimateObservation(
		&physical, &peak, nil, nil, nil)}}
	capacity, current, headroom := uint64(1_000), uint64(0), uint64(0)
	inputs := EstimateInputs{Identity: identity, HostCapacityBytes: &capacity,
		CurrentUseBytes: &current, AdditionalProcesses: 1, FramesPerProcess: 1,
		HeadroomBytes: &headroom}
	cases := []struct {
		name    string
		profile Profile
		inputs  EstimateInputs
	}{
		{"identity mismatch", profile, func() EstimateInputs {
			changed := inputs
			changed.Identity.Render = testIdentity().Source
			return changed
		}()},
		{"no observations", Profile{Identity: identity}, inputs},
		{"no host capacity", profile, func() EstimateInputs { changed := inputs; changed.HostCapacityBytes = nil; return changed }()},
		{"no current use", profile, func() EstimateInputs { changed := inputs; changed.CurrentUseBytes = nil; return changed }()},
		{"no explicit headroom", profile, func() EstimateInputs { changed := inputs; changed.HeadroomBytes = nil; return changed }()},
		{"zero independent processes", profile, func() EstimateInputs { changed := inputs; changed.AdditionalProcesses = 0; return changed }()},
		{"zero frames per process", profile, func() EstimateInputs { changed := inputs; changed.FramesPerProcess = 0; return changed }()},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := Estimate(testCase.profile, testCase.inputs)
			if got.Status != EstimateUnknown || len(got.Unknown) == 0 || got.RequiredBytes != nil {
				t.Fatalf("invalid inputs produced an estimate: %+v", got)
			}
		})
	}
}

func TestEstimateHandlesCapacityAndArithmeticOverflow(t *testing.T) {
	identity := testIdentity()
	perProcess := uint64(math.MaxUint64/2 + 1)
	profile := Profile{Identity: identity, Observations: []Observation{estimateObservation(
		&perProcess, &perProcess, nil, nil, nil)}}
	capacity, current, headroom := uint64(math.MaxUint64), uint64(0), uint64(0)
	inputs := EstimateInputs{Identity: identity, HostCapacityBytes: &capacity,
		CurrentUseBytes: &current, AdditionalProcesses: 2, FramesPerProcess: 1,
		HeadroomBytes: &headroom}
	got := Estimate(profile, inputs)
	if got.Status != EstimateUnknown || !hasUnknown(got, "overflow") {
		t.Fatalf("multiplication overflow not reported: %+v", got)
	}

	perProcess = 1
	profile.Observations[0] = estimateObservation(&perProcess, &perProcess, nil, nil, nil)
	current, headroom = math.MaxUint64, 1
	inputs.CurrentUseBytes, inputs.HeadroomBytes = &current, &headroom
	got = Estimate(profile, inputs)
	if got.Status != EstimateUnknown || !hasUnknown(got, "overflow") {
		t.Fatalf("addition overflow not reported: %+v", got)
	}

	current, headroom = 890, 10
	inputs.CurrentUseBytes, inputs.HeadroomBytes = &current, &headroom
	inputs.AdditionalProcesses = 1
	got = Estimate(profile, inputs)
	if got.Status != EstimateWithinCapacity || got.RequiredBytes == nil || *got.RequiredBytes != 901 {
		t.Fatalf("boundary estimate = %+v", got)
	}
	capacity = 900
	got = Estimate(profile, inputs)
	if got.Status != EstimateOverCapacity {
		t.Fatalf("over-capacity estimate status = %q, want over_capacity", got.Status)
	}
}

func estimateObservation(physical, lifetime, groupRSS, native, scratch *uint64) Observation {
	now := time.Now().UTC()
	wall := time.Millisecond
	return Observation{Run: "estimate-run", Sample: "sample-1", Frame: 0,
		At: now, Cache: "cold", Wall: wall, Physical: physical,
		PhysicalPeak: lifetime, GroupRSS: groupRSS, Native: native, Scratch: scratch}
}

func hasUnknown(estimate EstimateReport, fragment string) bool {
	for _, reason := range estimate.Unknown {
		if strings.Contains(reason, fragment) {
			return true
		}
	}
	return false
}

func testIdentity() Identity {
	return Identity{
		Source: strings.Repeat("1", 64),
		Build:  strings.Repeat("2", 64),
		Device: strings.Repeat("3", 64),
		Render: strings.Repeat("4", 64),
		Probe:  strings.Repeat("5", 64),
	}
}
