package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/azazel/internal/queue"
)

func TestWorkerServesPrometheusQueueMetrics(t *testing.T) {
	queueRoot := filepath.Join(t.TempDir(), "queue")
	if err := queue.At(queueRoot).Init(t.Context(), queue.Config{}); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	t.Setenv("SLAYER_METRICS_ADDR", address)
	t.Setenv("SLAYER_PPROF_ADDR", "")
	t.Setenv("AZAZEL_STATS_BARKEEP", "")
	t.Setenv("AZAZEL_STATS_VENDETTA_CLIENT", "")
	t.Setenv("AZAZEL_STATS_VENDETTA_ROOM", "")
	t.Setenv("AZAZEL_STATS_VENDETTA_AS", "")
	ctx, cancel := context.WithCancel(t.Context())
	var out, wrong bytes.Buffer
	done := make(chan int, 1)
	go func() { done <- queueTool(ctx).Run([]string{"work", "--serve", "--queue", queueRoot}, &out, &wrong) }()
	defer func() { cancel(); <-done }()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://" + address + "/metrics")
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			text := string(body)
			if response.StatusCode == 200 && strings.Contains(text, "azazel_queue_snapshot_available 1") && strings.Contains(text, "azazel_jobs_queued 0") && strings.Contains(text, "azazel_host_swap_available ") {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("worker did not serve queue and host metrics")
}
