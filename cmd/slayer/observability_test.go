package main

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestObservabilityListenerFailureDoesNotRefuseTheCommand(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("SLAYER_METRICS_ADDR", listener.Addr().String())
	t.Setenv("SLAYER_PPROF_ADDR", "")
	var output bytes.Buffer
	stop := startObservability(&output)
	stop()
	if !strings.Contains(output.String(), "observability unavailable") {
		t.Fatalf("listener failure was not reported: %q", output.String())
	}
}
