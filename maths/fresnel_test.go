package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Fresnel's share for unpolarised light, worked by hand at an index of
// 1.33, the water the trace tests use. fresnel.csv has each case.
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
func TestFresnelSharesByHand(test *testing.T) {
	rows := read(test, fresnelFile)
	if len(rows) == nought {
		test.Fatal("there are no worked shares")
	}
	for _, row := range rows {
		got := maths.Fresnel(float(test, row, "cosine"), float(test, row, "index"))
		want, tolerance := float(test, row, "want"), float(test, row, "tolerance")
		if !(math.Abs(got-want) <= tolerance) {
			test.Errorf("%s: share %v, want %v", row["name"], got, want)
		}
	}
}

// Fresnel holds a cosine outside zero to one at the nearest end of it.
func TestFresnelClampsTheCosine(test *testing.T) {
	water := picked(test, "water-index")
	past, below := picked(test, "cosine-past-one"), picked(test, "cosine-below-zero")
	if got, want := maths.Fresnel(past, water), maths.Fresnel(unity, water); got != want {
		test.Errorf("a cosine of %v gives %v, want %v", past, got, want)
	}
	if got := maths.Fresnel(below, water); got != unity {
		test.Errorf("a cosine of %v gives %v, want 1", below, got)
	}
}

// An index of zero or less, no index, and an infinite index are no medium,
// and the share into one is no number, at any cosine.
func TestFresnelIntoNoMediumIsNoNumber(test *testing.T) {
	rows := read(test, refusedIndexesFile)
	if len(rows) == nought {
		test.Fatal("there are no indexes to refuse")
	}
	for _, row := range rows {
		cosine, index := float(test, row, "cosine"), float(test, row, "index")
		if got := maths.Fresnel(cosine, index); !math.IsNaN(got) {
			test.Errorf("%s: share %v, want no number", row["name"], got)
		}
	}
}

// anti-test: at the cosines the refused indexes are asked at, a medium that
// is one is given a share from nought to one, so the refusal is not passed
// by a Fresnel that is no number everywhere.
func TestFresnelIntoAMediumIsAShare(test *testing.T) {
	water := picked(test, "water-index")
	for _, row := range read(test, refusedIndexesFile) {
		got := maths.Fresnel(float(test, row, "cosine"), water)
		if math.IsNaN(got) || got < nought || got > unity {
			test.Errorf("%s: into the water the share is %v", row["name"], got)
		}
	}
}
