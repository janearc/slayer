package progress

import (
	"strings"
	"testing"
	"time"
)

// a still's line says the rows finished of all, those taken up, the time
// taken, and the time left at the pace of the rows drawn: 20 of 40 rows
// with 10 taken up, in 20 s, leave 40 s.
func TestAStillLineSaysHowFarAndHowLong(t *testing.T) {
	got := StillLine(20, 40, 10, 20*time.Second)
	want := "progress rows=20/40 taken=10 elapsed=20s remaining~40s"
	if got != want {
		t.Errorf("the line is %q; want %q", got, want)
	}
}

// before MinRows rows are drawn, or MinElapsed has passed, no time left
// is said; rows taken up are not drawn.
func TestAStillLineSaysNoTimeLeftTooSoon(t *testing.T) {
	for name, line := range map[string]string{
		"too few drawn": StillLine(MinRows+3, 100, 4, time.Minute),
		"too soon":      StillLine(50, 100, 0, MinElapsed/2),
	} {
		if strings.Contains(line, "remaining") {
			t.Errorf("%s: the line is %q; want no time left", name, line)
		}
	}
}

// a pace that would overflow a duration says the most one holds, not a
// negative time.
func TestAStillLineNeverSaysANegativeTime(t *testing.T) {
	line := StillLine(MinRows, 1<<62, 0, time.Hour)
	if strings.Contains(line, "remaining~-") {
		t.Errorf("the line is %q; want no negative time left", line)
	}
}
