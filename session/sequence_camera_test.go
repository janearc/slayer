package session

import (
	"os"
	"strings"
	"testing"
)

func TestSequenceSchemaRejectsCameraTyposUnitsAndConflictingTiming(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	for _, change := range []struct{ before, after string }{
		{"unit: m\n", "unit: feet\n"},
		{"unit: m/s", "unit: km/h"},
		{"speed:", "speeed:"},
		{"value: 2", "value: 0"},
		{"speed: {value: 2, unit: m/s}", "speed: {value: 2, unit: m/s}\n      duration: {value: 2, unit: s}"},
		{"speed: {value: 2, unit: m/s}", "ease: smooth\n      speed: {value: 2, unit: m/s}"},
		{"z: 2}", "z: -2}"},
		{"{x: 0, y: 0, z: 2}", "{x: 0, y: 0}"},
	} {
		text := string(raw) + strings.Replace(cameraFixture, change.before, change.after, sequenceOne)
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			test.Fatal(err)
		}
		if _, err := readSequence(path); err == nil {
			test.Fatalf("accepted invalid camera %q", change.after)
		}
	}
}

// moveLook is the fixture's move given its own look: from the path's
// target at its start to a metre east of it at its end.
const moveLook = "speed: {value: 2, unit: m/s}\n" +
	"      look: {from: {x: 0, y: 1, z: 0}, to: {x: 1, y: 1, z: 0}}"

// the anti-test: a look that says only one of its two targets is
// refused by the schema before any frame.
func TestSequenceSchemaRefusesALookHalfSaid(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	half := strings.Replace(moveLook, ", to: {x: 1, y: 1, z: 0}", "",
		sequenceOne)
	camera := strings.Replace(cameraFixture,
		"speed: {value: 2, unit: m/s}", half, sequenceOne)
	sequenceWrite(test, path, append(raw, camera...))
	if _, err := readSequence(path); err == nil {
		test.Error("a look without its second target was taken")
	}
}
