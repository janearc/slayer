package stagewire

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MaxSource limits the unsaved buffer passed to a staging helper.
const MaxSource = 8 << 20

// Camera selects an authored viewpoint and an optional working film frame.
type Camera struct {
	Name       string
	Wide       float64
	PathRecipe string
	Frame      int
}

type commandFactory func(context.Context, string, ...string) *exec.Cmd

// Run starts and waits for one helper, with a temporary cache inside the
// configured cache and the requested shelf. Cancellation kills and reaps it.
func Run(ctx context.Context, helper, path, source, kind, cache, shelf, data string, processors int) (Snapshot, error) {
	return run(ctx, exec.CommandContext, helper, path, source, kind, cache, shelf, data, processors)
}

// Session keeps an isolated staging cache for the lifetime of one editor.
// Its calls are serialized, so a cancelled helper is reaped before another
// uses the cache. Closing removes the private cache and all of its contents.
type Session struct {
	mu     sync.Mutex
	caches map[string]string
	closed bool
}

// Run stages with a private cache retained between this session's requests.
func (session *Session) Run(ctx context.Context, helper, path, source, kind, cache, shelf, data string, processors int, camera Camera) (Snapshot, error) {
	return session.run(ctx, exec.CommandContext, helper, path, source, kind, cache, shelf, data, processors, camera, "")
}

// RunDetail requests a single authored placement or stand by its selection ID.
func (session *Session) RunDetail(ctx context.Context, helper, path, source, kind, cache, shelf, data string, processors int, camera Camera, detail string) (Snapshot, error) {
	return session.run(ctx, exec.CommandContext, helper, path, source, kind, cache, shelf, data, processors, camera, detail)
}

func (session *Session) run(ctx context.Context, makeCommand commandFactory, helper, path, source, kind, cache, shelf, data string, processors int, camera Camera, detail string) (Snapshot, error) {
	if err := validateRequest(ctx, helper, path, source, kind, cache, shelf, data, processors); err != nil {
		return Snapshot{}, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return Snapshot{}, fmt.Errorf("staging cache session is closed")
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if session.caches == nil {
		session.caches = make(map[string]string)
	}
	scratch := session.caches[cache]
	if scratch == "" {
		var err error
		scratch, err = os.MkdirTemp(cache, "ferruccio-stage-")
		if err != nil {
			return Snapshot{}, fmt.Errorf("create a temporary staging cache: %w", err)
		}
		session.caches[cache] = scratch
	}
	return runIn(ctx, makeCommand, helper, path, source, kind, scratch, shelf, data, processors, camera, detail)
}

// Close waits for any helper to exit, then removes this editor's caches.
func (session *Session) Close() error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil
	}
	session.closed = true
	var result error
	for _, cache := range session.caches {
		if err := os.RemoveAll(cache); err != nil {
			result = errors.Join(result, fmt.Errorf("remove temporary staging cache: %w", err))
		}
	}
	session.caches = nil
	return result
}

// run isolates child configuration, stores output on disk, and waits for exit.
func run(ctx context.Context, makeCommand commandFactory, helper, path, source, kind, cache, shelf, dataPath string, processors int) (result Snapshot, resultErr error) {
	if err := validateRequest(ctx, helper, path, source, kind, cache, shelf, dataPath, processors); err != nil {
		return Snapshot{}, err
	}
	scratch, err := os.MkdirTemp(cache, "ferruccio-stage-")
	if err != nil {
		return Snapshot{}, fmt.Errorf("create a temporary staging cache: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(scratch); err != nil {
			result = Snapshot{}
			resultErr = errors.Join(resultErr, fmt.Errorf("remove temporary staging cache: %w", err))
		}
	}()
	return runIn(ctx, makeCommand, helper, path, source, kind, scratch, shelf, dataPath, processors, Camera{}, "")
}

