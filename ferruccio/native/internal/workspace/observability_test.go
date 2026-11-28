package workspace

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/janearc/slayer/internal/observability"
)

func metricValue(t *testing.T, name string) float64 {
	t.Helper()
	w := httptest.NewRecorder()
	observability.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	for _, line := range strings.Split(w.Body.String(), "\n") {
		if strings.HasPrefix(line, name+" ") {
			var value float64
			if _, err := fmt.Sscanf(line[len(name)+1:], "%f", &value); err != nil {
				t.Fatal(err)
			}
			return value
		}
	}
	return 0
}

func TestWorkingMotionMetricsCountOnlyAcceptedFrames(t *testing.T) {
	const motion = "ferruccio_working_motion_to_pixels_seconds"
	const interval = "ferruccio_working_motion_frame_interval_seconds"
	const coalesced = "ferruccio_working_motion_coalesced_total"
	beforeMotion := metricValue(t, motion+"_count")
	beforeSum := metricValue(t, motion+"_sum")
	beforeInterval := metricValue(t, interval+"_count")
	beforeCoalesced := metricValue(t, coalesced)
	start := time.Now()
	timing := workingTiming{}
	timing.moved(start)
	timing.moved(start.Add(20 * time.Millisecond))
	timing.pixelsReady(start.Add(50 * time.Millisecond))
	timing.pixelsReady(start.Add(60 * time.Millisecond))
	timing.moved(start.Add(80 * time.Millisecond))
	timing.pixelsReady(start.Add(100 * time.Millisecond))
	if metricValue(t, motion+"_count")-beforeMotion != 2 ||
		metricValue(t, interval+"_count")-beforeInterval != 1 ||
		metricValue(t, coalesced)-beforeCoalesced != 1 {
		t.Fatal("motion was not coalesced into two accepted frames")
	}
	if delta := metricValue(t, motion+"_sum") - beforeSum; delta < .04999 || delta > .05001 {
		t.Fatalf("motion used a superseded timestamp: %g seconds", delta)
	}
}

func TestWorkspaceMetricsRecordInitializationAndTick(t *testing.T) {
	t.Setenv("SLAYER_METRICS_ADDR", "127.0.0.1:0")
	t.Setenv("SLAYER_PPROF_ADDR", "")
	stop, err := observability.Start("ferruccio")
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	beforeInit := metricValue(t, "ferruccio_workspace_init_seconds_count")
	beforeTick := metricValue(t, "ferruccio_ui_tick_seconds_count")
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{SuppressAutomaticPreview: true})
	if err != nil {
		t.Fatal(err)
	}
	work.Tick()
	if metricValue(t, "ferruccio_workspace_init_seconds_count")-beforeInit != 1 || metricValue(t, "ferruccio_ui_tick_seconds_count")-beforeTick != 1 {
		t.Fatal("workspace did not publish initialization and tick timings")
	}
}
