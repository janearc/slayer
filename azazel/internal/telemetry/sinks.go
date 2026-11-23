package telemetry

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
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/janearc/slayer/internal/observability"
)

type sink interface {
	Send(context.Context, string) (bool, error)
}

var queueReporterActive atomic.Bool

type namedSink struct {
	name string
	sink sink
}

type reporter struct {
	collect  func() []observability.Sample
	sinks    []namedSink
	interval time.Duration
	last     time.Time
	log      io.Writer
}

// report emits a summary when the interval has passed since the last one.
func (r *reporter) report(ctx context.Context, now time.Time) {
	if !r.last.IsZero() && now.Sub(r.last) < r.interval {
		return
	}
	r.emit(ctx, now, false)
}

// emit collects one set of readings and sends the summary to every sink. A
// final summary uses the stopping wording.
func (r *reporter) emit(ctx context.Context, now time.Time, final bool) {
	if ctx.Err() != nil {
		return
	}
	r.last = now
	samples := r.collect()
	if ctx.Err() != nil {
		return
	}
	message := summary(samples)
	if final {
		message = observability.CompactSummary(strings.Replace(message, "Azazel:", "Azazel (worker stopping):", 1))
	}
	for _, target := range r.sinks {
		if ctx.Err() != nil {
			return
		}
		call, cancel := context.WithTimeout(ctx, 2*time.Second)
		text := message
		if _, ok := target.sink.(barkeepSink); ok {
			text = barkeepSummary(samples)
			if final {
				text = strings.Replace(text, "Azazel:", "Azazel stop:", 1)
			}
		}
		sent, err := target.sink.Send(call, text)
		cancel()
		if err != nil {
			observability.Add("azazel_stats_"+target.name+"_failures_total", 1)
			observability.NoteSummaryFailure(r.log, fmt.Sprintf("stats %s: %v\n", target.name, err))
		} else if sent {
			observability.Add("azazel_stats_"+target.name+"_sent_total", 1)
		} else {
			observability.Add("azazel_stats_"+target.name+"_skipped_total", 1)
		}
	}
}

// finish sends a final summary, unless the run lasted under five seconds or a
// summary went out less than five seconds ago.
func (r *reporter) finish(ctx context.Context, started, now time.Time) {
	if now.Sub(started) < 5*time.Second || !r.last.IsZero() && now.Sub(r.last) < 5*time.Second {
		return
	}
	r.emit(ctx, now, true)
}

// reportInterval reads the summary interval in seconds. Empty means five
// minutes; the value must be from 60 to 86400.
func reportInterval(raw string) (time.Duration, error) {
	if raw == "" {
		return 5 * time.Minute, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 60 || seconds > 86400 {
		return 0, fmt.Errorf("interval must be between 60 and 86400 seconds")
	}
	return time.Duration(seconds) * time.Second, nil
}

// StartReporter starts only explicitly configured local sinks. Each receives
// periodically and at shutdown; errors never stop the worker.
func StartReporter(ctx context.Context, collector Collector, log io.Writer) func() {
	interval, err := reportInterval(os.Getenv("AZAZEL_STATS_INTERVAL_SECONDS"))
	if err != nil {
		observability.NoteSummaryFailure(log, fmt.Sprintf("stats: %v; using 300 seconds\n", err))
		interval = 5 * time.Minute
	}
	r := &reporter{collect: collector.Collect, interval: interval, log: log}
	if address := os.Getenv("AZAZEL_STATS_BARKEEP"); address != "" {
		kind := os.Getenv("AZAZEL_STATS_BARKEEP_KIND")
		normalized, err := barkeepAddress(address)
		if err != nil || kind == "" {
			observability.NoteSummaryFailure(log, "stats barkeep: need a loopback HTTP address and configured kind\n")
		} else {
			client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("redirect refused") }}
			r.sinks = append(r.sinks, namedSink{name: "barkeep", sink: barkeepSink{address: normalized, kind: kind, client: client}})
		}
	}
	client, room, name := os.Getenv("AZAZEL_STATS_VENDETTA_CLIENT"), os.Getenv("AZAZEL_STATS_VENDETTA_ROOM"), os.Getenv("AZAZEL_STATS_VENDETTA_AS")
	if client != "" || room != "" || name != "" {
		if !filepath.IsAbs(client) || !validDestination(room) || !validDestination(name) {
			observability.NoteSummaryFailure(log, "stats vendetta: need absolute client, explicit room and sender\n")
		} else {
			r.sinks = append(r.sinks, namedSink{name: "vendetta", sink: vendettaSink{client: client, room: room, name: name, run: runVendetta}})
		}
	}
	if len(r.sinks) == 0 {
		return func() {}
	}
	started := time.Now()
	stop, ok := observability.StartExclusiveReporter(&queueReporterActive, ctx, r.interval, observability.SummaryShutdownWait,
		func(ctx context.Context) {
			if ctx.Err() == nil {
				r.collect()
			}
		},
		func(ctx context.Context) { r.report(ctx, time.Now()) },
		func(ctx context.Context) { r.finish(ctx, started, time.Now()) })
	if !ok {
		observability.NoteSummaryFailure(log, "stats: previous reporter is still stopping\n")
		return func() {}
	}
	return stop
}

