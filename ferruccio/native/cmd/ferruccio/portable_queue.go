package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// renderQueue is the path to the selected queue and whether the app creates it.
type renderQueue struct {
	directory string
	private   bool
}

// defaultRenderQueue discovers a shared queue or reserves a private fallback.
func defaultRenderQueue(home string, bundled bool) renderQueue {
	if home == "" {
		return renderQueue{}
	}
	shared := filepath.Join(home, "mesh", "the-vorrh", "renders", "applications", "azazel", "queue")
	if info, err := os.Stat(shared); err == nil && info.IsDir() {
		return renderQueue{directory: shared}
	}
	if bundled {
		local := filepath.Join(home, "Library", "Application Support", "Ferruccio", "Queue")
		return renderQueue{directory: local, private: true}
	}
	return renderQueue{}
}

// prepare provisions only the managed fallback and lets Azazel own its format.
func (queue renderQueue) prepare(ctx context.Context, executable, selected string) error {
	if !queue.private || selected != queue.directory {
		return nil
	}
	if err := queue.ensure(ctx, executable); err != nil {
		return fmt.Errorf("prepare Ferruccio render queue %q: %w", queue.directory, err)
	}
	return nil
}

// ensure publishes only complete queues and keeps any incumbent in place.
func (queue renderQueue) ensure(ctx context.Context, executable string) error {
	if exists, err := privateQueueExists(queue.directory); exists || err != nil {
		return err
	}
	parent := filepath.Dir(queue.directory)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".queue-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	initialized := filepath.Join(staging, "queue")
	if err := initializeQueue(ctx, executable, initialized); err != nil {
		return err
	}
	if err := requirePrivateQueue(initialized); err != nil {
		return fmt.Errorf("Azazel init did not create a usable queue: %w", err)
	}
	if err := installPrivateQueue(initialized, queue.directory); !errors.Is(err, os.ErrExist) {
		return err
	}
	return requirePrivateQueue(queue.directory)
}

// requirePrivateQueue confirms a completed initialization exists.
func requirePrivateQueue(directory string) error {
	exists, err := privateQueueExists(directory)
	if err == nil && !exists {
		return os.ErrNotExist
	}
	return err
}

// privateQueueExists keeps a partial or unexpected directory for inspection.
func privateQueueExists(directory string) (bool, error) {
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("queue location is not a directory")
	}
	database, err := os.Lstat(filepath.Join(directory, "queue.db"))
	if err != nil {
		return false, err
	}
	if !database.Mode().IsRegular() {
		return false, fmt.Errorf("queue.db is not a regular file")
	}
	return true, nil
}

// initializeQueue uses the bundled protocol and bounds an unresponsive child.
func initializeQueue(ctx context.Context, executable, directory string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "init", "--queue", directory)
	command.WaitDelay = time.Second
	output, err := command.CombinedOutput()
	if err == nil {
		return ctx.Err()
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("Azazel init: %w: %s", err, strings.TrimSpace(string(output)))
}
