package snakes

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// sampleCoat chooses a point on a checked specimen's elliptical section.
func sampleCoat(p Profile, along, turn float64) maths.Light {
	width, height := section(p.Stations, along)
	angle := turn * math.Pi
	point := maths.V((along-.5)*p.Length, height*p.Length*math.Cos(angle), width*p.Length*math.Sin(angle))
	return (snakePaint{profile: p}).At(point)
}

func brightness(light maths.Light) float64 { return light.R + light.G + light.B }

// Outlined saddles vary in size from one marking to the next; a rigid
// sequence of identical transverse bands cannot pass this check.
func TestSaddlesHaveIrregularOutlinedDorsalBlotches(t *testing.T) {
	for _, name := range []string{"boa-constrictor-occidentalis", "python-bivittatus", "python-sebae"} {
		p := testProfile(t, name)
		t.Run(name, func(t *testing.T) {
			threshold := (brightness(p.Pattern.Base) + brightness(p.Pattern.Mark)) / 2
			fewest, most, outlined, displaced := 101, 0, 0, 0
			// A warped marking can cross a nominal cell boundary, so a
			// single world-space cell no longer measures a saddle's width.
			// Measure the underlying cell geometry without that displacement;
			// separately check coverage and outlines on the actual warped coat.
			unwarped := p
			unwarped.Pattern.Warp.Strength = 0
			var warpedCounts []int
			for cell := 5; cell < 13; cell++ {
				count, warped := 0, 0
				for sample := 0; sample < 100; sample++ {
					along := (float64(cell) + (float64(sample)+.5)/100) / p.Pattern.Repeat
					colour := brightness(sampleCoat(p, along, 0))
					straight := brightness(sampleCoat(unwarped, along, 0))
					if straight < threshold {
						count++
					}
					if math.Abs(colour-straight) > .01 {
						displaced++
					}
					if colour < threshold {
						warped++
					}
					if colour > brightness(p.Pattern.Base)+.01 {
						outlined++
					}
				}
				fewest, most = min(fewest, count), max(most, count)
				warpedCounts = append(warpedCounts, warped)
			}
			if outlined == 0 {
				t.Fatal("saddles have no pale outlines")
			}
			if p.Pattern.Warp.Strength > 0 && displaced == 0 {
				t.Fatal("coat warp did not displace any dorsal markings")
			}
			if fewest < 10 || most > 85 || most-fewest < 8 {
				t.Fatalf("saddle widths vary too little or cover incorrectly: %d..%d samples", fewest, most)
			}
			for i := 0; i+2 < len(warpedCounts); i++ {
				covered := warpedCounts[i] + warpedCounts[i+1] + warpedCounts[i+2]
				if covered < 30 || covered > 255 {
					t.Errorf("warped coat coverage in cells %d..%d: %d of 300", i+5, i+7, covered)
				}
			}
		})
	}
}

// A pair of dark dorsal ovals is separate from the smaller flank rings;
// checking the centre of each ring prevents solid spots passing as ocelli.
func TestAnacondaHasPairedDorsalMarksAndLightCentredFlanks(t *testing.T) {
	p := testProfile(t, "eunectes-murinus")
	along := 10.5 / p.Pattern.Repeat
	plain := p
	plain.Pattern.Kind = "plain"
	for _, side := range []float64{-1, 1} {
		dorsal := brightness(sampleCoat(p, along, side*.18))
		base := brightness(sampleCoat(plain, along, side*.18))
		if dorsal >= base*.9 {
			t.Fatalf("side %g has no dark dorsal mark", side)
		}
		centre := brightness(sampleCoat(p, along, side*.54))
		ring := brightness(sampleCoat(p, along+.13/p.Pattern.Repeat, side*.54))
		if centre <= ring+.08 {
			t.Fatalf("side %g flank mark lacks a light centre: centre %g, ring %g", side, centre, ring)
		}
	}
	if middle := brightness(sampleCoat(p, along, 0)); middle <= brightness(sampleCoat(p, along, .18))+.05 {
		t.Fatalf("dorsal pair merged at the spine: %g", middle)
	}
}

// The pale flank islands make a reticulated pattern more than a uniform
// crosshatched line grid. The two sides must both receive the pattern.
func TestReticulatedPythonHasPaleFlankIslandsWithinNet(t *testing.T) {
	p := testProfile(t, "malayopython-reticulatus")
	plain := p
	plain.Pattern.Kind = "plain"
	along := 10.5 / p.Pattern.Repeat
	for _, side := range []float64{-1, 1} {
		island := brightness(sampleCoat(p, along, side*.58))
		base := brightness(sampleCoat(plain, along, side*.58))
		outline := brightness(sampleCoat(p, along+.13/p.Pattern.Repeat, side*.58))
		if island <= base+.04 || island <= outline+.04 {
			t.Fatalf("side %g missing pale flank island and outline: island=%g base=%g outline=%g", side, island, base, outline)
		}
	}
}

// The checked contrast and palette govern the shared rule; a zero-contrast
// variant must not retain a hard-coded mark colour.
func TestPatternRespectsProfileContrast(t *testing.T) {
	for _, name := range []string{"boa-constrictor-occidentalis", "eunectes-murinus", "malayopython-reticulatus"} {
		p := testProfile(t, name)
		plain := p
		plain.Pattern.Kind = "plain"
		p.Pattern.Contrast = 0
		for _, turn := range []float64{0, .18, .54, -.54} {
			along := 10.5 / p.Pattern.Repeat
			if got, want := sampleCoat(p, along, turn), sampleCoat(plain, along, turn); got != want {
				t.Errorf("%s zero contrast at %g: got %v, want %v", name, turn, got, want)
			}
		}
	}
}

// A taper can close at a pole. Its paint must not divide zero by zero.
func TestSnakePaintKeepsATaperPoleFinite(t *testing.T) {
	p := testProfile(t, "bitis-gabonica")
	last := len(p.Stations) - 1
	p.Stations[last].Width, p.Stations[last].Height = 0, 0
	point := maths.V(p.Length/2, 0, 0)
	if got := (snakePaint{profile: p}).At(point); got != p.Pattern.Base {
		t.Fatalf("pole paint = %v, want finite base %v", got, p.Pattern.Base)
	}
}
