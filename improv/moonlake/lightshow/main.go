// Command lightshow is improv only: moonlake's lights for one moment.
// From the plate (the scene's raw) and the mask (the same scene against a
// sky-filling disc, so the sky is lit and everything standing is black),
// it adds strobes behind the rim that light the haze there, the trees
// cutting shafts through it, and club lasers: fans of beams swept by
// mirrors, glowing along their length and brighter in mist.
//
// It writes the sky (plate, strobes, beams) for the water to reflect,
// and the front (the beams below the far shore, to lay over the water),
// and says the strobes' level on stdout.
//
//	lightshow plate.pfm mask.pfm sky.pfm front.pfm seconds
//
// Everything is in units of the moon disc's light, taken from the plate.
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/slayer/improv/moonlake/rig"
)

// show is the moment's cues when CUES names a file of cueshow's
// key=value lines; without it the pass free-runs as it always did:
// random strobes, a green fan, mirrors on their own time.
var show = cues{rig: -1, sweep: -1, white: -1, magenta: -1,
	beams: fanBeams, gate: 1, parity: -1, master: 1, moon: 1,
	moonAlt: moonAlt, figure: "fan", gateLeft: -1, gateRight: -1,
	searchMove: "sweep", searchGate: 1}

type cues struct {
	rig, sweep     float64
	beams          int
	colour         [3]float64
	hasColour      bool
	white, magenta int
	// gate is the pulse's level now, 0 to 1; parity, when not -1, is
	// which half of the fan this hit lights, even beams or odd
	gate   float64
	parity int
	// the flank fixtures: each flood's flash now, the spots' level, and
	// where their sweep is, 0 to 2 across the three aims
	floodLeft, floodRight, spots, spotAim float64
	// the searchlights behind the rim: their level and sweep's phase
	search, searchPhase float64
	// master dims the moonlit scene; overhead is the ship's light, 0 to 1
	master, overhead float64
	// fire is the shore fires' level, 0 to 1, and fireSince how long ago
	// each burst still showing began, in seconds, newest first; fireLit
	// how long ago they were lit, -1 before
	fire      float64
	fireSince []float64
	fireLit   float64
	// moon is the disc's level, its pulse in it, and moonAlt where it
	// stands, in degrees
	moon, moonAlt float64
	// figure is the lasers' look now and figurePhase how far through its
	// bars; gateLeft and gateRight each bank's pulse, below nought when
	// the cues say none and the one gate holds
	figure                           string
	figurePhase, gateLeft, gateRight float64
	// the searchlights' move, how far through it, and their gate
	searchMove                  string
	searchMovePhase, searchGate float64
}

// readCues takes what cueshow said about this moment.
func readCues(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found {
			continue
		}
		switch key {
		case "rig":
			show.rig, err = strconv.ParseFloat(value, 64)
		case "sweep":
			show.sweep, err = strconv.ParseFloat(value, 64)
		case "beams":
			show.beams, err = strconv.Atoi(value)
		case "white":
			show.white, err = strconv.Atoi(value)
		case "magenta":
			show.magenta, err = strconv.Atoi(value)
		case "gate":
			show.gate, err = strconv.ParseFloat(value, 64)
		case "parity":
			show.parity, err = strconv.Atoi(value)
		case "flood_left":
			show.floodLeft, err = strconv.ParseFloat(value, 64)
		case "flood_right":
			show.floodRight, err = strconv.ParseFloat(value, 64)
		case "spots":
			show.spots, err = strconv.ParseFloat(value, 64)
		case "spot_aim":
			show.spotAim, err = strconv.ParseFloat(value, 64)
		case "search":
			show.search, err = strconv.ParseFloat(value, 64)
		case "search_phase":
			show.searchPhase, err = strconv.ParseFloat(value, 64)
		case "master":
			show.master, err = strconv.ParseFloat(value, 64)
		case "overhead":
			show.overhead, err = strconv.ParseFloat(value, 64)
		case "search_move":
			show.searchMove = value
		case "search_move_phase":
			show.searchMovePhase, err = strconv.ParseFloat(value, 64)
		case "search_gate":
			show.searchGate, err = strconv.ParseFloat(value, 64)
		case "figure":
			show.figure = value
		case "figure_phase":
			show.figurePhase, err = strconv.ParseFloat(value, 64)
		case "gate_left":
			show.gateLeft, err = strconv.ParseFloat(value, 64)
		case "gate_right":
			show.gateRight, err = strconv.ParseFloat(value, 64)
		case "moon":
			show.moon, err = strconv.ParseFloat(value, 64)
		case "moon_alt":
			show.moonAlt, err = strconv.ParseFloat(value, 64)
		case "fire":
			show.fire, err = strconv.ParseFloat(value, 64)
		case "fire_lit":
			show.fireLit, err = strconv.ParseFloat(value, 64)
		case "fire_since":
			show.fireSince = nil
			for _, part := range strings.Split(value, ",") {
				if part == "" {
					continue
				}
				ago, err := strconv.ParseFloat(part, 64)
				if err != nil {
					return fmt.Errorf("cue fire_since %q: %w", value, err)
				}
				show.fireSince = append(show.fireSince, ago)
			}
		case "colour":
			parts := strings.Split(value, ",")
			if len(parts) != 3 {
				return fmt.Errorf("cue colour %q", value)
			}
			for c, part := range parts {
				if show.colour[c], err = strconv.ParseFloat(part,
					64); err != nil {
					return err
				}
			}
			show.hasColour = true
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// moonlake's camera: x east, y north, z up, metres from the eye. The
// first view stood 2 m over the west rise, level, at 150 mm; the camera
// now is said by MOONLAKE_EYE (metres over the lake), MOONLAKE_LOOK (its
// tilt in degrees) and MOONLAKE_FOCAL (mm), the first view's when unset.
//
// The frame is the 15/70 aperture, 70.41 mm across; MOONLAKE_FRAME_HEIGHT
// is its height in mm, 39.61 for the 16:9 crop unless told, 51.61 for
// the whole of it. The frame's edges and the far shore, in degrees,
// follow from them.
const (
	azimuth   = 109.65
	gaugeW    = 70.41
	fanBeams  = 10
	beamReach = 4000.0
)

var (
	lookDown    = numberFrom("MOONLAKE_LOOK", -0.5)
	focal       = numberFrom("MOONLAKE_FOCAL", 150)
	gaugeH      = numberFrom("MOONLAKE_FRAME_HEIGHT", 39.61)
	eyeOverLake = numberFrom("MOONLAKE_EYE", 1129.6-1076)
	waterZ      = -eyeOverLake
	half        = math.Atan(gaugeH/2/focal) * 180 / math.Pi
	top, bottom = lookDown + half, lookDown - half
	shore       = math.Atan(-eyeOverLake/1660) * 180 / math.Pi
	// the far tree, from the eye: 1727.5 m east, 617.5 m south, and as
	// far up as its ground at 1195.5 m stands over the eye
	farTree = vec{1727.5, -617.5, 1195.5 - (1076 + eyeOverLake)}
)

// fail says what went wrong on stderr, where it cannot be taken for the
// strobes' level on stdout, and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "lightshow:", err)
	os.Exit(1)
}

// numberFrom is a number from the environment, or the default.
func numberFrom(key string, otherwise float64) float64 {
	if raw := os.Getenv(key); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
		fmt.Fprintf(os.Stderr, "%s=%q is not a number\n", key, raw)
		os.Exit(2)
	}
	return otherwise
}

// vec is a point or a way from the eye: metres east, north and up.
type vec struct{ x, y, z float64 }

// add is a plus b.
func (a vec) add(b vec) vec { return vec{a.x + b.x, a.y + b.y, a.z + b.z} }

// sub is a less b.
func (a vec) sub(b vec) vec { return vec{a.x - b.x, a.y - b.y, a.z - b.z} }

// scale is a so many times over.
func (a vec) scale(s float64) vec { return vec{a.x * s, a.y * s, a.z * s} }

// dot is the dot product of a and b.
func (a vec) dot(b vec) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }

// unit is a the same way, a unit long.
func (a vec) unit() vec { return a.scale(1 / math.Sqrt(a.dot(a))) }

