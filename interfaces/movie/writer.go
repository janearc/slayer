package movie

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ErrNeedsMacOS is every call that would read or write a movie, on a
// machine where AVFoundation cannot be reached: one not macOS, or a
// build without cgo.
var ErrNeedsMacOS = errors.New("movie: writing video needs macOS and cgo")

// errAborted is the failure of a writer its caller aborted.
var errAborted = errors.New("movie: aborted")

// pipe is the encoder a writer feeds: AVFoundation through cgo on
// macOS. It was given the whole sound when opened. The writer calls
// frame, finish, cancel and release one at a time; interrupt may come
// from any goroutine at any time, and once released does nothing.
type pipe interface {
	frame(index int, picture *image.RGBA) error
	finish(frames int) error
	cancel()
	interrupt()
	release()
}

// Stats is what a writer has done so far.
type Stats struct {
	frames int
	bytes  int64
	spent  time.Duration
}

// Frames is how many frames were taken.
func (stats Stats) Frames() int { return stats.frames }

// Bytes is the size of the published file; nought until Close is done.
func (stats Stats) Bytes() int64 { return stats.bytes }

// Spent is the time spent in Frame, encoding and waiting to.
func (stats Stats) Spent() time.Duration { return stats.spent }

// Writer takes frames in order and writes them to a movie.
type Writer struct {
	mu        sync.Mutex
	settings  Settings
	temporary string
	pipe      pipe
	ctx       context.Context
	unhook    func() bool
	next      int
	done      bool
	discarded bool
	failure   error
	stats     Stats
}

// New opens a writer for checked settings. The movie is written to a
// hidden file beside its path until Close publishes it. When ctx is
// cancelled the writer stops and its next call aborts it.
func New(ctx context.Context, settings Settings) (*Writer, error) {
	if ctx == nil {
		return nil, errors.New("movie: no context: give one, " +
			"context.Background() if nothing cancels the job")
	}
	if settings.path == "" {
		return nil, errors.New("movie: settings not made by " +
			"NewSettings: check a Request with it first")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("movie: not started: %w", err)
	}
	temporary, err := temporaryBeside(settings.path)
	if err != nil {
		return nil, err
	}
	opened, err := openPipe(settings, temporary)
	if err != nil {
		return nil, errors.Join(err, removeQuietly(temporary))
	}
	return newWriter(ctx, settings, temporary, opened), nil
}

// newWriter is a writer around an opened pipe, stopped when ctx is.
func newWriter(ctx context.Context, settings Settings, temporary string,
	opened pipe) *Writer {
	return &Writer{settings: settings, temporary: temporary, pipe: opened,
		ctx: ctx, unhook: context.AfterFunc(ctx, opened.interrupt)}
}

// temporaryBeside is a name for a hidden file beside the movie's path,
// free when chosen. AVFoundation will not write over a file, so the
// name is reserved and then the file is removed.
func temporaryBeside(path string) (string, error) {
	pattern := "." + filepath.Base(path) + "-*" + extension
	file, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return "", fmt.Errorf("movie: a temporary file beside %s: %w",
			path, err)
	}
	name := file.Name()
	return name, errors.Join(file.Close(), os.Remove(name))
}

// removeQuietly removes a file that may never have been made.
func removeQuietly(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("movie: removing %s: %w", path, err)
}

// Settings is what the writer was opened with.
func (writer *Writer) Settings() Settings { return writer.settings }

// Stats is what the writer has done so far.
func (writer *Writer) Stats() Stats {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.stats
}

// Frame encodes the picture at index, which must be the next: 0, then
// 1, and on to one short of the frame count. The picture is exactly
// the requested size. Anything refused fails the writer and removes
// what it had written; Close then says why.
func (writer *Writer) Frame(index int, picture *image.RGBA) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.done {
		return fmt.Errorf("movie: frame %d after Close or Abort", index)
	}
	if writer.failure != nil {
		return writer.failure
	}
	if err := writer.refusal(index, picture); err != nil {
		return writer.fail(err)
	}
	start := time.Now()
	err := writer.pipe.frame(index, picture)
	writer.stats.spent += time.Since(start)
	if err != nil {
		stopped := errors.Join(err, writer.ctx.Err())
		return writer.fail(stopped)
	}
	writer.next++
	writer.stats.frames++
	return nil
}

// refusal is why a frame cannot be taken, or nil when it can.
func (writer *Writer) refusal(index int, picture *image.RGBA) error {
	if err := writer.ctx.Err(); err != nil {
		return fmt.Errorf("movie: frame %d: stopped: %w", index, err)
	}
	if index >= writer.settings.frames {
		return fmt.Errorf("movie: frame %d is more frames than the "+
			"count of %d", index, writer.settings.frames)
	}
	if index != writer.next {
		return fmt.Errorf("movie: frame %d out of order: frame %d is "+
			"next", index, writer.next)
	}
	return pictureRefusal(index, picture, writer.settings.picture)
}

