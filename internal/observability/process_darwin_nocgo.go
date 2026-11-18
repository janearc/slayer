//go:build darwin && !cgo

package observability

import "syscall"

// getrusage uses Darwin's public libc trampoline even without cgo.
func processUsage() (rss uint64, rssOK bool, cpuSeconds float64, cpuOK bool) {
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) == nil {
		cpuSeconds = float64(usage.Utime.Sec+usage.Stime.Sec) +
			float64(usage.Utime.Usec+usage.Stime.Usec)/1e6
		cpuOK = true
	}
	memory := registeredProcessMemory()
	return memory.ResidentBytes, memory.ResidentAvailable, cpuSeconds, cpuOK
}

// processFootprint reads the footprint from the registered memory source.
func processFootprint() (current uint64, currentOK bool, peak uint64, peakOK bool) {
	memory := registeredProcessMemory()
	return memory.FootprintBytes, memory.FootprintAvailable,
		memory.PeakFootprintBytes, memory.PeakFootprintAvailable
}
