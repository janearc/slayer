package gas

import (
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// oldOptics is optics as it was written: the range found before the
// rule is asked whether it has one.
func oldOptics(g Gas, band maths.Band) (absorb, scatter float64) {
	nm := band.Nanometres
	if nm == 0 {
		nm = 550
	}
	r := thing.Range(maths.Band{Nanometres: nm})
	near := (r == radiation.Ultraviolet() || r == radiation.Infrared()) &&
		nm > 300 && nm < 1200
	if g.Visible != nil && (r == thing.Visible() || near) {
		return g.Visible(nm)
	}
	o, ok := thing.Lookup(g.Ranges, maths.Band{Nanometres: nm})
	if !ok {
		return math.Inf(1), 0
	}
	return o.Absorb, o.Scatter
}

// optics answers the same bits as it did, for every gas the package makes,
// one with no rule, and one with no table, over the special wavelengths and
// seeded random ones across the spectrum.
func TestOpticsIsTheOldDerivationBitForBit(t *testing.T) {
	noRule := Haze()
	noRule.Visible = nil
	noTable := Air()
	noTable.Ranges = nil
	gases := map[string]Gas{"air": Air(), "haze": Haze(), "venus": Venus(),
		"mars": Mars(), "no rule": noRule, "no table": noTable, "empty": {}}
	nms := []float64{0, 550, 300, 1200, 380, 750, 1e-3, 1e20, -5, math.Inf(1),
		math.Inf(-1), math.NaN(), 299.9999999, 300.0000001, 1199.99999, 1200.00001}
	r := rand.New(rand.NewSource(20260910))
	for i := 0; i < 20000; i++ {
		wavelength := r.Float64() * 1500
		weight := math.Exp(r.Float64()*40 - 12)
		nms = append(nms, wavelength, weight)
	}
	for name, g := range gases {
		for _, nm := range nms {
			band := maths.Band{Nanometres: nm}
			a, s := g.optics(band)
			wa, ws := oldOptics(g, band)
			if math.Float64bits(a) != math.Float64bits(wa) ||
				math.Float64bits(s) != math.Float64bits(ws) {
				t.Fatalf("%s at %v: (%v, %v), was (%v, %v)", name, nm, a, s, wa, ws)
			}
		}
	}
}
