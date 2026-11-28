//go:build darwin

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLaunchQueuePrefersTheSharedPath(t *testing.T) {
	home := t.TempDir()
	shared := filepath.Join(home, "mesh", "the-vorrh", "renders", "applications", "azazel", "queue")
	if err := os.MkdirAll(shared, 0o700); err != nil {
		t.Fatal(err)
	}
	if got := defaultRenderQueue(home, true); got.directory != shared || got.private {
		t.Fatalf("shared queue became %+v", got)
	}
}

func TestLaunchQueueOffersPrivateDefaultOnlyForTheBundle(t *testing.T) {
	home := t.TempDir()
	wanted := filepath.Join(home, "Library", "Application Support", "Ferruccio", "Queue")
	if got := defaultRenderQueue(home, true); got.directory != wanted || !got.private {
		t.Fatalf("private queue became %+v", got)
	}
	if _, err := os.Stat(wanted); !os.IsNotExist(err) {
		t.Fatalf("selecting a default changed disk state: %v", err)
	}
	if got := defaultRenderQueue(home, false); got.directory != "" || got.private {
		t.Fatalf("unpackaged command selected a private queue: %+v", got)
	}
}

func TestLaunchQueueRequiresTheBundledAppMarker(t *testing.T) {
	contents := filepath.Join(t.TempDir(), "Ferruccio.app", "Contents")
	executable := filepath.Join(contents, "MacOS", "ferruccio")
	if isBundleLaunch(executable) {
		t.Fatal("a bundle-shaped path without the packaged marker enabled worker management")
	}
	resources := filepath.Join(contents, "Resources")
	if err := os.MkdirAll(resources, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resources, "ferruccio-build-owned"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if !isBundleLaunch(executable) {
		t.Fatal("the packaged launch was not recognized")
	}
}

func TestLaunchQueueInstallsOneCompleteQueueAcrossConcurrentRenders(t *testing.T) {
	queue := defaultRenderQueue(t.TempDir(), true)
	executable := queueInitializer(t)
	var runners sync.WaitGroup
	for range 2 {
		runners.Go(func() {
			if err := queue.prepare(t.Context(), executable, queue.directory); err != nil {
				t.Error(err)
			}
		})
	}
	runners.Wait()
	data, err := os.ReadFile(filepath.Join(queue.directory, "queue.db"))
	if err != nil || string(data) != "created by init" {
		t.Fatalf("installed queue = %q, error %v", data, err)
	}
	if err := queue.prepare(t.Context(), "/missing/executable", queue.directory); err != nil {
		t.Fatalf("existing queue should reopen without initialization: %v", err)
	}
}

func TestLaunchQueueRetainsAConflictingDirectoryAndLeavesOtherSelectionsAlone(t *testing.T) {
	queue := defaultRenderQueue(t.TempDir(), true)
	if err := os.MkdirAll(queue.directory, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := queueInitializer(t)
	if err := queue.prepare(t.Context(), executable, queue.directory); err == nil {
		t.Fatal("expected a diagnostic for a queue directory without a database")
	}
	entries, err := os.ReadDir(queue.directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("existing directory was modified: %v, %v", entries, err)
	}
	configured := filepath.Join(t.TempDir(), "other queue")
	if err := queue.prepare(t.Context(), "/missing/executable", configured); err != nil {
		t.Fatalf("private initializer was applied to a different queue: %v", err)
	}
	if _, err := os.Stat(configured); !os.IsNotExist(err) {
		t.Fatalf("private initializer touched a different queue: %v", err)
	}
}

func TestLaunchQueueInstallCannotReplaceEvenAnEmptyDirectory(t *testing.T) {
	source, destination := t.TempDir(), t.TempDir()
	if err := installPrivateQueue(source, destination); !os.IsExist(err) {
		t.Fatalf("exclusive install returned %v for an existing empty directory", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("install moved its source: %v", err)
	}
}

func TestLaunchQueueFailedInitializationLeavesDestinationAvailableForRetry(t *testing.T) {
	queue := defaultRenderQueue(t.TempDir(), true)
	if err := queue.prepare(context.Background(), "/missing/executable", queue.directory); err == nil {
		t.Fatal("expected initializer failure")
	}
	entries, err := os.ReadDir(filepath.Dir(queue.directory))
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed initialization left scratch or final queue: %v, %v", entries, err)
	}
	if err := queue.prepare(t.Context(), queueInitializer(t), queue.directory); err != nil {
		t.Fatal(err)
	}
}

func queueInitializer(t *testing.T) string {
	t.Helper()
	program := filepath.Join(t.TempDir(), "test azazel")
	script := strings.TrimSpace(`
#!/bin/sh
set -eu
test "$#" -eq 3
test "$1" = init
test "$2" = --queue
umask 077
mkdir "$3"
printf 'created by init' > "$3/queue.db"
`)
	if err := os.WriteFile(program, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return program
}
