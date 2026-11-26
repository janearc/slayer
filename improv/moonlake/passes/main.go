// Command passes is improv/moonlake's: it writes one set
// per stage fixture, each sets/moonlake.yaml with the moon's light and
// disc off and that one fixture standing in it, so a render of each is
// the scene lit by that fixture alone. Light adds, so the show mixes
// them per frame in any amount.
//
//	passes sets/moonlake.yaml sets
//
// writes sets/moonlake-pass-NAME.yaml for each fixture below, the
// overhead, and the fire.
//
// The fixtures stand on the flanks of the west rise, out of frame: 10 m
// ahead of the camera and 20 m to either side. A fixture's head faces
// east before it is turned, and hillside turns it by -heading, so a
// heading of h points it at compass azimuth 90 + h: each aim below is
// an azimuth, and its heading is the aim less 90.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// fixture is one light on a stand: its name, kind, seed, where it
// stands, and the compass azimuth it is aimed at.
type fixture struct {
	name, kind  string
	seed        int
	east, north float64
	aim         float64
}

// the flanks: the camera at 263104.5 e, 2148423.5 n looking at azimuth
// 109.65; ahead is (0.9417, -0.3363), left is (0.3363, 0.9417)
const (
	leftEast, leftNorth   = 263120.6, 2148439.0
	rightEast, rightNorth = 263107.2, 2148401.4
)

// fixtures are the flood and three aims of the hard spot on each flank:
// on the near spruce on its own side, across in front of both, and out
// over the lake into the fog.
var fixtures = []fixture{
	{"flood-left", "strobe", 61, leftEast, leftNorth, 152},
	{"flood-right", "strobe", 62, rightEast, rightNorth, 67},
	{"spot-left-0", "spotlight", 71, leftEast + 1.5, leftNorth - 1, 150},
	{"spot-left-1", "spotlight", 71, leftEast + 1.5, leftNorth - 1, 125},
	{"spot-left-2", "spotlight", 71, leftEast + 1.5, leftNorth - 1, 100},
	{"spot-right-0", "spotlight", 72, rightEast + 1.5, rightNorth + 1, 70},
	{"spot-right-1", "spotlight", 72, rightEast + 1.5, rightNorth + 1, 95},
	{"spot-right-2", "spotlight", 72, rightEast + 1.5, rightNorth + 1, 120},
	// the backlights: strobes 45 m ahead, past the close spruces and 12 m
	// either side, aimed back at the camera, so the trees stand black
	// with lit edges and the haze glows between them. Below the frame.
	{"back-left", "strobe", 63, 263150.9, 2148419.7, 275},
	{"back-right", "strobe", 64, 263142.8, 2148397.1, 305},
}

// fireRecipe is the wood fire from fire-night's 13-night-film.yaml,
// "the frames for this came out STUNNING", its flames 24 m tall
// and its absorption a fifth, so a flame that size looks as deep.
//
// it bursts from the towers' top decks (2026-11-10: "bar 97.1 is
// when the fires should BURST from the towers"), so its sources are
// eight over a disc 5 m across, the deck's depth, not the lake fire's
// 24 over 24 m; sparks thrown from 16 m. No smoke yet.
const fireRecipe = `  recipes:
    the moonlake fire:
      kind: fire
      layout:
        pattern: cluster
        count: 8
        radius: {value: 2.5, unit: m}
        height-variation: 0.5
      flame:
        radius: {value: 2, unit: m}
        height: {value: 24, unit: m}
        absorption: {value: 0.12, unit: m^-1}
        temperature: {value: 1900, unit: K}
        edge-temperature: {value: 1100, unit: K}
        intermittency-start: 0.06
        temperature-profile: hot-tongues
      sparks:
        rate: {value: 400, unit: s^-1}
        lifetime: {value: 4, unit: s}
        radius: {value: 0.12, unit: m}
        release-height: {value: 16, unit: m}
        launch-speed: {value: 25, unit: m/s}
        spread: {value: 8, unit: m/s}
        updraft: {value: 20, unit: m/s}
        drag-time: {value: 0.8, unit: s}
        gravity: {value: 9.80665, unit: m/s^2}
        temperature: {value: 1650, unit: K}
        end-temperature: {value: 900, unit: K}
      light-cells: 4
`

