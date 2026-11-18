package footprint

import (
	"errors"
	"testing"
)

// bootTail is the part of a boot identifier after its first eight digits.
const bootTail = "-89ab-cdef-0123-456789abcdef"

// bootOne and bootTwo are two boot identifiers that differ in one digit.
const bootOne = "01234567" + bootTail
const bootTwo = "11234567" + bootTail

type leaderSource struct {
	identity Identity
	err      error
}

func (s leaderSource) Identity(int) (Identity, error) { return s.identity, s.err }

func TestCaptureLeaderRequiresSameValidBootAndExactGroup(t *testing.T) {
	good := leaderSource{identity: Identity{PID: 7, Group: 7, StartSeconds: 10, StartMicroseconds: 2}}
	for _, row := range []struct {
		name          string
		source        leaderSource
		first, second string
		bootErr       bool
		want          bool
	}{
		{"valid", good, bootOne, bootOne, false, true},
		{"boot changed", good, bootOne, bootTwo, false, false},
		{"boot unavailable", good, bootOne, bootOne, true, false},
		{"bad boot", good, "bad", "bad", false, false},
		{"identity unavailable", leaderSource{err: errors.New("gone")}, bootOne, bootOne, false, false},
		{"group mismatch", leaderSource{identity: Identity{PID: 7, Group: 8, StartSeconds: 10}}, bootOne, bootOne, false, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			calls := 0
			boot := func() (string, error) {
				calls++
				if row.bootErr {
					return "", errors.New("unknown")
				}
				if calls == 1 {
					return row.first, nil
				}
				return row.second, nil
			}
			leader, ok := CaptureLeader(7, row.source, boot)
			if ok != row.want || ok && (leader.BootID != bootOne || leader.Identity != good.identity) {
				t.Fatal(leader, ok)
			}
		})
	}
}

func TestLeaderMatchRejectsUnknownRebootAndPIDReuse(t *testing.T) {
	f := cleanFixture()
	observation := Sample(7, f)
	expected := Leader{BootID: bootOne, Identity: Identity{PID: 1, Group: 7, StartSeconds: 1}}
	// A group leader's PID equals its group; use a fixture for group 7.
	observation.Members[0].Identity = Identity{PID: 7, Group: 7, StartSeconds: 10, StartMicroseconds: 4}
	expected.Identity = observation.Members[0].Identity
	if !LeaderMatches(expected, bootOne, observation) {
		t.Fatal("same identity refused")
	}
	for _, test := range []struct {
		name, boot  string
		expected    Leader
		observation Observation
	}{
		{"reboot", bootTwo, expected, observation}, {"unknown boot", "", expected, observation},
		{"no metadata", bootOne, Leader{}, observation},
		{"reused pid", bootOne, Leader{BootID: bootOne, Identity: Identity{PID: 7, Group: 7, StartSeconds: 9}}, observation},
		{"incomplete", bootOne, expected, Observation{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if LeaderMatches(test.expected, test.boot, test.observation) {
				t.Fatal("accepted uncertain leader")
			}
		})
	}
}
