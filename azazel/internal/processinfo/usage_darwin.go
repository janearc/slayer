package processinfo

/*
#include <libproc.h>
#include <sys/resource.h>
*/
import "C"

import (
	"errors"
	"os"
	"unsafe"
)

// nativeUsageSuccess is the successful return defined by libproc.h.
const nativeUsageSuccess = 0

// nativePIDOrigin is the integer boundary below every valid positive PID.
const nativePIDOrigin = 0

// ReadUsage supplies one native PID/start/exit observation or its refusal.
func (darwinUsageReader) ReadUsage(query usageQuery) usageObservation {
	nativePID := C.int(query.pid)
	if query.pid <= nativePIDOrigin || int(nativePID) != query.pid {
		return usageObservation{state: usageFailed, err: os.ErrInvalid}
	}
	var native C.struct_rusage_info_v0
	code, cause := C.proc_pid_rusage(nativePID, C.RUSAGE_INFO_V0,
		(*C.rusage_info_t)(unsafe.Pointer(&native)))
	if code != nativeUsageSuccess {
		return usageObservation{state: usageFailed,
			err: errors.Join(errUsageRead, cause)}
	}
	return usageObservation{state: usageObserved, pid: query.pid,
		born:   uint64(native.ri_proc_start_abstime),
		exited: uint64(native.ri_proc_exit_abstime)}
}
