package observability

import (
	"encoding/json"
	"io"
	"os"
	"runtime"
	"time"
)

// MemoryStage is a process-wide observation, not an accounting of a GPU
// allocation. Go memory, physical footprint and lifetime peak overlap.
type MemoryStage struct {
	Event              string    `json:"event"`
	Stage              string    `json:"stage"`
	PID                int       `json:"pid"`
	ObservedAt         time.Time `json:"observed_at"`
	HeapAlloc          uint64    `json:"go_heap_alloc_bytes"`
	HeapInuse          uint64    `json:"go_heap_inuse_bytes"`
	HeapSys            uint64    `json:"go_heap_sys_bytes"`
	HeapReleased       uint64    `json:"go_heap_released_bytes"`
	Sys                uint64    `json:"go_sys_bytes"`
	GCCycles           uint32    `json:"go_gc_cycles"`
	Footprint          uint64    `json:"physical_footprint_bytes"`
	FootprintAvailable bool      `json:"physical_footprint_available"`
	Peak               uint64    `json:"lifetime_peak_footprint_bytes"`
	PeakAvailable      bool      `json:"lifetime_peak_footprint_available"`
}

// WriteMemoryStage samples this process without forcing a collection and
// writes one JSON line. The Go and physical samples are not simultaneous.
func WriteMemoryStage(out io.Writer, stage string) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	current, currentOK, peak, peakOK := processFootprint()
	return json.NewEncoder(out).Encode(MemoryStage{
		Event: "memory_stage", Stage: stage, PID: os.Getpid(), ObservedAt: time.Now(),
		HeapAlloc: mem.HeapAlloc, HeapInuse: mem.HeapInuse, HeapSys: mem.HeapSys,
		HeapReleased: mem.HeapReleased, Sys: mem.Sys, GCCycles: mem.NumGC,
		Footprint: current, FootprintAvailable: currentOK, Peak: peak, PeakAvailable: peakOK,
	})
}
