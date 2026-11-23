package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"github.com/janearc/slayer/internal/observability"
	"go.uber.org/mock/gomock"
)

type sinkFunc func(context.Context, string) (bool, error)

func (f sinkFunc) Send(ctx context.Context, text string) (bool, error) { return f(ctx, text) }

func TestReporterShutdownBoundsStuckSamplerAndPreventsLateSend(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	r := &reporter{collect: func() []observability.Sample { close(entered); <-release; return nil }, interval: time.Millisecond, log: io.Discard, sinks: []namedSink{{"fake", sinkFunc(func(context.Context, string) (bool, error) { calls.Add(1); return true, nil })}}}
	stop := startReporter(context.Background(), r, time.Now().Add(-time.Minute))
	<-entered
	started := time.Now()
	stop()
	if time.Since(started) > time.Second {
		t.Fatal("stop waited for sampler")
	}
	close(release)
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatal("late message")
	}
}

func TestReporterShutdownBoundsStuckSinkAndSkipsLaterSink(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var later atomic.Int32
	r := &reporter{collect: func() []observability.Sample { return nil }, interval: time.Millisecond, log: io.Discard, sinks: []namedSink{
		{"blocked", sinkFunc(func(context.Context, string) (bool, error) { close(entered); <-release; return false, nil })},
		{"later", sinkFunc(func(context.Context, string) (bool, error) { later.Add(1); return true, nil })},
	}}
	stop := startReporter(context.Background(), r, time.Now().Add(-time.Minute))
	<-entered
	started := time.Now()
	stop()
	if time.Since(started) > time.Second {
		t.Fatal("stop waited for sink")
	}
	close(release)
	time.Sleep(30 * time.Millisecond)
	if later.Load() != 0 {
		t.Fatal("later sink dispatched after stop")
	}
}

func TestReporterNormalStopSendsFinalWithinBudget(t *testing.T) {
	messages := make(chan string, 1)
	r := &reporter{collect: func() []observability.Sample { return nil }, interval: time.Hour, log: io.Discard, sinks: []namedSink{{"fake", sinkFunc(func(_ context.Context, text string) (bool, error) { messages <- text; return true, nil })}}}
	stop := startReporter(context.Background(), r, time.Now().Add(-time.Minute))
	stop()
	select {
	case message := <-messages:
		if !strings.Contains(message, "worker stopping") {
			t.Fatal(message)
		}
	default:
		t.Fatal("missing normal final")
	}
}

// waitsForEver is a write that never comes back.
func waitsForEver([]byte) (int, error) { select {} }

// neverWriter is a writer whose every write waits for ever.
func neverWriter(t *testing.T) *mocks.MockWriter {
	writer := mocks.NewMockWriter(gomock.NewController(t))
	writer.EXPECT().Write(gomock.Any()).DoAndReturn(waitsForEver).AnyTimes()
	return writer
}

func TestInvalidReporterConfigurationDoesNotWaitForLog(t *testing.T) {
	t.Setenv("AZAZEL_STATS_BARKEEP", "invalid")
	t.Setenv("AZAZEL_STATS_BARKEEP_KIND", "x")
	t.Setenv("AZAZEL_STATS_VENDETTA_CLIENT", "")
	t.Setenv("AZAZEL_STATS_VENDETTA_ROOM", "")
	t.Setenv("AZAZEL_STATS_VENDETTA_AS", "")
	before := observability.SummaryFailures()
	started := time.Now()
	stop := StartReporter(context.Background(), Collector{}, neverWriter(t))
	stop()
	if time.Since(started) > time.Second || observability.SummaryFailures() <= before {
		t.Fatal("invalid config blocked or was not counted")
	}
}

