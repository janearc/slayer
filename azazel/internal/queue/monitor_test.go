package queue

import (
	"context"
	"errors"
	"math"
	"os"
	"syscall"
	"testing"

	"github.com/janearc/slayer/azazel/internal/budget"
	"go.uber.org/mock/gomock"
)

// TestGroupMetricsRefuseUnknownsAndOverflow keeps accounting conservative.
func TestGroupMetricsRefuseUnknownsAndOverflow(t *testing.T) {
	broken := errors.New("native metric failed")
	for _, kind := range []string{"host", "group", "memory", "gone", "cancelled", "overflow", "ready", "host-only"} {
		t.Run(kind, func(t *testing.T) {
			system := NewMockmetrics(gomock.NewController(t))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			hostError := error(nil)
			if kind == "host" {
				hostError = broken
			}
			system.EXPECT().Host(ctx, "/output").Return(budget.HostObservation{Ready: true}, hostError)
			group := 12
			if kind == "host-only" {
				group = 0
			}
			if kind != "host" && group != 0 {
				groupError := error(nil)
				if kind == "group" {
					groupError = broken
				}
				system.EXPECT().Group(group).Return([]int{13, 14}, groupError)
				if kind == "cancelled" {
					cancel()
				}
				if kind != "group" && kind != "cancelled" {
					first := uint64(10)
					failure := error(nil)
					if kind == "memory" {
						failure = broken
					}
					if kind == "gone" {
						failure = syscall.ESRCH
					}
					if kind == "overflow" {
						first = math.MaxUint64
					}
					system.EXPECT().Resident(13).Return(first, failure)
					if kind != "memory" {
						system.EXPECT().Resident(14).Return(uint64(20), nil)
					}
				}
			}
			got, err := readVitals(ctx, "/output", group, system)
			fail := kind == "host" || kind == "group" || kind == "memory" || kind == "cancelled" || kind == "overflow"
			if (err != nil) != fail {
				t.Fatal(kind, got, err)
			}
			if kind == "ready" && got.Resident != 30 || kind == "gone" && got.Resident != 20 {
				t.Fatal(got)
			}
		})
	}
}

// TestNativeMetricResultBounds rejects truncated enumeration and native errors.
func TestNativeMetricResultBounds(t *testing.T) {
	for _, count := range []int{-1, 0, 1, 2} {
		pids, err := checkedGroup([]int{10, 11}, count, syscall.ENOMEM)
		if (err != nil) != (count < 0 || count >= 2) {
			t.Fatal(pids, err)
		}
	}
	if _, err := residentResult(-1, 123, syscall.EPERM); !errors.Is(err, syscall.EPERM) {
		t.Fatal(err)
	}
	if got, err := residentResult(0, 123, syscall.EPERM); err != nil || got != 123 {
		t.Fatal(got, err)
	}
}

// TestNativeMetricsReadTheCurrentProcess checks the actual SDK ABI.
func TestNativeMetricsReadTheCurrentProcess(t *testing.T) {
	native := nativeMetrics{}
	resident, err := native.Resident(os.Getpid())
	if err != nil || resident == 0 {
		t.Fatal(resident, err)
	}
	group, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	members, err := native.Group(group)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pid := range members {
		if pid == os.Getpid() {
			found = true
		}
	}
	if !found {
		t.Fatal("native group omitted its own process", members)
	}
	if _, err = (nativeMonitor{}).Read(t.Context(), "/nonexistent-queue-volume", group); err == nil {
		t.Fatal("accepted absent filesystem")
	}
}
