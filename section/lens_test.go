package section_test

import (
	"fmt"
	"image"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	thin "github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/section"
)

// What the lens trials pick, each by the test author.
const (
	// frameMillimetres are the width and height of a frame of film, 36
	// by 24: a full frame, which the pictures' 3 to 2 matches.
	frameWidthMM  = 36.0
	frameHeightMM = 24.0
	// farFocal, farStop and farFocus make a lens with an aperture of
	// 35 millimetres at half a stop, focused 10 metres off, so that a
	// ray starts up to 35 millimetres from the lens's middle and sees
	// 53 degrees: the scene of the mover trials fits in its view.
	farFocal = 35.0
	farStop  = 0.5
	farFocus = 10.0
	// closeFocal, closeStop and closeFocus make a lens whose aperture,
	// 100 millimetres across a radius, is wide beside what it sees: a
	// ray starts up to a tenth of a unit from the middle and sees 18
	// degrees, focused one unit off.
	closeFocal = 100.0
	closeStop  = 0.5
	closeFocus = 1.0
	// degrees150 and degrees179 are the wide lenses: a field of 150
	// degrees, and 179, the widest the pinhole allows short of 180.
	degrees150 = 150
	degrees179 = 179
	// shortFocal is a focal length, in millimetres, on a full frame that
	// sees 100 degrees: a 15 mm lens is a wide one.
	shortFocal = 15.0
	// balls is how many balls the lens trials aim along pixels' rays.
	balls = 4000
	// ballShare is the largest radius of an aimed ball as a share of its
	// distance: a twentieth, small beside the sections.
	ballShare = 0.05
	// farNearest and farSpread are how near and over how much further a
	// ball is aimed for a lens that sees a scene ten units off.
	farNearest = 0.5
	farSpread  = 40
	// closeNearest and closeSpread are the same for the close lens, which
	// is focused one unit off: from a hundredth of a unit to three.
	closeNearest = 0.01
	closeSpread  = 3
	// The sides of a section the lens trials try, in pixels: one, a few,
	// the trials' own, two of them, and the whole frame.
	sideOne    = 1
	sideFew    = 5
	sideDouble = trialSide * two
	sideFrame  = trialWide
	// oddWide and oddHigh are a frame of 100 by 37 pixels, whose last
	// column and row of sections are narrower than the others, and
	// oddSide a section of 33 pixels on it.
	oddWide = 100
	oddHigh = 37
	oddSide = 33
	// halfTurn is the widest field a pinhole is given here: 179.9
	// degrees, in a hundredth of a degree short of the 180 it refuses.
	halfTurn = 179.9
	// squashed is a pixel three times as wide as it is tall.
	squashed = 3.0
)

// sidesTried are the sides of a section the lens trials try.
func sidesTried() []int {
	return []int{sideOne, sideFew, trialSide, sideDouble, sideFrame}
}

// thinLens is a thin lens with an aperture, on a full frame.
func thinLens(t *testing.T, focal, stop, focus float64) thin.Thin {
	t.Helper()
	frame, err := thin.FrameOf(frameWidthMM, frameHeightMM)
	if err != nil {
		t.Fatal(err)
	}
	glass, err := thin.NewThin(frame, focal, stop, focus)
	if err != nil {
		t.Fatal(err)
	}
	return glass
}

// aim is the ball a lens trial puts along the ray of a pixel, by its
// number k, and the pixel, which its ray meets.
func aim(t *testing.T, view eye, k int64, nearest, spread float64) (ball,
	image.Point) {
	t.Helper()
	pixelColumn := stream(t, "seed-pixel-x")
	pixelRow := stream(t, "seed-pixel-y")
	x := int(noise.Hash(pixelColumn, k) * float64(view.wide))
	y := int(noise.Hash(pixelRow, k) * float64(view.high))
	ray := view.look(x, y)
	far := nearest + spread*noise.Hash(stream(t, "seed-far"), k)
	first := noise.Hash(stream(t, "seed-size-first"), k)
	second := noise.Hash(stream(t, "seed-size-second"), k)
	return ball{centre: ray.At.Add(ray.Dir.Scale(far)),
		radius: far * ballShare * first * second, far: far}, image.Pt(x, y)
}

