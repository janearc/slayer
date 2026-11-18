//go:build !darwin

package observability

import "testing"

func TestUnavailableProcessUsageDoesNotClaimMeasurements(t *testing.T) {
	rss, rssOK, cpu, cpuOK := processUsage()
	if rss != 0 || rssOK || cpu != 0 || cpuOK {
		t.Fatalf("unsupported process usage reported a value: %d %v %v %v", rss, rssOK, cpu, cpuOK)
	}
}

func TestUnavailableFootprintDoesNotClaimMeasurements(t *testing.T) {
	current, currentOK, peak, peakOK := processFootprint()
	if current != 0 || currentOK || peak != 0 || peakOK {
		t.Fatalf("unsupported footprint reported a value: %d %v %d %v", current, currentOK, peak, peakOK)
	}
}