func TestBarkeepDoesNotSayAfterStatusCancellation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var posts atomic.Int32
	client := &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			posts.Add(1)
		} else {
			close(entered)
			<-release
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"queued":0}`))}, nil
	})}
	bar := barkeepSink{address: "http://127.0.0.1:1", kind: "test", client: client}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { bar.Send(ctx, "text"); close(done) }()
	<-entered
	cancel()
	close(release)
	<-done
	if posts.Load() != 0 {
		t.Fatal("post after cancel")
	}
}

func TestBarkeepReceivesCompactSummary(t *testing.T) {
	messages := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/say" {
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			messages <- body["text"]
		}
		_, _ = io.WriteString(w, `{"queued":0}`)
	}))
	defer server.Close()
	samples := []observability.Sample{{Name: "azazel_queue_snapshot_available", Value: 1}, {Name: "azazel_jobs_queued", Value: 3}, {Name: "azazel_gpu_device_utilization_available", Value: 1}, {Name: "azazel_gpu_device_utilization_percent", Value: 88}}
	r := reporter{collect: func() []observability.Sample { return samples }, log: io.Discard, sinks: []namedSink{{"barkeep", barkeepSink{address: server.URL, kind: "test", client: server.Client()}}}}
	r.emit(context.Background(), time.Now(), false)
	select {
	case got := <-messages:
		if got != barkeepSummary(samples) || len(got) > 80 {
			t.Fatal(got)
		}
	default:
		t.Fatal("no Barkeep message")
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBarkeepDoesNotQueueBehindOtherMessages(t *testing.T) {
	queued, posts := 1, 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			posts++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["kind"] != "stats" || body["text"] != "status" {
				t.Fatal(body, err)
			}
		}
		digit := string(rune('0' + queued))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"queued":` + digit + `}`))}, nil
	})}
	sink := barkeepSink{address: "http://127.0.0.1:1", kind: "stats", client: client}
	if sent, err := sink.Send(context.Background(), "status"); sent || err != nil || posts != 0 {
		t.Fatal(sent, err, posts)
	}
	queued = 0
	if sent, err := sink.Send(context.Background(), "status"); !sent || err != nil || posts != 1 {
		t.Fatal(sent, err, posts)
	}
}

func TestVendettaUsesArgumentVectorAndExplicitDestination(t *testing.T) {
	var args []string
	sink := vendettaSink{client: "/bin/client", room: "stats", name: "azazel", run: func(_ context.Context, client, name string, argv ...string) error {
		args = append([]string{client, name}, argv...)
		return nil
	}}
	text := "gpu 53%; $(not a shell command)"
	if sent, err := sink.Send(context.Background(), text); !sent || err != nil {
		t.Fatal(sent, err)
	}
	if strings.Join(args, "|") != "/bin/client|azazel|say|stats|"+text {
		t.Fatal(args)
	}
}

func TestSummaryAndReporterRate(t *testing.T) {
	samples := []observability.Sample{{Name: "azazel_queue_snapshot_available", Value: 1}, {Name: "azazel_jobs_queued", Value: 2}, {Name: "azazel_jobs_running", Value: 1}, {Name: "azazel_gpu_device_utilization_available", Value: 1}, {Name: "azazel_gpu_device_utilization_percent", Value: 53}, {Name: "azazel_host_swap_available", Value: 0}}
	message := summary(samples)
	if !strings.Contains(message, "GPU 53%") || !strings.Contains(message, "swap ?") || !strings.Contains(message, "2 queued") {
		t.Fatal(message)
	}
	samples = append(samples, observability.Sample{Name: "azazel_render_group_footprint_known", Value: 1}, observability.Sample{Name: "azazel_render_group_footprint_bytes", Value: 2 << 30})
	if message = summary(samples); !strings.Contains(message, "render footprint 2 GiB") {
		t.Fatal(message)
	}
	sink := &recordSink{}
	reporter := reporter{collect: func() []observability.Sample { return samples }, sinks: []namedSink{{name: "test", sink: sink}}, interval: 5 * time.Minute, log: io.Discard}
	now := time.Unix(100, 0)
	reporter.report(context.Background(), now)
	reporter.report(context.Background(), now.Add(4*time.Minute))
	reporter.report(context.Background(), now.Add(5*time.Minute))
	if len(sink.messages) != 2 {
		t.Fatal(sink.messages)
	}
}

