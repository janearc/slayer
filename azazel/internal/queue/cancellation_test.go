package queue

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// TestWorkerStopDuringHostObservationRetainsUnstartedWork covers the hold race.
func TestWorkerStopDuringHostObservationRetainsUnstartedWork(t *testing.T) {
	store, job := executableFixture(t, "exit 99")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	monitor := unreadHost(t)
	monitor.EXPECT().Read(ctx, job.Request.Directory, 0).DoAndReturn(
		func(context.Context, string, int) (Vitals, error) {
			cancel()
			return Vitals{}, context.Canceled
		})
	result, ready := store.prepare(ctx, job, monitor)
	if ready || result.State == Cancelled || result.Uncertain || !result.Deferred {
		t.Fatal(result, ready)
	}
}

// TestUnknownHostWithoutCancellationRemainsHeld rejects the opposite shortcut.
func TestUnknownHostWithoutCancellationRemainsHeld(t *testing.T) {
	store, job := executableFixture(t, "exit 99")
	monitor := unreadHost(t)
	monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, 0).
		Return(Vitals{}, errors.New("host sample unavailable"))
	result, ready := store.prepare(t.Context(), job, monitor)
	if ready || !result.Deferred || result.State == Cancelled || result.Stop {
		t.Fatal(result, ready)
	}
}
