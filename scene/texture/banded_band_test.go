package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

var (
	red3   = maths.RGB(0.9, 0.2, 0.1)
	green3 = maths.RGB(0.2, 0.7, 0.3)
	blue3  = maths.RGB(0.1, 0.3, 0.9)
)

// A latitude is a place along the paints, evenly spaced from the south
// pole to the north, and the colour is the two paints either side of it
// blended by how far along. past a pole it is the pole's paint.
//
// with three paints the places are -1, 0 and 1, so -0.75 is a quarter of
// the way from the first to the second, and 0.25 a quarter of the way
// from the second to the third.
func TestBandIsTheTwoNearestPaintsBlended(t *testing.T) {
	b := Banded{Paints: []maths.Light{red3, green3, blue3}}
	mix := func(x, y maths.Light, share float64) maths.Light {
		return x.Scale(1 - share).Add(y.Scale(share))
	}
	for _, c := range []struct {
		lat  float64
		want maths.Light
	}{
		{-1.5, red3}, {-1, red3}, {-0.75, mix(red3, green3, 0.25)},
		{-0.5, mix(red3, green3, 0.5)}, {0, green3}, {0.25, mix(green3, blue3, 0.25)},
		{0.5, mix(green3, blue3, 0.5)}, {1, blue3}, {1.5, blue3},
	} {
		if got := b.band(c.lat); !near(got, c.want) {
			t.Errorf("latitude %v: %v, want %v", c.lat, got, c.want)
		}
	}
}

// With two paints the places are -1 and 1, and 0 is half and half.
// with one the paint is the paint at every latitude.
func TestBandOfTwoPaintsAndOfOne(t *testing.T) {
	two := Banded{Paints: []maths.Light{red3, blue3}}
	half := red3.Scale(0.5).Add(blue3.Scale(0.5))
	for lat, want := range map[float64]maths.Light{-1: red3, 0: half, 1: blue3, -2: red3, 2: blue3} {
		if got := two.band(lat); !near(got, want) {
			t.Errorf("two paints at %v: %v, want %v", lat, got, want)
		}
	}
	one := Banded{Paints: []maths.Light{green3}}
	for _, lat := range []float64{-3, -1, 0, 0.3, 1, 3} {
		if got := one.band(lat); got != green3 {
			t.Errorf("one paint at %v: %v", lat, got)
		}
	}
}

// The roil's size is a share of the body: zero is a tenth, below it is a tenth
// too, and anything above is as it is.
func TestBandedGrainOfNoneIsATenth(t *testing.T) {
	for _, c := range []struct{ grain, want float64 }{
		{0, 0.1}, {-3, 0.1}, {0.25, 0.25}, {1, 1}, {2.5, 2.5},
	} {
		if got := (Banded{Grain: c.grain}).grain(); got != c.want {
			t.Errorf("a grain of %v is %v, want %v", c.grain, got, c.want)
		}
	}
}

// North is the axis made one unit long, and up with no axis.
func TestBandedNorthIsTheAxisOrUp(t *testing.T) {
	for _, c := range []struct{ axis, want maths.Vec }{
		{maths.Vec{}, maths.Up}, {maths.V(0, 2, 0), maths.Up},
		{maths.V(0, 0, -3), maths.V(0, 0, -1)}, {maths.V(3, 0, 4), maths.V(0.6, 0, 0.8)},
	} {
		if got := (Banded{Axis: c.axis}).north(); !got.Near(c.want, 1e-12) {
			t.Errorf("an axis of %v has north %v, want %v", c.axis, got, c.want)
		}
	}
}

// A point is a latitude, how far along north, and a longitude, the
// angle round north from a side square to it. for the default up that
// side is +X and the angle grows toward -Z.
//
// the worked points are (1, 0, 0) at 0, 0; (0, 0, 1) at 0, -pi/2; (0,
// 0.6, 0.8) at 0.6, -pi/2; and (3, 4, 12) at 4/13, -arctangent of 4.
func TestBandedWhereOnTheDefaultAxis(t *testing.T) {
	b := Banded{}
	for _, c := range []struct {
		p        maths.Vec
		lat, lon float64
	}{
		{maths.V(1, 0, 0), 0, 0},
		{maths.V(0, 0, 1), 0, -math.Pi / 2},
		{maths.V(0, 0.6, 0.8), 0.6, -math.Pi / 2},
		{maths.V(3, 4, 12).Scale(5), 4.0 / 13, -math.Atan(4)},
		{maths.V(0, 1, 0), 1, 0},
	} {
		lat, lon := b.where(c.p)
		if !within(lat, c.lat, 1e-12) || !within(lon, c.lon, 1e-12) {
			t.Errorf("%v is at %v, %v; want %v, %v", c.p, lat, lon, c.lat, c.lon)
		}
	}
}

// With north along +X the side is +Z, and the longitude is the angle
// toward -Y: (0, 0, 1) is longitude 0, (0, 1, 0) is -pi/2.
func TestBandedWhereOnAnAxisAlongX(t *testing.T) {
	b := Banded{Axis: maths.XAxis}
	for _, c := range []struct {
		p        maths.Vec
		lat, lon float64
	}{
		{maths.V(0, 0, 1), 0, 0},
		{maths.V(0, 1, 0), 0, -math.Pi / 2},
		{maths.V(0, -1, 0), 0, math.Pi / 2},
		{maths.V(1, 0, 0), 1, 0},
		{maths.V(-0.6, 0, 0.8), -0.6, 0},
	} {
		lat, lon := b.where(c.p)
		if !within(lat, c.lat, 1e-12) || !within(lon, c.lon, 1e-12) {
			t.Errorf("%v is at %v, %v; want %v, %v", c.p, lat, lon, c.lat, c.lon)
		}
	}
}

// The side to measure the longitude from is +X, or +Z when north is
// past 0.9 along or against X, and not at 0.9 itself.
//
// an axis of (0.95, 0.31225, 0) is past it, so the side is +Z and (0, 0,
// 1) is at longitude 0. one at exactly 0.9 along X keeps +X, so its
// side is (0.43589, -0.9, 0) and (0, 0, 1) is at longitude -pi/2.
func TestBandedWhereSwitchesSideNearTheX(t *testing.T) {
	past := Banded{Axis: maths.V(0.95, math.Sqrt(1-0.95*0.95), 0)}
	if _, lon := past.where(maths.V(0, 0, 1)); !within(lon, 0, 1e-12) {
		t.Errorf("past 0.9 along X, (0, 0, 1) is at longitude %v, not 0", lon)
	}
	against := Banded{Axis: maths.V(-0.95, math.Sqrt(1-0.95*0.95), 0)}
	if _, lon := against.where(maths.V(0, 0, 1)); !within(lon, 0, 1e-12) {
		t.Errorf("past 0.9 against X, (0, 0, 1) is at longitude %v, not 0", lon)
	}
	exact := Banded{Axis: maths.V(0.9, math.Sqrt(1-0.9*0.9), 0)}
	if exact.north().Dot(maths.XAxis) != 0.9 {
		t.Fatalf("the axis was meant to be exactly 0.9 along X and is %v", exact.north().Dot(maths.XAxis))
	}
	if lat, lon := exact.where(maths.V(0, 0, 1)); !within(lat, 0, 1e-12) || !within(lon, -math.Pi/2, 1e-12) {
		t.Errorf("at exactly 0.9 along X, (0, 0, 1) is at %v, %v, not 0, -pi/2", lat, lon)
	}
}
