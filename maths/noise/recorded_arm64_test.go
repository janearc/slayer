//go:build arm64

package noise_test

import (
	"math"
	"testing"
)

// The Gradient and Fractal rows of recorded.csv are held bit for bit.
//
// They are recorded from the code on arm64, and only arm64 was measured.
// arm64 fuses the multiplies and adds of the quintic and the blends into
// one rounding each, so their last bits are the fused ones. On amd64 at the
// default GOAMD64 nothing fuses, and the same rows would likely differ in
// the last bit. The whole-number rows are the same everywhere and are held
// in noise_test.go.
func TestTheNoiseIsTheRecorded(test *testing.T) {
	holdToRecorded(test, recordedRows(test, true))
}

// anti-test: with each octave finer than the one before by a hair more than
// octaveRise, the reference Fractal of each Fractal row is not the recorded
// number, and with octaveRise it is. So the recorded rows are moved by a
// wrong constant, and the row that moves is the one the rise is in.
func TestAWrongOctaveRiseMovesTheRecordedFractals(test *testing.T) {
	wrong := octaveRise + picked(test, "rise-error")
	counted := nought
	for _, row := range recordedRows(test, true) {
		if row["call"] != "Fractal" {
			continue
		}
		counted++
		seeded, at := seed(test, row, "seed"), place{float(test, row, "a"),
			float(test, row, "b"), float(test, row, "c")}
		octaves, recorded := int(whole(test, row, "octaves")),
			bits(test, row, "want")
		right := math.Float64bits(fractalRising(octaveRise, seeded, at.x, at.y,
			at.z, octaves))
		if right != recorded {
			test.Errorf("%s: the reference fractal is %#016x, recorded is %s",
				row["name"], right, row["want"])
		}
		moved := math.Float64bits(fractalRising(wrong, seeded, at.x, at.y,
			at.z, octaves))
		if moved == recorded {
			test.Errorf("%s: a wrong rise still gives the recorded number",
				row["name"])
		}
	}
	if counted == nought {
		test.Fatal("no Fractal rows were tried")
	}
}
