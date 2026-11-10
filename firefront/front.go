// Package firefront schedules prescribed ignition over fuel locations.
// It does not predict wildfire spread or solve coupled atmosphere and fuel.
package firefront

import (
	"errors"
	"github.com/janearc/slayer/maths"
	"math"
	"slices"
	"sort"
	"time"
)

// Spec describes a plane moving horizontally at an authored speed in m/s.
// Cell is the spatial grouping width in metres, not a fuel or spread parameter.
type Spec struct {
	Origin, Direction maths.Vec
	Speed, Cell       float64
	Start, Duration   time.Duration
}

// Target is one fuel location. Seed identifies its repeatable flame pattern.
type Target struct {
	At   maths.Vec
	Seed uint64
}

// Event describes a prescribed ignition; Age is filled by Active.
type Event struct {
	At                   maths.Vec
	Seed                 uint64
	Start, Duration, Age time.Duration
}

// Schedule supplies immutable ignition groups, live targets and peak budgets.
// New returns an implementation of this contract.
type Schedule interface {
	Groups() [][]Event
	Active(time.Duration) []Event
	PeakGroups() int
	PeakTargets(time.Duration) int
}

type plan struct {
	groups [][]Event
	peak   int
}
type tile struct{ x, z int64 }
type crossing struct {
	at    time.Duration
	delta int
}

// finite refuses values that cannot identify a distance or time.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// pointOK checks every component before spatial grouping.
func pointOK(p maths.Vec) bool { return finite(p.X) && finite(p.Y) && finite(p.Z) }

// New copies targets into deterministic spatial groups of at most 64 sources.
// PeakGroups conservatively budgets overlapping group windows, even when no
// individual patch happens to be alight during a gap in its group's window.
func New(spec Spec, targets []Target) (*plan, error) {
	refuse := errors.New("firefront: invalid prescribed ignition plan")
	if !pointOK(spec.Origin) || !pointOK(spec.Direction) ||
		spec.Direction.Y != 0 || !(spec.Direction.Len() > 0) || !finite(spec.Direction.Len()) ||
		!(spec.Speed > 0) || !finite(spec.Speed) || !(spec.Cell > 0) || !finite(spec.Cell) ||
		spec.Start < -24*time.Hour || spec.Start > 24*time.Hour || spec.Duration <= 0 || spec.Duration > 24*time.Hour || len(targets) == 0 || len(targets) > 100000 {
		return nil, refuse
	}
	way := spec.Direction.Unit()
	tiles := map[tile][]Event{}
	for _, target := range targets {
		if !pointOK(target.At) {
			return nil, refuse
		}
		offset := target.At.Sub(spec.Origin)
		seconds := offset.Dot(way) / spec.Speed
		x, z := math.Floor(offset.X/spec.Cell), math.Floor(offset.Z/spec.Cell)
		// Bound conversion before it reaches integer time or tiles.
		if !finite(seconds) || math.Abs(seconds) > 86400 || !finite(x) || !finite(z) || math.Abs(x) > 1e9 || math.Abs(z) > 1e9 {
			return nil, refuse
		}
		when := spec.Start + time.Duration(seconds*float64(time.Second))
		key := tile{int64(x), int64(z)}
		tiles[key] = append(tiles[key], Event{At: target.At, Seed: target.Seed, Start: when, Duration: spec.Duration})
	}
	keys := make([]tile, 0, len(tiles))
	for key := range tiles {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].x != keys[j].x {
			return keys[i].x < keys[j].x
		}
		return keys[i].z < keys[j].z
	})
	made := &plan{}
	var edges []crossing
	for _, key := range keys {
		targets := tiles[key]
		// Stable order keeps each seed's neighbours on replay.
		for start := 0; start < len(targets); start += 64 {
			group := slices.Clone(targets[start:min(start+64, len(targets))])
			low, high := group[0].Start, group[0].Start+group[0].Duration
			for _, event := range group {
				low = min(low, event.Start)
				high = max(high, event.Start+event.Duration)
			}
			made.groups = append(made.groups, group)
			edges = append(edges, crossing{low, 1}, crossing{high, -1})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].at != edges[j].at {
			return edges[i].at < edges[j].at
		}
		return edges[i].delta < edges[j].delta
	})
	active := 0
	for _, edge := range edges {
		active += edge.delta
		made.peak = max(made.peak, active)
	}
	return made, nil
}

// Groups returns copied schedules; changing them cannot alter the plan.
func (made *plan) Groups() [][]Event {
	groups := make([][]Event, len(made.groups))
	for i, group := range made.groups {
		groups[i] = slices.Clone(group)
	}
	return groups
}

// Active returns fuel in its half-open burn window and group order.
func (made *plan) Active(at time.Duration) []Event {
	var active []Event
	for _, group := range made.groups {
		for _, event := range group {
			if at >= event.Start && at < event.Start+event.Duration {
				age := at - event.Start
				event.Age = age
				active = append(active, event)
			}
		}
	}
	return active
}

// PeakGroups is a conservative simultaneous volume count for this plan.
func (made *plan) PeakGroups() int { return made.peak }

// PeakTargets budgets live fuel plus a nonnegative particle afterlife.
func (made *plan) PeakTargets(tail time.Duration) int {
	tail = max(0, tail)
	var edges []crossing
	for _, group := range made.groups {
		for _, event := range group {
			end := event.Start + event.Duration
			// Saturate long afterlives so their end cannot wrap
			// before birth and understate the particle budget.
			if end > time.Duration(math.MaxInt64)-tail {
				end = time.Duration(math.MaxInt64)
			} else {
				end += tail
			}
			edges = append(edges, crossing{event.Start, 1}, crossing{end, -1})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].at != edges[j].at {
			return edges[i].at < edges[j].at
		}
		return edges[i].delta < edges[j].delta
	})
	active, peak := 0, 0
	for _, edge := range edges {
		active += edge.delta
		peak = max(peak, active)
	}
	return peak
}
