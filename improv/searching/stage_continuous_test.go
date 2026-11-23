package searching

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

// Former shot boundaries must not reset distance, direction or velocity.
func TestHeardContinuousRetreat(t *testing.T) {
	previous := 0.0
	for at := 189.1; at < 218; at += 1.0 / 60 {
		d := fieldDistance(at)
		if d < previous || !bounded(d, 1, 62e6+1) {
			t.Fatal("backward or unbounded retreat", at, d)
		}
		previous = d
	}
	for _, at := range []float64{196.363636, 203.636364, 210.909091} {
		a, b, c := fieldDistance(at-.001), fieldDistance(at), fieldDistance(at+.001)
		if math.Abs((b-a)/(c-b)-1) > .01 {
			t.Fatal("camera velocity jumps", at)
		}
	}
	if math.Abs(fieldDistance(230)-fieldDistance(250)) > .001 {
		t.Fatal("held camera moves")
	}
}

// Isolated spacecraft use distinct body rolls but agree on optical aim.
func TestHeardSpacecraftIsolationAndAim(t *testing.T) {
	var axis maths.Vec
	for i, name := range []string{"hubble", "webb", "spitzer"} {
		p := &stagePlan{Kind: "space", Craft: name}
		s, err := newHeardStage(p, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(s.objects) != 1 {
			t.Fatal("unrelated spacecraft in isolated shot")
		}
		s.updateSpace(p, State{At: 189}, 1)
		got := s.objects[0].things[0].At(0).Orientation.Out(maths.YAxis)
		if i > 0 && got.Sub(axis).Len() > 1e-9 {
			t.Fatal("optical axes disagree")
		}
		axis = got
	}
}
