package manufacture

import (
	"errors"
	"go/parser"
	"go/token"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
)

func mustLight(t *testing.T, colour spectrum, watts float64,
	over spread) light {
	t.Helper()
	made, err := newLight(colour, measure.Watts(watts), over)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func mustThermal(t *testing.T, kelvin float64) spectrum {
	t.Helper()
	colour, err := newThermal(measure.Kelvin(kelvin))
	if err != nil {
		t.Fatal(err)
	}
	return colour
}

func mustLine(t *testing.T, metres float64) spectrum {
	t.Helper()
	colour, err := newLine(measure.Metres(metres))
	if err != nil {
		t.Fatal(err)
	}
	return colour
}

func mustCone(t *testing.T, half, soft float64) spread {
	t.Helper()
	over, err := newCone(measure.Degrees(half), measure.Degrees(soft))
	if err != nil {
		t.Fatal(err)
	}
	return over
}

// gathered is a light's strength added up over every way it goes: the
// ball round it cut into thin rings by how far off the beam each is.
// it is worked out here by plain sums, apart from the code, and a
// plain sum across a hard edge is only as good as its rings are thin:
// about a part in ten thousand.
func gathered(made light) float64 {
	const thin = 400000
	total, width, strong := 0.0, math.Pi/thin, made.strength()
	for ring := range thin {
		off := (float64(ring) + 0.5) * width
		round := 2 * math.Pi * math.Sin(off) * width
		total += strong * made.along(off) * round
	}
	return total
}

// the truth for a light's strength is that none of its power is lost
// or made: added up over every way it goes, it is the power it was
// given, however it is spread.
func TestALightsStrengthAddsUpToItsPower(t *testing.T) {
	warm := mustThermal(t, 3200)
	wide, err := newAperture(measure.Metres(3))
	if err != nil {
		t.Fatal(err)
	}
	lights := map[string]light{
		"every way":       mustLight(t, warm, 60, everyway{}),
		"a narrow cone":   mustLight(t, warm, 1000, mustCone(t, 10, 0)),
		"a soft cone":     mustLight(t, warm, 1000, mustCone(t, 30, 8)),
		"half the sky":    mustLight(t, warm, 5, mustCone(t, 90, 0)),
		"through a dish":  mustLight(t, mustLine(t, 0.21), 1, wide),
		"a cone all soft": mustLight(t, warm, 1, mustCone(t, 1, 40)),
	}
	for name, made := range lights {
		power := made.power.InWatts()
		if got := gathered(made); math.Abs(got-power) > 2e-4*power {
			t.Errorf("%s: its strength adds up to %v watts of %v",
				name, got, power)
		}
	}
}

// a spot is brighter in its beam than a bare lamp of the same power,
// by the whole ball over the cap the beam covers; nobody says by how
// much.
func TestASpotIsBrighterThanABareLampOfItsPower(t *testing.T) {
	warm := mustThermal(t, 3200)
	bare := mustLight(t, warm, 100, everyway{})
	spot := mustLight(t, warm, 100, mustCone(t, 15, 0))
	cap := 2 * math.Pi * (1 - math.Cos(15*math.Pi/180))
	want := 4 * math.Pi / cap
	if got := spot.strength() / bare.strength(); math.Abs(got-
		want) > 1e-6*want {
		t.Errorf("a 15 degree spot is %v times a bare lamp, not %v",
			got, want)
	}
	if bare.along(0) != 1 || bare.along(math.Pi) != 1 {
		t.Error("a bare lamp is not as strong every way")
	}
	if spot.along(0.1) != 1 || spot.along(0.5) != 0 {
		t.Error("a spot is not all inside its cone and none outside")
	}
	// a soft edge fades: whole where it starts, half way through at
	// its middle, gone at its end
	soft := mustLight(t, warm, 100, mustCone(t, 20, 10))
	degree := math.Pi / 180
	fades := map[float64]float64{20: 1, 25: 0.5, 30: 0, 40: 0}
	for off, want := range fades {
		if got := soft.along(off * degree); math.Abs(got-want) > close {
			t.Errorf("%v degrees off a soft spot is %v of it, "+
				"not %v", off, got, want)
		}
	}
}

// a beam through an opening is as narrow as the opening allows: 1.22
// wavelengths over it, worked out and never given. a wider dish makes
// a narrower beam. it needs a line; a glow of every wavelength has no
// one width, and is refused.
func TestABeamThroughAnOpeningIsAsNarrowAsItAllows(t *testing.T) {
	const wave, dish = 0.21, 3.0
	through, err := newAperture(measure.Metres(dish))
	if err != nil {
		t.Fatal(err)
	}
	beam := mustLight(t, mustLine(t, wave), 1, through)
	half, soft := beam.angles()
	wide, edge, want := half.InRadians(), soft.InRadians(),
		1.22*wave/dish
	if math.Abs(wide-want) > close || edge != 0 {
		t.Errorf("a %v m wave through a %v m dish spreads %v with "+
			"an edge of %v, not %v", wave, dish, wide, edge, want)
	}
	wider, _ := newAperture(measure.Metres(2 * dish))
	narrow, _ := mustLight(t, mustLine(t, wave), 1, wider).angles()
	if narrow.InRadians() >= half.InRadians() {
		t.Error("a wider dish did not make a narrower beam")
	}
	_, err = newLight(mustThermal(t, 3000), measure.Watts(1), through)
	if !errors.Is(err, errLight) {
		t.Errorf("a glow through an opening: %v, not %v", err, errLight)
	}
}

// a beam far narrower than any row's is still a beam: its cap is
// near enough flat, pi times the half angle squared, and its strength
// is its power over that. found by review: worked out as one less a
// cosine, the cap was lost to rounding, and a beam of a hundred
// millionth of a radian came out endlessly strong.
func TestAVeryNarrowBeamKeepsItsStrength(t *testing.T) {
	warm := mustThermal(t, 3200)
	for _, half := range []float64{1e-3, 1e-7, 1e-9, 1e-12} {
		over, err := newCone(measure.Radians(half), measure.Radians(0))
		if err != nil {
			t.Fatal(err)
		}
		got := mustLight(t, warm, 1, over).strength()
		want := 1 / (math.Pi * half * half)
		if math.IsInf(got, 0) || math.Abs(got-want) > 1e-6*want {
			t.Errorf("a beam %v wide is %v strong, not %v", half,
				got, want)
		}
	}
}

// a long wave through a small opening spreads past half a turn by
// the rule, and half a turn is all the way there is: it shines every
// way. 21 cm through a hand's width is the case.
func TestAWaveLongerThanItsOpeningShinesEveryWay(t *testing.T) {
	small, err := newAperture(measure.Metres(0.05))
	if err != nil {
		t.Fatal(err)
	}
	half, soft := mustLight(t, mustLine(t, 0.21), 1, small).angles()
	if half.InRadians() != math.Pi || soft.InRadians() != 0 {
		t.Errorf("it spreads %v with an edge of %v, not half a turn",
			half.InRadians(), soft.InRadians())
	}
}

// a glow's colour is its heat's: a fire is redder than it is blue and
// a hot star the other way, by libtheme's own eye. a line is light in
// the band its wavelength falls in and nowhere else, and one far
// outside what an eye sees is dark to it.
func TestALightsColourIsItsHeatsOrItsLines(t *testing.T) {
	const red, blue = 650.0, 450.0
	fire := mustThermal(t, 1300).bands()
	star := mustThermal(t, 12000).bands()
	if fire.Power(bands.Of(red)) <= fire.Power(bands.Of(blue)) {
		t.Error("a fire is no redder than it is blue")
	}
	if star.Power(bands.Of(red)) >= star.Power(bands.Of(blue)) {
		t.Error("a hot star is no bluer than it is red")
	}
	green := mustLine(t, 550e-9).bands()
	if green.Power(bands.Of(550)) <= 0 || green.Power(bands.Of(red)) != 0 {
		t.Error("a green line is not in its own band alone")
	}
	radio := mustLine(t, 0.21).bands()
	first, last := radio.Span()
	if first != last || first != bands.Of(0.21e9) {
		t.Errorf("a 21 cm line spans bands %v to %v", first, last)
	}
	if radio.Power(bands.Of(550)) != 0 {
		t.Error("a 21 cm line can be seen")
	}
}

// a light that flickers is never brighter than it was made and never
// dimmer than its depth allows, changes as time passes, and flickers
// the same way again from the same seed. a light that does not is
// steady.
func TestAFlickeringLightStaysWithinItsDepth(t *testing.T) {
	steady := mustLight(t, mustThermal(t, 1300), 4000, everyway{})
	const depth = 0.4
	fire, err := steady.flickering(sow(3), depth)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := steady.flickering(sow(3), depth)
	other, _ := steady.flickering(sow(4), depth)
	seen := map[float64]bool{}
	differs := false
	for tick := range 400 {
		at := time.Duration(tick) * 37 * time.Millisecond
		share := fire.share(at)
		if share > 1 || share < 1-depth {
			t.Fatalf("at %v it is %v of itself, outside %v to 1",
				at, share, 1-depth)
		}
		if share != again.share(at) {
			t.Fatalf("at %v the same seed flickers another way", at)
		}
		differs = differs || share != other.share(at)
		seen[math.Round(share*100)] = true
		if still := steady.share(at); still != 1 {
			t.Fatalf("a steady light is %v of itself", still)
		}
	}
	if len(seen) < 10 || !differs {
		t.Errorf("it takes %d brightnesses in fifteen seconds, and "+
			"another seed differing is %v", len(seen), differs)
	}
}

// refusedLight fails unless an error is the light's refusal.
func refusedLight(t *testing.T, err error, what string, of any) {
	t.Helper()
	if !errors.Is(err, errLight) {
		t.Errorf("%s of %v: %v, not %v", what, of, err, errLight)
	}
}

func TestALightThatCannotBeIsRefused(t *testing.T) {
	nan, warm := math.NaN(), mustThermal(t, 3000)
	for _, kelvin := range []float64{0, -5, nan, math.Inf(1)} {
		_, err := newThermal(measure.Kelvin(kelvin))
		refusedLight(t, err, "a glow at a heat", kelvin)
	}
	for _, metres := range []float64{0, -1, nan, math.Inf(1)} {
		_, err := newLine(measure.Metres(metres))
		refusedLight(t, err, "a line", metres)
		_, err = newAperture(measure.Metres(metres))
		refusedLight(t, err, "an opening", metres)
	}
	cones := [][2]float64{{0, 0}, {-10, 0}, {181, 0}, {10, -1},
		{nan, 0}, {10, nan}, {170, 20}}
	for _, cone := range cones {
		_, err := newCone(measure.Degrees(cone[0]),
			measure.Degrees(cone[1]))
		refusedLight(t, err, "a cone", cone)
	}
	for _, watts := range []float64{0, -3, nan, math.Inf(1)} {
		_, err := newLight(warm, measure.Watts(watts), everyway{})
		refusedLight(t, err, "a light of a power", watts)
	}
	// a cone may be as wide as the whole ball, and no wider
	if _, err := newCone(measure.Radians(math.Pi),
		measure.Radians(0)); err != nil {
		t.Errorf("a cone of half a turn was refused: %v", err)
	}
	_, err := newLight(spectrum{}, measure.Watts(1), everyway{})
	refusedLight(t, err, "a light", "no colour")
	_, err = newLight(warm, measure.Watts(1), nil)
	refusedLight(t, err, "a light", "no spread")
	steady := mustLight(t, warm, 1, everyway{})
	for _, depth := range []float64{-0.1, 1.1, nan} {
		_, err := steady.flickering(sow(1), depth)
		refusedLight(t, err, "a flicker of a depth", depth)
	}
	_, err = steady.flickering(seed{}, 0.2)
	refusedLight(t, err, "a flicker", "no seed")
}

// the manufactory never calls emit, as ruled 2026-10-06. a light made
// here is a description, and emit is who reads it. this reads the
// package's own imports.
func TestTheManufactoryImportsNoEmit(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	files := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil,
			parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		files++
		for _, imported := range file.Imports {
			path := imported.Path.Value
			if strings.Contains(path, "slayer/emit") {
				t.Errorf("%s imports %s", name, path)
			}
		}
	}
	if files < 10 {
		t.Fatalf("only %d files were read", files)
	}
}
