package hillside

import (
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/internal/observability"
	"github.com/janearc/slayer/render/frame"
)

func TestGPUProfileObservationsKeepWallAndDeviceTimesSeparate(t *testing.T) {
	observed := map[string]float64{}
	added := map[string]float64{}
	gauges := map[string]float64{}
	observe := func(name string, value float64) { observed[name] += value }
	add := func(name string, value float64) { added[name] += value }
	set := func(name string, value float64) { gauges[name] = value }
	profile := gpu.Profile{
		Device:           "fixture",
		AccelerationWall: 5 * time.Second,
		AccelerationGPU:  2 * time.Second,
		DispatchWall:     8 * time.Second,
		TraceGPU:         3 * time.Second,
		InputBytes:       40,
		PeakSampleBytes:  20,
		Retained:         gpu.FrameInputs{SentBytes: 10},
	}
	recordGPUProfile(profile, observe, add, set)
	for name, want := range map[string]float64{
		"slayer_gpu_acceleration_wall_seconds":   5,
		"slayer_gpu_acceleration_device_seconds": 2,
		"slayer_gpu_dispatch_wall_seconds":       8,
		"slayer_gpu_trace_device_seconds":        3,
	} {
		if observed[name] != want {
			t.Errorf("%s = %v, want %v", name, observed[name], want)
		}
	}
	if added["slayer_gpu_logical_input_bytes_total"] != 40 || added["slayer_gpu_retained_sent_bytes_total"] != 10 {
		t.Fatalf("input and retained transfer bytes: %v", added)
	}
	if gauges["slayer_gpu_last_glimpse_peak_sample_readback_bytes"] != 20 {
		t.Fatalf("sample readback is a gauge, not whole-process memory: %v", gauges)
	}
	before := len(observed)
	recordGPUProfile(gpu.Profile{}, observe, add, set)
	if len(observed) != before || added["slayer_gpu_logical_input_bytes_total"] != 40 {
		t.Fatal("a CPU glimpse must not be recorded as a GPU glimpse")
	}
}

func TestMetalMemorySnapshotReportsAvailabilityAndNotPeak(t *testing.T) {
	gauges := map[string]float64{}
	set := func(name string, value float64) { gauges[name] = value }
	recordMetalMemory(func() (uint64, uint64, bool) { return 42, 100, true }, set)
	if gauges["slayer_gpu_after_glimpse_device_memory_available"] != 1 ||
		gauges["slayer_gpu_after_glimpse_device_allocated_bytes"] != 42 ||
		gauges["slayer_gpu_device_recommended_working_set_bytes"] != 100 {
		t.Fatal(gauges)
	}
	recordMetalMemory(func() (uint64, uint64, bool) { return 0, 0, false }, set)
	if gauges["slayer_gpu_after_glimpse_device_memory_available"] != 0 ||
		gauges["slayer_gpu_after_glimpse_device_allocated_bytes"] != 42 {
		t.Fatal("failure must mark the last successful values unavailable:", gauges)
	}
}

func TestCPUGlimpseDoesNotOpenMetalForMetrics(t *testing.T) {
	reads := 0
	read := func() (uint64, uint64, bool) { reads++; return 42, 100, true }
	set := func(string, float64) {}
	recordGPUMemory(gpu.Profile{}, read, set)
	if reads != 0 {
		t.Fatal("CPU glimpse sampled a Metal device")
	}
	recordGPUMemory(gpu.Profile{Device: "fixture"}, read, set)
	if reads != 1 {
		t.Fatalf("GPU glimpse made %d snapshots, want one", reads)
	}
}

type measuredFixture struct{ calls int }

func (source *measuredFixture) light(_ time.Duration, _ asked) (drawnLight, error) {
	source.calls++
	return drawnLight{drawn: frame.New(1, 1)}, nil
}

func scrapeValue(t *testing.T, name string) float64 {
	t.Helper()
	recorder := httptest.NewRecorder()
	observability.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	for _, line := range strings.Split(recorder.Body.String(), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == name {
			value, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
	}
	return 0
}

func TestEveryShutterGlimpseContributesToTheScrape(t *testing.T) {
	t.Setenv("SLAYER_METRICS_ADDR", "127.0.0.1:0")
	closeMetrics, err := observability.Start("slayer-test")
	if err != nil {
		t.Fatal(err)
	}
	defer closeMetrics()
	name := "slayer_glimpses_total"
	before := scrapeValue(t, name)
	source := &measuredFixture{}
	session := Session{source: source, offsets: []time.Duration{0, time.Second}}
	if _, err := session.glimpsed(context.Background(), 0, asked{}); err != nil {
		t.Fatal(err)
	}
	if got := scrapeValue(t, name) - before; got != 2 {
		t.Fatalf("two glimpses reported %v observations, want 2", got)
	}
}
