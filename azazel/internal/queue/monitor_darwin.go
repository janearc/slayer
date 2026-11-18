package queue

/*
#include <libproc.h>
#include <sys/resource.h>
*/
import "C"

import (
	"context"
	"unsafe"

	"github.com/janearc/slayer/azazel/internal/budget"
)

type nativeMonitor struct{}
type nativeMetrics struct{}

// Read takes a fresh host sample and sums resident bytes in the owned group.
func (nativeMonitor) Read(ctx context.Context, directory string, group int) (Vitals, error) {
	return readVitals(ctx, directory, group, nativeMetrics{})
}

// Host reuses Azazel's native admission sampler.
func (nativeMetrics) Host(ctx context.Context, directory string) (budget.HostObservation, error) {
	return budget.ObserveHost(ctx, directory)
}

// Group enumerates a bounded foreground group using the system SDK.
func (nativeMetrics) Group(group int) ([]int, error) {
	pids := make([]C.int, 4096)
	count, cause := C.proc_listpgrppids(C.pid_t(group), unsafe.Pointer(&pids[0]), C.int(len(pids))*C.int(C.sizeof_int))
	buffer := make([]int, len(pids))
	for i, pid := range pids {
		buffer[i] = int(pid)
	}
	return checkedGroup(buffer, int(count), cause)
}

// Resident reads the documented physical residency counter for one process.
func (nativeMetrics) Resident(pid int) (uint64, error) {
	var info C.struct_rusage_info_v0
	code, cause := C.proc_pid_rusage(C.int(pid), C.RUSAGE_INFO_V0, (*C.rusage_info_t)(unsafe.Pointer(&info)))
	return residentResult(int(code), uint64(info.ri_resident_size), cause)
}
