package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestVendettaClientWaitDelayBoundsInheritedPipe(t *testing.T) {
	directory := t.TempDir()
	client := filepath.Join(directory, "client")
	// The child holds the pipe for thirty seconds and the bound is ten, so
	// a pass is the pipe cut off and not waited out, however loaded the
	// machine is when the tests run together.
	script := "#!/bin/sh\nsleep 30 &\nexit 0\n"
	if err := os.WriteFile(client, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	err := RunVendettaSummary(context.Background(), client, "test", "test", "test")
	if !errors.Is(err, exec.ErrWaitDelay) || time.Since(start) > 10*time.Second {
		t.Fatal("inherited pipe not bounded", err, time.Since(start))
	}
}

type summarySinkFunc func(context.Context, string) (bool, error)

func (f summarySinkFunc) Send(ctx context.Context, text string) (bool, error) { return f(ctx, text) }

func TestSummaryDoesNotDispatchNextSinkAfterCancellation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var later atomic.Int32
	sinks := []summarySink{
		summarySinkFunc(func(context.Context, string) (bool, error) { close(entered); <-release; return false, nil }),
		summarySinkFunc(func(context.Context, string) (bool, error) { later.Add(1); return true, nil }),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { sendSummary(ctx, "slayer", sinks, io.Discard, make([]time.Time, len(sinks)), 0); close(done) }()
	<-entered
	cancel()
	close(release)
	<-done
	if later.Load() != 0 {
		t.Fatal("later sink dispatched after cancellation")
	}
}

func TestSummaryBarkeepChecksCancellationBeforeSay(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var posts atomic.Int32
	client := &http.Client{Transport: summaryRoundTrip(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			posts.Add(1)
		} else {
			close(entered)
			<-release
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"queued":0}`))}, nil
	})}
	bar := summaryBarkeep{address: "http://127.0.0.1:1", kind: "test", client: client}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { bar.Send(ctx, "text"); close(done) }()
	<-entered
	cancel()
	close(release)
	<-done
	if posts.Load() != 0 {
		t.Fatal("say after cancellation")
	}
}

type summaryRoundTrip func(*http.Request) (*http.Response, error)

func (f summaryRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func clearSummaryEnv(t *testing.T, prefix string) {
	t.Helper()
	for _, key := range []string{"BARKEEP", "BARKEEP_KIND", "VENDETTA_CLIENT", "VENDETTA_ROOM", "VENDETTA_AS"} {
		t.Setenv(prefix+key, "")
	}
	t.Setenv("SLAYER_METRICS_ADDR", "")
	t.Setenv("SLAYER_PPROF_ADDR", "")
}

func TestSummaryWithoutMetricsEndpointSendsPeriodicAndFinal(t *testing.T) {
	clearSummaryEnv(t, "SLAYER_STATS_")
	var mu sync.Mutex
	var messages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/say" {
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			mu.Lock()
			messages = append(messages, payload["text"])
			mu.Unlock()
			if payload["kind"] != "proof" || payload["for"] != "5s" {
				t.Errorf("unexpected Barkeep payload %v", payload)
			}
		}
		_, _ = io.WriteString(w, `{"queued":0}`)
	}))
	defer server.Close()
	t.Setenv("SLAYER_STATS_BARKEEP", server.URL)
	t.Setenv("SLAYER_STATS_BARKEEP_KIND", "proof")
	var log bytes.Buffer
	stop := startSummary("slayer", &log, 20*time.Millisecond, 0)
	if !Enabled() {
		t.Fatal("summary alone did not enable observations")
	}
	Add("slayer_frames_drawn_total", 1)
	Observe("slayer_frame_draw_seconds", 0.1)
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		count := len(messages)
		mu.Unlock()
		if count > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	stop()
	stop()
	mu.Lock()
	defer mu.Unlock()
	if len(messages) < 2 {
		t.Fatalf("wanted periodic and final summaries, got %d; %s", len(messages), log.String())
	}
	if !strings.Contains(messages[len(messages)-1], "frames") || !strings.Contains(messages[len(messages)-1], "draw 0.1s") || !strings.Contains(messages[len(messages)-1], "RSS ") {
		t.Fatalf("missing application timings: %s", messages[len(messages)-1])
	}
	if Enabled() {
		t.Fatal("summary left collection enabled")
	}
}

func TestSummaryVendettaReportsCutResponse(t *testing.T) {
	client := filepath.Join(t.TempDir(), "client")
	if err := os.WriteFile(client, []byte("#!/bin/sh\nprintf '%s\\n' 'cut to the longest a line may be'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	sent, err := (summaryVendetta{client: client, room: "stats", sender: "tester"}).Send(context.Background(), "summary")
	if sent || err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("cut response was reported as sent: sent=%v err=%v", sent, err)
	}
}

func TestSummariesFitVendettaAndLeadWithWork(t *testing.T) {
	for _, app := range []string{"slayer", "grissom", "ferruccio"} {
		message := applicationSummary(app)
		if len(message) > 190 || !strings.HasPrefix(message, app+": ") {
			t.Fatalf("invalid %s summary (%d bytes): %s", app, len(message), message)
		}
		if !strings.Contains(message, "RSS ") || !strings.Contains(message, "CPU time ") {
			t.Fatalf("missing process measurements: %s", message)
		}
		short := applicationSummaryFor(app, true)
		if len(short) > 80 || !strings.Contains(short, "RSS ") || !strings.Contains(short, "phys ") || !strings.Contains(short, "CPU time ") {
			t.Fatalf("Barkeep summary does not fit or lacks process measurements (%d bytes): %s", len(short), short)
		}
	}
	message := CompactSummary("app: " + strings.Repeat("1", 170) + "; tail longer than the budget")
	if len(message) > 190 || !strings.HasSuffix(message, "; ...") {
		t.Fatalf("omission was not marked: %q", message)
	}
}

func TestSummaryDoesNotInterruptBusyBarkeep(t *testing.T) {
	clearSummaryEnv(t, "GRISSOM_STATS_")
	said := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/say" {
			said = true
		}
		_, _ = io.WriteString(w, `{"queued":1}`)
	}))
	defer server.Close()
	t.Setenv("GRISSOM_STATS_BARKEEP", server.URL)
	t.Setenv("GRISSOM_STATS_BARKEEP_KIND", "proof")
	StartSummary("grissom", io.Discard)()
	if said {
		t.Fatal("summary interrupted a busy Barkeep")
	}
}

func TestSummaryDoesNotRepeatImmediatelyAfterPeriodicSend(t *testing.T) {
	clearSummaryEnv(t, "GRISSOM_STATS_")
	var mu sync.Mutex
	count := 0
	sent := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/say" {
			mu.Lock()
			count++
			mu.Unlock()
			select {
			case sent <- struct{}{}:
			default:
			}
		}
		_, _ = io.WriteString(w, `{"queued":0}`)
	}))
	defer server.Close()
	t.Setenv("GRISSOM_STATS_BARKEEP", server.URL)
	t.Setenv("GRISSOM_STATS_BARKEEP_KIND", "proof")
	stop := startSummary("grissom", io.Discard, 100*time.Millisecond, 5*time.Second)
	select {
	case <-sent:
	case <-time.After(2 * time.Second):
		t.Fatal("periodic summary did not arrive")
	}
	stop()
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Fatalf("sent %d messages around shutdown, want one", count)
	}
}

func TestSummaryVendettaRequiresExplicitDestinationAndUsesArguments(t *testing.T) {
	clearSummaryEnv(t, "FERRUCCIO_STATS_")
	dir := t.TempDir()
	output := filepath.Join(dir, "output")
	client := filepath.Join(dir, "client")
	if err := os.WriteFile(client, []byte("#!/bin/sh\nprintf '%s\\n%s\\n%s\\n%s\\n' \"$VENDETTA_AS\" \"$1\" \"$2\" \"$3\" > \"$SUMMARY_TEST_OUTPUT\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUMMARY_TEST_OUTPUT", output)
	t.Setenv("FERRUCCIO_STATS_VENDETTA_CLIENT", client)
	t.Setenv("FERRUCCIO_STATS_VENDETTA_AS", "tester")
	StartSummary("ferruccio", io.Discard)()
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("sent without explicit room: %v", err)
	}
	t.Setenv("FERRUCCIO_STATS_VENDETTA_ROOM", "test-room")
	StartSummary("ferruccio", io.Discard)()
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "tester\nsay\ntest-room\nferruccio:") {
		t.Fatalf("unexpected command arguments: %s", b)
	}
}

func TestSummaryRejectsRemoteBarkeep(t *testing.T) {
	for _, value := range []string{"http://example.org:1234", "https://127.0.0.1:1234", "http://127.0.0.1:1234/path", "http://user@127.0.0.1:1234"} {
		if validBarkeepAddress(value) {
			t.Errorf("accepted %s", value)
		}
	}
}

func TestSummaryRestartKeepsObservationsEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"queued":0}`)
	}))
	defer server.Close()
	t.Setenv("SLAYER_STATS_BARKEEP", server.URL)
	t.Setenv("SLAYER_STATS_BARKEEP_KIND", "proof")
	t.Setenv("SLAYER_METRICS_ADDR", "")
	t.Setenv("SLAYER_PPROF_ADDR", "")
	for i := 0; i < 25; i++ {
		oldStop := startSummary("slayer", io.Discard, time.Hour, 0)
		stopped := make(chan struct{})
		go func() { oldStop(); close(stopped) }()
		deadline := time.After(2 * time.Second)
		for summaryReporterActive.Load() {
			select {
			case <-deadline:
				t.Fatal("old reporter did not release its gate")
			default:
				time.Sleep(time.Millisecond)
			}
		}
		newStop := startSummary("slayer", io.Discard, time.Hour, 0)
		<-stopped
		if !Enabled() {
			newStop()
			t.Fatal("old stop disabled the new summary")
		}
		newStop()
		if Enabled() {
			t.Fatal("last summary left observations enabled")
		}
	}
}

func TestListenerAndSummaryOwnIndependentObservations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"queued":0}`)
	}))
	defer server.Close()
	t.Setenv("SLAYER_STATS_BARKEEP", server.URL)
	t.Setenv("SLAYER_STATS_BARKEEP_KIND", "proof")
	t.Setenv("SLAYER_METRICS_ADDR", "127.0.0.1:0")
	t.Setenv("SLAYER_PPROF_ADDR", "")
	stopListener, err := Start("slayer")
	if err != nil {
		t.Fatal(err)
	}
	stopSummary := startSummary("slayer", io.Discard, time.Hour, 0)
	stopListener()
	if !Enabled() {
		stopSummary()
		t.Fatal("listener stop disabled active summary")
	}
	stopSummary()
	if Enabled() {
		t.Fatal("stopped consumers left observations enabled")
	}

	stopListener, err = Start("slayer")
	if err != nil {
		t.Fatal(err)
	}
	stopSummary = startSummary("slayer", io.Discard, time.Hour, 0)
	stopSummary()
	if !Enabled() {
		stopListener()
		t.Fatal("summary stop disabled active listener")
	}
	stopListener()
	if Enabled() {
		t.Fatal("stopped consumers left observations enabled")
	}
}
