package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRefusesAShotWithNowhereToGo(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "bent"}, &out, &problems) != failed ||
		!strings.Contains(problems.String(), "-out") {
		t.Fatalf("a shot with no -out was taken: %q", problems.String())
	}
}

func TestRunRefusesAShotItDoesNotKnow(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "sideways", "-out", t.TempDir() + "/x.png"},
		&out, &problems) != failed || !strings.Contains(
		problems.String(), "sideways") {
		t.Fatalf("an unknown shot was taken: %q", problems.String())
	}
}

func TestRunRefusesAFlagItDoesNotKnow(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-colour", "red"}, &out, &problems) != failed {
		t.Fatal("an unknown flag was taken")
	}
}

func TestMeasureSaysWhereTheBandsLie(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "measure", "-out", "unused"}, &out,
		&problems) != ok {
		t.Fatal(problems.String())
	}
	for _, said := range []string{"edge", "order 0", "order 2",
		"close-in"} {
		if !strings.Contains(out.String(), said) {
			t.Errorf("measure does not say %q: %s", said, out.String())
		}
	}
}

func TestDiagramIsWritten(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "diagram", "-width", "120", "-out",
		t.TempDir() + "/d.png"}, &out, &problems) != ok {
		t.Fatal(problems.String())
	}
}
