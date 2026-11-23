package template

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/world"
)

// wantError fails the test unless err mentions every word given.
func wantError(t *testing.T, name string, err error, words ...string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: no error", name)
		return
	}
	for _, word := range words {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("%s: error %q does not mention %q", name, err, word)
		}
	}
}

// TestSmallMoonRefusesWhatNewBodyAndNewMoonRefuse checks a moon with no radius, and one on a negative orbit, are each refused.
func TestSmallMoonRefusesWhatNewBodyAndNewMoonRefuse(t *testing.T) {
	_, err := smallMoon("dot", km(0), kg(1), km(10), deg(0))
	wantError(t, "no radius", err, "dot", "radius")
	_, err = smallMoon("dot", km(1), kg(1), km(-10), deg(0))
	wantError(t, "negative orbit", err, "dot", "orbit")
	moon, err := smallMoon("dot", km(1), kg(1), km(10), deg(0))
	if err != nil || moon.Body().Name() != "dot" {
		t.Errorf("a good moon: %v, %v", moon, err)
	}
}

// TestGroundedMoonRefusesEachPart checks a bad terrain, radius or orbit is refused, the terrain's with the moon's name.
func TestGroundedMoonRefusesEachPart(t *testing.T) {
	good := world.Terrain{Seed: 1, Ocean: 0, Plates: 3}
	_, err := groundedMoon("pebble", km(1), kg(1), world.Terrain{Ocean: math.NaN()}, world.Look{}, km(5), deg(0))
	wantError(t, "bad terrain", err, "pebble:", "ocean")
	_, err = groundedMoon("pebble", km(0), kg(1), good, world.Look{}, km(5), deg(0))
	wantError(t, "no radius", err, "pebble", "radius")
	_, err = groundedMoon("pebble", km(1), kg(1), good, world.Look{}, km(-5), deg(0))
	wantError(t, "negative orbit", err, "pebble", "orbit")
}

// goodTail is a tail that NewTail accepts.
func goodTail() tailOf {
	return tailOf{length: km(100), base: km(10), loss: bodies.PerKilometre(0.01),
		colour: maths.RGB(1, 1, 1), glow: maths.RGB(0.5, 0.5, 0.5), wisp: maths.Percent(10), seed: 3}
}

// TestTailMadeRefusesAFlatTail checks a tail with no length is refused, and a good one is made.
func TestTailMadeRefusesAFlatTail(t *testing.T) {
	flat := goodTail()
	flat.length = km(0)
	_, err := flat.made()
	wantError(t, "flat tail", err, "tail")
	if _, err := goodTail().made(); err != nil {
		t.Errorf("a good tail: %v", err)
	}
}

// TestCometNamesTheRefusedPart checks each of a comet's parts, when refused, is named with the comet.
func TestCometNamesTheRefusedPart(t *testing.T) {
	loss := bodies.PerKilometre(0.01)
	colours := [2]maths.Light{maths.RGB(1, 1, 1), maths.RGB(0.5, 0.5, 0.5)}
	flat := goodTail()
	flat.length = km(0)
	build := func(radius, coma bodies.Span, dust, ion tailOf) error {
		_, err := comet("snowball", radius, kg(1e13), loss, coma, colours, dust, ion, 1)
		return err
	}
	noComa := build(km(5), km(0), goodTail(), goodTail())
	wantError(t, "no coma", noComa, "snowball:", "coma")
	flatDust := build(km(5), km(100), flat, goodTail())
	wantError(t, "flat dust tail", flatDust, "snowball:", "tail")
	flatIon := build(km(5), km(100), goodTail(), flat)
	wantError(t, "flat ion tail", flatIon, "snowball:", "tail")
	noNucleus := build(km(0), km(100), goodTail(), goodTail())
	wantError(t, "no nucleus", noNucleus, "snowball", "radius")
	if err := build(km(5), km(100), goodTail(), goodTail()); err != nil {
		t.Errorf("a good comet: %v", err)
	}
}
