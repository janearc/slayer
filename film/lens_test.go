package film

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A lens is a sensor; the zero lens is a 50 on Super 35, about 28
// degrees; a 24 is wide, a 200 long, and the angle follows the gauge.
func TestLensAngles(t *testing.T) {
	var _ Sensor = Lens{}
	deg := func(l Lens) float64 { return l.Wide() / maths.Degree }
	if d := deg(Lens{}); math.Abs(d-27.96) > 0.05 {
		t.Errorf("the zero lens sees %.2f degrees", d)
	}
	if d := deg(Lens{Focal: 24}); math.Abs(d-54.8) > 0.2 {
		t.Errorf("a 24 sees %.2f degrees", d)
	}
	if d := deg(Lens{Focal: 200}); math.Abs(d-7.12) > 0.05 {
		t.Errorf("a 200 sees %.2f degrees", d)
	}
	if d := deg(Lens{Focal: 50, Gauge: FullFrame}); math.Abs(d-39.6) > 0.1 {
		t.Errorf("a 50 on full frame sees %.2f degrees", d)
	}
	if d := deg(Lens{Focal: 50, Gauge: IMAX}); !(d > 60) {
		t.Errorf("a 50 on imax sees only %.2f degrees", d)
	}
}

// A swath of forty kilometres from four hundred up is five and three
// quarter degrees, and the lens that sees it undoes to that.
func TestSwath(t *testing.T) {
	l := Swath(40, 400)
	if d := l.Wide() / maths.Degree; math.Abs(d-5.72) > 0.02 {
		t.Errorf("a forty kilometre swath from four hundred is %.2f degrees", d)
	}
}

// Hubble here is a lens of about 220 arcseconds across: the bare
// telescope's field, wider than WFC3's real 162, as Hubble's doc says.
func TestHubbleIsALensOf220Arcseconds(t *testing.T) {
	arcsec := Hubble().Wide() / maths.Degree * 3600
	if math.Abs(arcsec-220) > 2 {
		t.Errorf("hubble sees %.1f arcseconds", arcsec)
	}
}

// Focal undoes Wide, and a lens's rays are the pinhole's at its angle.
func TestFocalUndoesWide(t *testing.T) {
	for _, f := range []float64{18, 35, 85, 400} {
		l := Lens{Focal: f, Gauge: Academy}
		if got := Focal(l.Wide(), Academy); math.Abs(got-f) > 1e-9 {
			t.Errorf("a %v came back as %v", f, got)
		}
	}
	l := Lens{Focal: 85, Pixel: 0.75}
	p := Pinhole{Wide: l.Wide(), Pixel: 0.75}
	for _, xy := range [][2]int{{0, 0}, {17, 40}, {357, 149}} {
		a := l.Ray(xy[0], xy[1], 358, 150, maths.Band{})
		b := p.Ray(xy[0], xy[1], 358, 150, maths.Band{})
		if !a.Dir.Near(b.Dir, 1e-12) {
			t.Fatalf("pixel %v: lens %v, pinhole %v", xy, a.Dir, b.Dir)
		}
	}
}
