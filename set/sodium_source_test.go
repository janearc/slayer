package set_test

import (
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
	"math"
	"os"
	"testing"
)

func TestNightCompositionKeepsThePinnedSodiumAirWavelengths(t *testing.T) {
	raw, err := os.ReadFile("../doc/fire-night.yaml")
	if err != nil {
		t.Fatal(err)
	}
	scene, err := set.Read("night.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := scene.Recipe("wood fire")
	if !ok {
		t.Fatal("night fire recipe missing")
	}
	spec, ok := row.Fire()
	if !ok || spec.Flame.Composition == nil {
		t.Fatal("sodium composition missing")
	}
	lines := spec.Flame.Composition.Lines()
	if len(lines) != 2 {
		t.Fatalf("sodium lines: %d", len(lines))
	}
	data, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := sources.Read("../sources.yaml")
	if err != nil {
		t.Fatal(err)
	}
	pin, ok := catalogue.Find("nist-handbook-sodium-lines")
	if !ok || !pin.Pinned() {
		t.Fatal("sodium handbook is not pinned")
	}
	for i, name := range []string{"sodium-d2-air-angstrom", "sodium-d1-air-angstrom"} {
		number, err := data.Get(name)
		if err != nil {
			t.Fatal(err)
		}
		// One angstrom is 0.1 nm. The YAML wavelengths use nm; the parsed
		// composition carries metres. Line radiance remains an authored input.
		want := number.Value * 1e-10
		line := lines[i]
		if number.Unit != "angstrom" || number.From == nil || number.From.Key != pin.Key() || line.Source != pin.From() || line.Species != "Na I" || line.Convention != "air" || !(math.Abs(line.Wavelength.InMetres()-want) < 1e-20) {
			t.Fatalf("line %d lost its pinned air wavelength or conversion: %+v", i, line)
		}
	}
}
