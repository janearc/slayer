package hillside

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/ninjafilm/lens"
)

// TestFastRefusalNamesWhatCannotBeDrawnWithAShortcut is each thing a
// shortcut is refused with, and beside each the same thing asked with
// no shortcut, which is left alone.
func TestFastRefusalNamesWhatCannotBeDrawnWithAShortcut(t *testing.T) {
	for _, each := range []struct {
		name  string
		given asked
		want  error
	}{{"on the cpu", asked{}, errFastOnCPU},
		{"through a rig", asked{onGPU: true, glass: &lens.Thin{}},
			errFastThroughGlass},
		{"beside the cached grid", asked{onGPU: true, cachedLighting: true},
			errFastAndCached},
		{"beside a proof", asked{onGPU: true, prove: "frame.pfm"},
			errFastProved},
		{"on the gpu through a pinhole", asked{onGPU: true}, nil}} {
		if err := fastRefusal(each.given); err != nil {
			t.Errorf("%s, no shortcut asked: refused with %v", each.name,
				err)
		}
		each.given.fastLighting = true
		if err := fastRefusal(each.given); !errors.Is(err, each.want) {
			t.Errorf("%s: refused with %v, want %v", each.name, err,
				each.want)
		}
	}
	alone := asked{onGPU: true, skyLooks: true}
	if err := fastRefusal(alone); !errors.Is(err, errSkyAlone) {
		t.Errorf("sky-looks alone says %v", err)
	}
	alone.fastLighting = true
	if err := fastRefusal(alone); err != nil {
		t.Errorf("sky-looks with lamp-store says %v", err)
	}
}

// setSays is a set's flags from a table, as set.Set.Flag gives them.
func setSays(said map[string]bool) func(string) (bool, bool) {
	return func(name string) (bool, bool) {
		on, has := said[name]
		return on, has
	}
}

// TestAShortcutIsDecidedByTheFirstThatSaysIt works the lamp store
// out from what the job or command line says and what the
// set says: a shortcut's own word, then fastlight, then vibes, each
// from the job before the set.
func TestAShortcutIsDecidedByTheFirstThatSaysIt(t *testing.T) {
	on, off := true, false
	for _, each := range []struct {
		name     string
		job, set map[string]bool
		store    bool
	}{{"nothing said", nil, nil, off},
		{"the set's fastlight", nil,
			map[string]bool{job.FlagFastlight: on}, on},
		{"the set's vibes", nil, map[string]bool{job.FlagVibes: on}, on},
		{"the job's fastlight off over the set's on",
			map[string]bool{job.FlagFastlight: off},
			map[string]bool{job.FlagFastlight: on}, off},
		{"the set's fastlight off over the job's vibes",
			map[string]bool{job.FlagVibes: on},
			map[string]bool{job.FlagFastlight: off}, off},
		{"the shortcut's own word off over every flag",
			map[string]bool{job.ShortcutLampStore: off,
				job.FlagFastlight: on}, map[string]bool{job.FlagVibes: on},
			off},
		{"the shortcut's own word on with every flag off",
			map[string]bool{job.ShortcutLampStore: on,
				job.FlagVibes: off},
			map[string]bool{job.FlagFastlight: off}, on},
	} {
		given := asked{switches: each.job}.resolved(setSays(each.set))
		if given.fastLighting != each.store {
			t.Errorf("%s: lamp-store %v", each.name, given.fastLighting)
		}
	}
}

// TestTheCommandLinesFlagsAreNotedOnlyWhenGiven parses hillside's own
// two flags: each given is noted on, and neither given notes nothing.
func TestTheCommandLinesFlagsAreNotedOnlyWhenGiven(t *testing.T) {
	if said := mustParse(t).flagged().switches; len(said) != 0 {
		t.Errorf("no flag given noted %v", said)
	}
	given := mustParse(t, "-fastlight").flagged().resolved(noSetFlags)
	if !given.fastLighting || !given.skyLooks {
		t.Errorf("-fastlight asks lamp-store %v, sky-looks %v",
			given.fastLighting, given.skyLooks)
	}
	given = mustParse(t, "-fastlight", "-exact-sky").flagged().resolved(
		noSetFlags)
	if !given.fastLighting || given.skyLooks {
		t.Errorf("-fastlight -exact-sky asks lamp-store %v, sky-looks %v",
			given.fastLighting, given.skyLooks)
	}
	given = mustParse(t, "-vibes").flagged().resolved(setSays(
		map[string]bool{job.FlagFastlight: false}))
	if given.fastLighting {
		t.Error("-vibes stood over the set's fastlight: off")
	}
}
