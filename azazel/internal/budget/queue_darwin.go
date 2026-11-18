package budget

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
)

// HostObservation reports current conditions without granting a reservation.
type StopCause string

const (
	MemoryPressureStop StopCause = "memory-pressure"
	DiskSpaceStop      StopCause = "disk-space"
)

type HostObservation struct {
	Cause    StopCause
	Ready    bool
	Stop     bool
	Reason   string
	Pressure string
}

// ObserveHost applies the shared development policy to a fresh native sample.
func ObserveHost(ctx context.Context, directory string) (HostObservation, error) {
	return observeQueueHost(ctx, directory, nativeAdmissionHost{})
}

// observeQueueHost checks disk first so an unknown load cannot mask a disk stop.
func observeQueueHost(ctx context.Context, directory string, adapter admissionHostAdapter) (HostObservation, error) {
	if err := ctx.Err(); err != nil {
		return HostObservation{}, err
	}
	file, err := os.Open(directory)
	if err != nil {
		return HostObservation{}, err
	}
	defer file.Close()
	policy := hostPolicy{loadLimit: 8, swapLimit: 8 << 30, diskFloor: byteCount(QueueDiskFloorBytes)}
	disk, err := adapter.Statfs(int(file.Fd()))
	if err != nil {
		return HostObservation{}, err
	}
	if disk.Bsize == 0 || disk.Bavail > disk.Bfree || disk.Bfree > disk.Blocks || disk.Bavail > math.MaxUint64/uint64(disk.Bsize) {
		return HostObservation{}, errAdmissionUnsafe
	}
	if disk.Bavail*uint64(disk.Bsize) < uint64(policy.diskFloor) {
		return queueHostDecision(hostDiskLow)
	}
	boot, err := admissionBoot(ctx, adapter)
	if err != nil {
		return HostObservation{}, err
	}
	sample, err := sampleAdmissionHost(ctx, file, admissionBinding{boot: boot, policy: policy, limit: 1}, adapter)
	if errors.Is(err, errAdmissionPressure) {
		return HostObservation{Stop: true, Cause: MemoryPressureStop, Pressure: "elevated", Reason: "system memory pressure is not normal"}, nil
	}
	if err != nil {
		return HostObservation{}, err
	}
	return queueHostDecision(policy.assess(sample.sample))
}

// queueHostDecision follows pressure sampling; load and accumulated swap are advisory.
func queueHostDecision(decision hostDecision) (HostObservation, error) {
	switch decision {
	case hostReady, hostSwapHigh, hostLoadHigh:
		return HostObservation{Ready: true, Pressure: "normal"}, nil
	case hostDiskLow:
		return HostObservation{Stop: true, Cause: DiskSpaceStop, Reason: "free disk below 30 GiB"}, nil
	default:
		return HostObservation{}, fmt.Errorf("unknown host decision")
	}
}
