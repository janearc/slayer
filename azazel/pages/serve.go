package pages

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// The coordinator's two files in the queue folder: its lock, which makes
// it the only one, and its own log, which it alone writes.
const (
	lockName = "pages.lock" // defined: the one coordinator's lock
	LogName  = "pages.log"  // defined: the coordinator's own log
)

// private is the mode of the coordinator's files in the queue folder.
const private = 0o600 // defined: the queue folder's own files are private

// Ledger is the queue as the coordinator reads it: the snapshot
// `azazel status` prints. queue.Store is one; its Status opens the
// database read-only, so a publisher's read never writes the store.
type Ledger interface {
	Status(context.Context) (queue.Snapshot, error)
}

// lock takes the one coordinator's lock in the queue folder and writes
// who holds it there, or says who already does.
func lock(queueRoot string, now time.Time) (*os.File, error) {
	path := filepath.Join(queueRoot, lockName)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW,
		private)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		holder, _ := os.ReadFile(path)
		return nil, errors.Join(fmt.Errorf("pages is already served for "+
			"this queue, by %s", strings.TrimSpace(string(holder))), file.Close())
	}
	if err == nil {
		err = file.Truncate(0)
	}
	if err == nil {
		_, err = fmt.Fprintf(file, "process %d, since %s\n", os.Getpid(),
			now.Format(time.RFC3339))
	}
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file, nil
}

// OpenLog opens the coordinator's own log in the queue folder, to add to.
func OpenLog(queueRoot string) (*os.File, error) {
	return os.OpenFile(filepath.Join(queueRoot, LogName),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE|syscall.O_NOFOLLOW, private)
}

// Serve is the one coordinator for a queue: it takes the queue folder's
// lock, or returns why it cannot, then reads the queue's ledger every
// tick and publishes for every job running or just finished, until the
// context ends. On a restart it rebuilds everything from the disc and the
// ledger alone. What it has to say goes to log.
func Serve(ctx context.Context, queueRoot string, settings Settings,
	log io.Writer) error {
	held, err := lock(queueRoot, settings.Now())
	if err != nil {
		return err
	}
	defer held.Close()
	New(settings, log).serve(ctx, queue.At(queueRoot))
	return nil
}

// serve looks at the ledger at once and then every tick, until the
// context ends, and waits for any film still being made.
func (c *Coordinator) serve(ctx context.Context, ledger Ledger) {
	ticker := time.NewTicker(c.every)
	defer ticker.Stop()
	defer c.encoding.Wait()
	defer func() {
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		c.look(flush, ledger)
	}()
	for {
		c.look(ctx, ledger)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// justFinished is how far back, on its first look, the coordinator takes
// a finished job as one whose end it must still publish.
// picked: barkeep's pace window; an end older than that has no line left
// to finish, and a coordinator is not down that long unnoticed.
const justFinished = paceWindow * time.Second

// look reads the ledger once, follows each job, and publishes.
func (c *Coordinator) look(ctx context.Context, ledger Ledger) {
	snapshot, err := ledger.Status(ctx)
	if err != nil {
		c.note("the queue could not be read: " + err.Error())
		return
	}
	first := !c.looked
	c.looked = true
	for _, job := range snapshot.Jobs {
		c.follow(ctx, job, first)
	}
	c.tick(ctx)
}

// follow starts a job the ledger shows running and ends one seen running
// that no longer is. Every look also catches recent jobs that completed
// between polls. Remembering each end prevents repeated publication.
func (c *Coordinator) follow(ctx context.Context, job queue.Job, _ bool) {
	c.state.Lock()
	_, known := c.jobs[job.Sequence]
	passed := c.passed[job.Sequence]
	ended, seen := c.ended[job.Sequence]
	retry := c.retryEnds[job.Sequence]
	c.state.Unlock()
	if passed {
		return
	}
	running := job.State == queue.Running
	if !running && seen && ended.Equal(job.Finished) {
		return
	}
	recent := !job.Finished.IsZero() &&
		c.publisher.now().Sub(job.Finished) <= justFinished
	var err error
	switch {
	case running && !known:
		c.Start(job)
	case !running && known:
		err = c.end(ctx, job, job.Result)
	case !running && (recent || retry):
		c.Start(job)
		err = c.end(ctx, job, job.Result)
	default:
		return
	}
	if !running {
		c.state.Lock()
		if err == nil {
			c.ended[job.Sequence] = job.Finished
			delete(c.retryEnds, job.Sequence)
		} else {
			c.retryEnds[job.Sequence] = true
		}
		c.state.Unlock()
	}
}
