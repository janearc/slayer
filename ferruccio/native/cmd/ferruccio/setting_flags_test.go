package main

import (
	"strings"
	"testing"
)

func TestEverySettingHasAFlagAndOpenRefusesAValueNotOnItsList(t *testing.T) {
	noSettings(t)
	for _, args := range [][]string{
		{"open", "--renderer", "quantum", "--dry-run"},
		{"open", "--processors", "3", "--dry-run"},
		{"proof", "--out", "unused.png", "--renderer", "quantum", "--dry-run"},
	} {
		code, output := runAtelier(args...)
		if code == 0 || !strings.Contains(output, "unknown") || !strings.Contains(output, "choose") {
			t.Fatalf("%v returned %d: %s", args, code, output)
		}
	}
}

func TestRendererAndProcessorsFlagsTakeEveryValueTheSheetOffers(t *testing.T) {
	noSettings(t)
	for _, args := range [][]string{
		{"open", "--renderer", "cpu", "--dry-run"},
		{"open", "--renderer", "gpu", "--dry-run"},
		{"open", "--processors", "2", "--dry-run"},
		{"open", "--processors", "4", "--dry-run"},
		{"open", "--processors", "8", "--dry-run"},
		{"open", "--processors", "all", "--dry-run"},
	} {
		if code, output := runAtelier(args...); code != 0 {
			t.Fatalf("%v returned %d: %s", args, code, output)
		}
	}
}

func TestProofHasASettingsSwitch(t *testing.T) {
	noSettings(t)
	if code, output := runAtelier("proof", "--out", "unused.png", "--settings", "--dry-run"); code != 0 {
		t.Fatalf("proof --settings returned %d: %s", code, output)
	}
}
