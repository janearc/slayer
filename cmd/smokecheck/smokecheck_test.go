package main

import (
	"os"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/learn"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/compare"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/scene/smoke"
)

// Every smoke slayer has today is honest: the terminal's picture is the real
// one, coarser, and most of it is away from an edge and so was compared.
// A smoke with a field is judged at two real pixels a sextant third, the
// least its light allows.
//
// A verdict this test binary already put in smokecheck's cache is served, and not
// drawn again; SMOKES_FRESH=1 draws every smoke again. Build it with
// -trimpath and another tree's test binary, built alike, shares them.
func TestEverySmokeIsHonest(t *testing.T) {
	sh := openShelf(os.Getenv("SMOKES_FRESH") == "1")
	defer sh.close()
	if sh.off != "" {
		t.Log("cache off:", sh.off)
	}
	for _, s := range smoke.All() {
		w, eye := s.Scene()
		got, err := sh.get(s, scene{w, eye}, Fineness(w))
		if err != nil {
			t.Fatal(err)
		}
		got.entry.Close()
		r := got.report
		if !r.Honest() {
			t.Errorf("smoke %d, %s: %+v", s.Number, s.Name, r)
		}
		if r.Compared < r.Pixels/2 {
			t.Errorf("smoke %d compared only %d of %d", s.Number,
				r.Compared, r.Pixels)
		}
	}
	t.Logf("drew %d, served %d", sh.drew, sh.served)
}

// liar is a smoke drawn for the terminal with a pixel shape and an exposure
// that may be wrong, compared with the real picture.
func liar(s smoke.Smoke, pixel float64, exposure float64) *compare.Report {
	w, eye := s.Scene()
	lens := film.Pinhole{Wide: s.Wide, Pixel: pixel}
	f := trace.Render(w, eye, lens, 0, textW, textH)
	text := learn.Chopped(terminal.Chop(colour.Develop(f, exposure)))
	w, eye = s.Scene()
	_, real := Both(s, scene{w, eye}, 1)
	r := compare.Cells(real, text, across, down)
	return &r
}

// A terminal picture that is not the real one is caught: the wrong pixel
// shape, which draws an egg, and the wrong exposure, which draws the right
// shapes too bright.
func TestALyingTerminalIsCaught(t *testing.T) {
	one, _ := smoke.Get(1)
	if r := liar(one, 1, 1); r.Honest() {
		t.Errorf("square sextants passed: %+v", r)
	}
	if r := liar(one, 0.75, 1.2); r.Honest() {
		t.Errorf("a brighter terminal passed: %+v", r)
	}
	if r := liar(one, 0.75, 1); !r.Honest() {
		t.Errorf("the honest terminal failed: %+v", r)
	}
}

// A smoke with a field cannot be judged at one real pixel a sextant third:
// the verb refuses, and the plain smokes still take one.
func TestAFieldSmokeRefusesOneRay(t *testing.T) {
	one, _ := smoke.Get(1)
	if w, _ := one.Scene(); Fineness(w) != 1 {
		k := Fineness(w)
		t.Errorf("smoke one wants %d, not 1", k)
	}
	ring, ok := smoke.Get(36)
	if !ok {
		t.Fatal("slayer has no smoke 36")
	}
	if w, _ := ring.Scene(); Fineness(w) != 2 {
		k := Fineness(w)
		t.Errorf("the photon ring wants %d, not 2", k)
	}
	if _, err := run([]string{"36", "-k", "1", "-out", t.TempDir()}); err == nil {
		t.Error("smokecheck judged the photon ring at one real pixel")
	}
}
