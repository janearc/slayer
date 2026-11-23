package encoder

import (
	"context"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestOutputPathProblemsAreRefused checks an output beneath a plain file, and one in a directory that does not exist, are each refused by name.
func TestOutputPathProblemsAreRefused(t *testing.T) {
	plain := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(plain, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := fakeEncoder(t, context.Background(), "", func(c *Config) { c.Output = filepath.Join(plain, "out.mp4") })
	if err == nil || !strings.Contains(err.Error(), "inspect output") {
		t.Errorf("output under a plain file: %v", err)
	}
	for _, overwrite := range []bool{false, true} {
		_, _, _, err = fakeEncoder(t, context.Background(), "", func(c *Config) {
			c.Output = filepath.Join(t.TempDir(), "absent", "out.mp4")
			c.Overwrite = overwrite
		})
		if err == nil || !strings.Contains(err.Error(), "temporary output") {
			t.Errorf("missing directory with overwrite %v: %v", overwrite, err)
		}
	}
}

// TestAFailedWriteIsKeptAndACanceledContextRefusesWrites checks the first failure is the one every later write returns, and a canceled context stops a write before any byte is sent.
func TestAFailedWriteIsKeptAndACanceledContextRefusesWrites(t *testing.T) {
	e, _, _, err := fakeEncoder(t, context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, first := e.WriteFrame(0, nil)
	if first == nil {
		t.Fatal("a nil frame was accepted")
	}
	blank := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if _, again := e.WriteFrame(0, blank); again != first {
		t.Errorf("second write returned %v, want the first failure %v", again, first)
	}
	ctx, cancel := context.WithCancel(context.Background())
	e, _, _, err = fakeEncoder(t, ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	blankAfterCancel := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if _, err = e.WriteFrame(0, blankAfterCancel); err == nil || !strings.Contains(err.Error(), "frame 0: context canceled") {
		t.Errorf("write after cancel: %v", err)
	}
	if e.Stats().Bytes != 0 {
		t.Errorf("%d bytes were sent after cancel", e.Stats().Bytes)
	}
}

// TestCloseReportsAnEncodedFileThatVanished checks that if the encoder's temporary file is gone when ffmpeg ends, nothing is published and the error says so.
func TestCloseReportsAnEncodedFileThatVanished(t *testing.T) {
	e, cfg, _, err := fakeEncoder(t, context.Background(), "empty", nil)
	if err != nil {
		t.Fatal(err)
	}
	blank := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if _, err = e.WriteFrame(0, blank); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(filepath.Dir(cfg.Output), "."+filepath.Base(cfg.Output)+"-*.mp4")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) != 1 {
		t.Fatalf("temporary files %v, %v", matches, err)
	}
	if err = os.Remove(matches[0]); err != nil {
		t.Fatal(err)
	}
	if err = e.Close(); err == nil || !strings.Contains(err.Error(), "inspect encoded output") {
		t.Fatalf("close: %v", err)
	}
	if _, err = os.Stat(cfg.Output); !os.IsNotExist(err) {
		t.Fatalf("output published: %v", err)
	}
}

// TestCloseReportsATemporaryFileThatCannotBeRemoved checks that when the output directory turns read-only, the failure to publish and the failure to remove the temporary file are both reported.
func TestCloseReportsATemporaryFileThatCannotBeRemoved(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir, records := t.TempDir(), t.TempDir()
	cfg := Config{FFmpeg: executable, Output: filepath.Join(dir, "stretch.mp4"), Width: 2, Height: 2, FPS: 24, FrameCount: 1}
	factory := func(ctx context.Context, binary string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, executable, append([]string{"-test.run=^TestEncoderProcess$", "--"}, args...)...)
		cmd.Env = append(os.Environ(), "OMNIPOSITOR_ENCODER_CHILD=1", "OMNIPOSITOR_ENCODER_MODE=",
			"OMNIPOSITOR_ENCODER_RECORD="+filepath.Join(records, "record.json"))
		return cmd
	}
	e, err := newEncoder(context.Background(), cfg, factory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Abort() })
	blank := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if _, err = e.WriteFrame(0, blank); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	err = e.Close()
	if err == nil || !strings.Contains(err.Error(), "publish output") || !strings.Contains(err.Error(), "remove temporary") {
		t.Fatalf("close: %v", err)
	}
}
