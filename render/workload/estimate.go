package workload

import (
	"fmt"
	"math"
)

// EstimateStatus is an advisory comparison with caller-supplied capacity.
type EstimateStatus string

const (
	EstimateUnknown        EstimateStatus = "unknown"
	EstimateWithinCapacity EstimateStatus = "within_capacity"
	EstimateOverCapacity   EstimateStatus = "over_capacity"
)

// EstimateInputs describes the host baseline and additional independent
// processes being considered. CurrentUseBytes must already include all work
// currently running; AdditionalProcesses counts only new process copies.
// FramesPerProcess is recorded as context and is never a memory multiplier:
// frames and workers may share one process. A zero headroom pointer is an
// explicit request for no additional margin; nil means unknown.
type EstimateInputs struct {
	Identity            Identity
	HostCapacityBytes   *uint64
	CurrentUseBytes     *uint64
	AdditionalProcesses uint64
	FramesPerProcess    uint64
	HeadroomBytes       *uint64
}

// EstimateReport preserves measured peaks separately and describes the
// assumptions used by the arithmetic. Unknown means no result should be used
// even as evidence-based advice. This report is never an admission guarantee.
type EstimateReport struct {
	Status              EstimateStatus `json:"status"`
	Identity            Identity       `json:"identity"`
	Peaks               Peaks          `json:"observed_peaks"`
	EvidenceSource      string         `json:"whole_process_evidence_source,omitempty"`
	PerProcessBytes     *uint64        `json:"per_process_bytes,omitempty"`
	AdditionalProcesses uint64         `json:"additional_processes"`
	FramesPerProcess    uint64         `json:"frames_per_process"`
	AdditionalBytes     *uint64        `json:"additional_process_bytes,omitempty"`
	CurrentUseBytes     *uint64        `json:"current_use_bytes,omitempty"`
	HeadroomBytes       *uint64        `json:"headroom_bytes,omitempty"`
	RequiredBytes       *uint64        `json:"required_bytes,omitempty"`
	HostCapacityBytes   *uint64        `json:"host_capacity_bytes,omitempty"`
	Unknown             []string       `json:"unknown,omitempty"`
	Unavailable         []string       `json:"unavailable_metrics,omitempty"`
	Assumptions         []string       `json:"assumptions"`
}

// Estimate compares caller-supplied host inputs with exact-key whole-process
// peaks in profile. The profile identity must equal inputs.Identity, which is
// expected to cover source, build, device, render settings and concurrency.
// It assumes the measurements represent each independent process being added.
//
// It does not model child/group processes, infer shared memory, or sum RSS,
// native and scratch fields into physical memory. FramesPerProcess is
// disclosed but not multiplied because frames and workers may share one
// process. This is evidence-based advice, never a hard upper bound, a safety
// guarantee, or authorization to start work or alter render settings.
func Estimate(profile Profile, inputs EstimateInputs) EstimateReport {
	report := EstimateReport{
		Status:              EstimateUnknown,
		Identity:            inputs.Identity,
		AdditionalProcesses: inputs.AdditionalProcesses,
		FramesPerProcess:    inputs.FramesPerProcess,
		HostCapacityBytes:   cloneUint64(inputs.HostCapacityBytes),
		CurrentUseBytes:     cloneUint64(inputs.CurrentUseBytes),
		HeadroomBytes:       cloneUint64(inputs.HeadroomBytes),
		Assumptions: []string{
			"Current use includes all work already running; additional processes are new independent copies.",
			"Frames and workers may share a process; frames per process are not multiplied.",
			"The selected measurement must represent each added process and exact source, build, device, render and concurrency identity.",
			"Group RSS, native and scratch are reported as separate peaks and are not added to physical footprint.",
			"This is evidence-based advice, not an admission guarantee or a bound on unobserved memory.",
		},
	}
	report.Peaks = profile.Peaks()
	unknown := make([]string, 0, 8)
	if _, err := inputs.Identity.Key(); err != nil {
		unknown = append(unknown, "expected identity is incomplete or invalid")
	}
	if _, err := profile.Identity.Key(); err != nil {
		unknown = append(unknown, "profile identity is incomplete or invalid")
	} else if profile.Identity != inputs.Identity {
		unknown = append(unknown, "profile identity does not match requested identity")
	}
	if len(profile.Observations) == 0 {
		unknown = append(unknown, "profile has no observations")
	} else {
		for i, observation := range profile.Observations {
			if err := observation.validate(); err != nil {
				unknown = append(unknown, fmt.Sprintf("observation %d is invalid: %v", i, err))
			}
		}
	}
	if inputs.HostCapacityBytes == nil {
		unknown = append(unknown, "host capacity is unknown")
	}
	if inputs.CurrentUseBytes == nil {
		unknown = append(unknown, "current host use is unknown")
	}
	if inputs.HeadroomBytes == nil {
		unknown = append(unknown, "explicit headroom is unknown")
	}
	if inputs.AdditionalProcesses == 0 {
		unknown = append(unknown, "additional independent process count must be positive")
	}
	if inputs.FramesPerProcess == 0 {
		unknown = append(unknown, "frames per process must be positive")
	}
	for _, metric := range []struct {
		name  string
		value *uint64
	}{
		{"physical footprint", report.Peaks.Physical},
		{"physical lifetime peak", report.Peaks.PhysicalPeak},
		{"group RSS", report.Peaks.GroupRSS},
		{"native allocation", report.Peaks.Native},
		{"scratch allocation", report.Peaks.Scratch},
	} {
		if metric.value == nil {
			report.Unavailable = append(report.Unavailable, metric.name)
		}
	}
	if report.Peaks.PhysicalPeak != nil {
		value := *report.Peaks.PhysicalPeak
		report.PerProcessBytes, report.EvidenceSource = &value, "physical_lifetime_peak"
	}
	if report.PerProcessBytes == nil {
		if report.Peaks.Physical != nil {
			unknown = append(unknown, "profile has current physical footprints but no lifetime high-water mark")
		} else {
			unknown = append(unknown, "profile has no whole-process physical footprint observations")
		}
	}
	if len(unknown) > 0 {
		report.Unknown = unknown
		return report
	}
	additional, ok := checkedMultiply(*report.PerProcessBytes, inputs.AdditionalProcesses)
	if !ok {
		report.Unknown = []string{"additional process memory multiplication overflowed uint64"}
		return report
	}
	required, ok := checkedAdd(*inputs.CurrentUseBytes, additional)
	if ok {
		required, ok = checkedAdd(required, *inputs.HeadroomBytes)
	}
	if !ok {
		report.Unknown = []string{"required memory addition overflowed uint64"}
		return report
	}
	report.AdditionalBytes = &additional
	report.RequiredBytes = &required
	if required > *inputs.HostCapacityBytes {
		report.Status = EstimateOverCapacity
	} else {
		report.Status = EstimateWithinCapacity
	}
	return report
}

// cloneUint64 returns a pointer to a copy of value, or nil when value is nil.
func cloneUint64(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// checkedMultiply returns a times b, and false when the product overflows.
func checkedMultiply(a, b uint64) (uint64, bool) {
	if a != 0 && b > math.MaxUint64/a {
		return 0, false
	}
	return a * b, true
}

// checkedAdd returns a plus b, and false when the sum overflows.
func checkedAdd(a, b uint64) (uint64, bool) {
	if b > math.MaxUint64-a {
		return 0, false
	}
	return a + b, true
}