func validateRequest(ctx context.Context, helper, path, source, kind, cache, shelf, dataPath string, processors int) error {
	if !filepath.IsAbs(cache) {
		return fmt.Errorf("staged working view needs a cache folder; choose one in Settings")
	}
	if !filepath.IsAbs(helper) || !filepath.IsAbs(path) || (shelf != "" && !filepath.IsAbs(shelf)) || (dataPath != "" && !filepath.IsAbs(dataPath)) ||
		processors < 1 || processors > 8 || len(source) > MaxSource || (kind != "set" && kind != "legacy-set" && kind != "job") {
		return fmt.Errorf("invalid staging helper request or configured paths")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func runIn(ctx context.Context, makeCommand commandFactory, helper, path, source, kind, scratch, shelf, dataPath string, processors int, camera Camera, detail string) (result Snapshot, resultErr error) {
	args := []string{"stage", "--path", path, "--kind", kind}
	if camera.Frame != 0 {
		args = append(args, "--frame", strconv.Itoa(camera.Frame))
	}
	if detail != "" {
		args = append(args, "--detail="+detail)
	}
	if camera.Name != "" {
		args = append(args, "--camera", camera.Name)
	}
	if camera.Wide != 0 {
		args = append(args, "--camera-wide", strconv.FormatFloat(camera.Wide, 'g', -1, 64))
	}
	if camera.PathRecipe != "" {
		args = append(args, "--still-camera", camera.PathRecipe)
	}
	cmd := makeCommand(ctx, helper, args...)
	env := cmd.Environ()
	filtered := make([]string, 0, len(env)+2)
	for _, entry := range env {
		if !strings.HasPrefix(entry, "SLAYER_CACHE=") && !strings.HasPrefix(entry, "SLAYER_SHELF=") && !(dataPath != "" && strings.HasPrefix(entry, "SLAYER_DATA=")) && !strings.HasPrefix(entry, "GOMAXPROCS=") {
			filtered = append(filtered, entry)
		}
	}
	cmd.Env = append(filtered, "SLAYER_CACHE="+scratch, "GOMAXPROCS="+strconv.Itoa(processors))
	if shelf != "" {
		cmd.Env = append(cmd.Env, "SLAYER_SHELF="+shelf)
	}
	if dataPath != "" {
		cmd.Env = append(cmd.Env, "SLAYER_DATA="+dataPath)
	}
	cmd.Stdin = strings.NewReader(source)
	out, err := os.CreateTemp(scratch, "snapshot-*.json")
	if err != nil {
		return Snapshot{}, fmt.Errorf("create staging snapshot: %w", err)
	}
	defer func() {
		if err := errors.Join(out.Close(), os.Remove(out.Name())); err != nil {
			result = Snapshot{}
			resultErr = errors.Join(resultErr, fmt.Errorf("remove temporary staging snapshot: %w", err))
		}
	}()
	diagnostic := &boundedOutput{limit: 8192, discard: true}
	cmd.Stdout, cmd.Stderr, cmd.WaitDelay = out, diagnostic, 2*time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return Snapshot{}, ctx.Err()
		}
		return Snapshot{}, fmt.Errorf("staging helper failed: %w: %s", err, strings.TrimSpace(diagnostic.buffer.String()))
	}
	if _, err := out.Seek(0, 0); err != nil {
		return Snapshot{}, fmt.Errorf("rewind staging snapshot: %w", err)
	}
	info, err := out.Stat()
	if err != nil {
		return Snapshot{}, fmt.Errorf("stat staging snapshot: %w", err)
	}
	snapshot, err := readSized(contextReader{ctx: ctx, source: out}, info.Size())
	if ctx.Err() != nil {
		return Snapshot{}, ctx.Err()
	}
	return snapshot, err
}

// contextReader also makes cancellation effective after the child exits,
// while a large staged snapshot is being decoded from its temporary file.
type contextReader struct {
	ctx    context.Context
	source io.Reader
}

// Read stops decoding promptly when the staging request is canceled.
func (reader contextReader) Read(data []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.source.Read(data)
}

// boundedOutput truncates diagnostic output without interfering with a
// helper that is already reporting.
type boundedOutput struct {
	buffer  bytes.Buffer
	limit   int
	discard bool
}

// Write accepts data within the limit or truncates diagnostics when configured.
func (output *boundedOutput) Write(data []byte) (int, error) {
	room := output.limit - output.buffer.Len()
	if len(data) > room {
		if output.discard {
			_, _ = output.buffer.Write(data[:room])
			return len(data), nil
		}
		return 0, fmt.Errorf("staging helper output exceeds %d bytes", output.limit)
	}
	return output.buffer.Write(data)
}
