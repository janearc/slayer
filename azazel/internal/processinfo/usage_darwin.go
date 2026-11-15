package processinfo

/*
#include <libproc.h>
#include <sys/resource.h>
*/
import "C"

import (
	"errors"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
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

type darwinOwnerReader struct{ darwinUsageReader }

// ReadBoot reads the boot session uuid from the kernel, lowercased. It
// refuses anything that is not a 36 character uuid ending in a zero byte.
func (darwinOwnerReader) ReadBoot() (string, error) {
	data, err := unix.SysctlRaw("kern.bootsessionuuid")
	if err != nil {
		return "", err
	}
	if len(data) != 37 || data[36] != 0 {
		return "", ErrOwnerUnknown
	}
	boot := strings.ToLower(string(data[:36]))
	if !validBoot(boot) {
		return "", ErrOwnerUnknown
	}
	return boot, nil
}

// ReadRelations reads the parent and process group of pid. It refuses a
// short read or an answer that names a different pid.
func (darwinOwnerReader) ReadRelations(pid int) (processRelations, error) {
	if !validOwnerPID(pid) {
		return processRelations{}, os.ErrInvalid
	}
	var native C.struct_proc_bsdinfo
	size := C.int(C.sizeof_struct_proc_bsdinfo)
	code, cause := C.proc_pidinfo(C.int(pid), C.PROC_PIDTBSDINFO, 0, unsafe.Pointer(&native), size)
	if code != size {
		return processRelations{}, errors.Join(ErrOwnerUnknown, cause)
	}
	if int(native.pbi_pid) != pid {
		return processRelations{}, ErrOwnerUnknown
	}
	return processRelations{pid: pid, parent: int(native.pbi_ppid), group: int(native.pbi_pgid)}, nil
}

// ListChildren lists the child processes of pid, up to limit.
func (darwinOwnerReader) ListChildren(pid, limit int) ([]int, error) {
	return nativeOwnerList(pid, limit, false)
}

// ListGroup lists the members of the process group pid, up to limit.
func (darwinOwnerReader) ListGroup(pid, limit int) ([]int, error) {
	return nativeOwnerList(pid, limit, true)
}

// nativeOwnerList asks the system for the pids that are children of pid, or
// the members of process group pid if group is true. limit is at most
// quiescenceMemberLimit. It refuses an empty result, a full buffer, an
// invalid pid, and a repeated pid.
func nativeOwnerList(pid, limit int, group bool) ([]int, error) {
	if !validOwnerPID(pid) || limit <= 0 || limit > quiescenceMemberLimit {
		return nil, os.ErrInvalid
	}
	buffer := make([]C.int, limit)
	size := C.int(len(buffer)) * C.int(C.sizeof_int)
	var code C.int
	var cause error
	if group {
		code, cause = C.proc_listpgrppids(C.pid_t(pid), unsafe.Pointer(&buffer[0]), size)
	} else {
		code, cause = C.proc_listchildpids(C.pid_t(pid), unsafe.Pointer(&buffer[0]), size)
	}
	if code <= 0 || int(code) >= len(buffer) {
		return nil, errors.Join(ErrOwnerUnknown, cause)
	}
	out := make([]int, int(code))
	seen := make(map[int]bool, len(out))
	for i := range out {
		out[i] = int(buffer[i])
		if !validOwnerPID(out[i]) || seen[out[i]] {
			return nil, ErrOwnerUnknown
		}
		seen[out[i]] = true
	}
	return out, nil
}
