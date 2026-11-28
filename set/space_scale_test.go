package set

import (
	"strings"
	"testing"
)

func TestSpaceThingScaleIsExplicitAndDefaultsToOne(t *testing.T) {
	for _, test := range []struct {
		line string
		want float64
		has  bool
	}{{"", 1, false}, {"        scale: 2.5\n", 2.5, true}, {"        scale: 1\n", 1, true}} {
		text := "  setting: space\n" + stood(specimen("a", "        east: 0\n        north: 0\n"+test.line))
		read, err := readV2(text)
		if err != nil {
			t.Fatal(err)
		}
		got, has := read.Things()[0].Scale()
		if got != test.want || has != test.has {
			t.Fatalf("scale = %v, %v", got, has)
		}
	}
}

func TestSpaceThingScaleRefusesInvalidAndUnsupportedPlacements(t *testing.T) {
	legacy := "scene:\n  name: t\n  seed: 1\n  light: ambient\n" + stood(specimen("a", "        scale: 2\n"))
	if _, err := Read("legacy.yaml", []byte(legacy)); err == nil || !strings.Contains(err.Error(), "scale") {
		t.Fatalf("version 1 scale was accepted: %v", err)
	}
	for _, value := range []string{"0", "-1", "0.009", "101", ".nan", ".inf"} {
		text := "  setting: space\n" + stood(specimen("a", "        east: 0\n        north: 0\n        scale: "+value+"\n"))
		if _, err := readV2(text); err == nil {
			t.Fatalf("accepted scale %q", value)
		}
	}
	for _, text := range []string{
		"  setting: space\n" + stood(specimen("a", partSpinBlock+"        scale: 2\n")),
	} {
		if _, err := readV2(text); err == nil || !strings.Contains(err.Error(), "scale") {
			t.Fatalf("unsupported scaled placement was accepted: %v", err)
		}
	}
}

func TestGroundThingScaleIsAcceptedForExplicitVersionTwoPlacement(t *testing.T) {
	text := "  setting: ground\n" + stood(specimen("a", "        east: 0\n        north: 0\n        scale: 2\n"))
	read, err := readV2(text)
	if err != nil {
		t.Fatal(err)
	}
	if scale, ok := read.Things()[0].Scale(); !ok || scale != 2 {
		t.Fatalf("ground scale %v %v", scale, ok)
	}
}

func TestTypedPositionScaleRequiresSpace(t *testing.T) {
	placement := stood(specimen("a", absoluteTree+"        scale: 2\n"))
	space, err := readV2("  setting: space\n" + placement)
	if err != nil {
		t.Fatal(err)
	}
	if scale, ok := space.Things()[0].Scale(); !ok || scale != 2 {
		t.Fatalf("typed space scale %v %v", scale, ok)
	}
	if _, err := readV2("  setting: ground\n" + placement); err == nil || !strings.Contains(err.Error(), "scale") {
		t.Fatalf("typed ground scale was accepted: %v", err)
	}
}
