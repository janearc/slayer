package queue

import (
	"slices"
	"testing"
)

func TestWorkerListenersAreNotInheritedByRenderer(t *testing.T) {
	t.Setenv("SLAYER_METRICS_ADDR", "127.0.0.1:7000")
	t.Setenv("SLAYER_PPROF_ADDR", "127.0.0.1:7001")
	env := environment(Request{})
	if slices.Contains(env, "SLAYER_METRICS_ADDR=127.0.0.1:7000") || slices.Contains(env, "SLAYER_PPROF_ADDR=127.0.0.1:7001") {
		t.Fatal("worker listener leaked into renderer", env)
	}
	env = environment(Request{Environment: map[string]string{"SLAYER_METRICS_ADDR": "127.0.0.1:8000", "SLAYER_PPROF_ADDR": "127.0.0.1:8001"}})
	if !slices.Contains(env, "SLAYER_METRICS_ADDR=127.0.0.1:8000") || !slices.Contains(env, "SLAYER_PPROF_ADDR=127.0.0.1:8001") {
		t.Fatal("explicit renderer listeners were lost", env)
	}
}
