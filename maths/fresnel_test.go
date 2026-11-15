package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Fresnel's share for unpolarised light, worked by hand at an index of
// 1.33, the water the trace tests use.
//
// Square on, ((n - 1) / (n + 1))^2 = (0.33 / 2.33)^2 = 0.141631^2 =
// 0.0200593.
//
// From the air at 45 degrees, the cosine of incidence i is 0.707107, the
// sine of the refracted angle is 0.707107 / 1.33 = 0.531659, and the
// cosine t of it is 0.846958:
//
//	rs = (i - n t) / (i + n t) = (0.707107 - 1.126455) / (0.707107 + 1.126455)
//	   = -0.419348 / 1.833562 = -0.228707
//	rp = (n i - t) / (n i + t) = (0.940452 - 0.846958) / (0.940452 + 0.846958)
//	   = 0.093494 / 1.787410 = 0.052307
//	R  = (rs^2 + rp^2) / 2 = (0.052307 + 0.002736) / 2 = 0.027521
//
// From the water at the refracted angle, cosine 0.846958, into the air,
// the boundary is the same one run backwards, and by reciprocity the share
// is the same 0.027521: the index going in is 1 / 1.33.
//
// The critical angle is asin(1 / 1.33) = 48.75 degrees. At 60 degrees,
// cosine 0.5, the sine is 0.866025, more than 1 / 1.33 = 0.751880, so
// nothing is transmitted and the share is exactly one.
func TestFresnelSharesByHand(t *testing.T) {
	const n = 1.33
	for _, c := range []struct {
		what       string
		cos, index float64
		want, tol  float64
	}{
		{"square on from the air", 1, n, 0.0200593, 1e-7},
		{"45 degrees from the air", 0.707107, n, 0.027521, 2e-6},
		{"the same boundary from the water", 0.846958, 1 / n, 0.027521, 2e-6},
		{"past the critical angle", 0.5, 1 / n, 1, 0},
		{"grazing from the air", 0, n, 1, 0},
	} {
		got := maths.Fresnel(c.cos, c.index)
		if !(math.Abs(got-c.want) <= c.tol) {
			t.Errorf("%s: share %v, want %v", c.what, got, c.want)
		}
	}
}

// Fresnel holds a cosine outside zero to one at the nearest end of it.
func TestFresnelClampsTheCosine(t *testing.T) {
	if got, want := maths.Fresnel(1.5, 1.33), maths.Fresnel(1, 1.33); got != want {
		t.Errorf("a cosine of 1.5 gives %v, want %v", got, want)
	}
	if got := maths.Fresnel(-0.5, 1.33); got != 1 {
		t.Errorf("a cosine of -0.5 gives %v, want 1", got)
	}
}
