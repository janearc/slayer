package encoder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type processRecord struct {
	Args   []string
	Pixels []byte
}

func TestEncoderProcess(t *testing.T) {
	if os.Getenv("OMNIPOSITOR_ENCODER_CHILD") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	args = args[1:]
	switch os.Getenv("OMNIPOSITOR_ENCODER_MODE") {
	case "stall":
		for {
			time.Sleep(time.Second)
		}
	case "fail":
		fmt.Fprint(os.Stderr, "fake encoder failure")
		os.Exit(7)
	}
	pixels, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(8)
	}
	data, _ := json.Marshal(processRecord{Args: args, Pixels: pixels})
	if os.WriteFile(os.Getenv("OMNIPOSITOR_ENCODER_RECORD"), data, 0600) != nil {
		os.Exit(9)
	}
	if os.Getenv("OMNIPOSITOR_ENCODER_MODE") != "empty" {
		if os.WriteFile(args[len(args)-1], []byte("fake mp4"), 0600) != nil {
			os.Exit(10)
		}
	}
	if os.Getenv("OMNIPOSITOR_ENCODER_MODE") == "stall-after-eof" {
		for {
			time.Sleep(time.Second)
		}
	}
	os.Exit(0)
}

func fakeEncoder(t *testing.T, ctx context.Context, mode string, change func(*Config)) (*Encoder, Config, string, error) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg := Config{FFmpeg: executable, Output: filepath.Join(dir, "stretch.mp4"), Width: 2, Height: 2, FPS: 24, FrameCount: 1}
	if change != nil {
		change(&cfg)
	}
	record := filepath.Join(dir, "record.json")
	factory := func(ctx context.Context, binary string, args ...string) *exec.Cmd {
		resolved, err := filepath.EvalSymlinks(executable)
		if err != nil || binary != resolved {
			t.Fatalf("binary = %q, want resolved %q: %v", binary, resolved, err)
		}
		cmd := exec.CommandContext(ctx, executable, append([]string{"-test.run=^TestEncoderProcess$", "--"}, args...)...)
		cmd.Env = append(os.Environ(), "OMNIPOSITOR_ENCODER_CHILD=1", "OMNIPOSITOR_ENCODER_MODE="+mode, "OMNIPOSITOR_ENCODER_RECORD="+record)
		return cmd
	}
	e, err := newEncoder(ctx, cfg, factory)
	if e != nil {
		t.Cleanup(func() { _ = e.Abort() })
	}
	return e, cfg, record, err
}

func readRecord(t *testing.T, path string) processRecord {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got processRecord
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func noTemporary(t *testing.T, output string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(output), "."+filepath.Base(output)+"-*.mp4"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary outputs: %v, %v", matches, err)
	}
}

func TestExactArgumentsAndPaddedRGBA(t *testing.T) {
	e, cfg, record, err := fakeEncoder(t, context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	frame := &image.RGBA{Rect: image.Rect(7, 9, 9, 11), Stride: 12, Pix: []byte{1, 2, 3, 255, 4, 5, 6, 255, 91, 92, 93, 94, 7, 8, 9, 255, 10, 11, 12, 255}}
	duration, err := e.WriteFrame(0, frame)
	if err != nil || duration <= 0 {
		t.Fatalf("write = %v, %v", duration, err)
	}
	if _, err := os.Stat(cfg.Output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published before close: %v", err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(); err != nil {
		t.Fatalf("repeat close: %v", err)
	}
	got := readRecord(t, record)
	wantArgs := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-f", "rawvideo", "-pixel_format", "rgba", "-video_size", "2x2", "-framerate", "24", "-i", "pipe:0", "-map", "0:v:0", "-an", "-frames:v", "1", "-vf", "pad=2:16:0:7:black,scale=iw:ih:out_color_matrix=bt709:out_range=tv,format=yuv420p", "-c:v", "hevc_videotoolbox", "-profile:v", "main", "-tag:v", "hvc1", "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709", "-color_range", "tv", "-b:v", "20M", "-maxrate", "30M", "-bufsize", "40M", "-g", "48", "-movflags", "+faststart", "-f", "mp4", got.Args[len(got.Args)-1]}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Fatalf("args = %q\nwant = %q", got.Args, wantArgs)
	}
	if filepath.Dir(got.Args[len(got.Args)-1]) != filepath.Dir(cfg.Output) || got.Args[len(got.Args)-1] == cfg.Output {
		t.Fatal("not a sibling temporary output")
	}
	if !bytes.Equal(got.Pixels, []byte{1, 2, 3, 255, 4, 5, 6, 255, 7, 8, 9, 255, 10, 11, 12, 255}) {
		t.Fatalf("RGBA/stride/order: %v", got.Pixels)
	}
	stats := e.Stats()
	if stats.Frames != 1 || stats.Bytes != 16 || stats.WriteTime != duration {
		t.Fatalf("stats = %+v", stats)
	}
	if data, err := os.ReadFile(cfg.Output); err != nil || string(data) != "fake mp4" {
		t.Fatalf("output = %q, %v", data, err)
	}
	noTemporary(t, cfg.Output)
}

