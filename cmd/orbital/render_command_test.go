package main

import (
	"strings"
	"testing"
)

func TestOrbitalRejectsBadTerminalRequestsBeforeBuildingAWorld(t *testing.T) {
	out, err := runCoverageCommand(t, "-cells", "bad")
	if err == nil || !strings.Contains(string(out), "orbital:") {
		t.Fatalf("%v %q", err, out)
	}
}
