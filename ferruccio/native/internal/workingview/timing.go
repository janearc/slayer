package workingview

import (
	"time"

	"github.com/janearc/slayer/internal/observability"
)

const (
	workingCullMetric   = "ferruccio_working_cull_seconds"
	workingUploadMetric = "ferruccio_working_upload_prepare_seconds"
	workingEncodeMetric = "ferruccio_working_encode_seconds"
	workingSubmitMetric = "ferruccio_working_submit_seconds"
)

// drawTiming splits host-side work without using additional clocks when disabled.
type drawTiming struct {
	enabled bool
	uploads time.Duration
}

func (timing *drawTiming) now() time.Time {
	if timing.enabled {
		return time.Now()
	}
	return time.Time{}
}

func (timing *drawTiming) uploaded(start time.Time) {
	if !start.IsZero() {
		timing.uploads += time.Since(start)
	}
}

func (timing *drawTiming) culling(start, end time.Time) {
	if timing.enabled {
		observability.Observe(workingCullMetric, (end.Sub(start) - timing.uploads).Seconds())
	}
}

func (timing *drawTiming) preparation(start, end time.Time) {
	if timing.enabled {
		observability.Observe(workingUploadMetric, (end.Sub(start) + timing.uploads).Seconds())
	}
}

func observeDrawPhase(name string, start time.Time) {
	if !start.IsZero() {
		observability.Observe(name, time.Since(start).Seconds())
	}
}