// cross is the cross product of a and b.
func (a vec) cross(b vec) vec {
	return vec{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}

// toward is the unit way to a compass azimuth and an altitude, degrees.
func toward(az, alt float64) vec { return vecOf(rig.Toward(az, alt)) }

// vecOf is a rig point as lightshow's vec.
func vecOf(at rig.Point) vec { return vec{at[0], at[1], at[2]} }

// layout is the show's geometry at this eye and these towers, from
// package rig, which the untraced films draw from too.
func layout() rig.Layout { return rig.NewLayout(eyeOverLake, rigDeck) }

// picture is a raw, rows from the top, three channels.
type picture struct {
	width, height int
	light         []float64
}

// at is where a pixel's three channels start in the picture's light.
func (p *picture) at(x, y int) int { return (y*p.width + x) * 3 }

// camera projects the world into the frame.
type camera struct {
	forward, right, up vec
	focal              float64
	width, height      float64
}

// newCamera is moonlake's camera for a frame of a size.
func newCamera(width, height int) camera {
	forward := toward(azimuth, lookDown)
	right := forward.cross(vec{0, 0, 1}).unit()
	return camera{forward: forward, right: right, up: right.cross(forward),
		focal: float64(width) / 2 / (gaugeW / 2 / focal),
		width: float64(width), height: float64(height)}
}

// project is where a point lands, and whether it is ahead.
func (c camera) project(p vec) (float64, float64, bool) {
	ahead := p.dot(c.forward)
	if ahead <= 1 {
		return 0, 0, false
	}
	return c.width/2 + p.dot(c.right)/ahead*c.focal,
		c.height/2 - p.dot(c.up)/ahead*c.focal, true
}

// main lays one moment's lights over the plate, and writes the sky and
// the front.
func main() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "lightshow plate.pfm mask.pfm sky.pfm front.pfm seconds")
		os.Exit(2)
	}
	seconds, err := strconv.ParseFloat(os.Args[5], 64)
	if err != nil {
		fail(err)
	}
	if path := os.Getenv("CUES"); path != "" {
		if err := readCues(path); err != nil {
			fail(err)
		}
	}
	plate, scale, err := read(os.Args[1])
	if err != nil {
		fail(err)
	}
	mask, _, err := read(os.Args[2])
	if err != nil {
		fail(err)
	}
	moon := brightest(plate)
	// MOON_DISC is the moon's disc rendered alone, at MOON_DISC_ALT
	// degrees, clear of the rim: the plate is then the scene without its
	// disc, and the disc is laid in where the moon stands now
	var disc *picture
	if path := os.Getenv("MOON_DISC"); path != "" {
		if disc, _, err = read(path); err != nil {
			fail(err)
		}
		if disc.width != plate.width || disc.height != plate.height {
			fail(fmt.Errorf("the moon disc is %dx%d, the plate %dx%d",
				disc.width, disc.height, plate.width, plate.height))
		}
		moon = brightest(disc)
	}
	cam := newCamera(plate.width, plate.height)
	open := skyOf(mask)
	rim := rimOf(open, plate.width, plate.height)
	near = nearOf(open, rim, plate.width, plate.height)
	// TREEMASK is the close spruces glowing alone: they stand in front of
	// every beam and of the fires
	var trees *picture
	if path := os.Getenv("TREEMASK"); path != "" {
		if trees, _, err = read(path); err != nil {
			fail(err)
		}
		closeTrees = coveredBy(trees)
	}
	// DEBUGMASK writes what the pass takes for sky (red), near trees
	// (green) and the rim line (blue)
	if path := os.Getenv("DEBUGMASK"); path != "" {
		shown := &picture{plate.width, plate.height, make([]float64, len(plate.light))}
		for y := range plate.height {
			for x := range plate.width {
				i := shown.at(x, y)
				if open[y*plate.width+x] {
					shown.light[i] = 1
				}
				if near[y*plate.width+x] {
					shown.light[i+1] = 1
				}
				if y == rim(x) {
					shown.light[i+2] = 1
				}
			}
		}
		if err := write(path, shown, -1); err != nil {
			fail(err)
		}
	}
	shoreRow := (top - shore) / (top - bottom) * float64(plate.height)

	sky := &picture{plate.width, plate.height, append([]float64(nil), plate.light...)}
	// the master dims the moonlit scene, and the moon's own disc with it
	// when the disc is in the plate; laid in, the disc has its own level
	for i := range sky.light {
		sky.light[i] *= show.master
	}
	if disc != nil {
		moonDisc(sky, disc, open, cam)
	}
	if dir := os.Getenv("PASSES"); dir != "" && show.overhead > 1e-4 {
		if err := overheadLight(sky, dir); err != nil {
			fail(err)
		}
	}
	front := &picture{plate.width, plate.height, make([]float64, len(plate.light))}
	// WATERMASK is the lake glowing alone: what stands in front of it
	// goes in the front, which the water does not cover
	var water *picture
	if path := os.Getenv("WATERMASK"); path != "" {
		if water, _, err = read(path); err != nil {
			fail(err)
		}
	}

	if dir := os.Getenv("PASSES"); dir != "" {
		if err := fixtures(sky, dir); err != nil {
			fail(err)
		}
	}
	if dir := os.Getenv("FIRE_FRAMES"); dir != "" && show.fire > 1e-4 {
		lay := fireLight
		if fireSwellDir != "" {
			lay = percussiveFire
		}
		fire := &picture{front.width, front.height, make([]float64, len(front.light))}
		if err := lay(fire, dir); err != nil {
			fail(err)
		}
		shoulder(fire, fireTop*struck(), math.Exp2(printStops))
		for i := range front.light {
			front.light[i] += fire.light[i]
		}
	}
	skyGlow(sky, open, rim, cam, moon)
	level := strobes(sky, open, rim, cam, moon, seconds)
	searchlights(sky, front, open, cam, moon, shoreRow)
	lasers(sky, front, open, rim, cam, moon, seconds, shoreRow)
	if trees != nil {
		backHaze(front, trees, cam, moon, seconds)
	}
	if show.overhead > 1e-4 {
		ufoBeam(sky, front, trees, water, cam, moon, seconds)
	}
	if err := write(os.Args[3], sky, scale); err != nil {
		fail(err)
	}
	if err := write(os.Args[4], front, scale); err != nil {
		fail(err)
	}
	fmt.Printf("%.4f\n", level)
}

// skyOf is, for each pixel, whether the mask shows the sky there.
func skyOf(mask *picture) []bool {
	lum := make([]float64, mask.width*mask.height)
	most := 0.0
	for i := range lum {
		lum[i] = mask.light[3*i] + mask.light[3*i+1] + mask.light[3*i+2]
		most = math.Max(most, lum[i])
	}
	open := make([]bool, len(lum))
	for i, l := range lum {
		open[i] = l > most*0.02
	}
	return open
}

// the fixtures' gains: how many times its rendered light each pass is
// laid in at full, so a flood flashes well over the moonlit scene and a
// spot pools on it; FLOOD_GAIN and SPOT_GAIN tune them
var (
	floodGain = numberFrom("FLOOD_GAIN", 400)
	spotGain  = numberFrom("SPOT_GAIN", 150)
)

// fixtures lays the flank fixtures' light passes into the sky, each its
// render of the scene lit by that fixture alone, by the cues: the left
// flood white and the right through a magenta gel, the spots blended
// between their neighbouring aims as they sweep, the left's way and the
// right's its mirror, tinted half toward the beat's colour.
func fixtures(sky *picture, dir string) error {
	tint := [3]float64{1, 1, 1}
	if show.hasColour {
		for c := range tint {
			tint[c] = 0.5 + 0.5*show.colour[c]
		}
	}
	type layer struct {
		name   string
		amount float64
		colour [3]float64
	}
	// FLOOD_PASSES names the two passes the floods fire, left then
	// right: the flank floods, or the backlights behind the close spruces
	floods := strings.Split(textFrom("FLOOD_PASSES", "flood-left,flood-right"), ",")
	if len(floods) != 2 {
		return fmt.Errorf("FLOOD_PASSES names %d passes, not two", len(floods))
	}
	// the floods' sum, package rig's, which the untraced films' traced
	// panes lay the same way
	var layers []layer
	for _, each := range rig.Floods([2]string{floods[0], floods[1]},
		show.floodLeft, show.floodRight, floodGain) {
		layers = append(layers, layer{each.Name, each.Amount, each.Colour})
	}
	aim := math.Max(0, math.Min(2, show.spotAim))
	near := math.Min(1, math.Floor(aim))
	f := aim - near
	for side, mirrored := range map[string]bool{"left": false, "right": true} {
		a, b := int(near), int(near)+1
		if mirrored {
			a, b = 2-a, 2-b
		}
		layers = append(layers,
			layer{fmt.Sprintf("spot-%s-%d", side, a), spotGain * show.spots * (1 - f), tint},
			layer{fmt.Sprintf("spot-%s-%d", side, b), spotGain * show.spots * f, tint})
	}
	for _, each := range layers {
		if each.amount < 1e-3 {
			continue
		}
		pass, _, err := read(filepath.Join(dir, each.name+".pfm"))
		if err != nil {
			return err
		}
		if pass.width != sky.width || pass.height != sky.height {
			return fmt.Errorf("%s is %dx%d, the frame %dx%d", each.name,
				pass.width, pass.height, sky.width, sky.height)
		}
		added := rig.Layer{Name: each.name, Amount: each.amount,
			Colour: each.colour}
		if err := added.Add(sky.light, pass.light); err != nil {
			return err
		}
	}
	return nil
}

