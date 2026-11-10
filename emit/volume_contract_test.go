package emit

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func TestVolumeAcceptsSmallAndDarkSourcesAndRejectsEachInvalidChannel(t *testing.T) {
	for _, power := range []maths.Light{{}, maths.RGB(.25, 0, 0), maths.RGB(0, .25, 0), maths.RGB(0, 0, .25)} {
		if _, err := NewVolume(Lit{Strength: power}, measure.Metres(.25)); err != nil {
			t.Fatalf("valid small/dark source: %v", err)
		}
	}
	for _, power := range []maths.Light{maths.RGB(-.1, 0, 0), maths.RGB(0, -.1, 0), maths.RGB(0, 0, -.1)} {
		if _, err := NewVolume(Lit{Strength: power}, measure.Metres(.25)); err == nil || !strings.Contains(err.Error(), "negative source strength") {
			t.Fatalf("negative channel: %v", err)
		}
	}
	for _, point := range []Lit{{At: maths.V(math.Inf(-1), 0, 0)}, {Strength: maths.Grey(math.Inf(-1))}} {
		if _, err := NewVolume(point, measure.Metres(.25)); err == nil || !strings.Contains(err.Error(), "nonfinite volume source") {
			t.Fatalf("negative infinity: %v", err)
		}
	}
}

func TestVolumeReturnsTheCentreRayAndIndependentSpectralFluence(t *testing.T) {
	point := Lit{At: maths.V(1, 2, 3), Strength: maths.RGB(.25, 0, .5),
		Spectrum: bands.Lines(map[float64]float64{550: .25})}
	source, err := NewVolume(point, measure.Metres(.5))
	if err != nil {
		t.Fatal(err)
	}
	axis := maths.V(2.0/7, 3.0/7, 6.0/7)
	for _, distance := range []float64{0, .125, .375, 1.5, 7.5} {
		at := point.At.Sub(axis.Scale(distance))
		direction, away, got := source.Light(at, maths.YAxis)
		wantDirection := axis
		if distance == 0 {
			wantDirection = maths.YAxis
		}
		if !(direction.Sub(wantDirection).Len() <= 1e-12) || !(math.Abs(away-distance) <= 1e-12) {
			t.Fatalf("centre ray at %g: direction %v distance %g", distance, direction, away)
		}
		want := rayIntegral(distance/.5) / (.5 * .5)
		if !(math.Abs(got.R-.25*want) <= 1e-6*want) || got.G != 0 || !(math.Abs(got.B-.5*want) <= 1e-6*want) {
			t.Fatalf("RGB fluence at %g: %v, integral %g", distance, got, want)
		}
		if landed := source.Lands(at, maths.Band{Nanometres: 550}); !(math.Abs(landed-.25*want) <= 1e-6*want) {
			t.Fatalf("spectral fluence at %g: %g, integral %g", distance, landed, want)
		}
	}
}

func TestPolarSourceExcludesItsPhysicalInterior(t *testing.T) {
	polar := upright()
	polar.Beyond = 2
	clock := time.Duration(0)
	beams := polar.Beams(&clock)
	for _, distance := range []float64{0, 1, 2} {
		way, away, light := beams.Light(maths.V(0, distance, 0), maths.YAxis)
		if way != (maths.Vec{}) || away != 0 || light != (maths.Light{}) {
			t.Fatalf("polar light inside source: %v %g %v", way, away, light)
		}
	}
}
