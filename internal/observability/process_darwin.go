//go:build darwin && cgo

package observability

/*
#include <libproc.h>
#include <sys/resource.h>
#include <unistd.h>

static int observation_resident(unsigned long long *bytes) {
	struct rusage_info_v0 info;
	if (proc_pid_rusage(getpid(), RUSAGE_INFO_V0, (rusage_info_t *)&info) != 0) return -1;
	*bytes = info.ri_resident_size;
	return 0;
}

static int observation_cpu(double *seconds) {
	struct rusage usage;
	if (getrusage(RUSAGE_SELF, &usage) != 0) return -1;
	*seconds = (double)usage.ru_utime.tv_sec + (double)usage.ru_stime.tv_sec +
		((double)usage.ru_utime.tv_usec + (double)usage.ru_stime.tv_usec) / 1000000.0;
	return 0;
}

// V4 adds the kernel's lifetime maximum. An older kernel can still supply
// the current footprint through V0, but cannot supply that maximum.
static int observation_footprint(unsigned long long *current,
		unsigned long long *peak) {
	struct rusage_info_v4 info = {0};
	if (proc_pid_rusage(getpid(), RUSAGE_INFO_V4, (rusage_info_t *)&info) == 0) {
		*current = info.ri_phys_footprint;
		*peak = info.ri_lifetime_max_phys_footprint;
		return 3;
	}
	struct rusage_info_v0 old = {0};
	if (proc_pid_rusage(getpid(), RUSAGE_INFO_V0, (rusage_info_t *)&old) == 0) {
		*current = old.ri_phys_footprint;
		return 1;
	}
	return 0;
}
*/
import "C"

// processUsage reports this process, not its children or GPU allocations.
// CPU is cumulative user plus system time, converted from getrusage's
// timeval units. RSS and CPU are sampled independently and may fail alone.
func processUsage() (rss uint64, rssOK bool, cpuSeconds float64, cpuOK bool) {
	var resident C.ulonglong
	var cpu C.double
	rssOK = C.observation_resident(&resident) == 0
	cpuOK = C.observation_cpu(&cpu) == 0
	return uint64(resident), rssOK, float64(cpu), cpuOK
}

// processFootprint reads this process's current and lifetime maximum physical
// footprint. It does not add child processes and is distinct from RSS.
func processFootprint() (current uint64, currentOK bool, peak uint64, peakOK bool) {
	var currentNative, peakNative C.ulonglong
	fields := int(C.observation_footprint(&currentNative, &peakNative))
	return footprintValues(uint64(currentNative), uint64(peakNative), fields)
}

// footprintValues reads the native footprint fields, and marks each one
// known or not.
func footprintValues(current, peak uint64, fields int) (uint64, bool, uint64, bool) {
	currentOK := fields&1 != 0 && current > 0
	peakOK := currentOK && fields&2 != 0 && peak >= current
	if !currentOK {
		current = 0
	}
	if !peakOK {
		peak = 0
	}
	return current, currentOK, peak, peakOK
}
