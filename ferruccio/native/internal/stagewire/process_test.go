package stagewire

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestManagedHelperProcess(t *testing.T) {
	mode := os.Getenv("FERRUCCIO_STAGE_TEST")
	if mode == "" {
		return
	}
	if mode == "wait" {
		if err := os.WriteFile(os.Getenv("FERRUCCIO_STAGE_MARKER"), []byte("started"), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	if filepath.Dir(os.Getenv("SLAYER_CACHE")) != os.Getenv("FERRUCCIO_EXPECT_CACHE_PARENT") || os.Getenv("SLAYER_SHELF") != "/test/shelf" || os.Getenv("SLAYER_DATA") != "/test/data" || os.Getenv("GOMAXPROCS") != "2" {
		os.Exit(3)
	}
	if mode == "reused" {
		if _, err := os.Stat(filepath.Join(os.Getenv("SLAYER_CACHE"), "scratch")); err != nil {
			os.Exit(6)
		}
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("SLAYER_CACHE"), "scratch"), []byte("test"), 0600); err != nil {
		os.Exit(5)
	}
	if err := Write(os.Stdout, sample()); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
}

func TestSessionReusesPrivateCacheAndCleansUp(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("FERRUCCIO_EXPECT_CACHE_PARENT", cache)
	session := &Session{}
	for _, mode := range []string{"snapshot", "reused"} {
		result, err := session.run(context.Background(), fakeCommand(mode), "/helper", "/scene.yaml", "source", "set", cache, "/test/shelf", "/test/data", 2, Camera{}, "")
		if err != nil || result.Scene.Width != 640 {
			t.Fatalf("%s: snapshot = %+v, %v", mode, result, err)
		}
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one private cache: %v %v", entries, err)
	}
	contents, err := os.ReadDir(filepath.Join(cache, entries[0].Name()))
	if err != nil || len(contents) != 1 || contents[0].Name() != "scratch" {
		t.Fatalf("temporary snapshots were retained: %v %v", contents, err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(cache); err != nil || len(entries) != 0 {
		t.Fatalf("close left the private cache behind: %v %v", entries, err)
	}
	if _, err := session.run(context.Background(), fakeCommand("snapshot"), "/helper", "/scene.yaml", "source", "set", cache, "/test/shelf", "/test/data", 2, Camera{}, ""); err == nil {
		t.Fatal("closed session accepted a new request")
	}
}

func TestSessionReapsCancelledHelperBeforeReusingCache(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("FERRUCCIO_EXPECT_CACHE_PARENT", cache)
	marker := filepath.Join(t.TempDir(), "started")
	t.Setenv("FERRUCCIO_STAGE_MARKER", marker)
	session := &Session{}
	defer session.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := session.run(ctx, fakeCommand("wait"), "/helper", "/scene.yaml", "source", "set", cache, "/test/shelf", "/test/data", 2, Camera{}, "")
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatal("helper did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatalf("cancel returned %v", err)
	}
	if _, err := session.run(context.Background(), fakeCommand("snapshot"), "/helper", "/scene.yaml", "source", "set", cache, "/test/shelf", "/test/data", 2, Camera{}, ""); err != nil {
		t.Fatalf("next helper could not use cache: %v", err)
	}
}

func fakeCommand(mode string) commandFactory {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestManagedHelperProcess$")
		cmd.Env = append(os.Environ(), "FERRUCCIO_STAGE_TEST="+mode)
		return cmd
	}
}

func TestProcessTransfersSnapshotAndChildConfiguration(t *testing.T) {
	t.Setenv("SLAYER_CACHE", "/old/cache")
	t.Setenv("SLAYER_SHELF", "/old/shelf")
	t.Setenv("SLAYER_DATA", "/old/data")
	cache := t.TempDir()
	t.Setenv("FERRUCCIO_EXPECT_CACHE_PARENT", cache)
	result, err := run(context.Background(), fakeCommand("snapshot"), "/helper", "/scene.yaml", "source", "set", cache, "/test/shelf", "/test/data", 2)
	if err != nil || result.Scene.Width != 640 {
		t.Fatalf("helper snapshot = %+v, %v", result, err)
	}
	if os.Getenv("SLAYER_CACHE") != "/old/cache" || os.Getenv("SLAYER_SHELF") != "/old/shelf" || os.Getenv("SLAYER_DATA") != "/old/data" {
		t.Fatal("helper changed parent environment")
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary child cache remained: %v %v", entries, err)
	}
}

func TestProcessForwardsWorkingFrameAndCamera(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("FERRUCCIO_EXPECT_CACHE_PARENT", cache)
	command := fakeCommand("snapshot")
	var captured []string
	makeCommand := func(ctx context.Context, helper string, args ...string) *exec.Cmd {
		captured = append([]string(nil), args...)
		return command(ctx, helper, args...)
	}
	session := &Session{}
	defer session.Close()
	camera := Camera{Name: "west", Frame: 3}
	if _, err := session.run(context.Background(), makeCommand, "/helper", "/job.yaml", "source", "job", cache, "/test/shelf", "/test/data", 2, camera, ""); err != nil {
		t.Fatal(err)
	}
	want := []string{"stage", "--path", "/job.yaml", "--kind", "job", "--frame", "3", "--camera", "west"}
	if !slices.Equal(captured, want) {
		t.Fatalf("helper args = %v, want %v", captured, want)
	}
}

func TestProcessCancellationAndRefusals(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	t.Setenv("FERRUCCIO_STAGE_MARKER", marker)
	cache := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := run(ctx, fakeCommand("wait"), "/helper", "/scene.yaml", "source", "set", cache, "", "", 2)
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatal("helper did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("cancel returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop and reap helper")
	}
	if entries, err := os.ReadDir(cache); err != nil || len(entries) != 0 {
		t.Fatalf("cancel left temporary child cache: %v %v", entries, err)
	}
	for _, test := range []struct{ path, source, cache string }{
		{"relative", "source", "/test/cache"},
		{"/scene.yaml", "source", ""},
		{"/scene.yaml", strings.Repeat("x", MaxSource+1), "/test/cache"},
	} {
		if _, err := run(context.Background(), fakeCommand("wait"), "/helper", test.path, test.source, "set", test.cache, "", "", 2); err == nil {
			t.Fatal("invalid request started a helper")
		}
	}
}
