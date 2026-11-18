package main

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// cueFile writes lines as a cue file and says where it is.
func cueFile(t *testing.T, lines string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cues")
	if err := os.WriteFile(path, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestReadCuesTakesEveryKey reads one line for each key and checks each
// lands in its field, with blank and unkeyed lines passed over.
func TestReadCuesTakesEveryKey(t *testing.T) {
	guard(t)
	show = cues{}
	path := cueFile(t, "rig=0.5\nsweep=0.25\nbeams=7\nwhite=1\nmagenta=0\n"+
		"gate=0.75\nparity=1\nflood_left=0.1\nflood_right=0.2\nspots=0.3\n"+
		"spot_aim=1.5\nsearch=0.4\nsearch_phase=0.6\nmaster=0.9\n"+
		"overhead=0.8\nsearch_move=chase\nsearch_move_phase=0.35\n"+
		"search_gate=0.45\nfigure=cone\nfigure_phase=0.55\ngate_left=0.65\n"+
		"gate_right=0.85\nmoon=0.95\nmoon_alt=12.5\nfire=0.7\nfire_lit=3\n"+
		"fire_since=0.5,,1.5\ncolour=0.1,0.2,0.3\n\nno equals sign here\n"+
		"unknown=5\n")
	if err := readCues(path); err != nil {
		t.Fatal(err)
	}
	want := cues{rig: 0.5, sweep: 0.25, beams: 7, white: 1, magenta: 0,
		gate: 0.75, parity: 1, floodLeft: 0.1, floodRight: 0.2, spots: 0.3,
		spotAim: 1.5, search: 0.4, searchPhase: 0.6, master: 0.9,
		overhead: 0.8, searchMove: "chase", searchMovePhase: 0.35,
		searchGate: 0.45, figure: "cone", figurePhase: 0.55, gateLeft: 0.65,
		gateRight: 0.85, moon: 0.95, moonAlt: 12.5, fire: 0.7, fireLit: 3,
		fireSince: []float64{0.5, 1.5}, colour: [3]float64{0.1, 0.2, 0.3},
		hasColour: true}
	got := show
	if len(got.fireSince) != 2 || got.fireSince[0] != 0.5 ||
		got.fireSince[1] != 1.5 {
		t.Errorf("fire_since read as %v", got.fireSince)
	}
	got.fireSince, want.fireSince = nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("read %+v\nwant %+v", got, want)
	}
}

// TestReadCuesRefusals: a missing file and each kind of bad value is an
// error that says what was wrong.
func TestReadCuesRefusals(t *testing.T) {
	guard(t)
	if err := readCues(filepath.Join(t.TempDir(), "none")); err == nil {
		t.Error("a missing cue file was read")
	}
	for _, c := range []struct{ line, mention string }{
		{"rig=lots", "invalid syntax"},
		{"beams=1.5", "invalid syntax"},
		{"colour=1,2", "colour"},
		{"colour=1,2,x", "invalid syntax"},
		{"fire_since=1,soon", "fire_since"},
	} {
		if err := readCues(cueFile(t, c.line)); err == nil ||
			!strings.Contains(err.Error(), c.mention) {
			t.Errorf("%q gave %v", c.line, err)
		}
	}
}

// TestCameraProjectsTheViewCentreToTheMiddle checks the camera's centre,
// its scale to the right, and a point behind it.
func TestCameraProjectsTheViewCentreToTheMiddle(t *testing.T) {
	cam := newCamera(5, 5)
	if want := 5 * focal / gaugeW; !within(cam.focal, want, 1e-9) {
		t.Errorf("focal is %v pixels, want %v", cam.focal, want)
	}
	ahead := cam.forward.scale(100)
	x, y, ok := cam.project(ahead)
	if !ok || !within(x, 2.5, 1e-9) || !within(y, 2.5, 1e-9) {
		t.Errorf("the view's centre is at %v %v, ahead %v", x, y, ok)
	}
	x, _, _ = cam.project(ahead.add(cam.right.scale(10)))
	if want := 2.5 + 0.1*cam.focal; !within(x, want, 1e-9) {
		t.Errorf("a point 10 m right of 100 m ahead is at x %v, want %v", x,
			want)
	}
	_, y, _ = cam.project(ahead.add(cam.up.scale(10)))
	if want := 2.5 - 0.1*cam.focal; !within(y, want, 1e-9) {
		t.Errorf("a point 10 m up is at y %v, want %v", y, want)
	}
	if _, _, ok := cam.project(cam.forward.scale(-100)); ok {
		t.Error("a point behind the camera was ahead")
	}
	if got := cam.ray(2, 2); !within(got.dot(cam.forward), 1, 1e-12) {
		t.Errorf("the middle pixel's ray is %v, not the way forward", got)
	}
	if got := cam.ray(4, 2); got.dot(cam.right) <= 0 {
		t.Errorf("a pixel to the right looks %v, not rightward", got)
	}
}

// TestSkyOfTakesPixelsAboveTwoPercentOfTheBrightest checks the cut.
func TestSkyOfTakesPixelsAboveTwoPercentOfTheBrightest(t *testing.T) {
	mask := &picture{3, 1, []float64{0, 0, 0, 1, 0, 0, 0.01, 0, 0}}
	got := skyOf(mask)
	if got[0] || !got[1] || got[2] {
		t.Errorf("sky is %v, want [false true false]", got)
	}
}

// TestRimOfIsTheLowestSkyNearby: a column's line is the lowest sky within
// 30 columns either side, and a column off the edge takes the edge's.
func TestRimOfIsTheLowestSkyNearby(t *testing.T) {
	width, height := 100, 10
	open := make([]bool, width*height)
	for y := 0; y <= 5; y++ {
		open[y*width] = true
	}
	rim := rimOf(open, width, height)
	for _, c := range []struct{ x, want int }{
		{0, 6}, {30, 6}, {31, 0}, {99, 0}, {-4, 6}, {500, 0},
	} {
		if got := rim(c.x); got != c.want {
			t.Errorf("rim(%d) is %d, want %d", c.x, got, c.want)
		}
	}
}

// TestNearOfFloodsFromTheTopRows: a spruce reaching the top is found down
// to two rows above the rim; a dark pixel alone below the top is not.
func TestNearOfFloodsFromTheTopRows(t *testing.T) {
	width, height := 5, 50
	open := allOpen(width, height)
	for y := 0; y <= 44; y++ {
		open[y*width+2] = false
	}
	open[45*width+4] = false
	rim := func(int) int { return 45 }
	found := nearOf(open, rim, width, height)
	for _, c := range []struct {
		x, y int
		want bool
	}{
		{2, 0, true}, {2, 39, true}, {2, 42, true}, {2, 43, false},
		{2, 44, false}, {4, 45, false}, {1, 10, false},
	} {
		if got := found[c.y*width+c.x]; got != c.want {
			t.Errorf("near at %d %d is %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

// TestCoveredByTakesWhatIsOverThirtyPercent checks the mask's cut and a
// dark mask.
func TestCoveredByTakesWhatIsOverThirtyPercent(t *testing.T) {
	got := coveredBy(&picture{3, 1, []float64{0, 0, 0, 1, 0, 0, 0.2, 0, 0}})
	if got[0] || !got[1] || got[2] {
		t.Errorf("covered is %v", got)
	}
	for i, v := range coveredBy(flat(2, 2, 0)) {
		if v {
			t.Errorf("a dark mask covers pixel %d", i)
		}
	}
}

// TestSplatSpreadsItsAmountAndChoosesALayer: the light laid is the amount
// asked, in the colour asked, in the sky above the shore row and in the
// front from it down.
func TestSplatSpreadsItsAmountAndChoosesALayer(t *testing.T) {
	sky, front := flat(40, 40, 0), flat(40, 40, 0)
	splat(sky, front, 20, 10, 1.5, 100, [3]float64{1, 0.5, 0}, 25)
	var red, green, blue float64
	for i := 0; i < len(sky.light); i += 3 {
		red += sky.light[i]
		green += sky.light[i+1]
		blue += sky.light[i+2]
	}
	if !within(red, 100, 1) || !within(green, 50, 0.5) || blue != 0 {
		t.Errorf("laid %v %v %v, want about 100 50 0", red, green, blue)
	}
	if total(front) != 0 {
		t.Error("light above the shore row went in the front")
	}
	splat(sky, front, 20, 30, 1.5, 100, [3]float64{1, 1, 1}, 25)
	if !within(total(front), 300, 3) {
		t.Errorf("the front has %v, want about 300", total(front))
	}
	// off the picture, it is clipped and does not fail
	before := total(sky)
	splat(sky, front, -50, -50, 1, 100, [3]float64{1, 1, 1}, 25)
	if total(sky) != before {
		t.Error("a splat off the frame laid light")
	}
}

// TestSplatSkyKeepsToOpenPixels lays a gaussian that falls half on closed
// pixels and half on open ones.
func TestSplatSkyKeepsToOpenPixels(t *testing.T) {
	sky := flat(20, 20, 0)
	open := make([]bool, 400)
	for y := 0; y < 20; y++ {
		for x := 10; x < 20; x++ {
			open[y*20+x] = true
		}
	}
	splatSky(sky, open, 10, 10, 1, 60, [3]float64{1, 1, 1})
	closed, opened := 0.0, 0.0
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			v := sky.light[sky.at(x, y)]
			if x < 10 {
				closed += v
			} else {
				opened += v
			}
		}
	}
	if closed != 0 || !within(opened, 30, 1) {
		t.Errorf("closed %v, open %v, want 0 and about 30", closed, opened)
	}
	before := total(sky)
	splatSky(sky, open, -100, 10, 1, 60, [3]float64{1, 1, 1})
	splatSky(sky, open, 10, 500, 1, 60, [3]float64{1, 1, 1})
	if total(sky) != before {
		t.Error("a gaussian far off the frame laid light")
	}
}

// pass is a frame directory's file: a size and a constant.
func pass(t *testing.T, dir, name string, width, height int, value float64) {
	t.Helper()
	keep(t, dir, name, flat(width, height, value), 1)
}

// TestFixturesAddFloodsAndSpots: the left flood at its gain, and the
// spots blended between their neighbouring aims, tinted by the colour.
func TestFixturesAddFloodsAndSpots(t *testing.T) {
	guard(t)
	dir := t.TempDir()
	pass(t, dir, "flood-left.pfm", 2, 1, 0.001)
	sky := flat(2, 1, 0)
	show = cues{floodLeft: 1}
	if err := fixtures(sky, dir); err != nil {
		t.Fatal(err)
	}
	for i, v := range sky.light {
		if !within(v, 0.4, 1e-6) {
			t.Errorf("flood light %d is %v, want 0.4", i, v)
		}
	}
	// spots half way between aims 0 and 1, tinted toward red
	for name, v := range map[string]float64{"spot-left-0": 0.01,
		"spot-left-1": 0.02, "spot-right-2": 0.04, "spot-right-1": 0.08} {
		pass(t, dir, name+".pfm", 2, 1, v)
	}
	sky = flat(2, 1, 0)
	show = cues{spots: 1, spotAim: 0.5, hasColour: true,
		colour: [3]float64{1, 0, 0}}
	if err := fixtures(sky, dir); err != nil {
		t.Fatal(err)
	}
	if !within(sky.light[0], 11.25, 1e-4) || !within(sky.light[1], 5.625, 1e-4) ||
		!within(sky.light[2], 5.625, 1e-4) {
		t.Errorf("spot light is %v, want 11.25 5.625 5.625", sky.light[:3])
	}
	// aim at the far end: only the last aim of each side is wanted, and the
	// passes for aims that have no share need not exist
	dir = t.TempDir()
	pass(t, dir, "spot-left-2.pfm", 2, 1, 1)
	pass(t, dir, "spot-right-0.pfm", 2, 1, 1)
	sky = flat(2, 1, 0)
	show = cues{spots: 1, spotAim: 9}
	if err := fixtures(sky, dir); err != nil {
		t.Fatal(err)
	}
	if !within(sky.light[0], 300, 1e-6) {
		t.Errorf("light at the far aim is %v, want 300", sky.light[0])
	}
}

// TestFixturesRefusals: a wrong count of flood passes, a missing pass and
// a pass of another size are each an error.
func TestFixturesRefusals(t *testing.T) {
	guard(t)
	dir := t.TempDir()
	show = cues{floodLeft: 1}
	if err := fixtures(flat(2, 1, 0), dir); err == nil {
		t.Error("a missing pass was taken")
	}
	pass(t, dir, "flood-left.pfm", 3, 3, 1)
	if err := fixtures(flat(2, 1, 0), dir); err == nil ||
		!strings.Contains(err.Error(), "3x3") {
		t.Errorf("a pass of another size gave %v", err)
	}
	t.Setenv("FLOOD_PASSES", "only-one")
	if err := fixtures(flat(2, 1, 0), dir); err == nil ||
		!strings.Contains(err.Error(), "1 passes") {
		t.Errorf("one flood pass gave %v", err)
	}
}

// TestOverheadLightLaysTheShipsColour: the pass is added at the cue's
// level in the overhead's colour, and refused at another size.
func TestOverheadLightLaysTheShipsColour(t *testing.T) {
	guard(t)
	dir := t.TempDir()
	show = cues{overhead: 0.5}
	if err := overheadLight(flat(1, 1, 0), dir); err == nil {
		t.Error("a missing overhead pass was taken")
	}
	pass(t, dir, "overhead.pfm", 1, 1, 2)
	sky := flat(1, 1, 0)
	if err := overheadLight(sky, dir); err != nil {
		t.Fatal(err)
	}
	for c, want := range []float64{0.62, 0.8, 1.0} {
		if !within(sky.light[c], want, 1e-6) {
			t.Errorf("channel %d is %v, want %v", c, sky.light[c], want)
		}
	}
	if err := overheadLight(flat(2, 2, 0), dir); err == nil {
		t.Error("an overhead pass of another size was taken")
	}
}

// frames writes n frames named 000.pfm on into dir, frame i holding the
// value i+base, and a file that is not a frame.
func frames(t *testing.T, dir string, n int, base float64, width, height int) {
	t.Helper()
	for i := 0; i < n; i++ {
		pass(t, dir, frameName(i), width, height, base+float64(i))
	}
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)
}

// frameName is the name of frame i.
func frameName(i int) string {
	return string([]byte{'0', '0' + byte(i/10), '0' + byte(i%10)}) + ".pfm"
}

// TestFramesIn counts the pfm files and refuses a missing directory.
func TestFramesIn(t *testing.T) {
	dir := t.TempDir()
	frames(t, dir, 3, 0, 1, 1)
	if n, err := framesIn(dir); err != nil || n != 3 {
		t.Errorf("framesIn is %d, %v, want 3", n, err)
	}
	if _, err := framesIn(filepath.Join(dir, "none")); err == nil {
		t.Error("a missing directory was counted")
	}
}

// TestFireLightLaysBurstsStillShowing: a burst a frame in is laid at the
// fire's gain, one past the frames is burnt out, and a close spruce hides
// the fire where it stands.
func TestFireLightLaysBurstsStillShowing(t *testing.T) {
	guard(t)
	dir := t.TempDir()
	frames(t, dir, 2, 10, 2, 1)
	show = cues{fire: 1, fireSince: []float64{0, 0.5}}
	front := flat(2, 1, 0)
	if err := fireLight(front, dir); err != nil {
		t.Fatal(err)
	}
	// only the burst at 0 is within the two frames: 0.1 * 1 * frame 0 (10)
	for i, v := range front.light {
		if !within(v, 1, 1e-9) {
			t.Errorf("number %d is %v, want 1", i, v)
		}
	}
	// an older burst fades by e to the minus ago over the decay of a second
	show.fireSince = []float64{1.0 / 24}
	front = flat(2, 1, 0)
	if err := fireLight(front, dir); err != nil {
		t.Fatal(err)
	}
	if want := 0.1 * math.Exp(-1.0/24) * 11; !within(front.light[0], want, 1e-6) {
		t.Errorf("the second frame's burst is %v, want %v", front.light[0], want)
	}
	closeTrees = []bool{true, false}
	show.fireSince = []float64{0}
	front = flat(2, 1, 0)
	if err := fireLight(front, dir); err != nil {
		t.Fatal(err)
	}
	if front.light[0] != 0 || front.light[3] == 0 {
		t.Errorf("the spruce at pixel 0 did not hide the fire: %v", front.light)
	}
}

// TestFireLightRefusals: a missing directory, a missing frame and a frame
// of another size are errors.
func TestFireLightRefusals(t *testing.T) {
	guard(t)
	show = cues{fire: 1, fireSince: []float64{0}}
	if err := fireLight(flat(2, 1, 0), filepath.Join(t.TempDir(), "none")); err == nil {
		t.Error("a missing directory was read")
	}
	hole := t.TempDir()
	pass(t, hole, "001.pfm", 2, 1, 1)
	if err := fireLight(flat(2, 1, 0), hole); err == nil {
		t.Error("a missing frame 000 was taken")
	}
	wrong := t.TempDir()
	frames(t, wrong, 1, 0, 3, 3)
	if err := fireLight(flat(2, 1, 0), wrong); err == nil ||
		!strings.Contains(err.Error(), "3x3") {
		t.Errorf("a frame of another size gave %v", err)
	}
}

// TestStruckRisesOnTheBeat: one when the fire is not percussive or not
// lit, the flash at the ignition, and the pulse on a later downbeat.
func TestStruckRisesOnTheBeat(t *testing.T) {
	guard(t)
	fireSwellDir = ""
	show = cues{fireLit: 0, fireSince: []float64{0}}
	if got := struck(); got != 1 {
		t.Errorf("a fire with no swell is struck %v times", got)
	}
	fireSwellDir = "swell"
	show.fireLit = -1
	if got := struck(); got != 1 {
		t.Errorf("an unlit fire is struck %v times", got)
	}
	show.fireLit = 0
	show.fireSince = nil
	if got := struck(); got != 1 {
		t.Errorf("a fire with no bursts is struck %v times", got)
	}
	show.fireSince = []float64{0}
	if got := struck(); !within(got, 9, 1e-9) {
		t.Errorf("the ignition is struck %v times, want 9", got)
	}
	show.fireSince = []float64{0.12}
	if want := 1 + 3/math.E; !within(struck(), want, 1e-9) {
		t.Errorf("a later downbeat is struck %v times, want %v", struck(), want)
	}
}

// TestPercussiveFire: the ignition's swell is laid at nine times the gain
// with its burst at twice; a far beat settles to the steady fire's last
// frame; an unlit fire lays nothing.
func TestPercussiveFire(t *testing.T) {
	guard(t)
	steady, swell, burst := t.TempDir(), t.TempDir(), t.TempDir()
	frames(t, steady, 2, 100, 2, 1)
	frames(t, swell, 1, 1, 2, 1)
	frames(t, burst, 2, 5, 2, 1)
	fireSwellDir, fireBurstDir = swell, burst
	show = cues{fire: 1, fireLit: -1, fireSince: []float64{0}}
	front := flat(2, 1, 0)
	if err := percussiveFire(front, steady); err != nil || total(front) != 0 {
		t.Errorf("an unlit fire laid %v, %v", total(front), err)
	}
	show.fireLit, show.fireSince = 0, nil
	if err := percussiveFire(front, steady); err != nil || total(front) != 0 {
		t.Errorf("a fire with no bursts laid %v, %v", total(front), err)
	}
	// at ignition: swell frame 0 (1) at 0.9, burst frame 0 (5) at 0.2; the
	// burst at 0.25 s is past the two frames
	show.fireSince = []float64{0, 0.25}
	if err := percussiveFire(front, steady); err != nil {
		t.Fatal(err)
	}
	for i, v := range front.light {
		if !within(v, 0.9+1.0, 1e-9) {
			t.Errorf("number %d is %v, want 1.9", i, v)
		}
	}
	// ten seconds on the swell has gone, and the steady fire's last frame
	// (101) is laid at the gain 0.1 with the burst dropped
	fireBurstDir = ""
	show.fireSince = []float64{10}
	front = flat(2, 1, 0)
	if err := percussiveFire(front, steady); err != nil {
		t.Fatal(err)
	}
	if !within(front.light[0], 0.1*101+0.1*1*math.Exp(-10/0.3), 1e-6) {
		t.Errorf("the settled fire is %v, want 10.1", front.light[0])
	}
}

// TestPercussiveFireRefusals: a missing burst directory and a swell with
// no frames are errors.
func TestPercussiveFireRefusals(t *testing.T) {
	guard(t)
	steady, empty := t.TempDir(), t.TempDir()
	frames(t, steady, 1, 1, 2, 1)
	fireSwellDir = steady
	show = cues{fire: 1, fireLit: 0, fireSince: []float64{0}}
	fireBurstDir = filepath.Join(t.TempDir(), "none")
	if err := percussiveFire(flat(2, 1, 0), steady); err == nil {
		t.Error("a missing burst directory was read")
	}
	fireSwellDir, fireBurstDir = empty, ""
	if err := percussiveFire(flat(2, 1, 0), steady); err == nil ||
		!strings.Contains(err.Error(), "no frames") {
		t.Errorf("a swell with no frames gave %v", err)
	}
	fireSwellDir = filepath.Join(t.TempDir(), "none")
	if err := percussiveFire(flat(2, 1, 0), steady); err == nil {
		t.Error("a missing swell directory was read")
	}
}

// TestLayFrame: a frame past the last is the last, an amount of nought
// lays nothing, and a frame of another size is refused.
func TestLayFrame(t *testing.T) {
	guard(t)
	dir := t.TempDir()
	frames(t, dir, 2, 3, 1, 1)
	front := flat(1, 1, 0)
	if err := layFrame(front, dir, 99, 2); err != nil || front.light[0] != 8 {
		t.Errorf("the last frame at 2 laid %v, %v, want 8", front.light, err)
	}
	if err := layFrame(front, "none", 0, 0); err != nil || front.light[0] != 8 {
		t.Errorf("an amount of nought did something: %v, %v", front.light, err)
	}
	if err := layFrame(flat(2, 2, 0), dir, 0, 1); err == nil {
		t.Error("a frame of another size was laid")
	}
}

// TestFireFramesRefusals: a steady fire with no frames, a burst frame of
// another size, a frame file that is missing from a directory that has
// others, and a frame hidden by a close spruce, each as the pass does.
func TestFireFramesRefusals(t *testing.T) {
	guard(t)
	empty, steady, swell, burst := t.TempDir(), t.TempDir(), t.TempDir(),
		t.TempDir()
	frames(t, steady, 2, 1, 2, 1)
	frames(t, swell, 1, 1, 2, 1)
	frames(t, burst, 1, 1, 3, 3)
	fireSwellDir, fireBurstDir = swell, burst
	show = cues{fire: 1, fireLit: 0, fireSince: []float64{10}}
	if err := percussiveFire(flat(2, 1, 0), empty); err == nil ||
		!strings.Contains(err.Error(), "no frames") {
		t.Errorf("a steady fire with no frames gave %v", err)
	}
	show.fireSince = []float64{0}
	if err := percussiveFire(flat(2, 1, 0), steady); err == nil ||
		!strings.Contains(err.Error(), "3x3") {
		t.Errorf("a burst frame of another size gave %v", err)
	}
	hole := t.TempDir()
	pass(t, hole, "005.pfm", 2, 1, 1)
	if err := layFrame(flat(2, 1, 0), hole, 0, 1); err == nil {
		t.Error("a missing frame file was laid")
	}
	closeTrees = []bool{true, false}
	front := flat(2, 1, 0)
	if err := layFrame(front, steady, 0, 1); err != nil {
		t.Fatal(err)
	}
	if front.light[0] != 0 || front.light[3] != 1 {
		t.Errorf("a spruce at pixel 0 did not hide the frame: %v", front.light)
	}
}