// pictureRefusal is why a picture is not one frame of that size, or
// nil when it is.
func pictureRefusal(index int, picture *image.RGBA, size image.Point) error {
	if picture == nil {
		return fmt.Errorf("movie: frame %d has no picture", index)
	}
	if picture.Rect.Size() != size {
		return fmt.Errorf("movie: frame %d is %v, the wrong size: the "+
			"movie's frames are %v", index, picture.Rect.Size(), size)
	}
	row := size.X * bytesPerPixel
	needed := (size.Y-one)*picture.Stride + row
	if picture.Stride < row || len(picture.Pix) < needed {
		return fmt.Errorf("movie: frame %d holds %d bytes at a stride of "+
			"%d, too few for its size", index, len(picture.Pix),
			picture.Stride)
	}
	return nil
}

// fail records why the writer failed, throws away what it wrote, and
// gives the reason back.
func (writer *Writer) fail(err error) error {
	writer.failure = err
	if discarded := writer.discard(); discarded != nil {
		writer.failure = errors.Join(err, discarded)
	}
	return writer.failure
}

// discard cancels the encoder and removes the temporary file, once.
func (writer *Writer) discard() error {
	if writer.discarded {
		return nil
	}
	writer.discarded = true
	writer.unhook()
	writer.pipe.cancel()
	writer.pipe.release()
	return errors.Join(removeQuietly(writer.temporary),
		removeLeftovers(writer.temporary))
}

// leftovers marks what AVFoundation leaves beside a file it was stopped
// writing: a directory named after the file, then this. It is what a
// cancelled writer that was then finished leaves on this machine.
const leftovers = ".sb-"

// removeLeftovers removes whatever AVFoundation left beside the
// temporary file when it was stopped.
func removeLeftovers(temporary string) error {
	folder := filepath.Dir(temporary)
	entries, err := os.ReadDir(folder)
	if err != nil {
		return fmt.Errorf("movie: looking for leftovers in %s: %w",
			folder, err)
	}
	prefix := filepath.Base(temporary) + leftovers
	var failures []error
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			path := filepath.Join(folder, entry.Name())
			failures = append(failures, os.RemoveAll(path))
		}
	}
	return errors.Join(failures...)
}

// Close finishes the movie and publishes it at its path. Fewer frames
// than the count, a cancelled context or an earlier refusal is an
// error, and leaves nothing. A second Close says what the first did.
func (writer *Writer) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.done {
		return writer.failure
	}
	writer.done = true
	if writer.failure != nil {
		return writer.failure
	}
	if writer.next < writer.settings.frames {
		return writer.fail(fmt.Errorf("movie: closed after %d of %d "+
			"frames: nothing was published", writer.next,
			writer.settings.frames))
	}
	if err := writer.ctx.Err(); err != nil {
		return writer.fail(fmt.Errorf("movie: stopped: %w", err))
	}
	if err := writer.pipe.finish(writer.settings.frames); err != nil {
		stopped := errors.Join(err, writer.ctx.Err())
		return writer.fail(stopped)
	}
	writer.unhook()
	writer.pipe.release()
	writer.discarded = true
	if err := writer.publish(); err != nil {
		writer.failure = errors.Join(err, removeQuietly(writer.temporary))
	}
	return writer.failure
}

// publish moves the finished file to its path: renamed over what is
// there when asked to replace it, and otherwise linked, so a file that
// appeared since the settings were checked is not lost.
func (writer *Writer) publish() error {
	path := writer.settings.path
	if writer.settings.replace {
		if err := os.Rename(writer.temporary, path); err != nil {
			return fmt.Errorf("movie: publishing %s: %w", path, err)
		}
	} else {
		if err := os.Link(writer.temporary, path); err != nil {
			return fmt.Errorf("movie: publishing %s: %w", path, err)
		}
		if err := os.Remove(writer.temporary); err != nil {
			return fmt.Errorf("movie: published %s, but: %w", path, err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("movie: published %s, but: %w", path, err)
	}
	writer.stats.bytes = info.Size()
	return nil
}

// Abort stops the writer and removes everything it wrote. It may be
// called from any goroutine, and after Close does nothing.
func (writer *Writer) Abort() error {
	writer.pipe.interrupt()
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.done {
		return nil
	}
	writer.done = true
	if writer.failure == nil {
		writer.failure = errAborted
	}
	return writer.discard()
}
