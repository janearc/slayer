package preview

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// resolveFixture freezes a request for a plan and returns it with a mock queue.
func resolveFixture(t *testing.T) (*Plan, Tools, queueRequest, *MockRunner) {
	t.Helper()
	plan, tools := queuedPlan(t)
	request, err := plan.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	return plan, tools, request, NewMockRunner(gomock.NewController(t))
}

// TestResolveLiftsWhenTheQueueSaysTheJobEnded checks both ways a job can be
// over: a confirmed terminal state, and no such job in the ledger.
func TestResolveLiftsWhenTheQueueSaysTheJobEnded(t *testing.T) {
	for _, variant := range []string{"terminal", "never there"} {
		t.Run(variant, func(t *testing.T) {
			plan, tools, request, runner := resolveFixture(t)
			expectStatus(t, runner, tools, func() ([]byte, error) {
				if variant == "never there" {
					return queueJSON(t, map[string]any{"version": 1, "jobs": []queueJob{}}), nil
				}
				return jobSnapshot(t, request, "cancelled", "", false), nil
			})
			_, ended, err := (queueClient{runner: runner}).resolve(context.Background(), tools, plan, 41)
			if err != nil || !ended {
				t.Fatalf("ended=%t err=%v; want the job ended", ended, err)
			}
		})
	}
}

// TestResolveCancelsAgainWhileTheJobIsLive is the anti-test: a live job is
// not resolved, and its cancellation is asked for again.
func TestResolveCancelsAgainWhileTheJobIsLive(t *testing.T) {
	plan, tools, request, runner := resolveFixture(t)
	status := expectStatus(t, runner, tools, func() ([]byte, error) { return jobSnapshot(t, request, "running", "", true), nil })
	cancelled := false
	runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		cancelled = len(call.Arguments) > 0 && call.Arguments[0] == "cancel"
		return nil, nil
	}).After(status)
	update, ended, err := (queueClient{runner: runner}).resolve(context.Background(), tools, plan, 41)
	if err != nil || ended || !cancelled || update.Sequence != 41 {
		t.Fatalf("ended=%t cancelled=%t update=%+v err=%v; want live and cancel asked again", ended, cancelled, update, err)
	}
}

// TestResolveKeepsAnUnreadableReplyUnresolved is the second anti-test: a
// failed status read neither lifts the refusal nor cancels anything.
func TestResolveKeepsAnUnreadableReplyUnresolved(t *testing.T) {
	plan, tools, _, runner := resolveFixture(t)
	expectStatus(t, runner, tools, func() ([]byte, error) { return nil, errors.New("status unavailable") })
	_, ended, err := (queueClient{runner: runner}).resolve(context.Background(), tools, plan, 41)
	if err == nil || ended {
		t.Fatalf("ended=%t err=%v; want unresolved with an error", ended, err)
	}
}
