package preview

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

// expectGiveUp registers what follows observation being given up: the
// owned job is cancelled first, then its status is read once.
func expectGiveUp(t *testing.T, runner *MockRunner, tools Tools, reply func() ([]byte, error), after *gomock.Call) {
	t.Helper()
	cancelling := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		if len(call.Arguments) == 0 || call.Arguments[0] != "cancel" {
			t.Fatalf("expected the job to be cancelled first, got %+v", call.Arguments)
		}
		return nil, nil
	}).After(after)
	expectStatus(t, runner, tools, reply).After(cancelling)
}

// TestQueueOneBadStatusReadDoesNotEndObservation checks that failed,
// malformed and unmatched reads are retried and the job is then seen through.
func TestQueueOneBadStatusReadDoesNotEndObservation(t *testing.T) {
	for _, bad := range []string{"failed", "malformed", "unmatched"} {
		t.Run(bad, func(t *testing.T) {
			plan, tools := queuedPlan(t)
			runner := NewMockRunner(gomock.NewController(t))
			var request queueRequest
			submit := expectSubmit(t, runner, tools, plan, &request, nil)
			first := expectStatus(t, runner, tools, func() ([]byte, error) {
				switch bad {
				case "failed":
					return nil, errors.New("status unavailable")
				case "malformed":
					return []byte("{"), nil
				}
				other := request
				other.Agent = "someone-else"
				return jobSnapshot(t, other, "running", "", true), nil
			}).After(submit)
			expectStatus(t, runner, tools, func() ([]byte, error) {
				queuePNG(t, plan, 16, 12)
				return jobSnapshot(t, request, "succeeded", "", false), nil
			}).After(first)
			err := (queueClient{runner: runner, attempts: 3}).run(context.Background(), tools, plan, io.Discard, nil)
			if err != nil {
				t.Fatalf("one %s read ended observation: %v", bad, err)
			}
		})
	}
}

// TestQueueGivingUpCancelsTheJobFirst is the anti-test: when every read in
// the bound fails, observation ends, the job is cancelled first and the
// error says so.
func TestQueueGivingUpCancelsTheJobFirst(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := expectSubmit(t, runner, tools, plan, &request, nil)
	failing := func() ([]byte, error) { return nil, errors.New("status unavailable") }
	reads := expectStatus(t, runner, tools, failing).After(submit).Times(3)
	expectGiveUp(t, runner, tools, failing, reads)
	err := (queueClient{runner: runner, attempts: 3}).run(context.Background(), tools, plan, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "observation given up") || !strings.Contains(err.Error(), "cancelling job 41 first") {
		t.Fatalf("error = %v; want the give-up and the cancellation named", err)
	}
}
