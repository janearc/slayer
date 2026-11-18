package observability

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type blockedLog struct{}

func (blockedLog) Write([]byte) (int, error) { select {} }

func TestSummaryFailureNeverCallsArbitraryBlockedWriter(t *testing.T) {
	before := SummaryFailures()
	start := time.Now()
	NoteSummaryFailure(blockedLog{}, "failed\n")
	if time.Since(start) > time.Second || SummaryFailures() != before+1 {
		t.Fatal("failure logging blocked or was not counted")
	}
}

func TestSummaryFailureReachesNormalFileWithoutMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reporter.log")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	NoteSummaryFailure(file, "optional reporter unavailable\n")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil && string(raw) == "optional reporter unavailable\n" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("warning did not reach file")
}
