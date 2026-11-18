package budget

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

// queueDisk is a healthy explicitly sized filesystem observation.
func queueDisk() (unix.Statfs_t, error) {
	return unix.Statfs_t{Bsize: 4096, Blocks: 16 << 20, Bfree: 8 << 20, Bavail: 8 << 20}, nil
}

// TestQueueHostStopsOnDiskBeforeUnknownLoad keeps the disk latch authoritative.
func TestQueueHostStopsOnDiskBeforeUnknownLoad(t *testing.T) {
	host := &fakeHost{sysctl: func(string, int) ([]byte, error) { t.Fatal("queried host beyond known low disk"); return nil, nil }}
	got, err := observeQueueHost(t.Context(), t.TempDir(), host)
	if err != nil || !got.Stop || got.Ready {
		t.Fatal(got, err)
	}
}

// TestQueueHostObservations exercises failures at the native adapter boundary.
func TestQueueHostObservations(t *testing.T) {
	broken := errors.New("native read failed")
	for _, kind := range []string{"ready", "disk-read", "disk-invalid", "boot", "sample", "cancelled", "missing"} {
		t.Run(kind, func(t *testing.T) {
			host := &fakeHost{statfs: queueDisk}
			path := t.TempDir()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch kind {
			case "disk-read":
				host.statfs = func() (unix.Statfs_t, error) { return unix.Statfs_t{}, broken }
			case "disk-invalid":
				host.statfs = func() (unix.Statfs_t, error) { return unix.Statfs_t{}, nil }
			case "boot":
				host.sysctl = func(string, int) ([]byte, error) { return nil, broken }
			case "sample":
				host.sysctl = func(key string, _ int) ([]byte, error) {
					if key == "vm.loadavg" {
						return nil, broken
					}
					return nil, nil
				}
			case "cancelled":
				cancel()
			case "missing":
				path += "/missing"
			}
			got, err := observeQueueHost(ctx, path, host)
			if kind == "ready" {
				if err != nil || !got.Ready {
					t.Fatal(got, err)
				}
			} else if err == nil || got.Ready {
				t.Fatal(got, err)
			}
		})
	}
}

// TestQueueHostDecisionNeverTreatsUnknownAsReady checks every result class.
func TestQueueHostDecisionNeverTreatsUnknownAsReady(t *testing.T) {
	for _, decision := range []hostDecision{hostReady, hostDiskLow, hostLoadHigh, hostSwapHigh, hostUnknown} {
		got, err := queueHostDecision(decision)
		if got.Ready != (decision == hostReady || decision == hostSwapHigh || decision == hostLoadHigh) || got.Stop != (decision == hostDiskLow) || (err != nil) != (decision == hostUnknown) {
			t.Fatal(decision, got, err)
		}
	}
}

// TestQueueNativeHostObservation exercises the public system-sampling facade.
func TestQueueNativeHostObservation(t *testing.T) {
	if _, err := ObserveHost(t.Context(), t.TempDir()); err != nil && !errors.Is(err, errAdmissionPressure) {
		t.Fatal(err)
	}
}
