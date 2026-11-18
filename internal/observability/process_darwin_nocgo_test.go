//go:build darwin && !cgo

package observability

import "testing"

func TestNoCGOProcessCPUAndOptionalMemory(t *testing.T) {
	_, rssOK, cpu, cpuOK := processUsage()
	if rssOK || !cpuOK || cpu < 0 {
		t.Fatalf("unregistered process reading: RSS available %v, CPU %v (%v)", rssOK, cpu, cpuOK)
	}
	stop := RegisterProcessMemorySampler(func() ProcessMemory {
		return ProcessMemory{ResidentBytes: 12, ResidentAvailable: true,
			FootprintBytes: 24, FootprintAvailable: true, PeakFootprintBytes: 40, PeakFootprintAvailable: true}
	})
	defer stop()
	rss, rssOK, _, _ := processUsage()
	current, currentOK, peak, peakOK := processFootprint()
	if !rssOK || rss != 12 || !currentOK || current != 24 || !peakOK || peak != 40 {
		t.Fatalf("registered memory: %d %v %d %v %d %v", rss, rssOK, current, currentOK, peak, peakOK)
	}
}
