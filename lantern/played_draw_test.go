//go:build draw

package lantern_test

import (
	"testing"
	"time"
)

// At rest, a played zappy's world holds exactly what Place put in it,
// before and after any channel has moved: the same things, the same lamps.
func TestAtRestTheWorldIsPlaces(t *testing.T) {
	made, world, things, lamps := played(t)
	if err := made.Hold(nil); err != nil {
		t.Fatal(err)
	}
	if err := made.Hold(map[string]float64{"star-brightness": twice,
		"shells-band": wider}); err != nil {
		t.Fatal(err)
	}
	if err := made.Hold(nil); err != nil {
		t.Fatal(err)
	}
	if moved := changed(world, things); len(moved) != 0 {
		t.Errorf("at rest, things %v are not Place's", moved)
	}
	if moved := lampsChanged(world, lamps); len(moved) != 0 {
		t.Errorf("at rest, lamps %v are not Place's", moved)
	}
}

// A shell's band and opening make its panels again; every shell's when
// all the shells' are asked, and the remaking is timed.
func TestBandAndOpenMakeThePanelsAgain(t *testing.T) {
	made, world, things, _ := played(t)
	start := time.Now()
	if err := made.Hold(map[string]float64{"shells-band": wider,
		"shells-open": quarter}); err != nil {
		t.Fatal(err)
	}
	t.Logf("every shell made again in %v", time.Since(start))
	if len(changed(world, things)) != len(things)-1 {
		t.Errorf("band and open moved %d things, want every sheet",
			len(changed(world, things)))
	}
	first := world.Things[1].Shape
	if err := made.Hold(nil); err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	if err := made.Hold(map[string]float64{"shells-band": wider,
		"shells-open": quarter}); err != nil {
		t.Fatal(err)
	}
	t.Logf("and held again in %v", time.Since(start))
	if world.Things[1].Shape != first {
		t.Error("a band and opening held again were made again")
	}
}
