package queue

import (
	"errors"
	"math"
	"syscall"
	"testing"
)

type residentFixture struct {
	pids        []int
	bytes       map[int]uint64
	groupErr    error
	residentErr error
}

func (f residentFixture) Group(int) ([]int, error)         { return f.pids, f.groupErr }
func (f residentFixture) Resident(pid int) (uint64, error) { return f.bytes[pid], f.residentErr }

func TestMetricsGroupResidentIsIndependentAndRejectsPartial(t *testing.T) {
	ok := residentFixture{pids: []int{1, 2}, bytes: map[int]uint64{1: 10, 2: 20}}
	if got, err := metricsGroupResident(7, ok); err != nil || got != 30 {
		t.Fatal(got, err)
	}
	for _, test := range []struct {
		name   string
		group  int
		source residentFixture
	}{
		{"invalid", 0, ok}, {"gone", 7, residentFixture{}},
		{"group error", 7, residentFixture{groupErr: errors.New("group")}},
		{"process error", 7, residentFixture{pids: []int{1}, residentErr: syscall.ESRCH}},
		{"overflow", 7, residentFixture{pids: []int{1, 2}, bytes: map[int]uint64{1: math.MaxUint64, 2: 1}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := metricsGroupResident(test.group, test.source); err == nil {
				t.Fatal("accepted incomplete observation")
			}
		})
	}
}
