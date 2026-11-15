//go:build draw

package smoke_test

import (
	"image"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/shortcut"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
	"github.com/janearc/slayer/trace"
)

// The numbers of arithmetic every test below leans on.
const (
	nought = 0 // defined: no amount
	unity  = 1 // defined: one
	pair   = 2 // defined: two, as in half and twice
	three  = 3 // defined: three
	four   = 4 // defined: four
	five   = 5 // defined: five
)

// The smokes the tests draw, by their number.
const (
	smokeFive       = 5  // defined: the number of the stage smoke forty-seven is made from
	smokeForty      = 40 // defined: the number of four atmospheres
	smokeFortyOne   = 41 // defined: the number of a laser in haze
	smokeFortyTwo   = 42 // defined: the number of a hill and a closed solid
	smokeFortyThree = 43 // defined: the number of a glass fizzing
	smokeFortyFour  = 44 // defined: the number of the earth by night
	smokeFortyFive  = 45 // defined: the number of three planets on a track
	smokeFortySix   = 46 // defined: the number of a baked puff
	smokeFortySeven = 47 // defined: the number of a stage lit by ninjafilm's lamps
)

// undrawnNoise is how far a float32 frame can be from exact: a sum stored
// once against two stored separately.
const undrawnNoise = 1e-6 // measured: the float32 spacing near one is 1.2e-7, so about ten of it

// The size the tests of the undrawn smokes draw at: a sextant's shape, a
// little under the terminal's, so a test costs seconds.
const (
	undrawnW     = 240  // picked: a sextant's shape, a little under the terminal's
	undrawnH     = 100  // picked: a sextant's shape, a little under the terminal's
	undrawnRays  = 2    // picked: two rays a side keep an edge's pixels from speckling
	sextantPixel = 0.75 // defined: a sextant pixel is half a cell wide and a third of one tall, on a cell twice as tall as it is wide
)

// withinExact is how near two numbers that the same arithmetic gives must be.
const withinExact = 1e-9 // picked: a thousand million to one, far above rounding and far below a drawn effect

// tiny is how near to zero a sum that should be zero must be.
const tiny = 1e-12 // picked: far under withinExact, for a density that is a product of two

// lensOf is the lens a smoke is drawn through at the tests' size: its field
// of view and a sextant's pixels.
func lensOf(t *testing.T, number int) film.Pinhole {
	t.Helper()
	made, ok := smoke.Get(number)
	if !ok {
		t.Fatalf("there is no smoke %d", number)
	}
	pinhole, err := film.PinholeOf(made.Wide)
	if err != nil {
		t.Fatalf("smoke %d: %v", number, err)
	}
	pinhole.Pixel = sextantPixel
	return *pinhole
}

// undrawnShot is smoke number's scene drawn at the test's size, with change
// applied to its world first when there is one.
func undrawnShot(t *testing.T, number int, change func(*thing.World)) *frame.Frame {
	t.Helper()
	made, ok := smoke.Get(number)
	if !ok {
		t.Fatalf("there is no smoke %d", number)
	}
	stage, eye := made.Scene()
	if change != nil {
		change(stage)
	}
	lens := lensOf(t, number)
	shot, err := trace.Sampled(stage, eye, lens, nought, undrawnW, undrawnH, undrawnRays)
	if err != nil {
		t.Fatalf("smoke %d: %v", number, err)
	}
	return shot
}

// dropVolumes takes every volume out of a world.
func dropVolumes(stage *thing.World) { stage.Volumes = nil }

// dropThings takes every thing out of a world.
func dropThings(stage *thing.World) { stage.Things = nil }

// keepFloor leaves a world its first thing, which is its floor.
func keepFloor(stage *thing.World) { stage.Things = stage.Things[:unity] }

// withoutVolumes is the picture of smoke number with its volumes taken away: what is
// behind them, for a test to take from the picture to leave only them.
func withoutVolumes(t *testing.T, number int) *frame.Frame {
	t.Helper()
	return undrawnShot(t, number, dropVolumes)
}

// added is the light a picture has over a picture of the same scene
// without the thing under test, summed over a box of pixels.
func added(with, without *frame.Frame, x0, x1, y0, y1 int) maths.Light {
	var sum maths.Light
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			lit, unlit := with.At(x, y).Light(), without.At(x, y).Light()
			sum = sum.Add(maths.RGB(lit.R-unlit.R, lit.G-unlit.G, lit.B-unlit.B))
		}
	}
	return sum
}

// brightest is the most light one pixel of a box has over the same pixel
// of a picture of the scene without the thing under test.
func brightest(with, without *frame.Frame, x0, x1, y0, y1 int) float64 {
	var most float64
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			lit, unlit := with.At(x, y).Light(), without.At(x, y).Light()
			most = math.Max(most, lit.Luminance()-unlit.Luminance())
		}
	}
	return most
}

// The balls of smoke forty, left to right, by their index in the row.
const (
	earthBall = iota // defined: the first of the row is the earth's air
	hazeBall         // defined: the second is the haze
	venusBall        // defined: the third is venus's air
	marsBall         // defined: the fourth is mars's air
)

// Smoke forty's geometry as the picture shows it: how far apart the balls
// stand, how far the eye is from them, how wide the frame is, where the
// first ball is and how much of a unit is a ball's box.
const (
	fortyApart    = 2.6               // picked: the smoke's own, from its builder
	fortyDistance = 11                // picked: the smoke's own, from its builder
	fortyHalfWide = 25 * maths.Degree // picked: half of fifty degrees across, the smoke's frame
	fortyFirst    = 1.5               // defined: a centred row of four starts one and a half gaps left of the middle
	fortyHalfBox  = 0.5               // picked: a box half a unit either side of a ball's middle, inside its limb
)

// What smoke forty's balls must show: how much bluer than red the earth's
// is, how many times redder than the earth's mars's is, how near to grey the
// haze is, and the least light a ball adds a pixel.
const (
	blueOverRed   = 1.5  // picked: rayleigh's fourth power gives 2.96 times, so this is well short
	marsOverEarth = 2    // picked: mars's red over blue is more than twice the earth's
	hazeGrey      = 0.15 // picked: a haze is the sun's colour to within a seventh
	leastBall     = 0.05 // picked: a fifth of mars's, the thinnest ball
)

// fortyRig is smoke forty drawn with its balls and without them, and the
// size of a unit in pixels at its distance.
type fortyRig struct {
	with, bare *frame.Frame
	unit       float64
	balls      [four]maths.Light
}

// newFortyRig draws smoke forty both ways and measures the light each ball
// adds.
func newFortyRig(t *testing.T) *fortyRig {
	t.Helper()
	rig := &fortyRig{
		with: undrawnShot(t, smokeForty, nil),
		bare: withoutVolumes(t, smokeForty),
	}
	// a ball's middle, in pixels: the balls stand 2.6 units apart in
	// the row, the middle two at plus and minus 1.3, seen from eleven
	// units under fifty degrees across, so a unit is W over twice
	// eleven times tan of twenty-five degrees pixels across
	rig.unit = undrawnW / (pair * fortyDistance * math.Tan(fortyHalfWide))
	for i := range rig.balls {
		x0, x1, y0, y1 := rig.span(i)
		rig.balls[i] = added(rig.with, rig.bare, x0, x1, y0, y1)
	}
	return rig
}

// span is the box of pixels of the ball at index i of the row.
func (rig *fortyRig) span(i int) (x0, x1, y0, y1 int) {
	x := undrawnW/pair + (float64(i)-fortyFirst)*fortyApart*rig.unit
	half := int(fortyHalfBox * rig.unit)
	mid := undrawnH / pair
	return int(x) - half, int(x) + half, mid - half, mid + half
}

// peak is the brightest pixel the ball at index i adds, over the whole ball,
// lit limb and all.
func (rig *fortyRig) peak(i int) float64 {
	x0, x1, y0, y1 := rig.span(i)
	half := int(rig.unit)
	margin := half / pair
	return brightest(rig.with, rig.bare, x0-margin, x1+margin, y0-margin, y1+margin)
}

