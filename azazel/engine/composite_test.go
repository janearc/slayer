package engine

import (
	"context"
	"errors"
	"testing"

	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
)

// never is the stop of a composite that does not stop: no context at all,
// which Composite refuses.
var never context.Context

func compositeRequest() CompositeRequest {
	return CompositeRequest{Config: compositor.Config{From: 2, To: 4}, Estimate: CompositeEstimate{MemoryBytes: 1024, DiskBytes: 2048, CPUWorkers: 1}}
}

func TestCompositeRefusesWithoutSharedAdmission(t *testing.T) {
	called := false
	err := Composite(context.Background(), compositeRequest(), func(CompositeFrame) error { called = true; return nil })
	if !errors.Is(err, ErrCompositeAdmissionUnavailable) || called {
		t.Fatalf("got %v, consumer called %v", err, called)
	}
}

func TestCompositeRejectsInvalidRequestsBeforeAdmission(t *testing.T) {
	for _, mutate := range []func(*CompositeRequest){
		func(r *CompositeRequest) { r.Estimate.MemoryBytes = 0 },
		func(r *CompositeRequest) { r.Estimate.DiskBytes = -1 },
		func(r *CompositeRequest) { r.Estimate.CPUWorkers = 0 },
		func(r *CompositeRequest) { r.Config.From = -1 },
		func(r *CompositeRequest) { r.Config.To = r.Config.From },
	} {
		r := compositeRequest()
		mutate(&r)
		err := Composite(context.Background(), r, func(CompositeFrame) error { t.Fatal("consumer called"); return nil })
		if err == nil || errors.Is(err, ErrCompositeAdmissionUnavailable) {
			t.Fatalf("invalid request: %v", err)
		}
	}
	if err := Composite(never, compositeRequest(), func(CompositeFrame) error { return nil }); err == nil {
		t.Fatal("nil context accepted")
	}
	if err := Composite(context.Background(), compositeRequest(), nil); err == nil {
		t.Fatal("nil consumer accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Composite(ctx, compositeRequest(), func(CompositeFrame) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestCompositeLibraryBindingRejectsForgedExecution(t *testing.T) {
	r := compositeRequest()
	_, err := newCompositeExecution(context.Background(), r)
	if !errors.Is(err, compositor.ErrUnboundExecutionSnapshot) {
		t.Fatalf("real execution binding: %v", err)
	}
}

type compositeTestExecution struct {
	prepared []int
	key      identity.Key
	failure  error
	rendered int
}
type compositeTestFrame struct {
	execution *compositeTestExecution
	index     int
}

func (e *compositeTestExecution) prepare(ctx context.Context, index int) (compositePreparedFrame, error) {
	e.prepared = append(e.prepared, index)
	if e.failure != nil {
		return nil, e.failure
	}
	return compositeTestFrame{e, index}, nil
}
func (f compositeTestFrame) Key() identity.Key { return f.execution.key }
func (f compositeTestFrame) Render(ctx context.Context, consume func(compositor.Result) error) error {
	f.execution.rendered++
	return consume(compositor.Result{Index: f.index})
}
func compositeTestKey(t *testing.T) identity.Key {
	t.Helper()
	key, err := identity.BuildPrehashed(identity.PrehashedSpec{BinarySHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Settings: []byte("test"), FPS: 24, Frame: 2, Backend: "cpu", Device: "cpu/test", Width: 1, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	return key
}
func TestCompositeRangeIdentityAndConsumerErrors(t *testing.T) {
	key := compositeTestKey(t)
	r := compositeRequest()
	r.Identity.Frames = map[int]identity.Key{2: key, 3: key}
	e := &compositeTestExecution{key: key}
	var indices []int
	err := runComposite(context.Background(), r, e, func(f CompositeFrame) error {
		if f.Key != key {
			t.Fatal("identity lost")
		}
		indices = append(indices, f.Result.Index)
		return nil
	})
	if err != nil || len(indices) != 2 || indices[0] != 2 || indices[1] != 3 || e.rendered != 2 {
		t.Fatalf("range: %v %v %+v", err, indices, e)
	}
	sentinel := errors.New("consumer failed")
	err = runComposite(context.Background(), r, e, func(CompositeFrame) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("consumer error: %v", err)
	}
	e.failure = sentinel
	err = runComposite(context.Background(), r, e, func(CompositeFrame) error { t.Fatal("consumer called"); return nil })
	if !errors.Is(err, sentinel) {
		t.Fatalf("preparation error: %v", err)
	}
}
func TestCompositeIdentityMismatchNeverRenders(t *testing.T) {
	for _, keys := range []map[int]identity.Key{nil, {2: {}}} {
		r := compositeRequest()
		r.Identity.Frames = keys
		e := &compositeTestExecution{key: compositeTestKey(t)}
		err := runComposite(context.Background(), r, e, func(CompositeFrame) error { t.Fatal("consumer called"); return nil })
		if !errors.Is(err, ErrCompositeIdentityMismatch) || e.rendered != 0 {
			t.Fatalf("identity: %v, rendered %d", err, e.rendered)
		}
	}
}
func TestCompositeCanceledExecutionDoesNotPrepare(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := &compositeTestExecution{}
	err := runComposite(ctx, compositeRequest(), e, func(CompositeFrame) error { t.Fatal("consumer called"); return nil })
	if !errors.Is(err, context.Canceled) || len(e.prepared) != 0 {
		t.Fatalf("got %v, prepared %v", err, e.prepared)
	}
}
