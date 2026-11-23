package session

import (
	"testing"
)

func TestTallyReportsSingleAndAbsentOptionalCounts(t *testing.T) {
	for _, check := range []struct {
		name string
		made tally
		want string
	}{
		{"empty", tally{}, "0 things of 0 faces, lit by 0 lamps"},
		{"one-spark", tally{sparks: 1}, "0 things of 0 faces, lit by 0 lamps, 1 sparks"},
		{"one-burning", tally{burning: 1}, "0 things of 0 faces, lit by 0 lamps, 1 alight"},
		{"one-unseen", tally{unseen: 1}, "0 things of 0 faces, lit by 0 lamps; 1 more shine where no eye sees, and light nothing in this picture"},
		{"all-kept", tally{things: 3, faces: 12, kept: 12, lamps: 2}, "3 things of 12 faces, lit by 2 lamps"},
		{"partly-kept", tally{things: 3, faces: 12, kept: 8, lamps: 2, sparks: 2, burning: 2}, "3 things of 12 faces (8 kept), lit by 2 lamps, 2 sparks, 2 alight"},
	} {
		t.Run(check.name, func(t *testing.T) {
			if got := check.made.String(); got != check.want {
				t.Fatalf("actual summary %q, want %q", got, check.want)
			}
		})
	}
}