// earthIsBlue checks that the earth's air scatters the short end most.
func (rig *fortyRig) earthIsBlue(t *testing.T) {
	// rayleigh's fourth power says 465 nm scatters (610/465)^4,
	// 2.96, times 610 nm; a lit ball is bluer than red by more
	// than that is shy of two
	earth := rig.balls[earthBall]
	if !(earth.B > blueOverRed*earth.R) {
		t.Errorf("earth's ball is red %.4f, blue %.4f", earth.R, earth.B)
	}
}

// marsIsRed checks that mars's dust scatters flat and takes the blue.
func (rig *fortyRig) marsIsRed(t *testing.T) {
	earth, mars := rig.balls[earthBall], rig.balls[marsBall]
	if !(mars.R > mars.B) {
		t.Errorf("mars's ball is red %.4f, blue %.4f", mars.R, mars.B)
	}
	if !(mars.R/mars.B > marsOverEarth*earth.R/earth.B) {
		t.Errorf("red over blue is %.3f for mars and %.3f for earth",
			mars.R/mars.B, earth.R/earth.B)
	}
}

// hazeIsGrey checks that the haze scatters every band alike.
func (rig *fortyRig) hazeIsGrey(t *testing.T) {
	haze := rig.balls[hazeBall]
	if !(math.Abs(haze.R/haze.B-unity) < hazeGrey) {
		t.Errorf("the haze is red %.4f, blue %.4f", haze.R, haze.B)
	}
}

// venusIsBrightest checks that venus's lit limb is the brightest.
func (rig *fortyRig) venusIsBrightest(t *testing.T) {
	// venus's visible scatter is 2e-3 a metre against the
	// earth's 1.16e-5, 172 times, so what a metre of it sends
	// toward the eye is far more, though a ball of it is no
	// thicker to green than the others
	venus, earth, mars := rig.peak(venusBall), rig.peak(earthBall), rig.peak(marsBall)
	if !(venus > earth && venus > mars) {
		t.Errorf("brightest pixel added: venus %.3f, earth %.3f, mars %.3f",
			venus, earth, mars)
	}
}

// ballsAddLight checks that each ball adds light over the dark behind it.
func (rig *fortyRig) ballsAddLight(t *testing.T) {
	// the least of them, mars's, is a fifth of a unit of light a
	// pixel; a twentieth is far from none and far from a ball
	// that is not there, which adds a millionth
	side := int(fortyHalfBox * rig.unit)
	area := float64(four * side * side)
	for i, light := range rig.balls {
		if !(light.Luminance()/area > leastBall) {
			t.Errorf("ball %d adds %v a pixel: %v", i, light.Luminance()/area, light)
		}
	}
}

// Smoke forty, four atmospheres: each ball is the colour its gas's numbers
// make it, and none is there without its gas.
func TestSmoke40FourAtmospheres(t *testing.T) {
	rig := newFortyRig(t)
	t.Run("earth's air is blue: the short end scatters most", rig.earthIsBlue)
	t.Run("mars is red of earth: its dust scatters flat and takes the blue", rig.marsIsRed)
	t.Run("the haze scatters every band alike: it is the sun's own colour", rig.hazeIsGrey)
	t.Run("venus's lit limb is the brightest: its air scatters the most a metre", rig.venusIsBrightest)
	t.Run("each ball adds light over the dark behind it", rig.ballsAddLight)
}

// brightestAt is where in a picture the light is greatest.
func brightestAt(shot *frame.Frame) (x, y int) {
	most := math.Inf(-unity)
	for row := range shot.H {
		for column := range shot.W {
			if light := shot.At(column, row).Light().Luminance(); light > most {
				most, x, y = light, column, row
			}
		}
	}
	return x, y
}

// What smoke forty-one's picture must show: how far the spot may move
// between clear air and haze, the columns the beam is looked for in, how
// much brighter it is than the dark, and how red.
const (
	spotSlackX   = 6    // picked: columns the haze may move the spot, under its width
	spotSlackY   = 4    // picked: rows the haze may move the spot
	beamFirst    = 5    // picked: the first column looked at, clear of the frame's edge
	beamStep     = 5    // picked: columns between the ones looked at
	beamStopShy  = 40   // picked: columns short of the spot the looking stops, where the beam meets its glare
	beamOverDark = 20   // picked: the beam is twenty times the dark of the same column
	beamFloor    = 1e-6 // picked: added so a column of pure black is not a tie
	clearDark    = 0.01 // picked: clear air is under a hundredth of the beam
	redOverGreen = 10   // picked: the laser is 150 to 5 in red to green, thirty times; this is well short
	beamColumns  = 3    // picked: fewer columns than this is no line
)

// hazeRig is smoke forty-one drawn in haze and in clear air, and where the
// spot is in clear air.
type hazeRig struct {
	hazy, clear  *frame.Frame
	spotX, spotY int
}

// spotIsOnTheFloor checks the spot is where it was in both pictures.
func (rig *hazeRig) spotIsOnTheFloor(t *testing.T) {
	// the spot is the picture's brightest in clear air; the haze
	// does not move it, it lies in the beam's way, a little
	// dimmer
	hazeX, hazeY := brightestAt(rig.hazy)
	if math.Abs(float64(hazeX-rig.spotX)) > spotSlackX || math.Abs(float64(hazeY-rig.spotY)) > spotSlackY {
		t.Errorf("the spot is at %d,%d in clear air and %d,%d in haze",
			rig.spotX, rig.spotY, hazeX, hazeY)
	}
}

// brightestRow is the row of a column of a picture where the light is
// greatest, and how much light that is.
func brightestRow(shot *frame.Frame, column int) (row int, most float64) {
	for y := range shot.H {
		if light := shot.At(column, y).Light().Luminance(); light > most {
			row, most = y, light
		}
	}
	return row, most
}

// checkColumn checks the beam at one column: it is bright against the dark,
// absent in clear air and red. It returns the row the beam is at.
func (rig *hazeRig) checkColumn(t *testing.T, x, lastRow int) int {
	t.Helper()
	row, most := brightestRow(rig.hazy, x)
	// the beam runs from the upper left down to the right: its row
	// never rises as the column goes right
	if row < lastRow {
		t.Errorf("column %d: the beam's row is %d after %d", x, row, lastRow)
	}
	// it is bright against the dark of the same column far from it
	far := (row + rig.hazy.H/pair) % rig.hazy.H
	dark := rig.hazy.At(x, far).Light().Luminance()
	if !(most > beamOverDark*dark+beamFloor) {
		t.Errorf("column %d: the beam is %.4f and the dark %.4f", x, most, dark)
	}
	// in clear air the same pixel is dark, and in haze it is red
	// by the laser's own colour, 150 to 5 in red to green
	if clearLight := rig.clear.At(x, row).Light(); !(clearLight.Luminance() < clearDark*most) {
		t.Errorf("column %d: clear air is lit %.4f where haze is %.4f", x,
			clearLight.Luminance(), most)
	}
	if hazeLight := rig.hazy.At(x, row).Light(); !(hazeLight.R > redOverGreen*hazeLight.G) {
		t.Errorf("column %d: the beam is %v, not the laser's red", x, hazeLight)
	}
	return row
}

// beamIsALine checks the beam is a line down to the spot, red, and absent in
// clear air.
func (rig *hazeRig) beamIsALine(t *testing.T) {
	lastRow, shown := nought, nought
	for x := beamFirst; x < rig.spotX-beamStopShy; x += beamStep {
		lastRow = rig.checkColumn(t, x, lastRow)
		shown++
	}
	if shown < beamColumns {
		t.Errorf("only %d columns were looked at", shown)
	}
}

// Smoke forty-one, a laser in haze: in the haze the beam shows as a line
// of its own colour from the upper left down to its spot; in clear air the
// same laser shows its spot and nothing of the beam.
func TestSmoke41ALaserInHaze(t *testing.T) {
	rig := &hazeRig{
		hazy:  undrawnShot(t, smokeFortyOne, nil),
		clear: withoutVolumes(t, smokeFortyOne),
	}
	rig.spotX, rig.spotY = brightestAt(rig.clear)
	t.Run("the spot is on the floor in both, and the beam only in the haze", rig.spotIsOnTheFloor)
	t.Run("the beam is a line down to the spot, red, and absent in clear air", rig.beamIsALine)
}

