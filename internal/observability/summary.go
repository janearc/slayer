package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const summaryInterval = 5 * time.Minute

var summaryReporterActive atomic.Bool

type summarySink interface {
	Send(context.Context, string) (bool, error)
}

// StartSummary sends only to explicitly configured destinations. It emits at
// most once per five minutes and once on normal exit; it never changes the
// application's result if a destination fails. Each app uses its own prefix:
// SLAYER_STATS_, GRISSOM_STATS_, or FERRUCCIO_STATS_.
func StartSummary(app string, log io.Writer) func() {
	return startSummary(app, log, summaryInterval, 5*time.Second)
}

// startSummary starts the summary reporter for app. It returns a no-op
// stop when app has no destination configured.
func startSummary(app string, log io.Writer, interval, finalSeparation time.Duration) func() {
	if app != "slayer" && app != "grissom" && app != "ferruccio" {
		return func() {}
	}
	if log == nil {
		log = io.Discard
	}
	prefix := strings.ToUpper(app) + "_STATS_"
	var sinks []summarySink
	if address := os.Getenv(prefix + "BARKEEP"); address != "" {
		kind := os.Getenv(prefix + "BARKEEP_KIND")
		if validBarkeepAddress(address) && kind != "" {
			sinks = append(sinks, summaryBarkeep{address: strings.TrimSuffix(address, "/"), kind: kind, client: &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("redirect refused") }}})
		} else {
			NoteSummaryFailure(log, app+" stats: Barkeep needs loopback HTTP address and kind\n")
		}
	}
	client, room, sender := os.Getenv(prefix+"VENDETTA_CLIENT"), os.Getenv(prefix+"VENDETTA_ROOM"), os.Getenv(prefix+"VENDETTA_AS")
	if client != "" || room != "" || sender != "" {
		if filepath.IsAbs(client) && validSummaryDestination(room) && validSummaryDestination(sender) {
			sinks = append(sinks, summaryVendetta{client: client, room: room, sender: sender})
		} else {
			NoteSummaryFailure(log, app+" stats: Vendetta needs an absolute client path, explicit room and sender\n")
		}
	}
	if len(sinks) == 0 {
		return func() {}
	}
	// Existing high-frequency measurements use Enabled to avoid idle overhead.
	// A reporter is itself an opt-in consumer even without an HTTP listener.
	lastSent := make([]time.Time, len(sinks))
	// Acquire before starting so an overlapping old stop cannot disable this consumer.
	consumers.Add(1)
	stop, ok := StartExclusiveReporter(&summaryReporterActive, context.Background(), interval, SummaryShutdownWait, nil,
		func(ctx context.Context) { sendSummary(ctx, app, sinks, log, lastSent, 0) },
		func(ctx context.Context) { sendSummary(ctx, app, sinks, log, lastSent, finalSeparation) })
	if !ok {
		consumers.Add(-1)
		NoteSummaryFailure(log, app+" stats: previous reporter is still stopping\n")
		return func() {}
	}
	var once sync.Once
	return func() { once.Do(func() { stop(); consumers.Add(-1) }) }
}

// sendSummary sends the summary to each sink whose separation has passed.
func sendSummary(parent context.Context, app string, sinks []summarySink, log io.Writer, lastSent []time.Time, separation time.Duration) {
	if parent.Err() != nil {
		return
	}
	message := applicationSummary(app)
	if parent.Err() != nil {
		return
	}
	for i, sink := range sinks {
		if parent.Err() != nil {
			return
		}
		if !lastSent[i].IsZero() && time.Since(lastSent[i]) < separation {
			continue
		}
		ctx, cancel := context.WithTimeout(parent, 2*time.Second)
		text := message
		if _, ok := sink.(summaryBarkeep); ok {
			text = applicationSummaryFor(app, true)
		}
		if ctx.Err() != nil {
			cancel()
			return
		}
		sent, err := sink.Send(ctx, text)
		cancel()
		if err != nil {
			NoteSummaryFailure(log, fmt.Sprintf("%s stats: %v\n", app, err))
		} else if sent {
			lastSent[i] = time.Now()
		}
	}
}

// applicationSummary copies only bounded aggregate metrics, never paths or
// other job-specific labels. CPU and resident memory belong to this process.
func applicationSummary(app string) string {
	return applicationSummaryFor(app, false)
}

