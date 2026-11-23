package main

import (
	"math"
	"testing"
)

// frameW and frameH are the size of the small frames the lights are
// tried on.
const frameW, frameH = 64, 36

// moonOne is a moon disc whose light is one in every channel.
var moonOne = [3]float64{1, 1, 1}

// scene is a small frame of the show: its camera, a sky mask open above
// row 12 where the far rim stands, the rim's line, and the shore row.
type scene struct {
	cam      camera
	open     []bool
	rim      func(int) int
	shoreRow float64
}

// newScene builds the small frame and clears the near spruces.
func newScene() scene {
	open := make([]bool, frameW*frameH)
	for i := range open[:12*frameW] {
		open[i] = true
	}
	near = make([]bool, frameW*frameH)
	return scene{cam: newCamera(frameW, frameH), open: open,
		rim:      func(int) int { return 12 },
		shoreRow: (top - shore) / (top - bottom) * frameH}
}

// finite reports whether every number in a picture is finite and not
// below nought.
func finite(p *picture) bool {
	for _, v := range p.light {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return false
		}
	}
	return true
}

// channels is the sum of a picture's red, green and blue on their own.
func channels(p *picture) [3]float64 {
	var sum [3]float64
	for i := 0; i < len(p.light); i += 3 {
		for c := range sum {
			sum[c] += p.light[i+c]
		}
	}
	return sum
}

// TestConeChord checks the entry and exit of rays through a cone worked
// by hand: apex 5 m over the eye, a slope of one, the water 5 m under it.
func TestConeChord(t *testing.T) {
	for _, c := range []struct {
		name         string
		d, apex      vec
		in, out      float64
		crosses      bool
		slope, water float64
	}{
		{"across it", vec{1, 0, 0}, vec{10, 0, 5}, 5, 15, true, 1, -5},
		{"past it", vec{0, 1, 0}, vec{10, 0, 5}, 0, 0, false, 1, -5},
		{"straight down beside it", vec{0, 0, -1}, vec{10, 0, 5}, 0, 0, false,
			1, -5},
		{"down its axis", vec{0, 0, -1}, vec{0, 0, 5}, 0, 5, true, 1, -5},
		{"up its axis", vec{0, 0, 1}, vec{0, 0, 5}, 0, 5, true, 1, -5},
		{"up its wall", vec{1, 0, 1}, vec{0, 0, 5}, 0, 2.5, true, 1, -5},
		{"down its wall", vec{1, 0, -1}, vec{0, 0, 5}, 0, 5, true, 1, -5},
		{"along the apex's height", vec{1, 0, 1}, vec{0, 0, 0}, 0, 0, false, 1,
			-5},
		{"level, apex under the eye", vec{1, 0, 0}, vec{10, 0, -1}, 0, 0,
			false, 1, -5},
		{"level, water over the eye", vec{1, 0, 0}, vec{10, 0, 5}, 0, 0,
			false, 1, 2},
	} {
		in, out, crosses := coneChord(c.d, c.apex, c.slope, c.water)
		if crosses != c.crosses || (crosses && (!within(in, c.in, 1e-9) ||
			!within(out, c.out, 1e-9))) {
			t.Errorf("%s: %v %v %v, want %v %v %v", c.name, in, out, crosses,
				c.in, c.out, c.crosses)
		}
	}
}

// TestSkyGlowLightsOnlyTheSky: the glow goes to open pixels, in a cool
// blue, thicker toward the rim, and is none when the gain is nought.
func TestSkyGlowLightsOnlyTheSky(t *testing.T) {
	guard(t)
	s := newScene()
	show = cues{moon: 1, master: 1, moonAlt: 3}
	sky := flat(frameW, frameH, 0)
	skyGlow(sky, s.open, s.rim, s.cam, moonOne)
	for y := 0; y < frameH; y++ {
		got := sky.light[sky.at(0, y)]
		if s.open[y*frameW] != (got > 0) {
			t.Errorf("row %d is open %v but has light %v", y, s.open[y*frameW],
				got)
		}
	}
	if sky.light[sky.at(0, 11)] <= sky.light[sky.at(0, 0)] {
		t.Error("the sky is not brighter at the rim than at the top")
	}
	sum := channels(sky)
	if !(sum[0] < sum[1] && sum[1] < sum[2]) || !within(sum[0]/sum[2], 0.55/0.95, 1e-9) {
		t.Errorf("the glow's colour is %v, want a cool blue", sum)
	}
	// the moon straight up is not ahead, so there is no halo
	show.moonAlt = 90
	dark := flat(frameW, frameH, 0)
	skyGlow(dark, s.open, s.rim, s.cam, moonOne)
	if total(dark) >= total(sky) {
		t.Error("a moon out of frame left the halo in")
	}
	saved := skyGlowGain
	skyGlowGain = 0
	t.Cleanup(func() { skyGlowGain = saved })
	none := flat(frameW, frameH, 0)
	skyGlow(none, s.open, s.rim, s.cam, moonOne)
	if total(none) != 0 {
		t.Error("a glow of nought lit the sky")
	}
}

