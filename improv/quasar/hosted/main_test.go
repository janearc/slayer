package main

import (
	"bytes"
	"testing"
)

func TestRunRefusesAShortWithNoTune(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-script", "x.yaml", "-out", "y.mp4"}, &out,
		&problems) != failed {
		t.Fatal("a short with no tune was taken")
	}
}

func TestRunRefusesAFlagItDoesNotKnow(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-colour", "red"}, &out, &problems) != failed {
		t.Fatal("an unknown flag was taken")
	}
}

func TestNamesAndSetsReadACommaList(t *testing.T) {
	got := names(" hairline, breaks,,")
	if len(got) != 2 || got[0] != "hairline" || got[1] != "breaks" {
		t.Fatalf("the list reads %q", got)
	}
	if !set("ice")["ice"] || set("")["ice"] {
		t.Fatal("the set is wrong")
	}
}
