package set_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// legacyScene exercises the reader independently of the opt-in schema.
func legacyScene(raw string) string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, "$schema:") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// omitSceneBlock removes one fixture key and its indented contents.
func omitSceneBlock(t *testing.T, raw, key string) string {
	t.Helper()
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if line != key {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		end := i + 1
		for end < len(lines) {
			next := lines[end]
			if strings.TrimSpace(next) != "" &&
				len(next)-len(strings.TrimLeft(next, " ")) <= indent {
				break
			}
			end++
		}
		return strings.Join(append(lines[:i], lines[end:]...), "\n")
	}
	t.Fatalf("fixture lacks %q", key)
	return ""
}

func TestLegacyEffectsReaderRefusesMissingAndMalformedContracts(t *testing.T) {
	fire, front := fireRecipeFixture(t), frontFixture(t)
	composed, bed := composedFire(t), bedFixture(t)
	sparks := strings.Replace(fire, "      flame:", sparkBlock+"      flame:", 1)
	ecosystem := ecosystemFixture(t)
	for _, test := range []struct{ name, raw, from, to, want string }{
		{"line list", composed, "          lines:", "          lines: {}\n          old-lines:", "lines takes at most 32"},
		{"line value", composed, "wavelength: {value: 588.995, unit: nm}", "wavelength: {value: 0, unit: nm}", "takes a number"},
		{"line keys", composed, "              convention: air\n", "", "this has no convention"},
		{"composition keys", composed, "          continuum: 0.7\n", "", "this has no continuum"},
		{"composition value", composed, "continuum: 0.7", "continuum: -1", "takes a number"},
		{"empty composition", composed, "species: Na I", "species: ' '", "invalid emission composition"},
		{"flame value", fire, "unit: K", "unit: C", "unit takes"},
		{"smoke value", fire, "soot-yield: 0.0024", "soot-yield: -1", "takes a number"},
		{"smoke keys", fire, "        albedo: 0.3\n", "", "this has no albedo"},
		{"spark value", sparks, "unit: s^-1", "unit: Hz-ish", "unit takes"},
		{"spark keys", sparks, "        gravity: {value: 9.80665, unit: m/s^2}\n", "", "this has no gravity"},
		{"front value", front, "speed: {value: 2", "speed: {value: -1", "takes a number"},
		{"front keys", front, "      fade-time: {value: 0.2, unit: s}\n", "", "this has no fade-time"},
		{"front fade", front, "fade-time: {value: 0.2", "fade-time: {value: 5", "fade exceeds half"},
		{"front recipe", front, "      recipe: wood fire\n      from:", "      recipe: missing\n      from:", "needs a known fire recipe"},
		{"bed value", bed, "width: 32", "width: 33", "takes a number"},
		{"bed keys", bed, "      fuel: 1\n", "", "this has no fuel"},
		{"bed dimensions", bed, "width: 32", "width: 31", "multiples of eight"},
		{"bed temperature", bed, "ignition-temperature: {value: 1200", "ignition-temperature: {value: 300", "hotter than ambient"},
		{"recipe keys", ecosystem, "      kind: optical-observatory\n", "", "needs a kind"},
		{"primary kind", ecosystem, "primary: segmented", "primary: triangular", "primary takes"},
		{"recipe type", ecosystem, "kind: optical-observatory", "kind: spruce", "not an optical observatory"},
		{"recipe scalar", ecosystem, "      kind: coast-redwood", "      kind: []", "kind takes"},
		{"mist value", ecosystem, "coverage: 0.4", "coverage: -1", "takes a number"},
		{"mist keys", ecosystem, "      seed: 23\n", "", "this has no seed"},
		{"cache backend", fire, "renderer: gpu-native", "renderer: cpu\n  volume-lighting: cached", "requires gpu-native"},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := strings.Replace(test.raw, test.from, test.to, 1)
			if raw == test.raw {
				t.Fatal("ineffective fixture change")
			}
			_, err := set.Read("legacy-effects.yaml", []byte(legacyScene(raw)))
			var schemaError *jsonschema.ValidationError
			if err == nil || errors.As(err, &schemaError) ||
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf("want reader refusal %q, got %v", test.want, err)
			}
		})
	}
	for _, test := range []struct{ raw, key, want string }{
		{fire, "      flame:", "this has no flame"},
		{bed, "  terrain:", "fuel bed needs ground scenery"},
	} {
		raw := omitSceneBlock(t, test.raw, test.key)
		_, err := set.Read("missing-effects.yaml", []byte(legacyScene(raw)))
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("missing %s: want %q, got %v", test.key, test.want, err)
		}
	}
}
