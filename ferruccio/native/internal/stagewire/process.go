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
	"time"
)

// MaxSource limits the unsaved buffer passed to a staging helper.
const MaxSource = 8 << 20

type commandFactory func(context.Context, string, ...string) *exec.Cmd

// Run starts and waits for one helper, with a temporary cache inside the
// configured cache and the requested shelf. Cancellation kills and reaps it.
func Run(ctx context.Context, helper, path, source, kind, cache, shelf, data string, processors int) (Snapshot, error) {
	return run(ctx, exec.CommandContext, helper, path, source, kind, cache, shelf, data, processors)
}

// run isolates child configuration, stores output on disk, and waits for exit.
func run(ctx context.Context, makeCommand commandFactory, helper, path, source, kind, cache, shelf, dataPath string, processors int) (result Snapshot, resultErr error) {
	if !filepath.IsAbs(cache) {
		return Snapshot{}, fmt.Errorf("staged working view needs a cache folder; choose one in Settings")
	}
	if !filepath.IsAbs(helper) || !filepath.IsAbs(path) || (shelf != "" && !filepath.IsAbs(shelf)) || (dataPath != "" && !filepath.IsAbs(dataPath)) ||
		processors < 1 || processors > 8 || len(source) > MaxSource || (kind != "set" && kind != "legacy-set" && kind != "job") {
		return Snapshot{}, fmt.Errorf("invalid staging helper request or configured paths")
	}
	if err := ctx.Err(); err != nil {
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
	cmd := makeCommand(ctx, helper, "stage", "--path", path, "--kind", kind)
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
	defer out.Close()
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
	snapshot, err := Read(contextReader{ctx: ctx, source: out})
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
