package set_test

import (
	"github.com/janearc/slayer/set"
	"os"
	"strings"
	"testing"
)

func bedFixture(t *testing.T) string {
	t.Helper()
	b, e := os.ReadFile("../doc/fire-fuel.yaml")
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func TestFuelBedReadsOnlyExplicitPhysicalInputs(t *testing.T) {
	s, e := set.Read("bed.yaml", []byte(bedFixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	beds := s.FuelBeds()
	if len(beds) != 1 {
		t.Fatal("lost bed")
	}
	b := beds[0]
	if b.Recipe() != "surface fire" || b.Cell() != .5 || b.Step() != .01 || b.Width() != 32 || b.Depth() != 32 || b.East() != 263812 || b.North() != 2148748 || b.Radius() != 1.6 || b.Hot() != 1200 || b.Ambient() != 300 || b.Fuel() != 1 {
		t.Fatal("lost units or settings", b)
	}
	beds[0] = set.FuelBed{}
	if s.FuelBeds()[0].Recipe() == "" {
		t.Fatal("mutable config")
	}
}
func TestFuelBedRefusesWrongUnitsUnknownModelsAndUnsupportedEffects(t *testing.T) {
	for _, pair := range [][2]string{
		{"model: mandel-grass", "model: crown"}, {"unit: s", "unit: ms"}, {"width: 32\n      depth", "width: 33\n      depth"},
		{"recipe: surface fire", "recipe: unknown"}, {"      fuel: 1", "      fuel: -1"}, {"value: 0.01, unit: s", "value: 1, unit: s"},
		{"ambient-temperature: {value: 300", "ambient-temperature: {value: 1500"},
		{"      fuel: 1", "      fuel: 1\n      burn-duration: 3"},
		{"      light-cells: 3", "      light-cells: 3\n      smoke:\n        heat-release: {value: 40, unit: kW}"},
	} {
		raw := strings.Replace(bedFixture(t), pair[0], pair[1], 1)
		if raw == bedFixture(t) {
			t.Fatal("ineffective anti-test", pair)
		}
		if _, e := set.Read("bad-bed.yaml", []byte(raw)); e == nil {
			t.Fatal("accepted", pair)
		}
	}
}

func TestFuelSurfaceChoiceDefaultsToFuelAndCanKeepTerrain(t *testing.T) {
	for _, surface := range []string{"", "fuel", "terrain"} {
		raw := bedFixture(t)
		if surface != "" {
			raw = strings.Replace(raw, "      fuel: 1", "      fuel: 1\n      surface: "+surface, 1)
		}
		got, err := set.Read("surface.yaml", []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		want := surface
		if want == "" {
			want = "fuel"
		}
		if got.FuelBeds()[0].Surface() != want {
			t.Fatal("surface choice changed", surface, got.FuelBeds()[0].Surface())
		}
	}
	raw := strings.Replace(bedFixture(t), "      fuel: 1", "      fuel: 1\n      surface: bark", 1)
	if _, err := set.Read("bad-surface.yaml", []byte(raw)); err == nil {
		t.Fatal("unknown surface accepted")
	}
}