// the ship's beam: a cone of coherent light from the ship BEAM_APEX
// metres over the water, far above the frame, widening as it falls to
// BEAM_RADIUS where it lands on the lake BEAM_OUT metres out along the
// view; BEAM_GAIN is how bright it is in its mist, over the moon disc's
// light, per metre crossed
var (
	beamOut    = numberFrom("BEAM_OUT", 900)
	beamRadius = numberFrom("BEAM_RADIUS", 90)
	beamApex   = numberFrom("BEAM_APEX", 450)
	beamGain   = numberFrom("BEAM_GAIN", 40)
	// BEAM_HALO is how bright the mist round the cone glows, lit by it,
	// over the cone's own brightness at its edge
	beamHalo = numberFrom("BEAM_HALO", 0.1)
)

// ray is the way from the eye through a pixel's middle, a unit long.
func (c camera) ray(x, y int) vec {
	return c.forward.add(c.right.scale((float64(x) + 0.5 - c.width/2) /
		c.focal)).add(c.up.scale(-(float64(y) + 0.5 - c.height/2) /
		c.focal)).unit()
}

// coneChord is where a ray from the eye is inside a cone hanging from an
// apex straight down, widening by slope metres a metre, between the
// water and the apex: its entry and exit in metres along the ray, and
// whether it crosses at all. Inside, the horizontal distance to the axis
// is at most slope times the depth below the apex:
//
//	(t dx - ax)^2 + (t dy - ay)^2 <= slope^2 (az - t dz)^2
//
// a quadratic in t whose negative stretch is inside both nappes; the
// heights keep only the lower one.
func coneChord(d, apex vec, slope, low float64) (float64, float64, bool) {
	k2 := slope * slope
	a := d.x*d.x + d.y*d.y - k2*d.z*d.z
	b := -2*(d.x*apex.x+d.y*apex.y) + 2*k2*apex.z*d.z
	c := apex.x*apex.x + apex.y*apex.y - k2*apex.z*apex.z
	// the stretches of t where the quadratic is not above nought
	var stretches [][2]float64
	if math.Abs(a) < 1e-12 {
		if b == 0 {
			return 0, 0, false
		}
		t := -c / b
		if b > 0 {
			stretches = append(stretches, [2]float64{math.Inf(-1), t})
		} else {
			stretches = append(stretches, [2]float64{t, math.Inf(1)})
		}
	} else {
		disc := b*b - 4*a*c
		if disc < 0 {
			if a < 0 {
				stretches = append(stretches, [2]float64{math.Inf(-1), math.Inf(1)})
			}
		} else {
			root := math.Sqrt(disc)
			t1, t2 := (-b-root)/(2*a), (-b+root)/(2*a)
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if a > 0 {
				stretches = append(stretches, [2]float64{t1, t2})
			} else {
				stretches = append(stretches,
					[2]float64{math.Inf(-1), t1}, [2]float64{t2, math.Inf(1)})
			}
		}
	}
	// the lower nappe only: between the water and the apex, ahead
	lowT, highT := 0.0, math.Inf(1)
	switch {
	case d.z > 0:
		lowT, highT = math.Max(0, low/d.z), apex.z/d.z
	case d.z < 0:
		lowT, highT = math.Max(0, apex.z/d.z), low/d.z
	default:
		if low > 0 || apex.z < 0 {
			return 0, 0, false
		}
	}
	for _, s := range stretches {
		in, out := math.Max(s[0], lowT), math.Min(s[1], highT)
		if out > in {
			return in, out, true
		}
	}
	return 0, 0, false
}

// ufoBeam lays the ship's beam: for each pixel, the stretch of its ray
// inside the column, lit by the mist there, a hard white-blue core with
// a thin red fringe and a fainter violet one outside it; and the pool
// it makes where it meets the water.
//
// The column is laid into the sky, so the water reflects it, and again
// over the water's pixels in the front, since the water is laid over the
// sky; the close spruces, nearer than the column, hide it; and water
// nearer than the column hides its foot.
func ufoBeam(sky, front, trees, water *picture, cam camera, moon [3]float64,
	seconds float64) {
	lum := moonLum(moon)
	axis := toward(azimuth, 0).scale(beamOut)
	apex := axis.add(vec{0, 0, waterZ + beamApex})
	slope := beamRadius / beamApex
	layers := []struct {
		slope  float64
		colour [3]float64
		gain   float64
	}{
		{slope, [3]float64{0.82, 0.9, 1.0}, 1},
		{slope * 1.07, [3]float64{1, 0.12, 0.08}, 0.5},
		{slope * 1.16, [3]float64{0.55, 0.15, 1}, 0.3},
	}
	most := func(p *picture) float64 {
		m := 0.0
		for _, v := range p.light {
			m = math.Max(m, v)
		}
		return m
	}
	treeMost, waterMost := 0.0, 0.0
	if trees != nil {
		treeMost = most(trees)
	}
	if water != nil {
		waterMost = most(water)
	}
	// on is whether a mask shows its thing at a pixel: more than a share
	// of its brightest
	on := func(p *picture, peak, share float64, x, y int) bool {
		if p == nil || peak == 0 {
			return false
		}
		i := p.at(x, y)
		return (p.light[i]+p.light[i+1]+p.light[i+2])/3 > share*peak
	}
	// the cone's axis on the screen: its foot on the water and its apex,
	// so a pixel's distance across from it, in the cone's own half-width
	// at that height, is known
	footX, footY, _ := cam.project(axis.add(vec{0, 0, waterZ}))
	headX, headY, _ := cam.project(apex)
	edgeX, _, _ := cam.project(axis.add(vec{0, 0, waterZ}).add(
		toward(azimuth+90, 0).scale(beamRadius)))
	footHalf := math.Max(1, math.Abs(edgeX-footX))
	for y := 0; y < sky.height; y++ {
		for x := 0; x < sky.width; x++ {
			if on(trees, treeMost, 0.3, x, y) {
				continue
			}
			// the mist round the cone, lit by it: falling off across the
			// screen within about one of the cone's half-widths, and below
			// its foot fading out over half of one
			if below := float64(y) - footY; below < 4*footHalf {
				fade := 1.0
				if below > 0 {
					fade = math.Exp(-below / (0.5 * footHalf))
				}
				f := 0.0
				if headY != footY {
					f = math.Max(0, (float64(y)-footY)/(headY-footY))
				}
				ax := footX + f*(headX-footX)
				halfWidth := math.Max(1, footHalf*(1-f))
				across := math.Abs(float64(x)-ax)/halfWidth - 1
				// below the foot no column fills the cone's own width, so
				// the mist there is lit as it is at the column's edge, not
				// left dark: a hole there showed as a hard-sided column
				// down the water (2026-11-11, at 2:27)
				if below > 0 && across < 0 {
					across = 0
				}
				if across > 0 || below > 0 {
					u := float64(x)/(60*float64(sky.width)/1280) + 0.2*seconds
					v := float64(y)/(25*float64(sky.width)/1280) - 0.1*seconds
					wisps := 0.4 + 0.6*smooth(0.3, 0.75, curled(u, v, seconds, 29))
					radius := beamRadius * (1 - f)
					halo := beamHalo * beamGain * show.overhead * lum *
						2 * radius * wisps * math.Exp(-across/0.9) * fade
					i := sky.at(x, y)
					wet := on(water, waterMost, 0.3, x, y)
					for c := range 3 {
						tint := [3]float64{0.7, 0.8, 1.0}[c]
						sky.light[i+c] += halo * tint
						if wet {
							front.light[i+c] += halo * tint
						}
					}
				}
			}
			d := cam.ray(x, y)
			// the water the ray meets first, if it meets any
			hitWater := math.Inf(1)
			if d.z < 0 {
				hitWater = waterZ / d.z
			}
			var light [3]float64
			inner := 0.0
			for _, layer := range layers {
				in, out, crosses := coneChord(d, apex, layer.slope, waterZ)
				if !crosses || in >= hitWater {
					continue
				}
				out = math.Min(out, hitWater)
				// the mist along the stretch, at three points
				mist := 0.0
				for _, f := range []float64{0.2, 0.5, 0.8} {
					p := d.scale(in + f*(out-in))
					mist += (0.35 + 6*fog(p, seconds)) / 3
				}
				length := out - in
				shell := length - inner
				inner = length
				if shell <= 0 {
					continue
				}
				amount := beamGain * show.overhead * lum * mist * shell * layer.gain
				for c := range 3 {
					light[c] += amount * layer.colour[c]
				}
			}
			// the pool: where the ray meets the water inside the column
			if !math.IsInf(hitWater, 1) {
				p := d.scale(hitWater)
				r := math.Hypot(p.x-axis.x, p.y-axis.y)
				if r < beamRadius*1.3 {
					edge := smooth(beamRadius*1.3, beamRadius*0.9, r)
					amount := beamGain * show.overhead * lum * 60 * edge
					for c := range 3 {
						light[c] += amount * layers[0].colour[c]
					}
				}
			}
			if light == [3]float64{} {
				continue
			}
			i := sky.at(x, y)
			wet := on(water, waterMost, 0.3, x, y)
			for c := range 3 {
				sky.light[i+c] += light[c]
				if wet {
					front.light[i+c] += light[c]
				}
			}
		}
	}
}