// startReporter starts the bounded reporter loop and returns its stop
// function.
func startReporter(ctx context.Context, r *reporter, started time.Time) func() {
	return observability.StartBoundedReporter(ctx, r.interval, observability.SummaryShutdownWait,
		func(ctx context.Context) {
			if ctx.Err() == nil {
				r.collect()
			}
		},
		func(ctx context.Context) { r.report(ctx, time.Now()) },
		func(ctx context.Context) { r.finish(ctx, started, time.Now()) })
}

// summary formats the samples as one compact Azazel line for Vendetta.
func summary(samples []observability.Sample) string {
	v := make(map[string]float64, len(samples))
	for _, sample := range samples {
		v[sample.Name] = sample.Value
	}
	queueText := "queue ?"
	if v["azazel_queue_snapshot_available"] == 1 {
		queueText = fmt.Sprintf("%.0f queued, %.0f running", v["azazel_jobs_queued"], v["azazel_jobs_running"])
	}
	gpu := "GPU ?"
	if v["azazel_gpu_device_utilization_available"] == 1 {
		gpu = fmt.Sprintf("GPU %.0f%%", v["azazel_gpu_device_utilization_percent"])
	}
	cpu := "CPU ?"
	if v["azazel_host_cpu_busy_known"] == 1 {
		cpu = fmt.Sprintf("CPU %.0f%%", v["azazel_host_cpu_busy_percent"])
	}
	swap := "swap ?"
	if v["azazel_host_swap_available"] == 1 {
		swap = fmt.Sprintf("swap %.2g GiB", v["azazel_host_swap_used_bytes"]/(1<<30))
	}
	pressure := "pressure ?"
	if v["azazel_host_memory_pressure_available"] == 1 {
		pressure = fmt.Sprintf("pressure %.0f", v["azazel_host_memory_pressure_level"])
	}
	resident := "render RSS ?"
	if v["azazel_render_group_resident_known"] == 1 {
		resident = fmt.Sprintf("render RSS %.2g GiB", v["azazel_render_group_resident_bytes"]/(1<<30))
	}
	foot := "render footprint ?"
	if v["azazel_render_group_footprint_known"] == 1 {
		foot = fmt.Sprintf("render footprint %.2g GiB", v["azazel_render_group_footprint_bytes"]/(1<<30))
	}
	return observability.CompactSummary(fmt.Sprintf("Azazel: %s; %s; %s; %s; %s; %s; %s", queueText, gpu, cpu, pressure, swap, resident, foot))
}

