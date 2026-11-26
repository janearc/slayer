//go:build draw

package session

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
)

// TestGroundedWorkingOriginFollowsTerrain stages two versions without
// rendering and compares their actual authored anchors.
func TestGroundedWorkingOriginFollowsTerrain(t *testing.T) {
	path := onFixture(t, "scene:\n", "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json\nscene:\n  setting: ground\n")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	origin := func(source string) maths.Vec {
		t.Helper()
		wanted, err := job.StillFrom(path, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		owner, err := Open(context.Background(), wanted, Stamp{})
		if err != nil {
			t.Fatal(err)
		}
		defer owner.Close()
		view, err := owner.WorkingViewShared(0)
		if err != nil {
			t.Fatal(err)
		}
		at, found := view.Origins["scene.set.things/the stone"]
		if !found {
			t.Fatal("the staged stone has no authored origin")
		}
		return at
	}
	before := origin(string(raw))
	moved := strings.Replace(string(raw), "        east: 1050\n        north: 8990", "        east: 1052\n        north: 8988", 1)
	if moved == string(raw) {
		t.Fatal("fixture did not move")
	}
	after := origin(moved)
	delta := after.Sub(before)
	if math.Abs(delta.X-2) > 1e-9 || math.Abs(delta.Z-2) > 1e-9 || delta.Y == 0 {
		t.Fatalf("ground anchor changed by %v; want 2 east, 2 south and a new terrain height", delta)
	}
}
