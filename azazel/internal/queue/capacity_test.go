package queue

import (
	"errors"
	"github.com/janearc/slayer/azazel/internal/budget"
	"go.uber.org/mock/gomock"
	"testing"
)

// TestCapacitySeparatesUnknownDeviceFromHostStop verifies every admission input class.
func TestCapacitySeparatesUnknownDeviceFromHostStop(t *testing.T) {
	for _, test := range []struct {
		name      string
		host      budget.HostObservation
		err       error
		percent   int
		known     bool
		wantKnown bool
	}{
		{"idle", budget.HostObservation{Ready: true}, nil, 0, true, true},
		{"busy", budget.HostObservation{Ready: true}, nil, 100, true, true},
		{"unknown", budget.HostObservation{Ready: true}, nil, 0, false, false},
		{"negative", budget.HostObservation{Ready: true}, nil, -1, true, false},
		{"overflow", budget.HostObservation{Ready: true}, nil, 101, true, false},
		{"pressure", budget.HostObservation{Stop: true, Pressure: "elevated", Reason: "system memory pressure is not normal"}, nil, 35, true, true},
		{"disk", budget.HostObservation{Stop: true, Reason: "free disk below 30 GiB"}, nil, 0, false, false},
		{"error", budget.HostObservation{}, errors.New("host failed"), 0, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			system := NewMockCapacitySystem(gomock.NewController(t))
			system.EXPECT().Host(gomock.Any(), "/output").Return(test.host, test.err)
			if test.err == nil {
				system.EXPECT().GPU().Return(test.percent, test.known)
			}
			got, err := readCapacity(t.Context(), "/output", system)
			if !errors.Is(err, test.err) || got.GPUKnown != test.wantKnown || got.Ready != test.host.Ready || got.Stop != test.host.Stop || got.Pressure != test.host.Pressure {
				t.Fatal(got, err)
			}
			if test.host.Reason != "" && got.Reason != test.host.Reason {
				t.Fatal(got)
			}
		})
	}
}

// TestNativeCapacityObservation reads hardware counters without submitting GPU work.
func TestNativeCapacityObservation(t *testing.T) {
	if _, err := (NativeCapacityProbe{}).Read(t.Context(), t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
