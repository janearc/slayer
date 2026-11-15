package encoder

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const MaxPixels = 32 << 20
const MaxFrames = 1 << 20

type Config struct {
	FFmpeg          string
	Output          string
	Width           int
	Height          int
	FPS             float64
	FrameCount      int
	Overwrite       bool
	FinalizeTimeout time.Duration
}

type Stats struct {
	Frames    int
	Bytes     int64
	WriteTime time.Duration
}

type Encoder struct {
	mu        sync.Mutex
	publishMu sync.Mutex
	aborted   bool
	ctx       context.Context
	cancel    context.CancelFunc
	config    Config
	command   *exec.Cmd
	stdin     io.WriteCloser
	stderr    boundedLog
	temporary string
	stats     Stats
	failure   error
	closed    bool
	result    error
}

type boundedLog struct{ data []byte }

// Write keeps the first 16384 bytes of ffmpeg's error output and drops the
// rest, reporting all of p as written.
func (b *boundedLog) Write(p []byte) (int, error) {
	n := len(p)
	if room := 16384 - len(b.data); room > 0 {
		if len(p) > room {
			p = p[:room]
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}

// New starts an Encoder for config, running ffmpeg with exec.CommandContext.
// Frames are then given to WriteFrame in order.
func New(ctx context.Context, config Config) (*Encoder, error) {
	return newEncoder(ctx, config, exec.CommandContext)
}

// newEncoder checks config and starts ffmpeg through command, which tests may
// replace. It refuses a nil or finished ctx, an odd or too large frame, a
// frame count outside [1, MaxFrames], fps outside (0, 240], a negative finish
// timeout (zero means 30 seconds), an ffmpeg that is not an absolute path to
// an executable file, and an existing output unless Overwrite is set. It
// encodes to a temporary file beside the output.
func newEncoder(ctx context.Context, config Config, command func(context.Context, string, ...string) *exec.Cmd) (*Encoder, error) {
	if ctx == nil {
		return nil, errors.New("encoder: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("encoder: %w", err)
	}
	// The width must be even, for the film's halved colour. The height
	// need not be: the picture is padded to a height that sixteen
	// divides before it is encoded, which is even whatever came in. The
	// preview's frames are 751 tall.
	if config.Width <= 0 || config.Height <= 0 || config.Width%2 != 0 || config.Width > MaxPixels/config.Height {
		return nil, fmt.Errorf("encoder: dimensions must be positive, the width even, at most %d pixels", MaxPixels)
	}
	if config.FrameCount <= 0 || config.FrameCount > MaxFrames {
		return nil, fmt.Errorf("encoder: frame count must be in [1,%d]", MaxFrames)
	}
	if math.IsNaN(config.FPS) || math.IsInf(config.FPS, 0) || config.FPS <= 0 || config.FPS > 240 {
		return nil, errors.New("encoder: fps must be finite and in (0,240]")
	}
	if config.FinalizeTimeout < 0 {
		return nil, errors.New("encoder: finalization timeout must not be negative")
	}
	if config.FinalizeTimeout == 0 {
		config.FinalizeTimeout = 30 * time.Second
	}
	if !filepath.IsAbs(config.FFmpeg) {
		return nil, errors.New("encoder: ffmpeg requires an explicit absolute binary path")
	}
	binary, err := filepath.EvalSymlinks(config.FFmpeg)
	if err != nil {
		return nil, fmt.Errorf("encoder: resolve ffmpeg: %w", err)
	}
	info, err := os.Stat(binary)
	if err != nil {
		return nil, fmt.Errorf("encoder: stat ffmpeg: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return nil, errors.New("encoder: ffmpeg must be an executable regular file")
	}
	if config.Output == "" {
		return nil, errors.New("encoder: output is required")
	}
	config.Output, err = filepath.Abs(config.Output)
	if err != nil {
		return nil, fmt.Errorf("encoder: resolve output: %w", err)
	}
	if !config.Overwrite {
		if _, err := os.Lstat(config.Output); err == nil {
			return nil, fmt.Errorf("encoder: output exists: %w", os.ErrExist)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("encoder: inspect output: %w", err)
		}
	}
	file, err := os.CreateTemp(filepath.Dir(config.Output), "."+filepath.Base(config.Output)+"-*.mp4")
	if err != nil {
		return nil, fmt.Errorf("encoder: temporary output: %w", err)
	}
	temporary := file.Name()
	if err := file.Close(); err != nil {
		return nil, errors.Join(fmt.Errorf("encoder: close temporary output: %w", err), os.Remove(temporary))
	}
	childCtx, cancel := context.WithCancel(ctx)
	e := &Encoder{ctx: ctx, cancel: cancel, config: config, temporary: temporary}
	e.command = command(childCtx, binary, arguments(config, temporary)...)
	e.command.Stderr = &e.stderr
	e.command.WaitDelay = 2 * time.Second
	e.stdin, err = e.command.StdinPipe()
	if err == nil {
		err = e.command.Start()
	}
	if err != nil {
		cancel()
		if e.stdin != nil {
			_ = e.stdin.Close()
		}
		return nil, errors.Join(fmt.Errorf("encoder: start ffmpeg: %w", err), os.Remove(temporary))
	}
	return e, nil
}

// arguments builds the ffmpeg command line: raw RGBA frames on standard input
// at c.FPS, padded with black to a height that sixteen divides, encoded as HEVC
// with VideoToolbox into an MP4 at temporary. The bitrate is higher above 2048
// pixels wide.
func arguments(c Config, temporary string) []string {
	height := (c.Height + 15) / 16 * 16
	bitrate, maxrate, buffer := "20M", "30M", "40M"
	if c.Width > 2048 {
		bitrate, maxrate, buffer = "60M", "90M", "120M"
	}
	return []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-f", "rawvideo", "-pixel_format", "rgba", "-video_size", fmt.Sprintf("%dx%d", c.Width, c.Height),
		"-framerate", strconv.FormatFloat(c.FPS, 'f', -1, 64), "-i", "pipe:0", "-map", "0:v:0", "-an", "-frames:v", strconv.Itoa(c.FrameCount),
		"-vf", fmt.Sprintf("pad=%d:%d:0:%d:black,scale=iw:ih:out_color_matrix=bt709:out_range=tv,format=yuv420p", c.Width, height, (height-c.Height)/2),
		"-c:v", "hevc_videotoolbox", "-profile:v", "main", "-tag:v", "hvc1",
		"-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709", "-color_range", "tv",
		"-b:v", bitrate, "-maxrate", maxrate, "-bufsize", buffer, "-g", "48", "-movflags", "+faststart", "-f", "mp4", temporary,
	}
}

// WriteFrame sends frame number index to ffmpeg and returns the time spent
// writing. The index must be the next one expected and below the frame count,
// and the frame must match the configured size. Any failure is kept, returned
// to later calls, and stops ffmpeg. It is refused after Close.
func (e *Encoder) WriteFrame(index int, frame *image.RGBA) (time.Duration, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return 0, errors.New("encoder: write after close")
	}
	if e.failure != nil {
		return 0, e.failure
	}
	fail := func(err error) (time.Duration, error) { e.failure = err; e.cancel(); return 0, err }
	if err := e.ctx.Err(); err != nil {
		return fail(fmt.Errorf("encoder: frame %d: %w", index, err))
	}
	if index != e.stats.Frames || index >= e.config.FrameCount {
		return fail(fmt.Errorf("encoder: frame %d: expected frame %d within count %d", index, e.stats.Frames, e.config.FrameCount))
	}
	width, height := e.config.Width, e.config.Height
	if frame == nil || frame.Rect.Max.X < frame.Rect.Min.X || frame.Rect.Max.Y < frame.Rect.Min.Y || frame.Rect.Dx() != width || frame.Rect.Dy() != height {
		return fail(fmt.Errorf("encoder: frame %d: RGBA dimensions mismatch", index))
	}
	row := width * 4
	if frame.Stride < row || len(frame.Pix) < row || (height > 1 && frame.Stride > (len(frame.Pix)-row)/(height-1)) {
		return fail(fmt.Errorf("encoder: frame %d: invalid RGBA stride or pixel length", index))
	}
	start := time.Now()
	var writeErr error
	for y := 0; y < height; y++ {
		n, err := e.stdin.Write(frame.Pix[y*frame.Stride : y*frame.Stride+row])
		e.stats.Bytes += int64(n)
		if err != nil {
			writeErr = err
			break
		}
		if n != row {
			writeErr = io.ErrShortWrite
			break
		}
	}
	elapsed := time.Since(start)
	e.stats.WriteTime += elapsed
	if writeErr == nil {
		writeErr = e.ctx.Err()
	}
	if writeErr != nil {
		e.failure = fmt.Errorf("encoder: frame %d: pipe write: %w", index, errors.Join(writeErr, e.ctx.Err()))
		e.cancel()
		return elapsed, e.failure
	}
	e.stats.Frames++
	return elapsed, nil
}

// Stats returns the frames written, the bytes sent and the time spent writing
// so far.
func (e *Encoder) Stats() Stats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats
}

// Close finishes the movie: it ends ffmpeg's input, waits for it, and
// publishes the output only if every frame arrived and ffmpeg succeeded.
func (e *Encoder) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.finish(false)
}

