//go:build darwin

package fuelbed

import "testing"

func TestRepeatedMetalSnapshotDoesNotResubmitCompletedTicks(t *testing.T) {
	real := openFailureSession(t)
	session := &stateAllocationSession{Session: real.Session}
	device, err := newDevice(quietBoundarySpec(), session)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	for _, at := range []float64{.5, .5, .75} {
		if _, err := device.At(at); err != nil {
			t.Fatal(err)
		}
	}
	if session.dispatches != 1 || session.commits != 1 {
		t.Fatalf("same completed tick submitted %d dispatches and %d commits", session.dispatches, session.commits)
	}
	if _, err := device.At(1); err != nil {
		t.Fatal(err)
	}
	if session.dispatches != 2 || session.commits != 2 {
		t.Fatalf("next completed tick submitted %d dispatches and %d commits", session.dispatches, session.commits)
	}
}

func TestMetalDispatchRefusalStopsBeforeCommit(t *testing.T) {
	real := openFailureSession(t)
	session := &stateAllocationSession{Session: real.Session}
	device, err := newDevice(quietBoundarySpec(), session)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	session.failDispatch = true
	_, err = device.At(1)
	if err == nil || err.Error() != "metal: no kernel fuel_step in the session" {
		t.Fatalf("lost original dispatch refusal: %v", err)
	}
	if session.dispatches != 1 || session.commits != 0 {
		t.Fatalf("continued after refusal: %d dispatches, %d commits", session.dispatches, session.commits)
	}
}