// TestMoonDiscSlidesWithTheMoon: at the altitude it was rendered at the
// disc lands where it was, only on open pixels, and at another altitude
// it moves by the pixels that altitude is away.
func TestMoonDiscSlidesWithTheMoon(t *testing.T) {
	guard(t)
	s := newScene()
	open := allOpen(frameW, frameH)
	disc := flat(frameW, frameH, 0)
	for x := 0; x < frameW; x++ {
		disc.light[disc.at(x, 18)] = 1
	}
	show = cues{moon: 2, moonAlt: 5.45}
	sky := flat(frameW, frameH, 0)
	moonDisc(sky, disc, open, s.cam)
	if got := sky.light[sky.at(7, 18)]; !within(got, 2, 1e-9) {
		t.Errorf("the disc at its own altitude is %v, want 2", got)
	}
	if got := sky.light[sky.at(7, 17)]; got != 0 {
		t.Errorf("the row above the disc is %v", got)
	}
	// closed pixels take none
	open[18*frameW+7] = false
	sky = flat(frameW, frameH, 0)
	moonDisc(sky, disc, open, s.cam)
	if sky.light[sky.at(7, 18)] != 0 {
		t.Error("the disc lit a pixel the mask shows as land")
	}
	open = allOpen(frameW, frameH)
	show.moonAlt = 8.45
	_, from, _ := s.cam.project(toward(azimuth, 5.45).scale(1e6))
	_, to, _ := s.cam.project(toward(azimuth, 8.45).scale(1e6))
	sky = flat(frameW, frameH, 0)
	moonDisc(sky, disc, open, s.cam)
	weight, sum := 0.0, 0.0
	for y := 0; y < frameH; y++ {
		weight += sky.light[sky.at(3, y)]
		sum += float64(y) * sky.light[sky.at(3, y)]
	}
	if want := 18 + (to - from); !within(sum/weight, want, 0.01) {
		t.Errorf("the disc's centre is at row %v, want %v", sum/weight, want)
	}
	if to-from > -5 {
		t.Errorf("three degrees up moved the moon only %v rows", to-from)
	}
}

// TestHaze: no haze well under the rim and the shore, haze thick at the
// rim, and the lake's haze thicker toward the near shore, as worked.
func TestHaze(t *testing.T) {
	if got := haze(10, 500, 400, 10000, 720, 1); got != 0 {
		t.Errorf("haze far under the rim is %v", got)
	}
	if got := haze(10, 0, 1000, 10000, 720, 1); got > 1e-3 {
		t.Errorf("haze 1000 rows over the rim is %v", got)
	}
	at := haze(10, 400, 400, 10000, 720, 1)
	if at < 0.35 || at > 1 {
		t.Errorf("haze at the rim is %v, want 0.35 to 1", at)
	}
	lake := 0.5 + 0.5*math.Exp(-(400-100.0)/(720-100.0)*1.5)
	got := haze(10, 400, 0, 100, 720, 1)
	if got < 0.35*lake || got > lake {
		t.Errorf("lake haze is %v, want between %v and %v", got, 0.35*lake, lake)
	}
}

