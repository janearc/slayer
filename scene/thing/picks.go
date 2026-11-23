package thing

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
// package's own files, which its tests find.
func readPicks(raw []byte) numbers.Numbers {
	read, err := numbers.Read("picks.yaml", raw)
	if err != nil {
		panic(fmt.Sprintf("thing: the book of chosen numbers is broken: %v",
			err))
	}
	return read
}

// pick is a chosen number by its name in picks.yaml. A name the book does
// not hold stops the program, as a broken book does.
func pick(name string) float64 {
	number, err := picks.Get(name)
	if err != nil {
		panic(fmt.Sprintf("thing: %v", err))
	}
	return number.Value
}