// swellRecipe is the fire at its pulse, 2026-11-10: "it's going to
// pulse and get much bigger and brighter and send off much more sparks,
// on the downbeat, and it's quick and violent".
//
// Half again as tall and broad as the fire, a hundred kelvin hotter, its sparks twice as many,
// thrown harder and living half as long, so four of them stay under the
// scene's 8192. lightshow cuts to it on each downbeat and lets it fall
// back to the fire.
const swellRecipe = `  recipes:
    the moonlake fire:
      kind: fire
      layout:
        pattern: cluster
        count: 12
        radius: {value: 3.5, unit: m}
        height-variation: 0.5
      flame:
        radius: {value: 3.2, unit: m}
        height: {value: 38, unit: m}
        absorption: {value: 0.1, unit: m^-1}
        temperature: {value: 2000, unit: K}
        edge-temperature: {value: 1150, unit: K}
        intermittency-start: 0.06
        temperature-profile: hot-tongues
      sparks:
        rate: {value: 900, unit: s^-1}
        lifetime: {value: 2, unit: s}
        radius: {value: 0.12, unit: m}
        release-height: {value: 24, unit: m}
        launch-speed: {value: 30, unit: m/s}
        spread: {value: 14, unit: m/s}
        updraft: {value: 20, unit: m/s}
        drag-time: {value: 0.8, unit: s}
        gravity: {value: 9.80665, unit: m/s^2}
        temperature: {value: 1750, unit: K}
        end-temperature: {value: 900, unit: K}
      light-cells: 4
`

// burstRecipe is sparks flying outward in a sphere, from "sparks
// flying outwards in a sphere" at the ignition and on each downbeat: a
// flame too small and thin to see, throwing sparks with no launch, 30 m/s
// of spread on every axis and almost no updraft, so they fly out and
// fall.
//
// Rendered with hillside -sparks-for 0.12s, so they are thrown at
// once: 600 a fire, at 5000 a second, which embers' 8192 living allows
// (rate by lifetime under it) and hillside counts by its stop.
const burstRecipe = `  recipes:
    the moonlake fire:
      kind: fire
      flame:
        radius: {value: 0.05, unit: m}
        height: {value: 0.1, unit: m}
        absorption: {value: 0.001, unit: m^-1}
        temperature: {value: 1900, unit: K}
        edge-temperature: {value: 1100, unit: K}
      sparks:
        rate: {value: 5000, unit: s^-1}
        lifetime: {value: 1.6, unit: s}
        radius: {value: 0.15, unit: m}
        release-height: {value: 2, unit: m}
        launch-speed: {value: 0, unit: m/s}
        spread: {value: 30, unit: m/s}
        updraft: {value: 2, unit: m/s}
        drag-time: {value: 1.2, unit: s}
        gravity: {value: 9.80665, unit: m/s^2}
        temperature: {value: 1800, unit: K}
        end-temperature: {value: 950, unit: K}
      light-cells: 3
`

// fires stand on the four laser towers (sets/moonlake.yaml's rigs), each
// on its top deck's flame units, 20.9 m over the tower's ground: the
// stage tower's deck at 20.08 m, the units half a metre, their nozzles
// 0.35 m more. a fire lifted off the ground has nothing under it to hide
// what falls below its base, so its sparks may be seen falling there.
const fires = `      # the pass's fires, on the towers' top decks
      fire rim left:
        recipe: the moonlake fire
        seed: 81
        east: 264872.4
        north: 2147919.0
        height: {value: 20.9, unit: m}
      fire rim right:
        recipe: the moonlake fire
        seed: 82
        east: 264791.6
        north: 2147693.0
        height: {value: 20.9, unit: m}
      fire shore left:
        recipe: the moonlake fire
        seed: 83
        east: 263652.4
        north: 2148337.2
        height: {value: 20.9, unit: m}
      fire shore right:
        recipe: the moonlake fire
        seed: 84
        east: 263578.1
        north: 2148129.0
        height: {value: 20.9, unit: m}
`

// fail says what went wrong on stderr and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "passes:", err)
	os.Exit(1)
}

// swapped is a set with each from replaced by its to, each found once.
func swapped(set, what string, swaps []struct{ from, to string }) string {
	for _, each := range swaps {
		if strings.Count(set, each.from) != 1 {
			fail(fmt.Errorf("%s: the set does not hold %q once", what, each.from))
		}
		set = strings.Replace(set, each.from, each.to, 1)
	}
	return set
}

// write writes one pass's set and says where.
func write(dir, name, set string) {
	path := filepath.Join(dir, "moonlake-pass-"+name+".yaml")
	if err := os.WriteFile(path, []byte(set), 0o644); err != nil {
		fail(err)
	}
	fmt.Println(path)
}

