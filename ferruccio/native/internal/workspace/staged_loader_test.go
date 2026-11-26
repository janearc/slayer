package workspace

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type testStageOwner struct{ closed *atomic.Int32 }

func (owner testStageOwner) Close() error { owner.closed.Add(1); return nil }

func TestStageLoaderCoalescesAndClosesStaleResults(t *testing.T) {
	started := make(chan string, 3)
	finishFirst := make(chan struct{})
	var firstClosed, lastClosed atomic.Int32
	loader := newStageLoader(func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		started <- source
		if source == "first" {
			<-finishFirst
			return &stagedView{owner: testStageOwner{&firstClosed}}, nil
		}
		return &stagedView{owner: testStageOwner{&lastClosed}}, nil
	})
	defer loader.close()
	loader.request("p", "first", "set")
	if got := <-started; got != "first" {
		t.Fatalf("started %q", got)
	}
	loader.request("p", "middle", "set")
	loader.request("p", "last", "set")
	close(finishFirst)
	if got := <-started; got != "last" {
		t.Fatalf("started stale %q", got)
	}
	deadline := time.After(3 * time.Second)
	for {
		if result, ok := loader.take(); ok {
			if result.source != "last" || result.err != nil {
				t.Fatalf("result = %+v", result)
			}
			if firstClosed.Load() != 1 {
				t.Fatal("stale result wasn't closed")
			}
			_ = result.view.close()
			if lastClosed.Load() != 1 {
				t.Fatal("accepted owner wasn't closed")
			}
			return
		}
		select {
		case <-deadline:
			t.Fatal("no latest result")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestStageLoaderCloseDiscardsLateResult(t *testing.T) {
	started := make(chan struct{})
	finish := make(chan struct{})
	closed := make(chan struct{})
	loader := newStageLoader(func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		close(started)
		<-finish
		return &stagedView{owner: signalStageOwner{closed}}, nil
	})
	loader.request("p", "first", "set")
	<-started
	loader.close()
	close(finish)
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("late result wasn't closed")
	}
	if _, ok := loader.take(); ok {
		t.Fatal("closed loader returned a result")
	}
}

func TestStageLoaderWaitsForClosedWorker(t *testing.T) {
	started := make(chan struct{})
	released := make(chan struct{})
	loader := newStageLoader(func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		close(started)
		<-ctx.Done()
		<-released
		return nil, ctx.Err()
	})
	loader.request("p", "source", "set")
	<-started
	loader.close()
	done := make(chan struct{})
	go func() { loader.wait(); close(done) }()
	select {
	case <-done:
		t.Fatal("wait returned before cancelled worker finished")
	default:
	}
	close(released)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("wait did not return after worker finished")
	}
}

func TestStageLoaderUsesRequestBuilderAndNotifies(t *testing.T) {
	notified := make(chan struct{}, 1)
	loader := newStageLoader(nil)
	loader.notify = func() { notified <- struct{}{} }
	defer func() { loader.close(); loader.wait() }()
	loader.requestBuild("p", "source", "set", func(_ context.Context, _, source, _ string) (*stagedView, error) {
		if source != "source" {
			t.Errorf("builder saw %q", source)
		}
		return &stagedView{notice: "ready"}, nil
	})
	select {
	case <-notified:
	case <-time.After(3 * time.Second):
		t.Fatal("completed worker did not wake the UI")
	}
	result, ok := loader.take()
	if !ok || result.view == nil || result.view.notice != "ready" {
		t.Fatalf("result = %+v, %v", result, ok)
	}
}

func TestStageLoaderDiscardRejectsOldResult(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var closed atomic.Int32
	loader := newStageLoader(func(_ context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-release
		return &stagedView{owner: testStageOwner{&closed}}, nil
	})
	loader.request("p", "old", "set")
	<-started
	loader.discard()
	close(release)
	loader.close()
	loader.wait()
	if _, ok := loader.take(); ok || closed.Load() != 1 {
		t.Fatal("discarded view was retained")
	}
}

type signalStageOwner struct{ closed chan struct{} }

func (owner signalStageOwner) Close() error { close(owner.closed); return nil }

func TestStageLoaderRetiresUntakenResultOffCaller(t *testing.T) {
	started := make(chan string, 2)
	retiring := make(chan struct{})
	finishClose := make(chan struct{})
	loader := newStageLoader(func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		started <- source
		if source == "first" {
			return &stagedView{owner: blockingStageOwner{retiring, finishClose}}, nil
		}
		return nil, nil
	})
	defer loader.close()
	loader.request("p", "first", "set")
	<-started
	deadline := time.After(3 * time.Second)
	for {
		loader.mu.Lock()
		ready := loader.ready != nil
		loader.mu.Unlock()
		if ready {
			break
		}
		select {
		case <-deadline:
			t.Fatal("first result did not finish")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	returned := make(chan struct{})
	go func() { loader.request("p", "last", "set"); close(returned) }()
	select {
	case <-returned:
	case <-time.After(3 * time.Second):
		t.Fatal("request blocked on closing old view")
	}
	<-retiring
	select {
	case got := <-started:
		t.Fatalf("started %q before retiring old view", got)
	default:
	}
	close(finishClose)
	select {
	case got := <-started:
		if got != "last" {
			t.Fatalf("started %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("latest build did not start after retirement")
	}
}

type blockingStageOwner struct{ started, finish chan struct{} }

func (owner blockingStageOwner) Close() error {
	close(owner.started)
	<-owner.finish
	return nil
}
