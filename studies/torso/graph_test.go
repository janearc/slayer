package main

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestConnectedDiffusionSeparatesNearbySkinSheets(t *testing.T) {
	pts := []maths.Vec{maths.V(0, 0, 0), maths.V(.002, 0, 0), maths.V(0, .002, 0), maths.V(0, 0, .0001), maths.V(.002, 0, .0001), maths.V(0, .002, .0001)}
	f := [][3]int{{0, 1, 2}, {3, 4, 5}}
	src := []maths.Light{maths.Grey(0), maths.Grey(1), maths.Grey(1), maths.Grey(.4), maths.Grey(.4), maths.Grey(.4)}
	d := diffusionSteps(pts, f, src, [3]float64{.0024, .00108, .00052})
	if !(d[0].X > d[0].Y && d[0].Y > d[0].Z) || d[0].X <= .1 {
		t.Fatal("red spread lost", d[0])
	}
	for _, v := range d[3:] {
		if math.Abs(v.X)+math.Abs(v.Y)+math.Abs(v.Z) > 1e-12 {
			t.Fatal("light crossed disconnected sheet", v)
		}
	}
}