func TestFourKSettings(t *testing.T) {
	e, cfg, record, err := fakeEncoder(t, context.Background(), "", func(c *Config) { c.Width = 4096; c.Height = 3002; c.FPS = 23.976 })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	args := readRecord(t, record).Args
	for option, value := range map[string]string{"-b:v": "60M", "-maxrate": "90M", "-bufsize": "120M", "-vf": "pad=4096:3008:0:3:black,scale=iw:ih:out_color_matrix=bt709:out_range=tv,format=yuv420p", "-framerate": "23.976"} {
		found := false
		for i := 0; i+1 < len(args); i++ {
			if args[i] == option && args[i+1] == value {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s %s: %q", option, value, args)
		}
	}
}

// An odd height is taken, as the preview's 751 is: the picture is padded
// to the next height sixteen divides, 752, with the odd row of black at
// the bottom, so what is encoded is even.
func TestAnOddHeightIsPaddedToAnEvenOne(t *testing.T) {
	e, _, _, err := fakeEncoder(t, context.Background(), "", func(c *Config) { c.Width, c.Height = 1024, 751 })
	if err != nil {
		t.Fatalf("an odd height was refused: %v", err)
	}
	_ = e.Abort()
	args := arguments(Config{Width: 1024, Height: 751, FPS: 24, FrameCount: 1}, "out.mp4")
	padded := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "pad=1024:752:0:0:black") {
			padded = true
		}
	}
	if !padded {
		t.Fatalf("a 751 high picture is not padded to 752: %q", args)
	}
}

