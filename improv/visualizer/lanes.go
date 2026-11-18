package main

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/janearc/slayer/cheeptune"
)

// percussion is the role whose events are hits, drawn a row a hit.
const percussion = "perc"

// pitchedWeight is a pitched lane's share of the lanes' height, in rows:
// a pitched lane is as tall as a percussion lane of this many hits.
const pitchedWeight = 2

// hitOrder is the order percussion rows are drawn in, top to bottom, by
// the score's hit names; hits not named here follow in the order heard.
var hitOrder = []string{"k", "s", "h", "o", "t"}

// hitNames are the words a row is labelled with, by the score's hit name.
var hitNames = map[string]string{
	"k": "kick", "s": "snare", "h": "hat", "o": "open hat", "t": "tom",
}

// lane is one instrument: its name, what it plays, and how it is laid
// out. A percussion lane has hits, one row each; a pitched lane has a
// pitch range instead.
type lane struct {
	name    string
	events  []cheeptune.Event
	hits    []string
	low     int
	high    int
	loudest float64
}

// weight is the lane's share of the lanes' height.
func (l lane) weight() int {
	if len(l.hits) > 0 {
		return len(l.hits)
	}
	return pitchedWeight
}

// row is the row a hit is drawn on.
func (l lane) row(hit string) int { return slices.Index(l.hits, hit) }

// readLanes reads each NAME=SCORE#ROLE spec, and returns the lanes and
// the first score, whose tempo and meter every other score must share.
func readLanes(specs []string) ([]lane, *cheeptune.Score, error) {
	var lanes []lane
	var first *cheeptune.Score
	for _, spec := range specs {
		name, path, role, err := splitSpec(spec)
		if err != nil {
			return nil, nil, err
		}
		score, err := cheeptune.ReadScore(path)
		if err != nil {
			return nil, nil, fmt.Errorf("lane %s: %w", name, err)
		}
		if first == nil {
			first = score
		}
		if !sameTime(first, score) {
			return nil, nil, fmt.Errorf("lane %s: tempo or meter differs "+
				"from the first lane's score", name)
		}
		made, err := newLane(name, score, role)
		if err != nil {
			return nil, nil, err
		}
		lanes = append(lanes, made)
	}
	return lanes, first, nil
}

// splitSpec takes NAME=SCORE#ROLE apart, the role after the last "#".
func splitSpec(spec string) (name, path, role string, err error) {
	name, rest, found := strings.Cut(spec, "=")
	at := strings.LastIndex(rest, "#")
	if !found || name == "" || at < 1 || at == len(rest)-1 {
		return "", "", "", fmt.Errorf("lane %q: want NAME=SCORE#ROLE", spec)
	}
	return name, rest[:at], rest[at+1:], nil
}

// sameTime is whether two scores keep the same tempo, tempo changes and
// meter, so one clock serves both.
func sameTime(a, b *cheeptune.Score) bool {
	return a.Tempo == b.Tempo && a.Meter == b.Meter &&
		slices.Equal(a.Tempos, b.Tempos)
}

// newLane finds the role's part in the score and lays it out.
func newLane(name string, score *cheeptune.Score, role string) (lane,
	error) {
	index := slices.IndexFunc(score.Parts, func(p cheeptune.Part) bool {
		return p.Role == role
	})
	if index < 0 {
		return lane{}, fmt.Errorf("lane %s: the score has no %s part",
			name, role)
	}
	made := lane{name: name, events: score.Parts[index].Events()}
	if len(made.events) == 0 {
		return lane{}, fmt.Errorf("lane %s: the %s part is all rests",
			name, role)
	}
	for _, e := range made.events {
		made.loudest = math.Max(made.loudest, e.Loud)
	}
	if role == percussion {
		made.hits = rows(made.events)
		return made, nil
	}
	made.low, made.high = pitchRange(made.events)
	return made, nil
}

// rows is the hits heard, in hitOrder and then in the order heard.
func rows(events []cheeptune.Event) []string {
	var heard []string
	for _, e := range events {
		if !slices.Contains(heard, e.Hit) {
			heard = append(heard, e.Hit)
		}
	}
	var out []string
	for _, hit := range hitOrder {
		if slices.Contains(heard, hit) {
			out = append(out, hit)
		}
	}
	for _, hit := range heard {
		if !slices.Contains(out, hit) {
			out = append(out, hit)
		}
	}
	return out
}

// pitchRange is the lowest and highest note played, a semitone apart
// from the lane's edges.
func pitchRange(events []cheeptune.Event) (low, high int) {
	low, high = math.MaxInt, math.MinInt
	for _, e := range events {
		for _, note := range e.Notes {
			low, high = min(low, note), max(high, note)
		}
	}
	return low - 1, high + 1
}

// rowName is the word a percussion row is labelled with.
func rowName(hit string) string {
	if word, ok := hitNames[hit]; ok {
		return word
	}
	return hit
}
