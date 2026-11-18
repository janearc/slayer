package queue

import (
	"errors"
	"fmt"
	"math"
	"syscall"
)

// MetricsGroupResident returns the sum of measured resident bytes across a
// process group. It is independent of host admission and is not a measure of
// unique physical memory; shared pages can be counted more than once.
func MetricsGroupResident(group int) (uint64, error) {
	return metricsGroupResident(group, nativeMetrics{})
}

type residentSource interface {
	Group(int) ([]int, error)
	Resident(int) (uint64, error)
}

// metricsGroupResident reads the resident bytes of the processes in group. It
// refuses an invalid or unreadable group.
func metricsGroupResident(group int, source residentSource) (uint64, error) {
	if group <= 0 {
		return 0, fmt.Errorf("invalid process group")
	}
	pids, err := source.Group(group)
	if err != nil {
		return 0, err
	}
	if len(pids) == 0 {
		return 0, fmt.Errorf("process group has exited")
	}
	var total uint64
	for _, pid := range pids {
		bytes, err := source.Resident(pid)
		if errors.Is(err, syscall.ESRCH) {
			return 0, fmt.Errorf("process exited during observation: %w", err)
		}
		if err != nil {
			return 0, err
		}
		if bytes > math.MaxUint64-total {
			return 0, fmt.Errorf("process memory overflow")
		}
		total += bytes
	}
	return total, nil
}
