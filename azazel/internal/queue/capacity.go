package queue

import (
	"context"
	"github.com/janearc/slayer/azazel/internal/budget"
)

// CapacitySystem supplies native observations without starting any workload.
type CapacitySystem interface {
	Host(context.Context, string) (budget.HostObservation, error)
	GPU() (int, bool)
}

// readCapacity reads host observations for directory, then the GPU counter.
// An unknown GPU reading holds GPU admission and sets a reason when none is
// set.
func readCapacity(ctx context.Context, directory string, system CapacitySystem) (Capacity, error) {
	host, err := system.Host(ctx, directory)
	if err != nil {
		return Capacity{}, err
	}
	sample := Capacity{Ready: host.Ready, Stop: host.Stop, Reason: host.Reason, Pressure: host.Pressure, Cause: host.Cause}
	percent, known := system.GPU()
	if known && percent >= 0 && percent <= 100 {
		sample.GPU, sample.GPUKnown = percent, true
	} else if sample.Reason == "" {
		sample.Reason = "GPU utilization unavailable; GPU admission held"
	}
	return sample, nil
}
