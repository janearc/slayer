package progress

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Plain is the meter's state as one line for a pipe or an agent, as words
// of key=value that keep their order and spelling:
//
//	progress frame=12/96 rows=890/2160 share=0.41 elapsed=12s remaining~17s
//
// The share is cut down to two places, not rounded, so it does not say 1.00
// before the job is done. The remaining word is left out while the meter
// does not offer the time. There are no escape bytes and no carriage
// returns, and the line has no newline: the caller ends it.
func Plain(meter *Meter) string {
	words := []string{
		"progress",
		fmt.Sprintf("frame=%d/%d", meter.Frame(), meter.Frames()),
		fmt.Sprintf("rows=%d/%d", meter.RowsDone(), meter.RowsTotal()),
		fmt.Sprintf("share=%.2f", math.Floor(meter.Share()*percentOf)/
			percentOf),
		"elapsed=" + whole(meter.Elapsed()),
	}
	if left, offered := meter.Remaining(); offered {
		words = append(words, "remaining~"+whole(left))
	}
	return strings.Join(words, " ")
}

// whole is a duration to the second, as Go writes one: 12s, 1m5s.
func whole(d time.Duration) string {
	return d.Round(time.Second).String()
}

// Human is the meter's state as a short line for a status row:
//
//	frame 12 of 96, 41%, about 17 s left
//
// The percentage is cut down, as Plain's share is. The time left is left
// out while it is not offered, and a finished job says done.
func Human(meter *Meter) string {
	line := fmt.Sprintf("frame %d of %d, %d%%", meter.Frame(),
		meter.Frames(), int(meter.Share()*percentOf))
	left, offered := meter.Remaining()
	switch {
	case offered && meter.FramesDone() == meter.Frames():
		return line + ", done"
	case offered:
		return line + ", about " + spoken(left) + " left"
	}
	return line
}

// spoken is a duration in words a status row holds: 17 s, 2 min 5 s.
func spoken(d time.Duration) string {
	seconds := int(math.Round(d.Seconds()))
	if seconds < secondsInMinute {
		return fmt.Sprintf("%d s", seconds)
	}
	return fmt.Sprintf("%d min %d s", seconds/secondsInMinute,
		seconds%secondsInMinute)
}
