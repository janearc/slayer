//go:build darwin && cgo

package observability

import (
	"math"
	"syscall"
	"testing"
)

func TestCurrentProcessUsageReportsResidentAndCumulativeCPU(t *testing.T) {
	rss, rssOK, cpu, cpuOK := processUsage()
	if !rssOK || rss == 0 || !cpuOK || cpu < 0 {
		t.Fatalf("invalid current-process sample: RSS %d (%v), CPU %v (%v)", rss, rssOK, cpu, cpuOK)
	}
	var native syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &native); err != nil {
		t.Fatal(err)
	}
	want := float64(native.Utime.Sec+native.Stime.Sec) +
		float64(native.Utime.Usec+native.Stime.Usec)/1e6
	if math.Abs(want-cpu) > 5 {
		t.Fatalf("CPU %v seconds differs from getrusage %v seconds", cpu, want)
	}
	_, _, later, laterOK := processUsage()
	if !laterOK || later < cpu {
		t.Fatalf("process CPU went backwards: %v to %v (%v)", cpu, later, laterOK)
	}
}

func TestPhysicalFootprintCurrentAndLifetimePeak(t *testing.T) {
	current, currentOK, peak, peakOK := processFootprint()
	if !currentOK || current == 0 {
		t.Fatalf("current physical footprint unavailable: %d (%v)", current, currentOK)
	}
	t.Logf("current footprint %d bytes; lifetime peak %d bytes (available %v)", current, peak, peakOK)
	if peakOK && peak < current {
		t.Fatalf("lifetime footprint peak %d is below current %d", peak, current)
	}
	_, _, later, laterOK := processFootprint()
	if peakOK && laterOK && later < peak {
		t.Fatalf("lifetime footprint peak went backwards: %d to %d", peak, later)
	}
}

func TestFootprintObservationNeverInventsAnUnavailablePeak(t *testing.T) {
	for _, check := range []struct {
		name                  string
		current, peak         uint64
		fields                int
		wantCurrent, wantPeak bool
	}{
		{"native failure", 0, 0, 0, false, false},
		{"older version", 100, 0, 1, true, false},
		{"unpopulated peak", 100, 0, 3, true, false},
		{"peak below current", 100, 99, 3, true, false},
		{"both fields", 100, 120, 3, true, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			_, currentOK, _, peakOK := footprintValues(check.current, check.peak, check.fields)
			if currentOK != check.wantCurrent || peakOK != check.wantPeak {
				t.Fatalf("availability = %v %v, want %v %v", currentOK, peakOK, check.wantCurrent, check.wantPeak)
			}
		})
	}
}
