// Package observability provides opt-in metrics and profiling endpoints.
package observability

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Sample is a gauge read at scrape time. Its name must be a static metric name.
type Sample struct {
	Name  string
	Value float64
}

var bounds = [...]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600, 1200, 3600, 14400}

type histogram struct {
	count   uint64
	sum     float64
	buckets [len(bounds)]uint64
}

var registry = struct {
	sync.RWMutex
	gauges        map[string]float64
	counters      map[string]float64
	histograms    map[string]histogram
	collectors    map[uint64]func() []Sample
	nextCollector uint64
}{gauges: make(map[string]float64), counters: make(map[string]float64), histograms: make(map[string]histogram), collectors: make(map[uint64]func() []Sample)}

// valid reports whether a metric name and value may be recorded.
func valid(name string, value float64) bool {
	if name == "" || name == "observability_summary_failures_total" || name == "observability_summary_log_dropped_total" || strings.HasPrefix(name, "go_") || strings.HasPrefix(name, "process_") || math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	for i, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == ':' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// Set records the current value of a gauge.
func Set(name string, value float64) {
	if !valid(name, value) {
		return
	}
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.counters[name]; ok {
		return
	}
	if _, ok := registry.histograms[name]; ok {
		return
	}
	if histogramPart(name, registry.histograms) {
		return
	}
	registry.gauges[name] = value
}

// Add increases a counter by a nonnegative amount.
func Add(name string, value float64) {
	if !valid(name, value) || value < 0 {
		return
	}
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.gauges[name]; ok {
		return
	}
	if _, ok := registry.histograms[name]; ok {
		return
	}
	if histogramPart(name, registry.histograms) {
		return
	}
	registry.counters[name] += value
}

// Observe records a nonnegative duration in seconds in a cumulative histogram.
func Observe(name string, value float64) {
	if !valid(name, value) || value < 0 {
		return
	}
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.gauges[name]; ok {
		return
	}
	if _, ok := registry.counters[name]; ok {
		return
	}
	if histogramPart(name, registry.histograms) {
		return
	}
	for _, suffix := range []string{"_bucket", "_sum", "_count"} {
		if _, ok := registry.gauges[name+suffix]; ok {
			return
		}
		if _, ok := registry.counters[name+suffix]; ok {
			return
		}
		if _, ok := registry.histograms[name+suffix]; ok {
			return
		}
	}
	h := registry.histograms[name]
	h.count++
	h.sum += value
	for i, bound := range bounds {
		if value <= bound {
			h.buckets[i]++
		}
	}
	registry.histograms[name] = h
}

// histogramPart reports whether name is a bucket, sum or count series of a
// registered histogram.
func histogramPart(name string, histograms map[string]histogram) bool {
	for _, suffix := range []string{"_bucket", "_sum", "_count"} {
		if strings.HasSuffix(name, suffix) {
			if _, ok := histograms[strings.TrimSuffix(name, suffix)]; ok {
				return true
			}
		}
	}
	return false
}

// RegisterCollector adds scrape-time gauges; callbacks must be concurrency safe.
func RegisterCollector(collect func() []Sample) func() {
	if collect == nil {
		return func() {}
	}
	registry.Lock()
	id := registry.nextCollector
	registry.nextCollector++
	registry.collectors[id] = collect
	registry.Unlock()
	return func() { registry.Lock(); delete(registry.collectors, id); registry.Unlock() }
}

// number formats a metric value in the shortest general form.
func number(value float64) string { return strconv.FormatFloat(value, 'g', -1, 64) }

// Handler serves Prometheus text format on /metrics, without profiling routes.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		writeMetrics(w)
	})
	return mux
}