// shapeNamed is the shape of the thing of smoke number whose material has this
// name, for a test to ask it questions in its own space.
func shapeNamed(t *testing.T, number int, name string) shape.Shape {
	t.Helper()
	made, _ := smoke.Get(number)
	stage, _ := made.Scene()
	for _, each := range stage.Things {
		if each.Material != nil && each.Material.Name == name {
			return each.Shape
		}
	}
	t.Fatalf("smoke %d has no thing named %q", number, name)
	return nil
}

// The gem of smoke forty-two, in units: how many sides it has, where the
// points inside and outside it lie along each axis.
const (
	gemSides     = 7    // picked: the smoke's own, from its builder
	gemHalfway   = 0.45 // picked: half way to a corner of 0.9
	gemNearSide  = 0.85 // picked: near a corner, 0.85 of 0.9
	gemNearPoint = 1.0  // picked: near a point, short of 1.15
	gemPastPoint = 1.3  // picked: past a point, which is 1.15
	gemFarOff    = 3.0  // picked: far from a gem a unit or so across
)

// gemInside are points that lie inside the gem, by name.
func gemInside() map[string]maths.Vec {
	return map[string]maths.Vec{
		"the middle":                     maths.V(nought, nought, nought),
		"half way to a corner":           maths.V(gemHalfway, nought, nought),
		"near a corner, 0.85 of 0.9":     maths.V(gemNearSide, nought, nought),
		"near the top, short of 1.15":    maths.V(nought, gemNearPoint, nought),
		"near the bottom, short of 1.15": maths.V(nought, -gemNearPoint, nought),
	}
}

// gemOutside are points that lie outside the gem, by name.
func gemOutside() map[string]maths.Vec {
	edge := math.Pi / gemSides
	return map[string]maths.Vec{
		"past the top":                   maths.V(nought, gemPastPoint, nought),
		"past the bottom":                maths.V(nought, -gemPastPoint, nought),
		"0.85 from the middle at a side": maths.V(gemNearSide*math.Cos(edge), nought, -gemNearSide*math.Sin(edge)),
		"far off":                        maths.V(gemFarOff, nought, nought),
	}
}

// gemIsClosed checks the gem is closed and a point is inside it by its
// faces.
func gemIsClosed(t *testing.T) {
	gem, ok := shapeNamed(t, smokeFortyTwo, "gem").(*shape.Mesh)
	if !ok {
		t.Fatal("the gem is not a mesh")
	}
	if !gem.Closed() {
		t.Error("the gem is not closed")
	}
	// the girdle is a seven-sided ring of radius 0.9, whose first
	// corner is at x = 0.9, and whose sides are 0.9 cos(pi/7) =
	// 0.811 from the middle, at an angle of pi/7 to a corner
	for name, point := range gemInside() {
		if !gem.Inside(point) {
			t.Errorf("%s, %v, is not inside", name, point)
		}
	}
	for name, point := range gemOutside() {
		if gem.Inside(point) {
			t.Errorf("%s, %v, is inside", name, point)
		}
	}
}

// The hill of smoke forty-two: how far a point is moved to be sure it is
// either side of the surface, where the checked points are, and how high
// the hill is there.
const (
	hillNear           = 0.01   // picked: a hundredth of a unit, either side of the surface
	hillMiddleHeight   = 0.9    // picked: the smoke's own, from its builder
	hillShoulder       = 0.2    // picked: a point off the middle, where the ripple is not zero
	hillShoulderHeight = 0.8698 // measured: the bell and the ripple worked by hand from the smoke's numbers
)

// checkHillAt checks that a point a hundredth under the surface at x and
// z is under it, and a hundredth over it is over.
func checkHillAt(t *testing.T, hill *shape.HeightField, x, z, height float64) {
	t.Helper()
	under := hill.Distance(maths.V(x, height-hillNear, z))
	over := hill.Distance(maths.V(x, height+hillNear, z))
	if !(under < nought) || !(over > nought) {
		t.Errorf("at %v, %v: %v a hundredth under %v, over %v, want "+
			"negative and positive", x, z, height, under, over)
	}
}

// hillHasItsRipple checks the hill is a height field with its ripple.
func hillHasItsRipple(t *testing.T) {
	hill, ok := shapeNamed(t, smokeFortyTwo, "hill").(*shape.HeightField)
	if !ok {
		t.Fatal("the hill is not a height field")
	}
	// over a point the surface is at a height, and a point a
	// hundredth under it is under the surface, a hundredth over it
	// over it: the middle is 0.9, where the ripple is zero; at
	// 0.2, 0.2, the bell is (1 - 0.08/1.69)^2 = 0.9077 of 0.9
	// plus the ripple 0.06 sin(1.4)^2 = 0.0583, 0.8698; without
	// the ripple it would be 0.817
	checkHillAt(t, hill, nought, nought, hillMiddleHeight)
	checkHillAt(t, hill, hillShoulder, hillShoulder, hillShoulderHeight)
}

// What smoke forty-two's picture must show: how much light a pixel must
// change by to be counted changed, and the share of its half of the frame
// each of the hill and the gem must change.
const (
	changedBy    = 0.05 // picked: a twentieth of a unit of light, far above the noise
	changedShare = 0.1  // picked: a tenth of its half of the frame; the hill takes a fifth and the gem a sixth
)

// changedShareOf is the share of the pixels of the columns x0 up to x1 whose
// light differs between two pictures by more than changedBy.
func changedShareOf(with, bare *frame.Frame, x0, x1 int) float64 {
	var count int
	for y := range with.H {
		for x := x0; x < x1; x++ {
			lit, unlit := with.At(x, y).Light(), bare.At(x, y).Light()
			if math.Abs(lit.Luminance()-unlit.Luminance()) > changedBy {
				count++
			}
		}
	}
	return float64(count) / float64((x1-x0)*with.H)
}

// hillAndGemAreDrawn checks the hill and the gem are both in the picture.
func hillAndGemAreDrawn(t *testing.T) {
	with := undrawnShot(t, smokeFortyTwo, nil)
	bare := undrawnShot(t, smokeFortyTwo, keepFloor)
	left := changedShareOf(with, bare, nought, undrawnW/pair)
	right := changedShareOf(with, bare, undrawnW/pair, undrawnW)
	// the hill takes a fifth of its half of the frame and the gem
	// a sixth of its half; a tenth of each is well short of both
	if !(left > changedShare) || !(right > changedShare) {
		t.Errorf("the hill changes %.3f of its half, the gem %.3f", left, right)
	}
}

// Smoke forty-two, a hill and a closed solid: the gem is a closed mesh
// and says what is inside it, flat faces and all; the hill is a height
// field with the ripple it was given; and both are drawn.
func TestSmoke42AHillAndAClosedSolid(t *testing.T) {
	t.Run("the gem is closed and a point is inside it by its faces", gemIsClosed)
	t.Run("the hill is a height field with its ripple", hillHasItsRipple)
	t.Run("the hill and the gem are both in the picture", hillAndGemAreDrawn)
}

// onlyVolume is the index of the one volume smoke forty-three has.
const onlyVolume = nought // defined: the first of one

// fizzOf is the fizz of smoke 43 and the same fizz with no churn, so a
// test can tell the profile from the ragged.
func fizzOf(t *testing.T) (churned, plain medium.Fizz) {
	t.Helper()
	made, _ := smoke.Get(smokeFortyThree)
	stage, _ := made.Scene()
	if len(stage.Volumes) != unity {
		t.Fatalf("smoke 43 has %d volumes, want 1", len(stage.Volumes))
	}
	field := stage.Volumes[onlyVolume].Medium
	churned, ok := field.(medium.Fizz)
	if !ok {
		t.Fatalf("smoke 43's volume is %T, want a Fizz", field)
	}
	plain = churned
	plain.Churn = nought
	return churned, plain
}

