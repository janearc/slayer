package footprint

import (
	"errors"
	"math"
	"testing"
)

type fixture struct {
	groups     [][]int
	identities map[int][]Identity
	values     map[int]Reading
	fail       string
}

func (f *fixture) Group(int) ([]int, error) {
	if f.fail == "group" {
		return nil, errors.New("group")
	}
	if len(f.groups) == 0 {
		return nil, nil
	}
	result := f.groups[0]
	if len(f.groups) > 1 {
		f.groups = f.groups[1:]
	}
	return result, nil
}
func (f *fixture) Identity(pid int) (Identity, error) {
	if f.fail == "identity" {
		return Identity{}, errors.New("identity")
	}
	values := f.identities[pid]
	if len(values) == 0 {
		return Identity{}, errors.New("missing")
	}
	result := values[0]
	if len(values) > 1 {
		f.identities[pid] = values[1:]
	}
	return result, nil
}
func (f *fixture) Footprint(pid int) (Reading, error) {
	if f.fail == "footprint" {
		return Reading{}, errors.New("footprint")
	}
	return f.values[pid], nil
}

func cleanFixture() *fixture {
	return &fixture{groups: [][]int{{2, 1}, {1, 2}}, identities: map[int][]Identity{
		1: {{PID: 1, Group: 7, StartSeconds: 1}}, 2: {{PID: 2, Group: 7, StartSeconds: 2}},
	}, values: map[int]Reading{1: {Current: 10, LifetimePeak: 50, PeakKnown: true}, 2: {Current: 20, LifetimePeak: 80, PeakKnown: true}}}
}

func TestSampleSeparatesCurrentGroupSumFromMemberLifetimePeaks(t *testing.T) {
	got := Sample(7, cleanFixture())
	if !got.Complete || got.GroupCurrentBytes != 30 || !got.LifetimePeakSumKnown || got.MemberLifetimePeakSumBytes != 130 || len(got.Members) != 2 || got.Members[0].Identity.PID != 1 {
		t.Fatal(got)
	}
	f := cleanFixture()
	f.values[2] = Reading{Current: 20}
	got = Sample(7, f)
	if !got.Complete || got.GroupCurrentBytes != 30 || got.LifetimePeakSumKnown || got.MemberLifetimePeakSumBytes != 0 {
		t.Fatal(got)
	}
	f = cleanFixture()
	f.values[1] = Reading{Current: 10, LifetimePeak: 5, PeakKnown: true}
	got = Sample(7, f)
	if !got.Complete || got.LifetimePeakSumKnown || got.Members[0].Reading.PeakKnown || got.Members[0].Reading.LifetimePeak != 0 {
		t.Fatal(got)
	}
}

func TestIncompleteNeverPublishesPartialGroupSum(t *testing.T) {
	for _, kind := range []string{"invalid", "group", "empty", "unbounded", "duplicate", "churn", "identity", "reused", "wrong-group", "footprint", "overflow", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			f := cleanFixture()
			group := 7
			switch kind {
			case "invalid":
				group = 0
			case "group", "identity", "footprint":
				f.fail = kind
			case "empty":
				f.groups = [][]int{{}}
			case "unbounded":
				f.groups = [][]int{make([]int, MaximumMembers)}
			case "duplicate":
				f.groups = [][]int{{1, 1}}
			case "churn":
				f.groups = [][]int{{1, 2}, {1}}
			case "reused":
				f.identities[1] = []Identity{{PID: 1, Group: 7, StartSeconds: 1}, {PID: 1, Group: 7, StartSeconds: 3}}
			case "wrong-group":
				f.identities[1] = []Identity{{PID: 1, Group: 8, StartSeconds: 1}}
			case "overflow":
				f.values[1] = Reading{Current: math.MaxUint64}
				f.values[2] = Reading{Current: 1}
			case "unknown":
				f.values[1] = Reading{}
			}
			got := Sample(group, f)
			if got.Complete || got.GroupCurrentBytes != 0 || len(got.Members) != 0 || got.Reason == "" {
				t.Fatal(got)
			}
		})
	}
}

func TestWindowMarksGapsAndChurnAndTracksSampledPeak(t *testing.T) {
	a := Sample(7, cleanFixture())
	b := Sample(7, cleanFixture())
	b.GroupCurrentBytes = 40
	b.Members[0].Identity.StartSeconds++
	var window Window
	window.Observe(a)
	window.Observe(Observation{Reason: "read failed"})
	window.Observe(b)
	if window.Samples != 3 || window.CompleteSamples != 2 || window.Gaps != 1 || !window.MembershipChanged || window.PeakObservedGroupBytes != 40 {
		t.Fatal(window)
	}
}
