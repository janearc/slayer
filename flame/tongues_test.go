package flame

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func TestAuthoredTonguesOpenTheContinuousCoreAndKeepOccupiedTipsHot(t *testing.T) {
	spec := Fire{Radius: 1, Height: 4, Absorption: 1, Hottest: measure.Kelvin(1900), Edge: measure.Kelvin(900), Seed: 5}
	legacy, err := NewBody(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	spec.Intermittency = .08
	mean, err := NewBody(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	spec.HotTongues = true
	hot, err := NewBody(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	gaps, warmer, occupied := 0, 0, 0
	for step := range 2000 {
		at := time.Duration(step) * 37 * time.Millisecond
		a := legacy.Tick(at).(*Body)
		b := mean.Tick(at).(*Body)
		c := hot.Tick(at).(*Body)
		for _, y := range []float64{1.4, 1.5, 1.59, 2.8} {
			p := maths.V(.02*float64(step%45)-.45, y, .015*float64((step/45)%30)-.2)
			d := b.Density(p)
			if c.Density(p) != d {
				t.Fatal("temperature choice changed opacity")
			}
			if y < 1.6 && a.Density(p) > 0 && d == 0 {
				gaps++
			}
			if d > 0 {
				occupied++
				if c.Heat(p).InKelvin() > b.Heat(p).InKelvin()+10 {
					warmer++
				}
			}
		}
	}
	if gaps < 10 || warmer < 10 || occupied < 10 {
		t.Fatalf("no useful tongues: gaps %d warmer %d occupied %d", gaps, warmer, occupied)
	}
	t.Logf("tongues: %d gaps, %d warmer of %d occupied samples", gaps, warmer, occupied)
	// These are authored fixture floors, not measured flame statistics.
	// A handful of altered samples must not certify the visual controls.
	if float64(gaps)/8000 < .02 || float64(warmer)/float64(occupied) < .1 {
		t.Fatal("tongue controls changed too little of the sampled body")
	}
	for _, v := range []float64{-1, 1, 2, math.NaN(), math.Inf(1)} {
		spec.Intermittency = v
		if _, err := NewBody(spec, 1); err == nil {
			t.Fatalf("bad intermittency %v accepted", v)
		}
	}
}

func TestUnspecifiedTonguesKeepTheLegacyLaw(t *testing.T) {
	spec := Fire{Radius: 1, Height: 4, Absorption: 1, Hottest: measure.Kelvin(1700), Edge: measure.Kelvin(900), Seed: 1}
	a, err := NewBody(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	spec.Intermittency = .4
	b, err := NewBody(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	for step := range 100 {
		p := maths.V(.01*float64(step%17), .039*float64(step), 0)
		if a.Density(p) != b.Density(p) || a.Heat(p) != b.Heat(p) {
			t.Fatal("legacy defaults changed")
		}
	}
}