// the overhead: OVERHEAD_GAIN times the pass's rendered daylight at
// full, through package rig's OverheadColour
var overheadGain = numberFrom("OVERHEAD_GAIN", 1)

// the fires: FIRE_FRAMES is the fire pass rendered from ignition, NNN.pfm
// at FIRE_FPS a second; FIRE_GAIN how much of its rendered light is laid
// in at full, and FIRE_DECAY the seconds a burst takes to fall to a third
var (
	fireGain  = numberFrom("FIRE_GAIN", 0.1)
	fireDecay = numberFrom("FIRE_DECAY", 1)
	fireFPS   = numberFrom("FIRE_FPS", 24)
)

// fireLight lays in the fires: for each burst still showing, its fire
// that long after ignition, flaring at the downbeat and falling by
// FIRE_DECAY, so each burst's sparks climb on under the next. A burst
// past the rendered frames has burnt out; a frame missing among them is
// an error.
//
// The pass is rendered with the lake, and the GPU mirrors the flames and
// the sparks in it, so the pass carries its own reflection, true to the
// geometry. All of it goes in the front, which the water does not lay
// over and does not reflect again.
//
// Until 2026-11-10 the light that was not over water went in the sky for
// the water to reflect, and the water mirrored the sparks a second time,
// in the wrong place: their reflection was seen to start somewhere the
// fire's did not, and the sparks over the fire begin at a flat line, the
// far shore, where they changed layers.
func fireLight(front *picture, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	frames := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pfm") {
			frames++
		}
	}
	for _, ago := range show.fireSince {
		index := int(math.Round(ago * fireFPS))
		if index >= frames {
			continue
		}
		pass, _, err := read(filepath.Join(dir, fmt.Sprintf("%03d.pfm", index)))
		if err != nil {
			return err
		}
		if pass.width != front.width || pass.height != front.height {
			return fmt.Errorf("fire frame %d is %dx%d, the frame %dx%d",
				index, pass.width, pass.height, front.width, front.height)
		}
		amount := fireGain * show.fire * math.Exp(-ago/fireDecay)
		for i := range front.light {
			// the fire pass is rendered without the close spruces; where
			// they stand, they hide it
			if closeTrees != nil && closeTrees[i/3] {
				continue
			}
			front.light[i] += amount * pass.light[i]
		}
	}
	return nil
}

// the percussive fire, 2026-11-10: "it needs to be percussive.
// because something is igniting a huge fire in a hurry ... the fire is
// going to pulse on the downbeat ... and it's quick and violent". With
// FIRE_SWELL (the swell pass's frames) and FIRE_BURST (the burst's), the
// fire burns at full from the moment it is lit, and every downbeat
// strikes it.
//
// The steady fire is FIRE_FRAMES from FIRE_STEADY_FROM on, a burning
// fire, restarted on each downbeat so its seam hides under the strike.
//
// On the beat its light leaps FIRE_PULSE times over, FIRE_FLASH at the
// ignition, and falls back within FIRE_SNAP seconds; the swell, the fire
// half again as big, cuts in on the beat and gives way to the steady
// fire over FIRE_SWELL_FALL; and each beat throws a burst of sparks, the
// ignition's twice as bright.
var (
	fireSwellDir   = os.Getenv("FIRE_SWELL")
	fireBurstDir   = os.Getenv("FIRE_BURST")
	fireSteadyFrom = numberFrom("FIRE_STEADY_FROM", 42)
	firePulse      = numberFrom("FIRE_PULSE", 3)
	fireFlash      = numberFrom("FIRE_FLASH", 8)
	fireSnap       = numberFrom("FIRE_SNAP", 0.12)
	fireSwellFall  = numberFrom("FIRE_SWELL_FALL", 0.3)
	fireBurstGain  = numberFrom("FIRE_BURST_GAIN", 1)
)

// percussiveFire lays in the fire struck on the downbeats, from the
// steady frames in dir, the swell's and the burst's.
func percussiveFire(front *picture, dir string) error {
	if show.fireLit < 0 || len(show.fireSince) == 0 {
		return nil
	}
	beat := show.fireSince[0]
	level := fireGain * show.fire * struck()
	swell := math.Exp(-beat / fireSwellFall)
	steady := int(fireSteadyFrom) + int(beat*fireFPS)
	if err := layFrame(front, dir, steady, level*(1-swell)); err != nil {
		return err
	}
	if err := layFrame(front, fireSwellDir, int(beat*fireFPS), level*swell); err != nil {
		return err
	}
	if fireBurstDir == "" {
		return nil
	}
	for _, ago := range show.fireSince {
		thrown := fireGain * show.fire * fireBurstGain
		if math.Abs(ago-show.fireLit) < 1e-3 {
			thrown *= 2
		}
		frames, err := framesIn(fireBurstDir)
		if err != nil {
			return err
		}
		if index := int(math.Round(ago * fireFPS)); index < frames {
			if err := layFrame(front, fireBurstDir, index, thrown); err != nil {
				return err
			}
		}
	}
	return nil
}

// struck is how many times over the fire burns for the newest
// downbeat's strike: FIRE_FLASH more at the ignition, FIRE_PULSE more at
// any other downbeat, falling back within FIRE_SNAP seconds. It is one
// when the fire is not percussive, and before the fire is lit.
func struck() float64 {
	if fireSwellDir == "" || show.fireLit < 0 || len(show.fireSince) == 0 {
		return 1
	}
	beat := show.fireSince[0]
	spike := firePulse
	if math.Abs(beat-show.fireLit) < 1e-3 {
		spike = fireFlash
	}
	return 1 + spike*math.Exp(-beat/fireSnap)
}

// The fire's shoulder. The fire passes reach the print at hundreds of
// times white, and develop's shoulder prints anything past its top as
// plain white, so a flame laid in as it is prints as a white rod.
//
// The fire is rolled off by its own luminance before it is laid in, so it
// never passes FIRE_TOP whites of the print and keeps its colour. The
// ceiling rises with each strike, by struck, so the ignition and the
// downbeats flash white-hot and settle back to orange.
//
// `STOPS` is the print's exposure, as render.sh develops it: light of one
// over two to the `STOPS` prints as white.
var (
	fireTop    = numberFrom("FIRE_TOP", 0.6)
	printStops = numberFrom("STOPS", 23)
)

// shoulder rolls each of the fire's pixels off by its luminance, where
// gain is the whites of the print in a unit of light, so that it nears
// top whites and never passes them: top times one minus e to the minus
// luminance over top. A faint pixel keeps nearly all its light.
//
// Red, green and blue keep their proportions, so the colour is kept. A
// pixel of endless light, whose colour cannot be told, is the ceiling
// in white.
func shoulder(fire *picture, top, gain float64) {
	for i := 0; i+2 < len(fire.light); i += 3 {
		r, g, b := fire.light[i], fire.light[i+1], fire.light[i+2]
		y := (0.2126*r + 0.7152*g + 0.0722*b) * gain
		if math.IsInf(y, 1) {
			for c := range 3 {
				fire.light[i+c] = top / gain
			}
			continue
		}
		if !(y > 0) {
			continue
		}
		scale := -top * math.Expm1(-y/top) / y
		fire.light[i] = r * scale
		fire.light[i+1] = g * scale
		fire.light[i+2] = b * scale
	}
}

