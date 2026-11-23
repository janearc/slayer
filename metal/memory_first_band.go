//go:build darwin && cgo

package metal

import (
	"io"
	"os"

	"github.com/janearc/slayer/internal/observability"
)

// firstBandMemory observes the first successful band of each draw only.
// Its samples are process-wide snapshots, not GPU allocation accounting.
// The copy sample may overlap execution of the next band.
type firstBandMemory struct {
	seen uint8
	emit func(string)
}

const (
	bandMemoryCommit = iota
	bandMemoryWait
	bandMemoryCopy
	bandMemoryConsume
)

var firstBandMemoryNames = [...]string{
	"first-band-committed",
	"first-band-waited",
	"first-band-copied",
	"first-band-consumed",
}

// newFirstBandMemory observes the first band of a draw, reporting to
// standard error when memory stages are enabled.
func newFirstBandMemory() firstBandMemory {
	return newFirstBandMemoryWith(os.Stderr, observability.MemoryStagesEnabled(), MemorySnapshot)
}

// newFirstBandMemoryWith observes the first band of a draw, reporting to out
// with the device sampled by sample; with enabled false it does nothing.
func newFirstBandMemoryWith(out io.Writer, enabled bool, sample deviceMemorySampler) firstBandMemory {
	if !enabled {
		return firstBandMemory{}
	}
	return firstBandMemory{emit: func(stage string) {
		_ = observability.WriteMemoryStage(out, stage)
		reportDeviceMemoryStage(out, true, stage, sample)
	}}
}

// mark reports the stage named for a band event, once, and never again for
// the same event.
func (marks *firstBandMemory) mark(which int) {
	if marks.emit == nil || marks.seen&(1<<which) != 0 {
		return
	}
	marks.seen |= 1 << which
	marks.emit(firstBandMemoryNames[which])
}
