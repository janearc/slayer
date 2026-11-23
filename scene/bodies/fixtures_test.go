package bodies_test

import (
	"os"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// What the tests count in, each true by definition.
const (
	nought = 0 // defined: no amount, and the first place in a list
	unity  = 1 // defined: one amount, the whole of a share
)

// numbersFile holds every number the tests of this file's kind choose,
// each with what it is and why. A test asks for one by its name.
const numbersFile = "testdata/numbers.yaml"

// chosenNumbers is numbers.yaml, read once.
var chosenNumbers = sync.OnceValues(readChosenNumbers)

// readChosenNumbers reads numbers.yaml.
func readChosenNumbers() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// picked is a number the tests chose, by its name in numbers.yaml. A name
// the file does not hold stops the test.
func picked(t *testing.T, name string) float64 {
	t.Helper()
	chosen, err := chosenNumbers()
	if err != nil {
		t.Fatal(err)
	}
	number, err := chosen.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// pickedCount is a picked number that counts something, as an unsigned
// integer.
func pickedCount(t *testing.T, name string) uint {
	t.Helper()
	return uint(picked(t, name))
}
