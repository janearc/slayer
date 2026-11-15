//go:build draw

package hillside

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/janearc/slayer/set"
)

// TestDrawRefusesAShortcutBeforeDrawing asks drawMeasured for a
// shortcut on the cpu with no world: it refuses before touching one.
func TestDrawRefusesAShortcutBeforeDrawing(t *testing.T) {
	given := mustParse(t)
	given.fastLighting = true
	if _, _, err := drawMeasured(nil, given, 1); !errors.Is(err,
		errFastOnCPU) {
		t.Fatalf("a shortcut on the cpu says %v", err)
	}
}

// TestASetSaysFastlightAndOnlyThenIsItAsked reads the fixture with and
// without the flag under its scene.
func TestASetSaysFastlightAndOnlyThenIsItAsked(t *testing.T) {
	for _, each := range []struct {
		said string
		want bool
	}{{"scene:\n  renderer: gpu-native\n  fastlight: on", true},
		{"scene:\n  renderer: gpu-native\n  vibes: on", true},
		{"scene:\n  renderer: gpu-native\n  fastlight: off", false},
		{"scene:\n  renderer: gpu-native", false}} {
		path := onFixture(t, "scene:", each.said)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		read, err := set.Read(path, raw)
		if err != nil {
			t.Fatal(err)
		}
		given, err := filmed(read, mustParse(t))
		if err != nil {
			t.Fatal(err)
		}
		if given.fastLighting != each.want {
			t.Errorf("%q: lamp-store %v", each.said, given.fastLighting)
		}
	}
}

// TestAJobsFlagsReachTheSessionAndANamedSceneRefusesThem reads a job's
// flags into what is asked, and opens a named scene with one.
func TestAJobsFlagsReachTheSessionAndANamedSceneRefusesThem(t *testing.T) {
	for _, each := range []struct {
		said  string
		store bool
	}{{"fastlight: on\n", true},
		{"vibes: on\nshortcuts: {lamp-store: off}\n", false},
		{"fastlight: off\n", false}, {"", false}} {
		given, err := jobAsked(stillOf(t, sessionWidow, "", each.said))
		if err != nil {
			t.Fatal(err)
		}
		if given.fastLighting != each.store {
			t.Errorf("%q: lamp-store %v", each.said, given.fastLighting)
		}
	}
	given, err := jobAsked(stillOf(t, sessionWidow, "",
		"fastlight: on\nshortcuts: {sky-looks: off}\n"))
	if err != nil || !given.fastLighting || given.skyLooks {
		t.Errorf("sky-looks off under fastlight: lamp-store %v, "+
			"sky-looks %v, %v", given.fastLighting, given.skyLooks, err)
	}
	_, err = Open(context.Background(), jobFrom(t,
		namedJob("named: swarm", "fastlight: on\n")), Stamp{})
	if !errors.Is(err, errFastNamed) {
		t.Errorf("a named scene with fastlight says %v", err)
	}
}
