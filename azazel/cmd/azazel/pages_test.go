package main

import (
	"bytes"
	"context"
	"github.com/janearc/slayer/azazel/internal/queue"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestWorkPublishesWithoutAnotherOperatorCommand exercises the main-worker hook.
func TestWorkPublishesWithoutAnotherOperatorCommand(t *testing.T) {
	root := publishRoot(t)
	queueRoot := filepath.Join(t.TempDir(), "queue")
	store := queue.At(queueRoot)
	if err := store.Init(t.Context(), queue.Config{Game: "/usr/bin/true"}); err != nil {
		t.Fatal(err)
	}
	job, err := store.Submit(t.Context(), queue.Request{Agent: "fixture", Key: "visible", Command: "/usr/bin/true", Args: []string{"--out", filepath.Join(root, "frames/2026-11-15-studio-one/one.png")}, Directory: root, Lane: "tests", TimeoutSeconds: 5})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Cancel(t.Context(), job.Sequence, "fixture"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan int, 1)
	var out, wrong bytes.Buffer
	go func() {
		done <- queueTool(ctx).Run([]string{"work", "--serve", "--queue", queueRoot, "--renders", root}, &out, &wrong)
	}()
	deadline := time.After(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "latest.html")); err == nil {
			break
		}
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatal("worker failed to publish", wrong.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-done
	if _, err := os.Stat(filepath.Join(root, "frames/2026-11-15-studio-one/index.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(queueRoot, "pages.log")); err != nil {
		t.Fatal(err)
	}
}

// serving runs the pages verb and returns its exit, output and
// complaint.
func serving(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, wrong bytes.Buffer
	code := queueTool(t.Context()).Run(append([]string{"pages"}, args...),
		&out, &wrong)
	return code, out.String(), wrong.String()
}

// TestPagesRefusesASecondCoordinator: with the queue's lock held, pages
// --serve exits at once and says the queue is already served.
func TestPagesRefusesASecondCoordinator(t *testing.T) {
	queueRoot, root := t.TempDir(), publishRoot(t)
	held, err := os.OpenFile(filepath.Join(queueRoot, "pages.lock"),
		os.O_RDWR|os.O_CREATE, 0o600) // defined: private, as the queue's
	if err == nil {
		descriptor := int(held.Fd())
		err = syscall.Flock(descriptor, syscall.LOCK_EX)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	code, _, wrong := serving(t, "--serve", "--queue", queueRoot,
		"--renders", root)
	if code == 0 || !strings.Contains(wrong, "already served") {
		t.Fatal(code, wrong)
	}
}

// TestPagesDryRunAndRefusalsTouchNothing is the anti-test: a preview, a
// call without --serve and one with a relative root take no lock.
func TestPagesDryRunAndRefusalsTouchNothing(t *testing.T) {
	queueRoot, root := t.TempDir(), publishRoot(t)
	code, out, wrong := serving(t, "--serve", "--queue", queueRoot,
		"--renders", root, "--dry-run")
	if code != 0 || !strings.Contains(out, `"dry_run":true`) {
		t.Fatal(code, out, wrong)
	}
	for _, args := range [][]string{
		{"--queue", queueRoot, "--renders", root},
		{"--serve", "--queue", queueRoot, "--renders", "renders"},
	} {
		if code, _, _ := serving(t, args...); code == 0 {
			t.Error("pages took", args)
		}
	}
	entries, err := os.ReadDir(queueRoot)
	if err != nil || len(entries) != 0 {
		t.Fatal("pages wrote into the queue", entries, err)
	}
}
