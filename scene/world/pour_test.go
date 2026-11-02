package world

import (
	"math"
	"testing"
)

// poured is the heights after the sea is poured at a share.
func poured(h []float64, ocean float64) []float64 {
	w := &World{Terrain: Terrain{Ocean: ocean}, height: append([]float64(nil), h...)}
	w.pourSea()
	return w.height
}

// a hundred and one heights, the whole numbers 0 to 100 in a scrambled
// order (37 times the place, modulo the prime 101, takes each once), so
// the sorted height at place k is k, the last place is 100, and the top
// and bottom marks, 0.985 and 0.015 of the way, fall at places 98 and 1.
//
// a share of the sea of s puts the water line at place m = round(100 s),
// which leaves m tiles under it, counted here one by one.
//
// a height k is then (k - m) / (98 - m) above the line, if that is any,
// and below it (k - m) / (m - 1), so the tile at 98 is at one and the
// one at 1 is at minus one.
func TestTheSeaIsPouredToTheShareAndScaledToOne(t *testing.T) {
	h := make([]float64, 101)
	for i := range h {
		h[i] = float64(i * 37 % 101)
	}
	for _, c := range []struct {
		ocean float64
		m     int
	}{{0.6, 60}, {0.25, 25}, {0.5, 50}, {0.98, 98}, {0.01, 1}, {1, 100},
		{0, 0}, {1.5, 100}, {-0.3, 0}} {
		got := poured(h, c.ocean)
		under := 0
		for i, e := range got {
			k := h[i]
			want := k - float64(c.m)
			if top := 98 - float64(c.m); want > 0 && top > 0 {
				want /= top
			}
			if bottom := float64(c.m) - 1; want < 0 && bottom > 0 {
				want /= bottom
			}
			if !close12(e, want) {
				t.Fatalf("ocean %v: the tile of height %v is %v, want %v",
					c.ocean, k, e, want)
			}
			if e < 0 {
				under++
			}
		}
		if under != c.m {
			t.Errorf("ocean %v: %d tiles under water, want %d", c.ocean, under, c.m)
		}
	}
}

// the same on heights under one, where the top and bottom are less than
// one wide and are scaled all the same. 0 to 1 by hundredths, sea at
// 0.6: the water line is 0.6, the top mark 0.98 and the bottom 0.01, so
// 0.98 is at (0.98 - 0.6) / 0.38 = 1 and 0.01 at (0.01 - 0.6) / 0.59 = -1.
func TestTheSeaScalesHeightsUnderOneToo(t *testing.T) {
	h := make([]float64, 101)
	for i := range h {
		h[i] = float64(i*37%101) / 100
	}
	got := poured(h, 0.6)
	for i, e := range got {
		k := h[i]
		want := (k - 0.6) / 0.38
		if k < 0.6 {
			want = (k - 0.6) / 0.59
		}
		if math.Abs(e-want) > 1e-9 {
			t.Fatalf("the tile of height %v is %v, want %v", k, e, want)
		}
	}
}
