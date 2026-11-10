package film_test

import (
	"github.com/janearc/slayer/film"
	"math"
	"testing"
)

// zeroAngle is defined as no field of view.
const zeroAngle = 0

// unitAngle is defined as one radian, also the positive infinity sign.
const unitAngle = 1

func TestPinholeOfKeepsTheRequestedFieldOfView(test *testing.T) {
	sensor, err := film.PinholeOf(unitAngle)
	if err != nil {
		test.Fatal(err)
	}
	if sensor.Wide != unitAngle || sensor.Pixel != zeroAngle {
		test.Fatalf("changed the requested pinhole: %+v", sensor)
	}
}

func TestPinholeOfRefusesInvalidAngles(test *testing.T) {
	for _, angle := range []float64{zeroAngle, -unitAngle, math.Pi,
		math.Inf(unitAngle), math.Inf(-unitAngle), math.NaN()} {
		sensor, err := film.PinholeOf(angle)
		if err == nil || sensor != nil {
			test.Fatalf("accepted angle %g", angle)
		}
	}
}