// Smoke forty-three's numbers as its builder wrote them, and what the
// tests of it hold it to.
const (
	fizzThick      = 4.0  // picked: the smoke's thickness, as written in its builder
	fizzChurn      = 0.75 // picked: the smoke's churn, as written in its builder
	fizzRise       = 0.3  // picked: the smoke's rise a second, as written in its builder
	fizzSeconds    = 2    // picked: long enough for the rise to be well over the noise
	fizzHalfway    = 0.5  // defined: half way up the column
	fizzGrid       = 20   // picked: nineteen points a side, a few hundred in all, enough to be ragged
	fizzStepAcross = 0.01 // picked: a hundredth of a unit across the column a step
	fizzStepUp     = 0.1  // picked: a tenth of a unit up the column a step
	fizzThinned    = 0.9  // picked: a point is thinned if the churn takes a tenth of it
	fizzRaggedOf   = 4    // picked: a quarter of the points must be thinned
	fizzFootOver   = 1.5  // picked: the middle of the column is brighter than its head by half
)

// fizzRig is the fizz of smoke forty-three, churned and plain.
type fizzRig struct {
	churned, plain medium.Fizz
}

// checkDensity checks the plain fizz's density at a share of the way up
// the column.
func (rig *fizzRig) checkDensity(t *testing.T, up, want float64) {
	t.Helper()
	at := maths.V(nought, up*rig.plain.Height, nought)
	if got := rig.plain.Density(at); !(math.Abs(got-want) <= withinExact) {
		t.Errorf("density %v of the way up is %v, want %v", up, got, want)
	}
}

// profileIsTheSmoothStep checks the column is thick at the foot, half as
// thick half way up and nothing at the top.
func (rig *fizzRig) profileIsTheSmoothStep(t *testing.T) {
	// by the fizz's own law, thick times one minus a smooth step
	// of the way up: at the foot, 4; half way, the smooth step is
	// one half, so 2; at the top, 0
	rig.checkDensity(t, nought, fizzThick)
	rig.checkDensity(t, fizzHalfway, fizzThick/pair)
	rig.checkDensity(t, unity, nought)
}

// raggedPoint is a point of the column the churn is checked at, by its
// two steps.
func raggedPoint(i, step int) maths.Vec {
	across, up := fizzStepAcross*float64(i), fizzStepUp*float64(step)
	return maths.V(across, up, fizzStepAcross*float64(i+step))
}

// churnIsRagged checks the churn takes no more than it was given, and
// takes some from most of the column.
func (rig *fizzRig) churnIsRagged(t *testing.T) {
	// at every point the churned density is the plain one times
	// one minus the thinning, which is at most the churn of 0.75:
	// between a quarter and all of it
	ragged, points := nought, nought
	for i := unity; i < fizzGrid; i++ {
		for step := unity; step < fizzGrid; step++ {
			at := raggedPoint(i, step)
			was, now := rig.plain.Density(at), rig.churned.Density(at)
			if was == nought {
				continue
			}
			points++
			if !(now <= was+tiny) || !(now >= (unity-fizzChurn)*was-tiny) {
				t.Fatalf("at %v the churned density %v is outside the "+
					"quarter to all of %v", at, now, was)
			}
			if now < fizzThinned*was {
				ragged++
			}
		}
	}
	if ragged < points/fizzRaggedOf {
		t.Errorf("only %d of %d points are thinned by a tenth or more", ragged, points)
	}
}

// Where smoke forty-three's churn is looked at to see it rise.
var (
	riseNear = maths.V(0.05, 0.9, 0.02) // picked: a point inside the column, off its middle
	riseFar  = maths.V(-0.1, 1.2, 0.07) // picked: a point higher up and on the other side
)

// churnRises checks the churn rises with the bubbles.
func (rig *fizzRig) churnRises(t *testing.T) {
	// the thinning, the churned density over the plain, at a time
	// later is what it was a rise lower: 0.3 a second for two
	// seconds is 0.6 of a unit
	const rise = fizzRise * fizzSeconds
	later := rig.churned.Tick(fizzSeconds * time.Second)
	for _, at := range []maths.Vec{riseNear, riseFar} {
		low := maths.V(at.X, at.Y-rise, at.Z)
		now := later.Density(at) / rig.plain.Density(at)
		then := rig.churned.Density(low) / rig.plain.Density(low)
		// the profile is the plain fizz's, which does not move
		// with it, so compare the thinning alone
		if !(math.Abs(now-then) <= withinExact) {
			t.Errorf("at %v the thinning is %v, and %v a rise lower at t=0",
				at, now, then)
		}
	}
}

// footIsBrighter checks the foot is brighter than the head in the picture.
func footIsBrighter(t *testing.T) {
	with := undrawnShot(t, smokeFortyThree, nil)
	bare := withoutVolumes(t, smokeFortyThree)
	wide, high := with.W, with.H
	// the column is in the middle third across; above the horizon
	// at half height is the dark sky, and the floor is below it,
	// so look at the column's head, the top fifth, and its middle
	head := added(with, bare, wide/three, pair*wide/three, high/(four*five), high/five)
	mid := added(with, bare, wide/three, pair*wide/three, pair*high/five, three*high/five)
	if !(mid.Luminance() > fizzFootOver*head.Luminance()) {
		t.Errorf("light added: head %.2f, middle %.2f", head.Luminance(), mid.Luminance())
	}
}

// Smoke forty-three, a glass fizzing: the column is thick at its foot and
// gone at its top, ragged by the churn and no more than the churn takes,
// the churn rises with the bubbles; and in the picture the foot is brighter
// than the head.
func TestSmoke43AGlassFizzing(t *testing.T) {
	churned, plain := fizzOf(t)
	rig := &fizzRig{churned: churned, plain: plain}
	t.Run("thick at the foot, half as thick half way up, nothing at the top", rig.profileIsTheSmoothStep)
	t.Run("the churn is ragged and takes no more than it was given", rig.churnIsRagged)
	t.Run("the churn rises with the bubbles", rig.churnRises)
	t.Run("the foot is brighter than the head in the picture", footIsBrighter)
}

// reshone changes the picture of lights of the earth in smoke 44: it
// hands each thing whose material shines by a picture to change, and puts
// back what change returns.
func reshone(stage *thing.World, change func(texture.Picture) texture.Picture) {
	for _, each := range stage.Things {
		if each.Material == nil {
			continue
		}
		if lights, ok := each.Material.Shine.(texture.Picture); ok {
			each.Material.Shine = change(lights)
		}
	}
}

// blackPicture is a picture of lights with every light put out.
func blackPicture(lights texture.Picture) texture.Picture {
	lights.Image = image.NewRGBA(lights.Image.Bounds())
	return lights
}

// doubledPicture is a picture of lights with its gain doubled.
func doubledPicture(lights texture.Picture) texture.Picture {
	lights.Gain *= pair
	return lights
}

// blackened puts out the lights of the earth in a world of smoke 44.
func blackened(stage *thing.World) { reshone(stage, blackPicture) }

// doubled doubles the gain of the lights of the earth in a world of
// smoke 44.
func doubled(stage *thing.World) { reshone(stage, doubledPicture) }

// lightOf is a frame's light, the sum of the luminance of every pixel
// above a floor, and the greatest luminance of any, and how many pixels
// are above that floor.
func lightOf(shot *frame.Frame, floor float64) (sum, most float64, lit int) {
	for row := range shot.H {
		for column := range shot.W {
			light := shot.At(column, row).Light().Luminance()
			most = math.Max(most, light)
			if light > floor {
				sum += light
				lit++
			}
		}
	}
	return sum, most, lit
}

// What smoke forty-four's night side must show, as shares of the glow its
// builder gave the lights: how far over and under the glow its brightest
// pixel may be, the floor a lit pixel is above, and the share of the globe
// the lines may light.
const (
	nightGlow     = 0.004 // picked: the smoke's share, as written in its builder
	glowOver      = 1.01  // picked: a hundredth over the share, for rounding of the sum
	glowUnder     = 0.5   // picked: the lines are wide enough that some pixel is most of the way there
	glowFloor     = 0.1   // picked: a tenth of the glow, far above the picture's noise
	linesLeast    = 0.01  // picked: the lines light more than a hundredth of the globe
	linesMost     = 0.15  // picked: and under a seventh of it
	doubledTol    = 0.02  // picked: twice the share is twice the light to two in a hundred
	belowEveryOne = -1    // picked: below every luminance, so every pixel of the globe counts
)

