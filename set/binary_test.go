package set

import (
	"strings"
	"testing"
)

// pairSet is a set that says a generated pair of stars, with a line the
// test may spoil.
func pairSet(spoil func(string) string) []byte {
	source := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: a pair
  film: {width: 64, height: 36, rays: 1, exposure: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -5, 2]}
  world:
    unit: 1
    binaries:
      pair:
        generate:
          primary: {mass: 2.45, age: 1.2e7}
          companion: {mass: 0.62, temperature: 25000, radius: 0.0122}
          separation: 9.2
          eccentricity: 0.16
        middle: [1, 2, 3]
        time: 4
`
	if spoil != nil {
		source = spoil(source)
	}
	return []byte(source)
}

// A set says a generated pair of stars as it says a generated planet:
// the numbers come back in the units the generator is made in.
func TestAPairOfStarsReadsItsNumbers(t *testing.T) {
	made, err := Read("pair.yaml", pairSet(nil))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.World()
	pairs := world.Binaries()
	if len(pairs) != 1 {
		t.Fatalf("read %d pairs, want 1", len(pairs))
	}
	binary := pairs[0].Generated()
	if got := binary.Primary.Mass.InSolarMasses(); got < 2.449 || got > 2.451 {
		t.Errorf("primary mass %v", got)
	}
	if got := binary.Companion.Temperature.InKelvin(); got < 24999 || got > 25001 {
		t.Errorf("companion temperature %v", got)
	}
	if got := binary.Orbit.Eccentricity; got != 0.16 {
		t.Errorf("eccentricity %v", got)
	}
	if got := float64(pairs[0].Time()); got != 4 {
		t.Errorf("time %v", got)
	}
	if got := pairs[0].Middle(); got.X != 1 || got.Y != 3 || got.Z != -2 {
		t.Errorf("middle %v, want east 1, up 3, south 2", got)
	}
}

// The anti-tests: a pair that leaves out a number, or says one out of
// range, is refused by the reader and not drawn with a guess.
func TestAPairMissingOrOutOfRangeIsRefused(t *testing.T) {
	for name, spoil := range map[string]func(string) string{
		"no separation": func(source string) string {
			return strings.Replace(source, "          separation: 9.2\n", "", 1)
		},
		"no radius": func(source string) string {
			return strings.Replace(source, ", radius: 0.0122", "", 1)
		},
		"no time": func(source string) string {
			return strings.Replace(source, "        time: 4\n", "", 1)
		},
		"an eccentricity of one": func(source string) string {
			return strings.Replace(source, "eccentricity: 0.16", "eccentricity: 1", 1)
		},
	} {
		if _, err := Read("pair.yaml", pairSet(spoil)); err == nil {
			t.Errorf("a pair with %s was read", name)
		}
	}
}