// Keep the most useful fields first: Barkeep clips to its terminal width and
// may reserve additional columns for signals.
func barkeepSummary(samples []observability.Sample) string {
	v := make(map[string]float64, len(samples))
	for _, sample := range samples {
		v[sample.Name] = sample.Value
	}
	queue, gpu, cpu, pressure, swap, footprint := "q ? run ?", "GPU ?", "CPU ?", "mem ?", "swap ?", "phys ?"
	if v["azazel_queue_snapshot_available"] == 1 {
		queue = fmt.Sprintf("q %.3g run %.3g", v["azazel_jobs_queued"], v["azazel_jobs_running"])
	}
	if v["azazel_gpu_device_utilization_available"] == 1 {
		gpu = fmt.Sprintf("GPU %.0f%%", v["azazel_gpu_device_utilization_percent"])
	}
	if v["azazel_host_cpu_busy_known"] == 1 {
		cpu = fmt.Sprintf("CPU %.0f%%", v["azazel_host_cpu_busy_percent"])
	}
	if v["azazel_host_memory_pressure_available"] == 1 {
		switch v["azazel_host_memory_pressure_level"] {
		case 1:
			pressure = "mem OK"
		case 2:
			pressure = "mem warn"
		case 4:
			pressure = "mem critical"
		}
	}
	if v["azazel_host_swap_available"] == 1 {
		swap = fmt.Sprintf("swap %.2gG", v["azazel_host_swap_used_bytes"]/(1<<30))
	}
	if v["azazel_render_group_footprint_known"] == 1 {
		footprint = fmt.Sprintf("phys %.2gG", v["azazel_render_group_footprint_bytes"]/(1<<30))
	}
	return observability.CompactSummary(fmt.Sprintf("Azazel: %s; %s; %s; %s; %s; %s", queue, gpu, cpu, pressure, swap, footprint))
}

type barkeepSink struct {
	address, kind string
	client        *http.Client
}

// barkeepAddress checks that address is a loopback HTTP address with a port
// and no path, query, fragment or user.
func barkeepAddress(address string) (string, error) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Port() == "" || (parsed.Path != "" && parsed.Path != "/") || !net.ParseIP(parsed.Hostname()).IsLoopback() {
		return "", fmt.Errorf("the Barkeep requires a loopback HTTP address")
	}
	return strings.TrimSuffix(address, "/"), nil
}

// Send posts message to Barkeep when its queue is empty. It reports false,
// with no error, when Barkeep already has items queued.
func (bar barkeepSink) Send(ctx context.Context, message string) (bool, error) {
	status, err := bar.request(ctx, "/status", nil)
	if err != nil || status > 0 {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, err = bar.request(ctx, "/say", map[string]string{"text": message, "kind": bar.kind, "for": "5s"})
	return err == nil, err
}

// request calls a Barkeep path, with JSON when payload is set, and returns the
// queued count from the reply.
func (bar barkeepSink) request(ctx context.Context, path string, payload any) (int, error) {
	method, raw := http.MethodGet, []byte(nil)
	if payload != nil {
		method = http.MethodPost
		raw, _ = json.Marshal(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, bar.address+path, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := bar.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8193))
	if err != nil {
		return 0, err
	}
	if len(body) > 8192 || response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("the Barkeep %s returned %s", path, response.Status)
	}
	var status struct {
		Queued *int `json:"queued"`
	}
	if err := json.Unmarshal(body, &status); err != nil || status.Queued == nil || *status.Queued < 0 {
		return 0, fmt.Errorf("invalid Barkeep response")
	}
	return *status.Queued, nil
}

type vendettaSink struct {
	client, room, name string
	run                func(context.Context, string, string, ...string) error
}

// Send hands message to the Vendetta client as a say command.
func (v vendettaSink) Send(ctx context.Context, message string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	err := v.run(ctx, v.client, v.name, "say", v.room, message)
	return err == nil, err
}

// validDestination reports whether value is a short name made of letters,
// digits and the characters - _ . #.
func validDestination(value string) bool {
	if value == "" || len(value) > 80 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == '#') {
			return false
		}
	}
	return true
}

// runVendetta checks the say arguments, then sends them through the Vendetta
// summary helper.
func runVendetta(ctx context.Context, client, name string, args ...string) error {
	if len(args) != 3 || args[0] != "say" {
		return fmt.Errorf("invalid Vendetta summary arguments")
	}
	return observability.RunVendettaSummary(ctx, client, name, args[1], args[2])
}