// framesIn is how many NNN.pfm frames a pass's directory holds.
func framesIn(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	frames := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pfm") {
			frames++
		}
	}
	return frames, nil
}

// layFrame adds a pass's frame at index, or its last when there are not
// so many, into the front, amount times its light, where the close
// spruces do not hide it (the fire passes are rendered without them).
func layFrame(front *picture, dir string, index int, amount float64) error {
	if amount <= 0 {
		return nil
	}
	frames, err := framesIn(dir)
	if err != nil {
		return err
	}
	if frames == 0 {
		return fmt.Errorf("%s holds no frames", dir)
	}
	index = min(index, frames-1)
	pass, _, err := read(filepath.Join(dir, fmt.Sprintf("%03d.pfm", index)))
	if err != nil {
		return err
	}
	if pass.width != front.width || pass.height != front.height {
		return fmt.Errorf("%s frame %d is %dx%d, the frame %dx%d", dir, index,
			pass.width, pass.height, front.width, front.height)
	}
	for i := range front.light {
		if closeTrees != nil && closeTrees[i/3] {
			continue
		}
		front.light[i] += amount * pass.light[i]
	}
	return nil
}

// overheadLight lays the overhead pass, the scene lit from nearly
// straight above at daylight strength, in at the cue's level.
func overheadLight(sky *picture, dir string) error {
	pass, _, err := read(filepath.Join(dir, "overhead.pfm"))
	if err != nil {
		return err
	}
	if pass.width != sky.width || pass.height != sky.height {
		return fmt.Errorf("overhead is %dx%d, the frame %dx%d",
			pass.width, pass.height, sky.width, sky.height)
	}
	return rig.Overhead(show.overhead, overheadGain).Add(sky.light,
		pass.light)
}

// moonDisc lays the moon's disc in where it stands now: the disc pass,
// the scene moonlit with the disc at MOON_DISC_ALT, its sky only (black
// but for the disc), slid up or down to the cue's altitude along the
// moon's own azimuth, at the cue's level, and only where the frame shows
// sky, so the rim and its trees stand in front of it at any height.
func moonDisc(sky, disc *picture, open []bool, cam camera) {
	_, from, _ := cam.project(toward(azimuth, numberFrom("MOON_DISC_ALT", 5.45)).scale(1e6))
	_, to, _ := cam.project(toward(azimuth, show.moonAlt).scale(1e6))
	shift := to - from
	// the pass's light at a pixel when the mask shows sky there, or none:
	// its moonlit land is the plate's to give
	skyOnly := func(x, y, c int) float64 {
		if !open[y*disc.width+x] {
			return 0
		}
		return disc.light[disc.at(x, y)+c]
	}
	for y := range sky.height {
		source := float64(y) - shift
		low := int(math.Floor(source))
		if low < 0 || low+1 >= disc.height {
			continue
		}
		f := source - float64(low)
		for x := range sky.width {
			if !open[y*sky.width+x] {
				continue
			}
			i := sky.at(x, y)
			for c := range 3 {
				sky.light[i+c] += show.moon *
					((1-f)*skyOnly(x, low, c) + f*skyOnly(x, low+1, c))
			}
		}
	}
}

// skyGlowGain (SKY_GLOW) is how bright the night sky is, over the moon
// disc's light: the rim and its trees stand black against it, so the
// land has a shape and the moon has something to be behind. The moon is
// at MOONLAKE_MOON_ALT degrees, its disc MOONLAKE_MOON_SCALE true radii.
var (
	skyGlowGain = numberFrom("SKY_GLOW", 0.02)
	moonAlt     = numberFrom("MOONLAKE_MOON_ALT", 3.0)
	moonScale   = numberFrom("MOONLAKE_MOON_SCALE", 6)
)

// skyGlow lays the night sky where the frame shows sky: a cool glow,
// stronger toward the horizon, and a halo round the moon falling over
// some four of its radii.
func skyGlow(sky *picture, open []bool, rim func(int) int, cam camera,
	moon [3]float64) {
	if skyGlowGain <= 0 {
		return
	}
	lum := moonLum(moon)
	mx, my, ahead := cam.project(toward(azimuth, show.moonAlt).scale(1e6))
	radius := math.Tan(moonScale*0.259*math.Pi/180) * cam.focal
	colour := [3]float64{0.55, 0.66, 0.95}
	for y := 0; y < sky.height; y++ {
		for x := 0; x < sky.width; x++ {
			if !open[y*sky.width+x] {
				continue
			}
			above := float64(rim(x) - y)
			band := 0.4 + 0.6*math.Exp(-above/(0.18*float64(sky.height)))
			halo := 0.0
			if ahead {
				d := math.Hypot(float64(x)-mx, float64(y)-my) / radius
				halo = 3 * math.Exp(-math.Max(0, d-1)/3)
			}
			glow := skyGlowGain * lum * (band + halo*show.moon) * show.master
			i := sky.at(x, y)
			for c := range 3 {
				sky.light[i+c] += glow * colour[c]
			}
		}
	}
}

// searchGain (SEARCH_GAIN) is how bright a searchlight's beam is in the
// high haze, over the moon disc's light, per pixel of its length
var searchGain = numberFrom("SEARCH_GAIN", 0.05)

// searchCount (SEARCH_COUNT) is how many searchlights stand behind the
// rim, half on either side of the moon: eight, from "four times as
// many", 2026-11-10
var searchCount = int(numberFrom("SEARCH_COUNT", 8))

// searchlights draws big beams from behind the rim, 200 m past the far
// tree and 140 m apart either side of it: each a cone widening 2.5
// degrees either side, lit by the high haze, thinning as it climbs, at
// the cue's level struck on the beat, and seen only where the frame
// shows sky, so the rim and its trees cut its foot. How they move is the
// cue's move:
//
//   - sweep: tilted 30 to 55 degrees up and swung outward, the further
//     out the further, each on the cue's phase and its own offset
//   - chase: the same, one light at a time bright, running across them
//     twice in the move's bars, the rest low
//   - fan: all of them opening outward and down together and closing
//     upward again, once in the move's bars
//   - viewer: all of them turned on the eye, a little apart, so they come
//     across the lake at the camera, the close spruces before them, and
//     each flares where it clears the rim, the harder the straighter at us
func searchlights(sky, front *picture, open []bool, cam camera,
	moon [3]float64, shoreRow float64) {
	level := show.search * show.searchGate
	if level < 0.01 {
		return
	}
	lum := moonLum(moon)
	colour := [3]float64{0.85, 0.9, 1}
	stage := layout()
	running := rig.Running(searchCount, show.searchMovePhase)
	for k := range searchCount {
		origin := vecOf(stage.Searchlight(k))
		bright := level
		if show.searchMove == "chase" && k != running {
			bright *= 0.12
		}
		dir := vecOf(stage.SearchWay(show.searchMove, k, searchCount,
			show.searchPhase, show.searchMovePhase))
		toEye := dir.dot(origin.scale(-1).unit())
		cleared := false
		const spread = 2.5 * math.Pi / 180
		for s := 5.0; s < 6000; {
			p := origin.add(dir.scale(s))
			x, y, ahead := cam.project(p)
			dist := math.Sqrt(p.dot(p))
			sigma := math.Max(0.8, s*math.Tan(spread)/dist*cam.focal)
			// a step of half the beam's width on the screen, in metres
			step := math.Max(1, 0.5*sigma*dist/cam.focal)
			s += step
			if !ahead || dist < 30 {
				continue
			}
			// a beam at us stops 500 m out, wider by then than the frame
			if show.searchMove == "viewer" && dist < 500 {
				break
			}
			// the haze thins as the beam climbs, and is no thicker below
			// the rim than at it
			haze := math.Min(1, math.Exp(-(p.z-farTree.z)/900))
			sigma = math.Min(sigma, 40*cam.width/1280)
			pixels := step / dist * cam.focal
			amount := searchGain * bright * lum * haze * pixels * sigma
			if show.searchMove != "viewer" {
				splatSky(sky, open, x, y, sigma, amount, colour)
				continue
			}
			// coming at us the beam crosses the lake below the rim, in
			// front of the water; the close spruces stand before it
			xi, yi := int(x), int(y)
			if xi < 0 || yi < 0 || xi >= sky.width || yi >= sky.height {
				continue
			}
			if closeTrees != nil && closeTrees[yi*sky.width+xi] {
				continue
			}
			if !cleared && open[yi*sky.width+xi] {
				cleared = true
				// where it clears the rim, the source's glare: the harder
				// the more nearly the beam points at the eye
				glare := math.Pow(math.Max(0, (toEye-0.995)/0.005), 3)
				k := cam.width / 1280
				splat(sky, front, x, y, 6*k, glare*lum*bright*30*k, colour, shoreRow)
				splat(sky, front, x, y, 1.2*k, glare*lum*bright*20*k, colour, shoreRow)
			}
			splat(sky, front, x, y, sigma, amount, colour, shoreRow)
		}
	}
}

