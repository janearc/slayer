//go:build darwin

package fuelbed

import (
	"errors"
	"github.com/janearc/slayer/metal"
	"strings"
	"testing"
)

// interruptedSession closes an actual Metal session at a resource boundary.
// Arithmetic is still supplied by the real stencil in conformance tests.
type interruptedSession struct {
	*metal.Session
	bufferCalls, failBuffer, closes int
	failDispatch, failCommit        bool
}

func (s *interruptedSession) Buffer(count int) (metal.Buffer, error) {
	s.bufferCalls++
	if s.bufferCalls == s.failBuffer {
		s.Session.Close()
	}
	return s.Session.Buffer(count)
}
func (s *interruptedSession) Dispatch(name string, w, h int, p []byte, b ...metal.Buffer) error {
	if s.failDispatch {
		s.Session.Close()
	}
	return s.Session.Dispatch(name, w, h, p, b...)
}
func (s *interruptedSession) Commit() error {
	if s.failCommit {
		s.Session.Close()
	}
	return s.Session.Commit()
}
func (s *interruptedSession) Close() { s.closes++; s.Session.Close() }
func failureSpec() Spec {
	return Spec{Width: 1, Depth: 1, Cell: 1, Step: .1, Heating: 1, Activation: 550, Consumption: .1, Ambient: 300, Initial: []Cell{{Excess: 900, Fuel: 1}}}
}
func openFailureSession(t *testing.T) *interruptedSession {
	t.Helper()
	real, err := metal.OpenSession(stencil, "fuel_step")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(real.Close)
	return &interruptedSession{Session: real}
}
func TestMetalSolverReleasesPartialAllocations(t *testing.T) {
	for _, allocation := range []int{1, 2} {
		session := openFailureSession(t)
		session.failBuffer = allocation
		device, err := newDevice(failureSpec(), session)
		if device != nil || err == nil || !strings.Contains(err.Error(), "session is closed") || session.closes != 1 || session.bufferCalls != allocation {
			t.Fatalf("allocation %d: device %v error %v closes %d calls %d", allocation, device, err, session.closes, session.bufferCalls)
		}
	}
}
func TestMetalSolverClosesAfterDispatchAndCommitFailures(t *testing.T) {
	for _, commit := range []bool{false, true} {
		session := openFailureSession(t)
		device, err := newDevice(failureSpec(), session)
		if err != nil {
			t.Fatal(err)
		}
		session.failCommit = commit
		session.failDispatch = !commit
		if _, err := device.At(1); err == nil || !strings.Contains(err.Error(), "metal:") {
			t.Fatalf("submission: %v", err)
		}
		if session.closes != 1 {
			t.Fatalf("session closed %d times", session.closes)
		}
		if _, err := device.At(1); err == nil || !strings.Contains(err.Error(), "closed Metal solver") {
			t.Fatalf("failed solver reused: %v", err)
		}
		device.Close()
		if session.closes != 1 {
			t.Fatal("repeat close re-released session")
		}
	}
}
func TestMetalSolverReportsAStencilCompilationFailure(t *testing.T) {
	before := stencil
	t.Cleanup(func() { stencil = before })
	stencil = "this is not Metal source"
	device, err := NewMetal(failureSpec())
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip(err)
	}
	if device != nil || err == nil || !strings.Contains(err.Error(), "metal:") {
		t.Fatalf("bad stencil: %v %v", device, err)
	}
}