// applicationSummaryFor builds the summary line for app, shortened when
// short is set.
func applicationSummaryFor(app string, short bool) string {
	registry.RLock()
	c := make(map[string]float64, len(registry.counters))
	for k, v := range registry.counters {
		c[k] = v
	}
	h := make(map[string]histogram, len(registry.histograms))
	for k, v := range registry.histograms {
		h[k] = v
	}
	registry.RUnlock()
	phase := func(name string) string {
		v, ok := h[name]
		if !ok || v.count == 0 {
			return "?"
		}
		return fmt.Sprintf("%.2gs", v.sum/float64(v.count))
	}
	var work string
	switch app {
	case "slayer":
		work = fmt.Sprintf("%.0f drawn, %.0f kept; draw %s; accel wall %s, GPU trace %s", c["slayer_frames_drawn_total"], c["slayer_frames_kept_total"], phase("slayer_frame_draw_seconds"), phase("slayer_gpu_acceleration_wall_seconds"), phase("slayer_gpu_trace_device_seconds"))
	case "grissom":
		work = fmt.Sprintf("%.0f frames, %.0f shots; film %s; draw %s", c["grissom_film_frames_total"], c["grissom_shots_total"], phase("grissom_film_frame_seconds"), phase("grissom_offscreen_draw_seconds"))
	case "ferruccio":
		work = fmt.Sprintf("preview %.0f ok, %.0f failed, %.0f cancelled; round trip %s; UI %s", c["ferruccio_preview_completed_total"], c["ferruccio_preview_failed_total"], c["ferruccio_preview_cancelled_total"], phase("ferruccio_preview_roundtrip_seconds"), phase("ferruccio_ui_tick_seconds"))
	}
	rss, rssOK, cpu, cpuOK := processUsage()
	resident, cpuText := "RSS ?", "CPU time ?"
	if rssOK {
		resident = fmt.Sprintf("RSS %.2g GiB", float64(rss)/(1<<30))
	}
	if cpuOK {
		cpuText = fmt.Sprintf("CPU time %.2gs", cpu)
	}
	footprint, footprintOK, peak, peakOK := processFootprint()
	footprintText := "footprint ?"
	if footprintOK {
		footprintText = fmt.Sprintf("footprint %.2g GiB", float64(footprint)/(1<<30))
	}
	if peakOK {
		footprintText += fmt.Sprintf(" (peak %.2g)", float64(peak)/(1<<30))
	}
	if short {
		switch app {
		case "slayer":
			work = fmt.Sprintf("%.3g frames; draw %s; GPU %s", c["slayer_frames_drawn_total"], phase("slayer_frame_draw_seconds"), phase("slayer_gpu_trace_device_seconds"))
		case "grissom":
			work = fmt.Sprintf("%.3g frames; draw %s", c["grissom_film_frames_total"], phase("grissom_offscreen_draw_seconds"))
		case "ferruccio":
			work = fmt.Sprintf("%.3g previews; round trip %s", c["ferruccio_preview_completed_total"], phase("ferruccio_preview_roundtrip_seconds"))
		}
		resident, footprintText = "RSS ?", "phys ?"
		if rssOK {
			resident = fmt.Sprintf("RSS %.2gG", float64(rss)/(1<<30))
		}
		if footprintOK {
			footprintText = fmt.Sprintf("phys %.2gG", float64(footprint)/(1<<30))
		}
	}
	return CompactSummary(fmt.Sprintf("%s: %s; %s; %s; %s", app, work, resident, footprintText, cpuText))
}

// CompactSummary leaves room under Vendetta's default 200-character line
// limit, marking omitted fields if unusually large values exceed that budget.
func CompactSummary(s string) string {
	const limit = 190
	if len(s) <= limit {
		return s
	}
	if end := strings.LastIndex(s[:limit-5], ";"); end > 0 {
		return s[:end] + "; ..."
	}
	return s[:limit-3] + "..."
}

type summaryBarkeep struct {
	address, kind string
	client        *http.Client
}

// validBarkeepAddress reports whether address is a plain loopback HTTP URL.
func validBarkeepAddress(address string) bool {
	u, err := url.Parse(address)
	return err == nil && u.Scheme == "http" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Port() != "" && (u.Path == "" || u.Path == "/") && net.ParseIP(u.Hostname()).IsLoopback()
}

// Send posts the message to Barkeep when nothing is queued ahead of it,
// and reports whether it was sent.
func (b summaryBarkeep) Send(ctx context.Context, message string) (bool, error) {
	queued, err := b.request(ctx, "/status", nil)
	if err != nil || queued > 0 {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, err = b.request(ctx, "/say", map[string]string{"text": message, "kind": b.kind, "for": "5s"})
	return err == nil, err
}

// request sends one GET or POST to a Barkeep path, and returns the queued
// count from its reply.
func (b summaryBarkeep) request(ctx context.Context, path string, payload any) (int, error) {
	method := http.MethodGet
	var raw []byte
	if payload != nil {
		method = http.MethodPost
		raw, _ = json.Marshal(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, b.address+path, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := b.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8193))
	if err != nil {
		return 0, err
	}
	if len(body) > 8192 || res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("the Barkeep %s returned %s", path, res.Status)
	}
	var status struct {
		Queued *int `json:"queued"`
	}
	if err := json.Unmarshal(body, &status); err != nil || status.Queued == nil || *status.Queued < 0 {
		return 0, fmt.Errorf("invalid Barkeep response")
	}
	return *status.Queued, nil
}

type summaryVendetta struct{ client, room, sender string }

// validSummaryDestination reports whether s is a short, plain room or
// sender name.
func validSummaryDestination(s string) bool {
	if s == "" || len(s) > 80 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == '#') {
			return false
		}
	}
	return true
}

// Send hands the message to the Vendetta client, and reports whether the
// client accepted it.
func (v summaryVendetta) Send(ctx context.Context, message string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	err := RunVendettaSummary(ctx, v.client, v.sender, v.room, message)
	return err == nil, err
}

// RunVendettaSummary reports queue admission, and treats the client's
// successful-but-cut response as a failure. It does not retry ambiguous sends.
func RunVendettaSummary(ctx context.Context, client, sender, room, message string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, client, "say", room, message)
	cmd.WaitDelay = 200 * time.Millisecond
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "VENDETTA_AS=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "VENDETTA_AS="+sender)
	var output limitedSummaryOutput
	cmd.Stdout, cmd.Stderr = &output, io.Discard
	if err := cmd.Run(); err != nil {
		return err
	}
	if strings.Contains(output.String(), "cut to the longest a line may be") {
		return fmt.Errorf("the Vendetta accepted a truncated summary")
	}
	return nil
}

// Keep diagnostics bounded even if a misconfigured client is noisy.
type limitedSummaryOutput struct{ bytes.Buffer }

// Write keeps at most 1024 bytes of output and discards the rest, reporting
// all of p as written.
func (b *limitedSummaryOutput) Write(p []byte) (int, error) {
	remaining := 1024 - b.Len()
	if remaining > 0 {
		keptBytes := min(remaining, len(p))
		_, _ = b.Buffer.Write(p[:keptBytes])
	}
	return len(p), nil
}
