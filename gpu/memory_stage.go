package gpu

import (
	"io"
	"os"

	"github.com/janearc/slayer/internal/observability"
)

// memoryStage is diagnostic only. A missing or failed report does not affect
// a render; enabling it incurs process sampling and stderr I/O.
func memoryStage(stage string) {
	reportMemoryStage(os.Stderr, observability.MemoryStagesEnabled(), stage)
}

// reportMemoryStage writes the memory report for a stage to out when
// enabled, and writes nothing when not.
func reportMemoryStage(out io.Writer, enabled bool, stage string) {
	if enabled {
		_ = observability.WriteMemoryStage(out, stage)
	}
}
