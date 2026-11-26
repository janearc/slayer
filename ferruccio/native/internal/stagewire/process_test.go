package stagewire

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
	if err := os.WriteFile(filepath.Join(os.Getenv("SLAYER_CACHE"), "scratch"), []byte("test"), 0600); err != nil {
		os.Exit(5)
	}
	if err := Write(os.Stdout, sample()); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
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
