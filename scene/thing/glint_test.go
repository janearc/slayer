package thing

import (
	"math"
	"testing"
)

// returned is how much of the light landing on a surface from every
// way its highlights send toward an eye looking at it from cosine
// viewed, for a GGX width alpha, before Fresnel's share: the lobe,
// which holds each lamp's cosine already, summed over the hemisphere by
// the midpoint rule.
func returned(viewed, alpha float64) float64 {
	eye := [3]float64{math.Sqrt(1 - viewed*viewed), viewed, 0}
	const rings, turns = 1200, 600
	sum := 0.0
	for i := range rings {
		tilt := (float64(i) + 0.5) / rings * math.Pi / 2
		for j := range turns {
			turn := (float64(j) + 0.5) / turns * 2 * math.Pi
			lamp := [3]float64{math.Sin(tilt) * math.Cos(turn), math.Cos(tilt),
				math.Sin(tilt) * math.Sin(turn)}
			half := [3]float64{lamp[0] + eye[0], lamp[1] + eye[1], lamp[2] + eye[2]}
			length := math.Sqrt(half[0]*half[0] + half[1]*half[1] + half[2]*half[2])
			facing := half[1] / length
			area := math.Sin(tilt) * (math.Pi / 2 / rings) * (2 * math.Pi / turns)
			sum += Microfacet(facing, lamp[1], viewed, alpha) * area
		}
	}
	return sum
}

// A rough surface never sends back more than lands on it: its lobe,
// summed over every way a lamp could stand, is at most one. Smith's
// shadowing takes more the rougher it is and the more slant the look,
// so a near-polished surface seen face on keeps nearly all of it.
func TestAHighlightNeverSendsBackMoreThanLands(t *testing.T) {
	for _, alpha := range []float64{0.1, 0.3, 0.6, 1} {
		for _, viewed := range []float64{1, 0.7, 0.3} {
			if got := returned(viewed, alpha); !(got <= 1.0005) {
				t.Errorf("width %v seen from cosine %v sends back %v", alpha,
					viewed, got)
			}
		}
	}
	if got := returned(1, 0.1); !(got >= 0.98) {
		t.Errorf("a near-polished surface seen face on sends back only %v", got)
	}
}

// The lobe is brightest where the lamp stands at the mirror way, and
// the smoother the surface the narrower and taller it is there.
func TestAHighlightPeaksAtTheMirrorWay(t *testing.T) {
	smooth, rough := 0.05, 0.5
	at := Microfacet(1, 0.8, 0.8, smooth)
	off := Microfacet(0.95, 0.8, 0.8, smooth)
	if !(at > off) {
		t.Errorf("at the mirror way %v, just off it %v", at, off)
	}
	if !(Microfacet(1, 0.8, 0.8, smooth) > Microfacet(1, 0.8, 0.8, rough)) {
		t.Error("the smooth peak is no taller than the rough")
	}
	if got := Microfacet(0, 0.8, 0.8, rough); got != 0 {
		t.Errorf("a half way edge on gives %v", got)
	}
	if got := Microfacet(1, 0.8, 0.8, 0); got != 0 {
		t.Errorf("a polished surface gives %v, a point no picture holds", got)
	}
}