// nightRig is smoke forty-four with its lights, with none and with twice
// the gain.
type nightRig struct {
	lit, dark, double *frame.Frame
}

// brightestIsTheGlow checks that no pixel is brighter than the share of
// the sun the lights are scaled by.
func (rig *nightRig) brightestIsTheGlow(t *testing.T) {
	// a white pixel of the picture gives the share at most, so
	// nothing the lights give is above it, and the lines are wide
	// enough that some pixel of the frame is most of the way there
	_, most, _ := lightOf(added2(rig.lit, rig.dark), nought)
	if !(most <= glowOver*nightGlow) || !(most >= glowUnder*nightGlow) {
		t.Errorf("the lights' brightest pixel is %v, want between %v and %v",
			most, glowUnder*nightGlow, glowOver*nightGlow)
	}
}

// linesAreThin checks the lights are thin lines and the black picture has
// none.
func (rig *nightRig) linesAreThin(t *testing.T) {
	// the lines are three pixels in every sixty across and down,
	// a tenth of the picture, and the globe is under a third of
	// the frame: a few percent of the frame, more than a hundredth
	// and under a seventh
	_, _, lines := lightOf(added2(rig.lit, rig.dark), glowFloor*nightGlow)
	_, _, black := lightOf(added2(rig.dark, rig.dark), glowFloor*nightGlow)
	_, _, globe := lightOf(rig.dark, belowEveryOne)
	if black != nought {
		t.Errorf("the black picture lights %d pixels", black)
	}
	share := float64(lines) / float64(globe)
	if !(share > linesLeast) || !(share < linesMost) {
		t.Errorf("the lines light %.3f of the frame, want a few percent", share)
	}
}

// doubleIsTwice checks twice the share is twice the light.
func (rig *nightRig) doubleIsTwice(t *testing.T) {
	one, _, _ := lightOf(added2(rig.lit, rig.dark), nought)
	two, _, _ := lightOf(added2(rig.double, rig.dark), nought)
	if !(math.Abs(two/one-pair) <= doubledTol) {
		t.Errorf("twice the glow gives %v times the light", two/one)
	}
}

// Smoke forty-four, the earth by night from a picture of its lights: the
// night side shows the picture and nothing else, as bright as the share
// of the sun's light it is scaled by, and twice that share is twice the
// light.
func TestSmoke44TheEarthByNightFromItsLights(t *testing.T) {
	rig := &nightRig{
		lit:    undrawnShot(t, smokeFortyFour, nil),
		dark:   undrawnShot(t, smokeFortyFour, blackened),
		double: undrawnShot(t, smokeFortyFour, doubled),
	}
	t.Run("no pixel is brighter than the share of the sun the lights are scaled by", rig.brightestIsTheGlow)
	t.Run("the lights are thin lines, and the black picture has none", rig.linesAreThin)
	t.Run("twice the share is twice the light", rig.doubleIsTwice)
}

// added2 is a frame whose pixels are those of a over b, light only: the
// light a has that b has not, nothing below zero.
func added2(over, under *frame.Frame) *frame.Frame {
	out := frame.New(over.W, over.H)
	for y := range over.H {
		for x := range over.W {
			top, bottom := over.At(x, y), under.At(x, y)
			var pixel frame.Pixel
			pixel.R = max(nought, top.R-bottom.R)
			pixel.G = max(nought, top.G-bottom.G)
			pixel.B = max(nought, top.B-bottom.B)
			out.Set(x, y, pixel)
		}
	}
	return out
}

// Smoke forty-five's numbers as its builder wrote them, and what the
// tests of it hold it to.
const (
	planets     = 3    // picked: the smoke's three planets
	trackDetail = 8    // picked: the smoke's detail of the ground it finds the coast in
	trackRadius = 7    // picked: the smoke's track, seven units from the middle
	planetThird = 4    // picked: the middle half of each third of the frame, where its planet is
	litPerPixel = 0.02 // picked: a planet's lit disc adds a twentieth a pixel, and none is far under a fiftieth
)

// trackRig is the three planets of smoke forty-five on their tracks and
// the ground they are made from.
type trackRig struct {
	tracks []thing.Orbit
	ground *world.World
	face   maths.Vec
}

// newTrackRig finds the three orbits of smoke forty-five and the ground of
// the earth they are made from.
func newTrackRig(t *testing.T) *trackRig {
	t.Helper()
	made, _ := smoke.Get(smokeFortyFive)
	stage, _ := made.Scene()
	rig := &trackRig{}
	for _, each := range stage.Things {
		if orbit, ok := each.Motion.(thing.Orbit); ok {
			rig.tracks = append(rig.tracks, orbit)
		}
	}
	if len(rig.tracks) != planets {
		t.Fatalf("smoke 45 has %d things on a track, want 3", len(rig.tracks))
	}
	earth, err := template.Earth()
	if err != nil {
		t.Fatal(err)
	}
	rig.ground, err = earth.World(trackDetail)
	if err != nil {
		t.Fatal(err)
	}
	return rig
}

// checkPlanetAt checks one planet at one time: it is on the circle, and the
// face it shows the middle is the face planet 0 showed at the start. It
// returns the face.
func (rig *trackRig) checkPlanetAt(t *testing.T, i int, at time.Duration, face maths.Vec) maths.Vec {
	t.Helper()
	space := rig.tracks[i].At(at)
	// the circle is the track's, seven units from the middle
	if got := space.Origin.Len(); !(math.Abs(got-trackRadius) <= withinExact) {
		t.Errorf("planet %d at %v is %v from the middle, want 7", i, at, got)
	}
	// the way to the middle, in the planet's own space: the
	// face it shows the middle
	local := space.Orientation.In(space.Origin.Neg().Unit())
	if i == nought && at == nought {
		face = local
	}
	if gap := local.Sub(face).Len(); !(gap <= withinExact) {
		t.Errorf("planet %d at %v shows %v to the middle, planet 0 at "+
			"the start shows %v", i, at, local, face)
	}
	return face
}

// planetsKeepTheirFace checks each planet keeps one face to the middle of
// the track, the same in all three, through the circuit.
func (rig *trackRig) planetsKeepTheirFace(t *testing.T) {
	var face maths.Vec
	for i, track := range rig.tracks {
		// a quarter and a half of the way round, and the start
		for _, at := range []time.Duration{nought, track.Period / four, track.Period / pair} {
			face = rig.checkPlanetAt(t, i, at, face)
		}
	}
	rig.face = face
}

// faceIsACoast checks the face is a beach with the sea beside it.
func (rig *trackRig) faceIsACoast(t *testing.T) {
	tile := rig.ground.Tiling.Nearest(rig.face)
	if biome := rig.ground.Biome(tile); biome != world.Beach {
		t.Errorf("the face %v is %v, not a beach", rig.face, biome)
	}
	sea := false
	for _, next := range rig.ground.Tiling.Neighbours(tile) {
		sea = sea || world.Water(rig.ground.Biome(next))
	}
	if !sea {
		t.Errorf("the beach at the face %v has no sea beside it", rig.face)
	}
}

// allThreeAreDrawn checks all three planets are in the picture.
func allThreeAreDrawn(t *testing.T) {
	shot := undrawnShot(t, smokeFortyFive, nil)
	bare := undrawnShot(t, smokeFortyFive, dropThings)
	third := shot.W / three
	for i := range planets {
		// the middle of each third, where its planet is: the
		// planets are three radii apart and the frame sixty
		// degrees across; a planet's lit disc adds about a
		// twentieth of a unit of light a pixel there, and none
		// added is far under a fiftieth
		mid := added(shot, bare, i*third+third/planetThird, (i+unity)*third-third/planetThird,
			shot.H/three, pair*shot.H/three)
		if !(mid.Luminance() > litPerPixel*float64(third/pair*shot.H/three)) {
			t.Errorf("planet %d adds only %v", i, mid.Luminance())
		}
	}
}

