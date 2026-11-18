package preview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Update is one immutable observation of this plan's external queue job.
// Sequence is a durable queue job identity; explicit direct transport uses zero.
type Update struct {
	State, Held string
	Sequence    uint64
	// Terminal confirms the process outcome is a cleanup authority; interrupted is false.
	Terminal bool
}

const (
	// pollInterval keeps control traffic bounded while displaying admission waits.
	pollInterval = 500 * time.Millisecond
	// cancellationWindow bounds recovery after the caller's context has ended.
	cancellationWindow = 5 * time.Second
)

// RunObserved uses Azazel by default; direct shared Slayer requires explicit selection.
func RunObserved(ctx context.Context, tools Tools, plan *Plan, output io.Writer, notify func(Update)) error {
	switch tools.Transport {
	case "", "azazel":
		return (queueClient{runner: processRunner{}, interval: pollInterval}).run(ctx, tools, plan, output, notify)
	case "slayer":
		return runDirect(ctx, tools, plan, output, notify)
	default:
		return fmt.Errorf("unknown preview transport %q", tools.Transport)
	}
}

// queueClient owns only short-lived Azazel subprocesses, never a queue worker.
type queueClient struct {
	runner   Runner
	interval time.Duration
}

// run binds one prepared plan to one durable external request.
func (client queueClient) run(ctx context.Context, tools Tools, plan *Plan, output io.Writer, notify func(Update)) error {
	if ctx == nil || plan == nil || plan.state == nil || client.runner == nil {
		return fmt.Errorf("prepare a preview and configure its queue client")
	}
	plan.state.mu.Lock()
	if plan.state.active {
		plan.state.mu.Unlock()
		return fmt.Errorf("this preview is already being observed")
	}
	plan.state.active = true
	plan.state.mu.Unlock()
	defer func() { plan.state.mu.Lock(); plan.state.active = false; plan.state.mu.Unlock() }()
	if err := ctx.Err(); err != nil {
		return err
	}
	request, err := plan.freezeRequest(tools)
	if err != nil {
		return err
	}
	raw, submitError := client.invoke(ctx, tools, plan, output, "submit", "--queue", tools.Queue, filepath.Join(plan.Directory, requestName))
	job, ackError := decodeJob(raw, request, 0)
	if submitError != nil || ackError != nil {
		recovered, recoveryError := client.recoverAcknowledgement(tools, plan, request, output)
		if recoveryError != nil {
			return fmt.Errorf("Azazel acknowledgement uncertain; frozen input retained: %w", errors.Join(submitError, ackError, recoveryError))
		}
		job, raw = recovered.job, recovered.raw
	}
	if err := os.WriteFile(filepath.Join(plan.Directory, "azazel-acknowledgement.json"), raw, 0600); err != nil {
		return err
	}
	if notify != nil {
		notify((observedJob{job: job}).update())
	}
	return client.observe(ctx, tools, plan, request, job.Sequence, output, notify)
}

// invoke runs only the configured external Azazel control executable.
func (client queueClient) invoke(ctx context.Context, tools Tools, plan *Plan, output io.Writer, args ...string) ([]byte, error) {
	if output == nil {
		output = io.Discard
	}
	return client.runner.Run(ctx, invocation{Executable: tools.Azazel, Arguments: args, Directory: filepath.Dir(plan.Input), Environment: environment(tools), Output: output})
}

// recoverAcknowledgement resolves a lost response by exact agent/key/content.
func (client queueClient) recoverAcknowledgement(tools Tools, plan *Plan, request queueRequest, output io.Writer) (observedJob, error) {
	fresh, cancel := context.WithTimeout(context.Background(), cancellationWindow)
	defer cancel()
	raw, err := client.invoke(fresh, tools, plan, output, "status", "--queue", tools.Queue)
	if err != nil {
		return observedJob{}, err
	}
	return readObservation(raw, request, 0)
}

// observe accepts pixels only after this plan's verified job succeeds.
func (client queueClient) observe(ctx context.Context, tools Tools, plan *Plan, request queueRequest, sequence uint64, output io.Writer, notify func(Update)) error {
	for {
		if err := ctx.Err(); err != nil {
			return client.cancelOwned(tools, plan, request, sequence, output, notify, err)
		}
		raw, err := client.invoke(ctx, tools, plan, output, "status", "--queue", tools.Queue)
		if err != nil {
			if ctx.Err() != nil {
				return client.cancelOwned(tools, plan, request, sequence, output, notify, ctx.Err())
			}
			return fmt.Errorf("Azazel status unavailable; frozen input retained: %w", err)
		}
		observation, err := readObservation(raw, request, sequence)
		if err != nil {
			return fmt.Errorf("Azazel status unconfirmed; frozen input retained: %w", err)
		}
		if notify != nil {
			notify(observation.update())
		}
		if observation.job.State == "interrupted" {
			return fmt.Errorf("%w; frozen input retained pending confirmed descendant quiescence", jobFailure(tools, observation.job))
		}
		if terminalJob(observation.job) {
			if err := plan.retainTerminal(observation); err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if observation.job.State != "succeeded" || observation.job.Result.ExitCode != 0 {
				return jobFailure(tools, observation.job)
			}
			if err := plan.record(tools); err != nil {
				return err
			}
			return plan.recordQueueIdentity(tools, observation.job)
		}
		if err := client.wait(ctx); err != nil {
			return client.cancelOwned(tools, plan, request, sequence, output, notify, err)
		}
	}
}

// cancelOwned uses a fresh bounded context and confirms the same job terminal.
func (client queueClient) cancelOwned(tools Tools, plan *Plan, request queueRequest, sequence uint64, output io.Writer, notify func(Update), cause error) error {
	fresh, cancel := context.WithTimeout(context.Background(), cancellationWindow)
	defer cancel()
	_, cancelError := client.invoke(fresh, tools, plan, output, "cancel", "--queue", tools.Queue, "--agent", request.Agent, strconv.FormatUint(sequence, 10))
	for {
		raw, err := client.invoke(fresh, tools, plan, output, "status", "--queue", tools.Queue)
		if err != nil {
			return errors.Join(cause, cancelError, fmt.Errorf("cancellation unconfirmed; frozen input retained: %w", err))
		}
		observation, err := readObservation(raw, request, sequence)
		if err != nil {
			return errors.Join(cause, cancelError, fmt.Errorf("cancellation unconfirmed; frozen input retained: %w", err))
		}
		if notify != nil {
			notify(observation.update())
		}
		if observation.job.State == "interrupted" {
			return errors.Join(cause, fmt.Errorf("%w; frozen input retained pending confirmed descendant quiescence", jobFailure(tools, observation.job)))
		}
		if terminalJob(observation.job) {
			return errors.Join(cause, plan.retainTerminal(observation))
		}
		if err := client.wait(fresh); err != nil {
			return errors.Join(cause, cancelError, fmt.Errorf("cancellation unconfirmed; frozen input retained: %w", err))
		}
	}
}

// wait makes polling interruptible without leaving a sleeping worker behind.
func (client queueClient) wait(ctx context.Context) error {
	if client.interval <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(client.interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retainTerminal keeps the exact observed owner job as the cleanup authority.
func (plan *Plan) retainTerminal(observation observedJob) error {
	return os.WriteFile(filepath.Join(plan.Directory, terminalName), observation.raw, 0600)
}
