package session

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// the small job's own numbers, written in its film below: the picture's
// size, its rays and its exposure
const (
	smallJobWidth    = 24
	smallJobHeight   = 16
	smallJobRays     = 3
	smallJobExposure = 1500.0
)

// the usual stand's counts, as usual says them above
const (
	usualSpruces  = 5
	usualFerns    = 24
	usualBoulders = 8
)

// TestUsualAskedIsWhatParseGivesNobody holds that the usual request is,
// field for field and the unexported ones too, what the command's parse
// gives when it is given no arguments. asked holds no function and
// nothing else DeepEqual cannot compare, so DeepEqual is the whole
// comparison.
func TestUsualAskedIsWhatParseGivesNobody(t *testing.T) {
	parsed, err := parse(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(usualAsked(), parsed) {
		t.Errorf("the usual request is not what parse gives:\n%+v\n%+v",
			usualAsked(), parsed)
	}
}

// anti-test of TestUsualAskedIsWhatParseGivesNobody
// TestUsualAskedComparisonSeesOneDefaultChanged holds that the same
// comparison says they differ when one default is changed in a copy,
// a scalar, a count and a switch each.
func TestUsualAskedComparisonSeesOneDefaultChanged(t *testing.T) {
	parsed, err := parse(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	changes := map[string]func(*asked){
		"seed":   func(given *asked) { given.seed++ },
		"width":  func(given *asked) { given.width++ },
		"count":  func(given *asked) { given.counts["spruce"]++ },
		"switch": func(given *asked) { given.switches["vibes"] = true },
	}
	for name, change := range changes {
		copied := usualAsked()
		change(&copied)
		if reflect.DeepEqual(copied, parsed) {
			t.Errorf("a changed %s was not seen", name)
		}
	}
}

// TestUsualAskedCountsAreTheUsualStand holds the counts by value: the
// stand, ferns and stones usual says, and none of a kind it does not.
func TestUsualAskedCountsAreTheUsualStand(t *testing.T) {
	counts := usualAsked().counts
	if counts["spruce"] != usualSpruces || counts["fern"] != usualFerns ||
		counts["boulder"] != usualBoulders || counts["bonfire"] != 0 {
		t.Errorf("the usual counts were %v", counts)
	}
	if len(counts) != len(flagKinds()) {
		t.Errorf("%d counts for %d kinds", len(counts), len(flagKinds()))
	}
}

// TestJobAskedForASmallJob holds what a job of a set and a film asks:
// the set's path, the film's size, rays and exposure, each marked said,
// and the usual for all the rest, none of it said. The expectation is
// read from jobAsked's code and the film below.
func TestJobAskedForASmallJob(t *testing.T) {
	set := filepath.Join(t.TempDir(), "small.yaml")
	if err := os.WriteFile(set, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	given, err := jobAsked(jobFrom(t, "scene: {set: "+set+"}\n"+
		"film: {width: 24, height: 16, rays: 3, exposure: 1500}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if given.set != set {
		t.Errorf("the set was %q, not %q", given.set, set)
	}
	if given.width != smallJobWidth || given.height != smallJobHeight ||
		given.samples != smallJobRays || given.exposure != smallJobExposure {
		t.Errorf("the film was %dx%d, %d rays, exposure %v", given.width,
			given.height, given.samples, given.exposure)
	}
	said := given.said
	if !said.width || !said.height || !said.samples || !said.exposure {
		t.Errorf("the film's own words were not marked said: %+v", said)
	}
	if said.renderer || said.fantasy != nil || len(given.switches) != 0 {
		t.Errorf("a word the job did not say was marked: %+v", given)
	}
	if given.seed != usualSeed || given.counts["spruce"] != usualSpruces ||
		given.apart != usualApart || given.onGPU {
		t.Errorf("the rest was not the usual: %+v", given)
	}
}

// anti-test of TestJobAskedForASmallJob
// TestJobAskedIsNotTheUsualWhereTheJobSpeaks holds that a job's request
// is not the usual one: the film's words stand over it.
func TestJobAskedIsNotTheUsualWhereTheJobSpeaks(t *testing.T) {
	given, err := jobAsked(jobFrom(t, "scene: {smoke: 1}\n"+
		"film: {width: 24, height: 16, rays: 3}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(given, usualAsked()) {
		t.Error("a job that names a film asked for the usual request")
	}
}

// TestSequenceAskedIsTheUsualOfTheSet holds what a sequence draws from:
// the set file's path and the usual for all the rest, the size and the
// rays not said, so that the set's film stands over them. Every word
// said is read from sequenceAsked's code, which sets only the path.
func TestSequenceAskedIsTheUsualOfTheSet(t *testing.T) {
	scene := filepath.Join(t.TempDir(), "scene.yaml")
	given, err := sequenceAsked(scene)
	if err != nil {
		t.Fatal(err)
	}
	if given.set != scene {
		t.Errorf("the set was %q, not %q", given.set, scene)
	}
	if given.width != usualWide || given.height != usualTall ||
		given.samples != usualRays {
		t.Errorf("the picture was %dx%d, %d rays", given.width, given.height,
			given.samples)
	}
	if given.said.width || given.said.height || given.said.samples ||
		given.said.fantasy != nil || given.switches == nil ||
		len(given.switches) != 0 {
		t.Errorf("a word was marked said that nobody said: %+v", given)
	}
}

// anti-test of TestSequenceAskedIsTheUsualOfTheSet
// TestSequenceAskedDiffersFromTheUsualOnlyInTheSet holds that the
// request of a set is the usual one with the path and nothing more:
// with the path put back, it is the usual, and with it, it is not.
func TestSequenceAskedDiffersFromTheUsualOnlyInTheSet(t *testing.T) {
	given, err := sequenceAsked("scene.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(given, usualAsked()) {
		t.Error("a request of a set was the usual one")
	}
	given.set = ""
	if !reflect.DeepEqual(given, usualAsked()) {
		t.Errorf("the request of a set differs beyond its path: %+v", given)
	}
}
