package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
)

// writeTrace writes the trace of profile to a new file at path. It writes a
// temporary file in the same directory, syncs it, and hard links it into
// place, so an existing file is never replaced. The temporary file is removed
// and the directory synced.
func writeTrace(path string, profile *compositor.Profile) (resultErr error) {
	parent := filepath.Dir(path)
	file, err := os.CreateTemp(parent, ".omnipositor-trace-*.json")
	if err != nil {
		return fmt.Errorf("trace: create temporary file: %w", err)
	}
	temporary := file.Name()
	defer func() {
		if err := os.Remove(temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	if err := profile.WriteTrace(file); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Link(temporary, path); err != nil {
		return fmt.Errorf("trace: publish without replacing existing file: %w", err)
	}
	if err := os.Remove(temporary); err != nil {
		return fmt.Errorf("trace: remove temporary file: %w", err)
	}
	directory, err := os.Open(parent)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
