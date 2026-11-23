package manufacture

import "testing"

func TestATerraceAtTheDemoLevelCarriesALightForEachLampAndNoOtherLevelDoes(
	t *testing.T) {
	row := terraceRow{culture: "portofino"}
	for _, level := range []string{"hull", "quick"} {
		made, err := row.madeAt(sow(3), level)
		if err != nil {
			t.Fatal(err)
		}
		if len(made.lights) != 0 {
			t.Errorf("%d lights at %s", len(made.lights), level)
		}
	}
	made, err := row.madeAt(sow(3), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(made.lights) == 0 {
		t.Fatal("no light at the demo level, where lit windows have lamps")
	}
	for _, one := range made.lights {
		if !one.made.colour.made() || one.beyond.InMetres() != lampBeyond {
			t.Errorf("a lamp light %+v", one)
		}
	}
}
