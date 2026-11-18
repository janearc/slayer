package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// the kilauea iki set lit by the sun through Venus's air, and the lines
// that say it, in place of its moonlight.
const (
	moonlit    = "  light: moon\n"
	underVenus = "  light: sun\n  air: venus\n" +
		"  sun:\n    azimuth: 135\n    altitude: 55\n"
)

// aired is the kilauea iki set with its moonlight line said otherwise.
func aired(t *testing.T, said string) (set.Set, error) {
	t.Helper()
	written := strings.Replace(kilaueaIki(t), moonlit, said, 1)
	return set.Read("kilauea-iki.yaml", []byte(written))
}

// inTheSky is the giant set's planet put in a ground set's sky, and the
// line that names an air.
const (
	inTheSky = "    sky: {azimuth: 120, altitude: 40, distance: 1e9}\n"
	anAir    = "  light: sun\n  air: venus\n"
)

// skied is the giant set with its planet in the sky, and with what more
// its light line says.
func skied(t *testing.T, light string) (set.Set, error) {
	t.Helper()
	written := strings.Replace(giant, "    from_star:",
		inTheSky+"    from_star:", 1)
	written = strings.Replace(written, "  light: sun\n", light, 1)
	return set.Read("skied.yaml", []byte(written))
}

// an air with a planet in the sky is refused, saying the computed sky
// does not hold a planet's light.
func TestAnAirWithAPlanetInTheSkyIsRefused(t *testing.T) {
	_, err := skied(t, anAir)
	if err == nil || !strings.Contains(err.Error(), "planet's light") {
		t.Errorf("an air with a planet in the sky: %v", err)
	}
}

// the anti-test: the same planet in the same sky with no air is read:
// the refusal is of the two together.
func TestAPlanetInTheSkyWithNoAirIsRead(t *testing.T) {
	made, err := skied(t, "  light: sun\n")
	if err != nil {
		t.Fatal(err)
	}
	world, has := made.Planet()
	if _, _, _, inSky := world.InSky(); !has || !inSky {
		t.Errorf("the planet %v, in the sky %v", has, inSky)
	}
}

// a set names its air under the sun, and a set that names none has none.
func TestASetNamesItsAir(t *testing.T) {
	made, err := aired(t, underVenus)
	if err != nil {
		t.Fatal(err)
	}
	if made.Air() != "venus" || made.Light() != "sun" {
		t.Errorf("the air %q under %q", made.Air(), made.Light())
	}
	if air := kilaueaIkiRead(t).Air(); air != "" {
		t.Errorf("a set that names no air has %q", air)
	}
}

// the anti-test: an air nobody knows is refused, naming the airs there
// are.
func TestAnUnknownAirIsRefusedNamingTheKnown(t *testing.T) {
	_, err := aired(t, strings.Replace(underVenus, "venus", "titan", 1))
	if err == nil || !strings.Contains(err.Error(), "venus") {
		t.Errorf("an unknown air: %v", err)
	}
}

// an air under the moon is refused, saying the air takes the sun.
func TestAnAirWithoutTheSunIsRefused(t *testing.T) {
	_, err := aired(t, moonlit+"  air: venus\n")
	if err == nil || !strings.Contains(err.Error(), "light: sun") {
		t.Errorf("an air under the moon: %v", err)
	}
}

// the anti-test: the same air under the sun, with no sun block, is read:
// the refusal is of the light, not of the air.
func TestAnAirUnderTheSunIsRead(t *testing.T) {
	if _, err := aired(t, "  light: sun\n  air: venus\n"); err != nil {
		t.Errorf("an air under the sun: %v", err)
	}
}
