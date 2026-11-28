package preview

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

// Azazel publishes its scheduling time about once a second. Allow several
// ticks for the next write before attempting to start a replacement.
const workerRetry = 5 * time.Second
const workerLog = "azazel-worker.log"

// workerDispatch tracks a drain process started for an observed request.
// The queue lock belongs to Azazel. This observer never signals the process.
type workerDispatch struct {
	done      <-chan error
	retryAt   time.Time
	lastError error
}

// workerNeeded requires a verified pending job and an unpaused queue.
func workerNeeded(status observedJob, now time.Time) bool {
	if !status.hasSnapshot || status.paused != "" {
		return false
	}
	if status.job.State != "queued" && status.job.State != "running" {
		return false
	}
	age := now.Sub(status.lastSchedule)
	return age > workerRetry || age < -workerRetry
}

// update starts a drain only when the scheduler has not been heard from.
func (worker *workerDispatch) update(tools Tools, plan *Plan, status observedJob, now time.Time) string {
	if !tools.ManageWorker || !workerNeeded(status, now) {
		return ""
	}
	worker.collect(now)
	if worker.done == nil && !now.Before(worker.retryAt) {
		worker.retryAt = now.Add(workerRetry)
		worker.done, worker.lastError = startQueueWorker(tools, plan.Directory)
	}
	if worker.lastError != nil {
		return fmt.Sprintf("Azazel worker failed: %v · retrying", worker.lastError)
	}
	if worker.done != nil && !now.Before(worker.retryAt) {
		return "Azazel worker has not reported activity · see " + workerLog
	}
	return "Starting Azazel worker"
}

// collect observes termination without waiting for the drain to end.
func (worker *workerDispatch) collect(now time.Time) {
	if worker.done == nil {
		return
	}
	select {
	case err := <-worker.done:
		worker.done = nil
		worker.lastError = err
		if err != nil {
			worker.retryAt = now.Add(workerRetry)
		}
	default:
	}
}

// startQueueWorker gives Azazel an independent lifetime. A regular file
// remains writable when Ferruccio quits while the queue still has work.
func startQueueWorker(tools Tools, directory string) (<-chan error, error) {
	log, err := os.OpenFile(filepath.Join(directory, workerLog), os.O_RDWR|os.O_CREATE|os.O_APPEND|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	offset, err := log.Seek(0, io.SeekEnd)
	if err != nil {
		log.Close()
		return nil, err
	}
	command := exec.Command(tools.Azazel, "work", "--queue", tools.Queue)
	command.Env = slices.DeleteFunc(ChildEnvironment(), func(entry string) bool {
		return strings.HasPrefix(entry, "GAME_LANES=")
	})
	command.Stdout, command.Stderr = log, log
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		log.Close()
		return nil, err
	}
	done := make(chan error, 1)
	go func() {
		defer log.Close()
		if err := command.Wait(); err != nil {
			done <- workerFailure(log, offset, err)
			return
		}
		done <- nil
	}()
	return done, nil
}

// workerFailure displays a bounded diagnostic from this startup attempt.
func workerFailure(log *os.File, offset int64, cause error) error {
	const maximumDiagnosticBytes = 512
	data, _ := io.ReadAll(io.NewSectionReader(log, offset, maximumDiagnosticBytes))
	if line, _, _ := strings.Cut(strings.TrimSpace(string(data)), "\n"); line != "" {
		return fmt.Errorf("%w: %s", cause, strings.ToValidUTF8(line, ""))
	}
	return fmt.Errorf("%w (see %s)", cause, workerLog)
}
