//go:build darwin && cgo

package footprint

/*
#include <libproc.h>
#include <sys/resource.h>
#include <stdint.h>

static int footprint_identity(int pid, int *group, uint64_t *seconds,
    uint64_t *microseconds) {
    struct proc_bsdinfo info = {0};
    int n = proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof(info));
    if (n != sizeof(info)) return -1;
    *group = info.pbi_pgid;
    *seconds = info.pbi_start_tvsec;
    *microseconds = info.pbi_start_tvusec;
    return 0;
}

static int footprint_read(int pid, uint64_t *current, uint64_t *peak) {
    struct rusage_info_v4 info = {0};
    if (proc_pid_rusage(pid, RUSAGE_INFO_V4, (rusage_info_t *)&info) == 0) {
        *current = info.ri_phys_footprint;
        *peak = info.ri_lifetime_max_phys_footprint;
        return 2;
    }
    struct rusage_info_v0 old = {0};
    if (proc_pid_rusage(pid, RUSAGE_INFO_V0, (rusage_info_t *)&old) == 0) {
        *current = old.ri_phys_footprint;
        return 1;
    }
    return 0;
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

type native struct{}

// Native returns the local macOS process information source.
func Native() Source { return native{} }

// BootID reads the macOS boot-session UUID and normalizes it for comparison.
func BootID() (string, error) {
	raw, err := unix.SysctlRaw("kern.bootsessionuuid")
	if err != nil || len(raw) != 37 || raw[36] != 0 {
		return "", fmt.Errorf("boot identity unavailable")
	}
	value := strings.ToLower(string(raw[:36]))
	if !ValidBootID(value) {
		return "", fmt.Errorf("boot identity invalid")
	}
	return value, nil
}

// Group lists the process IDs in group. A truncated listing is refused.
func (native) Group(group int) ([]int, error) {
	buffer := make([]C.int, MaximumMembers)
	count := int(C.proc_listpgrppids(C.pid_t(group), unsafe.Pointer(&buffer[0]), C.int(len(buffer))*C.int(C.sizeof_int)))
	if count <= 0 || count >= len(buffer) {
		return nil, fmt.Errorf("process group enumeration failed or was truncated")
	}
	result := make([]int, count)
	for i := range result {
		result[i] = int(buffer[i])
	}
	return result, nil
}

// Identity reads the group and start time of pid from the kernel.
func (native) Identity(pid int) (Identity, error) {
	var group C.int
	var seconds, microseconds C.uint64_t
	if C.footprint_identity(C.int(pid), &group, &seconds, &microseconds) != 0 {
		return Identity{}, fmt.Errorf("process identity unavailable")
	}
	return Identity{PID: pid, Group: int(group), StartSeconds: uint64(seconds), StartMicroseconds: uint64(microseconds)}, nil
}

// Footprint reads the current and lifetime-peak footprint of pid. PeakKnown
// is false when only the older counter set is available.
func (native) Footprint(pid int) (Reading, error) {
	var current, peak C.uint64_t
	status := int(C.footprint_read(C.int(pid), &current, &peak))
	if status == 0 {
		return Reading{}, fmt.Errorf("process footprint unavailable")
	}
	return Reading{Current: uint64(current), LifetimePeak: uint64(peak), PeakKnown: status == 2}, nil
}
