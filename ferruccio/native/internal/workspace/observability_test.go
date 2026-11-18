package workspace

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

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
