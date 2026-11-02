package gas

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// Air from its own numbers: blue is scattered four times as much as
// red, the limb's and the sky's slope; a column of eight kilometres
// stops gamma to nothing and passes the visible mostly; the ten micron
// window passes; ultraviolet at 250 nm is stopped within metres;
// radio comes through; the phase functions integrate to one over the
// sphere; and the spectrum for a unit says the same as beer's law.
func TestAirFromItsOwnNumbers(t *testing.T) {
	air := Air()
	blue, red := air.Scatter(maths.Band{Nanometres: 450}), air.Scatter(maths.Band{Nanometres: 650})
	if ratio := blue / red; math.Abs(ratio-math.Pow(650.0/450, 4)) > 1e-9 {
		t.Errorf("blue scatters %v times red; the fourth power says %v", ratio, math.Pow(650.0/450, 4))
	}
	column := 8000.0 // the air's scale height, the column's worth at sea level density
	// 5e-4 nm is gamma; 1e-3, the range's top, is x-ray, its top being
	// open. by hand, 3.3e-3 a metre over 8000 metres is 26.4 radiation
	// lengths (air's X0, 36.6 g/cm2 over 1.225e-3 g/cm3, is 299 m)
	if g := air.Transmit(maths.Band{Nanometres: 5e-4}, column); math.Abs(math.Log(g)+26.4) > 1e-9 {
		t.Errorf("gamma through the column is %g; twenty-six radiation lengths say e to the minus 26.4", g)
	}
	if v := air.Transmit(maths.Band{Nanometres: 550}, column); !(v > 0.85 && v < 0.95) {
		t.Errorf("the visible through the column is %v", v)
	}
	if w := air.Transmit(maths.Band{Nanometres: 10000}, column); !(w > 0.6) {
		t.Errorf("the ten micron window passes %v", w)
	}
	if uv := air.Transmit(maths.Band{Nanometres: 250}, 500); uv > 0.01 {
		t.Errorf("ultraviolet at 250 nm through 500 m is %v", uv)
	}
	if r := air.Transmit(maths.Band{Nanometres: 1e9}, column); r < 0.99 {
		t.Errorf("radio through the column is %v", r)
	}
	for _, g := range []Gas{air, Haze(), {Phase: -0.3}} {
		sum := 0.0
		n := 2000
		for i := 0; i < n; i++ {
			cos := -1 + (float64(i)+0.5)*2/float64(n)
			sum += g.PhaseAt(cos) * 2 * math.Pi * 2 / float64(n)
		}
		if math.Abs(sum-1) > 1e-3 {
			t.Errorf("the phase at asymmetry %v integrates to %v over the sphere", g.Phase, sum)
		}
	}
	if fwd, back := Haze().PhaseAt(1), Haze().PhaseAt(-1); !(fwd > 10*back) {
		t.Errorf("haze throws %v forward and %v back", fwd, back)
	}
	s := air.Spectrum(1000)
	if math.Abs(s.Passes[thing.Visible]-air.Transmit(maths.Band{Nanometres: 550}, 1000)) > 1e-12 {
		t.Errorf("the spectrum passes %v of the visible a kilometre; beer says %v", s.Passes[thing.Visible], air.Transmit(maths.Band{Nanometres: 550}, 1000))
	}
	if s.Passes[radiation.Gamma()] > 0.05 {
		t.Errorf("the spectrum passes %v of gamma a kilometre", s.Passes[radiation.Gamma()])
	}
	in := air.Inscatter(maths.Band{Nanometres: 450}, maths.Grey(1), 0)
	if in.R != blue*air.PhaseAt(0) {
		t.Errorf("inscatter is %v", in)
	}
	if n := len(air.Sources); n < 5 {
		t.Errorf("air names %d sources", n)
	}
}

