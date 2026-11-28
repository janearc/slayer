package workingview

import (
	"math"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/observability"
)

func timingSample(t *testing.T, name string) float64 {
	t.Helper()
	report := httptest.NewRecorder()
	observability.Handler().ServeHTTP(report, httptest.NewRequest("GET", "/metrics", nil))
	for _, line := range strings.Split(report.Body.String(), "\n") {
		if value, found := strings.CutPrefix(line, name+" "); found {
			number, err := strconv.ParseFloat(value, 64)
			if err != nil {
				t.Fatal(err)
			}
			return number
		}
	}
	return 0
}

func TestDrawTimingSeparatesUploadsFromCulling(t *testing.T) {
	start := time.Now()
	cullBefore := timingSample(t, workingCullMetric+"_sum")
	uploadBefore := timingSample(t, workingUploadMetric+"_sum")
	timing := drawTiming{enabled: true, uploads: 20 * time.Millisecond}
	timing.culling(start, start.Add(50*time.Millisecond))
	timing.preparation(start, start.Add(10*time.Millisecond))
	for _, metric := range []struct {
		name   string
		before float64
	}{{workingCullMetric, cullBefore}, {workingUploadMetric, uploadBefore}} {
		if got := timingSample(t, metric.name+"_sum") - metric.before; math.Abs(got-.03) > 1e-8 {
			t.Errorf("%s recorded %g s, want .03 s", metric.name, got)
		}
	}
}

func TestDisabledDrawTimingPublishesNoSamples(t *testing.T) {
	timing := drawTiming{}
	start := timing.now()
	if !start.IsZero() {
		t.Fatal("disabled working metrics read a timestamp")
	}
	for _, name := range []string{workingCullMetric, workingUploadMetric, workingEncodeMetric, workingSubmitMetric} {
		before := timingSample(t, name+"_count")
		timing.culling(start, start)
		timing.preparation(start, start)
		observeDrawPhase(name, start)
		if timingSample(t, name+"_count") != before {
			t.Fatalf("disabled working metrics recorded %s", name)
		}
	}
}