// Smoke forty-five, a planet on a track of the caller's: each of the three
// planets goes round the same circle, and each keeps one face to the middle
// of it, which is the same face in all three and a coast.
func TestSmoke45ThreePlanetsOnATrack(t *testing.T) {
	rig := newTrackRig(t)
	rig.planetsKeepTheirFace(t)
	t.Run("the face is a coast: a beach with the sea beside it", rig.faceIsACoast)
	t.Run("all three are in the picture", allThreeAreDrawn)
}

// exactOf is a copy of the world of smoke 46 with the baked puff put
// back to the medium it was baked from, so that both puffs are exact.
func exactOf(t *testing.T, stage *thing.World) *thing.World {
	t.Helper()
	out := *stage
	out.Volumes = nil
	for _, volume := range stage.Volumes {
		each := *volume
		if baked, ok := volume.Medium.(*shortcut.Baked); ok {
			each.Medium = baked.Of
		}
		out.Volumes = append(out.Volumes, &each)
	}
	return &out
}

// puffNamed is the volume of a world whose material has the name.
func puffNamed(t *testing.T, stage *thing.World, name string) *thing.Volume {
	t.Helper()
	for _, volume := range stage.Volumes {
		if volume.Material != nil && volume.Material.Name == name {
			return volume
		}
	}
	t.Fatalf("no volume is named %q", name)
	return nil
}

// What smoke forty-six's baked puff is held to: the sample points along
// the puff, how thick the puff is, and how near the bake is to it.
const (
	bakeSamples = 12   // picked: eleven points along the diagonal of the puff
	bakeStepX   = 0.07 // picked: the sample point's step along x
	bakeStepY   = 0.05 // picked: and along y
	bakeStepZ   = 0.04 // picked: and along z
	bakeBackX   = 0.4  // picked: where the diagonal starts, inside the puff, along x
	bakeBackY   = 0.3  // picked: and along y
	bakeBackZ   = 0.2  // picked: and along z
	bakeThick   = 2.5  // picked: the puff's thickness, as the smoke wrote it
	bakeNear    = 0.05 // picked: a bake of sixty-four cells over two units is within a twentieth of the thickness
	bakeDiffers = 1e-9 // picked: a thousand million to one, far above rounding, for a density that differs
)

// bakeRig is smoke forty-six drawn through its own eye and lens.
type bakeRig struct {
	stage *thing.World
	exact *thing.World
	eye   maths.Subspace
	lens  film.Pinhole
}

// draw is a world drawn through the smoke's eye and lens at the tests' size.
func (rig *bakeRig) draw(t *testing.T, stage *thing.World) *frame.Frame {
	t.Helper()
	shot, err := trace.Sampled(stage, rig.eye, rig.lens, nought, undrawnW, undrawnH, undrawnRays)
	if err != nil {
		t.Fatal(err)
	}
	return shot
}

// bakeOnly is a plan that bakes the puff the smoke names and no other.
func bakeOnly(t *testing.T) shortcut.Plan {
	t.Helper()
	taken, err := shortcut.Parse("baked(64)")
	if err != nil {
		t.Fatal(err)
	}
	var plan shortcut.Plan
	plan.Global = shortcut.None()
	return plan.With("baked puff", taken)
}

// planBakesOnlyTheNamedPuff checks the plan bakes the puff it names and no
// other.
func (rig *bakeRig) planBakesOnlyTheNamedPuff(t *testing.T) {
	baked := puffNamed(t, rig.stage, "baked puff").Medium
	if _, ok := baked.(*shortcut.Baked); !ok {
		t.Errorf("the baked puff's medium is %T", baked)
	}
	if _, ok := puffNamed(t, rig.stage, "exact puff").Medium.(*shortcut.Baked); ok {
		t.Error("the exact puff is baked")
	}
	plan := bakeOnly(t)
	if !plan.For("baked puff").Baked || plan.For("exact puff").Any() {
		t.Errorf("the plan says %v", plan.Record())
	}
	want := []string{"global: none", "baked puff: baked(64)"}
	if got := plan.Record(); !slices.Equal(got, want) {
		t.Errorf("the plan's record is %q, want %q", got, want)
	}
}

// bakeIsWithinBudget checks the bake is within the budget of an eye of
// the exact picture.
func (rig *bakeRig) bakeIsWithinBudget(t *testing.T) {
	// libtheme's two hundredths, as an oklab distance: the budget
	// the package states for a bake
	var tone colour.Tone
	worst, mean := shortcut.Error(rig.draw(t, rig.exact), rig.draw(t, rig.stage), tone)
	if !shortcut.Honest("baked", worst) || !(mean <= worst) {
		t.Errorf("a baked puff is %v from exact at worst, %v on average, "+
			"over the budget %v", worst, mean, shortcut.Budgets["baked"])
	}
}

// bakedDensityIsNearExact checks the baked density is not the exact
// density, and is near it.
func (rig *bakeRig) bakedDensityIsNearExact(t *testing.T) {
	bakedMedium := puffNamed(t, rig.stage, "baked puff").Medium
	exactMedium := puffNamed(t, rig.exact, "baked puff").Medium
	differ, nearest := nought, true
	for i := unity; i < bakeSamples; i++ {
		step := float64(i)
		at := maths.V(bakeStepX*step-bakeBackX, bakeStepY*step-bakeBackY, bakeStepZ*step-bakeBackZ)
		bakedAt, exactAt := bakedMedium.Density(at), exactMedium.Density(at)
		if math.Abs(bakedAt-exactAt) > bakeDiffers {
			differ++
		}
		// the puff is two and a half thick: a bake of sixty-four
		// cells over two units is within a twentieth of that
		nearest = nearest && math.Abs(bakedAt-exactAt) <= bakeNear*bakeThick
	}
	if differ == nought || !nearest {
		t.Errorf("%d of 11 points differ, all within a twentieth: %v",
			differ, nearest)
	}
}

// noShortcutLeavesThePicture checks a plan that bakes nothing leaves the
// picture bit for bit.
func (rig *bakeRig) noShortcutLeavesThePicture(t *testing.T) {
	var plan shortcut.Plan
	plan.Global = shortcut.None()
	same := shortcut.Apply(rig.exact, plan)
	before, after := rig.draw(t, rig.exact), rig.draw(t, same)
	for i := range before.Pix {
		if before.Pix[i] != after.Pix[i] {
			t.Fatalf("pixel %d is %+v and with no shortcut %+v", i, before.Pix[i], after.Pix[i])
		}
	}
}

// Smoke forty-six, a baked puff beside the exact one: only the puff the
// plan names is baked; its picture is within the bake's budget of the
// exact one, and a plan that bakes nothing leaves the picture bit for bit.
func TestSmoke46ABakedPuffBesideTheExactOne(t *testing.T) {
	made, _ := smoke.Get(smokeFortySix)
	stage, eye := made.Scene()
	rig := &bakeRig{stage: stage, eye: eye, lens: lensOf(t, smokeFortySix)}
	rig.exact = exactOf(t, stage)
	t.Run("the plan bakes the puff it names and no other", rig.planBakesOnlyTheNamedPuff)
	t.Run("the bake is within the budget of an eye of the exact picture", rig.bakeIsWithinBudget)
	t.Run("the baked density is not the exact density, and is near it", rig.bakedDensityIsNearExact)
	t.Run("a plan that bakes nothing leaves the picture as it was", rig.noShortcutLeavesThePicture)
}

// The five lamps of smoke forty-seven, in the order the smoke makes them.
const (
	sunLamp   = iota // defined: the sun is the first lamp
	floodLamp        // defined: the flood is the second
	spotLamp         // defined: the spot is the third
	laserLamp        // defined: the laser is the fourth
	radarLamp        // defined: the radar is the fifth
	lampCount        // defined: how many lamps there are
)

// What smoke forty-seven's lamps are held to: how much more light a pixel
// must have to be counted, where the laser aims, and how near the dot must
// be.
const (
	sunCounts  = 0.01 // picked: a hundredth of a unit of light, far above the noise
	dotNear    = 0.1  // picked: a tenth of a unit, about a pixel at this distance
	ballRadius = 1.0  // picked: the smoke's ball, as its builder wrote it
	rayFar     = 100  // picked: farther than anything in the smoke
)

