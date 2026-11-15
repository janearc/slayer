package film

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// aDisc is a sun drawn as a disc, two degrees across so the sums below
// are quick, a little off straight up.
func aDisc() Sun {
	return Sun{Toward: maths.V(0.1, 1, -0.2), Strength: maths.RGB(3, 2, 1),
		Size: 2 * math.Pi / 180, Disc: true}
}

// The disc's light, summed over the disc it fills, lights a surface
// facing it as Strength says: its radiance is Strength over the disc's
// solid angle, and the sum of that over the disc's solid angle is
// Strength again. Summed here by the midpoint rule over rings and turns
// about the sun's middle.
func TestASunsDiscLightsAsItsStrengthSays(t *testing.T) {
	sun := aDisc()
	middle := sun.Toward.Unit()
	across := middle.Cross(maths.V(1, 0, 0)).Unit()
	other := middle.Cross(across)
	const rings, turns = 400, 200
	limit := sun.Size / 2 * 1.02
	var sum maths.Light
	for i := range rings {
		out := (float64(i) + 0.5) / rings * limit
		for j := range turns {
			turn := (float64(j) + 0.5) / turns * 2 * math.Pi
			aside := across.Scale(math.Cos(turn)).Add(other.Scale(math.Sin(turn)))
			way := middle.Scale(math.Cos(out)).Add(aside.Scale(math.Sin(out)))
			area := math.Sin(out) * (limit / rings) * (2 * math.Pi / turns)
			sum = sum.Add(sun.Shown(way).Scale(area * way.Dot(middle)))
		}
	}
	want := sun.Strength
	if !(math.Abs(sum.R-want.R) <= 0.01*want.R && math.Abs(sum.B-want.B) <= 0.01*want.B) {
		t.Errorf("the disc lights with %v, its strength is %v", sum, want)
	}
	if got := sun.Shown(maths.V(1, 0, 0)); got != (maths.Light{}) {
		t.Errorf("far from the disc the sky shows %v of it", got)
	}
}

// The ways a shadow is sought along lie within the disc, eight of them,
// gathered about its middle; a sun not drawn as a disc has the one way.
func TestTheWaysAcrossADiscLieWithinIt(t *testing.T) {
	sun := aDisc()
	ways := sun.Spread()
	if len(ways) != 8 {
		t.Fatalf("%d ways across the disc, want 8", len(ways))
	}
	middle := sun.Toward.Unit()
	var mean maths.Vec
	for i, way := range ways {
		apart := math.Acos(math.Min(1, way.Dot(middle)))
		if !(apart <= sun.Size/2) {
			t.Errorf("way %d is %v from the middle, past the edge %v", i,
				apart, sun.Size/2)
		}
		mean = mean.Add(way)
	}
	if off := math.Acos(math.Min(1, mean.Unit().Dot(middle))); !(off <= sun.Size/8) {
		t.Errorf("the ways gather %v from the middle", off)
	}
	point := sun
	point.Disc = false
	if got := point.Spread(); len(got) != 1 || got[0] != middle {
		t.Errorf("a point sun's ways are %v", got)
	}
	if got := point.Shown(middle); got != (maths.Light{}) {
		t.Errorf("a point sun shows %v in the sky", got)
	}
}

// Discs gathers what every sun drawn as a disc shows, and nothing from
// any other lamp: Disced keeps the suns drawn as discs, in order, and no
// sun drawn as a point, and no other kind of lamp.
func TestDiscsSeeOnlySunsDrawnAsDiscs(t *testing.T) {
	sun := aDisc()
	point := Sun{Toward: sun.Toward, Strength: maths.Grey(9),
		Size: sun.Size}
	second := aDisc()
	second.Toward = maths.V(0, 0, 1)
	suns := Disced([]Emitter{sun, point, Radar{}, second})
	if len(suns) != 2 || suns[0] != sun || suns[1] != second {
		t.Fatalf("the suns drawn as discs are %v", suns)
	}
	if got, want := Discs(suns, sun.Toward), sun.Shown(sun.Toward); got != want {
		t.Errorf("toward the sun, the discs show %v, want %v", got, want)
	}
}
