package set

import (
	"strings"
	"testing"
)

const partSpinBlock = "        east: 0\n        north: 0\n        part-spin:\n" +
	"          material: propeller blade\n" +
	"          pivot-metres: [0, 4.75, 38.9]\n" +
	"          axis: [0, 0, 1]\n" +
	"          rpm: -300\n"

func TestPartSpinReadsNativeUnitsPhaseAndBothSettingOrders(t *testing.T) {
	for _, text := range []string{
		"  setting: space\n" + stood(specimen("a", partSpinBlock)),
		stood(specimen("a", partSpinBlock)) + "  setting: space\n",
	} {
		read, err := readV2(text)
		if err != nil {
			t.Fatal(err)
		}
		got, has := read.Things()[0].PartSpin()
		if !has || got.Material() != "propeller blade" || got.PivotMetres() != ([3]float64{0, 4.75, 38.9}) || got.Axis() != ([3]float64{0, 0, 1}) || got.RPM() != -300 || got.PhaseRadians() != 0 {
			t.Fatalf("spin=%+v has%t", got, has)
		}
	}
	plant := specimen("a", partSpinBlock+"          phase-radians: 1.5707963267948966\n")
	read, err := readV2("  setting: space\n" + stood(plant))
	if err != nil {
		t.Fatal(err)
	}
	spin, _ := read.Things()[0].PartSpin()
	if spin.PhaseRadians() != 1.5707963267948966 {
		t.Fatal(spin)
	}
}
func TestPartSpinOmissionAndRefusals(t *testing.T) {
	plant := specimen("a", "        east: 0\n        north: 0\n")
	read, err := readV2(stood(plant))
	if err != nil {
		t.Fatal(err)
	}
	if _, has := read.Things()[0].PartSpin(); has {
		t.Fatal("omitted spin present")
	}
	for _, row := range []struct{ block, setting string }{
		{partSpinBlock, "ground"},
		{strings.Replace(partSpinBlock, "axis: [0, 0, 1]", "axis: [0, 0, 0]", 1), "space"},
		{strings.Replace(partSpinBlock, "rpm: -300", "rpm: .nan", 1), "space"},
		{strings.Replace(partSpinBlock, "rpm: -300", "rpm: 1000001", 1), "space"},
		{strings.Replace(partSpinBlock, "material: propeller blade", "material: '   '", 1), "space"},
		{strings.Replace(partSpinBlock, "          axis: [0, 0, 1]\n", "", 1), "space"},
		{partSpinBlock + "          mystery: 1\n", "space"},
	} {
		plant := specimen("a", row.block)
		if _, err := readV2("  setting: " + row.setting + "\n" + stood(plant)); err == nil {
			t.Fatalf("invalid part spin accepted: %s%s", row.setting, row.block)
		}
	}
}

func TestPartSpinSharesAnExplicitRPMAnchor(t *testing.T) {
	first := strings.Replace(partSpinBlock, "rpm: -300", "rpm: &shaft -300", 1)
	second := strings.Replace(partSpinBlock, "rpm: -300", "rpm: *shaft", 1)
	plantA := specimen("a", first)
	plantB := specimen("b", second)
	read, err := readV2("  setting: space\n" + stood(plantA, plantB))
	if err != nil {
		t.Fatal(err)
	}
	one, _ := read.Things()[0].PartSpin()
	two, _ := read.Things()[1].PartSpin()
	if one.RPM() != -300 || two.RPM() != one.RPM() {
		t.Fatal("explicit shared RPM anchor was not preserved")
	}
}