// the anti-test of the one above: an odd width is still refused, for
// nothing pads the width
func TestAnOddWidthIsStillRefused(t *testing.T) {
	if e, _, _, err := fakeEncoder(t, context.Background(), "", func(c *Config) { c.Width, c.Height = 1023, 751 }); err == nil {
		_ = e.Abort()
		t.Fatal("an odd width was taken")
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for name, change := range map[string]func(*Config){
		"binary relative": func(c *Config) { c.FFmpeg = "ffmpeg" },
		"binary missing":  func(c *Config) { c.FFmpeg = "/not-an-encoder" },
		"output empty":    func(c *Config) { c.Output = "" },
		"width zero":      func(c *Config) { c.Width = 0 },
		"width odd":       func(c *Config) { c.Width = 3 },
		"height zero":     func(c *Config) { c.Height = 0 },
		"height negative": func(c *Config) { c.Height = -2 },
		"dimension huge":  func(c *Config) { c.Width = math.MaxInt },
		"pixel budget":    func(c *Config) { c.Width = 8192; c.Height = 8192 },
		"count zero":      func(c *Config) { c.FrameCount = 0 },
		"count huge":      func(c *Config) { c.FrameCount = math.MaxInt },
		"fps zero":        func(c *Config) { c.FPS = 0 },
		"fps nan":         func(c *Config) { c.FPS = math.NaN() },
		"fps infinity":    func(c *Config) { c.FPS = math.Inf(1) },
		"fps huge":        func(c *Config) { c.FPS = 1000 },
	} {
		t.Run(name, func(t *testing.T) {
			if e, _, _, err := fakeEncoder(t, context.Background(), "", change); err == nil {
				_ = e.Abort()
				t.Fatal("accepted invalid config")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, _, err := fakeEncoder(t, ctx, "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled startup: %v", err)
	}
}

func TestInvalidFramesNeverPublish(t *testing.T) {
	for name, frame := range map[string]*image.RGBA{
		"nil":             nil,
		"size":            image.NewRGBA(image.Rect(0, 0, 4, 2)),
		"short":           {Rect: image.Rect(0, 0, 2, 2), Stride: 8, Pix: make([]byte, 15)},
		"stride":          {Rect: image.Rect(0, 0, 2, 2), Stride: 7, Pix: make([]byte, 16)},
		"negative stride": {Rect: image.Rect(0, 0, 2, 2), Stride: -8, Pix: make([]byte, 16)},
		"overflow stride": {Rect: image.Rect(0, 0, 2, 2), Stride: math.MaxInt, Pix: make([]byte, 16)},
	} {
		t.Run(name, func(t *testing.T) {
			e, cfg, _, err := fakeEncoder(t, context.Background(), "", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = e.WriteFrame(0, frame); err == nil {
				t.Fatal("accepted invalid frame")
			}
			if err = e.Close(); err == nil {
				t.Fatal("published invalid stream")
			}
			if _, err = os.Stat(cfg.Output); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("output exists: %v", err)
			}
			noTemporary(t, cfg.Output)
		})
	}
}

func TestCountAndOrdering(t *testing.T) {
	for _, mode := range []string{"missing", "extra", "out of order"} {
		t.Run(mode, func(t *testing.T) {
			e, cfg, _, err := fakeEncoder(t, context.Background(), "", nil)
			if err != nil {
				t.Fatal(err)
			}
			frame := image.NewRGBA(image.Rect(0, 0, 2, 2))
			if mode == "extra" {
				if _, err = e.WriteFrame(0, frame); err != nil {
					t.Fatal(err)
				}
			}
			if mode != "missing" {
				if _, err = e.WriteFrame(1, frame); err == nil {
					t.Fatal("accepted wrong count/order")
				}
			}
			if err = e.Close(); err == nil {
				t.Fatal("accepted incomplete/extra stream")
			}
			noTemporary(t, cfg.Output)
		})
	}
}

func TestChildFailureAndEmptyOutput(t *testing.T) {
	for _, mode := range []string{"fail", "empty"} {
		t.Run(mode, func(t *testing.T) {
			e, cfg, _, err := fakeEncoder(t, context.Background(), mode, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, writeErr := e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, 2, 2)))
			err = e.Close()
			if err == nil {
				t.Fatalf("accepted child failure, write: %v", writeErr)
			}
			if mode == "fail" && !strings.Contains(err.Error(), "fake encoder failure") {
				t.Fatalf("missing stderr: %v", err)
			}
			if _, err = os.Stat(cfg.Output); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("output exists: %v", err)
			}
			noTemporary(t, cfg.Output)
		})
	}
}

func TestCancellationUnblocksBackpressure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e, cfg, _, err := fakeEncoder(t, ctx, "stall", func(c *Config) { c.Width = 2048; c.Height = 1152 })
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))); done <- err }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("write cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("blocked pipe did not cancel")
	}
	if err := e.Close(); !errors.Is(err, context.Canceled) {
		t.Fatalf("close cancellation: %v", err)
	}
	if e.Stats().WriteTime <= 0 {
		t.Fatal("missing failed-write timing")
	}
	noTemporary(t, cfg.Output)
}