// splatSky lays a gaussian into the sky only where the frame shows sky.
func splatSky(sky *picture, open []bool, x, y, sigma, amount float64,
	colour [3]float64) {
	r := int(math.Ceil(2.5 * sigma))
	if x < -float64(r) || y < -float64(r) || x > float64(sky.width+r) ||
		y > float64(sky.height+r) {
		return
	}
	norm := amount / (2 * math.Pi * sigma * sigma)
	for yy := max(0, int(y)-r); yy <= min(sky.height-1, int(y)+r); yy++ {
		for xx := max(0, int(x)-r); xx <= min(sky.width-1, int(x)+r); xx++ {
			if !open[yy*sky.width+xx] {
				continue
			}
			dx, dy := float64(xx)+0.5-x, float64(yy)+0.5-y
			wgt := norm * math.Exp(-(dx*dx+dy*dy)/(2*sigma*sigma))
			i := sky.at(xx, yy)
			for c := range 3 {
				sky.light[i+c] += wgt * colour[c]
			}
		}
	}
}

// textFrom is a word from the environment, or the default.
func textFrom(key, otherwise string) string {
	if raw := os.Getenv(key); raw != "" {
		return raw
	}
	return otherwise
}

// the backlights stand 45 m ahead past the close spruces, 12 m either
// side, a metre and a half over their ground at about 1125 m; hazeGain
// (HAZE_GAIN) is how bright the haze they light is, over the moon's disc
var hazeGain = numberFrom("HAZE_GAIN", 0.6)

// backlights is where the two backlights stand, from the eye.
func backlights() [2]vec {
	at := layout().Backlights()
	return [2]vec{vecOf(at[0]), vecOf(at[1])}
}

// backHaze lays the haze each backlight lights into the front, over the
// water: each pixel's mist, gathered along the line back to the light,
// so the close spruces (the tree mask, which stands nearer than the
// haze) cut black shafts through it, and the trees themselves stay dark.
// The left light is white, the right through the magenta gel.
func backHaze(front, trees *picture, cam camera, moon [3]float64,
	seconds float64) {
	most := 0.0
	for _, v := range trees.light {
		most = math.Max(most, v)
	}
	if most == 0 {
		return
	}
	tree := func(x, y int) bool {
		i := trees.at(x, y)
		return trees.light[i]+trees.light[i+1]+trees.light[i+2] > 0.3*most
	}
	lum := moonLum(moon)
	colours := [2][3]float64{{1, 1, 1}, {1, 0.25, 0.85}}
	levels := [2]float64{show.floodLeft, show.floodRight}
	for side, at := range backlights() {
		level := levels[side]
		if level < 0.01 {
			continue
		}
		sx, sy, ahead := cam.project(at)
		if !ahead {
			continue
		}
		const steps = 32
		for y := 0; y < front.height; y++ {
			for x := 0; x < front.width; x++ {
				if tree(x, y) {
					continue
				}
				dx, dy := sx-float64(x), sy-float64(y)
				jitter := lattice(int64(x), int64(y), 41)
				gathered := 0.0
				for k := 0; k < steps; k++ {
					f := (float64(k) + jitter) / steps
					px, py := int(float64(x)+dx*f), int(float64(y)+dy*f)
					if px < 0 || py < 0 || px >= front.width || py >= front.height {
						gathered += 1.0 / steps
						continue
					}
					if tree(px, py) {
						break
					}
					gathered += 1.0 / steps
				}
				// the noise and the reach in widths of the frame, so a
				// draft and a final show the same haze
				scale := float64(front.width) / 1280
				u := float64(x)/(240*scale) + 0.15*seconds
				v := float64(y)/(90*scale) - 0.05*seconds
				mist := 0.35 + 0.65*smooth(0.35, 0.75, curled(u, v, seconds, 23))
				reach := 0.25 * float64(front.width)
				d2 := (dx*dx + dy*dy) / (reach * reach)
				glow := hazeGain * lum * level * gathered * mist / (1 + d2)
				i := front.at(x, y)
				for c := range 3 {
					front.light[i+c] += glow * colours[side][c]
				}
			}
		}
	}
}

// rimOf is, for each column, the row below which nothing is sky: the
// rim's line, so a near tree standing above it is not taken for it.
func rimOf(open []bool, width, height int) func(x int) int {
	rows := make([]int, width)
	for x := range width {
		rows[x] = 0
		for y := height - 1; y >= 0; y-- {
			if open[y*width+x] {
				rows[x] = y + 1
				break
			}
		}
	}
	// the lowest sky over 61 columns: a tree raises a column's line, and
	// the land's line is the lowest the sky reaches nearby
	const half = 30
	smoothed := make([]int, width)
	for x := range width {
		for k := max(0, x-half); k <= min(width-1, x+half); k++ {
			smoothed[x] = max(smoothed[x], rows[k])
		}
	}
	return func(x int) int { return smoothed[clampInt(x, width)] }
}

// near is the near spruces' pixels: what stands black above the rim and
// reaches the top 40 rows, found by filling from there. They stand 30 m
// off, in front of the haze, so they cut no shafts in it.
var near []bool

// closeTrees is the close spruces' pixels, from TREEMASK, or nil: they
// stand 7 to 12 m off, in front of every beam and of the fires, so no
// beam lights them and no fire shows through them.
var closeTrees []bool

// coveredBy is the pixels a mask shows lit, at least 0.3 of its
// brightest, as backHaze takes them.
func coveredBy(mask *picture) []bool {
	most := 0.0
	for _, v := range mask.light {
		most = math.Max(most, v)
	}
	covered := make([]bool, mask.width*mask.height)
	if most == 0 {
		return covered
	}
	for i := range covered {
		covered[i] = mask.light[3*i]+mask.light[3*i+1]+mask.light[3*i+2] > 0.3*most
	}
	return covered
}

// nearOf finds the near spruces' pixels (see near): what is not sky,
// above the rim, filled from the top rows down.
func nearOf(open []bool, rim func(int) int, width, height int) []bool {
	found := make([]bool, len(open))
	var stack []int
	// a near spruce reaches the top 40 rows; the rim's trees do not
	for i := range min(40, height) * width {
		if !open[i] {
			found[i] = true
			stack = append(stack, i)
		}
	}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := i%width, i/width
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= width || ny >= height ||
				ny >= rim(nx)-2 {
				continue
			}
			j := ny*width + nx
			if !open[j] && !found[j] {
				found[j] = true
				stack = append(stack, j)
			}
		}
	}
	return found
}

// haze is how much mist hangs at a pixel at a time: the lake's, low and
// thick, and the crater's above the rim, thinning upward, both curling.
func haze(x, y, rim, shoreRow, height float64, seconds float64) float64 {
	// lengths in a 720-row frame's pixels, scaled to this frame's
	k := height / 720
	above := rim - y
	behind := 0.0
	if above > -40*k {
		behind = math.Exp(-math.Max(0, above)/(90*k)) * smooth(-40*k, 0, above)
	}
	lake := 0.0
	if y > shoreRow-30*k {
		lake = smooth(shoreRow-30*k, shoreRow, y) * (0.5 + 0.5*math.Exp(-(y-shoreRow)/(height-shoreRow)*1.5))
	}
	u := x/(220*k) + 0.1*seconds
	v := y/(60*k) - 0.03*seconds
	wisps := 0.35 + 0.65*smooth(0.35, 0.75, curled(u, v, seconds, 7))
	return (behind + lake) * wisps
}

