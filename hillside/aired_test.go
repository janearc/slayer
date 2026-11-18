package hillside

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/airlight"
	"github.com/janearc/slayer/airs"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
)

// A set under Venus's air, tinted or not, and the exposures the choice
// of one is checked on.
const (
	airedSet = "scene:\n  name: under an air\n  light: sun\n" +
		"  air: venus\n  sun: {azimuth: 135, altitude: 55%s}\n"
	tinted       = ", tint: [1, 0.5, 0.25]"
	saidExposure = 3.5  // picked: any exposure a job or film says
	meteredAt    = 1234 // picked: any exposure the meter gives
)

// sunUnder is the sun of the set under Venus's air, with what more its
// sun block says.
func sunUnder(t *testing.T, more string) set.Sun {
	t.Helper()
	read, err := set.Read("aired.yaml", []byte(fmt.Sprintf(airedSet, more)))
	if err != nil {
		t.Fatal(err)
	}
	sun, has := read.Sunlit()
	if !has || read.Air() != "venus" {
		t.Fatalf("the set's sun %v under %q", has, read.Air())
	}
	return sun
}

// The flat sun Venus's plan is lit by here, in the spectrum's own form:
// the plan's words do not hang on the sun's numbers.
const (
	sunHeading = "wavelength (nm),irradiance (W/m^2/nm)"
	flatFirst  = 300 // picked: below the plan's first band
	flatLast   = 900 // picked: above its last
	flatPower  = 1.5 // picked: watts a square metre a nanometre
)

// airlightPlan is Venus's plan from its mean radius, under a flat sun
// of flatPower at every whole nanometre from flatFirst to flatLast.
func airlightPlan(t *testing.T) airlight.Plan {
	t.Helper()
	var text strings.Builder
	text.WriteString(sunHeading + "\n")
	for nanometres := flatFirst; nanometres <= flatLast; nanometres++ {
		fmt.Fprintf(&text, "%d,%g\n", nanometres, flatPower)
	}
	sun, err := airs.ReadSun(strings.NewReader(text.String()))
	if err != nil {
		t.Fatal(err)
	}
	known, err := airlight.Named("venus", sun)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := airlight.NewPlan(known, nothing)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// -sky black under an air is refused, saying an air's sky is its air's.
func TestABlackSkyUnderAnAirIsRefused(t *testing.T) {
	err := refusedBlack(asked{sky: "black", set: "aired.yaml"})
	if err == nil || !strings.Contains(err.Error(), "-sky black") ||
		!strings.Contains(err.Error(), "air's") {
		t.Errorf("-sky black under an air: %v", err)
	}
}

// the anti-test: the sky as it is left, painted, is taken under an air,
// whose sky stands in its place.
func TestThePaintedSkyFlagUnderAnAirIsTaken(t *testing.T) {
	if err := refusedBlack(asked{sky: "painted"}); err != nil {
		t.Errorf("-sky painted under an air: %v", err)
	}
}

// stagedIn is a stage whose set stands in a directory, or the test
// skipped where no shelf is named, as the tree's other tests that read
// the shelf are. The pin's check is kept in the test's own cache.
func stagedIn(t *testing.T, dir string) *staging {
	t.Helper()
	if os.Getenv(sources.Placed) == "" {
		t.Skip("no shelf named in SLAYER_SHELF")
	}
	cachedHome(t)
	return &staging{given: asked{set: filepath.Join(dir, "aired.yaml")}}
}

// the sun's spectrum is read from the shelf, held to the pin in this
// tree's sources.yaml: over the visible it has light at every band.
func TestTheSunIsReadFromTheShelf(t *testing.T) {
	sun, err := stagedIn(t, "..").sunSpectrum()
	if err != nil {
		t.Fatal(err)
	}
	if band, err := sun.Band(flatFirst, flatLast); err != nil ||
		!(band > nothing) {
		t.Errorf("the shelf's sun from %d to %d nm: %g, %v", flatFirst,
			flatLast, band, err)
	}
}

// the anti-test: a set with no sources.yaml above it has no pin to hold
// the shelf's file to, and its sun is refused.
func TestASunWithNoPinIsRefused(t *testing.T) {
	if _, err := stagedIn(t, t.TempDir()).sunSpectrum(); err == nil {
		t.Error("a sun was read with no sources.yaml to pin it")
	}
}

// a sun the set tints under an air is refused, saying why.
func TestATintedSunUnderAnAirIsRefused(t *testing.T) {
	err := refusedTint(sunUnder(t, tinted), "aired.yaml")
	if err == nil || !strings.Contains(err.Error(), "tint") {
		t.Errorf("a tinted sun under an air: %v", err)
	}
}

// the anti-test: the same sun untinted is taken.
func TestAnUntintedSunUnderAnAirIsTaken(t *testing.T) {
	if err := refusedTint(sunUnder(t, ""), "aired.yaml"); err != nil {
		t.Errorf("an untinted sun under an air: %v", err)
	}
}

// meterGives is a meter that gives so much, and notes it was read.
func meterGives(read *bool) func() (float64, error) {
	return func() (float64, error) {
		*read = true
		return meteredAt, nil
	}
}

// an exposure the job or the film said stands, and the meter is not
// read.
func TestASaidExposureStands(t *testing.T) {
	read := false
	got, metered, err := exposureFor(asked{exposure: saidExposure},
		meterGives(&read))
	if err != nil || got != saidExposure || metered || read {
		t.Errorf("a said exposure gives %g, metered %v, read %v, %v", got,
			metered, read, err)
	}
}

// the anti-test: with none said, the meter's is taken, and its refusal
// is passed on.
func TestNoExposureSaidIsMetered(t *testing.T) {
	read := false
	got, metered, err := exposureFor(asked{}, meterGives(&read))
	if err != nil || got != meteredAt || !metered || !read {
		t.Errorf("no exposure said gives %g, metered %v, read %v, %v", got,
			metered, read, err)
	}
	refused := errors.New("unlit")
	_, _, err = exposureFor(asked{}, func() (float64, error) {
		return nothing, refused
	})
	if !errors.Is(err, refused) {
		t.Errorf("the meter's refusal is lost: %v", err)
	}
}

// a run under an air says the sky is computed, from what, and how its
// exposure was had; and a note told is said as it is.
func TestASkyFromAnAirIsSaid(t *testing.T) {
	note := cacheNote{what: skySaid(airlightPlan(t), cacheNote{}, meteredAt,
		true), told: true}
	for _, word := range []string{"computed", "venus", "metered",
		"worked out"} {
		if !strings.Contains(note.String(), word) {
			t.Errorf("%q does not say %q", note.String(), word)
		}
	}
}

// the anti-test: a sky from the cache says so, and an exposure said is
// not called metered.
func TestASkyFromTheCacheIsSaidSo(t *testing.T) {
	said := skySaid(airlightPlan(t), cacheNote{hit: true}, saidExposure,
		false)
	if !strings.Contains(said, "from the cache") ||
		strings.Contains(said, "metered") {
		t.Errorf("a cached sky with a said exposure: %q", said)
	}
}
