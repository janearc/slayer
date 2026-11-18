package preview

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

func TestQueueMalformedLedgerAndMissingOutputCannotAcceptPixels(t *testing.T) {
	for _, variant := range []string{"malformed", "version", "missing job", "duplicate job", "unknown state", "mismatched terminal result", "missing PNG", "bad PNG", "status unavailable"} {
		t.Run(variant, func(t *testing.T) {
			plan, tools := queuedPlan(t)
			runner := NewMockRunner(gomock.NewController(t))
			var request queueRequest
			submit := expectSubmit(t, runner, tools, plan, &request, nil)
			expectStatus(t, runner, tools, func() ([]byte, error) {
				job := queueJob{Sequence: 41, Request: request, State: "succeeded", Result: queueResult{State: "succeeded"}}
				switch variant {
				case "malformed":
					return []byte("{"), nil
				case "version":
					return queueJSON(t, map[string]any{"version": 2, "jobs": []queueJob{job}}), nil
				case "missing job":
					return queueJSON(t, map[string]any{"version": 1, "jobs": []queueJob{}}), nil
				case "duplicate job":
					return queueJSON(t, map[string]any{"version": 1, "jobs": []queueJob{job, job}}), nil
				case "unknown state":
					job.State = "finished"
				case "mismatched terminal result":
					job.Result.State = "failed"
				case "bad PNG":
					if err := os.WriteFile(plan.Output, []byte("not a PNG"), 0600); err != nil {
						t.Fatal(err)
					}
				case "status unavailable":
					return nil, errors.New("queue status unavailable")
				}
				return queueJSON(t, map[string]any{"version": 1, "jobs": []queueJob{job}}), nil
			}).After(submit)
			var last Update
			err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, func(update Update) { last = update })
			if err == nil {
				t.Fatal("invalid ledger/output accepted")
			}
			if _, err := os.Stat(filepath.Join(plan.Directory, "receipt.json")); !os.IsNotExist(err) {
				t.Fatal("invalid result received image receipt")
			}
			if err := plan.Close(); err != nil {
				t.Fatal(err)
			}
			_, inputError := os.Stat(plan.Input)
			if variant == "missing PNG" || variant == "bad PNG" {
				if !last.Terminal || !os.IsNotExist(inputError) {
					t.Fatal("verified completed process did not permit input cleanup")
				}
			} else if inputError != nil {
				t.Fatal("unconfirmed ledger input removed", inputError)
			}
		})
	}
}

func TestQueueValidationRejectsUnsafeToolsAndChangedSnapshotBeforeSubprocess(t *testing.T) {
	for _, variant := range []string{"relative queue", "relative producer", "invalid agent", "changed snapshot"} {
		t.Run(variant, func(t *testing.T) {
			plan, tools := queuedPlan(t)
			switch variant {
			case "relative queue":
				tools.Queue = "relative"
			case "relative producer":
				tools.Slayer = "slayer"
			case "invalid agent":
				tools.Agent = "has spaces"
			case "changed snapshot":
				if err := os.WriteFile(plan.Input, []byte("scene: {name: changed}"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			runner := NewMockRunner(gomock.NewController(t))
			if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err == nil {
				t.Fatal("invalid metadata reached queue")
			}
		})
	}
}

func TestQueueCorruptCleanupEvidenceRetainsInput(t *testing.T) {
	plan, tools := queuedPlan(t)
	if _, err := plan.freezeRequest(tools); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.Directory, terminalName), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); err != nil {
		t.Fatal("corrupt terminal evidence deleted input", err)
	}
	if err := os.WriteFile(filepath.Join(plan.Directory, requestName), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); err != nil {
		t.Fatal("corrupt request evidence deleted input", err)
	}
}

func TestQueuePollingWaitIsBoundedAndInterruptible(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (queueClient{interval: time.Hour}).wait(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatal("poll wait ignored cancellation", err)
	}
	if err := (queueClient{interval: time.Nanosecond}).wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := (queueClient{}).wait(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatal("immediate poll ignored cancellation")
	}
}

func TestInterruptedUpdateNeverAuthorizesCleanup(t *testing.T) {
	job := queueJob{Sequence: 41, State: "interrupted", Result: queueResult{State: "interrupted", ExitCode: -1}}
	update := (observedJob{job: job}).update()
	if update.State != "interrupted" || update.Terminal {
		t.Fatal("interrupted owner state invented verified quiescence", update)
	}
}

func TestPreparedQueueRequestDoesNotExposeInheritedEnvironmentInDiagnostics(t *testing.T) {
	call := invocation{Executable: "/tools/azazel", Arguments: []string{"status", "--queue", "/queue/team"}, Environment: []string{"PRIVATE_VALUE=do-not-log"}}
	text := call.String()
	if strings.Contains(text, "PRIVATE_VALUE") || strings.Contains(text, "do-not-log") || !strings.Contains(text, "status") {
		t.Fatal("control-call diagnostics disclosed inherited environment")
	}
}

func TestOwnerTerminalResultRemainsAuthoritativeWithHistoricalExecutingMarker(t *testing.T) {
	for _, state := range []string{"succeeded", "failed", "cancelled"} {
		t.Run(state, func(t *testing.T) {
			request := queueRequest{Agent: "preview-owner", Key: "identity", Command: "/producer", Args: []string{"draw"}, Directory: "/inputs", Inputs: []string{"/inputs/source.yaml"}, Lane: "tests", TimeoutSeconds: 300}
			for _, uncertain := range []bool{false, true} {
				original := queueJob{Sequence: 3, Request: request, State: state, Executing: true, Result: queueResult{State: state, Uncertain: uncertain}}
				job, err := decodeJob(queueJSON(t, original), request, 3)
				if uncertain {
					if err == nil {
						t.Fatal("uncertain result became terminal authority")
					}
				} else if err != nil || !terminalJob(job) || !(observedJob{job: job}).update().Terminal {
					t.Fatal("owner-confirmed terminal state rejected due historical execution marker", err)
				}
			}
		})
	}
}