func TestNoOverwriteAndAtomicPublication(t *testing.T) {
	for _, mode := range []string{"existing", "raced", "overwrite"} {
		t.Run(mode, func(t *testing.T) {
			e, cfg, _, err := fakeEncoder(t, context.Background(), "", func(c *Config) {
				c.Overwrite = mode == "overwrite"
				if mode != "raced" {
					if err := os.WriteFile(c.Output, []byte("original"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			})
			if mode == "existing" {
				if !errors.Is(err, os.ErrExist) {
					t.Fatalf("existing output: %v", err)
				}
				noTemporary(t, cfg.Output)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "raced" {
				if err := os.WriteFile(cfg.Output, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			err = e.Close()
			want := "fake mp4"
			if mode == "raced" {
				want = "original"
				if !errors.Is(err, os.ErrExist) {
					t.Fatalf("racing output: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(cfg.Output)
			if err != nil || string(data) != want {
				t.Fatalf("output = %q, %v", data, err)
			}
			noTemporary(t, cfg.Output)
		})
	}
}

func TestPackedRGBAAndAlreadyAlignedHeight(t *testing.T) {
	e, cfg, record, err := fakeEncoder(t, context.Background(), "", func(c *Config) { c.Height = 16; c.FrameCount = 2 })
	if err != nil {
		t.Fatal(err)
	}
	frame := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	for i := range frame.Pix {
		frame.Pix[i] = byte(i)
	}
	for i := 0; i < 2; i++ {
		if _, err = e.WriteFrame(i, frame); err != nil {
			t.Fatal(err)
		}
	}
	if err = e.Close(); err != nil {
		t.Fatal(err)
	}
	got := readRecord(t, record)
	if !bytes.Equal(got.Pixels, append(append([]byte{}, frame.Pix...), frame.Pix...)) {
		t.Fatal("packed frame ordering changed")
	}
	if !strings.Contains(strings.Join(got.Args, " "), "pad=2:16:0:0:black,scale=iw:ih:out_color_matrix=bt709:out_range=tv,format=yuv420p") {
		t.Fatalf("aligned padding: %q", got.Args)
	}
	if e.Stats().Frames != 2 || e.Stats().Bytes != 256 {
		t.Fatalf("stats: %+v", e.Stats())
	}
}

func TestAbortUnblocksWrite(t *testing.T) {
	e, cfg, _, err := fakeEncoder(t, context.Background(), "stall", func(c *Config) { c.Width = 2048; c.Height = 1152 })
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))); done <- err }()
	time.Sleep(50 * time.Millisecond)
	if err = e.Abort(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("write succeeded after abort")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("abort did not unblock write")
	}
	noTemporary(t, cfg.Output)
}

func TestBoundedDiagnosticsAndBinaryValidation(t *testing.T) {
	var log boundedLog
	input := bytes.Repeat([]byte("x"), 20000)
	for i := 0; i < 2; i++ {
		if n, err := log.Write(input); n != len(input) || err != nil {
			t.Fatalf("log write: %d, %v", n, err)
		}
	}
	if len(log.data) != 16384 {
		t.Fatalf("unbounded stderr: %d", len(log.data))
	}
	if _, err := New(nil, Config{}); err == nil {
		t.Fatal("accepted nil context")
	}
	path := filepath.Join(t.TempDir(), "not-executable")
	if err := os.WriteFile(path, []byte("not executable"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, binary := range []string{path, filepath.Dir(path)} {
		if _, err := New(context.Background(), Config{FFmpeg: binary, Output: filepath.Join(t.TempDir(), "out.mp4"), Width: 2, Height: 2, FPS: 24, FrameCount: 1}); err == nil {
			t.Fatalf("accepted non executable binary %s", binary)
		}
	}
}

func TestFinalizationTimeoutAfterEOF(t *testing.T) {
	e, cfg, record, err := fakeEncoder(t, context.Background(), "stall-after-eof", func(c *Config) { c.FinalizeTimeout = 100 * time.Millisecond })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- e.Close() }()
	select {
	case err = <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout error: %v", err)
		}
	case <-time.After(3 * time.Second):
		_ = e.Abort()
		t.Fatal("Close hung after EOF")
	}
	if len(readRecord(t, record).Pixels) != 16 {
		t.Fatal("child did not receive EOF and full frame")
	}
	if _, err = os.Stat(cfg.Output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published timed-out output: %v", err)
	}
	noTemporary(t, cfg.Output)
}

func TestFinalizationTimeoutConfiguration(t *testing.T) {
	if _, _, _, err := fakeEncoder(t, context.Background(), "", func(c *Config) { c.FinalizeTimeout = -time.Second }); err == nil {
		t.Fatal("accepted negative finalization timeout")
	}
	e, _, _, err := fakeEncoder(t, context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if e.config.FinalizeTimeout != 30*time.Second {
		t.Fatalf("default timeout: %v", e.config.FinalizeTimeout)
	}
}

func TestAbortIntentPreventsClosePublication(t *testing.T) {
	e, cfg, _, err := fakeEncoder(t, context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	e.mu.Lock()
	aborted := make(chan error, 1)
	go func() { aborted <- e.Abort() }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		e.publishMu.Lock()
		intent := e.aborted
		e.publishMu.Unlock()
		if intent {
			break
		}
		if time.Now().After(deadline) {
			e.mu.Unlock()
			t.Fatal("Abort did not register intent independently of Close lock")
		}
		time.Sleep(time.Millisecond)
	}
	closed := make(chan error, 1)
	go func() { closed <- e.Close() }()
	e.mu.Unlock()
	if err := <-aborted; err != nil {
		t.Fatal(err)
	}
	if err := <-closed; !errors.Is(err, context.Canceled) {
		t.Fatalf("Close lost abort intent: %v", err)
	}
	if _, err = os.Stat(cfg.Output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published aborted output: %v", err)
	}
	noTemporary(t, cfg.Output)
}

func TestAbortAfterSuccessfulChildBeforePublication(t *testing.T) {
	e, cfg, record, err := fakeEncoder(t, context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	paused := make(chan struct{})
	resume := make(chan struct{})
	abortRegistered := make(chan struct{})
	originalCancel := e.cancel
	var calls atomic.Int32
	e.cancel = func() {
		originalCancel()
		switch calls.Add(1) {
		case 1:
			close(paused)
			<-resume
		case 2:
			close(abortRegistered)
		}
	}
	defer close(resume)
	closed := make(chan error, 1)
	go func() { closed <- e.Close() }()
	select {
	case <-paused:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not reach post-Wait cancellation hook")
	}
	if e.command.ProcessState == nil || !e.command.ProcessState.Success() {
		t.Fatal("Close paused without successful child completion")
	}
	if len(readRecord(t, record).Pixels) != 16 {
		t.Fatal("child did not consume full frame")
	}
	if info, err := os.Stat(e.temporary); err != nil || info.Size() == 0 {
		t.Fatalf("missing completed temporary output: %v", err)
	}
	if _, err := os.Stat(cfg.Output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output published before publication gate: %v", err)
	}
	aborted := make(chan error, 1)
	go func() { aborted <- e.Abort() }()
	select {
	case <-abortRegistered:
	case <-time.After(5 * time.Second):
		t.Fatal("Abort did not register while Close was paused")
	}
	e.publishMu.Lock()
	intent := e.aborted
	e.publishMu.Unlock()
	if !intent {
		t.Fatal("Abort canceled without registering publication intent")
	}
	resume <- struct{}{}
	select {
	case err := <-closed:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Close lost late abort intent: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not finish after resume")
	}
	select {
	case err := <-aborted:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Abort did not finish")
	}
	if _, err := os.Stat(cfg.Output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published output despite late Abort: %v", err)
	}
	noTemporary(t, cfg.Output)
}

func TestAbortAndStartupFailure(t *testing.T) {
	e, cfg, _, err := fakeEncoder(t, context.Background(), "stall", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Abort(); err != nil {
		t.Fatal(err)
	}
	if err = e.Abort(); err != nil {
		t.Fatal(err)
	}
	if _, err = e.WriteFrame(0, image.NewRGBA(image.Rect(0, 0, 2, 2))); err == nil {
		t.Fatal("write after abort")
	}
	if err = e.Close(); err == nil {
		t.Fatal("close after abort")
	}
	noTemporary(t, cfg.Output)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg.FFmpeg = executable
	_, err = newEncoder(context.Background(), cfg, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, filepath.Join(t.TempDir(), "missing"))
	})
	if err == nil {
		t.Fatal("accepted failed start")
	}
	noTemporary(t, cfg.Output)
}
