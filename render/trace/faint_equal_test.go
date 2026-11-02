package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/thing"
)

// thousandth is faint, as a number written here, so that a test of it
// does not move with it.
const thousandth = 1e-3

// exactly finds an extinction for which a keep of one, taken down once a
// step by the factor that extinction gives, is exactly a thousandth after
// some number of steps from 12 to 44, and gives the extinction and the
// number.
//
// it looks on the doubles each side of the extinction that would give a
// thousandth to within rounding, by the arithmetic of this machine. when
// there is none it says so.
func exactly(factor func(ext float64) float64, step float64) (float64, int, bool) {
	for n := 12; n <= 44; n++ {
		base := -math.Log(thousandth) / (float64(n) * step)
		down, up := base, base
		for i := 0; i < 4000; i++ {
			for _, ext := range []float64{down, up} {
				keep := 1.0
				for s := 1; s <= n; s++ {
					keep *= factor(ext)
				}
				if keep == thousandth {
					return ext, n, true
				}
			}
			down, up = math.Nextafter(down, 0), math.Nextafter(up, math.Inf(1))
		}
	}
	return 0, 0, false
}

// a march stops when no channel keeps more than a thousandth, and not
// when a channel keeps exactly that. with one channel of a gas brought
// to exactly a thousandth, and the other two long gone, the picture is
// the sky by exactly a thousandth there, and not by the step more.
//
// the extinction of the channel is found by looking, since a keep is a
// product of rounded numbers and a thousandth only the right one
// reaches. the other two are of an extinction that takes them under at
// once. if this machine's arithmetic reaches none, the test is skipped.
func TestAMarchStopsAtAKeepOfExactlyAThousandth(t *testing.T) {
	w := thing.NewWorld()
	w.AddVolume(gasBlock(1, 1, tinted(redOptics, greenOptics, blueOptics), 1, maths.Vec{}, 1))
	cs := crossings(stage(w, 0).vs, along(0, 0), math.Inf(1))
	step := (cs[0].leave - cs[0].enter) / 48
	thick := 1.0 * step / 1.0
	ext, n, ok := exactly(func(e float64) float64 { return 1 - (1 - math.Exp(-e*thick)) }, thick)
	if !ok {
		t.Skip("this arithmetic reaches no extinction that makes a keep of exactly a thousandth")
	}
	for c, name := range []string{"red", "green", "blue"} {
		var exts [3]float64
		for i := range exts {
			exts[i] = 200
		}
		exts[c] = ext
		g := tinted(gas.Optics{Absorb: exts[0]}, gas.Optics{Absorb: exts[1]}, gas.Optics{Absorb: exts[2]})
		w := thing.NewWorld()
		w.Sky = even(maths.Grey(1))
		w.AddVolume(gasBlock(1, 1, g, 1, maths.Vec{}, 1))
		got := Trace(w, along(0, 0), 0)

		keep := [3]float64{1, 1, 1}
		for s := 0; s < 48; s++ {
			if !(keep[0] > thousandth || keep[1] > thousandth || keep[2] > thousandth) {
				break
			}
			for i := range keep {
				keep[i] *= 1 - (1 - math.Exp(-exts[i]*thick))
			}
		}
		if keep[c] != thousandth {
			t.Fatalf("%s: the walk here ended at %v after %d steps, not at a thousandth", name, keep[c], n)
		}
		if got.R != float32(keep[0]) || got.G != float32(keep[1]) || got.B != float32(keep[2]) {
			t.Errorf("%s: %+v, want %v", name, got, keep)
		}
	}
}

// the same in a band: a volume that thins the band by a gas's extinction
// there stops the march at exactly a thousandth, and the thing behind is
// seen by that, 5 by a thousandth.
func TestABandMarchStopsAtAKeepOfExactlyAThousandth(t *testing.T) {
	g := func(ext float64) gas.Gas {
		return gas.Gas{Name: "x", Ranges: map[radiation.Range]gas.Optics{radiation.XRay(): {Absorb: ext}}}
	}
	block1 := func(ext float64) *thing.World {
		w := thing.NewWorld()
		w.Add(radiantBall(-20, 1, radiates{gives: 5}))
		w.AddVolume(thing.Volume{Medium: gas.Of{Shape: block{k: 1, half: 1}, In: g(ext), Unit: 1},
			Motion: at(maths.V(0, 0, -10), 1)})
		return w
	}
	cs := crossings(stage(block1(1), 0).vs, out, math.Inf(1))
	step := (cs[0].leave - cs[0].enter) / 48
	ext, _, ok := exactly(func(e float64) float64 { return math.Exp(-e * step) }, step)
	if !ok {
		t.Skip("this arithmetic reaches no extinction that makes a keep of exactly a thousandth")
	}
	got := inBand(bandStage(block1(ext)), out, xray, passes)
	if want := float32(5 * thousandth); got.R != want {
		t.Errorf("behind a march that ends at a thousandth: %v, want %v", got.R, want)
	}
}
