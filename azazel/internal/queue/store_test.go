package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// fixture creates an isolated queue and a foreground command request.
func fixture(t *testing.T) (Store, Request) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "queue")
	s := At(root)
	if err := s.Init(t.Context(), Config{Game: "/usr/bin/true"}); err != nil {
		t.Fatal(err)
	}
	r := Request{Agent: "studio", Key: "one", Command: "/usr/bin/true", Directory: t.TempDir(), Args: []string{"draw", "file with spaces.yaml", "--vibes", "on"}, Lane: "gpu", TimeoutSeconds: 60, MemoryBytes: 128 << 20}
	return s, r
}

// TestQueueFIFOAndIdempotency tests retry acknowledgements without duplicates.
func TestQueueFIFOAndIdempotency(t *testing.T) {
	s, r := fixture(t)
	first, err := s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Submit(t.Context(), r)
	if err != nil || again.Sequence != first.Sequence {
		t.Fatal(again, err)
	}
	r.Args = append(r.Args, "--different")
	if _, err = s.Submit(t.Context(), r); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	r.Agent = "other"
	second, err := s.Submit(t.Context(), r)
	if err != nil || second.Sequence != 2 {
		t.Fatal(second, err)
	}
	got, err := s.claim(t.Context(), "worker-a")
	if err != nil || got.Sequence != 1 {
		t.Fatal(got, err)
	}
	if _, err = s.claim(t.Context(), "worker-b"); !errors.Is(err, ErrActive) {
		t.Fatal(err)
	}
	if err = s.finish(t.Context(), 1, "wrong", Result{State: Succeeded}); !errors.Is(err, ErrOwner) {
		t.Fatal(err)
	}
	if err = s.finish(t.Context(), 1, "worker-a", Result{State: Failed, ExitCode: 7, Reason: "test failed"}); err != nil {
		t.Fatal(err)
	}
	got, err = s.claim(t.Context(), "worker-b")
	if err != nil || got.Sequence != 2 {
		t.Fatal(got, err)
	}
	status, err := s.Status(t.Context())
	if err != nil || len(status.Jobs) != 2 || status.Jobs[0].State != Failed || status.Jobs[0].Result.ExitCode != 7 {
		t.Fatal(status, err)
	}
}

// TestConcurrentSubmitters gives each agent independently acknowledged work.
func TestConcurrentSubmitters(t *testing.T) {
	s, r := fixture(t)
	const count = 16
	var wg sync.WaitGroup
	results := make(chan error, count)
	for i := range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request := r
			request.Agent = fmt.Sprint("agent-", i)
			_, err := s.Submit(t.Context(), request)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	status, err := s.Status(t.Context())
	if err != nil || len(status.Jobs) != count {
		t.Fatal(status, err)
	}
	for i, j := range status.Jobs {
		if j.Sequence != uint64(i+1) || j.State != Queued {
			t.Fatal(j)
		}
	}
}

// TestCancellationAndPause keeps held and cancelled work from starting.
func TestCancellationAndPause(t *testing.T) {
	s, r := fixture(t)
	j, err := s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Cancel(t.Context(), j.Sequence, "other"); !errors.Is(err, ErrOwner) {
		t.Fatal(err)
	}
	if err = s.Pause(t.Context(), "disk below floor"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.claim(t.Context(), "worker"); !errors.Is(err, ErrPaused) {
		t.Fatal(err)
	}
	if err = s.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = s.Cancel(t.Context(), j.Sequence, r.Agent); err != nil {
		t.Fatal(err)
	}
	if err = s.Cancel(t.Context(), j.Sequence, r.Agent); err != nil {
		t.Fatal(err)
	}
	if _, err = s.claim(t.Context(), "worker"); !errors.Is(err, ErrEmpty) {
		t.Fatal(err)
	}
	r.Key = "two"
	j, err = s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.claim(t.Context(), "worker"); err != nil {
		t.Fatal(err)
	}
	if err = s.Cancel(t.Context(), j.Sequence, r.Agent); err != nil {
		t.Fatal(err)
	}
	status, err := s.Status(t.Context())
	if err != nil || !status.Jobs[1].CancelRequested || status.Jobs[1].State != Running {
		t.Fatal(status, err)
	}
}

// TestRequestRefusals prevents omitted limits or invalid identities entering.
func TestRequestRefusals(t *testing.T) {
	s, r := fixture(t)
	for name, change := range map[string]func(*Request){
		"agent": func(r *Request) { r.Agent = "" }, "key": func(r *Request) { r.Key = "" },
		"relative": func(r *Request) { r.Command = "slayer" }, "cwd": func(r *Request) { r.Directory = "." },
		"lane": func(r *Request) { r.Lane = "unknown" }, "timeout": func(r *Request) { r.TimeoutSeconds = 0 },
		"nul":         func(r *Request) { r.Args = []string{"a\x00b"} },
		"environment": func(r *Request) { r.Environment = map[string]string{"GAME_LANES": "gpu=stolen"} },
	} {
		t.Run(name, func(t *testing.T) {
			bad := r
			change(&bad)
			if _, err := s.Submit(t.Context(), bad); err == nil {
				t.Fatal("accepted invalid request")
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.Status(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// TestInputIdentityIsFrozen checks changes before execution are visible.
func TestInputIdentityIsFrozen(t *testing.T) {
	s, r := fixture(t)
	input := filepath.Join(r.Directory, "job.yaml")
	if err := os.WriteFile(input, []byte("scene: fizz\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r.Inputs = []string{input}
	j, err := s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	if err = j.verify(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(input, []byte("scene: changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = j.verify(); err == nil {
		t.Fatal("accepted changed input")
	}
}
