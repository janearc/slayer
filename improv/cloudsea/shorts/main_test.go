package main

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/cheeptune"
)

// barSlack is how near a bar's start must be to its arithmetic. Picked:
// float64 rounding.
const barSlack = 1e-9

// steady is a score of three bars of four at 60 a minute: a second a
// beat, four a bar.
const steady = `{"name": "steady", "tempo": 60, "meter": 4, "loop": false,
 "parts": [{"role": "lead", "bars": ["c4:4", "c4:4", "c4:4"]}]}`

// TestBarStart: the first bar starts at nought and the third eight
// seconds on; a bar before the first or past the last is refused.
func TestBarStart(t *testing.T) {
	score, err := cheeptune.ParseScore([]byte(steady))
	if err != nil {
		t.Fatal(err)
	}
	for bar, want := range map[int]float64{1: 0, 3: 8} {
		got, err := barStart(score, bar)
		if err != nil || math.Abs(got-want) > barSlack {
			t.Errorf("bar %d: %v, %v; want %v", bar, got, err, want)
		}
	}
	for _, bar := range []int{0, 4, -1} {
		if _, err := barStart(score, bar); !errors.Is(err, errAsked) {
			t.Errorf("bar %d was taken: %v", bar, err)
		}
	}
}