// writeMetrics writes every series to w in Prometheus text format.
func writeMetrics(w io.Writer) {
	registry.RLock()
	gauges := make(map[string]float64, len(registry.gauges))
	for k, v := range registry.gauges {
		gauges[k] = v
	}
	counters := make(map[string]float64, len(registry.counters))
	for k, v := range registry.counters {
		counters[k] = v
	}
	histograms := make(map[string]histogram, len(registry.histograms))
	for k, v := range registry.histograms {
		histograms[k] = v
	}
	collectors := make([]func() []Sample, 0, len(registry.collectors))
	for _, collect := range registry.collectors {
		collectors = append(collectors, collect)
	}
	registry.RUnlock()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	gauges["go_goroutines"] = float64(runtime.NumGoroutine())
	gauges["go_gomaxprocs"] = float64(runtime.GOMAXPROCS(0))
	gauges["go_memstats_heap_alloc_bytes"] = float64(memory.HeapAlloc)
	gauges["go_memstats_heap_sys_bytes"] = float64(memory.HeapSys)
	gauges["go_memstats_heap_inuse_bytes"] = float64(memory.HeapInuse)
	gauges["go_memstats_heap_idle_bytes"] = float64(memory.HeapIdle)
	gauges["go_memstats_heap_released_bytes"] = float64(memory.HeapReleased)
	gauges["go_memstats_heap_objects"] = float64(memory.HeapObjects)
	gauges["go_memstats_stack_inuse_bytes"] = float64(memory.StackInuse)
	gauges["go_memstats_sys_bytes"] = float64(memory.Sys)
	gauges["go_gc_cpu_fraction"] = memory.GCCPUFraction
	counters["go_gc_cycles_total"] = float64(memory.NumGC)
	counters["go_gc_pause_seconds_total"] = float64(memory.PauseTotalNs) / 1e9
	counters["go_memstats_alloc_bytes_total"] = float64(memory.TotalAlloc)
	counters["observability_summary_failures_total"] = float64(SummaryFailures())
	counters["observability_summary_log_dropped_total"] = float64(SummaryLogDrops())
	rss, rssOK, cpuSeconds, cpuOK := processUsage()
	if rssOK {
		gauges["process_resident_memory_available"] = 1
		gauges["process_resident_memory_bytes"] = float64(rss)
	} else {
		gauges["process_resident_memory_available"] = 0
	}
	if cpuOK {
		gauges["process_cpu_available"] = 1
		counters["process_cpu_seconds_total"] = cpuSeconds
	} else {
		gauges["process_cpu_available"] = 0
	}
	footprint, footprintOK, peak, peakOK := processFootprint()
	if footprintOK {
		gauges["process_physical_footprint_available"] = 1
		gauges["process_physical_footprint_bytes"] = float64(footprint)
	} else {
		gauges["process_physical_footprint_available"] = 0
	}
	if peakOK {
		gauges["process_peak_physical_footprint_available"] = 1
		gauges["process_peak_physical_footprint_bytes"] = float64(peak)
	} else {
		gauges["process_peak_physical_footprint_available"] = 0
	}
	for _, collect := range collectors {
		for _, sample := range collect() {
			_, gaugeExists := gauges[sample.Name]
			_, counterExists := counters[sample.Name]
			_, histogramExists := histograms[sample.Name]
			if valid(sample.Name, sample.Value) && !gaugeExists && !counterExists && !histogramExists && !histogramPart(sample.Name, histograms) {
				gauges[sample.Name] = sample.Value
			}
		}
	}
	names := make([]string, 0, len(gauges)+len(counters)+len(histograms))
	for name := range gauges {
		names = append(names, name)
	}
	for name := range counters {
		names = append(names, name)
	}
	for name := range histograms {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		switch {
		case histograms[name].count > 0:
			h := histograms[name]
			fmt.Fprintf(w, "# TYPE %s histogram\n", name)
			for i, bound := range bounds {
				fmt.Fprintf(w, "%s_bucket{le=\"%s\"} %d\n", name, number(bound), h.buckets[i])
			}
			fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n%s_sum %s\n%s_count %d\n", name, h.count, name, number(h.sum), name, h.count)
		case hasCounter(counters, name):
			fmt.Fprintf(w, "# TYPE %s counter\n%s %s\n", name, name, number(counters[name]))
		default:
			fmt.Fprintf(w, "# TYPE %s gauge\n%s %s\n", name, name, number(gauges[name]))
		}
	}
}

// hasCounter reports whether name is a registered counter.
func hasCounter(counters map[string]float64, name string) bool {
	_, ok := counters[name]
	return ok
}
