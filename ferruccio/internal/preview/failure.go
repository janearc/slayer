package preview

import (
	"fmt"
	"path/filepath"
	"unicode/utf8"
)

// jobFailure reports only this verified job, with bounded owner diagnostics.
func jobFailure(tools Tools, job queueJob) error {
	reason := job.Result.Reason
	const maximumReasonBytes = 1024
	if len(reason) > maximumReasonBytes {
		end := maximumReasonBytes
		for end > 0 && !utf8.RuneStart(reason[end]) {
			end--
		}
		reason = reason[:end] + "…"
	}
	detail := fmt.Sprintf("Azazel job %d %s (exit %d): %s", job.Sequence, job.State, job.Result.ExitCode, reason)
	if job.Attempt > 0 {
		logName := fmt.Sprintf("%020d.%04d.log", job.Sequence, job.Attempt)
		detail += "; attempt log: " + filepath.Join(tools.Queue, logName)
	}
	return fmt.Errorf("%s", detail)
}