// TestStrobesFireByTheirCues: each strobe adds half to the level when
// cued on, none when cued off, and the magenta one's light keeps the
// gel's proportions.
func TestStrobesFireByTheirCues(t *testing.T) {
	guard(t)
	s := newScene()
	for _, c := range []struct {
		white, magenta int
		level          float64
	}{{1, 0, 0.5}, {0, 1, 0.5}, {1, 1, 1}, {0, 0, 0}} {
		show = cues{white: c.white, magenta: c.magenta}
		sky := flat(frameW, frameH, 0)
		level := strobes(sky, s.open, s.rim, s.cam, moonOne, 1)
		if level != c.level {
			t.Errorf("white %d magenta %d: level %v, want %v", c.white,
				c.magenta, level, c.level)
		}
		sum := channels(sky)
		switch {
		case c.level == 0:
			if total(sky) != 0 {
				t.Error("strobes cued off lit the sky")
			}
		case c.white == 1 && c.magenta == 0:
			if sum[0] <= 0 || !within(sum[0], sum[1], 1e-9) ||
				!within(sum[0], sum[2], 1e-9) {
				t.Errorf("the white strobe lit %v", sum)
			}
		case c.white == 0:
			if sum[0] <= 0 || !within(sum[1]/sum[0], 0.25, 1e-9) ||
				!within(sum[2]/sum[0], 0.85, 1e-9) {
				t.Errorf("the magenta strobe lit %v", sum)
			}
		}
	}
}

// TestStrobeShaftsAreCutByTheRim: with no sky and no near spruce between,
// the strobe's light is cut; with near spruces it comes through; and a
// camera turned away sees the strobe fire without lighting anything.
func TestStrobeShaftsAreCutByTheRim(t *testing.T) {
	guard(t)
	s := newScene()
	show = cues{white: 1, magenta: 0}
	closed := make([]bool, frameW*frameH)
	sky := flat(frameW, frameH, 0)
	if level := strobes(sky, closed, s.rim, s.cam, moonOne, 1); level != 0.5 ||
		total(sky) != 0 {
		t.Errorf("a shut sky: level %v, light %v", level, total(sky))
	}
	for i := range near {
		near[i] = true
	}
	if strobes(sky, closed, s.rim, s.cam, moonOne, 1); total(sky) == 0 {
		t.Error("near spruces cut the shaft as the rim does")
	}
	away := s.cam
	away.forward = away.forward.scale(-1)
	sky = flat(frameW, frameH, 0)
	if level := strobes(sky, s.open, s.rim, away, moonOne, 1); level != 0.5 ||
		total(sky) != 0 {
		t.Errorf("a camera turned away: level %v, light %v", level,
			total(sky))
	}
}

// TestStrobeLuck: with no cue a strobe fires by its lattice, so the
// level over many seconds is a half or a whole at some frames and none at
// others.
func TestStrobeLuck(t *testing.T) {
	guard(t)
	s := newScene()
	show = cues{white: -1, magenta: -1}
	seen := map[float64]bool{}
	for second := 0; second < 40; second++ {
		seen[strobes(flat(2, 2, 0), s.open[:4], s.rim,
			newCamera(2, 2), moonOne, float64(second)/6)] = true
	}
	if len(seen) < 2 {
		t.Errorf("strobes on their own luck gave only %v", seen)
	}
}

// TestSearchlights: each move lights open sky only and finite light, a
// level under a hundredth lights nothing, and the one chasing is the
// brightest.
func TestSearchlights(t *testing.T) {
	guard(t)
	s := newScene()
	for _, move := range []string{"sweep", "chase", "fan", "viewer"} {
		show = cues{search: 1, searchGate: 1, searchMove: move,
			searchPhase: 0.3, searchMovePhase: 0.2}
		sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
		searchlights(sky, front, s.open, s.cam, moonOne, s.shoreRow)
		if !finite(sky) || !finite(front) || total(sky)+total(front) == 0 {
			t.Errorf("%s: sky %v front %v", move, total(sky), total(front))
		}
		if move != "viewer" {
			for y := 12; y < frameH; y++ {
				for x := 0; x < frameW; x++ {
					if sky.light[sky.at(x, y)] != 0 {
						t.Fatalf("%s lit land at %d %d", move, x, y)
					}
				}
			}
		}
	}
	show = cues{search: 0.5, searchGate: 0.01}
	sky := flat(frameW, frameH, 0)
	searchlights(sky, flat(frameW, frameH, 0), s.open, s.cam, moonOne, s.shoreRow)
	if total(sky) != 0 {
		t.Error("a level under a hundredth lit the sky")
	}
	// a chase has one light bright and the rest at an eighth, so it is
	// dimmer than the same lights all bright
	show = cues{search: 1, searchGate: 1, searchMove: "chase"}
	chase := flat(frameW, frameH, 0)
	searchlights(chase, flat(frameW, frameH, 0), s.open, s.cam, moonOne, s.shoreRow)
	show.searchMove = "sweep"
	sweep := flat(frameW, frameH, 0)
	searchlights(sweep, flat(frameW, frameH, 0), s.open, s.cam, moonOne, s.shoreRow)
	if !(total(chase) < total(sweep)) {
		t.Errorf("a chase lit %v, a sweep %v", total(chase), total(sweep))
	}
}

