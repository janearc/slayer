package lines_test

import (
	"fmt"
	"os"

	"github.com/janearc/puzzlemouth/numbers"
)

// numbersFile is where the tests' picked numbers are, each with what it
// is and who picked it.
const numbersFile = "testdata/numbers.yaml"

// picked is a picked number by its name. A file that will not read, or
// a name it does not hold, is a mistake in the tests, so it stops them
// and says which.
func picked(name string) float64 {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		panic(fmt.Errorf("lines: %w", err))
	}
	chosen, err := numbers.Read(numbersFile, raw)
	if err != nil {
		panic(fmt.Errorf("lines: %w", err))
	}
	number, err := chosen.Get(name)
	if err != nil {
		panic(fmt.Errorf("lines: %w", err))
	}
	return number.Value
}
