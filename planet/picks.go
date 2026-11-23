package planet

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/numbers"
)

// picksRaw is picks.yaml, every number this package chose that is not true
// by definition, each with what it is and why it was chosen.
//
//go:embed picks.yaml
var picksRaw []byte

// picks is picks.yaml read, before any number is asked for.
var picks = readPicks(picksRaw)

// readPicks reads the embedded book of chosen numbers, and stops the
// program if it cannot be read: a book that is broken is a mistake in the
// package's own files, which its tests find. The refusal names the file and
// each row it refuses.
func readPicks(raw []byte) numbers.Numbers {
	read, err := numbers.Read("planet/picks.yaml", raw)
	if err != nil {
		panic(fmt.Sprintf("planet: the book of chosen numbers is broken: %v",
			err))
	}
	return read
}

// pick is a chosen number by its name in picks.yaml. A name the book does
// not hold stops the program, as a broken book does.
func pick(name string) float64 {
	number, err := picks.Get(name)
	if err != nil {
		panic(fmt.Sprintf("planet: %v", err))
	}
	return number.Value
}

// pickCount is a chosen number that counts things, by its name in
// picks.yaml. A number that is not a whole one stops the program, as a
// broken book does.
func pickCount(name string) int {
	value := pick(name)
	count := int(value)
	if float64(count) != value {
		panic(fmt.Sprintf("planet: %s is %g, which counts nothing", name,
			value))
	}
	return count
}