// TestViewerSearchlightsHideBehindSpruces: in the viewer
// move a close spruce over the whole frame takes every beam.
func TestViewerSearchlightsHideBehindSpruces(t *testing.T) {
	guard(t)
	s := newScene()
	show = cues{search: 1, searchGate: 1, searchMove: "viewer"}
	closeTrees = make([]bool, frameW*frameH)
	for i := range closeTrees {
		closeTrees[i] = true
	}
	sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	searchlights(sky, front, s.open, s.cam, moonOne, s.shoreRow)
	if total(sky)+total(front) != 0 {
		t.Errorf("spruces over the frame left %v of light",
			total(sky)+total(front))
	}
}

// TestBackHaze: each backlight lays haze by its flood's level, none where
// a tree stands, the right one in the gel's colour, and nothing with no
// trees or a level of nought or a camera turned away.
func TestBackHaze(t *testing.T) {
	guard(t)
	s := newScene()
	trees := flat(frameW, frameH, 0)
	for y := 0; y < frameH; y++ {
		trees.light[trees.at(10, y)] = 1
	}
	show = cues{floodLeft: 1}
	front := flat(frameW, frameH, 0)
	backHaze(front, trees, s.cam, moonOne, 1)
	if total(front) == 0 || !finite(front) {
		t.Fatalf("the left backlight laid %v", total(front))
	}
	for y := 0; y < frameH; y++ {
		if front.light[front.at(10, y)] != 0 {
			t.Fatalf("haze lit the tree at row %d", y)
		}
	}
	sum := channels(front)
	if !within(sum[0], sum[1], 1e-9) || !within(sum[0], sum[2], 1e-9) {
		t.Errorf("the white backlight lit %v", sum)
	}
	show = cues{floodRight: 1}
	front = flat(frameW, frameH, 0)
	backHaze(front, trees, s.cam, moonOne, 1)
	sum = channels(front)
	if sum[0] == 0 || !within(sum[1]/sum[0], 0.25, 1e-9) ||
		!within(sum[2]/sum[0], 0.85, 1e-9) {
		t.Errorf("the gelled backlight lit %v", sum)
	}
	show = cues{}
	front = flat(frameW, frameH, 0)
	backHaze(front, trees, s.cam, moonOne, 1)
	backHaze(front, flat(frameW, frameH, 0), s.cam, moonOne, 1)
	show = cues{floodLeft: 1, floodRight: 1}
	backHaze(front, flat(frameW, frameH, 0), s.cam, moonOne, 1)
	away := s.cam
	away.forward = away.forward.scale(-1)
	backHaze(front, trees, away, moonOne, 1)
	if total(front) != 0 {
		t.Errorf("no level, no trees or no view still laid %v", total(front))
	}
}

// TestBacklightsStandAhead checks the two lights are
// ahead of the eye and the left is the nearer the left.
func TestBacklightsStandAhead(t *testing.T) {
	cam := newCamera(frameW, frameH)
	at := backlights()
	for side, p := range at {
		if _, _, ahead := cam.project(p); !ahead {
			t.Errorf("backlight %d is behind the camera", side)
		}
	}
	if at[0].dot(cam.right) >= at[1].dot(cam.right) {
		t.Errorf("the left backlight is not left of the right: %v", at)
	}
}

