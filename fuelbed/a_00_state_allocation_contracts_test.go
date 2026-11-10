//go:build darwin

package fuelbed

import (
	"github.com/janearc/slayer/metal"
	"testing"
)

// Track actual allocations through the real Metal session. A retained state
// uses exactly two packed float words per cell: excess heat and fuel. This
// bounds memory for large batches and prevents a short output allocation.
type stateAllocationSession struct {
	*metal.Session
	counts              []int
	dispatches, commits int
	failDispatch        bool
}

func (s *stateAllocationSession) Buffer(count int) (metal.Buffer, error) {
	s.counts = append(s.counts, count)
	return s.Session.Buffer(count)
}

func (s *stateAllocationSession) Dispatch(name string, w, h int, params []byte, buffers ...metal.Buffer) error {
	s.dispatches++
	if s.failDispatch {
		s.Session.Close()
	}
	return s.Session.Dispatch(name, w, h, params, buffers...)
}

func (s *stateAllocationSession) Commit() error {
	s.commits++
	return s.Session.Commit()
}

func TestRetainedStateAllocationsMatchPackedCells(t *testing.T) {
	real := openFailureSession(t)
	session := &stateAllocationSession{Session: real.Session}
	s := quietBoundarySpec()
	s.Width, s.Depth = 3, 5
	s.Initial = make([]Cell, 15)
	device, err := newDevice(s, session)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	if len(session.counts) != 2 || session.counts[0] != 30 || session.counts[1] != 30 {
		t.Fatalf("packed heat/fuel states requested %v words; want [30 30]", session.counts)
	}
}
