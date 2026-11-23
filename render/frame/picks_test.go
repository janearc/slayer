package frame_test

import (
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/janearc/puzzlemouth/numbers"
)

// numbersFile is where the tests' picked numbers are, each with what it is
// and why it was picked.
const numbersFile = "testdata/numbers.yaml"

// chosen is the picked numbers as the file holds them, and why the file could
// not be read if it could not. When the file will not read the numbers stay
// empty, and every name asked for is refused with the reason.
var chosen, unread = readChosen()

// readChosen reads the numbers file.
func readChosen() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// pickedFloat is a picked number by its name. Asking for a name the file does
// not hold is a mistake in the tests, so it stops them and says which.
func pickedFloat(name string) float64 {
	number, err := chosen.Get(name)
	if err != nil {
		missing := fmt.Errorf("frame: %w", err)
		panic(errors.Join(unread, missing))
	}
	return number.Value
}

// pickedWhole is a picked number by its name, as a whole count. A number that
// is not whole is a mistake in the file, and stops the tests too.
func pickedWhole(name string) int {
	value := pickedFloat(name)
	if value != math.Trunc(value) {
		panic(fmt.Errorf("frame: %s is %v, not a whole count", name, value))
	}
	return int(value)
}