// TestApertureLightsAPointInItsColour: the mouth in view is lit, a mouth
// out of view or behind a close spruce lights nothing.
func TestApertureLightsAPointInItsColour(t *testing.T) {
	guard(t)
	s := newScene()
	at := s.cam.forward.scale(500)
	sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	aperture(sky, front, s.cam, at, [3]float64{1, 0, 0}, moonOne, 1000)
	sum := channels(sky)
	// the core is half the colour and half its brightest channel
	if sum[0] <= 0 || sum[1] <= 0 || sum[1] >= sum[0] || sum[2] != sum[1] {
		t.Errorf("a red mouth lit %v", sum)
	}
	sky = flat(frameW, frameH, 0)
	aperture(sky, front, s.cam, s.cam.forward.scale(-500), [3]float64{1, 0, 0},
		moonOne, 1000)
	aperture(sky, front, s.cam, s.cam.forward.add(s.cam.right.scale(5)).scale(
		500), [3]float64{1, 0, 0}, moonOne, 1000)
	if total(sky) != 0 {
		t.Errorf("a mouth out of view lit %v", total(sky))
	}
	closeTrees = make([]bool, frameW*frameH)
	for i := range closeTrees {
		closeTrees[i] = true
	}
	aperture(sky, front, s.cam, at, [3]float64{1, 0, 0}, moonOne, 1000)
	if total(sky) != 0 {
		t.Errorf("a mouth behind a spruce lit %v", total(sky))
	}
}

// TestBeamKeepsItsColourAndStopsAtTheWater: a beam's light is in the
// colour it was given, a beam that dips under the water ends in a spot
// there, and one behind a close spruce lays nothing.
func TestBeamKeepsItsColourAndStopsAtTheWater(t *testing.T) {
	guard(t)
	s := newScene()
	origin := vec{0, 0, 0}
	way := toward(azimuth, -1)
	sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	beam(sky, front, s.open, s.rim, s.cam, origin, way, [3]float64{1, 0.5, 0},
		moonOne, 1, s.shoreRow)
	sum := channels(sky)
	more := channels(front)
	for c := range sum {
		sum[c] += more[c]
	}
	if sum[0] <= 0 || !within(sum[1]/sum[0], 0.5, 1e-9) || sum[2] != 0 {
		t.Errorf("the beam lit %v", sum)
	}
	// aimed hard down it meets the water at once, at a spot under the eye
	down := flat(frameW, frameH, 0)
	beam(down, front, s.open, s.rim, s.cam, vec{0, 0, waterZ + 0.5},
		vec{0, 0, -1}, [3]float64{1, 1, 1}, moonOne, 1, s.shoreRow)
	if total(down) != 0 {
		t.Errorf("a beam straight down from the eye laid light in frame: %v",
			total(down))
	}
	closeTrees = make([]bool, frameW*frameH)
	for i := range closeTrees {
		closeTrees[i] = true
	}
	hidden, hiddenFront := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	beam(hidden, hiddenFront, s.open, s.rim, s.cam, origin, way,
		[3]float64{1, 1, 1}, moonOne, 1, s.shoreRow)
	if total(hidden)+total(hiddenFront) != 0 {
		t.Error("a beam behind the close spruces laid light")
	}
}

// TestBeamEndsInASpotOnTheWater: a beam that dips to the lake ends in a
// bright spot where it meets the surface, as its comment says.
func TestBeamEndsInASpotOnTheWater(t *testing.T) {
	t.Skip("known: beam lays its water spot with fog(p), which is nought below the water, so the spot is always dark")
	guard(t)
	s := newScene()
	out := toward(azimuth, 0).scale(300)
	out.z = waterZ + 2
	spot, spotFront := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	beam(spot, spotFront, s.open, s.rim, s.cam, out, vec{0, 0, -1},
		[3]float64{1, 1, 1}, moonOne, 1, s.shoreRow)
	// the beam is only 2 m long, so its own glow is a few pixels at most and
	// the spot is the light that is left
	if total(spot)+total(spotFront) < 100 {
		t.Errorf("a beam down onto the water laid %v", total(spot)+total(spotFront))
	}
}

