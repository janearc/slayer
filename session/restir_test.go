package session

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
)

// restir is a light shortcut: fastlight and vibes turn it on, its own word
// stands over both, and with nothing said it is off.
func TestRestirIsDecidedLikeTheOtherLightShortcuts(t *testing.T) {
	on, off := true, false
	for _, each := range []struct {
		name string
		job  map[string]bool
		want bool
	}{{"nothing said", nil, off},
		{"vibes", map[string]bool{job.FlagVibes: on}, on},
		{"fastlight", map[string]bool{job.FlagFastlight: on}, on},
		{"vibes off", map[string]bool{job.FlagVibes: off}, off},
		{"its own word over vibes", map[string]bool{job.FlagVibes: on,
			job.ShortcutRestir: off}, off},
		{"its own word alone", map[string]bool{job.ShortcutRestir: on}, on},
		// anti: the other shortcut's word does not turn it on
		{"lamp-store alone", map[string]bool{job.ShortcutLampStore: on}, off},
	} {
		got := asked{switches: each.job}.resolved(noSetFlags).restir
		if got != each.want {
			t.Errorf("%s: restir is %v, want %v", each.name, got, each.want)
		}
	}
}

// what restir is refused with, each by name, and the same asked with it
// off, which is left alone.
func TestRestirIsRefusedWhereTheOtherShortcutsAre(t *testing.T) {
	for _, each := range []struct {
		name  string
		given asked
		want  error
	}{{"on the cpu", asked{}, errFastOnCPU},
		{"through a rig", asked{onGPU: true, glass: &lens.Thin{}},
			errFastThroughGlass},
		{"beside a proof", asked{onGPU: true, prove: "frame.pfm"}, errFastProved},
		{"beside the cached grid", asked{onGPU: true, cachedLighting: true},
			errRestirCached},
		{"beside an open shutter", asked{onGPU: true,
			optics: optics{closing: &maths.Subspace{}}}, errRestirShutter},
	} {
		if err := fastRefusal(each.given); err != nil {
			t.Errorf("%s, restir off: refused with %v", each.name, err)
		}
		each.given.restir = true
		if err := fastRefusal(each.given); !errors.Is(err, each.want) {
			t.Errorf("%s: refused with %v, want %v", each.name, err, each.want)
		}
	}
	if err := fastRefusal(asked{onGPU: true, restir: true}); err != nil {
		t.Errorf("restir on the gpu through a pinhole: %v", err)
	}
}
