// Package processinfo holds private native process observation primitives.
package processinfo

import "errors"

// errUsageRead gives a failed native query context without inventing an errno.
var errUsageRead = errors.New("native process usage read failed")

// usageQuery selects one PID without a process enumeration or cached lookup.
type usageQuery struct{ pid int }

// usageState distinguishes an observation from failed or omitted reads.
type usageState uint8

const (
	usageUnknown usageState = iota
	usageFailed
	usageObserved
)

// usageObservation stores raw timestamps, not process ownership.
// A zero exit stamp does not guarantee liveness after the call.
type usageObservation struct {
	pid          int
	born, exited uint64
	state        usageState
	err          error
}

// darwinUsageReader reads the public libproc interface without choosing policy.
type darwinUsageReader struct{}
