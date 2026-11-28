package workspace

import (
	"context"
	"sync"
)

// stageRequest identifies the exact buffer and location a worker sees.
type stageRequest struct {
	path, source, kind string
	generation         uint64
	build              stageBuild
	prepare            stagePrepare
}

// stageResult transfers ownership of the staged view to its receiver.
type stageResult struct {
	stageRequest
	view *stagedView
	err  error
}

type stageBuild func(context.Context, string, string, string) (*stagedView, error)

// stagePrepare may supply a lightweight scene without invoking the helper.
type stagePrepare func(context.Context, string, string, string) (*stagedView, error)

// stageLoader runs at most one build and keeps only the latest waiting request.
type stageLoader struct {
	mu             sync.Mutex
	workers        sync.WaitGroup
	build          stageBuild
	notify         func()
	generation     uint64
	pending        *stageRequest
	ready          *stageResult
	cancel         context.CancelFunc
	active, closed bool
	preparing      bool
}

// newStageLoader creates an idle loader around build.
func newStageLoader(build stageBuild) *stageLoader { return &stageLoader{build: build} }

// request invalidates any old result and replaces a waiting request.
func (loader *stageLoader) request(path, source, kind string) {
	loader.requestBuild(path, source, kind, loader.build)
}

// requestBuild captures the configuration belonging to this particular text.
func (loader *stageLoader) requestBuild(path, source, kind string, build stageBuild) {
	loader.requestPreparedBuild(path, source, kind, nil, build)
}

func (loader *stageLoader) requestPreparedBuild(path, source, kind string, prepare stagePrepare, build stageBuild) {
	loader.mu.Lock()
	if loader.closed {
		loader.mu.Unlock()
		return
	}
	loader.generation++
	loader.pending = &stageRequest{path: path, source: source, kind: kind, generation: loader.generation, build: build, prepare: prepare}
	stale := loader.ready
	loader.ready = nil
	if loader.cancel != nil {
		loader.cancel()
	}
	if stale != nil {
		// An untaken result only exists when no build is active. Retire it
		// before starting the next build, without blocking the caller.
		loader.active = true
		loader.workers.Add(1)
		go loader.retire(stale.view)
	} else {
		loader.startLocked()
	}
	loader.mu.Unlock()
}

// discard cancels obsolete work without starting another staged build.
func (loader *stageLoader) discard() {
	loader.mu.Lock()
	if loader.closed {
		loader.mu.Unlock()
		return
	}
	loader.generation++
	loader.pending = nil
	if loader.cancel != nil {
		loader.cancel()
	}
	stale := loader.ready
	loader.ready = nil
	if stale != nil {
		loader.active = true
		loader.workers.Add(1)
		go loader.retire(stale.view)
	}
	loader.mu.Unlock()
}

// retire releases an untaken result before allowing the next build.
func (loader *stageLoader) retire(view *stagedView) {
	defer loader.workers.Done()
	_ = view.close()
	loader.mu.Lock()
	loader.active = false
	loader.startLocked()
	loader.mu.Unlock()
}

// startLocked starts the next request while holding the loader's mutex.
func (loader *stageLoader) startLocked() {
	if loader.active || loader.closed || loader.pending == nil {
		return
	}
	request := *loader.pending
	loader.pending = nil
	ctx, cancel := context.WithCancel(context.Background())
	loader.cancel = cancel
	loader.active = true
	loader.preparing = request.prepare != nil
	loader.workers.Add(1)
	go loader.run(ctx, request)
}

// run disposes a superseded result before starting the newest request.
func (loader *stageLoader) run(ctx context.Context, request stageRequest) {
	defer loader.workers.Done()
	var view *stagedView
	var err error
	if request.prepare != nil {
		view, err = request.prepare(ctx, request.path, request.source, request.kind)
		loader.mu.Lock()
		loader.preparing = false
		if err == nil {
			err = ctx.Err()
		}
		loader.mu.Unlock()
	}
	if err == nil && view == nil {
		view, err = request.build(ctx, request.path, request.source, request.kind)
	}
	loader.mu.Lock()
	ready := false
	if loader.closed || request.generation != loader.generation {
		// Releasing a session can take time; do not hold the UI-facing lock.
		loader.mu.Unlock()
		_ = view.close()
		loader.mu.Lock()
	} else {
		loader.ready = &stageResult{stageRequest: request, view: view, err: err}
		ready = true
	}
	loader.active = false
	loader.cancel = nil
	loader.startLocked()
	if ready && loader.notify != nil {
		loader.notify()
	}
	loader.mu.Unlock()
}

// wakePreparing serializes the window wake with close and supersession.
func (loader *stageLoader) wakePreparing(ctx context.Context) {
	loader.mu.Lock()
	defer loader.mu.Unlock()
	if !loader.closed && ctx.Err() == nil && loader.notify != nil {
		loader.notify()
	}
}

// take transfers a completed latest result, without waiting for a build.
func (loader *stageLoader) take() (stageResult, bool) {
	loader.mu.Lock()
	defer loader.mu.Unlock()
	if loader.ready == nil || loader.closed {
		return stageResult{}, false
	}
	result := *loader.ready
	loader.ready = nil
	return result, true
}

// close cancels active work and releases a result not yet taken. A true
// result means only read-only preparation is active. It may finish later.
func (loader *stageLoader) close() bool {
	loader.mu.Lock()
	if loader.closed {
		loader.mu.Unlock()
		return false
	}
	loader.closed = true
	preparing := loader.preparing
	loader.pending = nil
	if loader.cancel != nil {
		loader.cancel()
	}
	ready := loader.ready
	loader.ready = nil
	if ready != nil {
		loader.workers.Add(1)
	}
	loader.mu.Unlock()
	if ready != nil {
		go func() { defer loader.workers.Done(); _ = ready.view.close() }()
	}
	return preparing
}

// wait joins workers after close has prevented any new requests.
func (loader *stageLoader) wait() { loader.workers.Wait() }
