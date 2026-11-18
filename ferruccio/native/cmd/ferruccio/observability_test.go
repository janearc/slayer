package main

import (
	"net"
	"os"
	"strings"
	"testing"
)

func TestCommandRunsWhenObservabilityPortIsUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("SLAYER_METRICS_ADDR", listener.Addr().String())
	t.Setenv("SLAYER_PPROF_ADDR", "")
	old := os.Args
	os.Args = []string{"ferruccio", "open", "--dry-run"}
	defer func() { os.Args = old }()
	if code := execute(); code != 0 {
		t.Fatalf("dry-run returned %d after metrics listener failed", code)
	}
}

func TestRegressionChildDoesNotInheritFerruccioSummaryDestinations(t *testing.T) {
	t.Setenv("FERRUCCIO_STATS_VENDETTA_ROOM", "test-room")
	t.Setenv("FERRUCCIO_STATS_BARKEEP", "127.0.0.1:1234")
	t.Setenv("SLAYER_STATS_BARKEEP", "127.0.0.1:5678")
	joined := "\n" + strings.Join(regressionChildEnvironment(), "\n") + "\n"
	if strings.Contains(joined, "\nFERRUCCIO_STATS_") {
		t.Fatal("regression child inherited Ferruccio summary settings")
	}
	if !strings.Contains(joined, "\nSLAYER_STATS_BARKEEP=127.0.0.1:5678\n") {
		t.Fatal("regression child lost explicitly configured Slayer summary settings")
	}
}