// strobes fires two strobes behind the rim at moments a seed picks, and
// lays their light on the haze behind the trees: each sky pixel's haze,
// gathered along the line back to the strobe, so whatever stands between
// cuts a shaft. It is their level now, 0 to 1.
func strobes(sky *picture, open []bool, rim func(int) int, cam camera,
	moon [3]float64, seconds float64) float64 {
	frame := int(math.Round(seconds * 12))
	// the two strobes behind the far tree (package rig): white, and a
	// magenta gel
	places := layout().Strobes()
	lights := []struct {
		at     vec
		colour [3]float64
		seed   uint64
	}{
		{vecOf(places[0]), [3]float64{1, 1, 1}, 1},
		{vecOf(places[1]), [3]float64{1, 0.25, 0.85}, 2},
	}
	shoreRow := (top - shore) / (top - bottom) * float64(sky.height)
	total := 0.0
	for _, light := range lights {
		// a cue overrides the lattice: 0 holds a strobe dark, 1 fires
		// it, and less than nought leaves it to its own luck (rig)
		forced := show.white
		if light.seed == 2 {
			forced = show.magenta
		}
		if !rig.StrobeFires(light.seed, forced, frame) {
			continue
		}
		total += 0.5
		sx, sy, ahead := cam.project(light.at)
		if !ahead {
			continue
		}
		const steps = 48
		for y := 0; y < sky.height; y++ {
			for x := 0; x < sky.width; x++ {
				dx, dy := sx-float64(x), sy-float64(y)
				gathered := 0.0
				// a jittered start, so the steps leave no stripes
				jitter := lattice(int64(x), int64(y), 99)
				for k := 0; k < steps; k++ {
					f := (float64(k) + jitter) / steps
					px := int(float64(x) + dx*f)
					py := int(float64(y) + dy*f)
					if px < 0 || py < 0 || px >= sky.width || py >= sky.height {
						continue
					}
					j := py*sky.width + px
					if !open[j] && !near[j] {
						// the rim or its trees in the way: the shaft is cut
						break
					}
					gathered += 1.0 / steps
				}
				h := haze(float64(x), float64(y), float64(rim(x)), shoreRow,
					float64(sky.height), seconds)
				reach := 420 * float64(sky.width) / 1280
				d2 := (dx*dx + dy*dy) / (reach * reach)
				glow := 1.4 * gathered * h / (1 + d2)
				// the trees' edges catch it too, a little
				if !open[y*sky.width+x] && y < rim(x)+6 {
					glow *= 0.08
				}
				i := sky.at(x, y)
				for c := range 3 {
					sky.light[i+c] += glow * light.colour[c] * moonLum(moon)
				}
			}
		}
	}
	return total
}

// the shore installations, on the left shore and the right tongue, cast
// from row 640 of the 2K frame, 8 m over the water, from the eye; the
// right beside its fire, the left where its fire first stood
var (
	shoreLeft  = vec{263652.4 - 263104.5, 2148337.2 - 2148423.5, waterZ + 8}
	shoreRight = vec{263578.1 - 263104.5, 2148129.0 - 2148423.5, waterZ + 8}
)

// the laser towers, sets/moonlake.yaml's rigs: the ground under each by
// the lidar (improv/moonlake/rig-heights.yaml), rim left and right, then
// shore left and right; and RIG_DECK, how far over its ground a head
// stands on its tower's top deck, about 20.4 m on the stage tower.
// unset, the heads stand where they stood before the towers: 6 m over
// the far tree's ground, and 8 m over the water.
var (
	rigGround = [4]float64{1180.2, 1191.3, 1076.1, 1079.4}
	rigDeck   = numberFrom("RIG_DECK", 0)
)

// onDeck is a head's place lifted onto its tower's top deck, when the
// towers stand, and where it was when they do not.
func onDeck(head vec, tower int) vec {
	if rigDeck <= 0 {
		return head
	}
	head.z = rigGround[tower] - (1076 + eyeOverLake) + rigDeck
	return head
}

// bearing is the compass azimuth from one point to another, in degrees.
func bearing(from, to vec) float64 {
	return math.Atan2(to.x-from.x, to.y-from.y) * 180 / math.Pi
}

// lasers plays the heads: four, two on the far rim and two on the shores,
// in two banks, the left heads and the right, each bank on its own pulse,
// each head wearing the moment's figure (see figureWays), mirrors
// sweeping its heading and tilt and opening and closing it, and each
// lighting its aperture as it fires, so the beams come from somewhere.
func lasers(sky, front *picture, open []bool, rim func(int) int, cam camera,
	moon [3]float64, seconds, shoreRow float64) {
	// two heads on the rim, 120 m either side of the far tree, the moon
	// between them, aimed back at us; the right one mirrors the left's
	// sweep, and on a split pulse lights the half of its fan the left one
	// does not.
	//
	// and two on the shores, 8 m up, aimed across the lake at each other
	// and lifted 7 degrees from the rim heads' tilt, so their fans run
	// low through the thickest mist, under the rim's
	heads := layout().Heads()
	// green, 532 nm, as linear srgb near enough
	colour := [3]float64{0.08, 1, 0.12}
	// a cue names the moment's colour and scales the fan by the rig;
	// a rig of nought is the blackout
	if show.rig == 0 {
		return
	}
	if show.hasColour {
		colour = show.colour
	}
	if show.rig > 0 {
		for c := range colour {
			colour[c] *= show.rig
		}
	}
	// the pulse: each bank snaps on with its hits and falls before the
	// next; without banks from the cues, both play the one gate
	gates := [2]float64{show.gate, show.gate}
	if show.gateLeft >= 0 {
		gates[0] = show.gateLeft
	}
	if show.gateRight >= 0 {
		gates[1] = show.gateRight
	}
	if gates[0] < 0.01 && gates[1] < 0.01 {
		return
	}
	// the fan tilts down 2.3 to 6.7 degrees: the rim stands 65.9 m
	// over the eye at the default MOONLAKE_EYE, so a beam must dip more
	// than 2.1 to pass below it, and at these it comes down through the
	// mist on the lake in front of us.
	// it sweeps 45 degrees either way, so some beams cross side-on.
	// a cued sweep runs on the music's bars, not the clock's seconds
	swing, tilt, spread := rig.Sweep(seconds, show.sweep)
	for _, head := range heads {
		gate := gates[head.Bank]
		if gate < 0.01 {
			continue
		}
		lit := colour
		for c := range lit {
			lit[c] *= gate
		}
		heading := head.Aim + head.Mirror*swing
		ways, weight, split := figureWays(show.figure, heading,
			tilt+head.Lift, spread, show.beams, show.figurePhase)
		for b, way := range ways {
			// a split pulse lights the even beams on one hit, the odd on
			// the next; the right head the other half
			if split && show.parity >= 0 && (b+head.Flip)%2 != show.parity {
				continue
			}
			var each [3]float64
			for c := range each {
				each[c] = lit[c] * weight
			}
			beam(sky, front, open, rim, cam, vecOf(head.Origin), way, each, moon,
				seconds, shoreRow)
		}
		aperture(sky, front, cam, vecOf(head.Origin), lit, moon, shoreRow)
	}
}

// figureWays is the ways a head's beams go in a figure, and how bright
// each is, and whether a split pulse halves it: package rig's, which the
// untraced films draw from too.
func figureWays(figure string, heading, tilt, spread float64, beams int,
	phase float64) ([]vec, float64, bool) {
	points, weight, split := rig.FigureWays(figure, heading, tilt, spread,
		beams, phase)
	ways := make([]vec, len(points))
	for i, at := range points {
		ways[i] = vecOf(at)
	}
	return ways, weight, split
}

// aperture lights a head's mouth as it fires: a hard point and a soft
// glow round it in the beams' colour going white at the core, so the
// beams come out of something.
func aperture(sky, front *picture, cam camera, at vec, colour,
	moon [3]float64, shoreRow float64) {
	x, y, ahead := cam.project(at)
	if !ahead || x < 0 || y < 0 || x >= cam.width || y >= cam.height {
		return
	}
	if closeTrees != nil && closeTrees[int(y)*sky.width+int(x)] {
		return
	}
	lum := moonLum(moon)
	k := cam.width / 1280
	most := math.Max(colour[0], math.Max(colour[1], colour[2]))
	core := [3]float64{}
	for c := range core {
		core[c] = 0.5*colour[c] + 0.5*most
	}
	splat(sky, front, x, y, 0.8*k, lum*40*k, core, shoreRow)
	splat(sky, front, x, y, 5*k, lum*25*k, colour, shoreRow)
}

// fog is how much mist hangs at a point in the world at a time: a layer
// filling the crater, thick at the water and thinning upward over some
// ninety metres, in drifting blobs with clear air between. The lasers
// are seen by it.
func fog(p vec, seconds float64) float64 {
	above := p.z - waterZ
	if above < 0 {
		return 0
	}
	layer := math.Exp(-above / 90)
	blobs := fbm3(p.x/45+0.6*seconds, p.y/45-0.2*seconds,
		p.z/18+0.1*seconds, 17)
	return layer * (0.03 + 0.97*smooth(0.45, 0.7, blobs))
}

