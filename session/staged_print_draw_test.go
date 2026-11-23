//go:build draw

package session

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/render/frame"
)

// printRows is a film's print rows, as the tests below say them.
const printRows = "    exposure: 100000\n    print: {knee: 0.3, top: 4, " +
	"source: eye}\n"

// fixtureExposure is the fixture's film exposure row.
const fixtureExposure = "    exposure: 100000\n"

// dimGrey is the grey of the picture a stand-in framer draws, a fifth of
// the meter's grey, so the meter asks for five times the light.
const dimGrey = 0.036

// dimmest is a framer that draws one flat grey and nothing else.
type dimmest struct{}

// light draws a flat grey picture, whatever it is asked.
func (dimmest) light(_ time.Duration, given asked) (drawnLight, error) {
	drawn := &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{
		{R: dimGrey, G: dimGrey, B: dimGrey}}}
	return drawnLight{drawn: drawn, given: given}, nil
}

// TestAStagedSetIsDevelopedThroughItsFilmsPrint says that a set whose
// light the session stages is developed with the knee and top its film says,
// and that its exposure stays with the lamps, not the tone.
func TestAStagedSetIsDevelopedThroughItsFilmsPrint(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, fixtureExposure, printRows)
	got, err := filmed(readSet(t, given.set), given)
	if err != nil {
		t.Fatal(err)
	}
	tone := got.tone()
	if tone.Knee != 0.3 || tone.Top != 4 || tone.Exposure != nothing {
		t.Errorf("the tone is %+v, want knee 0.3, top 4, no exposure", tone)
	}
}

// TestAStagedSetWithNoPrintRowsIsDevelopedAsHillsideDoes is the anti-test
// of the above: a film that says no print leaves hillside's own tone.
func TestAStagedSetWithNoPrintRowsIsDevelopedAsHillsideDoes(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	got, err := filmed(readSet(t, given.set), given)
	if err != nil {
		t.Fatal(err)
	}
	if tone := got.tone(); tone != developing {
		t.Errorf("the tone is %+v, want hillside's %+v", tone, developing)
	}
}

// TestTheMeterKeepsTheStagedPrintsKneeAndTop says that the picture meter
// sets the tone's exposure and leaves the knee and top the film said.
func TestTheMeterKeepsTheStagedPrintsKneeAndTop(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, fixtureExposure, printRows)
	got, err := filmed(readSet(t, given.set), given)
	if err != nil {
		t.Fatal(err)
	}
	session := &Session{source: dimmest{}, given: got, tone: got.tone(),
		times: []time.Duration{nothing}}
	if err = session.pictureMetered(); err != nil {
		t.Fatal(err)
	}
	want := meterGrey / dimGrey
	if math.Abs(session.tone.Exposure-want) > 1e-6 ||
		session.tone.Knee != 0.3 || session.tone.Top != 4 {
		t.Errorf("the tone is %+v, want exposure %v, knee 0.3, top 4",
			session.tone, want)
	}
}
