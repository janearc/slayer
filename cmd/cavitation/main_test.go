package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCavitationCloudCommandUsesNativeConstructor(t *testing.T) {
	var output, errs bytes.Buffer
	status := run([]string{"cloud", "--count", "2", "--seed", "7052026"},
		&output, &errs)
	if status != 0 {
		t.Fatalf("command failed (%d): %s", status, errs.String())
	}
	for _, want := range []string{"spheres:", "cavitation-01:", "cavitation-02:",
		"material: air-bubble", "source: eye"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("generated sphere block lacks %q:\n%s", want, output.String())
		}
	}
}

func TestCavitationCloudCommandRefusesInvalidBounds(t *testing.T) {
	var output, errs bytes.Buffer
	status := run([]string{"cloud", "--count", "513"}, &output, &errs)
	if status == 0 {
		t.Fatal("command accepted an over-budget cloud")
	}
}

func TestCavitationFieldCommandUsesTheNativeBoundedMedium(t *testing.T) {
	var output, errs bytes.Buffer
	status := run([]string{"field", "--rpm", "240"}, &output, &errs)
	if status != 0 {
		t.Fatalf("field command failed (%d): %s", status, errs.String())
	}
	for _, want := range []string{"vapor-cavities:", "main-screw:",
		"blades: 5", "fragment-contrast: 0", "fragment-seed: 705",
		"tip-radius: 2.1", "rpm: 240", "slipstream-mps: 20",
		"pressure-deficit-pa: 12000", "recovery-pa-per-m: 600",
		"visual-onset: true", "clear-water: true", "source: eye"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("field YAML lacks %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "spheres:") {
		t.Fatal("the analytic field command expanded into per-particle geometry")
	}
}

func TestCavitationFieldCommandRefusesUnsupportedRPM(t *testing.T) {
	var output, errs bytes.Buffer
	status := run([]string{"field", "--rpm", "601"}, &output, &errs)
	if status == 0 {
		t.Fatal("field command accepted an out-of-contract phase rate")
	}
}

func TestCavitationFieldCommandAuthorsGenericBladeCount(t *testing.T) {
	var output, errs bytes.Buffer
	if got := run([]string{"field", "--blades", "2"}, &output, &errs); got != 0 {
		t.Fatalf("two-blade field refused: %s", errs.String())
	}
	if !strings.Contains(output.String(), "blades: 2") {
		t.Fatalf("authored blade count missing: %s", output.String())
	}
	for _, bad := range []string{"0", "17", "2.5"} {
		output.Reset()
		errs.Reset()
		if got := run([]string{"field", "--blades", bad}, &output, &errs); got == 0 {
			t.Fatalf("unsupported blade count %s accepted", bad)
		}
	}
}
