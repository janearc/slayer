package progress

import "time"

// Due says whether to write a line now. last is when a line was last
// written, zero if none was. A line that reports a frame's end is always
// due. So is the first line, and any line when the clock has gone back past
// last, since waiting out a backwards clock would be silence. Otherwise a
// line is due when Interval has passed.
func Due(last, now time.Time, frameEnd bool) bool {
	if frameEnd || last.IsZero() || now.Before(last) {
		return true
	}
	return now.Sub(last) >= Interval
}
