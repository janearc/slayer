package flow

import (
	"fmt"
	"math"
)

// MetalFlow owns retained spectral velocity on a Metal device.
// It has one caller at a time and must be closed when finished.
type MetalFlow struct {
	accepted *Flow
	backend  metalStepper
	closed   bool
}

// metalStepper advances a candidate without altering accepted Go state.
type metalStepper interface {
	step(seconds, nextTime float64) (Velocity, Diagnostics, error)
	close()
}

// OpenMetal validates and projects a velocity before preparing Metal.
func OpenMetal(config Config, initial Velocity) (*MetalFlow, error) {
	accepted, err := New(config, initial)
	if err != nil {
		return nil, err
	}
	backend, err := newMetalStepper(config, accepted.velocity)
	if err != nil {
		return nil, err
	}
	return &MetalFlow{accepted: accepted, backend: backend}, nil
}

// Step advances one accepted interval with no intermediate readback.
func (flow *MetalFlow) Step(seconds float64) (Diagnostics, error) {
	if flow == nil || flow.accepted == nil {
		return Diagnostics{}, fmt.Errorf("flow: no live Metal flow")
	}
	previous := flow.accepted.diagnostics
	if flow.closed || flow.backend == nil {
		return previous, fmt.Errorf("flow: Metal flow is closed")
	}
	if !flowFinite(seconds) || seconds <= 0 || seconds > flow.accepted.config.MaxStepSeconds {
		return previous, fmt.Errorf("flow: invalid Metal step %g", seconds)
	}
	rounded := float32(seconds)
	if float32(seconds/6) < 0x1p-126 || math.IsInf(float64(rounded), 0) {
		return previous, fmt.Errorf("flow: step outside float32 range")
	}
	if err := flowGuard(flow.accepted.config, flow.accepted.velocity, seconds); err != nil {
		return previous, err
	}
	nextTime := previous.TimeSeconds + seconds
	if !flowFinite(nextTime) {
		return previous, fmt.Errorf("flow: time overflows")
	}
	velocity, diagnostics, err := flow.backend.step(seconds, nextTime)
	if err != nil {
		flow.Close()
		return previous, err
	}
	flow.accepted.velocity, flow.accepted.diagnostics = velocity, diagnostics
	return diagnostics, nil
}

// Velocity returns independent copies of the last accepted state.
func (flow *MetalFlow) Velocity() Velocity {
	if flow == nil {
		return Velocity{}
	}
	return flow.accepted.Velocity()
}

// Diagnostics returns the last accepted scalar diagnostics.
func (flow *MetalFlow) Diagnostics() Diagnostics {
	if flow == nil {
		return Diagnostics{}
	}
	return flow.accepted.Diagnostics()
}

// Close releases resources while preserving accepted snapshots.
func (flow *MetalFlow) Close() {
	if flow == nil || flow.closed {
		return
	}
	flow.closed = true
	if flow.backend != nil {
		flow.backend.close()
	}
	flow.backend = nil
}