// Abort stops ffmpeg and discards the temporary file without publishing the
// output. It returns only an error from removing the file, and nothing if the
// encoder is already closed.
func (e *Encoder) Abort() error {
	e.publishMu.Lock()
	e.aborted = true
	e.cancel()
	e.publishMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	return e.finish(true)
}

// finish ends ffmpeg's input and waits for it, for at most the finish timeout.
// With no failure and every frame received it puts the temporary file at the
// output, by rename if Overwrite is set and by hard link otherwise. It always
// removes the temporary file. It runs once and later calls return the saved
// result. The caller holds e.mu.
func (e *Encoder) finish(abort bool) error {
	if e.closed {
		return e.result
	}
	e.closed = true
	if abort {
		e.failure = fmt.Errorf("encoder: aborted: %w", context.Canceled)
	}
	if e.stats.Frames != e.config.FrameCount {
		e.failure = errors.Join(e.failure, fmt.Errorf("encoder: received %d of %d frames", e.stats.Frames, e.config.FrameCount))
	}
	if e.failure != nil {
		e.cancel()
	}
	closeErr := e.stdin.Close()
	waited := make(chan error, 1)
	go func() { waited <- e.command.Wait() }()
	timer := time.NewTimer(e.config.FinalizeTimeout)
	var waitErr, timeoutErr error
	select {
	case waitErr = <-waited:
	case <-timer.C:
		timeoutErr = fmt.Errorf("encoder: finalization exceeded %s: %w", e.config.FinalizeTimeout, context.DeadlineExceeded)
		e.cancel()
		waitErr = <-waited
	}
	timer.Stop()
	e.cancel()
	failure := errors.Join(e.failure, e.ctx.Err(), closeErr, timeoutErr)
	if waitErr != nil {
		failure = errors.Join(failure, fmt.Errorf("encoder: ffmpeg: %w: %s", waitErr, e.stderr.data))
	}
	if failure == nil {
		info, err := os.Lstat(e.temporary)
		if err != nil {
			failure = fmt.Errorf("encoder: inspect encoded output: %w", err)
		} else if !info.Mode().IsRegular() || info.Size() == 0 {
			failure = errors.New("encoder: empty or nonregular encoded output")
		}
	}
	e.publishMu.Lock()
	if e.aborted {
		failure = errors.Join(failure, fmt.Errorf("encoder: aborted: %w", context.Canceled))
	}
	failure = errors.Join(failure, e.ctx.Err())
	if failure == nil {
		var err error
		if e.config.Overwrite {
			err = os.Rename(e.temporary, e.config.Output)
		} else {
			err = os.Link(e.temporary, e.config.Output)
		}
		if err != nil {
			failure = fmt.Errorf("encoder: publish output: %w", err)
		}
	}
	e.publishMu.Unlock()
	var cleanupErr error
	if err := os.Remove(e.temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
		cleanupErr = fmt.Errorf("encoder: remove temporary: %w", err)
	}
	e.result = errors.Join(failure, cleanupErr)
	if abort {
		return cleanupErr
	}
	return e.result
}