// main writes every pass's set beside the set it is given.
func main() {
	if len(os.Args) != 3 {
		fail(errors.New("passes sets/moonlake.yaml sets"))
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail(err)
	}
	// Profiles are relative to the source set; generated passes can be in a
	// different directory, so preserve the referenced book when moving them.
	profile, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[1]), "../manufacture/plants/conifers/conifer-profiles.yaml"))
	if err != nil {
		fail(err)
	}
	out, err := filepath.Abs(os.Args[2])
	if err != nil {
		fail(err)
	}
	relative, err := filepath.Rel(out, profile)
	if err != nil {
		fail(err)
	}
	source := strings.ReplaceAll(string(raw), "file: ../manufacture/plants/conifers/conifer-profiles.yaml", "file: "+strconv.Quote(filepath.ToSlash(relative)))
	base := swapped(source, "the base", []struct{ from, to string }{
		{"  light: moon\n", "  # a pass: the moon's light off\n  light: none\n"},
		{"    face: svs-moon-kit-color\n",
			"    face: svs-moon-kit-color\n    # a pass: the moon's disc off\n    disc: false\n"},
	})
	const things = "  set:\n    things:\n"
	for _, f := range fixtures {
		heading := f.aim - 90
		for heading < 0 {
			heading += 360
		}
		block := fmt.Sprintf("%s      # the pass's fixture, aimed at azimuth %g\n"+
			"      %s:\n        kind: %s\n        seed: %d\n"+
			"        east: %.1f\n        north: %.1f\n"+
			"        heading: {value: %g, unit: deg}\n",
			things, f.aim, f.name, f.kind, f.seed, f.east, f.north, heading)
		write(os.Args[2], f.name, swapped(base, f.name,
			[]struct{ from, to string }{{things, block}}))
	}
	// the overhead: the ship's light, the whole scene lit from nearly
	// straight above by a sun at daylight strength; its colour is the
	// mix's to give
	write(os.Args[2], "overhead", swapped(base, "the overhead",
		[]struct{ from, to string }{
			{"  # a pass: the moon's light off\n  light: none\n",
				"  # a pass: the overhead, a sun nearly straight up\n  light: sun\n"},
			{"  sun:\n    azimuth: 304\n    altitude: -12\n",
				"  sun:\n    azimuth: 109.65\n    altitude: 86\n"},
		}))
	// the fire: the two fires alone, burning; rendered at -at a time, a
	// frame of their burning. Without the close spruces: they stand in
	// front of the fires, and lightshow hides the fires behind their
	// mask, so the fire need not be rendered again when they change
	fire := swapped(base, "the fire",
		[]struct{ from, to string }{{"  recipes:\n", fireRecipe}, {things, things + fires}})
	for _, close := range []string{
		"      close left:\n        recipe: plant-spruce\n        complexity: demo\n" +
			"        seed: 51\n        east: 263115.3\n        north: 2148423.9\n",
		"      close right:\n        recipe: plant-spruce\n        complexity: demo\n" +
			"        seed: 52\n        east: 263114.1\n        north: 2148414.8\n",
		"      close edge:\n        recipe: plant-spruce\n        complexity: demo\n" +
			"        seed: 53\n        east: 263113.8\n        north: 2148428.7\n",
	} {
		fire = swapped(fire, "the fire without the close spruces",
			[]struct{ from, to string }{{close, ""}})
	}
	write(os.Args[2], "fire", fire)
	// the swell and the burst: the same fires with the pulse's recipe and
	// the sparks' alone, where they stand
	write(os.Args[2], "fire-swell", swapped(fire, "the swell",
		[]struct{ from, to string }{{fireRecipe, swellRecipe}}))
	write(os.Args[2], "fire-burst", swapped(fire, "the burst",
		[]struct{ from, to string }{{fireRecipe, burstRecipe}}))
	// the moon, for its own layer: the scene moonlit without its disc,
	// and the disc, for lightshow to slide to where the cues stand it.
	//
	// The disc glows only by the moon's light (unlit, its pass is black),
	// so its pass is the scene moonlit with the disc at 5.45 degrees,
	// clear of the rim's giants (their tops near 3.75) and the frame's
	// top (7.1). The sky is black but for it; lightshow takes the sky
	write(os.Args[2], "nodisc", swapped(source, "the scene without its disc",
		[]struct{ from, to string }{{"    face: svs-moon-kit-color\n",
			"    face: svs-moon-kit-color\n    # a pass: the moon's disc off\n    disc: false\n"}}))
	write(os.Args[2], "disc", swapped(source, "the disc",
		[]struct{ from, to string }{
			{"    altitude: 2.3\n", "    # a pass: clear of the rim\n    altitude: 5.45\n"},
		}))
}