// misses is how many of the balls aimed along rays are met by the ray
// of a pixel and said no to by the sight of its section.
func misses(t *testing.T, view eye, side int, nearest, spread float64) (
	count int, first string) {
	t.Helper()
	across := (view.wide + side - one) / side
	boxes := section.Split(view.wide, view.high, side)
	sights := make([]section.Sight, len(boxes))
	for at, box := range boxes {
		sights[at] = section.SightOf(view.look, box)
	}
	for k := range int64(balls) {
		around, pixel := aim(t, view, k, nearest, spread)
		sight := sights[pixel.Y/side*across+pixel.X/side]
		if !sight.Meets(around.centre, around.radius) {
			if count == nothing {
				first = pixelNote(pixel, around)
			}
			count++
		}
	}
	return count, first
}

// pixelNote says which pixel's ray met which ball.
func pixelNote(pixel image.Point, around ball) string {
	return fmt.Sprintf("pixel %v at distance %g radius %g", pixel,
		around.far, around.radius)
}

// sightsHold fails the test for each side of a section whose sights say
// no to a ball that a ray of the section meets.
func sightsHold(t *testing.T, name string, view eye, nearest,
	spread float64) {
	t.Helper()
	for _, side := range sidesTried() {
		count, first := misses(t, view, side, nearest, spread)
		if count != nothing {
			t.Errorf("%s, sections %d pixels a side: %d of %d balls a "+
				"ray meets were refused by the sight, the first %s", name,
				side, count, balls, first)
		}
	}
}

// the sights of sections never say no to a ball that a ray meets, through
// a lens whose rays start across a small aperture and not from one point.
func TestASightMeetsWhatRaysFromAnApertureMeet(t *testing.T) {
	far := eye{lens: thinLens(t, farFocal, farStop, farFocus),
		wide: trialWide, high: trialHigh}
	sightsHold(t, "a far lens", far, farNearest, farSpread)
}

// the same through a lens whose aperture is wide beside what it sees, and
// balls from near its glass out to three units.
func TestASightMeetsWhatRaysFromAWideApertureMeet(t *testing.T) {
	close := eye{lens: thinLens(t, closeFocal, closeStop, closeFocus),
		wide: trialWide, high: trialHigh}
	sightsHold(t, "a wide aperture", close, closeNearest, closeSpread)
}

// the same of a pinhole that sees 150 degrees and one that sees 179, and
// a 15 mm lens: the sight's cone opened by a twentieth is wide enough.
func TestASightMeetsWhatRaysOfAWideLensMeet(t *testing.T) {
	for _, wide := range []float64{degrees150, degrees179} {
		view := eye{lens: film.Pinhole{Wide: wide * maths.Degree},
			wide: trialWide, high: trialHigh}
		sightsHold(t, "a pinhole", view, farNearest, farSpread)
	}
	short := eye{lens: film.Lens{Focal: shortFocal, Gauge: film.FullFrame},
		wide: trialWide, high: trialHigh}
	sightsHold(t, "a short lens", short, farNearest, farSpread)
}

// oddView is a frame of 100 by 37 pixels seen through a pinhole of
// nearly a half turn, with a pixel of a shape.
func oddView(shape float64) eye {
	return eye{lens: film.Pinhole{Wide: halfTurn * maths.Degree,
		Pixel: shape}, wide: oddWide, high: oddHigh}
}

// the sight never says no to a ball that a ray meets, in a frame whose
// sections do not divide it, through the widest pinhole; and with a
// pixel three times as wide as tall.
func TestASightMeetsWhatRaysOfANearlyHalfTurnLensMeet(t *testing.T) {
	for _, shape := range []float64{nothing, squashed} {
		count, first := misses(t, oddView(shape), oddSide, farNearest,
			farSpread)
		if count != nothing {
			t.Errorf("pixel shape %g: %d of %d balls a ray meets were "+
				"refused by the sight, the first %s", shape, count, balls,
				first)
		}
	}
}
