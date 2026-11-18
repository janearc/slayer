// Package footprint observes process-group physical footprint without
// granting admission or controlling render processes.
package footprint

import (
	"fmt"
	"math"
	"slices"
	"time"
)

// MaximumMembers is the enumeration buffer size. A full buffer is refused
// because it could have been truncated.
const MaximumMembers = 4096

// Identity binds a PID to its kernel process group and creation time.
type Identity struct {
	PID, Group        int
	StartSeconds      uint64
	StartMicroseconds uint64
}

// Reading is a single member's physical footprint. LifetimePeak belongs to
// that process's lifetime and may precede this sampling window.
type Reading struct {
	Current      uint64
	LifetimePeak uint64
	PeakKnown    bool
}

// Source separates native operations so incomplete reads can be tested.
type Source interface {
	Group(int) ([]int, error)
	Identity(int) (Identity, error)
	Footprint(int) (Reading, error)
}

// Member binds one footprint reading to its revalidated process identity.
type Member struct {
	Identity Identity
	Reading  Reading
}

// Observation is a bounded sample, not an atomic kernel snapshot. Complete
// means all member reads and pre/post checks succeeded; a process can still
// begin and end entirely between observations or between the checks.
type Observation struct {
	Started, Finished          time.Time
	Complete                   bool
	Reason                     string
	Members                    []Member
	GroupCurrentBytes          uint64
	MemberLifetimePeakSumBytes uint64
	LifetimePeakSumKnown       bool
}

// Sample takes one group observation without changing process or queue state.
// A partial or untrustworthy sum is withheld and Complete is false. Group
// sums are sums of process footprints, not deduplicated system-wide memory.
func Sample(group int, source Source) Observation {
	return sample(group, source, time.Now())
}

// sample reads one group observation that starts at started. A read that
// changes between the checks marks the observation incomplete.
func sample(group int, source Source, started time.Time) (out Observation) {
	out.Started = started
	defer func() { out.Finished = time.Now() }()
	if group <= 0 || source == nil {
		out.Reason = "invalid group or source"
		return
	}
	before, err := checkedGroup(group, source)
	if err != nil {
		out.Reason = err.Error()
		return
	}
	members := make([]Member, 0, len(before))
	var current, peaks uint64
	peaksKnown := true
	for _, pid := range before {
		identity, err := source.Identity(pid)
		if err != nil || !validIdentity(identity, pid, group) {
			out.Reason = "process identity unavailable"
			return
		}
		reading, err := source.Footprint(pid)
		if err != nil || reading.Current == 0 {
			out.Reason = "process footprint unavailable"
			return
		}
		again, err := source.Identity(pid)
		if err != nil || again != identity {
			out.Reason = "process identity changed"
			return
		}
		if reading.Current > math.MaxUint64-current {
			out.Reason = "group footprint overflow"
			return
		}
		current += reading.Current
		if reading.PeakKnown && reading.LifetimePeak < reading.Current {
			reading.PeakKnown, reading.LifetimePeak = false, 0
		}
		if !reading.PeakKnown || reading.LifetimePeak > math.MaxUint64-peaks {
			peaksKnown = false
		} else {
			peaks += reading.LifetimePeak
		}
		members = append(members, Member{Identity: identity, Reading: reading})
	}
	after, err := checkedGroup(group, source)
	if err != nil || !slices.Equal(before, after) {
		out.Reason = "process group changed"
		return
	}
	for _, member := range members {
		again, err := source.Identity(member.Identity.PID)
		if err != nil || again != member.Identity {
			out.Reason = "process identity changed"
			return
		}
	}
	out.Complete, out.Members, out.GroupCurrentBytes = true, members, current
	out.LifetimePeakSumKnown = peaksKnown
	if peaksKnown {
		out.MemberLifetimePeakSumBytes = peaks
	}
	return
}

// checkedGroup lists the process IDs in group, sorted. It refuses an empty,
// full or invalid list, and a repeated process ID.
func checkedGroup(group int, source Source) ([]int, error) {
	pids, err := source.Group(group)
	if err != nil || len(pids) == 0 || len(pids) >= MaximumMembers {
		return nil, fmt.Errorf("process group unavailable or unbounded")
	}
	pids = slices.Clone(pids)
	slices.Sort(pids)
	for i, pid := range pids {
		if pid <= 0 || i > 0 && pids[i-1] == pid {
			return nil, fmt.Errorf("invalid process group membership")
		}
	}
	return pids, nil
}

// validIdentity reports whether identity names pid in group, with a plausible
// start time.
func validIdentity(identity Identity, pid, group int) bool {
	return identity.PID == pid && identity.Group == group && identity.StartSeconds > 0 && identity.StartMicroseconds < 1_000_000
}

// Window tracks observations but cannot prove an unsampled true maximum.
type Window struct {
	Samples, CompleteSamples, Gaps uint64
	PeakObservedGroupBytes         uint64
	MembershipChanged              bool
	last                           []Identity
}

// Observe records a sample and marks gaps and changes in observed members.
func (w *Window) Observe(observation Observation) {
	w.Samples++
	if !observation.Complete {
		w.Gaps++
		return
	}
	w.CompleteSamples++
	w.PeakObservedGroupBytes = max(w.PeakObservedGroupBytes, observation.GroupCurrentBytes)
	identities := make([]Identity, len(observation.Members))
	for i, member := range observation.Members {
		identities[i] = member.Identity
	}
	if w.last != nil && !slices.Equal(w.last, identities) {
		w.MembershipChanged = true
	}
	w.last = identities
}
