package main

import (
	"bytes"
	"testing"
)

func TestRunRefusesAShortWithNoScript(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-short", "x", "-out", "y"}, &out, &problems) !=
		failed {
		t.Fatal("a short with no script was taken")
	}
}

func TestRunRefusesAFlagItDoesNotKnow(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-colour", "red"}, &out, &problems) != failed {
		t.Fatal("an unknown flag was taken")
	}
}

func TestRunRefusesAScriptThatIsNotThere(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-script", t.TempDir() + "/none.yaml", "-short",
		"x", "-out", "y"}, &out, &problems) != failed {
		t.Fatal("a missing script was taken")
	}
}
