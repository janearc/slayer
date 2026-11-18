//go:build netcdf

package main

import (
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestViewerRunsWhenObservabilityPortIsUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("SLAYER_METRICS_ADDR", listener.Addr().String())
	t.Setenv("SLAYER_PPROF_ADDR", "")
	out, err := viewerRun(t, "--help")
	if err != nil || !strings.Contains(string(out), "grissom observability:") || !strings.Contains(string(out), "--earth-depth") {
		t.Fatalf("help with occupied metrics listener: %v; %s", err, out)
	}
}

func TestFilmMetricsCountOnlyFramesActuallyWritten(t *testing.T) {
	t.Setenv("SLAYER_METRICS_ADDR", "127.0.0.1:0")
	t.Setenv("SLAYER_PPROF_ADDR", "")
	stop, err := observability.Start("grissom")
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	beforeCount := metricValue(t, "grissom_film_frames_total")
	beforeSamples := metricValue(t, "grissom_film_frame_seconds_count")
	var paths []string
	err = drawFilm(options{into: t.TempDir(), frames: 3}, func(path string) { paths = append(paths, path) })
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 || metricValue(t, "grissom_film_frames_total")-beforeCount != 3 || metricValue(t, "grissom_film_frame_seconds_count")-beforeSamples != 3 {
		t.Fatalf("film did not publish one counter and duration per written frame: %v", paths)
	}
}
