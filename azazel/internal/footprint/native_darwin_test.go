//go:build darwin && cgo

package footprint

import (
	"os"
	"slices"
	"syscall"
	"testing"
)

func TestNativeReadsCurrentProcessIdentityAndFootprint(t *testing.T) {
	source := Native()
	pid := os.Getpid()
	id, err := source.Identity(pid)
	if err != nil {
		t.Fatal(err)
	}
	group, err := syscall.Getpgid(pid)
	if err != nil || !validIdentity(id, pid, group) {
		t.Fatal(id, group, err)
	}
	members, err := source.Group(group)
	if err != nil || !slices.Contains(members, pid) {
		t.Fatal(members, err)
	}
	reading, err := source.Footprint(pid)
	if err != nil || reading.Current == 0 || reading.PeakKnown && reading.LifetimePeak < reading.Current {
		t.Fatal(reading, err)
	}
}
