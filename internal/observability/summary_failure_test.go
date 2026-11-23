package observability

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
)

// waitsForEver is a write that never comes back.
func waitsForEver([]byte) (int, error) { select {} }

func TestSummaryFailureNeverCallsArbitraryBlockedWriter(t *testing.T) {
	blocked := mocks.NewMockWriter(gomock.NewController(t))
	blocked.EXPECT().Write(gomock.Any()).DoAndReturn(waitsForEver).AnyTimes()
	before := SummaryFailures()
	start := time.Now()
	NoteSummaryFailure(blocked, "failed\n")
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