// The ball of smoke forty-seven and where the laser's dot lands on it: the
// ball's middle, and where the laser's ray meets it.
var (
	ballMiddle = maths.V(nought, unity, nought)  // picked: the smoke's ball, a unit above the floor
	dotWant    = maths.V(0.7562, 1.1296, 0.6482) // measured: worked by hand from the laser's place and aim and the ball's radius
)

// lampRig is smoke forty-seven and smoke five, drawn through smoke five's eye.
type lampRig struct {
	made smoke.Smoke
	five *thing.World
	eye  maths.Subspace
	lens film.Pinhole
	all  *frame.Frame
}

// draw is a world drawn through smoke five's eye and smoke forty-seven's
// lens.
func (rig *lampRig) draw(t *testing.T, stage *thing.World) *frame.Frame {
	t.Helper()
	shot, err := trace.Sampled(stage, rig.eye, rig.lens, nought, undrawnW, undrawnH, undrawnRays)
	if err != nil {
		t.Fatal(err)
	}
	return shot
}

// lamped is smoke forty-seven with only the lamps named, drawn.
func (rig *lampRig) lamped(t *testing.T, keep ...int) *frame.Frame {
	t.Helper()
	stage, _ := rig.made.Scene()
	if len(stage.Lamps) != lampCount {
		t.Fatalf("smoke 47 has %d lamps, want 5", len(stage.Lamps))
	}
	every := stage.Lamps
	stage.Lamps = nil
	for _, i := range keep {
		stage.Lamps = append(stage.Lamps, every[i])
	}
	return rig.draw(t, stage)
}

// pixelsMatch checks two pictures are the same to the noise of a float32
// frame, naming what they are.
func pixelsMatch(t *testing.T, mine, five *frame.Frame) {
	t.Helper()
	for i, got := range mine.Pix {
		want := five.Pix[i]
		if math.Abs(float64(got.R-want.R)) > undrawnNoise || math.Abs(float64(got.G-want.G)) > undrawnNoise ||
			math.Abs(float64(got.B-want.B)) > undrawnNoise {
			t.Fatalf("pixel %d is %+v and smoke five's is %+v", i, got, want)
		}
	}
}

// spotAndFloodAreSmokeFives checks the spot and the flood are smoke five's,
// number for number.
func (rig *lampRig) spotAndFloodAreSmokeFives(t *testing.T) {
	pixelsMatch(t, rig.lamped(t, floodLamp, spotLamp), rig.draw(t, rig.five))
}

// radarLightsNothing checks the radar lights nothing the eye sees.
func (rig *lampRig) radarLightsNothing(t *testing.T) {
	without := rig.lamped(t, sunLamp, floodLamp, spotLamp, laserLamp)
	for i, got := range rig.all.Pix {
		if got != without.Pix[i] {
			t.Fatalf("pixel %d is %+v and without the radar %+v", i, got, without.Pix[i])
		}
	}
}

// sunAddsLight checks the sun adds light and takes none away.
func (rig *lampRig) sunAddsLight(t *testing.T) {
	with, without := rig.all, rig.lamped(t, floodLamp, spotLamp, laserLamp, radarLamp)
	// the sun adds light and never takes it away
	more, less := nought, nought
	for i, got := range with.Pix {
		gap := got.Light().Luminance() - without.Pix[i].Light().Luminance()
		if gap > sunCounts {
			more++
		}
		if gap < -undrawnNoise {
			less++
		}
	}
	if more == nought || less != nought {
		t.Errorf("the sun lights %d pixels more and %d less", more, less)
	}
}

// reddest is where in a picture the laser adds the most red over a picture
// without it: it adds red, and little else.
func reddest(with, without *frame.Frame) (bx, by int) {
	var most float64
	for y := range with.H {
		for x := range with.W {
			lit, unlit := with.At(x, y).Light(), without.At(x, y).Light()
			if gap := (lit.R - unlit.R) - (lit.B - unlit.B); gap > most {
				bx, by, most = x, y, gap
			}
		}
	}
	return bx, by
}

// dotIsWhereTheAimMeetsTheBall checks the laser's dot is on the ball where
// its aim meets it.
func (rig *lampRig) dotIsWhereTheAimMeetsTheBall(t *testing.T) {
	with, without := rig.all, rig.lamped(t, sunLamp, floodLamp, spotLamp, radarLamp)
	bx, by := reddest(with, without)
	// the laser aims from (3.5, 1.6, 3) at the ball's middle,
	// (0, 1, 0), along (-3.5, -0.6, -3), 4.63 long; it meets the
	// ball a radius short of the middle, at (0.756, 1.130, 0.648)
	var visible maths.Band
	look := rig.lens.Ray(bx, by, with.W, with.H, visible)
	ray := look
	ray.At = rig.eye.Origin
	ray.Dir = rig.eye.Orientation.Out(look.Dir)
	var ball shape.Sphere
	ball.Radius = ballRadius
	inBall := ray
	inBall.At = ray.At.Sub(ballMiddle)
	hit, ok := ball.Hit(inBall, nought, rayFar)
	if !ok {
		t.Fatalf("the brightest red at %d,%d is not on the ball", bx, by)
	}
	got := ray.Point(hit)
	if gap := got.Sub(dotWant).Len(); !(gap <= dotNear) {
		t.Errorf("the dot is at %v, want %v (%.3f off)", got, dotWant, gap)
	}
}

// Smoke forty-seven, a stage lit by ninjafilm's lamps: the spot and flood
// give smoke five's picture, the radar adds nothing the eye sees, and the
// laser's dot lands where its aim says on the ball.
func TestSmoke47AStageLitByNinjafilmsLamps(t *testing.T) {
	five, _ := smoke.Get(smokeFive)
	stage5, eye := five.Scene()
	made, _ := smoke.Get(smokeFortySeven)
	rig := &lampRig{made: made, five: stage5, eye: eye, lens: lensOf(t, smokeFortySeven)}
	rig.all = rig.lamped(t, sunLamp, floodLamp, spotLamp, laserLamp, radarLamp)
	t.Run("the spot and the flood are smoke five's, number for number", rig.spotAndFloodAreSmokeFives)
	t.Run("the radar lights nothing the eye sees", rig.radarLightsNothing)
	t.Run("the sun adds light and takes none away", rig.sunAddsLight)
	t.Run("the laser's dot is on the ball where its aim meets it", rig.dotIsWhereTheAimMeetsTheBall)
}

// Smoke forty-eight and what its film is held to: how far across its
// balls are, how near a ball's middle must be to its rows' share, and
// the film where no ball stands.
const (
	smokeFortyEight = 48   // defined: the number of lead, aluminium and water on an x-ray film
	xrayAcross      = 0.02 // picked: the smoke's balls are two centimetres across, in metres, as its builder wrote them
	xrayNear        = 2e-3 // picked: a middle pixel's ray misses the ball's very middle by half a pixel, which shortens the way through by parts in ten thousand
	xrayBare        = 1.0  // defined: the film where the beam arrives whole, the source's strength
	softXRay        = 1.0  // picked: a nanometre, an x-ray no row of NIST's was read at for the smoke
)

// The things of smoke forty-eight, in the order its builder adds them,
// and the thirds of the film its balls stand in.
const (
	xraySource    = iota // defined: the source is added first
	xrayLead             // defined: lead is the left ball, in the first third
	xrayAluminium        // defined: aluminium is the middle ball
	xrayWater            // defined: water is the right ball, in the last third
	xrayThings           // defined: how many things the smoke has
)

// xrayBalls are the balls of smoke forty-eight by the third of the film
// each stands in, left to right.
var xrayBalls = []string{"lead", "aluminium", "water"}

// xrayRig is smoke forty-eight, drawn in its band.
type xrayRig struct {
	made smoke.Smoke
	eye  maths.Subspace
	lens film.Pinhole
	film *frame.Frame
}

