package carry_test

import (
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/janearc/puzzlemouth/numbers"
)

// numbersFile is where the tests' picked numbers are, each with what it
// is and who picked it.
const numbersFile = "testdata/numbers.yaml"

// chosen is the picked numbers as the file holds them, and why the file
// could not be read if it could not. When the file will not read the
// numbers stay empty, and every name asked for is refused with the
// reason.
var chosen, unread = readChosen()

// readChosen reads the numbers file.
func readChosen() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// pickedFloat is a picked number by its name. Asking for a name the file
// does not hold is a mistake in the tests, so it stops them and says
// which.
func pickedFloat(name string) float32 {
	number, err := chosen.Get(name)
	if err != nil {
		panic(errors.Join(unread, fmt.Errorf("carry: %w", err)))
	}
	return float32(number.Value)
}

// pickedWhole is a picked number by its name, as a whole count. Numbers
// that are not whole are a mistake in the file, and stop the tests too.
func pickedWhole(name string) int {
	value := float64(pickedFloat(name))
	if value != math.Trunc(value) {
		panic(fmt.Errorf("carry: %s is %v, not a whole count", name, value))
	}
	return int(value)
}