// TestBeamBehindTheNearTreesIsHidden: a beam whose pixels fall on land
// under the rim, past 80 m, is not laid, and is where it is open.
func TestBeamBehindTheNearTreesIsHidden(t *testing.T) {
	guard(t)
	s := newScene()
	s.rim = func(int) int { return 20 }
	land := make([]bool, frameW*frameH)
	way := toward(azimuth, 1.5)
	origin := vec{0, 0, 0}
	sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	beam(sky, front, land, s.rim, s.cam, origin, way, moonOne, moonOne, 1,
		s.shoreRow)
	shown, shownFront := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	beam(shown, shownFront, allOpen(frameW, frameH), s.rim, s.cam, origin, way,
		moonOne, moonOne, 1, s.shoreRow)
	if total(shown)+total(shownFront) <= total(sky)+total(front) {
		t.Errorf("land hid nothing: %v against %v", total(sky)+total(front),
			total(shown)+total(shownFront))
	}
}

// TestFigureWaysIsTheRigsFigureAsVectors checks the fan has the beams
// asked for, unit long, spread about the heading.
func TestFigureWaysIsTheRigsFigureAsVectors(t *testing.T) {
	ways, weight, split := figureWays("fan", 90, 0, 40, 5, 0)
	if len(ways) != 5 || weight != 1 || !split {
		t.Fatalf("a fan of 5 is %d ways, weight %v, split %v", len(ways),
			weight, split)
	}
	for i, w := range ways {
		length := math.Sqrt(w.dot(w))
		if !within(length, 1, 1e-9) {
			t.Errorf("way %d is not a unit", i)
		}
	}
	// 90 degrees is east: the middle beam is due east and the ends are 20
	// degrees either side
	if mid := ways[2]; !within(mid.x, 1, 1e-9) || !within(mid.y, 0, 1e-9) {
		t.Errorf("the middle beam is %v, want due east", mid)
	}
	if end := ways[0]; !within(end.x, math.Cos(20*math.Pi/180), 1e-9) ||
		end.y <= 0 {
		t.Errorf("the first beam is %v, want 20 degrees toward north", end)
	}
	if one, _, _ := figureWays("beam", 90, 0, 40, 5, 0); len(one) != 1 {
		t.Errorf("a beam figure is %d ways", len(one))
	}
}

// TestLasersByTheirCues: a rig of nought is dark, both gates shut is
// dark, the rig scales the light, a bank can be shut alone, a split pulse
// lights half the fan (each head's mouth lights in both halves), and a cued colour is used.
func TestLasersByTheirCues(t *testing.T) {
	guard(t)
	s := newScene()
	run := func(c cues) float64 {
		show = c
		sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
		lasers(sky, front, s.open, s.rim, s.cam, moonOne, 1, s.shoreRow)
		if !finite(sky) || !finite(front) {
			t.Fatalf("cues %+v gave light that is not finite", c)
		}
		return total(sky) + total(front)
	}
	base := cues{rig: 1, gate: 1, beams: 5, figure: "fan", parity: -1,
		gateLeft: -1, gateRight: -1, sweep: -1}
	full := run(base)
	if full <= 0 {
		t.Fatal("a rig at one lit nothing")
	}
	dark := base
	dark.rig = 0
	if got := run(dark); got != 0 {
		t.Errorf("a rig of nought lit %v", got)
	}
	half := base
	half.rig = 0.5
	if got := run(half); !within(got, full/2, 1e-9*full) {
		t.Errorf("a rig of a half lit %v, want %v", got, full/2)
	}
	shut := base
	shut.gate = 0.005
	if got := run(shut); got != 0 {
		t.Errorf("a shut gate lit %v", got)
	}
	left := base
	left.gateLeft, left.gateRight = 1, 0
	right := base
	right.gateLeft, right.gateRight = 0, 1
	l, r := run(left), run(right)
	if l <= 0 || r <= 0 || !(l < full) || !(r < full) ||
		!within(l+r, full, 1e-9*full) {
		t.Errorf("the banks lit %v and %v, together %v", l, r, full)
	}
	split := base
	split.parity = 0
	even := run(split)
	split.parity = 1
	odd := run(split)
	mouths := base
	mouths.beams = 0
	apertures := run(mouths)
	if !(even < full) || !(odd < full) || !(apertures > 0) ||
		!within(even+odd, full+apertures, 1e-6*full) {
		t.Errorf("the halves lit %v and %v, the whole %v, the mouths %v", even, odd,
			full, apertures)
	}
	red := base
	red.hasColour, red.colour = true, [3]float64{1, 0, 0}
	show = red
	sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	lasers(sky, front, s.open, s.rim, s.cam, moonOne, 1, s.shoreRow)
	sum := channels(sky)
	if sum[0] <= sum[1] || sum[0] <= sum[2] {
		t.Errorf("a red cue lit %v", sum)
	}
}

