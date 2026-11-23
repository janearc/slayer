package telemetry

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestCPUUtilizationUsesTwoSamplesAndReportsUnknown(t *testing.T) {
	var meter cpuMeter
	now := time.Unix(100, 0)
	read := func(ticks [4]uint32) func() ([4]uint32, error) {
		return func() ([4]uint32, error) { return ticks, nil }
	}
	firstSample := meter.sample(now, read([4]uint32{10, 20, 30, 40}))
	first := values(firstSample)
	if first["azazel_host_cpu_busy_known"] != 0 {
		t.Fatal(first)
	}
	secondSample := meter.sample(now.Add(5*time.Second), read([4]uint32{20, 30, 50, 40}))
	second := values(secondSample)
	if second["azazel_host_cpu_busy_known"] != 1 || math.Abs(second["azazel_host_cpu_busy_percent"]-50) > 0.001 || second["azazel_host_cpu_sample_window_seconds"] != 5 {
		t.Fatal(second)
	}
	failedSample := meter.sample(now.Add(6*time.Second), func() ([4]uint32, error) { return [4]uint32{}, errors.New("unavailable") })
	failed := values(failedSample)
	if failed["azazel_host_cpu_busy_known"] != 0 {
		t.Fatal(failed)
	}
	if _, ok := failed["azazel_host_cpu_busy_percent"]; ok {
		t.Fatal("invented CPU usage", failed)
	}
	unchangedSample := meter.sample(now.Add(7*time.Second), read([4]uint32{20, 30, 50, 40}))
	unchanged := values(unchangedSample)
	if unchanged["azazel_host_cpu_busy_known"] != 0 {
		t.Fatal(unchanged)
	}
}

func TestNativeCPUTicksReadWithoutWork(t *testing.T) {
	ticks, err := nativeCPUTicks()
	if err != nil || ticks == [4]uint32{} {
		t.Fatal(ticks, err)
	}
}

func TestCPUReversedSampleTimeKeepsTheNewerBaseline(t *testing.T) {
	var meter cpuMeter
	now := time.Unix(100, 0)
	read := func(ticks [4]uint32) func() ([4]uint32, error) {
		return func() ([4]uint32, error) { return ticks, nil }
	}
	meter.sample(now, read([4]uint32{10, 10, 10, 10}))
	meter.sample(now.Add(10*time.Second), read([4]uint32{20, 20, 30, 10}))
	reversedSample := meter.sample(now.Add(5*time.Second), read([4]uint32{25, 25, 35, 10}))
	got := values(reversedSample)
	if got["azazel_host_cpu_busy_known"] != 0 {
		t.Fatal(got)
	}
	newerSample := meter.sample(now.Add(20*time.Second), read([4]uint32{30, 30, 50, 10}))
	got = values(newerSample)
	if got["azazel_host_cpu_sample_window_seconds"] != 10 || got["azazel_host_cpu_busy_percent"] != 50 {
		t.Fatal(got)
	}
}

func TestCPUShortIntervalsAccumulateBeforeReporting(t *testing.T) {
	var meter cpuMeter
	now := time.Unix(100, 0)
	read := func(ticks [4]uint32) func() ([4]uint32, error) {
		return func() ([4]uint32, error) { return ticks, nil }
	}
	meter.sample(now, read([4]uint32{10, 10, 10, 10}))
	for _, offset := range []time.Duration{100 * time.Millisecond, 500 * time.Millisecond} {
		sampled := meter.sample(now.Add(offset), read([4]uint32{11, 10, 11, 10}))
		got := values(sampled)
		if got["azazel_host_cpu_busy_known"] != 0 {
			t.Fatal(got)
		}
	}
	windowSample := meter.sample(now.Add(2*time.Second), read([4]uint32{20, 20, 30, 10}))
	got := values(windowSample)
	if got["azazel_host_cpu_sample_window_seconds"] != 2 || got["azazel_host_cpu_busy_percent"] != 50 {
		t.Fatal(got)
	}
}
