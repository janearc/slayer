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
}

// stageResult transfers ownership of the staged view to its receiver.
type stageResult struct {
	stageRequest
	view *stagedView
	err  error
}

type stageBuild func(context.Context, string, string, string) (*stagedView, error)

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
}

// newStageLoader creates an idle loader around build.
func newStageLoader(build stageBuild) *stageLoader { return &stageLoader{build: build} }

// request invalidates any old result and replaces a waiting request.
func (loader *stageLoader) request(path, source, kind string) {
	loader.requestBuild(path, source, kind, loader.build)
}

// requestBuild captures the configuration belonging to this particular text.
func (loader *stageLoader) requestBuild(path, source, kind string, build stageBuild) {
	loader.mu.Lock()
	if loader.closed {
		loader.mu.Unlock()
		return
	}
	loader.generation++
	loader.pending = &stageRequest{path: path, source: source, kind: kind, generation: loader.generation, build: build}
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
	loader.workers.Add(1)
	go loader.run(ctx, request)
}

// run disposes a superseded result before starting the newest request.
func (loader *stageLoader) run(ctx context.Context, request stageRequest) {
	defer loader.workers.Done()
	view, err := request.build(ctx, request.path, request.source, request.kind)
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
	loader.mu.Unlock()
	if ready && loader.notify != nil {
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

// close cancels active work and releases any result not yet taken.
func (loader *stageLoader) close() {
	loader.mu.Lock()
	if loader.closed {
		loader.mu.Unlock()
		return
	}
	loader.closed = true
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
}

// wait joins workers after close has prevented any new requests.
func (loader *stageLoader) wait() { loader.workers.Wait() }
