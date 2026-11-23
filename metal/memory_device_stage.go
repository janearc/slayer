//go:build darwin && cgo

package metal

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/janearc/slayer/internal/observability"
)

type deviceMemorySampler func() (allocated, recommended uint64, available bool)

// ReportDeviceMemoryStage is opt-in and must be called after a Metal device
// is selected. This snapshot is device-wide and may include other processes;
// it is not additive to the process's physical footprint.
func ReportDeviceMemoryStage(stage string) {
	reportDeviceMemoryStage(os.Stderr, observability.MemoryStagesEnabled(), stage, MemorySnapshot)
}

// reportDeviceMemoryStage writes one line of JSON to out when enabled: the
// stage, and the default device's allocated and recommended bytes as the
// sampler reads them.
func reportDeviceMemoryStage(out io.Writer, enabled bool, stage string, sample deviceMemorySampler) {
	if !enabled {
		return
	}
	allocated, recommended, available := sample()
	_ = json.NewEncoder(out).Encode(struct {
		Event       string    `json:"event"`
		Stage       string    `json:"stage"`
		Scope       string    `json:"scope"`
		PID         int       `json:"pid"`
		ObservedAt  time.Time `json:"observed_at"`
		Available   bool      `json:"available"`
		Allocated   uint64    `json:"allocated_bytes"`
		Recommended uint64    `json:"recommended_working_set_bytes"`
	}{"device_memory_stage", stage, "default_metal_device", os.Getpid(), time.Now(), available, allocated, recommended})
}
