package preview

import (
	"strings"
	"testing"
)

func TestPreviewChildrenDoNotInheritFrontendListenerAddresses(t *testing.T) {
	t.Setenv("SLAYER_METRICS_ADDR", "127.0.0.1:9466")
	t.Setenv("SLAYER_PPROF_ADDR", "127.0.0.1:6066")
	t.Setenv("SLAYER_RENDER_TEST_VALUE", "kept")
	for _, env := range [][]string{ChildEnvironment(), environment(Tools{})} {
		joined := "\n" + strings.Join(env, "\n") + "\n"
		for _, name := range []string{"SLAYER_METRICS_ADDR", "SLAYER_PPROF_ADDR"} {
			if strings.Contains(joined, "\n"+name+"=") {
				t.Fatalf("child inherited %s", name)
			}
		}
		if !strings.Contains(joined, "\nSLAYER_RENDER_TEST_VALUE=kept\n") {
			t.Fatal("child lost unrelated Slayer configuration")
		}
	}
	queued := queueEnvironment(Tools{})
	if _, ok := queued["SLAYER_METRICS_ADDR"]; ok {
		t.Fatal("queue request inherited metrics listener")
	}
	if _, ok := queued["SLAYER_PPROF_ADDR"]; ok {
		t.Fatal("queue request inherited profiling listener")
	}
	if queued["SLAYER_RENDER_TEST_VALUE"] != "kept" {
		t.Fatal("queue request lost unrelated Slayer configuration")
	}
}
