package set

import (
	"strings"
	"testing"
)

const vaporCavitationSet = `scene:
  world:
    vapor-cavities:
      screw:
        colour: {srgb_linear: [0.8, 0.82, 0.84]}
        origin: [-1500, 0, -55]
        axis: [0.342, -0.94, 0]
        blades: 5
        tip-radius: 2.1
        rpm: 300
        slipstream-mps: 20
        length-m: 40
        core-radius-m: 0.2
        density-per-m: 0.8
        fragment-scale-m: 0.5
        fragment-contrast: 0
        fragment-seed: 705
        pressure-deficit-pa: 12000
        recovery-pa-per-m: 600
        visual-onset: true
        clear-water: true
        source: eye
`

func TestVaporCavitationReadsFiveTipVisualFieldControls(t *testing.T) {
	world, err := Read("vapor.yaml", []byte(vaporCavitationSet))
	if err != nil {
		t.Fatal(err)
	}
	contents, hasWorld := world.World()
	if !hasWorld {
		t.Fatal("set lost its world")
	}
	fields := contents.VaporCavities()
	if len(fields) != 1 {
		t.Fatalf("read %d propeller fields", len(fields))
	}
	field := fields[0]
	if field.Name() != "screw" || !field.VisualOnset() || !field.ClearWater() {
		t.Fatalf("field identity or opt-ins lost: %+v", field)
	}
	if radius, rpm, slip, length := field.Controls(); radius != 2.1 || rpm != 300 || slip != 20 || length != 40 {
		t.Fatalf("field controls %g, %g, %g, %g", radius, rpm, slip, length)
	}
	if got := field.Origin(); got.X != -1500 || got.Y != -55 || got.Z != 0 {
		t.Fatalf("map origin converted to %v", got)
	}
	if got := field.Axis(); got.X != 0.342 || got.Y != 0 || got.Z != 0.94 {
		t.Fatalf("map axis converted to %v", got)
	}
}

func TestVaporCavitationRequiresExplicitVisualAndTransportChoices(t *testing.T) {
	for _, removal := range []string{"        visual-onset: true\n", "        clear-water: true\n", "        source: eye\n"} {
		changed := strings.Replace(vaporCavitationSet, removal, "", 1)
		if _, err := Read("vapor.yaml", []byte(changed)); err == nil {
			t.Errorf("accepted field without %q", strings.TrimSpace(removal))
		}
	}
	changed := strings.Replace(vaporCavitationSet, "        visual-onset: true", "        visual-onset: false", 1)
	world, err := Read("vapor.yaml", []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	contents, _ := world.World()
	if contents.VaporCavities()[0].VisualOnset() {
		t.Fatal("false visual-onset became true")
	}
}

func TestVaporCavitationIsInBothSchemaVersionsAndReader(t *testing.T) {
	versions := []schemaVersion{{"v1", ecosystemRaw, false}, {"v2", ecosystemV2Raw, true}}
	for _, version := range versions {
		if !strings.Contains(string(version.raw), `"vapor-cavities"`) {
			t.Errorf("%s schema has no vapor-cavities key", version.name)
		}
		keySetsStayInStep(t, version)
	}
}
