package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/janearc/slayer/improv/moonlake/lights"
	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

var ErrCompositeAdmissionUnavailable = errors.New("composite: shared admission, resource reservations, locking and publication are unavailable; direct starts are refused")
var ErrCompositeIdentityMismatch = errors.New("composite: dependency identity does not match execution")

type CompositeEstimate struct {
	MemoryBytes int64
	DiskBytes   int64
	CPUWorkers  int
}

type CompositeIdentity struct {
	BinarySHA256 string
	Backend      string
	Device       string
	Score        []byte
	Cues         []byte
	Frames       map[int]identity.Key
}

type CompositeRequest struct {
	Settings settings.Settings
	Config   compositor.Config
	Lighting lights.Config
	Identity CompositeIdentity
	Estimate CompositeEstimate
}

type CompositeFrame struct {
	Key    identity.Key
	Result compositor.Result
}

// Composite checks the request and then always refuses it. It returns an
// error for a nil ctx or consume, a cancelled ctx, an empty or negative
// frame range, or an estimate that is not positive. Otherwise it returns
// ErrCompositeAdmissionUnavailable and renders nothing; consume is never
// called.
func Composite(ctx context.Context, request CompositeRequest, consume func(CompositeFrame) error) error {
	if ctx == nil {
		return errors.New("composite: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if consume == nil {
		return errors.New("composite: nil consumer")
	}
	if request.Config.From < 0 || request.Config.To <= request.Config.From {
		return errors.New("composite: expected nonnegative, nonempty half-open frame range")
	}
	if request.Estimate.MemoryBytes <= 0 || request.Estimate.DiskBytes <= 0 || request.Estimate.CPUWorkers <= 0 {
		return errors.New("composite: explicit positive memory, disk and CPU estimates required")
	}
	return ErrCompositeAdmissionUnavailable
}

type compositePreparedFrame interface {
	Key() identity.Key
	Render(context.Context, func(compositor.Result) error) error
}

type compositeExecution interface {
	prepare(context.Context, int) (compositePreparedFrame, error)
}

type compositeLibraryExecution struct {
	snapshot *compositor.ExecutionSnapshot
}

// newCompositeExecution builds an execution from the snapshot of the
// settings, configs, and identity that the request holds.
func newCompositeExecution(ctx context.Context, request CompositeRequest) (compositeExecution, error) {
	snapshot, err := compositor.NewExecutionSnapshot(ctx, request.Settings, request.Config, request.Lighting, request.Identity.Score, request.Identity.Cues, request.Identity.BinarySHA256, request.Identity.Backend, request.Identity.Device)
	if err != nil {
		return nil, fmt.Errorf("composite: execution: %w", err)
	}
	return compositeLibraryExecution{snapshot: snapshot}, nil
}

// prepare readies frame index from the snapshot.
func (e compositeLibraryExecution) prepare(ctx context.Context, index int) (compositePreparedFrame, error) {
	return e.snapshot.PrepareFrame(ctx, index)
}

// runComposite renders each frame from Config.From up to but not including
// Config.To through execution, handing each result to consume. It stops at
// the first error. A frame whose prepared key, or whose result index, is
// not the one the request names in Identity.Frames is refused.
func runComposite(ctx context.Context, request CompositeRequest, execution compositeExecution, consume func(CompositeFrame) error) error {
	for index := request.Config.From; index < request.Config.To; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		expected, ok := request.Identity.Frames[index]
		if !ok || expected.String() == "" {
			return fmt.Errorf("%w: frame %d missing", ErrCompositeIdentityMismatch, index)
		}
		prepared, err := execution.prepare(ctx, index)
		if err != nil {
			return fmt.Errorf("composite: frame %d: prepare: %w", index, err)
		}
		if prepared.Key() != expected {
			return fmt.Errorf("%w: frame %d", ErrCompositeIdentityMismatch, index)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		err = prepared.Render(ctx, func(result compositor.Result) error {
			if result.Index != index {
				return fmt.Errorf("%w: result frame %d, expected %d", ErrCompositeIdentityMismatch, result.Index, index)
			}
			return consume(CompositeFrame{Key: expected, Result: result})
		})
		if err != nil {
			return fmt.Errorf("composite: frame %d: render: %w", index, err)
		}
	}
	return nil
}
