//go:build draw

package session

import (
	"context"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/job"
)

// TestGroundScaleMovesAttachedLight checks staging only; it draws no image.
func TestGroundScaleMovesAttachedLight(t *testing.T) {
	path := onFixture(t, "", "")
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Ground light scale
  setting: ground
  light: moon
  timeofday: 2200
  terrain: {from: fixture, file: fixture-plain.tif, west: 1001, north: 8999, across: 150, down: 100, step: 2}
  film: {width: 24, height: 16, rays: 1, exposure: 100000}
  cameras:
    view: {orientation: {point_to: lamp, alpha: -25}, location: {loc_relative: {object: lamp, standoff_x: 20, standoff_y: 0, standoff_z: 10}}}
  set:
    things:
      lamp: {kind: spotlight, seed: 4, east: 1050, north: 8990, heading: {value: 30, unit: deg}}
`
	open := func(text string) *Session {
		t.Helper()
		wanted, err := job.StillFrom(path, []byte(text))
		if err != nil {
			t.Fatal(err)
		}
		owner, err := Open(context.Background(), wanted, Stamp{})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = owner.Close() })
		return owner
	}
	base := open(source)
	scaled := open(strings.Replace(source, "kind: spotlight, seed: 4", "kind: spotlight, scale: 2, seed: 4", 1))
	oldView, err := base.WorkingViewShared(0)
	if err != nil {
		t.Fatal(err)
	}
	newView, err := scaled.WorkingViewShared(0)
	if err != nil {
		t.Fatal(err)
	}
	origin := oldView.Origins["scene.set.things/lamp"]
	if origin != newView.Origins["scene.set.things/lamp"] {
		t.Fatal("scaling moved the terrain anchor")
	}
	oldWorld := base.source.(stillFramer).world
	newWorld := scaled.source.(stillFramer).world
	if len(oldWorld.Lamps) == 0 || len(oldWorld.Lamps) != len(newWorld.Lamps) {
		t.Fatalf("attached lights %d %d", len(oldWorld.Lamps), len(newWorld.Lamps))
	}
	a, ok := oldWorld.Lamps[len(oldWorld.Lamps)-1].(emit.Lit)
	b, ok2 := newWorld.Lamps[len(newWorld.Lamps)-1].(emit.Lit)
	if !ok || !ok2 {
		t.Fatal("attached light type changed")
	}
	if b.At.Sub(origin).Sub(a.At.Sub(origin).Scale(2)).Len() > 1e-8 || b.Beyond != a.Beyond*2 || b.Strength != a.Strength {
		t.Fatalf("scaled lamp %+v from %+v at %v", b, a, origin)
	}
}
