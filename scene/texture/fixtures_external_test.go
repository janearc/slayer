package texture_test

import (
	"os"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
)

// nothing is defined: no amount.
const nothing = 0

// whole is defined: one.
const whole = 1

// numbersFile is the file every number a test of this package chooses is
// in, asked for by its name.
const numbersFile = "testdata/numbers.yaml"

// readChosen is the numbers file, read and checked.
func readChosen() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return nil, err
	}
	return numbers.Read(numbersFile, raw)
}

// chosenNumbers is the numbers file, read once.
var chosenNumbers = sync.OnceValues(readChosen)

// chosen is a number a test picked, by its name in the numbers file. A
// file that cannot be read, or a name it does not hold, stops the test.
func chosen(test *testing.T, name string) float64 {
	test.Helper()
	read, err := chosenNumbers()
	if err != nil {
		test.Fatal(err)
	}
	number, err := read.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return number.Value
}

// grey is the light a test names by the grey in the numbers file.
func grey(test *testing.T, name string) maths.Light {
	test.Helper()
	return maths.Grey(chosen(test, name))
}

// sampleAt is a point whose three parts a test names in the numbers file.
func sampleAt(test *testing.T, across, up, along string) maths.Vec {
	test.Helper()
	return maths.V(chosen(test, across), chosen(test, up), chosen(test, along))
}
