package progress

import (
	"fmt"
	"time"
)

// A still's draw says how far it is as a line every StillEvery, whichever
// tracer draws it: the processor's rows kept by door, or the GPU's bands.
// The line and its interval are made here and nowhere else.

// StillEvery is the time between two progress lines of a still's draw.
// Picked: a line a quarter minute is enough to see a draw move without
// filling a log over an hours-long draw.
const StillEvery = 15 * time.Second

// StillLine is a still's progress as words of key=value, as Plain writes
// a meter's:
//
//	progress rows=412/900 taken=300 elapsed=4m10s remaining~5m2s
//
// done is the rows finished of all, taken those of them taken up from an
// earlier draw, and elapsed the time this draw has taken. The time left
// is at the pace of the rows this draw drew, offered as a Meter offers
// it: not before MinElapsed has passed and MinRows rows are drawn.
func StillLine(done, all, taken int, elapsed time.Duration) string {
	elapsed = max(elapsed, none)
	line := fmt.Sprintf("progress rows=%d/%d taken=%d elapsed=%s", done,
		all, taken, elapsed.Round(time.Second))
	drawn := done - taken
	if elapsed < MinElapsed || drawn < MinRows {
		return line
	}
	left := scaled(elapsed, float64(all-done)/float64(drawn))
	return line + fmt.Sprintf(" remaining~%s", left.Round(time.Second))
}