// fbm3 is five octaves of value noise in three dimensions, 0 to about 1.
func fbm3(u, v, w float64, seed uint64) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for octave := range 4 {
		sum += amp * noise3(u, v, w, seed+uint64(octave)*7919)
		norm += amp
		u, v, w, amp = u*2.03, v*2.03, w*2.03, amp*0.5
	}
	return sum / norm
}

// noise3 is smooth value noise on a unit lattice in three dimensions.
func noise3(u, v, w float64, seed uint64) float64 {
	x0, y0, z0 := math.Floor(u), math.Floor(v), math.Floor(w)
	fx, fy, fz := u-x0, v-y0, w-z0
	sx, sy, sz := fx*fx*(3-2*fx), fy*fy*(3-2*fy), fz*fz*(3-2*fz)
	ix, iy, iz := int64(x0), int64(y0), int64(z0)
	at := func(dx, dy, dz int64) float64 {
		return lattice(ix+dx, iy+dy*7349+(iz+dz)*104729, seed)
	}
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	bottom := lerp(lerp(at(0, 0, 0), at(1, 0, 0), sx),
		lerp(at(0, 1, 0), at(1, 1, 0), sx), sy)
	top := lerp(lerp(at(0, 0, 1), at(1, 0, 1), sx),
		lerp(at(0, 1, 1), at(1, 1, 1), sx), sy)
	return lerp(bottom, top, sz)
}

// beam marches one beam from its origin until it meets the water or its
// reach, splatting its glow so a pixel gets the same whatever the step.
func beam(sky, front *picture, open []bool, rim func(int) int, cam camera,
	origin, dir vec, colour [3]float64, moon [3]float64, seconds,
	shoreRow float64) {
	s := 0.0
	lum := moonLum(moon)
	for s < beamReach {
		p := origin.add(dir.scale(s))
		if p.z < waterZ {
			// where it meets the water, a bright point: the surface and
			// the mist on it lit from inside
			if x, y, ahead := cam.project(p); ahead {
				splat(sky, front, x, y, 1.2, lum*60*fog(p, seconds), colour,
					shoreRow)
			}
			return
		}
		x, y, ahead := cam.project(p)
		dist := math.Sqrt(p.dot(p))
		// a step of about half a pixel
		step := math.Max(0.05, 0.5*dist/cam.focal)
		s += step
		if !ahead || x < -4 || y < -4 || x >= cam.width+4 || y >= cam.height+4 {
			continue
		}
		xi, yi := int(x), int(y)
		if xi >= 0 && yi >= 0 && xi < sky.width && yi < sky.height &&
			!open[yi*sky.width+xi] && yi < rim(xi) && dist > 80 {
			// behind the near trees
			continue
		}
		if closeTrees != nil && xi >= 0 && yi >= 0 && xi < sky.width &&
			yi < sky.height && closeTrees[yi*sky.width+xi] {
			// behind the close spruces
			continue
		}
		// clear air shows a beam faintly; mist shows it hard
		// a beam is a pixel wide at any size, so on a frame narrower
		// than 1280 it is dimmed as a downscale would average it
		bright := lum * (0.12 + 24*fog(p, seconds)) * 2.2 *
			math.Min(1, float64(sky.width)/1280)
		pixels := step / dist * cam.focal
		// a hard core about a pixel wide, and a thin halo: a laser, not
		// a light sabre
		splat(sky, front, x, y, 0.5, bright*pixels*0.88, colour, shoreRow)
		splat(sky, front, x, y, 1.6, bright*pixels*0.12, colour, shoreRow)
	}
}

// splat lays a gaussian of a width and a total amount at a point, into
// the sky above the shore and the front below it.
func splat(sky, front *picture, x, y, sigma, amount float64,
	colour [3]float64, shoreRow float64) {
	r := int(math.Ceil(3 * sigma))
	norm := amount / (2 * math.Pi * sigma * sigma)
	for yy := int(y) - r; yy <= int(y)+r; yy++ {
		for xx := int(x) - r; xx <= int(x)+r; xx++ {
			if xx < 0 || yy < 0 || xx >= sky.width || yy >= sky.height {
				continue
			}
			dx, dy := float64(xx)+0.5-x, float64(yy)+0.5-y
			w := norm * math.Exp(-(dx*dx+dy*dy)/(2*sigma*sigma))
			target := sky
			if float64(yy) >= shoreRow {
				target = front
			}
			i := target.at(xx, yy)
			for c := range 3 {
				target.light[i+c] += w * colour[c]
			}
		}
	}
}

// clampInt is v held between nought and one less than n.
func clampInt(v, n int) int { return max(0, min(n-1, v)) }

// moonLum is the moon disc's light as one number, its channels' mean.
func moonLum(moon [3]float64) float64 { return (moon[0] + moon[1] + moon[2]) / 3 }

// brightest is the moon disc's light: the mean of the pixels at least
// half as bright as the brightest.
func brightest(p *picture) [3]float64 {
	most := 0.0
	for i := 0; i < len(p.light); i += 3 {
		most = math.Max(most, p.light[i]+p.light[i+1]+p.light[i+2])
	}
	var sum [3]float64
	count := 0.0
	for i := 0; i < len(p.light); i += 3 {
		if p.light[i]+p.light[i+1]+p.light[i+2] < most/2 {
			continue
		}
		for c := range 3 {
			sum[c] += p.light[i+c]
		}
		count++
	}
	for c := range sum {
		sum[c] /= count
	}
	return sum
}

// curled is fbm warped by two more fbm fields drifting with time, so it
// curls as mist does.
func curled(u, v, seconds float64, seed uint64) float64 {
	qx := fbm(u+0.09*seconds, v, seed)
	qy := fbm(u+5.2-0.07*seconds, v+1.3, seed+1)
	return fbm(u+1.9*qx, v+0.8*qy, seed+2)
}

// fbm is five octaves of value noise, each twice as fine and half as
// strong, nought to one.
func fbm(u, v float64, seed uint64) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for octave := range 5 {
		sum += amp * noise(u, v, seed+uint64(octave)*7919)
		norm += amp
		u, v, amp = u*2.03, v*2.03, amp*0.5
	}
	return sum / norm
}

// noise is value noise: the lattice's numbers at the four corners round
// a point, eased between.
func noise(u, v float64, seed uint64) float64 {
	x0, y0 := math.Floor(u), math.Floor(v)
	fx, fy := u-x0, v-y0
	sx, sy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	ix, iy := int64(x0), int64(y0)
	a := lattice(ix, iy, seed)
	b := lattice(ix+1, iy, seed)
	c := lattice(ix, iy+1, seed)
	d := lattice(ix+1, iy+1, seed)
	return a + (b-a)*sx + (c-a)*sy + (a-b-c+d)*sx*sy
}

// lattice is a number from nought to one for a lattice point and a seed,
// the same every time.
func lattice(x, y int64, seed uint64) float64 { return rig.Lattice(x, y, seed) }

// smooth is nought below a, one above b, and eased between.
func smooth(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}

// read is a colour pfm as a picture, rows from the top.
func read(path string) (*picture, float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	in := bufio.NewReader(file)
	var kind string
	var width, height int
	var scale float64
	if _, err := fmt.Fscan(in, &kind, &width, &height, &scale); err != nil {
		return nil, 0, err
	}
	in.ReadByte()
	if kind != "PF" {
		return nil, 0, fmt.Errorf("%s is not a colour pfm", path)
	}
	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}
	raw := make([]float32, width*height*3)
	if err := binary.Read(in, order, raw); err != nil {
		return nil, 0, err
	}
	p := &picture{width, height, make([]float64, len(raw))}
	for y := range height {
		from := (height - 1 - y) * width * 3
		for i := range width * 3 {
			p.light[y*width*3+i] = float64(raw[from+i])
		}
	}
	return p, scale, nil
}

// write keeps a picture as a pfm, rows bottom up.
func write(path string, p *picture, scale float64) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	out := bufio.NewWriter(file)
	fmt.Fprintf(out, "PF\n%d %d\n%g\n", p.width, p.height, scale)
	raw := make([]float32, len(p.light))
	for y := range p.height {
		to := (p.height - 1 - y) * p.width * 3
		for i := range p.width * 3 {
			raw[to+i] = float32(p.light[y*p.width*3+i])
		}
	}
	if err := binary.Write(out, order(scale), raw); err != nil {
		return err
	}
	return out.Flush()
}

// order is a pfm's byte order by its scale's sign: above nought, big
// endian.
func order(scale float64) binary.ByteOrder {
	if scale > 0 {
		return binary.BigEndian
	}
	return binary.LittleEndian
}
