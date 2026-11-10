package main

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// Use the public angle constructor to hold the scene adapter's units. The
// Moon model itself has its separate scientific source and library tests.
func TestMoonlightAdapterKeepsLowAltitudesAndDegreeUnits(t *testing.T) {
	for _, altitude := range []float64{0, .1, 1, 35} {
		t.Run(fmt.Sprintf("altitude%g", altitude), func(t *testing.T) {
			text := strings.Replace(fixtureScene, "light: moon",
				fmt.Sprintf("light: moon\n  moon: {azimuth: 90, altitude: %g, disc: false}", altitude), 1)
			read, err := set.Read("moon-units.yaml", []byte(text))
			if err != nil {
				t.Fatal(err)
			}
			moon, has := read.Moonlit()
			if !has {
				t.Fatal("reader lost the actual lunar angle")
			}
			stage := staging{given: mustParse(t), world: thing.NewWorld()}
			_, err = stage.moonlit(moon)
			if altitude == 0 {
				if err == nil || !strings.Contains(err.Error(), "not above the horizon") {
					t.Fatalf("moon at the actual horizon lost its refusal: %v", err)
				}
				return
			}
			if err != nil || len(stage.world.Lamps) != 1 || stage.made.lamps != 1 {
				t.Fatalf("valid low moon lost its actual single source: %v", err)
			}
			arriving, err := moonlight.Full().Through(measure.Metres(0), measure.Degrees(altitude))
			if err != nil {
				t.Fatal(err)
			}
			want := arriving.Picture(stage.given.exposure)
			way, distance, got := stage.world.Lamps[0].Light(maths.Vec{}, maths.YAxis)
			radians := altitude * math.Pi / 180
			if !(way.Sub(maths.V(math.Cos(radians), math.Sin(radians), 0)).Len() <= 1e-12 &&
				math.IsInf(distance, 1) && want.Luminance() > 0) {
				t.Fatal("lunar adapter lost its direction, distance or nonzero spectrum control")
			}
			for _, pair := range [][2]float64{{got.R, want.R}, {got.G, want.G}, {got.B, want.B}} {
				if !(math.Abs(pair[0]-pair[1]) <= 1e-11*math.Max(1e-30, math.Abs(pair[1]))) {
					t.Fatalf("lunar degree conversion changed transmitted light: %v, want %v", got, want)
				}
			}
		})
	}
}

// Two actual refusals are possible: an atmospheric height outside the model,
// and an unavailable face catalogue entry. Keep the first physical failure
// instead of continuing to the unrelated asset failure with a nil spectrum.
func TestLunarDiscKeepsItsActualAtmosphereRefusalBeforeFaceLookup(t *testing.T) {
	_, moon := boundaryMoon(t, ", disc: true, face: absent-mosaic")
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	stage := staging{given: given, world: thing.NewWorld(), lowest: 1e9}
	_, cause := moonlight.Disc().Through(measure.Metres(1e9), measure.Degrees(35))
	if cause == nil || !strings.Contains(cause.Error(), "outside the standard") {
		t.Fatalf("public atmospheric constructor lost its real refusal: %v", cause)
	}
	_, err := stage.moonlit(moon)
	if err == nil || err.Error() != cause.Error() {
		t.Fatalf("actual atmospheric cause was hidden by later face work: %v, want %v", err, cause)
	}
	if len(stage.world.Things) != 0 || len(stage.world.Lamps) != 0 {
		t.Fatal("failed lunar preparation added scene objects")
	}
}