// newXRayRig is smoke forty-eight and its film, drawn once.
func newXRayRig(t *testing.T) *xrayRig {
	t.Helper()
	made, ok := smoke.Get(smokeFortyEight)
	if !ok {
		t.Fatal("there is no smoke 48")
	}
	if len(made.Bands) != unity {
		t.Fatalf("smoke 48 is seen in %d bands, want its one", len(made.Bands))
	}
	stage, eye := made.Scene()
	if len(stage.Things) != xrayThings {
		t.Fatalf("smoke 48 has %d things, want a source and three balls",
			len(stage.Things))
	}
	rig := &xrayRig{made: made, eye: eye, lens: lensOf(t, smokeFortyEight)}
	rig.film = rig.draw(t, stage, made.Bands[nought])
	return rig
}

// draw is a world on the smoke's film, in a band.
func (rig *xrayRig) draw(t *testing.T, stage *thing.World,
	band maths.Band) *frame.Frame {
	t.Helper()
	shot, err := trace.RenderIn(stage, rig.eye, rig.lens, nought,
		undrawnW, undrawnH, band)
	if err != nil {
		t.Fatal(err)
	}
	return shot
}

// darkest is the least light on a film's middle row within one third of
// its width: the first third, the middle, or the last.
func darkest(shot *frame.Frame, third int) float64 {
	width := shot.W / three
	least := math.Inf(unity)
	for x := third * width; x < (third+unity)*width; x++ {
		least = math.Min(least, float64(shot.At(x, shot.H/pair).R))
	}
	return least
}

// xrayShare is what a ball of smoke forty-eight passes through its
// middle, from its material's rows alone.
func xrayShare(t *testing.T, name string) float64 {
	t.Helper()
	kind, err := materials.ByName(name)
	if err != nil {
		t.Fatal(err)
	}
	share, err := kind.Passes(materials.FilmEnergy, xrayAcross)
	if err != nil {
		t.Fatal(err)
	}
	return share
}

// middlesPassWhatTheRowsGive checks the darkest of each ball's shadow,
// its middle, is the share its material's rows give for two centimetres.
func (rig *xrayRig) middlesPassWhatTheRowsGive(t *testing.T) {
	// picked: what counts as nothing on the film. xrayNear is two parts
	// in a thousand, right for aluminium and water and far too loose for
	// lead, whose rows give a share no film can hold: a ball that should
	// pass nothing is held to passing a billionth or less.
	const nothing = 1e-9
	for third, name := range xrayBalls {
		got, want := darkest(rig.film, third), xrayShare(t, name)
		if math.Abs(got-want) > xrayNear {
			t.Errorf("the middle of the %s ball passes %v; its rows give %v",
				name, got, want)
		}
		if want < nothing && got > nothing {
			t.Errorf("the middle of the %s ball passes %v, and its rows "+
				"give nothing", name, got)
		}
	}
}

// filmIsWhiteBesideTheBalls checks the beam arrives whole where no ball
// stands: the film's corners and the gaps between the balls.
func (rig *xrayRig) filmIsWhiteBesideTheBalls(t *testing.T) {
	shot := rig.film
	row := shot.H / pair
	beside := map[string]frame.Pixel{
		"the top left corner":    shot.At(nought, nought),
		"the bottom right":       shot.At(shot.W-unity, shot.H-unity),
		"between lead and metal": shot.At(shot.W/three, row),
		"between metal and soft": shot.At(pair*shot.W/three, row),
	}
	for name, got := range beside {
		if math.Abs(float64(got.R)-xrayBare) > undrawnNoise {
			t.Errorf("%s is %v, not the whole beam", name, got.R)
		}
	}
}

// shadowsDeepenInward checks aluminium's shadow never lightens from its
// edge to its middle, where the way through is longest, and is darker at
// the middle than at the edge.
func (rig *xrayRig) shadowsDeepenInward(t *testing.T) {
	shot := rig.film
	row := shot.H / pair
	from, to := shot.W/three, shot.W/pair
	last := float64(shot.At(from, row).R)
	for x := from + unity; x <= to; x++ {
		here := float64(shot.At(x, row).R)
		if here > last+undrawnNoise {
			t.Fatalf("the shadow lightens going in, at %d: %v after %v",
				x, here, last)
		}
		last = here
	}
	if edge := float64(shot.At(from, row).R); !(last < edge) {
		t.Errorf("the middle, %v, is no darker than the edge, %v", last, edge)
	}
}

// eyeSeesNothing checks the room is dark: no lamp is lit, so the picture
// in the light we see is black all over.
func (rig *xrayRig) eyeSeesNothing(t *testing.T) {
	stage, _ := rig.made.Scene()
	if len(stage.Lamps) != nought {
		t.Fatalf("smoke 48 has %d lamps lit in its dark room", len(stage.Lamps))
	}
	seen, err := trace.Render(stage, rig.eye, rig.lens, nought,
		undrawnW, undrawnH)
	if err != nil {
		t.Fatal(err)
	}
	for i, got := range seen.Pix {
		if got.R != nought || got.G != nought || got.B != nought {
			t.Fatalf("pixel %d is %+v in the dark", i, got)
		}
	}
}

// bareFilmIsWhite is an anti-test of the shadows: with the balls taken
// away the film is the whole beam all over, so the shadows are the
// balls' and nothing else's.
func (rig *xrayRig) bareFilmIsWhite(t *testing.T) {
	stage, _ := rig.made.Scene()
	stage.Things = stage.Things[:xrayLead]
	bare := rig.draw(t, stage, rig.made.Bands[nought])
	for i, got := range bare.Pix {
		if math.Abs(float64(got.R)-xrayBare) > undrawnNoise {
			t.Fatalf("pixel %d of the bare film is %v", i, got.R)
		}
	}
}

// orderIsTheMaterials is an anti-test of the middles: with lead's and
// water's materials changed over, the shadows change over too, so a
// shadow follows what its ball is made of and not where it stands.
func (rig *xrayRig) orderIsTheMaterials(t *testing.T) {
	stage, _ := rig.made.Scene()
	lead, water := stage.Things[xrayLead], stage.Things[xrayWater]
	lead.Material, water.Material = water.Material, lead.Material
	swapped := rig.draw(t, stage, rig.made.Bands[nought])
	left, right := darkest(swapped, nought), darkest(swapped, pair)
	if want := xrayShare(t, "water"); math.Abs(left-want) > xrayNear {
		t.Errorf("the left ball, now water, passes %v; water's rows give %v",
			left, want)
	}
	if want := xrayShare(t, "lead"); math.Abs(right-want) > xrayNear {
		t.Errorf("the right ball, now lead, passes %v; lead's rows give %v",
			right, want)
	}
}

// noRowNoShare is an anti-test of the derivation: in an x-ray band no
// row was read at, the balls pass nothing at all, though the source
// still gives the whole range. Nothing is guessed between rows.
func (rig *xrayRig) noRowNoShare(t *testing.T) {
	stage, _ := rig.made.Scene()
	soft := rig.draw(t, stage, maths.Band{Nanometres: softXRay})
	for third, name := range xrayBalls {
		if got := darkest(soft, third); got != nought {
			t.Errorf("the %s ball passes %v of a band it has no row at",
				name, got)
		}
	}
	if got := soft.At(nought, nought).R; math.Abs(float64(got)-xrayBare) > undrawnNoise {
		t.Errorf("the source gives %v of the soft band, not the whole beam", got)
	}
}

// Smoke forty-eight, lead, aluminium and water on an x-ray film: each
// ball's shadow is the share its material's rows give, the film is white
// beside them, and the eye sees nothing in the dark. The anti-tests:
// without the balls the film is white all over; a shadow follows its
// material and not its place; and a band no row was read at passes
// nothing.
func TestSmoke48LeadAluminiumAndWaterOnAnXRayFilm(t *testing.T) {
	rig := newXRayRig(t)
	t.Run("the middle of each ball passes what its rows give", rig.middlesPassWhatTheRowsGive)
	t.Run("the film is white where no ball stands", rig.filmIsWhiteBesideTheBalls)
	t.Run("a shadow deepens toward its ball's middle", rig.shadowsDeepenInward)
	t.Run("the eye sees nothing in the dark", rig.eyeSeesNothing)
	t.Run("without the balls the film is white all over", rig.bareFilmIsWhite)
	t.Run("a shadow follows its material, not its place", rig.orderIsTheMaterials)
	t.Run("a band no row was read at passes nothing", rig.noRowNoShare)
}
