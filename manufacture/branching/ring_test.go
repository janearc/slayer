package branching

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

type ringDraws struct{ calls []int }

func (s *ringDraws) Draw(row, index, which int) float64 {
	s.calls = append(s.calls, which+10)
	return .25
}
func (s *ringDraws) Swing(row, index, which int) float64 {
	s.calls = append(s.calls, which)
	return .5
}

func TestRingPlacesAnUntwistedPartAndKeepsDrawOrder(t *testing.T) {
	ring, err := NewRing(4, 2, 0, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	draws := &ringDraws{}
	got := ring.Place(0, 0, draws)
	if got.Origin != maths.V(2, 0, 0) || got.Scale != 1 || got.Orientation.Y != maths.YAxis {
		t.Fatalf("placement: %+v", got)
	}
	if !reflect.DeepEqual(draws.calls, []int{0, 11, 2}) {
		t.Fatalf("draws were reordered: %v", draws.calls)
	}
}

func TestRingRefusesInvalidGeometry(t *testing.T) {
	for _, values := range []struct {
		count                              int
		radius, lean, stray, uneven, twist float64
	}{
		{0, 1, 0, 0, 0, 0}, {1, -1, 0, 0, 0, 0},
		{1, math.NaN(), 0, 0, 0, 0}, {1, 1, math.Inf(1), 0, 0, 0},
		{1, 1, 0, 1, 0, 0}, {1, 1, 0, 0, 1, 0},
		{1, 1, 0, 0, 0, 1.1},
	} {
		if _, err := NewRing(values.count, values.radius, values.lean, values.stray, values.uneven, values.twist); err == nil {
			t.Fatalf("accepted invalid ring: %+v", values)
		}
	}
	if _, err := NewRing(1, 0, 0, 0, 0, 0); err != nil {
		t.Fatalf("refused a part at the centre: %v", err)
	}
}