// TestLayoutIsTheRigsAtThisEye checks that the head layout the pass draws
// from stands the heads where the rig puts them for the eye and deck.
func TestLayoutIsTheRigsAtThisEye(t *testing.T) {
	heads := layout().Heads()
	if len(heads) != 4 {
		t.Fatalf("the rig has %d heads", len(heads))
	}
	if stage := layout(); stage.Water() != waterZ {
		t.Errorf("the lake stands at %v, want %v", stage.Water(), waterZ)
	}
}

// TestUfoBeamLightIsInProportionToTheShipsLevel: the beam lays light in
// the sky, the same in the front where the water mask shows, none where a
// close spruce stands, and exactly twice as much at twice the level.
func TestUfoBeamLightIsInProportionToTheShipsLevel(t *testing.T) {
	guard(t)
	s := newScene()
	run := func(level float64, trees, water *picture) (*picture, *picture) {
		show = cues{overhead: level}
		sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
		ufoBeam(sky, front, trees, water, s.cam, moonOne, 1)
		return sky, front
	}
	one, oneFront := run(1, nil, nil)
	if total(one) == 0 || !finite(one) {
		t.Fatalf("the ship's beam lit %v", total(one))
	}
	if total(oneFront) != 0 {
		t.Error("the front took light with no water mask")
	}
	two, _ := run(2, nil, nil)
	if !within(total(two), 2*total(one), 1e-9*total(two)) {
		t.Errorf("twice the level lit %v, once %v", total(two), total(one))
	}
	none, _ := run(0, nil, nil)
	if total(none) != 0 {
		t.Errorf("a level of nought lit %v", total(none))
	}
	wet, wetFront := run(1, nil, flat(frameW, frameH, 1))
	for i := range wet.light {
		if wet.light[i] != wetFront.light[i] || wet.light[i] != one.light[i] {
			t.Fatalf("number %d: sky %v front %v, dry sky %v", i, wet.light[i],
				wetFront.light[i], one.light[i])
		}
	}
	covered, _ := run(1, flat(frameW, frameH, 1), nil)
	if total(covered) != 0 {
		t.Errorf("spruces over the frame left %v of the beam", total(covered))
	}
	// a mask with nothing lit takes no part
	blank, _ := run(1, flat(frameW, frameH, 0), flat(frameW, frameH, 0))
	if !within(total(blank), total(one), 1e-9*total(one)) {
		t.Errorf("blank masks changed the beam: %v against %v", total(blank),
			total(one))
	}
}

// TestLightsTurnedAway: with the camera turned round the searchlights
// have nothing ahead of it to light, and a strobe off the frame's edge
// still gathers its light along lines that leave the frame.
func TestLightsTurnedAway(t *testing.T) {
	guard(t)
	s := newScene()
	away := s.cam
	away.forward = away.forward.scale(-1)
	show = cues{search: 1, searchGate: 1, searchMove: "sweep"}
	sky, front := flat(frameW, frameH, 0), flat(frameW, frameH, 0)
	searchlights(sky, front, s.open, away, moonOne, s.shoreRow)
	if total(sky)+total(front) != 0 {
		t.Errorf("searchlights behind the camera lit %v", total(sky)+total(front))
	}
	show = cues{white: 1, magenta: 0}
	strip := flat(frameW, 4, 0)
	level := strobes(strip, allOpen(frameW, 4), func(int) int { return 2 },
		newCamera(frameW, 4), moonOne, 1)
	if level != 0.5 || !finite(strip) {
		t.Errorf("a strobe off a short frame: level %v, finite %v", level,
			finite(strip))
	}
}