func TestSummaryFitsVendettaAndPutsQueueGPUAndMemoryFirst(t *testing.T) {
	samples := []observability.Sample{
		{Name: "azazel_queue_snapshot_available", Value: 1}, {Name: "azazel_jobs_queued", Value: 12}, {Name: "azazel_jobs_running", Value: 2},
		{Name: "azazel_gpu_device_utilization_available", Value: 1}, {Name: "azazel_gpu_device_utilization_percent", Value: 98},
		{Name: "azazel_host_cpu_busy_known", Value: 1}, {Name: "azazel_host_cpu_busy_percent", Value: 70},
		{Name: "azazel_host_memory_pressure_available", Value: 1}, {Name: "azazel_host_memory_pressure_level", Value: 2},
		{Name: "azazel_host_swap_available", Value: 1}, {Name: "azazel_host_swap_used_bytes", Value: 3 << 30},
		{Name: "azazel_render_group_resident_known", Value: 1}, {Name: "azazel_render_group_resident_bytes", Value: 5 << 30},
		{Name: "azazel_render_group_footprint_known", Value: 1}, {Name: "azazel_render_group_footprint_bytes", Value: 6 << 30},
	}
	message := summary(samples)
	if len(message) > 190 {
		t.Fatalf("summary too long (%d): %s", len(message), message)
	}
	for _, field := range []string{"12 queued", "2 running", "GPU 98%", "CPU 70%", "pressure 2", "swap 3 GiB", "render RSS 5 GiB", "render footprint 6 GiB"} {
		if !strings.Contains(message, field) {
			t.Errorf("missing %q in %s", field, message)
		}
	}
	if len(message) < 80 || !strings.Contains(message[:80], "pressure 2") || !strings.Contains(message[:80], "swap 3 GiB") {
		t.Fatalf("queue, utilization and memory should lead the display: %s", message)
	}
	short := barkeepSummary(samples)
	if len(short) > 72 {
		t.Fatalf("Barkeep summary too long (%d): %s", len(short), short)
	}
	for _, field := range []string{"q 12", "run 2", "GPU 98%", "CPU 70%", "mem warn", "swap 3G", "phys 6G"} {
		if !strings.Contains(short, field) {
			t.Errorf("missing %q in Barkeep summary %s", field, short)
		}
	}
}

func TestReporterFinalAndInterval(t *testing.T) {
	for _, raw := range []string{"0", "59", "86401", "no"} {
		if _, err := reportInterval(raw); err == nil {
			t.Fatal("accepted", raw)
		}
	}
	if got, err := reportInterval("60"); err != nil || got != time.Minute {
		t.Fatal(got, err)
	}
	if got, err := reportInterval(""); err != nil || got != 5*time.Minute {
		t.Fatal(got, err)
	}
	sink := &recordSink{}
	r := reporter{collect: func() []observability.Sample { return nil }, sinks: []namedSink{{name: "test", sink: sink}}, interval: time.Minute, log: io.Discard}
	start := time.Unix(100, 0)
	r.finish(context.Background(), start, start.Add(4*time.Second))
	if len(sink.messages) != 0 {
		t.Fatal(sink.messages)
	}
	r.finish(context.Background(), start, start.Add(10*time.Second))
	if len(sink.messages) != 1 || !strings.Contains(sink.messages[0], "worker stopping") {
		t.Fatal(sink.messages)
	}
	r.finish(context.Background(), start, start.Add(12*time.Second))
	if len(sink.messages) != 1 {
		t.Fatal(sink.messages)
	}
}

type recordSink struct{ messages []string }

func (s *recordSink) Send(_ context.Context, message string) (bool, error) {
	s.messages = append(s.messages, message)
	return true, nil
}
