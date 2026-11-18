package queue

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"github.com/janearc/slayer/azazel/internal/budget"
)

// metrics separates native observations from queue decisions.
type metrics interface {
	Host(context.Context, string) (budget.HostObservation, error)
	Group(int) ([]int, error)
	Resident(int) (uint64, error)
}

// readVitals holds unknown measurements and counts each observed group member.
func readVitals(ctx context.Context, directory string, group int, system metrics) (Vitals, error) {
	host, err := system.Host(ctx, directory)
	if err != nil {
		return Vitals{}, err
	}
	out := Vitals{Ready: host.Ready, Stop: host.Stop, Reason: host.Reason, Pressure: host.Pressure, Cause: host.Cause}
	if group == 0 || host.Stop {
		return out, nil
	}
	pids, err := system.Group(group)
	if err != nil {
		return out, err
	}
	for _, pid := range pids {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		bytes, err := system.Resident(pid)
		if err != nil {
			if errors.Is(err, syscall.ESRCH) {
				continue
			}
			return out, fmt.Errorf("process memory unavailable: %w", err)
		}
		if out.Resident+bytes < out.Resident {
			return out, fmt.Errorf("process memory overflow")
		}
		out.Resident += bytes
	}
	return out, nil
}

// checkedGroup refuses a full or malformed enumeration instead of undercounting.
func checkedGroup(buffer []int, count int, cause error) ([]int, error) {
	if count < 0 || count >= len(buffer) {
		return nil, fmt.Errorf("process group could not be bounded: %v", cause)
	}
	return buffer[:count], nil
}

// residentResult preserves the native failure cause and never invents usage.
func residentResult(code int, bytes uint64, cause error) (uint64, error) {
	if code != 0 {
		return 0, fmt.Errorf("process usage failed: %w", cause)
	}
	return bytes, nil
}