// A laser through haze, seen side-on from a fixed distance, falls off
// along its beam as beer's law says, through Single with a uniform
// gas: at twice the distance the log of the brightness has fallen by
// the extinction times the distance, and off the beam there is nothing.
func TestALaserThroughHazeFallsByBeersLaw(t *testing.T) {
	haze := Haze().Per(1000) // a world unit is a kilometre
	laser := film.Laser{At: maths.V(0, 0, 0), Aim: maths.V(1, 0, 0), Width: 0.01, Strength: maths.RGB(0, 1, 0), Length: 100}
	lamps := []thing.Lamp{laser}
	through := Uniform(haze, 1)
	band := maths.Band{Nanometres: 532}
	see := func(x float64) float64 {
		// the eye looks from ten units off the beam, square to it
		return Single(haze, 1, band, lamps, maths.V(x, 0, 0), maths.V(0, 0, 1), through).G
	}
	a, b := see(2), see(4)
	if !(a > 0 && b > 0 && b < a) {
		t.Fatalf("the beam at two is %g and at four %g", a, b)
	}
	k := haze.Extinction(band)
	if got, want := math.Log(a/b), k*2; math.Abs(got-want) > 1e-9 {
		t.Errorf("over two units the beam fell by %g in the log; beer's law says %g", got, want)
	}
	if off := Single(haze, 1, band, lamps, maths.V(2, 1, 0), maths.V(0, 0, 1), through).G; off != 0 {
		t.Errorf("a unit off the beam there is %g", off)
	}
	if none := Single(haze, 0, band, lamps, maths.V(2, 0, 0), maths.V(0, 0, 1), through).G; none != 0 {
		t.Errorf("with no gas there is %g", none)
	}
	// haze throws forward: an eye past the beam's end looking back at
	// the source, which from the point lies along the beam, sees more
	// than one on the source's side looking down it
	forward := Single(haze, 1, band, lamps, maths.V(2, 0, 0), maths.V(1, 0, 0), through).G
	back := Single(haze, 1, band, lamps, maths.V(2, 0, 0), maths.V(-1, 0, 0), through).G
	if !(forward > 10*back) {
		t.Errorf("facing the source the haze shows %g, from behind it %g", forward, back)
	}
	// a medium of the gas keeps its shape and knows its optics per unit
	m := Of{Shape: medium.Puff{Radius: 2, Thick: 1}, In: Haze(), Unit: 1000}
	var o Optical = m
	if o.Gas().Scatter(band) != haze.Scatter(band) || m.Density(maths.Vec{}) != 1 {
		t.Errorf("the medium's gas scatters %g a unit, want %g", o.Gas().Scatter(band), haze.Scatter(band))
	}
	if _, isTimed := m.Tick(time.Second).(Of); !isTimed {
		t.Error("a ticked medium of gas is not one")
	}
}

// Venus's and mars's airs have the shape their sources give them: venus
// scatters about two hundred times what the earth does at 550 nm and
// is clear in radio, which is how its ground was mapped; mars's gas is
// nearly clear and its dust absorbs the blue more than the red, and its
// radio is clear. Every gas names its sources.
func TestVenusAndMarsHaveTheirOwnAirs(t *testing.T) {
	green := maths.Band{Nanometres: 550}
	radio := maths.Band{Nanometres: 1.26e8}
	venus, mars, earth := Venus(), Mars(), Air()
	if r := venus.Scatter(green) / earth.Scatter(green); r < 100 || r > 300 {
		t.Errorf("venus scatters %v times the earth", r)
	}
	if venus.Transmit(radio, 50e3) < 0.99 {
		t.Errorf("venus passes %v of radio through fifty kilometres", venus.Transmit(radio, 50e3))
	}
	if mars.Scatter(green) > 1e-4 || mars.Transmit(radio, 50e3) < 0.99 {
		t.Errorf("mars: scatter %v, radio %v", mars.Scatter(green), mars.Transmit(radio, 50e3))
	}
	blue, red := maths.Band{Nanometres: 465}, maths.Band{Nanometres: 610}
	if mars.Absorb(blue) <= mars.Absorb(red) {
		t.Errorf("mars's dust absorbs blue %v and red %v", mars.Absorb(blue), mars.Absorb(red))
	}
	for _, g := range []Gas{venus, mars} {
		if len(g.Sources) < 4 {
			t.Errorf("%s names %d sources", g.Name, len(g.Sources))
		}
	}
}
