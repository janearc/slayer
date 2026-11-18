package door

import (
	"context"
	"image"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/observability"
	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
)

func metric(t *testing.T, name string) float64 {
	t.Helper()
	response := httptest.NewRecorder()
	observability.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	for _, line := range strings.Split(response.Body.String(), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == name {
			value, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
	}
	return 0
}

func TestFrameMetricsRecordDistinctPipelineStages(t *testing.T) {
	t.Setenv("SLAYER_METRICS_ADDR", "127.0.0.1:0")
	closeMetrics, err := observability.Start("slayer")
	if err != nil {
		t.Fatal(err)
	}
	defer closeMetrics()
	names := []string{"slayer_frame_draw_seconds_sum", "slayer_frame_develop_seconds_sum", "slayer_frame_titles_seconds_sum", "slayer_frames_drawn_total"}
	before := make([]float64, len(names))
	for i, name := range names {
		before[i] = metric(t, name)
	}
	control := gomock.NewController(t)
	source := NewMocksource(control)
	developer := NewMockdeveloper(control)
	drawn := NewMockdrawn(control)
	clock := &stepping{at: time.Unix(0, 0)}
	light := &frame.Frame{}
	picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	source.EXPECT().Frame(gomock.Any(), 0).Return(drawn, nil).Do(func(context.Context, int) { clock.on(1) })
	drawn.EXPECT().Light().Return(light).AnyTimes()
	developer.EXPECT().Develop(0, []*frame.Frame{light}).Return(picture, nil).Do(func(int, []*frame.Frame) { clock.on(2) })
	drawn.EXPECT().Overlay(picture).Return(nil).Do(func(*image.RGBA) { clock.on(3) })
	run := runner{ctx: context.Background(), now: clock.now}
	if _, err := run.frame(kit{source: source, developer: developer}, 0, 0); err != nil {
		t.Fatal(err)
	}
	for i, want := range []float64{1, 2, 3, 1} {
		if got := metric(t, names[i]) - before[i]; got != want {
			t.Errorf("%s advanced by %v, want %v", names[i], got, want)
		}
	}
}
