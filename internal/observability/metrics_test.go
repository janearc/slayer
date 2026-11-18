package observability

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectorCanBeRemoved(t *testing.T) {
	remove := RegisterCollector(func() []Sample { return []Sample{{Name: "test_removable_gauge", Value: 1}} })
	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(response.Body.String(), "test_removable_gauge 1") {
		t.Fatal("collector missing")
	}
	remove()
	response = httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if strings.Contains(response.Body.String(), "test_removable_gauge") {
		t.Fatal("removed collector still exported")
	}
}

func TestProcessFootprintScrapeUsesAvailability(t *testing.T) {
	_, currentOK, _, peakOK := processFootprint()
	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	text := response.Body.String()
	for _, sample := range []struct {
		name      string
		available bool
	}{
		{"process_physical_footprint", currentOK},
		{"process_peak_physical_footprint", peakOK},
	} {
		flag := "0"
		if sample.available {
			flag = "1"
		}
		if !strings.Contains(text, sample.name+"_available "+flag+"\n") {
			t.Fatalf("missing availability for %s: %s", sample.name, text)
		}
		if strings.Contains(text, sample.name+"_bytes ") != sample.available {
			t.Fatalf("incorrect footprint value for %s: %s", sample.name, text)
		}
	}
}

func TestMetricNamesHaveOneType(t *testing.T) {
	Set("test_type_collision", 7)
	Add("test_type_collision", 5)
	Observe("test_type_collision", 2)
	Observe("test_long_render_seconds", 1200)
	Set("test_long_render_seconds_count", 999)
	remove := RegisterCollector(func() []Sample {
		return []Sample{{Name: "test_type_collision", Value: 42}, {Name: "test_long_render_seconds_count", Value: 999}}
	})
	defer remove()
	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	text := response.Body.String()
	if strings.Count(text, "# TYPE test_type_collision ") != 1 || !strings.Contains(text, "test_type_collision 7\n") {
		t.Fatalf("conflicting types: %s", text)
	}
	if !strings.Contains(text, "test_long_render_seconds_bucket{le=\"1200\"} 1\n") || strings.Count(text, "test_long_render_seconds_count ") != 1 {
		t.Fatalf("histogram collision: %s", text)
	}
}

func TestProfilerListenerAndFailedStartCleanup(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := probe.Addr().String()
	probe.Close()
	t.Setenv("SLAYER_METRICS_ADDR", address)
	t.Setenv("SLAYER_PPROF_ADDR", "0.0.0.0:1")
	if _, err := Start("test"); err == nil {
		t.Fatal("accepted non-loopback profiler")
	}
	probe, err = net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("failed start leaked metrics listener: %v", err)
	}
	profileAddress := probe.Addr().String()
	probe.Close()
	t.Setenv("SLAYER_METRICS_ADDR", "")
	t.Setenv("SLAYER_PPROF_ADDR", profileAddress)
	stop, err := Start("test")
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Get("http://" + profileAddress + "/debug/pprof/")
	if err != nil {
		stop()
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		stop()
		t.Fatalf("profiler status %d", response.StatusCode)
	}
	if _, err := Start("test"); err == nil {
		stop()
		t.Fatal("accepted overlapping start")
	}
	stop()
	stop()
	probe, err = net.Listen("tcp", profileAddress)
	if err != nil {
		t.Fatalf("stop leaked profiler listener: %v", err)
	}
	probe.Close()
}

func TestListenersServeAndStop(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_METRICS_ADDR", address)
	t.Setenv("SLAYER_PPROF_ADDR", "")
	stop, err := Start("test")
	if err != nil {
		t.Fatal(err)
	}
	if !Enabled() {
		t.Fatal("listener not marked enabled")
	}
	response, err := http.Get("http://" + address + "/metrics")
	if err != nil {
		stop()
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || !strings.Contains(string(body), "test_observability_up 1") {
		stop()
		t.Fatalf("bad scrape: %s (%v)", body, err)
	}
	stop()
	if Enabled() {
		t.Fatal("listener still marked enabled")
	}
	if response, err = http.Get("http://" + address + "/metrics"); err == nil {
		response.Body.Close()
		t.Fatal("listener still serving")
	}
}

func TestMetricsExposeValuesAndHistogram(t *testing.T) {
	Set("test_queue_depth", 3)
	Add("test_completed_total", 2)
	Observe("test_frame_seconds", 0.02)
	Observe("test_frame_seconds", 0.2)
	RegisterCollector(func() []Sample { return []Sample{{Name: "test_host_available", Value: 1}} })
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	for _, want := range []string{
		"test_queue_depth 3\n", "test_completed_total 2\n",
		"test_frame_seconds_bucket{le=\"0.05\"} 1\n",
		"test_frame_seconds_bucket{le=\"+Inf\"} 2\n",
		"test_frame_seconds_count 2\n", "test_host_available 1\n",
		"go_goroutines ", "go_gomaxprocs ", "go_memstats_heap_alloc_bytes ",
	} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing %q in %s", want, w.Body.String())
		}
	}
	w = httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("pprof exposed on metrics endpoint: %d", w.Code)
	}
}

func TestInvalidNamesAndValuesAreNotExported(t *testing.T) {
	Set("bad name", 1)
	Add("bad\nname", 1)
	RegisterCollector(func() []Sample { return []Sample{{Name: "bad name", Value: 2}} })
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if strings.Contains(w.Body.String(), "bad name") || strings.Contains(w.Body.String(), "bad\nname") {
		t.Fatal("invalid metric reached output")
	}
}

func TestStartIsOptInAndRejectsRemoteProfiling(t *testing.T) {
	t.Setenv("SLAYER_METRICS_ADDR", "")
	t.Setenv("SLAYER_PPROF_ADDR", "")
	stop, err := Start("test")
	if err != nil {
		t.Fatal(err)
	}
	stop()
	t.Setenv("SLAYER_PPROF_ADDR", "0.0.0.0:0")
	if _, err = Start("test"); err == nil {
		t.Fatal("remote profiling was allowed")
	}
}
