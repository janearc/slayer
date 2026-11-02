package trace

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// asked is one call of an exported function with something wrong in it, and a
// word its error must say. rays of zero with band unset is Render; with a band,
// RenderIn; with rays, Sampled.
//
// eye is where the picture is seen from, and nil is overTheScene.
type asked struct {
	name          string
	world         *thing.World
	eye           *maths.Subspace
	width, height int
	rays          int
	sampled       bool
	band          maths.Band
	says          string
}

// call makes the call an asking describes.
func (ask asked) call() (*frame.Frame, error) {
	lens, eye := film.Pinhole{}, overTheScene
	if ask.eye != nil {
		eye = *ask.eye
	}
	switch {
	case ask.sampled:
		return Sampled(ask.world, eye, lens, 0, ask.width, ask.height,
			ask.rays)
	case ask.band != (maths.Band{}):
		return RenderIn(ask.world, eye, lens, 0, ask.width, ask.height,
			ask.band)
	}
	return Render(ask.world, eye, lens, 0, ask.width, ask.height)
}

// wrongly is every way the exported functions are asked wrongly: no world, no
// size, a size less than zero, no rays and too many, an eye that is nowhere,
// and a band that is no wavelength.
func wrongly() []asked {
	world := smoking()
	nowhere := lookingAt(maths.V(math.NaN(), 2, 6), centre)
	unscaled := overTheScene
	unscaled.Scale = math.Inf(1)
	notANumber := maths.Band{Nanometres: math.NaN()}
	return []asked{
		{name: "Render from an eye at no place", world: world,
			eye: &nowhere, width: 4, height: 4, says: "an eye"},
		{name: "Sampled from an eye of no size", world: world,
			eye: &unscaled, width: 4, height: 4, rays: 2,
			sampled: true, says: "an eye"},
		{name: "RenderIn at a band that is not a number", world: world,
			width: 4, height: 4, band: notANumber, says: "a band"},
		{name: "RenderIn at a band below nought", world: world,
			width: 4, height: 4, band: maths.Band{Nanometres: -5},
			says: "-5 nanometres"},
		{name: "Render of no world", width: 4, height: 4,
			says: "no world"},
		{name: "Render no pixels wide", world: world, height: 4,
			says: "0 by 4"},
		{name: "Render under nought tall", world: world, width: 4,
			height: -1, says: "4 by -1"},
		{name: "Render too big", world: world, width: 1 << 20,
			height: 1 << 20, says: "more than"},
		{name: "RenderIn of no world", width: 4, height: 4,
			band: gammaBand, says: "no world"},
		{name: "RenderIn under nought wide", world: world, width: -3,
			height: 4, band: gammaBand, says: "-3 by 4"},
		{name: "Sampled with no rays", world: world, width: 4,
			height: 4, sampled: true, says: "asks for 0"},
		{name: "Sampled no pixels tall", world: world, width: 4,
			rays: 2, sampled: true, says: "4 by 0"},
		{name: "Sampled with too many rays", world: world, width: 64,
			height: 64, rays: 100000, sampled: true,
			says: "rays each way"},
	}
}

// Asked wrongly, an exported function says what is wrong and draws
// nothing. None panics, and none hands back an empty frame in silence.
func TestAskedWronglyTheTracerSaysSo(t *testing.T) {
	for _, ask := range wrongly() {
		picture, err := ask.call()
		if err == nil || !strings.Contains(err.Error(), ask.says) {
			t.Errorf("%s: the error is %v, and should say %q",
				ask.name, err, ask.says)
		}
		if picture != nil {
			t.Errorf("%s: a frame came with the error", ask.name)
		}
	}
}

// A picture may have mostPixels rays and not one more: the most is
// taken, and a pixel over, or a second ray each way through as many,
// is refused. Nothing is drawn to find out.
func TestTheMostPixelsIsTakenAndOneMoreIsNot(t *testing.T) {
	world, lens := smoking(), film.Pinhole{}
	view := func(width, height, rays int) error {
		_, err := newView(world, overTheScene, lens, 0, width, height,
			rays)
		return err
	}
	if err := view(mostPixels, 1, 1); err != nil {
		t.Errorf("a picture of mostPixels pixels is refused: %v", err)
	}
	if err := view(mostPixels/4, 1, 2); err != nil {
		t.Errorf("mostPixels rays in four to a pixel are refused: %v",
			err)
	}
	if err := view(mostPixels+1, 1, 1); err == nil {
		t.Error("a picture of one pixel more than mostPixels is taken")
	}
	if err := view(mostPixels/4+1, 1, 2); err == nil {
		t.Error("one pixel more than a quarter of mostPixels, at two " +
			"rays each way, is taken")
	}
}

// A view's fine view has a pixel for each of its rays.
func TestAViewsFineViewHasAPixelARay(t *testing.T) {
	seen, err := newView(smoking(), overTheScene, film.Pinhole{}, 0, 8,
		6, 3)
	if err != nil {
		t.Fatal(err)
	}
	fine := seen.fine()
	if fine.width != 24 || fine.height != 18 || fine.rays != 1 {
		t.Errorf("the fine view is %d by %d at %d rays", fine.width,
			fine.height, fine.rays)
	}
}
